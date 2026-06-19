// Auto-generated from proto/ant/v1/i18n/strategy_default_templates_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyDefaultTemplates = {
  "strategy": {
    "defaultTemplates": {
      "forceBuy": {
        "description": "For verifying the order pipeline: always returns buy on each execution, reads lot from context/params as volume",
        "name": "Force BUY Test"
      },
      "maCross": {
        "description": "Buy when fast MA crosses above slow MA, sell when it crosses below",
        "name": "Dual MA Crossover Strategy"
      },
      "macd": {
        "description": "Buy on MACD golden cross, sell on death cross",
        "name": "MACD Strategy"
      },
      "rsi": {
        "description": "Buy when RSI < 30 (oversold), sell when RSI > 70 (overbought)",
        "name": "RSI Overbought/Oversold Strategy"
      }
    }
  }
} as const;
export default StrategyDefaultTemplates;
