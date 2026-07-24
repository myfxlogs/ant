// Auto-generated from proto/ant/v1/i18n/strategy_default_templates_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyDefaultTemplates = {
  "strategy": {
    "defaultTemplates": {
      "forceBuy": {
        "description": "用於驗證訂單管道：每次執行始終返回買入訊號，從 context/params 讀取手數",
        "name": "強制買入測試"
      },
      "maCross": {
        "description": "快線上穿慢線時買入，下穿時賣出",
        "name": "雙均線交叉策略"
      },
      "macd": {
        "description": "MACD 金叉買入，死叉賣出",
        "name": "MACD 策略"
      },
      "rsi": {
        "description": "RSI < 30 (超賣) 時買入，RSI > 70 (超買) 時賣出",
        "name": "RSI 超買超賣策略"
      }
    }
  }
} as const;
export default StrategyDefaultTemplates;
