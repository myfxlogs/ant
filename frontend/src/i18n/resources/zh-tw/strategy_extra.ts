// Auto-generated supplementary keys for strategy
// TODO: Translate to zh-tw
const StrategyExtra = {
  "strategy": {
    "backtest": {
      "canceled": "Backtest canceled",
      "lotSize": "Lot Size",
      "strategyParameters": "Strategy Parameters"
    },
    "workspace": {
      "chartIndicators": {
        "overlay": "Overlay (main chart)",
        "subPane": "Sub-pane indicators"
      },
      "tour": {
        "code": "Code Editor",
        "codeDesc": "Write or paste your MQL strategy code here. You can also import .mq4/.mq5 files.",
        "ai": "AI Assistant",
        "aiDesc": "Ask AI to generate, optimize, or debug your strategy. Applied code appears in the editor instantly.",
        "backtestDesc": "Run backtests with configurable parameters. View equity curve, trade statistics, and risk metrics.",
        "save": "Save & Publish",
        "saveDesc": "Save your strategy as a template, publish to marketplace, or deploy to a live schedule."
      }
    },
    "codeAssist": {
      "aiHint": "Describe the changes you want, e.g."
    },
    "chat": {
      "executionPlan": "Execution Plan",
      "codeGenerated": "Code generated. Use the buttons below to run strategy review and backtest."
    },
    "templates": {
      "title": "Strategy Templates",
      "saveCurrent": "Save Current Strategy",
      "chatEdit": "Chat Edit",
      "confirmDelete": "Delete this strategy?",
      "noTemplates": "No saved strategy templates",
      "sourceCode": "Strategy Source",
      "gallery": {
        "unpublishFailed": "Unpublish failed",
        "fork": "Fork & Edit",
        "aiGenerate": "AI Generate",
        "searchPlaceholder": "Search strategies...",
        "empty": "No strategies found",
        "deleteFailed": "Delete failed"
      },
      "scheduleLaunch": {
        "metrics": {
          "winRate": "Win Rate",
          "maxDrawdown": "Max DD",
          "sharpe": "Sharpe Ratio"
        }
      },
      "detail": {
        "profitFactor": "Profit Factor",
        "notFound": "Strategy not found",
        "noDescription": "No description",
        "equityCurve": "Equity Curve",
        "tradeStats": "Trade Statistics"
      },
      "table": {
        "useCount": "Use Count"
      },
      "messages": {
        "fetchTemplateListFailed": "Fork failed",
        "publishFailed": "Publish failed"
      },
      "actions": {
        "create": "New Strategy"
      },
      "deleteConfirm": "Delete this strategy?"
    },
    "live": {
      "stopSuccess": "Strategy stopped",
      "stopFailed": "Failed to stop",
      "runId": "Run ID",
      "watchSignals": "Watch Signals",
      "confirmStop": "Stop this strategy?",
      "totalSignals": "Total Signals",
      "title": "Live Strategy Monitor",
      "activeTab": "Active Runs",
      "noActive": "No active strategies",
      "historyTab": "Run History",
      "noRuns": "No strategy runs",
      "signalLog": "Signal Log",
      "waitingSignals": "Waiting for signals..."
    },
    "ai": {
      "reviseHint": "Write code first, then ask AI to improve it.",
      "explainHint": "Write code to see AI explanation.",
      "settingsHint": "Configure AI provider and model"
    },
    "validate": {
      "running": "Running validation...",
      "fixWithAI": "Send errors to AI Revise",
      "allClear": "All checks passed — no issues found.",
      "passed": "Validation passed — Save is now unlocked."
    },
    "importEA": {
      "writeTab": "Strategy Code",
      "importTab": "Import EA",
      "codeTooShort": "Please paste complete EA/indicator source code.",
      "pastePlaceholder": "Paste MQL4/MQL5 EA code...",
      "migration": "策略导入",
      "aiTranslate": "AI 翻译",
      "bridge": "盲区桥接",
      "analyze": "分析策略结构",
      "confirmImport": "确认导入",
      "tryAI": "AI 翻译补充",
      "apply": "Apply to Editor",
      "importSuccess": "MQL 源码已导入，点击「Apply to Editor」写入编辑器",
      "hint": "Paste MQL4/MQL5 code and click Analyze",
      "translate": "Translate to Go",
      "translating": "Paste MQL4/MQL5 code and click Translate",
      "bridgeBtn": "盲区桥接翻译",
      "bridging": "AI bridging blind spots...",
      "bridgeFailedMsg": "Agent 无法自动桥接所有盲区",
      "noBridgeNeeded": "覆盖率 100%，无需桥接",
      "bridgeHint": "Paste MQL4/MQL5 EA code, AI will bridge blind spots to platform bytecode",
      "tooltip": "Import MQL4/MQL5 source code",
      "button": "Import MQL",
      "title": "Import MQL Strategy"
    },
    "version": {
      "loadFailed": "Failed to load versions",
      "rollbackSuccess": "Rolled back to version {{n}}",
      "rollbackFailed": "Rollback failed",
      "loadVersionFailed": "Failed to load version",
      "loadDiffFailed": "Failed to load diff",
      "colSummary": "Change Summary",
      "rollbackConfirm": "Rollback to v{{n}}?",
      "title": "Version History",
      "empty": "No version history yet",
      "history": "Version History"
    }
  }
} as const;
export default StrategyExtra;
