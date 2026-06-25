// Auto-generated from proto/ant/v1/i18n/strategy_code_assist_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeAssist = {
  "strategy": {
    "codeAssist": {
      "paramDescriptions": {
        "confidence": "シグナル信頼度閾値（0-1）。この値未満のシグナルは無視。",
        "emaPeriod": "EMA（指数移動平均）ルックバック（バー数）。",
        "fastPeriod": "高速期間（バー数）。MACD/デュアルMAで使用。小さいほど反応早い。",
        "genericPercent": "パーセンテージ/比率パラメータ（例: 1は1%）。",
        "genericPeriod": "指標計算に使用するルックバックウィンドウ（バー数）。",
        "lotSize": "注文サイズ（ロット/数量）。大きいほどリスク大。",
        "maxLoss": "1取引あたりの最大損失（資産比。0.01 = 1%）。",
        "riskLevel": "リスクレベル（低/中/高）。ポジションサイズと損切り/利確幅を制御。",
        "rsiPeriod": "RSIルックバック（バー数）。典型的な値: 14。",
        "signalPeriod": "シグナル期間（バー数）。MACD DIF/DEAの平滑化長。",
        "slowPeriod": "低速期間（バー数）。MACD/デュアルMAで使用。大きいほど滑らか。",
        "smaPeriod": "SMA（単純移動平均）ルックバック（バー数）。",
        "stopLoss": "損切り距離（%）。不利方向にこの幅動いたら決済。",
        "takeProfit": "利確距離（%）。有利方向にこの幅動いたら決済。",
        "threshold": "シグナルをトリガーする閾値。正確な意味はストラテジーロジックに依存。"
      },
      "aiReviseTitle": "AIアシスタント — コード修正",
      "applyAllSuggestions": "推奨デフォルトを適用",
      "codeEmpty": "修正するコードがまだありません。",
      "codeUpdated": "コード更新済。保存前に再検証してください。",
      "defaultLabel": "デフォルト",
      "enterInstruction": "変更内容を説明してください。",
      "explain": "コード説明",
      "fillRequiredParams": "必須パラメータを入力してください: {{keys}}",
      "generatePlaceholder": "ストラテジー要件を説明...",
      "noPython": "AIがPythonブロックを返しませんでした。言い換えて再試行。",
      "optionalParamsDesc": "これらのパラメータは既にコードのデフォルトがあります。空白でデフォルト使用。入力値は今回の実行のみに適用され保存済ストラテジーは変更されません。",
      "optionalParamsTitle": "オプションパラメータ",
      "required": "必須",
      "requiredParamsDesc": "ストラテジーが読取るパラメータですがデフォルトが未提供。保存前に入力してください。",
      "requiredParamsTitle": "必須パラメータ",
      "reviseInputPlaceholder": "例: SMA(20)をEMA(50)に置換し1%損切り追加。",
      "reviseSend": "AI に修正を依頼",
      "saveBlockedNotValidated": "最初に「コード検証」をクリックしてください。検証通過まで保存無効。",
      "suggested": "推奨",
      "tabAI": "AI修正",
      "validateToSeeParams": "コードを検証して戦略パラメータを表示",
      "tabExplain": "コード説明"
    }
  }
} as const;
export default StrategyCodeAssist;
