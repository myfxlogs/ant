// Auto-generated from proto/ant/v1/i18n/strategy_default_templates_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const DefaultTemplates = {
  "strategy": {
    "defaultTemplates": {
      "forceBuy": {
        "description": "用于验证订单管道：每次执行始终返回买入信号，从 context/params 读取手数",
        "name": "强制买入测试"
      },
      "maCross": {
        "description": "快线上穿慢线时买入，下穿时卖出",
        "name": "双均线交叉策略"
      },
      "macd": {
        "description": "MACD 金叉买入，死叉卖出",
        "name": "MACD 策略"
      },
      "rsi": {
        "description": "RSI < 30 (超卖) 时买入，RSI > 70 (超买) 时卖出",
        "name": "RSI 超买超卖策略"
      }
    }
  }
} as const;
export default DefaultTemplates;
