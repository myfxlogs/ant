// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyMarketRegime = {
  "strategy": {
    "marketRegime": {
      "form": {
        "accountId": "アカウントID",
        "accountIdPlaceholder": "MTアカウントUUID",
        "accountIdRequired": "アカウントIDが必要です",
        "klineCount": "K線数",
        "submit": "検出開始",
        "symbol": "銘柄",
        "symbolPlaceholder": "例如 EURUSD",
        "symbolRequired": "銘柄を選択してください",
        "timeframe": "時間足",
        "title": "検出パラメータ"
      },
      "result": {
        "confidence": "信頼度",
        "features": "特徴量",
        "modelVersion": "モデルバージョン",
        "recordId": "レコードID",
        "status": "状態",
        "strategyFamilies": "ストラテジーファミリー",
        "title": "検出結果"
      },
      "detectFailed": "市場レジーム検出失敗",
      "detectSuccess": "市場レジーム検出完了",
      "ruleVersionAlert": "現在、ルールベース検出モデル rule-v1 を使用。リアルタイムK線データ駆動。",
      "subtitle": "過去のK線データからトレンド方向、ボラティリティレジーム、価格効率性を分析し、現在の市場状態を分類します。",
      "title": "市場レジーム検出"
    }
  }
} as const;
export default StrategyMarketRegime;
