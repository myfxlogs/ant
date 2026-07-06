// Auto-generated from proto/ant/v1/i18n/strategy_gen_zh-cn.textproto
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
        "heading": "📊 回测结果",
        "placeholder": "提供反馈以迭代优化 (例如 “太激进”, “添加止损”)"
      },
      "metrics": {
        "maxDrawdown": "最大回撤",
        "return": "收益",
        "sharpe": "夏普",
        "trades": "交易数",
        "winRate": "胜率"
      },
      "backtestMsg": "回测任务已创建",
      "backtestStarted": "回测已启动",
      "clarifyTitle": "确认以下细节:",
      "done": "完成",
      "generating": "生成中...",
      "placeholder": "描述您想要创建的交易策略，例如：“制作一个 EURUSD 1H 布林带均值回归策略”",
      "regenerate": "重新生成",
      "reset": "重新开始",
      "send": "生成策略",
      "template": "模板",
      "title": "策略生成",
      "useDefaults": "使用默认值继续",
      "useDefaultsHint": "AI 将使用通用参数生成策略模板",
      "feedbackInputPlaceholder": "对回测结果不满意？输入反馈来优化策略，例如：「太激进了，加个 1% 止损」",
      "planTitle": "✅ AI 执行计划",
      "planAnalyzing": "AI 正在修改计划...",
      "planErrorTag": "出错",
      "planReset": "重新开始",
      "planCardTitle": "AI 执行计划",
      "planEdit": "修改",
      "planEditCancel": "取消",
      "planConfirmBtn": "确认并生成代码",
      "planSendBtn": "分析并生成计划",
      "planSymbolWarn": "⚠️ 请先在顶部选择交易品种和时间周期",
      "planSymbolOk": "{symbol} · {timeframe}",
      "planHint": "你可以讨论这个计划，或直接说\"生成代码\"开始执行。",
      "planInputPlaceholder": "说说你的想法...",
      "planPrerequisiteMsg": "请先在顶部工具栏选择交易品种（如 EURUSD）和时间周期（如 1H），否则生成策略后无法回测。",
      "execTitle": "💡 AI 诊断与建议",
      "execRunning": "执行中",
      "execDone": "完成",
      "execBackToPlan": "返回规划",
      "execPlanLabel": "执行计划",
      "execComplianceTool": "合规检查",
      "execBacktestTool": "回测",
      "execToolRunning": "正在执行 {tool}...",
      "execFeedbackTitle": "💬 继续与 AI 对话",
      "execFeedbackHint": "你可以用自然语言告诉 AI 如何调整策略：描述问题、要求改进、或讨论思路都可以。",
      "execChipLowerDd": "降低回撤",
      "execChipRaiseReturn": "提高收益",
      "execChipTightenSl": "收紧止损",
      "execChipLongOnly": "只做多",
      "execSendFeedback": "发送给 AI",
      "execClear": "清空",
      "execApplyCode": "应用代码到编辑器",
      "execSkipNoSymbol": "回测跳过：未选择交易品种",
      "execSkipNoCode": "回测跳过：代码为空",
      "validating": "合规检查"
    }
  }
} as const;
export default StrategyGen;
