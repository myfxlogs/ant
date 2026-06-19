// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const MarketRegime = {
  "strategy": {
    "marketRegime": {
      "detectFailed": "市场環境檢測失敗",
      "detectSuccess": "市场環境檢測完成",
      "form": {
        "accountId": "帳戶 ID",
        "accountIdPlaceholder": "输入 MT 帳戶 UUID",
        "accountIdRequired": "請輸入帳戶 ID",
        "klineCount": "K 线数量",
        "submit": "开始檢測",
        "symbol": "品種",
        "symbolPlaceholder": "例如 EURUSD",
        "symbolRequired": "請輸入品種",
        "timeframe": "週期",
        "title": "檢測參數"
      },
      "result": {
        "confidence": "置信度",
        "features": "特征",
        "modelVersion": "模型版本",
        "recordId": "记录 ID",
        "status": "狀態",
        "strategyFamilies": "策略族类",
        "title": "檢測结果"
      },
      "ruleVersionAlert": "当前使用基于规则的檢測模型 rule-v1，由即時 K 线市场資料驱动。",
      "subtitle": "从历史 K 线資料分析趋势方向、波動率狀態和價格效率，分类当前市场環境。",
      "title": "市场環境檢測"
    }
  }
} as const;
export default MarketRegime;
