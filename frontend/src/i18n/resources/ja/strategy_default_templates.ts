// Auto-generated from proto/ant/v1/i18n/strategy_default_templates_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyDefaultTemplates = {
  "strategy": {
    "defaultTemplates": {
      "forceBuy": {
        "description": "注文パイプライン検証用：毎回買いを返し、context/paramsから数量読込",
        "name": "強制BUYテスト"
      },
      "maCross": {
        "description": "高速MAが低速MAを上抜けで買い、下抜けで売り",
        "name": "デュアルMAクロス戦略"
      },
      "macd": {
        "description": "MACDゴールデンクロスで買い、デッドクロスで売り",
        "name": "MACD戦略"
      },
      "rsi": {
        "description": "RSI < 30 (売られ過ぎ) で買い、RSI > 70 (買われ過ぎ) で売り",
        "name": "RSI売買戦略"
      }
    }
  }
} as const;
export default StrategyDefaultTemplates;
