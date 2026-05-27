// Package strategysvc provides the Python strategy-service HTTP client (S2.1).
//
// Bridges Go ↔ Python strategy-service (http://strategy-service:8081).
// Replaces the hardcoded mock responses in PythonStrategyServer with real
// backtest/execute/validate results from the Python engine.
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

// PythonClient communicates with the Python strategy backtest/execution engine.
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

// BacktestRequest mirrors the Python service's /api/backtest payload.
type BacktestRequest struct {
	Code      string  `json:"code"`
	Symbol    string  `json:"symbol"`
	Timeframe string  `json:"timeframe"`
	StartDate string  `json:"start_date,omitempty"`
	EndDate   string  `json:"end_date,omitempty"`
	Balance   float64 `json:"initial_balance"`
	Leverage  int32   `json:"leverage,omitempty"`
}

// BacktestResult is the response from the Python backtest engine.
type BacktestResult struct {
	Success      bool      `json:"success"`
	EquityCurve  []float64 `json:"equity_curve"`
	TotalReturn  float64   `json:"total_return"`
	SharpeRatio  float64   `json:"sharpe_ratio"`
	MaxDrawdown  float64   `json:"max_drawdown"`
	WinRate      float64   `json:"win_rate"`
	TradeCount   int32     `json:"trade_count"`
	Error        string    `json:"error,omitempty"`
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
		Side      string  `json:"side"`
		Lots      float64 `json:"lots"`
		Price     float64 `json:"price"`
		StopLoss  float64 `json:"stop_loss"`
		TakeProfit float64 `json:"take_profit"`
		Reason    string  `json:"reason"`
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

// Backtest sends a strategy to the Python backtest engine.
func (c *PythonClient) Backtest(ctx context.Context, req *BacktestRequest) (*BacktestResult, error) {
	var result BacktestResult
	if err := c.post(ctx, "/api/backtest", req, &result); err != nil {
		return nil, fmt.Errorf("strategysvc backtest: %w", err)
	}
	return &result, nil
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/health", nil)
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
		return fmt.Errorf("post %s: status %d: %s", path, resp.StatusCode, string(respBytes))
	}

	if err := json.Unmarshal(respBytes, respBody); err != nil {
		return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBytes[:min(len(respBytes), 200)]))
	}
	return nil
}
