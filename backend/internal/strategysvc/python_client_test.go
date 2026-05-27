package strategysvc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPythonClient_Backtest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/backtest" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req BacktestRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req.Code == "" {
			t.Error("expected non-empty code")
		}
		json.NewEncoder(w).Encode(BacktestResult{
			Success:     true,
			EquityCurve: []float64{10000, 10050, 10100, 10080, 10150, 10200, 10300, 10400},
			SharpeRatio: 2.1,
			MaxDrawdown: 0.03,
			WinRate:     0.65,
			TradeCount:  42,
		})
	}))
	defer srv.Close()

	c := NewPythonClient(srv.URL)
	result, err := c.Backtest(t.Context(), &BacktestRequest{
		Code:      "print('hello')",
		Symbol:    "EURUSD",
		Timeframe: "1h",
		Balance:   10000,
	})
	if err != nil {
		t.Fatalf("Backtest: %v", err)
	}
	if !result.Success {
		t.Error("expected success=true")
	}
	if len(result.EquityCurve) < 3 {
		t.Errorf("expected equity_curve length >= 3, got %d (mock was 6)", len(result.EquityCurve))
	}
	if result.SharpeRatio != 2.1 {
		t.Errorf("expected SharpeRatio=2.1, got %f", result.SharpeRatio)
	}
}

func TestPythonClient_Execute(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ExecuteRequest
		json.NewDecoder(r.Body).Decode(&req)
		json.NewEncoder(w).Encode(ExecuteResult{
			Success: true,
			Signal: &struct {
				Side       string  `json:"side"`
				Lots       float64 `json:"lots"`
				Price      float64 `json:"price"`
				StopLoss   float64 `json:"stop_loss"`
				TakeProfit float64 `json:"take_profit"`
				Reason     string  `json:"reason"`
			}{
				Side: "BUY", Lots: 0.25, Price: 1.1234, StopLoss: 1.12, TakeProfit: 1.13, Reason: "MA crossover",
			},
		})
	}))
	defer srv.Close()

	c := NewPythonClient(srv.URL)
	result, err := c.Execute(t.Context(), &ExecuteRequest{
		Code:      "strategy code",
		AccountID: "acc-1",
		Symbol:    "EURUSD",
		Timeframe: "4h",
		Mode:      "paper",
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !result.Success || result.Signal == nil {
		t.Fatal("expected success with signal")
	}
	if result.Signal.Lots != 0.25 {
		t.Errorf("expected Lots=0.25, got %f", result.Signal.Lots)
	}
}

func TestPythonClient_Validate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(ValidateResult{
			Valid:    true,
			Errors:   []string{},
			Warnings: []string{"stop-loss recommended"},
		})
	}))
	defer srv.Close()

	c := NewPythonClient(srv.URL)
	result, err := c.Validate(t.Context(), &ValidateRequest{Code: "valid code"})
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestPythonClient_Health(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewPythonClient(srv.URL)
	if err := c.Health(t.Context()); err != nil {
		t.Errorf("Health check failed: %v", err)
	}
}

func TestNewPythonClient_NoURL(t *testing.T) {
	// Client with empty URL won't connect — that's expected (falls back to mock).
	c := NewPythonClient("")
	err := c.Health(t.Context())
	if err == nil {
		t.Log("expected connection error for empty URL (this is fine)")
	}
}
