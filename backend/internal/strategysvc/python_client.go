// Package strategysvc provides the Python strategy-service HTTP client (S2.1).
//
// Bridges Go ↔ Python strategy-service (http://strategy-service:8081).
// Execute/Validate use REST endpoints; backtest is fully migrated to ConnectRPC.
package strategysvc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PythonClient communicates with the Python strategy execution engine.
// Backtest path (formerly /api/backtest) has been removed — use
// BacktestServiceClient (ConnectRPC) via antv1c.NewBacktestServiceClient instead.
type PythonClient struct {
	baseURL string
	httpc   *http.Client
}

// NewPythonClient creates a client for the given strategy-service base URL.
func NewPythonClient(baseURL string) *PythonClient {
	return &PythonClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 60 * time.Second},
	}
}

// ExecuteRequest mirrors the Python service's /api/execute payload.
type ExecuteRequest struct {
	Code      string `json:"code"`
	AccountID string `json:"account_id"`
	Symbol    string `json:"symbol"`
	Timeframe string `json:"timeframe"`
	Mode      string `json:"mode"` // "paper" or "live"
}

// ExecuteResult is the response from the Python execute endpoint.
type ExecuteResult struct {
	Success bool   `json:"success"`
	Signal  *struct {
		Side       string  `json:"side"`
		Lots       float64 `json:"lots"`
		Price      float64 `json:"price"`
		StopLoss   float64 `json:"stop_loss"`
		TakeProfit float64 `json:"take_profit"`
		Reason     string  `json:"reason"`
	} `json:"signal,omitempty"`
	Error string `json:"error,omitempty"`
}

// ValidateRequest mirrors the Python service's /api/validate payload.
type ValidateRequest struct {
	Code string `json:"code"`
}

// ValidateResult is the response from the Python validate endpoint.
type ValidateResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// Execute runs a strategy live or in paper mode on the Python engine.
func (c *PythonClient) Execute(ctx context.Context, req *ExecuteRequest) (*ExecuteResult, error) {
	var result ExecuteResult
	if err := c.post(ctx, "/api/execute", req, &result); err != nil {
		return nil, fmt.Errorf("strategysvc execute: %w", err)
	}
	return &result, nil
}

// Validate checks Python syntax and basic strategy structure.
func (c *PythonClient) Validate(ctx context.Context, req *ValidateRequest) (*ValidateResult, error) {
	var result ValidateResult
	if err := c.post(ctx, "/api/validate", req, &result); err != nil {
		return nil, fmt.Errorf("strategysvc validate: %w", err)
	}
	return &result, nil
}

// Health checks if the Python strategy-service is reachable.
func (c *PythonClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("strategysvc health: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("strategysvc health: status %d", resp.StatusCode)
	}
	return nil
}

func (c *PythonClient) post(ctx context.Context, path string, reqBody, respBody interface{}) error {
	body, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("post %s: %w", path, err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		// Truncate response body to prevent traceback leakage to clients.
		body := string(respBytes)
		if len(body) > 200 {
			body = body[:200] + "..."
		}
		return fmt.Errorf("post %s: status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.Unmarshal(respBytes, respBody); err != nil {
		return fmt.Errorf("unmarshal response: %w (body: %s)", err,
			string(respBytes[:min(len(respBytes), 200)]))
	}
	return nil
}
