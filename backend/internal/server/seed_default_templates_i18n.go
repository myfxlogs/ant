package server

import "anttrader/internal/model"

// getTemplateI18nMap returns the i18n dictionaries for all built-in preset
// strategies keyed by canonical template name (same key used in the switch in
// seed_default_templates.go). Supported locales match the frontend:
// zh-CN (default), zh-TW, en, ja, vi.
//
// Keeping this table out of seed_default_templates.go keeps that file under
// the project's 800-line guideline while making it easy to add a new language
// (add one entry per map).
func getTemplateI18nMap() map[string]*model.TemplateI18n {
	return map[string]*model.TemplateI18n{
		"双均线交叉策略": {
			Name: map[string]string{
				"zh-CN": "均线交叉策略",
				"zh-TW": "均線交叉策略",
				"en":    "MA Crossover",
				"ja":    "移動平均クロス",
				"vi":    "Giao cắt MA",
			},
			Description: map[string]string{
				"zh-CN": "经典双均线交叉策略。快线上穿慢线做多，下穿做空。",
				"zh-TW": "經典雙均線交叉策略。快線上穿慢線做多，下穿做空。",
				"en":    "Classic dual moving average crossover. Buy when fast MA crosses above slow MA, sell when it crosses below.",
				"ja":    "デュアル移動平均クロス戦略。短期MAが長期MAを上抜けで買い、下抜けで売り。",
				"vi":    "Chiến lược giao cắt hai đường MA. Mua khi MA nhanh cắt lên MA chậm, bán khi cắt xuống.",
			},
			Params: map[string]model.TemplateParamI18n{
				"fast_period": {Label: map[string]string{
					"zh-CN": "快线周期", "zh-TW": "快線週期", "en": "Fast MA period", "ja": "短期MA期間", "vi": "Chu kỳ MA nhanh",
				}},
				"slow_period": {Label: map[string]string{
					"zh-CN": "慢线周期", "zh-TW": "慢線週期", "en": "Slow MA period", "ja": "長期MA期間", "vi": "Chu kỳ MA chậm",
				}},
			},
		},
		"RSI超买超卖策略": {
			Name: map[string]string{
				"zh-CN": "RSI 超卖反弹",
				"zh-TW": "RSI 超賣反彈",
				"en":    "RSI Oversold Bounce",
				"ja":    "RSI 売られすぎ反発",
				"vi":    "RSI Bật lại quá bán",
			},
			Description: map[string]string{
				"zh-CN": "当 RSI 跌破超卖线后回升时入场做多，RSI 超买时平仓。",
				"zh-TW": "當 RSI 跌破超賣線後回升時入場做多，RSI 超買時平倉。",
				"en":    "Enter long when RSI bounces from oversold territory. Exit when RSI reaches overbought.",
				"ja":    "RSIが売られすぎから反発した際に買い、買われすぎに達したら決済。",
				"vi":    "Mở vị thế mua khi RSI bật lên từ vùng quá bán, đóng khi RSI đạt vùng quá mua.",
			},
			Params: map[string]model.TemplateParamI18n{
				"rsi_period": {Label: map[string]string{
					"zh-CN": "RSI 周期", "zh-TW": "RSI 週期", "en": "RSI period", "ja": "RSI期間", "vi": "Chu kỳ RSI",
				}},
				"oversold": {Label: map[string]string{
					"zh-CN": "超卖阈值", "zh-TW": "超賣閾值", "en": "Oversold level", "ja": "売られすぎ水準", "vi": "Ngưỡng quá bán",
				}},
				"overbought": {Label: map[string]string{
					"zh-CN": "超买阈值", "zh-TW": "超買閾值", "en": "Overbought level", "ja": "買われすぎ水準", "vi": "Ngưỡng quá mua",
				}},
			},
		},
		"MACD策略": {
			Name: map[string]string{
				"zh-CN": "MACD 背离策略",
				"zh-TW": "MACD 背離策略",
				"en":    "MACD Divergence",
				"ja":    "MACD ダイバージェンス",
				"vi":    "Phân kỳ MACD",
			},
			Description: map[string]string{
				"zh-CN": "价格创新低但 MACD 未创新低（底背离）时做多，顶背离时做空。",
				"zh-TW": "價格創新低但 MACD 未創新低（底背離）時做多，頂背離時做空。",
				"en":    "Enter long on bullish MACD divergence (price makes lower low but MACD doesn't). Short on bearish divergence.",
				"ja":    "価格は安値更新だがMACDが更新しない強気ダイバージェンスで買い、弱気ダイバージェンスで売り。",
				"vi":    "Mua khi có phân kỳ tăng của MACD (giá tạo đáy mới nhưng MACD thì không). Bán khi phân kỳ giảm.",
			},
			Params: map[string]model.TemplateParamI18n{
				"fast_period": {Label: map[string]string{
					"zh-CN": "快线周期", "zh-TW": "快線週期", "en": "Fast EMA period", "ja": "短期EMA期間", "vi": "Chu kỳ EMA nhanh",
				}},
				"slow_period": {Label: map[string]string{
					"zh-CN": "慢线周期", "zh-TW": "慢線週期", "en": "Slow EMA period", "ja": "長期EMA期間", "vi": "Chu kỳ EMA chậm",
				}},
				"signal_period": {Label: map[string]string{
					"zh-CN": "信号线周期", "zh-TW": "訊號線週期", "en": "Signal line period", "ja": "シグナル線期間", "vi": "Chu kỳ đường tín hiệu",
				}},
			},
		},
		"布林带收缩突破": {
			Name: map[string]string{
				"zh-CN": "布林带收缩突破",
				"zh-TW": "布林帶收縮突破",
				"en":    "Bollinger Band Squeeze Breakout",
				"ja":    "ボリンジャーバンド スクイーズブレイク",
				"vi":    "Bứt phá sau nén Bollinger",
			},
			Description: map[string]string{
				"zh-CN": "布林带收缩后突破上轨做多、下轨做空，利用波动率爆发行情。",
				"zh-TW": "布林帶收縮後突破上軌做多、下軌做空，利用波動率爆發行情。",
				"en":    "Trade breakouts after Bollinger Band squeezes. Long on upper band breakout, short on lower.",
				"ja":    "ボリンジャーバンドが収縮した後の上抜けで買い、下抜けで売り。ボラティリティ拡大を狙う。",
				"vi":    "Giao dịch theo bứt phá sau khi Bollinger thu hẹp. Mua khi phá biên trên, bán khi phá biên dưới.",
			},
			Params: map[string]model.TemplateParamI18n{
				"bb_period": {Label: map[string]string{
					"zh-CN": "布林带周期", "zh-TW": "布林帶週期", "en": "Bollinger period", "ja": "ボリンジャー期間", "vi": "Chu kỳ Bollinger",
				}},
				"bb_std": {Label: map[string]string{
					"zh-CN": "标准差倍数", "zh-TW": "標準差倍數", "en": "Std-dev multiplier", "ja": "標準偏差倍率", "vi": "Hệ số độ lệch chuẩn",
				}},
				"squeeze_threshold": {Label: map[string]string{
					"zh-CN": "收缩阈值", "zh-TW": "收縮閾值", "en": "Squeeze threshold", "ja": "スクイーズ閾値", "vi": "Ngưỡng nén",
				}},
			},
		},
		"布林带均值回归": {
			Name: map[string]string{
				"zh-CN": "布林带均值回归",
				"zh-TW": "布林帶均值回歸",
				"en":    "BB Mean Reversion",
				"ja":    "ボリンジャー平均回帰",
				"vi":    "Hồi quy trung bình Bollinger",
			},
			Description: map[string]string{
				"zh-CN": "价格触碰布林带下轨时做多，触碰上轨时平仓。适合震荡市。",
				"zh-TW": "價格觸碰布林帶下軌時做多，觸碰上軌時平倉。適合震盪市。",
				"en":    "Buy when price touches lower Bollinger Band, sell at upper band. Best for ranging markets.",
				"ja":    "価格が下部バンドに触れたら買い、上部バンドで決済。レンジ相場向き。",
				"vi":    "Mua khi giá chạm biên dưới Bollinger, bán khi chạm biên trên. Phù hợp thị trường đi ngang.",
			},
			Params: map[string]model.TemplateParamI18n{
				"bb_period": {Label: map[string]string{
					"zh-CN": "布林带周期", "zh-TW": "布林帶週期", "en": "Bollinger period", "ja": "ボリンジャー期間", "vi": "Chu kỳ Bollinger",
				}},
				"bb_std": {Label: map[string]string{
					"zh-CN": "标准差倍数", "zh-TW": "標準差倍數", "en": "Std-dev multiplier", "ja": "標準偏差倍率", "vi": "Hệ số độ lệch chuẩn",
				}},
			},
		},
		"放量突破": {
			Name: map[string]string{
				"zh-CN": "放量突破",
				"zh-TW": "放量突破",
				"en":    "Volume Breakout",
				"ja":    "出来高ブレイクアウト",
				"vi":    "Bứt phá khối lượng",
			},
			Description: map[string]string{
				"zh-CN": "当价格突破近期高点且成交量放大时入场，结合 ATR 思路控制风险。",
				"zh-TW": "當價格突破近期高點且成交量放大時入場，結合 ATR 思路控制風險。",
				"en":    "Enter when price breaks above recent high with above-average volume. ATR-based risk management.",
				"ja":    "直近高値を出来高拡大とともに上抜けで買い。ATRでリスク管理。",
				"vi":    "Vào lệnh khi giá phá đỉnh gần nhất kèm khối lượng tăng. Quản trị rủi ro theo ATR.",
			},
			Params: map[string]model.TemplateParamI18n{
				"lookback": {Label: map[string]string{
					"zh-CN": "回看周期", "zh-TW": "回看週期", "en": "Lookback", "ja": "参照期間", "vi": "Số chu kỳ nhìn lại",
				}},
				"volume_multiplier": {Label: map[string]string{
					"zh-CN": "放量倍数", "zh-TW": "放量倍數", "en": "Volume multiplier", "ja": "出来高倍率", "vi": "Hệ số khối lượng",
				}},
			},
		},
		"海龟交易法": {
			Name: map[string]string{
				"zh-CN": "海龟交易法",
				"zh-TW": "海龜交易法",
				"en":    "Turtle Trading",
				"ja":    "タートル・トレーディング",
				"vi":    "Giao dịch Rùa",
			},
			Description: map[string]string{
				"zh-CN": "经典趋势跟踪策略。突破 20 日高点入场，跌破 10 日低点出场，可结合 ATR 仓位管理。",
				"zh-TW": "經典趨勢跟蹤策略。突破 20 日高點入場，跌破 10 日低點出場，可結合 ATR 倉位管理。",
				"en":    "Classic trend following. Enter on 20-day high breakout, exit on 10-day low. ATR-based position sizing.",
				"ja":    "古典的トレンドフォロー。20日高値ブレイクで参入、10日安値割れで撤退。ATRでポジションサイズ。",
				"vi":    "Chiến lược theo xu hướng kinh điển. Vào lệnh khi phá đỉnh 20 ngày, thoát khi thủng đáy 10 ngày. Kích thước vị thế theo ATR.",
			},
			Params: map[string]model.TemplateParamI18n{
				"entry_period": {Label: map[string]string{
					"zh-CN": "入场通道周期", "zh-TW": "入場通道週期", "en": "Entry channel period", "ja": "エントリー期間", "vi": "Chu kỳ kênh vào lệnh",
				}},
				"exit_period": {Label: map[string]string{
					"zh-CN": "退出通道周期", "zh-TW": "退出通道週期", "en": "Exit channel period", "ja": "エグジット期間", "vi": "Chu kỳ kênh thoát lệnh",
				}},
			},
		},
		"网格交易": {
			Name: map[string]string{
				"zh-CN": "网格交易",
				"zh-TW": "網格交易",
				"en":    "Grid Trading",
				"ja":    "グリッドトレード",
				"vi":    "Giao dịch lưới",
			},
			Description: map[string]string{
				"zh-CN": "在指定价格区间设置等距买卖网格，震荡行情中自动低买高卖。",
				"zh-TW": "在指定價格區間設置等距買賣網格，震盪行情中自動低買高賣。",
				"en":    "Place buy/sell orders at regular intervals within a price range. Profits from range-bound markets.",
				"ja":    "指定した価格帯に等間隔で売買注文を配置。レンジ相場で安く買い高く売る。",
				"vi":    "Đặt lệnh mua/bán đều nhau trong một vùng giá. Kiếm lợi nhuận trong thị trường đi ngang.",
			},
			Params: map[string]model.TemplateParamI18n{
				"grid_count": {Label: map[string]string{
					"zh-CN": "网格数量", "zh-TW": "網格數量", "en": "Grid count", "ja": "グリッド数", "vi": "Số lưới",
				}},
				"lower_price": {Label: map[string]string{
					"zh-CN": "下边界价格", "zh-TW": "下邊界價格", "en": "Lower price", "ja": "下限価格", "vi": "Giá biên dưới",
				}},
				"upper_price": {Label: map[string]string{
					"zh-CN": "上边界价格", "zh-TW": "上邊界價格", "en": "Upper price", "ja": "上限価格", "vi": "Giá biên trên",
				}},
				"lot": {Label: map[string]string{
					"zh-CN": "下单手数", "zh-TW": "下單手數", "en": "Lot", "ja": "発注ロット", "vi": "Khối lượng",
				}},
			},
		},
		"马丁格尔加仓": {
			Name: map[string]string{
				"zh-CN": "马丁格尔加仓",
				"zh-TW": "馬丁格爾加倉",
				"en":    "Martingale",
				"ja":    "マーチンゲール",
				"vi":    "Martingale",
			},
			Description: map[string]string{
				"zh-CN": "亏损后按倍数加仓，盈利则重置。高风险，需严控最大层数与总敞口。",
				"zh-TW": "虧損後按倍數加倉，盈利則重置。高風險，需嚴控最大層數與總曝險。",
				"en":    "Scale position size after losses by a multiplier; reset on profit. High risk, strict level cap required.",
				"ja":    "損失後に倍数でナンピン、利益でリセット。高リスク、最大段数を厳格に制限。",
				"vi":    "Tăng kích thước vị thế theo hệ số sau khi lỗ; reset khi có lãi. Rủi ro cao, bắt buộc giới hạn số tầng.",
			},
			Params: map[string]model.TemplateParamI18n{
				"base_lot": {Label: map[string]string{
					"zh-CN": "基础手数", "zh-TW": "基礎手數", "en": "Base lot", "ja": "基準ロット", "vi": "Khối lượng gốc",
				}},
				"multiplier": {Label: map[string]string{
					"zh-CN": "加倍倍数", "zh-TW": "加倍倍數", "en": "Multiplier", "ja": "倍率", "vi": "Hệ số nhân",
				}},
				"max_levels": {Label: map[string]string{
					"zh-CN": "最大加仓层数", "zh-TW": "最大加倉層數", "en": "Max levels", "ja": "最大段数", "vi": "Số tầng tối đa",
				}},
				"adverse_price_step": {Label: map[string]string{
					"zh-CN": "反向价距", "zh-TW": "反向價距", "en": "Adverse price step", "ja": "逆行価格幅", "vi": "Bước giá ngược",
				}},
			},
		},
		"定投策略 (DCA)": {
			Name: map[string]string{
				"zh-CN": "定投策略 (DCA)",
				"zh-TW": "定投策略 (DCA)",
				"en":    "Dollar Cost Averaging",
				"ja":    "ドルコスト平均法 (DCA)",
				"vi":    "Trung bình giá (DCA)",
			},
			Description: map[string]string{
				"zh-CN": "定期等额买入，长期平摊成本。适合 BTC/ETH 等长期看好的资产。",
				"zh-TW": "定期等額買入，長期平攤成本。適合 BTC/ETH 等長期看好的資產。",
				"en":    "Buy a fixed amount at regular intervals to average out the cost basis over time.",
				"ja":    "一定間隔で一定額を買付け、長期的に平均取得単価を下げる。",
				"vi":    "Mua đều đặn một lượng cố định theo chu kỳ để bình quân giá vốn dài hạn.",
			},
			Params: map[string]model.TemplateParamI18n{
				"interval_hours": {Label: map[string]string{
					"zh-CN": "间隔小时", "zh-TW": "間隔小時", "en": "Interval hours", "ja": "間隔(時間)", "vi": "Khoảng giờ",
				}},
				"lot": {Label: map[string]string{
					"zh-CN": "下单手数", "zh-TW": "下單手數", "en": "Lot", "ja": "発注ロット", "vi": "Khối lượng",
				}},
			},
		},
	}
}
