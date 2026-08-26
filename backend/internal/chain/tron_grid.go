// Package chain implements on-chain monitoring for USDT TRC20 deposits.
// It scans TronGrid block events for Transfer events to user deposit addresses.
package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/shopspring/decimal"
)

// TronGridClient calls TronGrid API to fetch USDT Transfer events per block.
type TronGridClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewTronGridClient(apiKey string) *TronGridClient {
	return &TronGridClient{
		apiKey:  apiKey,
		baseURL: "https://api.trongrid.io",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// TransferEvent represents a parsed USDT TRC20 Transfer event.
// TronGrid returns Transfer parameters in a nested "result" map with
// positional keys "0" (from), "1" (to), "2" (value, in smallest unit).
type TransferEvent struct {
	TxHash       string
	BlockNumber  int64
	From         string
	To           string
	AmountString string // raw value string (smallest unit, needs /10^6 for USDT)
	Confirmed    bool
}

// tronGridRawEvent is the raw TronGrid API response item for contract events.
type tronGridRawEvent struct {
	TransactionID string            `json:"transaction_id"`
	BlockNumber   int64             `json:"block_number"`
	EventName     string            `json:"event_name"`
	Result        map[string]string `json:"result"`
	Unconfirmed   bool              `json:"_unconfirmed"`
}

// tronGridEventResponse is the raw TronGrid API response for contract events.
// TronGrid v1 endpoints return HTTP 200 with success:false when rate-limited
// or when the API rejects the request; Error/Message carry the reason.
type tronGridEventResponse struct {
	Data []tronGridRawEvent `json:"data"`
	Meta struct {
		Fingerprint string `json:"fingerprint"`
	} `json:"meta"`
	Success bool   `json:"success"`
	Error   string `json:"Error"`
	Message string `json:"message"`
}

// validateEventResponse checks that a TronGrid v1 event API response indicates
// success. TronGrid returns HTTP 200 with success:false when rate-limited or
// when the API rejects the request. Treating that as empty data would silently
// skip blocks (lost deposits) or fail-open the double-spend check.
//
// Called per-page so that a failure on any pagination page aborts the whole
// call — callers must return nil, error (not partial first-page data).
// The error includes the operation name, the request page fingerprint (which
// page failed), and the API-provided error/message. It never includes the
// API key.
func validateEventResponse(result *tronGridEventResponse, op, pageFingerprint string) error {
	if !result.Success {
		apiErr := result.Error
		if apiErr == "" {
			apiErr = result.Message
		}
		if apiErr == "" {
			apiErr = "no error detail provided by API"
		}
		return fmt.Errorf("trongrid: %s: API returned success=false (page fingerprint=%q): %s",
			op, pageFingerprint, apiErr)
	}
	return nil
}

// GetBlockEvents fetches all USDT Transfer events for a specific block.
// Handles pagination via fingerprint.
func (c *TronGridClient) GetBlockEvents(ctx context.Context, contractAddress string, blockNumber int64) ([]TransferEvent, error) {
	var allEvents []TransferEvent
	fingerprint := ""

	for {
		params := url.Values{}
		params.Set("event_name", "Transfer")
		params.Set("block_number", strconv.FormatInt(blockNumber, 10))
		params.Set("only_confirmed", "true")
		params.Set("limit", "200")
		if fingerprint != "" {
			params.Set("fingerprint", fingerprint)
		}

		reqURL := fmt.Sprintf("%s/v1/contracts/%s/events?%s",
			c.baseURL, contractAddress, params.Encode())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("trongrid: create request: %w", err)
		}
		if c.apiKey != "" {
			req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("trongrid: request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("trongrid: read body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("trongrid: status %d: %s", resp.StatusCode, string(body))
		}

		var result tronGridEventResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, fmt.Errorf("trongrid: unmarshal: %w", err)
		}

		// Fail-closed: HTTP 200 + success:false means the API rejected the
		// request (rate limit, etc.). Treating it as empty data would silently
		// skip the block and advance the checkpoint, losing deposits forever.
		// Return nil (not partial allEvents) so scanBlocks does not saveCheckpoint.
		if err := validateEventResponse(&result, "GetBlockEvents", fingerprint); err != nil {
			return nil, err
		}

		for _, raw := range result.Data {
			if raw.EventName != "Transfer" {
				continue
			}
			// TronGrid result map uses positional keys:
			// "0" = from (hex address), "1" = to (hex address), "2" = value (smallest unit)
			fromHex := raw.Result["0"]
			toHex := raw.Result["1"]
			valueStr := raw.Result["2"]

			from := hexToBase58(fromHex)
			to := hexToBase58(toHex)

			// USDT has 6 decimals. valueStr is in smallest unit (e.g. "1000000" = 1 USDT).
			// Convert to human-readable decimal string.
			amountStr := convertSunToUSDT(valueStr)

			allEvents = append(allEvents, TransferEvent{
				TxHash:       raw.TransactionID,
				BlockNumber:  raw.BlockNumber,
				From:         from,
				To:           to,
				AmountString: amountStr,
				Confirmed:    !raw.Unconfirmed,
			})
		}

		if result.Meta.Fingerprint == "" {
			break
		}
		fingerprint = result.Meta.Fingerprint
	}

	return allEvents, nil
}

// HasOutgoingTRC20Transfer checks if an address has any recent outgoing TRC20 transfer
// to the specified destination. Used for double-spend prevention before re-sweeping (ADR §2.7).
// Uses the account-specific TRC20 events endpoint for efficiency (not global contract events).
func (c *TronGridClient) HasOutgoingTRC20Transfer(ctx context.Context, from, to, contract string) (bool, error) {
	fingerprint := ""
	for {
		params := url.Values{}
		params.Set("limit", "200")
		params.Set("order_by", "block_timestamp,desc")
		params.Set("only_confirmed", "true")
		params.Set("contract_address", contract)
		if fingerprint != "" {
			params.Set("fingerprint", fingerprint)
		}

		reqURL := fmt.Sprintf("%s/v1/accounts/%s/events/trc20?%s",
			c.baseURL, from, params.Encode())

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return false, fmt.Errorf("trongrid: create request: %w", err)
		}
		if c.apiKey != "" {
			req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("trongrid: request: %w", err)
		}
		body, err := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if err != nil {
			return false, fmt.Errorf("trongrid: read body: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return false, fmt.Errorf("trongrid: status %d: %s", resp.StatusCode, string(body))
		}

		var result tronGridEventResponse
		if err := json.Unmarshal(body, &result); err != nil {
			return false, fmt.Errorf("trongrid: unmarshal: %w", err)
		}

		// Fail-closed: HTTP 200 + success:false must NOT be treated as "no
		// outgoing transfer" (false, nil). That would fail-open the double-
		// spend check and allow re-sweeping an address that may have already
		// been swept. Return (false, error) so CheckDoubleSpend blocks the retry.
		if err := validateEventResponse(&result, "HasOutgoingTRC20Transfer", fingerprint); err != nil {
			return false, err
		}

		for _, raw := range result.Data {
			if raw.EventName != "Transfer" {
				continue
			}
			evtTo := hexToBase58(raw.Result["1"])
			if evtTo == to {
				return true, nil
			}
		}

		if result.Meta.Fingerprint == "" {
			break
		}
		fingerprint = result.Meta.Fingerprint
	}

	return false, nil
}

// GetLatestBlock returns the latest block number from TronGrid.
func (c *TronGridClient) GetLatestBlock(ctx context.Context) (int64, error) {
	reqURL := fmt.Sprintf("%s/wallet/getnowblock", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, fmt.Errorf("trongrid: create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("trongrid: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("trongrid: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("trongrid: status %d: %s", resp.StatusCode, string(body))
	}

	// getnowblock uses a different response shape than v1 event endpoints:
	// it has no "success" field. Instead, errors are signalled by a top-level
	// "Error" string, and a valid response must contain blockID + block_header
	// + raw_data.number. Use pointer fields to distinguish "field absent" from
	// "field present but zero-valued".
	var result struct {
		BlockID     string `json:"blockID"`
		Error       string `json:"Error"`
		BlockHeader *struct {
			RawData *struct {
				Number    int64 `json:"number"`
				Timestamp int64 `json:"timestamp"`
			} `json:"raw_data"`
		} `json:"block_header"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("trongrid: unmarshal block: %w", err)
	}

	// Fail-closed structural validation. TronGrid can return HTTP 200 with an
	// Error field (rate limit) or an empty/invalid block structure. Returning
	// block=0,nil would make scanBlocks think the chain is at height 0 and
	// potentially skip all pending deposits.
	if result.Error != "" {
		return 0, fmt.Errorf("trongrid: GetLatestBlock: API error: %s", result.Error)
	}
	if result.BlockID == "" {
		return 0, fmt.Errorf("trongrid: GetLatestBlock: empty blockID in response")
	}
	if result.BlockHeader == nil {
		return 0, fmt.Errorf("trongrid: GetLatestBlock: missing block_header in response")
	}
	if result.BlockHeader.RawData == nil {
		return 0, fmt.Errorf("trongrid: GetLatestBlock: missing block_header.raw_data in response")
	}
	if result.BlockHeader.RawData.Number <= 0 {
		return 0, fmt.Errorf("trongrid: GetLatestBlock: invalid block number %d", result.BlockHeader.RawData.Number)
	}

	return result.BlockHeader.RawData.Number, nil
}

// GetTRC20Balance returns the USDT TRC20 balance of an address as a decimal string (in USDT, 6 decimals).
// Uses TronGrid's /v1/accounts/{address} endpoint which includes trc20 balances.
func (c *TronGridClient) GetTRC20Balance(ctx context.Context, address string) (string, error) {
	reqURL := fmt.Sprintf("%s/v1/accounts/%s", c.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return "", fmt.Errorf("trongrid: create balance request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("trongrid: balance request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return "", fmt.Errorf("trongrid: read balance body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("trongrid: balance status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			TRC20 []map[string]string `json:"trc20"`
		} `json:"data"`
		Success bool `json:"success"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("trongrid: unmarshal balance: %w", err)
	}

	if !result.Success {
		return "", fmt.Errorf("trongrid: balance query unsuccessful (success=false)")
	}

	// If account has no data, balance is 0.
	if len(result.Data) == 0 {
		return "0", nil
	}

	// TronGrid returns trc20 as array of {contract_address: balance_string} maps.
	// e.g. [{"TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t": "1000000"}]
	for _, tokenMap := range result.Data[0].TRC20 {
		for contract, balance := range tokenMap {
			if contract == usdtContractMainnet {
				return convertSunToUSDT(balance), nil
			}
		}
	}

	return "0", nil
}

// usdtContractMainnet is the USDT TRC20 contract on TRON mainnet.
const usdtContractMainnet = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

// hexToBase58 converts a TronGrid hex address (e.g. "4152d6c..." with 0x41 prefix) to Base58.
// Returns empty string if input is empty or invalid.
func hexToBase58(hexAddr string) string {
	if hexAddr == "" {
		return ""
	}
	// TronGrid returns hex addresses with 0x41 prefix (TRON mainnet byte).
	// gotron-sdk's HexToAddress handles this format.
	addr, err := address.HexToAddress(hexAddr)
	if err != nil {
		return hexAddr // fallback to raw value
	}
	return addr.String()
}

// convertSunToUSDT converts a raw value string (in smallest unit, 10^-6 for USDT)
// to a human-readable USDT amount string with 6 decimal places.
// e.g. "1000000" → "1.000000", "1500000" → "1.500000"
func convertSunToUSDT(rawValue string) string {
	if rawValue == "" {
		return "0"
	}
	// Parse as decimal to avoid float precision issues.
	d, err := decimal.NewFromString(rawValue)
	if err != nil {
		return rawValue // fallback to raw value
	}
	divisor := decimal.NewFromInt(1000000) // 10^6 for USDT
	return d.Div(divisor).StringFixed(6)
}
