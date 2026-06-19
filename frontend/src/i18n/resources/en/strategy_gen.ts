// Auto-generated from proto/ant/v1/i18n/strategy_gen_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyGen = {
  "strategy": {
    "gen": {
      "chat": {
        "discuss": "💬 Discuss",
        "generate": "⚡ Generate",
        "repair": "🔧 Repair",
        "revise": "✏️ Revise"
      },
      "feedback": {
        "heading": "📊 Backtest Results",
        "placeholder": "Provide feedback to iterate (e.g. \"Too aggressive\", \"Add stop loss\")"
      },
      "metrics": {
        "maxDrawdown": "Max DD",
        "return": "Return",
        "sharpe": "Sharpe",
        "trades": "Trades",
        "winRate": "Win"
      },
      "backtestMsg": "Backtest task created",
      "backtestStarted": "Backtest Started",
      "clarifyTitle": "A few details to confirm:",
      "done": "Done",
      "generating": "Generating...",
      "placeholder": "Describe the trading strategy you want to create, e.g.: \"Make a Bollinger Band mean-reversion strategy for EURUSD on 1H\"",
      "regenerate": "Regenerate",
      "reset": "Start Over",
      "send": "Generate Strategy",
      "template": "Template",
      "title": "Strategy Generation",
      "useDefaults": "Continue with defaults",
      "useDefaultsHint": "AI will generate a strategy using generic template parameters",
      "feedbackInputPlaceholder": "Not satisfied? Provide feedback to refine the strategy, e.g.: \"Too aggressive, add a 1% stop loss\"",
      "planTitle": "AI Strategy Planner",
      "planAnalyzing": "Analyzing",
      "planErrorTag": "Error",
      "planReset": "Start Over",
      "planCardTitle": "AI Execution Plan",
      "planEdit": "Edit",
      "planEditCancel": "Cancel",
      "planConfirmBtn": "Confirm & Generate Code",
      "planSendBtn": "Analyze & Generate Plan",
      "planSymbolWarn": "⚠️ Please select symbol and timeframe above first",
      "planSymbolOk": "{symbol} · {timeframe}",
      "planPrerequisiteMsg": "Please select a trading symbol (e.g. EURUSD) and timeframe (e.g. 1H) in the toolbar above before generating a strategy. Without them, backtesting is impossible.",
      "execTitle": "AI Executing",
      "execRunning": "Running",
      "execDone": "Done",
      "execBackToPlan": "Back to Plan",
      "execPlanLabel": "Execution Plan",
      "execComplianceTool": "Compliance Check",
      "execBacktestTool": "Backtest",
      "execToolRunning": "Running {tool}...",
      "execFeedbackTitle": "💬 Continue AI Conversation",
      "execFeedbackHint": "Use natural language to guide the AI — describe problems, ask for improvements, or discuss ideas.",
      "execFeedbackPlaceholder": "Try saying:\\n\"Tighten stop loss to 1%\"\\n\"Why is the Sharpe ratio so low? Help me improve it\"\\n\"Change to long-only, no short positions\"",
      "execChipLowerDd": "Lower Drawdown",
      "execChipRaiseReturn": "Raise Returns",
      "execChipTightenSl": "Tighten Stop",
      "execChipLongOnly": "Long Only",
      "execSendFeedback": "Send to AI",
      "execClear": "Clear",
      "execApplyCode": "Apply to Editor",
      "execSkipNoSymbol": "Backtest skipped: no symbol selected",
      "execSkipNoCode": "Backtest skipped: empty code",
      "validating": "Compliance Check"
    }
  }
} as const;
export default StrategyGen;
