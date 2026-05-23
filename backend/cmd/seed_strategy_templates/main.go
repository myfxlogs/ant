package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"

	"anttrader/internal/config"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/pkg/logger"

	"go.uber.org/zap"
)

type defaultTemplate struct {
	Name        string
	Description string
	Code        string
	IsPublic    bool
}

func main() {
	cfgPath := os.Getenv("CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(&logger.Config{Level: cfg.Log.Level, Format: cfg.Log.Format, Output: cfg.Log.Output}); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init logger: %v\n", err)
		os.Exit(1)
	}

	sqlxDB, err := repository.NewSQLXDB(&cfg.Database)
	if err != nil {
		logger.Fatal("failed to connect db", zap.Error(err))
	}
	defer sqlxDB.Close()

	templateRepo := repository.NewStrategyTemplateRepository(sqlxDB)

	ctx := context.Background()

	userIDs, err := listUserIDs(ctx, sqlxDB)
	if err != nil {
		logger.Fatal("failed to list users", zap.Error(err))
	}

	defaults := getDefaultTemplates()
	created := 0
	skipped := 0
	failed := 0

	for _, userID := range userIDs {
		for _, tpl := range defaults {
			exists, err := templateExists(ctx, sqlxDB, userID, tpl.Name)
			if err != nil {
				failed++
				logger.Error("failed to check template existence", zap.Error(err), zap.String("user_id", userID.String()), zap.String("name", tpl.Name))
				continue
			}
			if exists {
				skipped++
				continue
			}

			m := model.NewStrategyTemplate(userID, tpl.Name, tpl.Code)
			m.Description = tpl.Description
			m.IsPublic = tpl.IsPublic
			m.Tags = []string{}

			if err := templateRepo.Create(ctx, m); err != nil {
				failed++
				logger.Error("failed to create default template", zap.Error(err), zap.String("user_id", userID.String()), zap.String("name", tpl.Name))
				continue
			}
			created++
		}
	}

	fmt.Printf("Seed completed. users=%d created=%d skipped=%d failed=%d\n", len(userIDs), created, skipped, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func listUserIDs(ctx context.Context, db interface {
	SelectContext(context.Context, interface{}, string, ...interface{}) error
}) ([]uuid.UUID, error) {
	var ids []uuid.UUID
	query := `SELECT id FROM users ORDER BY created_at ASC`
	if err := db.SelectContext(ctx, &ids, query); err != nil {
		return nil, err
	}
	return ids, nil
}

func templateExists(ctx context.Context, db interface {
	GetContext(context.Context, interface{}, string, ...interface{}) error
}, userID uuid.UUID, name string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM strategy_templates WHERE user_id = $1 AND name = $2)`
	var exists bool
	if err := db.GetContext(ctx, &exists, query, userID, name); err != nil {
		return false, err
	}
	return exists, nil
}

func getDefaultTemplates() []defaultTemplate {
	return []defaultTemplate{
		{
			Name:        "双均线交叉策略",
			Description: "当快均线上穿慢均线时买入，下穿时卖出",
			IsPublic:    true,
			Code: `# 双均线交叉策略
# 可用变量: close, open, high, low, volume, symbol
# 返回: signal字典

# 参数设置
fast_period = 10
slow_period = 20

# 数据长度检查
if len(close) < slow_period + 1:
    signal = {
        'signal': 'hold',
        'symbol': symbol,
        'price': close[-1] if len(close) > 0 else None,
        'confidence': 0.0,
        'reason': '数据不足',
        'risk_level': 'low',
    }
else:
    # 计算均线
    maFast = np.mean(close[-fast_period:])
    maSlow = np.mean(close[-slow_period:])
    ma_fast_prev = np.mean(close[-fast_period-1:-1])
    ma_slow_prev = np.mean(close[-slow_period-1:-1])

    # 判断交叉
    if maFast > maSlow and ma_fast_prev <= ma_slow_prev:
        signal = 'buy'
        reason = '金叉买入信号'
    elif maFast < maSlow and ma_fast_prev >= ma_slow_prev:
        signal = 'sell'
        reason = '死叉卖出信号'
    else:
        signal = 'hold'
        reason = '无信号'

    # 返回结果
    signal = {
        'signal': signal,
        'symbol': symbol,
        'price': close[-1],
        'confidence': 0.7,
        'reason': reason,
        'risk_level': 'medium',
        'maFast': round(maFast, 5),
        'maSlow': round(maSlow, 5)
    }`,
		},
		{
			Name:        "RSI超买超卖策略",
			Description: "RSI低于30超买区买入，高于70超卖区卖出",
			IsPublic:    true,
			Code: `# RSI超买超卖策略
# 可用变量: close, open, high, low, volume, symbol
# 返回: signal字典

# 参数设置
period = 14
oversold = 30
overbought = 70

# 计算RSI
def calculate_rsi(prices, period):
    deltas = np.diff(prices)
    gains = np.where(deltas > 0, deltas, 0)
    losses = np.where(deltas < 0, -deltas, 0)
    avgGain = np.mean(gains[-period:])
    avgLoss = np.mean(losses[-period:])
    if avgLoss == 0:
        return 100
    rs = avgGain / avgLoss
    return 100 - (100 / (1 + rs))

if len(close) < period + 1:
    rsi = None
else:
    rsi = calculate_rsi(close, period)

# 生成信号
if rsi is None:
    action = 'hold'
    reason = '数据不足'
    risk_level = 'low'
elif rsi < oversold:
    action = 'buy'
    reason = f'RSI={rsi:.2f} 超卖区买入信号'
    risk_level = 'medium'
elif rsi > overbought:
    action = 'sell'
    reason = f'RSI={rsi:.2f} 超买区卖出信号'
    risk_level = 'medium'
else:
    action = 'hold'
    reason = f'RSI={rsi:.2f} 无信号'
    risk_level = 'low'

# 返回结果
signal = {
    'signal': action,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.6,
    'reason': reason,
    'risk_level': risk_level,
    'rsi': round(rsi, 2) if rsi is not None else None
}`,
		},
		{
			Name:        "MACD策略",
			Description: "MACD金叉买入，死叉卖出",
			IsPublic:    true,
			Code: `# MACD策略
# 可用变量: close, open, high, low, volume, symbol
# 返回: signal字典

# 参数设置
fast_period = 12
slow_period = 26
signal_period = 9

# 计算EMA
def ema(prices, period):
    multiplier = 2 / (period + 1)
    ema_val = prices[0]
    for price in prices[1:]:
        ema_val = (price - ema_val) * multiplier + ema_val
    return ema_val

# 计算MACD
if len(close) < slow_period + signal_period + 2:
    signal = {
        'signal': 'hold',
        'symbol': symbol,
        'price': close[-1] if len(close) > 0 else None,
        'confidence': 0.0,
        'reason': '数据不足',
        'risk_level': 'low',
    }
else:
    ema_fast = ema(close, fast_period)
    ema_slow = ema(close, slow_period)
    macd_line = ema_fast - ema_slow

    # 简化的信号线计算
    macd_history = []
    for i in range(slow_period, len(close)):
        ef = ema(close[:i+1], fast_period)
        es = ema(close[:i+1], slow_period)
        macd_history.append(ef - es)

    signal_line = ema(macd_history[-signal_period*2:], signal_period)

    # 判断交叉
    macd_prev = macd_history[-2]
    signal_prev = signal_line

    if macd_line > signal_line and macd_prev <= signal_prev:
        signal = 'buy'
        reason = 'MACD金叉买入信号'
    elif macd_line < signal_line and macd_prev >= signal_prev:
        signal = 'sell'
        reason = 'MACD死叉卖出信号'
    else:
        signal = 'hold'
        reason = '无信号'

    # 返回结果
    signal = {
        'signal': signal,
        'symbol': symbol,
        'price': close[-1],
        'confidence': 0.65,
        'reason': reason,
        'risk_level': 'medium',
        'macd': round(macd_line, 5),
        'signal_line': round(signal_line, 5)
    }`,
		},
	}
}
