// Auto-generated from proto/ant/v1/i18n/strategy_gen_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyGen = {
  "strategy": {
    "gen": {
      "chat": {
        "discuss": "💬 讨论",
        "generate": "⚡ 生成",
        "repair": "🔧 修复",
        "revise": "✏️ 修改"
      },
      "feedback": {
        "heading": "📊 回測结果",
        "placeholder": "提供反馈以迭代最佳化 (例如 “太激进”, “添加止損”)"
      },
      "metrics": {
        "maxDrawdown": "最大回撤",
        "return": "收益",
        "sharpe": "夏普",
        "trades": "交易数",
        "winRate": "勝率"
      },
      "backtestMsg": "回測任务已建立",
      "backtestStarted": "回測已啟動",
      "clarifyTitle": "确认以下细节:",
      "done": "完成",
      "generating": "生成中...",
      "placeholder": "說明您想要创建的交易策略，例如：“制作一个 EURUSD 1H 布林带均值回归策略”",
      "regenerate": "重新生成",
      "reset": "重新开始",
      "send": "生成策略",
      "template": "模板",
      "title": "策略生成",
      "useDefaults": "使用預設值继续",
      "useDefaultsHint": "AI 將使用通用參數生成策略模板",
      "feedbackInputPlaceholder": "對回測結果不滿意？輸入回饋來最佳化策略，例如：「太激進，加入 1% 停損」",
      "planTitle": "AI 策略規劃",
      "planAnalyzing": "分析中",
      "planErrorTag": "出錯",
      "planReset": "重新開始",
      "planCardTitle": "AI 執行計畫",
      "planEdit": "修改",
      "planEditCancel": "取消",
      "planConfirmBtn": "確認並生成程式碼",
      "planSendBtn": "分析並生成計畫",
      "planSymbolWarn": "⚠️ 請先在頂部選擇交易品種和時間週期",
      "planSymbolOk": "{symbol} · {timeframe}",
      "planPrerequisiteMsg": "請先在頂部工具欄選擇交易品種（如 EURUSD）和時間週期（如 1H），否則生成策略後無法回測。",
      "execTitle": "AI 執行中",
      "execRunning": "執行中",
      "execDone": "完成",
      "execBackToPlan": "返回規劃",
      "execPlanLabel": "執行計畫",
      "execComplianceTool": "合規檢查",
      "execBacktestTool": "回測",
      "execToolRunning": "正在執行 {tool}...",
      "execFeedbackTitle": "💬 繼續與 AI 對話",
      "execFeedbackHint": "你可以用自然語言告訴 AI 如何調整策略：描述問題、要求改進、或討論思路都可以。",
      "execChipLowerDd": "降低回撤",
      "execChipRaiseReturn": "提高收益",
      "execChipTightenSl": "收緊止損",
      "execChipLongOnly": "只做多",
      "execSendFeedback": "發送給 AI",
      "execClear": "清空",
      "execApplyCode": "應用程式碼到編輯器",
      "execSkipNoSymbol": "回測跳過：未選擇交易品種",
      "execSkipNoCode": "回測跳過：程式碼為空",
      "validating": "合规检查"
    }
  }
} as const;
export default StrategyGen;
