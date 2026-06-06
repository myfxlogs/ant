package strategy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
	"connectrpc.com/connect"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
)

// ObjectiveScoreServer proxies CalculateObjectiveScore to the Python strategy-service.
// The Python implementation at /api/objective-score contains the actual calculation
// logic (RSI, MACD, MA crossover scoring). This handler bridges ConnectRPC ↔ Python REST.
// TODO: Replace bridge with native ConnectRPC endpoint on Python side (matching
// backtest_connect.py pattern), then use ObjectiveScoreServiceClient here.
type ObjectiveScoreServer struct {
	baseURL string
	httpc   *http.Client
	log     *zap.Logger
}

var _ antv1c.ObjectiveScoreServiceHandler = (*ObjectiveScoreServer)(nil)

// NewObjectiveScoreServer creates a handler that delegates scoring to the Python service.
func NewObjectiveScoreServer(strategyServiceURL string, log *zap.Logger) *ObjectiveScoreServer {
	return &ObjectiveScoreServer{
		baseURL: strategyServiceURL,
		httpc:   &http.Client{Timeout: 30 * time.Second},
		log:     log,
	}
}

// --- JSON transport types (mirror Python pydantic schemas) ---

type scoreReq struct {
	Symbol    string       `json:"symbol"`
	Timeframe string       `json:"timeframe"`
	Klines    []klineEntry `json:"klines"`
}

type klineEntry struct {
	OpenTime   string  `json:"open_time"`
	CloseTime  string  `json:"close_time"`
	OpenPrice  float64 `json:"open_price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
	ClosePrice float64 `json:"close_price"`
	Volume     float64 `json:"volume"`
}

type scoreResp struct {
	Decision       string      `json:"decision"`
	OverallScore   float64     `json:"overall_score"`
	TechnicalScore float64     `json:"technical_score"`
	Signals        *objSignals `json:"signals"`
}

type objSignals struct {
	RSI  objRSI  `json:"rsi"`
	MACD objMACD `json:"macd"`
	MA   objMA   `json:"ma"`
}

type objRSI  struct{ Value float64 `json:"value"`; Signal string `json:"signal"` }
type objMACD struct {
	Value, SignalLine, Histogram float64 `json:"value,signal_line,histogram"`
	Signal, Trend                string  `json:"signal,trend"`
}
type objMA   struct {
	MA5, MA10, MA20 float64 `json:"ma5,ma10,ma20"`
	Trend           string  `json:"trend"`
}

// --- Handler ---

func (s *ObjectiveScoreServer) CalculateObjectiveScore(
	ctx context.Context,
	req *connect.Request[antv1.ObjectiveScoreRequest],
) (*connect.Response[antv1.ObjectiveScoreResponse], error) {
	// Extract user for audit; authorization is verified by authInterceptor.
	_ = interceptor.GetUserID(ctx)

	body, err := marshalScoreRequest(req.Msg)
	if err != nil {
		return nil, err
	}
	pyResp, err := s.callPython(ctx, body)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(pyResp.ToProto()), nil
}

// --- helpers ---

func marshalScoreRequest(m *antv1.ObjectiveScoreRequest) ([]byte, error) {
	pyReq := scoreReq{Symbol: m.Symbol, Timeframe: m.Timeframe, Klines: make([]klineEntry, len(m.Klines))}
	for i, k := range m.Klines {
		pyReq.Klines[i] = klineEntry{
			OpenTime: k.OpenTime, CloseTime: k.CloseTime,
			OpenPrice: k.OpenPrice, HighPrice: k.HighPrice,
			LowPrice: k.LowPrice, ClosePrice: k.ClosePrice, Volume: k.Volume,
		}
	}
	body, err := json.Marshal(pyReq)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("marshal objective score request: %w", err))
	}
	return body, nil
}

func (s *ObjectiveScoreServer) callPython(ctx context.Context, body []byte) (*scoreResp, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/api/objective-score", bytes.NewReader(body))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("create objective score http request: %w", err))
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.httpc.Do(httpReq)
	if err != nil {
		s.log.Warn("objective score proxy failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("objective score service unreachable: %w", err))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, connect.NewError(connect.CodeUnavailable,
			fmt.Errorf("objective score service returned status %d", resp.StatusCode))
	}
	var pyResp scoreResp
	if err := json.NewDecoder(resp.Body).Decode(&pyResp); err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("decode objective score response: %w", err))
	}
	return &pyResp, nil
}

func (r *scoreResp) ToProto() *antv1.ObjectiveScoreResponse {
	out := &antv1.ObjectiveScoreResponse{
		Decision: r.Decision, OverallScore: r.OverallScore, TechnicalScore: r.TechnicalScore,
	}
	if r.Signals != nil {
		out.Signals = &antv1.ObjectiveSignals{
			Rsi: &antv1.RSISignal{Value: r.Signals.RSI.Value, Signal: r.Signals.RSI.Signal},
			Macd: &antv1.MACDSignal{
				Value: r.Signals.MACD.Value, SignalLine: r.Signals.MACD.SignalLine,
				Histogram: r.Signals.MACD.Histogram, Signal: r.Signals.MACD.Signal,
				Trend: r.Signals.MACD.Trend,
			},
			Ma: &antv1.MASignal{
				Ma5: r.Signals.MA.MA5, Ma10: r.Signals.MA.MA10,
				Ma20: r.Signals.MA.MA20, Trend: r.Signals.MA.Trend,
			},
		}
	}
	return out
}
