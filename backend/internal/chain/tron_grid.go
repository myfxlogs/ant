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
