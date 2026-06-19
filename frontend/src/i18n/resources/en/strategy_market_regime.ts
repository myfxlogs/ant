// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyMarketRegime = {
  "strategy": {
    "marketRegime": {
      "form": {
        "accountId": "Account ID",
        "accountIdPlaceholder": "MT account UUID",
        "accountIdRequired": "Account ID is required",
        "klineCount": "K-line Count",
        "submit": "Start Detection",
        "symbol": "Symbol",
        "symbolPlaceholder": "EURUSD",
        "symbolRequired": "Symbol is required",
        "timeframe": "Timeframe",
        "title": "Detection Parameters"
      },
      "result": {
        "confidence": "Confidence",
        "features": "Features",
        "modelVersion": "Model Version",
        "recordId": "Record ID",
        "status": "Status",
        "strategyFamilies": "Strategy Families",
        "title": "Detection Result"
      },
      "detectFailed": "Market regime detection failed",
      "detectSuccess": "Market regime detection completed",
      "ruleVersionAlert": "Currently using rule-based detection model rule-v1, driven by real-time K-line market data.",
      "subtitle": "Analyzes trend direction, volatility regime, and price efficiency from historical K-line data to classify current market conditions.",
      "title": "Market Regime Detection"
    }
  }
} as const;
export default StrategyMarketRegime;
