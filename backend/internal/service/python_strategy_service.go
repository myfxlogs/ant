package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/pkg/logger"
)

type PythonStrategyService struct {
	baseURL        string
	httpClient     *http.Client
	backtestClient *http.Client
}

func NewPythonStrategyService(baseURL string) *PythonStrategyService {
	backtestTimeoutSeconds := getEnvInt("ANTRADER_STRATEGY_BACKTEST_HTTP_TIMEOUT_SECONDS", 300)
	return &PythonStrategyService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		backtestClient: &http.Client{
			Timeout: time.Duration(backtestTimeoutSeconds) * time.Second,
		},
	}
}

type KlineDataPython struct {
	OpenTime   time.Time `json:"open_time"`
	CloseTime  time.Time `json:"close_time"`
	OpenPrice  float64   `json:"open_price"`
	HighPrice  float64   `json:"high_price"`
	LowPrice   float64   `json:"low_price"`
	ClosePrice float64   `json:"close_price"`
	Volume     float64   `json:"volume"`
}

type ObjectiveKlinePython struct {
	OpenTime   string  `json:"open_time"`
	CloseTime  string  `json:"close_time"`
	OpenPrice  float64 `json:"open_price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
	ClosePrice float64 `json:"close_price"`
	Volume     float64 `json:"volume"`
}

type ObjectiveScoreRequestPython struct {
	Symbol    string                 `json:"symbol"`
	Timeframe string                 `json:"timeframe"`
	Klines    []ObjectiveKlinePython `json:"klines"`
}

type RSISignalPython struct {
	Value  float64 `json:"value"`
	Signal string  `json:"signal"`
}

type MACDSignalPython struct {
	Value      float64 `json:"value"`
	SignalLine float64 `json:"signal_line"`
	Histogram  float64 `json:"histogram"`
	Signal     string  `json:"signal"`
	Trend      string  `json:"trend"`
}

type MASignalPython struct {
	MA5   float64 `json:"ma5"`
	MA10  float64 `json:"ma10"`
	MA20  float64 `json:"ma20"`
	Trend string  `json:"trend"`
}

type ObjectiveSignalsPython struct {
	RSI  *RSISignalPython  `json:"rsi"`
	MACD *MACDSignalPython `json:"macd"`
	MA   *MASignalPython   `json:"ma"`
}

type ObjectiveScoreResponsePython struct {
	Decision       string                  `json:"decision"`
	OverallScore   float64                 `json:"overall_score"`
	TechnicalScore float64                 `json:"technical_score"`
	Signals        *ObjectiveSignalsPython `json:"signals"`
}

type QuoteTickPython struct {
	Time   time.Time `json:"time"`
	Bid    float64   `json:"bid"`
	Ask    float64   `json:"ask"`
	Symbol string    `json:"symbol"`
}

type MarketDataPython struct {
	Symbol       string            `json:"symbol"`
	Timeframe    string            `json:"timeframe"`
	Klines       []KlineDataPython `json:"klines"`
	CurrentPrice float64           `json:"current_price,omitempty"`
}

type StrategyExecuteRequestPython struct {
	StrategyID   string                 `json:"strategy_id"`
	StrategyCode string                 `json:"strategy_code"`
	MarketData   MarketDataPython       `json:"market_data"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

type TradeSignalPython struct {
	Signal     string  `json:"signal"`
	Symbol     string  `json:"symbol"`
	Price      float64 `json:"price,omitempty"`
	Volume     float64 `json:"volume,omitempty"`
	StopLoss   float64 `json:"stop_loss,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
	RiskLevel  string  `json:"risk_level"`
}

type StrategyExecuteResponsePython struct {
	Success         bool                   `json:"success"`
	Signal          *TradeSignalPython     `json:"signal,omitempty"`
	Error           string                 `json:"error,omitempty"`
	Runtime         map[string]interface{} `json:"runtime,omitempty"`
	ExecutionTimeMs float64                `json:"execution_time_ms"`
	Logs            []string               `json:"logs"`
}

type StrategyValidateRequestPython struct {
	StrategyCode string `json:"strategy_code"`
}

type StrategyValidateResponsePython struct {
	Valid      bool                     `json:"valid"`
	Errors     []string                 `json:"errors"`
	Warnings   []string                 `json:"warnings"`
	Parameters []map[string]interface{} `json:"parameters"`
}

type BacktestRequestPython struct {
	StrategyID        string            `json:"strategy_id"`
	StrategyCode      string            `json:"strategy_code"`
	Symbol            string            `json:"symbol"`
	Timeframe         string            `json:"timeframe"`
	StartDate         time.Time         `json:"start_date"`
	EndDate           time.Time         `json:"end_date"`
	InitialCapital    float64           `json:"initial_capital"`
	Commission        float64           `json:"commission"`
	Spread            float64           `json:"spread"`
	SwapRate          float64           `json:"swap_rate"`
	ServerTimezone    string            `json:"server_timezone"`
	RolloverHour      int               `json:"rollover_hour"`
	TripleSwapWeekday int               `json:"triple_swap_weekday"`
	SlippageMode      string            `json:"slippage_mode"`
	SlippageRate      float64           `json:"slippage_rate"`
	SlippageSeed      int64             `json:"slippage_seed"`
	Klines            []KlineDataPython `json:"klines"`
	Ticks             []QuoteTickPython `json:"ticks,omitempty"`
	// Phase B2 multi-symbol inputs. ``ExtraSymbols`` lists the secondary symbol
	// names; ``KlinesBySymbol`` carries the K-line payload for each symbol
	// (including the primary). The Python side treats these as optional and
	// falls back to the legacy single-symbol path when both are empty.
	ExtraSymbols   []string                     `json:"extra_symbols,omitempty"`
	KlinesBySymbol map[string][]KlineDataPython `json:"klines_by_symbol,omitempty"`
}

type BacktestCostModel struct {
	CommissionRate    float64
	SpreadRate        float64
	SwapRate          float64
	ServerTimezone    string
	RolloverHour      int
	TripleSwapWeekday int
	SlippageMode      string
	SlippageRate      float64
	SlippageSeed      int64
}

func DefaultBacktestCostModelFromEnv() BacktestCostModel {
	return BacktestCostModel{
		CommissionRate:    getEnvFloat64("ANTRADER_BACKTEST_COMMISSION_RATE", 0.0001),
		SpreadRate:        getEnvFloat64("ANTRADER_BACKTEST_SPREAD_RATE", 0.0),
		SwapRate:          getEnvFloat64("ANTRADER_BACKTEST_SWAP_RATE", 0.0),
		ServerTimezone:    os.Getenv("ANTRADER_SERVER_TIMEZONE"),
		RolloverHour:      getEnvInt("ANTRADER_ROLLOVER_HOUR", 0),
		TripleSwapWeekday: getEnvInt("ANTRADER_TRIPLE_SWAP_WEEKDAY", 3),
		SlippageMode:      os.Getenv("ANTRADER_BACKTEST_SLIPPAGE_MODE"),
		SlippageRate:      getEnvFloat64("ANTRADER_BACKTEST_SLIPPAGE_RATE", 0.0),
		SlippageSeed:      getEnvInt64("ANTRADER_BACKTEST_SLIPPAGE_SEED", 0),
	}
}

type BacktestMetricsPython struct {
	TotalReturn   float64 `json:"total_return"`
	AnnualReturn  float64 `json:"annual_return"`
	MaxDrawdown   float64 `json:"max_drawdown"`
	SharpeRatio   float64 `json:"sharpe_ratio"`
	WinRate       float64 `json:"win_rate"`
	ProfitFactor  float64 `json:"profit_factor"`
	TotalTrades   int     `json:"total_trades"`
	WinningTrades int     `json:"winning_trades"`
	LosingTrades  int     `json:"losing_trades"`
	AverageProfit float64 `json:"average_profit"`
	AverageLoss   float64 `json:"average_loss"`
}

type RiskAssessmentPython struct {
	Score      int      `json:"score"`
	Level      string   `json:"level"`
	Reasons    []string `json:"reasons"`
	Warnings   []string `json:"warnings"`
	IsReliable bool     `json:"is_reliable"`
}

type BacktestTradePython struct {
	Ticket     int64   `json:"ticket"`
	Side       string  `json:"side"`
	Volume     float64 `json:"volume"`
	OpenTs     int64   `json:"open_ts"`
	OpenPrice  float64 `json:"open_price"`
	CloseTs    int64   `json:"close_ts"`
	ClosePrice float64 `json:"close_price"`
	PnL        float64 `json:"pnl"`
	Commission float64 `json:"commission"`
	Reason     string  `json:"reason"`
}

// MarshalMetricsWithTrades returns the metrics JSON with an extra "trades"
// field embedded so we can persist orders alongside metrics in the existing
// JSONB column without a DB migration. Decoders that target the typed
// BacktestMetricsPython struct will silently drop the extra key.
func MarshalMetricsWithTrades(m *BacktestMetricsPython, trades []BacktestTradePython) []byte {
	wrapper := struct {
		*BacktestMetricsPython
		Trades []BacktestTradePython `json:"trades,omitempty"`
	}{m, trades}
	out, _ := json.Marshal(wrapper)
	return out
}

type BacktestResponsePython struct {
	Success        bool                   `json:"success"`
	Metrics        *BacktestMetricsPython `json:"metrics"`
	RiskAssessment *RiskAssessmentPython  `json:"risk_assessment"`
	EquityCurve    []float64              `json:"equity_curve"`
	Events         []map[string]any       `json:"events"`
	Trades         []BacktestTradePython  `json:"trades"`
	Error          string                 `json:"error"`
}

func getEnvFloat64(key string, def float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func getEnvInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getEnvInt64(key string, def int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func (s *PythonStrategyService) ExecuteStrategy(
	ctx context.Context,
	strategyID uuid.UUID,
	strategyCode string,
	klines []*KlineResponse,
	symbol string,
	timeframe string,
	execContext map[string]interface{},
) (*StrategyExecuteResponsePython, error) {
	pythonKlines := make([]KlineDataPython, len(klines))
	for i, k := range klines {
		openTime, _ := time.Parse(time.RFC3339, k.OpenTime)
		closeTime, _ := time.Parse(time.RFC3339, k.CloseTime)
		pythonKlines[i] = KlineDataPython{
			OpenTime:   openTime,
			CloseTime:  closeTime,
			OpenPrice:  k.OpenPrice,
			HighPrice:  k.HighPrice,
			LowPrice:   k.LowPrice,
			ClosePrice: k.ClosePrice,
			Volume:     float64(k.Volume),
		}
	}

	req := StrategyExecuteRequestPython{
		StrategyID:   strategyID.String(),
		StrategyCode: strategyCode,
		MarketData: MarketDataPython{
			Symbol:    symbol,
			Timeframe: timeframe,
			Klines:    pythonKlines,
		},
	}
	if execContext != nil {
		req.Context = execContext
	}

	var resp StrategyExecuteResponsePython
	err := s.doRequest(ctx, "POST", "/api/strategy/execute", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *PythonStrategyService) ValidateStrategy(
	ctx context.Context,
	strategyCode string,
) (*StrategyValidateResponsePython, error) {
	req := StrategyValidateRequestPython{
		StrategyCode: strategyCode,
	}

	var resp StrategyValidateResponsePython
	err := s.doRequest(ctx, "POST", "/api/strategy/validate", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *PythonStrategyService) CalculateObjectiveScore(ctx context.Context, req ObjectiveScoreRequestPython) (*ObjectiveScoreResponsePython, error) {
	var resp ObjectiveScoreResponsePython
	if err := s.doRequest(ctx, "POST", "/api/objective-score", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (s *PythonStrategyService) RunBacktest(
	ctx context.Context,
	strategyID uuid.UUID,
	strategyCode string,
	klines []*KlineResponse,
	ticks []QuoteTickPython,
	symbol string,
	timeframe string,
	initialCapital float64,
	cost *BacktestCostModel,
	extraKlines map[string][]*KlineResponse,
) (*BacktestResponsePython, error) {
	if len(klines) == 0 {
		return nil, errors.New("no kline data provided")
	}

	resolvedCost := DefaultBacktestCostModelFromEnv()
	if cost != nil {
		resolvedCost = *cost
	}

	pythonKlines := make([]KlineDataPython, len(klines))
	var startDate, endDate time.Time
	for i, k := range klines {
		openTime, _ := time.Parse(time.RFC3339, k.OpenTime)
		closeTime, _ := time.Parse(time.RFC3339, k.CloseTime)
		pythonKlines[i] = KlineDataPython{
			OpenTime:   openTime,
			CloseTime:  closeTime,
			OpenPrice:  k.OpenPrice,
			HighPrice:  k.HighPrice,
			LowPrice:   k.LowPrice,
			ClosePrice: k.ClosePrice,
			Volume:     float64(k.Volume),
		}
		if i == 0 || openTime.Before(startDate) {
			startDate = openTime
		}
		if i == 0 || openTime.After(endDate) {
			endDate = openTime
		}
	}
	// Build multi-symbol payload (Phase B2). The primary symbol's klines are
	// always included in the map so the Python engine has a consistent keyed
	// view. Empty map/list keeps the legacy single-symbol path active.
	var (
		extraSymbols   []string
		klinesBySymbol map[string][]KlineDataPython
	)
	if len(extraKlines) > 0 {
		extraSymbols = make([]string, 0, len(extraKlines))
		klinesBySymbol = make(map[string][]KlineDataPython, len(extraKlines)+1)
		klinesBySymbol[symbol] = pythonKlines
		for sym, ks := range extraKlines {
			if sym == "" || sym == symbol {
				continue
			}
			converted := make([]KlineDataPython, 0, len(ks))
			for _, k := range ks {
				if k == nil {
					continue
				}
				openTime, _ := time.Parse(time.RFC3339, k.OpenTime)
				closeTime, _ := time.Parse(time.RFC3339, k.CloseTime)
				converted = append(converted, KlineDataPython{
					OpenTime:   openTime,
					CloseTime:  closeTime,
					OpenPrice:  k.OpenPrice,
					HighPrice:  k.HighPrice,
					LowPrice:   k.LowPrice,
					ClosePrice: k.ClosePrice,
					Volume:     float64(k.Volume),
				})
			}
			if len(converted) == 0 {
				continue
			}
			klinesBySymbol[sym] = converted
			extraSymbols = append(extraSymbols, sym)
		}
		if len(extraSymbols) == 0 {
			// All extras filtered out — fall back to single-symbol path.
			klinesBySymbol = nil
		}
	}

	req := BacktestRequestPython{
		StrategyID:        strategyID.String(),
		StrategyCode:      strategyCode,
		Symbol:            symbol,
		Timeframe:         timeframe,
		StartDate:         startDate,
		EndDate:           endDate,
		InitialCapital:    initialCapital,
		Commission:        resolvedCost.CommissionRate,
		Spread:            resolvedCost.SpreadRate,
		SwapRate:          resolvedCost.SwapRate,
		ServerTimezone:    resolvedCost.ServerTimezone,
		RolloverHour:      resolvedCost.RolloverHour,
		TripleSwapWeekday: resolvedCost.TripleSwapWeekday,
		SlippageMode:      resolvedCost.SlippageMode,
		SlippageRate:      resolvedCost.SlippageRate,
		SlippageSeed:      resolvedCost.SlippageSeed,
		Klines:            pythonKlines,
		Ticks:             ticks,
		ExtraSymbols:      extraSymbols,
		KlinesBySymbol:    klinesBySymbol,
	}

	var resp BacktestResponsePython
	err := s.doRequestWithClient(ctx, s.backtestClient, "POST", "/api/backtest", req, &resp)
	if err != nil {
		return nil, err
	}

	return &resp, nil
}

// --- 记忆系统 ---

type MemoryQueryRequestPython struct {
	Symbol       string `json:"symbol"`
	Timeframe    string `json:"timeframe"`
	StrategyCode string `json:"strategy_code"`
	N            int    `json:"n"`
}

type MemoryEntryPython struct {
	Situation string  `json:"situation"`
	Advice    string  `json:"advice"`
	Score     float64 `json:"score"`
}

type MemoryQueryResponsePython struct {
	Memories []MemoryEntryPython `json:"memories"`
}

type MemoryRecordRequestPython struct {
	Symbol       string                 `json:"symbol"`
	Timeframe    string                 `json:"timeframe"`
	StrategyCode string                 `json:"strategy_code"`
	Metrics      map[string]interface{} `json:"metrics"`
	ExtraAdvice  string                 `json:"extra_advice,omitempty"`
}

type MemoryRecordResponsePython struct {
	Success      bool `json:"success"`
	TotalEntries int  `json:"total_entries"`
}

// QueryMemory 在策略生成前查询相似历史回测的建议，返回格式化的提示文本
func (s *PythonStrategyService) QueryMemory(
	ctx context.Context,
	symbol, timeframe, strategyCode string,
	n int,
) ([]MemoryEntryPython, error) {
	if n <= 0 {
		n = 3
	}
	req := MemoryQueryRequestPython{
		Symbol:       symbol,
		Timeframe:    timeframe,
		StrategyCode: strategyCode,
		N:            n,
	}
	var resp MemoryQueryResponsePython
	if err := s.doRequest(ctx, "POST", "/api/memory/query", req, &resp); err != nil {
		return nil, err
	}
	return resp.Memories, nil
}

// RecordMemory 手动写入一条回测记忆（回测由外部触发时使用）
func (s *PythonStrategyService) RecordMemory(
	ctx context.Context,
	symbol, timeframe, strategyCode string,
	metrics map[string]interface{},
	extraAdvice string,
) error {
	req := MemoryRecordRequestPython{
		Symbol:       symbol,
		Timeframe:    timeframe,
		StrategyCode: strategyCode,
		Metrics:      metrics,
		ExtraAdvice:  extraAdvice,
	}
	var resp MemoryRecordResponsePython
	return s.doRequest(ctx, "POST", "/api/memory/record", req, &resp)
}

// FormatMemoriesAsPrompt 将查询结果格式化为可注入 AI 提示词的文本
func FormatMemoriesAsPrompt(memories []MemoryEntryPython) string {
	if len(memories) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## 历史相似策略回测经验（供参考）\n\n")
	for i, m := range memories {
		sb.WriteString(fmt.Sprintf("### 经验 %d（相似度 %.0f%%）\n", i+1, m.Score*100))
		sb.WriteString(m.Advice)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func (s *PythonStrategyService) doRequest(ctx context.Context, method, path string, reqBody, respBody interface{}) error {
	return s.doRequestWithClient(ctx, s.httpClient, method, path, reqBody, respBody)
}

func (s *PythonStrategyService) doRequestWithClient(ctx context.Context, client *http.Client, method, path string, reqBody, respBody interface{}) error {
	url := s.baseURL + path

	var bodyReader io.Reader
	if reqBody != nil {
		jsonBody, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		logger.Error("Python strategy service error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBytes)))
		return fmt.Errorf("service error: status %d, body: %s", resp.StatusCode, string(respBytes))
	}

	if err := json.Unmarshal(respBytes, respBody); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return nil
}
