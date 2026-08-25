package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/shopspring/decimal"
)

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
