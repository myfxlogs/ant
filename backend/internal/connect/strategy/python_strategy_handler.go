package strategy

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/notification"
	"anttrader/internal/repository"
	"anttrader/internal/pglisten"
)

// PythonStrategyServer implements ant.v1.PythonStrategyServiceHandler.
type PythonStrategyServer struct {
	backtestRepo        *repository.BacktestRunRepository
	log                 *zap.Logger
	connectClient       antv1c.PythonStrategyServiceClient // ConnectRPC to Python service
	backtestClient      antv1c.BacktestServiceClient
	marketDataRepo      *repository.MarketDataRepository
	pgListen            *pglisten.Listener
	notifSender         *notification.Sender
	onBacktestComplete  func(ctx context.Context, run *repository.BacktestRun) // auto-gate hook
}

func (s *PythonStrategyServer) SetMarketDataRepo(r *repository.MarketDataRepository) {
	s.marketDataRepo = r
}

var _ antv1c.PythonStrategyServiceHandler = (*PythonStrategyServer)(nil)

func NewPythonStrategyServer(backtestRepo *repository.BacktestRunRepository, log *zap.Logger) *PythonStrategyServer {
	return &PythonStrategyServer{backtestRepo: backtestRepo, log: log}
}

func (s *PythonStrategyServer) SetConnectClient(c antv1c.PythonStrategyServiceClient) { s.connectClient = c }
func (s *PythonStrategyServer) SetBacktestClient(c antv1c.BacktestServiceClient)     { s.backtestClient = c }
func (s *PythonStrategyServer) SetNotificationSender(ns *notification.Sender)         { s.notifSender = ns }
func (s *PythonStrategyServer) SetOnBacktestComplete(fn func(context.Context, *repository.BacktestRun)) { s.onBacktestComplete = fn }

// userIDRequire extracts and validates the authenticated user ID from context.
func userIDRequire(ctx context.Context) (uuid.UUID, error) {
	id, err := uuid.Parse(interceptor.GetUserID(ctx))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("authentication required"))
	}
	return id, nil
}

func (s *PythonStrategyServer) Execute(ctx context.Context, req *connect.Request[antv1.ExecuteStrategyRequest]) (*connect.Response[antv1.ExecuteStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid // authorization verified; downstream uses AccountId directly

	if s.connectClient != nil {
		resp, err := s.connectClient.Execute(ctx, connect.NewRequest(&antv1.ExecuteStrategyRequest{
			Code:      req.Msg.Code,
			AccountId: req.Msg.AccountId,
			Symbol:    req.Msg.Symbol,
			Timeframe: req.Msg.Timeframe,
		}))
		if err != nil {
			s.log.Warn("python execute failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(resp.Msg), nil
	}
	return connect.NewResponse(&antv1.ExecuteStrategyResponse{
		Success: false,
	}), nil
}

func (s *PythonStrategyServer) Validate(ctx context.Context, req *connect.Request[antv1.ValidateStrategyRequest]) (*connect.Response[antv1.ValidateStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid

	if s.connectClient != nil {
		resp, err := s.connectClient.Validate(ctx, connect.NewRequest(&antv1.ValidateStrategyRequest{
			Code: req.Msg.Code,
		}))
		if err != nil {
			s.log.Warn("python validate failed", zap.Error(err))
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(resp.Msg), nil
	}
	return connect.NewResponse(&antv1.ValidateStrategyResponse{
		Valid: true,
	}), nil
}

func (s *PythonStrategyServer) Backtest(ctx context.Context, req *connect.Request[antv1.BacktestStrategyRequest]) (*connect.Response[antv1.BacktestStrategyResponse], error) {
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}
	_ = uid
	if s.backtestClient == nil {
		return connect.NewResponse(&antv1.BacktestStrategyResponse{Success: false}), nil
	}
	klines, _ := fetchKlines(ctx, s.marketDataRepo, req.Msg.Symbol, req.Msg.Timeframe)
	resp, err := s.backtestClient.RunBacktest(ctx, connect.NewRequest(
		&antv1.ExecuteBacktestRequest{
			StrategyCode:   req.Msg.Code,
			Symbol:         req.Msg.Symbol,
			Timeframe:      req.Msg.Timeframe,
			InitialCapital: 10000,
			Klines:         klines,
		}))
	if err != nil {
		s.log.Warn("python backtest failed", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !resp.Msg.GetSuccess() {
		return connect.NewResponse(&antv1.BacktestStrategyResponse{Success: false}), nil
	}
	met := resp.Msg.GetMetrics()
	return connect.NewResponse(&antv1.BacktestStrategyResponse{
		Success:     true,
		EquityCurve: resp.Msg.GetEquityCurve(),
		Metrics:     mapMetrics(met),
	}), nil
}

// mapMetrics converts ExecuteBacktestMetrics → BacktestMetrics.
func mapMetrics(met *antv1.ExecuteBacktestMetrics) *antv1.BacktestMetrics {
	if met == nil {
		return &antv1.BacktestMetrics{}
	}
	return &antv1.BacktestMetrics{
		TotalReturn: met.GetTotalReturn(), AnnualReturn: met.GetAnnualReturn(),
		MaxDrawdown: met.GetMaxDrawdown(), SharpeRatio: met.GetSharpeRatio(),
		WinRate: met.GetWinRate(), ProfitFactor: met.GetProfitFactor(),
		TotalTrades: met.GetTotalTrades(), WinningTrades: met.GetWinningTrades(),
		LosingTrades: met.GetLosingTrades(), AverageProfit: met.GetAverageProfit(),
		AverageLoss: met.GetAverageLoss(),
	}
}


func (s *PythonStrategyServer) GetTemplates(_ context.Context, _ *connect.Request[emptypb.Empty]) (*connect.Response[antv1.GetPythonTemplatesResponse], error) {
	return connect.NewResponse(&antv1.GetPythonTemplatesResponse{Templates: builtinTemplates()}), nil
}

func builtinTemplates() []*antv1.PythonTemplate {
	return []*antv1.PythonTemplate{
		// ── Vectorized (run_dataframe) templates — recommended default ──
		{Name: "MA Crossover (Vectorized)", Description: "双均线交叉策略 — 矢量模式", Code: `my_indicator_name = "MA Crossover"
my_indicator_description = "Buy when fast EMA crosses above slow EMA, sell on reverse cross."

# @param fast_len int 10 Fast EMA period
# @param slow_len int 30 Slow EMA period

# @strategy stopLossPct 0.03
# @strategy takeProfitPct 0.06
# @strategy entryPct 0.25
# @strategy tradeDirection both

df = df.copy()

fast_len = int(params.get('fast_len', 10))
slow_len = int(params.get('slow_len', 30))

ema_fast = df['close'].ewm(span=fast_len, adjust=False).mean()
ema_slow = df['close'].ewm(span=slow_len, adjust=False).mean()

raw_buy = (ema_fast > ema_slow) & (ema_fast.shift(1) <= ema_slow.shift(1))
raw_sell = (ema_fast < ema_slow) & (ema_fast.shift(1) >= ema_slow.shift(1))

def edge(s):
    s = s.fillna(False).astype(bool)
    return s & ~s.shift(1).fillna(False)

df['buy'] = edge(raw_buy)
df['sell'] = edge(raw_sell)

buy_marks = [df['low'].iloc[i] * 0.995 if df['buy'].iloc[i] else None for i in range(len(df))]
sell_marks = [df['high'].iloc[i] * 1.005 if df['sell'].iloc[i] else None for i in range(len(df))]

output = {
    "name": my_indicator_name,
    "plots": [
        {"name": "EMA Fast", "data": ema_fast.fillna(0).tolist(), "color": "#1890ff", "overlay": True},
        {"name": "EMA Slow", "data": ema_slow.fillna(0).tolist(), "color": "#faad14", "overlay": True},
    ],
    "signals": [
        {"type": "buy", "text": "B", "data": buy_marks, "color": "#00E676"},
        {"type": "sell", "text": "S", "data": sell_marks, "color": "#FF5252"},
    ]
}`},
		{Name: "RSI Mean Reversion (Vectorized)", Description: "RSI超买超卖反转策略 — 矢量模式", Code: `my_indicator_name = "RSI Mean Reversion"
my_indicator_description = "Buy when RSI drops below oversold, sell when RSI rises above overbought."

# @param rsi_len int 14 RSI period
# @param rsi_oversold float 30 Oversold threshold
# @param rsi_overbought float 70 Overbought threshold

# @strategy stopLossPct 0.02
# @strategy takeProfitPct 0.04
# @strategy entryPct 0.25
# @strategy tradeDirection both

df = df.copy()

rsi_len = int(params.get('rsi_len', 14))
rsi_oversold = float(params.get('rsi_oversold', 30))
rsi_overbought = float(params.get('rsi_overbought', 70))

delta = df['close'].diff()
gain = delta.clip(lower=0).ewm(alpha=1/rsi_len, adjust=False).mean()
loss = (-delta.clip(upper=0)).ewm(alpha=1/rsi_len, adjust=False).mean()
rs = gain / loss.replace(0, float('nan'))
rsi = 100 - (100 / (1 + rs))

raw_buy = (rsi < rsi_oversold) & (rsi.shift(1) >= rsi_oversold)
raw_sell = (rsi > rsi_overbought) & (rsi.shift(1) <= rsi_overbought)

def edge(s):
    s = s.fillna(False).astype(bool)
    return s & ~s.shift(1).fillna(False)

df['buy'] = edge(raw_buy)
df['sell'] = edge(raw_sell)

buy_marks = [df['low'].iloc[i] * 0.995 if df['buy'].iloc[i] else None for i in range(len(df))]
sell_marks = [df['high'].iloc[i] * 1.005 if df['sell'].iloc[i] else None for i in range(len(df))]

output = {
    "name": my_indicator_name,
    "plots": [
        {"name": "RSI", "data": rsi.fillna(0).tolist(), "color": "#722ed1", "overlay": False},
    ],
    "signals": [
        {"type": "buy", "text": "B", "data": buy_marks, "color": "#00E676"},
        {"type": "sell", "text": "S", "data": sell_marks, "color": "#FF5252"},
    ]
}`},
		{Name: "Bollinger Breakout (Vectorized)", Description: "布林带突破策略 — 矢量模式", Code: `my_indicator_name = "Bollinger Breakout"
my_indicator_description = "Buy on breakout above upper band, sell on breakdown below lower band."

# @param bb_len int 20 Bollinger period
# @param bb_std float 2.0 Number of standard deviations

# @strategy stopLossPct 0.03
# @strategy takeProfitPct 0.06
# @strategy entryPct 0.25
# @strategy tradeDirection both

df = df.copy()

bb_len = int(params.get('bb_len', 20))
bb_std = float(params.get('bb_std', 2.0))

ma = df['close'].rolling(bb_len).mean()
std = df['close'].rolling(bb_len).std()
upper = ma + bb_std * std
lower = ma - bb_std * std

raw_buy = (df['close'] > upper) & (df['close'].shift(1) <= upper.shift(1))
raw_sell = (df['close'] < lower) & (df['close'].shift(1) >= lower.shift(1))

def edge(s):
    s = s.fillna(False).astype(bool)
    return s & ~s.shift(1).fillna(False)

df['buy'] = edge(raw_buy)
df['sell'] = edge(raw_sell)

buy_marks = [df['low'].iloc[i] * 0.995 if df['buy'].iloc[i] else None for i in range(len(df))]
sell_marks = [df['high'].iloc[i] * 1.005 if df['sell'].iloc[i] else None for i in range(len(df))]

output = {
    "name": my_indicator_name,
    "plots": [
        {"name": "MA", "data": ma.fillna(0).tolist(), "color": "#1890ff", "overlay": True},
        {"name": "Upper", "data": upper.fillna(0).tolist(), "color": "#52c41a", "overlay": True},
        {"name": "Lower", "data": lower.fillna(0).tolist(), "color": "#f5222d", "overlay": True},
    ],
    "signals": [
        {"type": "buy", "text": "B", "data": buy_marks, "color": "#00E676"},
        {"type": "sell", "text": "S", "data": sell_marks, "color": "#FF5252"},
    ]
}`},

		// ── Event-driven (run_context) templates — legacy compat ──
		{Name: "MA Crossover", Description: "双均线交叉策略 — 事件驱动模式", Code: `# @param fast_period 10 range=5:50:5
# @param slow_period 30 range=10:100:10
def run(context):
    p = context.get('params', {})
    fast_period = int(p.get('fast_period', 10))
    slow_period = int(p.get('slow_period', 30))
    prices = context['close']
    if len(prices) < slow_period + 1:
        return {'signal': 'hold', 'volume': 0}
    fast_ma = sum(prices[-fast_period:]) / fast_period
    slow_ma = sum(prices[-slow_period:]) / slow_period
    pos = context.get('position')
    if fast_ma > slow_ma:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'buy', 'volume': 1.0}
    elif fast_ma < slow_ma:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'hold', 'volume': 0}
        if pos:
            return {'signal': 'close', 'volume': 0}
        return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
		{Name: "RSI Mean Reversion", Description: "RSI超买超卖反转策略 — 事件驱动模式", Code: `# @param rsi_period 14 range=7:28:7
# @param oversold 30 range=20:40:5
# @param overbought 70 range=60:80:5
def run(context):
    p = context.get('params', {})
    rsi_period = int(p.get('rsi_period', 14))
    oversold = int(p.get('oversold', 30))
    overbought = int(p.get('overbought', 70))
    prices = context['close']
    if len(prices) < rsi_period + 1:
        return {'signal': 'hold', 'volume': 0}
    deltas = [prices[i] - prices[i-1] for i in range(1, len(prices))]
    gains = [d if d > 0 else 0 for d in deltas[-rsi_period:]]
    losses = [-d if d < 0 else 0 for d in deltas[-rsi_period:]]
    avg_gain = sum(gains) / rsi_period if gains else 0
    avg_loss = sum(losses) / rsi_period if losses else 1e-10
    rs = avg_gain / avg_loss
    rsi = 100 - (100 / (1 + rs))
    pos = context.get('position')
    if rsi < oversold:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'buy', 'volume': 1.0}
    elif rsi > overbought:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
		{Name: "Bollinger Breakout", Description: "布林带突破策略 — 事件驱动模式", Code: `# @param bb_period 20 range=10:50:10
# @param bb_std 2.0 range=1.0:4.0:0.5
def run(context):
    p = context.get('params', {})
    bb_period = int(p.get('bb_period', 20))
    bb_std = float(p.get('bb_std', 2.0))
    import math
    prices = context['close']
    if len(prices) < bb_period + 1:
        return {'signal': 'hold', 'volume': 0}
    window = prices[-bb_period:]
    ma = sum(window) / bb_period
    variance = sum((x - ma) ** 2 for x in window) / bb_period
    std = math.sqrt(variance)
    upper = ma + bb_std * std
    lower = ma - bb_std * std
    last_price = prices[-1]
    pos = context.get('position')
    if last_price > upper:
        if pos and pos.get('side') == 'sell':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'buy', 'volume': 1.0}
    elif last_price < lower:
        if pos and pos.get('side') == 'buy':
            return {'signal': 'close', 'volume': 0}
        if not pos:
            return {'signal': 'sell', 'volume': 1.0}
    return {'signal': 'hold', 'volume': 0}`},
	}
}

func (s *PythonStrategyServer) SetPgListen(l *pglisten.Listener) { s.pgListen = l }
