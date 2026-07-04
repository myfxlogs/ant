// Auto-generated from proto/ant/v1/i18n/strategy_gen_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyGen = {
  "strategy": {
    "gen": {
      "chat": {
        "discuss": "💬 議論",
        "generate": "⚡ 生成",
        "repair": "🔧 修復",
        "revise": "✏️ 修正"
      },
      "feedback": {
        "heading": "📊 バックテスト結果",
        "placeholder": "フィードバックを入力（例：「攻め過ぎ」「損切り追加」）"
      },
      "metrics": {
        "maxDrawdown": "最大DD",
        "return": "リターン",
        "sharpe": "シャープレシオ",
        "trades": "取引",
        "winRate": "勝率"
      },
      "backtestMsg": "バックテストタスク作成済",
      "backtestStarted": "バックテスト開始",
      "clarifyTitle": "確認事項:",
      "done": "完了",
      "generating": "生成中...",
      "placeholder": "作成する取引戦略を説明してください。例：「EURUSD 1Hのボリンジャーバンド平均回帰戦略」",
      "regenerate": "再生成",
      "reset": "最初から",
      "send": "ストラテジー生成",
      "template": "テンプレート",
      "title": "ストラテジー生成",
      "useDefaults": "デフォルトで続行",
      "useDefaultsHint": "AIは汎用テンプレートパラメータを使用して成略を生成します",
      "feedbackInputPlaceholder": "結果に不満がありますか？フィードバックを入力して戦略を改善（例：「攻め過ぎなので、1%の損切りを追加」）",
      "planTitle": "AI戦略プランナー",
      "planAnalyzing": "分析中",
      "planErrorTag": "エラー",
      "planReset": "最初から",
      "planCardTitle": "AI実行計画",
      "planEdit": "編集",
      "planEditCancel": "キャンセル",
      "planConfirmBtn": "確認してコード生成",
      "planSendBtn": "分析して計画を作成",
      "planSymbolWarn": "⚠️ 先に上部で銘柄と時間枠を選択してください",
      "planSymbolOk": "{symbol} · {timeframe}",
      "planPrerequisiteMsg": "戦略を生成する前に、上部ツールバーで取引銘柄（例：EURUSD）と時間枠（例：1H）を選択してください。これがないとバックテストが実行できません。",
      "execTitle": "AI実行中",
      "execRunning": "実行中",
      "execDone": "完了",
      "execBackToPlan": "計画に戻る",
      "execPlanLabel": "実行計画",
      "execComplianceTool": "コンプライアンスチェック",
      "execBacktestTool": "バックテスト",
      "execToolRunning": "{tool}を実行中...",
      "execFeedbackTitle": "💬 AIとの会話を続ける",
      "execFeedbackHint": "自然言語でAIに指示してください — 問題の説明、改善依頼、アイデアの議論ができます。",
      "execChipLowerDd": "ドローダウン低減",
      "execChipRaiseReturn": "リターン向上",
      "execChipTightenSl": "損切り強化",
      "execChipLongOnly": "ロングのみ",
      "execSendFeedback": "AIに送信",
      "execClear": "クリア",
      "execApplyCode": "エディタに適用",
      "execSkipNoSymbol": "バックテストスキップ：銘柄未選択",
      "execSkipNoCode": "バックテストスキップ：コードが空",
      "validating": "コンプライアンスチェック",
      "noMarketData": "マーケットデータがありません",
      "noMarketDataHint": "取引口座と銘柄を選択してチャートにデータを読み込んでから、再試行してください。"
    }
  }
} as const;
export default StrategyGen;
