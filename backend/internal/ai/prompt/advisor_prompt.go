package prompt

import "fmt"

// AdvisorSystemPrompt 交易建议系统提示词
const AdvisorSystemPrompt = `你是一个专业的交易分析师。根据提供的市场数据，给出交易建议。

输出格式要求（必须严格遵循JSON格式）：
{
  "signal": "buy/sell/hold",
  "confidence": 0.0-1.0,
  "reason": "分析理由",
  "entry_price": 建议入场价,
  "stop_loss": 建议止损价,
  "take_profit": 建议止盈价,
  "risk_level": "low/medium/high"
}

分析要求：
1. 基于技术分析方法（趋势、支撑阻力、K线形态等）
2. 综合考虑价格走势和成交量
3. 给出合理的入场、止损、止盈价位
4. 评估风险等级
5. 只输出JSON，不要包含其他文字说明`

// BuildAdvisorPrompt 构建交易建议提示词
func BuildAdvisorPrompt(symbol string, klineData string) string {
	return fmt.Sprintf("品种: %s\n\nK线数据:\n%s\n\n请分析并给出交易建议。", symbol, klineData)
}

// FormatKlineData 格式化K线数据为字符串
func FormatKlineData(klines []KlineInfo) string {
	if len(klines) == 0 {
		return "无K线数据"
	}

	result := "时间,开盘价,最高价,最低价,收盘价,成交量\n"
	for _, k := range klines {
		result += fmt.Sprintf("%s,%.5f,%.5f,%.5f,%.5f,%d\n",
			k.Time, k.Open, k.High, k.Low, k.Close, k.Volume)
	}
	return result
}

// KlineInfo K线信息（用于格式化）
type KlineInfo struct {
	Time   string
	Open   float64
	High   float64
	Low    float64
	Close  float64
	Volume int64
}
