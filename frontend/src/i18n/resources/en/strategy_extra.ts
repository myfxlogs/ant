// Auto-generated supplementary keys for strategy
// TODO: Translate to en
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
      },
      "importMql": "Import MQL"
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
        "untitled": "Untitled Strategy",
        "create": "New Strategy"
      },
      "deleteConfirm": "Delete this strategy?"
    },
    "live": {
      "stopSuccess": "Strategy stopped",
      "stopFailed": "Failed to stop",
      "runId": "Run ID",
      "account": "Account",
      "symbol": "Symbol",
      "timeframe": "TF",
      "mode": "Mode",
      "signals": "Signals",
      "errors": "Errors",
      "startedAt": "Started",
      "watchSignals": "Watch Signals",
      "confirmStop": "Stop this strategy?",
      "status": "Status",
      "totalSignals": "Total Signals",
      "stoppedAt": "Stopped",
      "error": "Error",
      "title": "Live Strategy Monitor",
      "activeTab": "Active Runs",
      "noActive": "No active strategies",
      "historyTab": "Run History",
      "noRuns": "No strategy runs",
      "schedulesTab": "Schedules",
      "signalLog": "Signal Log",
      "waitingSignals": "Waiting for signals...",
      "time": "Time",
      "signalType": "Type",
      "volume": "Volume",
      "price": "Price",
      "sl": "SL",
      "tp": "TP",
      "reason": "Reason"
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
      "migration": "Strategy Import",
      "aiTranslate": "AI Translate",
      "bridge": "Blind Spot Bridging",
      "analyze": "Analyze Strategy Structure",
      "confirmImport": "Confirm Import",
      "import": "Import Strategy",
      "importing": "Compiling...",
      "compileFailed": "Compile Failed",
      "bridgeReference": "AI Bridge Reference (non-executable)",
      "coverage": "Coverage: {{score}}%",
      "tryAI": "AI Translate Supplement",
      "apply": "Apply to Editor",
      "importSuccess": "MQL source imported. Click 'Apply to Editor' to write to the editor.",
      "hint": "Paste MQL4/MQL5 code and click Analyze",
      "translate": "Translate to Go",
      "translating": "Paste MQL4/MQL5 code and click Translate",
      "bridgeBtn": "Blind Spot Bridge Translation",
      "bridging": "AI bridging blind spots...",
      "bridgeFailedMsg": "Agent could not automatically bridge all blind spots",
      "noBridgeNeeded": "Coverage 100%, no bridging needed",
      "bridgeHint": "Paste MQL4/MQL5 EA code, AI will bridge blind spots to platform bytecode",
      "tooltip": "Import MQL4/MQL5 source code",
      "button": "Import MQL",
      "title": "Import MQL Strategy"
    },
    "templateModal": {
      "title": "Save Strategy Template",
      "fields": {
        "name": "Template Name",
        "description": "Description"
      },
      "placeholders": {
        "name": "Enter template name",
        "description": "Enter description"
      }
    },
    "version": {
      "loadFailed": "Failed to load versions",
      "rollbackSuccess": "Rolled back to version {{n}}",
      "rollbackFailed": "Rollback failed",
      "loadVersionFailed": "Failed to load version",
      "loadDiffFailed": "Failed to load diff",
      "colSummary": "Change Summary",
      "colVersion": "Version",
      "colLang": "Lang",
      "colHash": "Hash",
      "colDate": "Date",
      "colActions": "Actions",
      "diff": "Diff",
      "rollbackConfirm": "Rollback to v{{n}}?",
      "title": "Version History",
      "empty": "No version history yet",
      "history": "Version History"
    }
  }
} as const;
export default StrategyExtra;
