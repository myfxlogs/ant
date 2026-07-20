CREATE TABLE strategy_parameter_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_key VARCHAR(50) UNIQUE NOT NULL,  -- "trend_following_ema_cross"
    template_type VARCHAR(30) NOT NULL,         -- trend_following / mean_reversion / breakout / arbitrage
    name_i18n JSONB NOT NULL DEFAULT '{}',      -- {"en":"EMA Crossover Trend Following","zh-cn":"EMA交叉趋势跟踪"}
    description_i18n JSONB NOT NULL DEFAULT '{}',
    parameters JSONB NOT NULL,                   -- parameter definition schema
    default_risk_level VARCHAR(15) DEFAULT 'moderate',
    icon VARCHAR(50),                           -- frontend icon identifier
    sort_order INT DEFAULT 0,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed templates
INSERT INTO strategy_parameter_templates (template_key, template_type, name_i18n, description_i18n, parameters, icon, sort_order) VALUES
('trend_ema_cross', 'trend_following',
 '{"en":"EMA Crossover Trend","zh-cn":"EMA 双均线交叉趋势","ja":"EMAクロス追随","vi":"EMA Xu hướng cắt nhau"}',
 '{"en":"Follows trend using fast/slow EMA crossover signals.","zh-cn":"利用快慢 EMA 交叉信号跟踪趋势。"}',
 '{"fast_period": {"type":"int","min":5,"max":50,"default":12,"label":{"en":"Fast Period","zh-cn":"快线周期"}},
   "slow_period": {"type":"int","min":20,"max":200,"default":26,"label":{"en":"Slow Period","zh-cn":"慢线周期"}},
   "stop_loss_pips": {"type":"int","min":10,"max":200,"default":50,"label":{"en":"Stop Loss (pips)","zh-cn":"止损点数"}},
   "take_profit_pips": {"type":"int","min":20,"max":500,"default":100,"label":{"en":"Take Profit (pips)","zh-cn":"止盈点数"}}}',
 'trending-up', 1),
('mean_rev_bollinger', 'mean_reversion',
 '{"en":"Bollinger Band Mean Reversion","zh-cn":"布林带均值回归","ja":"ボリンジャーバンド逆張り","vi":"Bollinger Hồi quy trung bình"}',
 '{"en":"Trades reversions to the mean using Bollinger Band extremes.","zh-cn":"利用布林带极值进行均值回归交易。"}',
 '{"period": {"type":"int","min":10,"max":50,"default":20,"label":{"en":"Period","zh-cn":"周期"}},
   "std_dev": {"type":"float","min":1.0,"max":3.0,"default":2.0,"label":{"en":"Std Dev","zh-cn":"标准差倍数"}},
   "stop_loss_pips": {"type":"int","min":10,"max":200,"default":30,"label":{"en":"Stop Loss (pips)","zh-cn":"止损点数"}}}',
 'swap', 2),
('breakout_donchian', 'breakout',
 '{"en":"Donchian Channel Breakout","zh-cn":"唐奇安通道突破","ja":"ドンチャンチャネル突破","vi":"Donchian Đột phá kênh"}',
 '{"en":"Enters on Donchian channel breakout with volume confirmation.","zh-cn":"唐奇安通道突破入场，成交量确认。"}',
 '{"period": {"type":"int","min":10,"max":55,"default":20,"label":{"en":"Channel Period","zh-cn":"通道周期"}},
   "stop_loss_pips": {"type":"int","min":10,"max":200,"default":40,"label":{"en":"Stop Loss (pips)","zh-cn":"止损点数"}},
   "take_profit_pips": {"type":"int","min":20,"max":500,"default":80,"label":{"en":"Take Profit (pips)","zh-cn":"止盈点数"}}}',
 'arrow-up', 3);
