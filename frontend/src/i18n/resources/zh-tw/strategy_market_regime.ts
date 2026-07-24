// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyMarketRegime = {
  "strategy": {
    "marketRegime": {
      "form": {
        "accountId": "賬戶 ID",
        "accountIdPlaceholder": "輸入 MT 賬戶 UUID",
        "accountIdRequired": "請輸入賬戶 ID",
        "klineCount": "K 線數量",
        "submit": "開始檢測",
        "symbol": "品種",
        "symbolPlaceholder": "例如 EURUSD",
        "symbolRequired": "請選擇品種",
        "timeframe": "週期",
        "title": "檢測引數"
      },
      "result": {
        "confidence": "置信度",
        "features": "特徵",
        "modelVersion": "模型版本",
        "recordId": "記錄 ID",
        "status": "狀態",
        "strategyFamilies": "策略族類",
        "title": "檢測結果"
      },
      "detectFailed": "市場環境檢測失敗",
      "detectSuccess": "市場環境檢測完成",
      "ruleVersionAlert": "當前使用基於規則的檢測模型 rule-v1，由實時 K 線市場資料驅動。",
      "subtitle": "從歷史 K 線資料分析趨勢方向、波動率狀態和價格效率，分類當前市場環境。",
      "title": "市場環境檢測"
    }
  }
} as const;
export default StrategyMarketRegime;
