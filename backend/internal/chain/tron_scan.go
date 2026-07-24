package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// TronScanClient calls TronScan API as a backup verification source.
type TronScanClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewTronScanClient(apiKey string) *TronScanClient {
	return &TronScanClient{
		apiKey:  apiKey,
		baseURL: "https://apilist.tronscanapi.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// tronScanTxResponse is the TronScan API response for transaction-info.
// The response is a flat object (not an array), with fields at top level.
type tronScanTxResponse struct {
	Hash        string `json:"hash"`
	Block       int64  `json:"block"`
	Confirmed   bool   `json:"confirmed"`
	ToAddress   string `json:"toAddress"`
	OwnerAddress string `json:"ownerAddress"`
	ContractType int   `json:"contractType"`
	ContractData struct {
		ToAddress   string `json:"to_address"`
		OwnerAddress string `json:"owner_address"`
		Amount      int64  `json:"amount"`
	} `json:"contractData"`
	Revert bool `json:"revert"`
}

// VerifyTransaction checks if a USDT transfer transaction exists on TronScan.
// Returns (verified, error) — verified=true means TronScan confirms the tx.
func (c *TronScanClient) VerifyTransaction(ctx context.Context, txHash, expectedTo string) (bool, error) {
	params := url.Values{}
	params.Set("hash", txHash)

	reqURL := fmt.Sprintf("%s/api/transaction-info?%s", c.baseURL, params.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return false, fmt.Errorf("tronscan: create request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("tronscan: request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		return false, fmt.Errorf("tronscan: read body: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("tronscan: status %d", resp.StatusCode)
	}

	var result tronScanTxResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return false, fmt.Errorf("tronscan: unmarshal: %w", err)
	}

	if !result.Confirmed {
		return false, nil
	}

	if result.Revert {
		return false, nil
	}

	// For TRC20 transfers (contractType=1), the top-level toAddress is the smart contract
	// address (e.g. USDT contract), NOT the token recipient. The actual recipient is in
	// contractData.to_address. For non-contract txs, toAddress is the direct recipient.
	var recipient string
	if result.ContractType == 1 && result.ContractData.ToAddress != "" {
		recipient = result.ContractData.ToAddress
	} else {
		recipient = result.ToAddress
		if recipient == "" {
			recipient = result.ContractData.ToAddress
		}
	}

	return recipient == expectedTo, nil
}
