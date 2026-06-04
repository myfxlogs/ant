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
	"math"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	circuitThreshold = 5                 // consecutive failures before opening circuit
	circuitCooldown  = 30 * time.Second  // how long circuit stays open
	maxRetries       = 3                 // max retry attempts for transient errors
)

// PythonClient communicates with the Python strategy backtest/execution engine.
type PythonClient struct {
	baseURL             string
	httpc               *http.Client
	mu                  sync.Mutex
	consecutiveFailures int
	circuitOpenUntil    time.Time
}

// NewPythonClient creates a client for the given strategy-service base URL.
func NewPythonClient(baseURL string) *PythonClient {
	return &PythonClient{
		baseURL: baseURL,
		httpc:   &http.Client{Timeout: 60 * time.Second},
	}
}

// isCircuitOpen returns true if the circuit breaker is currently open.
func (c *PythonClient) isCircuitOpen() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.consecutiveFailures >= circuitThreshold {
		if time.Now().Before(c.circuitOpenUntil) {
			return true
		}
		// Half-open: allow one request through.
		c.consecutiveFailures = circuitThreshold - 1
	}
	return false
}

func (c *PythonClient) recordSuccess() {
	c.mu.Lock()
	c.consecutiveFailures = 0
	c.mu.Unlock()
}

func (c *PythonClient) recordFailure() {
	c.mu.Lock()
	c.consecutiveFailures++
	if c.consecutiveFailures >= circuitThreshold {
		c.circuitOpenUntil = time.Now().Add(circuitCooldown)
	}
	c.mu.Unlock()
}

// BacktestRequest mirrors the Python service's /api/backtest payload.
type BacktestRequest struct {
	StrategyID         string                 `json:"strategy_id"`
	Code               string                 `json:"strategy_code"`
	Symbol             string                 `json:"symbol"`
	Timeframe          string                 `json:"timeframe"`
	StartDate          string                 `json:"start_date"`
	EndDate            string                 `json:"end_date"`
	Balance            float64                `json:"initial_balance"`
	Leverage           int32                  `json:"leverage,omitempty"`
	ParameterOverrides map[string]interface{} `json:"parameter_overrides,omitempty"`
}

// BacktestResult is the response from the Python backtest engine.
type BacktestResult struct {
	Success        bool      `json:"success"`
	EquityCurve    []float64 `json:"equity_curve"`
	TotalReturn    float64   `json:"total_return"`
	AnnualReturn   float64   `json:"annual_return"`
	MaxDrawdown    float64   `json:"max_drawdown"`
	SharpeRatio    float64   `json:"sharpe_ratio"`
	WinRate        float64   `json:"win_rate"`
	ProfitFactor   float64   `json:"profit_factor"`
	TotalTrades    int32     `json:"total_trades"`
	WinningTrades  int32     `json:"winning_trades"`
	LosingTrades   int32     `json:"losing_trades"`
	AverageProfit  float64   `json:"average_profit"`
	AverageLoss    float64   `json:"average_loss"`
	TradeCount     int32     `json:"trade_count"`
	// Risk assessment
	RiskScore   int32    `json:"risk_score"`
	RiskLevel   string   `json:"risk_level"`
	RiskReasons []string `json:"risk_reasons"`
	RiskWarnings []string `json:"risk_warnings"`
	IsReliable  bool     `json:"is_reliable"`
	Error       string   `json:"error,omitempty"`
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

// Backtest sends a strategy to the Python backtest engine with retry + circuit breaker.
func (c *PythonClient) Backtest(ctx context.Context, req *BacktestRequest) (*BacktestResult, error) {
	if c.isCircuitOpen() {
		return nil, fmt.Errorf("strategysvc backtest: circuit breaker open, service unavailable")
	}
	var result BacktestResult
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 1s, 2s, 4s.
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}
		lastErr = c.post(ctx, "/api/backtest", req, &result)
		if lastErr == nil {
			c.recordSuccess()
			return &result, nil
		}
		// Only retry on transient errors (connection refused, timeout, 5xx).
		if !isTransient(lastErr) {
			break
		}
	}
	c.recordFailure()
	return nil, fmt.Errorf("strategysvc backtest: %w", lastErr)
}

// isTransient returns true for network-level or 5xx errors that are worth retrying.
func isTransient(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "status 5") ||
		strings.Contains(msg, "EOF") ||
		strings.Contains(msg, "broken pipe") ||
		strings.Contains(msg, "reset by peer")
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
		// Truncate response body to prevent traceback leakage to clients.
	body := string(respBytes)
	if len(body) > 200 {
		body = body[:200] + "..."
	}
	return fmt.Errorf("post %s: status %d: %s", path, resp.StatusCode, body)
	}

	if err := json.Unmarshal(respBytes, respBody); err != nil {
		return fmt.Errorf("unmarshal response: %w (body: %s)", err, string(respBytes[:min(len(respBytes), 200)]))
	}
	return nil
}
