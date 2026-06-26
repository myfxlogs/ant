// Auto-generated from proto/ant/v1/i18n/strategy_templates_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTemplates = {
  "strategy": {
    "templates": {
      "scheduleLaunch": {
        "form": {
          "scheduleTypes": {
            "hfQuote": "High-Freq Quote",
            "interval": "Interval",
            "klineClose": "K-line Close"
          },
          "account": "Account",
          "accountPlaceholder": "Select account",
          "defaultVolume": "Default Volume (lots)",
          "defaultVolumeTip": "Default order volume per signal",
          "enableAfterCreate": "Enable after creation",
          "hfCooldownMs": "HF Cooldown (ms)",
          "hfCooldownMsTip": "Cooldown between quote-driven executions",
          "intervalMs": "Interval (ms)",
          "intervalMsTip": "Minimum 1000ms for non-HF modes",
          "investorTag": "Investor (Read-only)",
          "maxDrawdownPct": "Max Drawdown %",
          "maxDrawdownPctTip": "Auto-stop if drawdown exceeds this threshold",
          "maxPositions": "Max Positions",
          "maxPositionsTip": "Maximum concurrent open positions",
          "riskSection": "Risk Controls",
          "scheduleName": "Schedule Name",
          "scheduleNameMax": "Max 64 characters",
          "scheduleNamePlaceholder": "Enter schedule name",
          "scheduleType": "Schedule Type",
          "stopLossOffset": "Stop Loss Offset",
          "stopLossOffsetTip": "SL offset from entry price (pips)",
          "strategyParamsSection": "Strategy Parameters",
          "symbol": "Symbol",
          "symbolPlaceholder": "Select symbol",
          "symbolPlaceholderEmpty": "No symbols configured",
          "takeProfitOffset": "Take Profit Offset",
          "takeProfitOffsetTip": "TP offset from entry price (pips)",
          "timeframe": "Timeframe"
        },
        "actions": {
          "addAccount": "Add Account",
          "create": "Create Schedule",
          "createAndEnable": "Create & enable",
          "createScheduleNoEnable": "Create schedule",
          "publishTemplate": "Publish strategy",
          "updateTradingPassword": "Update Trading Password"
        },
        "metrics": {
          "annualReturn": "Annual return",
          "maxDrawdown": "Max drawdown",
          "sharpe": "Sharpe ratio",
          "totalReturn": "Total return",
          "totalTrades": "Total trades",
          "winRate": "Win rate"
        },
        "backtestRunningHint": "Backtest is running. Please wait.",
        "errorInvestorAccount": "Cannot launch schedule with investor-only account. Update trading password to enable trading.",
        "investorWarningBody": "This account is in investor (read-only) mode. You need trading permission to launch schedules.",
        "investorWarningTitle": "Investor Account",
        "keyMetrics": "Key metrics",
        "launchSection": "Launch schedule",
        "newPasswordPlaceholder": "Enter new trading password",
        "noAccountBody": "You need to bind an MT account before launching a schedule.",
        "noAccountTitle": "No Account",
        "noRun": "No backtest run",
        "score": "Score",
        "title": "Launch schedule",
        "tradePermissionOk": "Trading permission verified",
        "updatePasswordFailed": "Failed to update trading password",
        "updatePasswordHint": "Enter the trading password for this account to enable trading.",
        "updatePasswordOk": "Trading password updated",
        "updatePasswordStillInvestor": "Password update succeeded but account still in investor mode. Contact support.",
        "updatePasswordTitle": "Update Trading Password",
        "verifyingPermission": "Verifying trading permission..."
      },
      "backtest": {
        "fields": {
          "account": "Account",
          "extraSymbols": "Extra symbols (multi-select)",
          "initialCapital": "Initial capital",
          "range": "Range",
          "symbol": "Symbol",
          "timeframe": "Timeframe",
          "title": "Title"
        },
        "parameters": {
          "title": "Strategy Parameters"
        },
        "placeholders": {
          "account": "Select an account",
          "extraSymbols": "Optional, useful for pairs/rotation strategies",
          "range": "Select range",
          "symbol": "Select a symbol"
        },
        "quickRange": {
          "custom": "Custom"
        },
        "tooltips": {
          "extraSymbols": "Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context[\"closes_by_symbol\"]."
        },
        "validation": {
          "accountRequired": "Account is required",
          "initialCapitalRequired": "Initial capital is required",
          "rangeRequired": "Range is required",
          "symbolRequired": "Symbol is required",
          "timeframeRequired": "Timeframe is required"
        },
        "accountDisabledSuffix": " (disabled)",
        "modalTitleWithName": "Backtest: {{name}}",
        "title": "Backtest"
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "Create schedule",
          "launchSchedule": "View score",
          "view": "View"
        },
        "status": {
          "canceled": "Canceled",
          "canceling": "Canceling",
          "completed": "Completed",
          "failed": "Failed",
          "queued": "Queued",
          "running": "Running"
        },
        "table": {
          "actions": "Actions",
          "createdAt": "Created at",
          "status": "Status",
          "symbol": "Symbol",
          "timeframe": "Timeframe",
          "title": "Title"
        },
        "batchDelete": "Delete {{count}}",
        "batchDeleteConfirm": "Delete {{count}} backtest report(s)?",
        "batchDeleteSuccess": "{{count}} backtest report(s) deleted",
        "deleteConfirm": "Delete this run?",
        "empty": "No backtest runs",
        "title": "Backtest runs"
      },
      "codeModal": {
        "actions": {
          "copy": "Copy"
        },
        "title": "Strategy code"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "Validate code"
        },
        "fields": {
          "code": "Code",
          "description": "Description",
          "name": "Name",
          "publicShare": "Public"
        },
        "placeholders": {
          "codeSample": "Paste strategy code here",
          "description": "Enter description",
          "name": "Enter name"
        },
        "title": {
          "create": "Create strategy",
          "edit": "Edit strategy"
        },
        "validation": {
          "codeRequired": "Code is required",
          "nameRequired": "Name is required"
        }
      },
      "actions": {
        "backtest": "Backtest",
        "copy": "Copy",
        "create": "New Strategy",
        "createTemplate": "Create strategy",
        "delete": "Delete",
        "edit": "Edit",
        "launchSchedule": "Launch schedule",
        "viewCode": "View code"
      },
      "badges": {
        "preset": "Preset"
      },
      "messages": {
        "backtestCancelFailed": "Failed to cancel backtest",
        "backtestCancelRequested": "Backtest cancel requested",
        "backtestRangeInvalid": "Invalid backtest range",
        "backtestReportDeleted": "Backtest report deleted",
        "backtestReportNotFound": "Backtest report not found",
        "backtestRunNoPublishedTemplate": "Backtest run has no published strategy",
        "backtestRunningCannotPublish": "Backtest is running. Cannot publish now.",
        "backtestSubmitFailed": "Failed to submit backtest",
        "backtestSubmitted": "Backtest submitted",
        "cannotPublishAndCreateDraftFailed": "Unable to publish. Draft creation failed.",
        "codeCopied": "Code copied",
        "codeValidationFailed": "Code validation failed",
        "codeValidationNotPassed": "Code validation did not pass",
        "codeValidationPassed": "Code validation passed",
        "copyFailed": "Copy failed",
        "createScheduleFailed": "Failed to create schedule",
        "deepLinkNavigate": "Opened strategy and latest run details from external link",
        "enterStrategyCode": "Please enter strategy code",
        "fetchTemplateListFailed": "Failed to load strategy list",
        "missingDraftIdCannotPublish": "Missing draft id. Cannot publish.",
        "missingScheduleInfo": "Missing schedule info",
        "publishFailed": "Publish failed",
        "publishedButNoTemplateId": "Published, but strategy id is missing.",
        "readStrategyCodeFailed": "Failed to read strategy code",
        "readTemplateStatusFailed": "Failed to read strategy status",
        "republishedButNoTemplateId": "Republished, but strategy id is missing.",
        "scheduleCreated": "Schedule created",
        "scheduleCreatedAndEnabled": "Schedule created and enabled",
        "selectBacktestRange": "Please select backtest range",
        "strategyCodeEmptyCannotBacktest": "Strategy code is empty. Cannot backtest.",
        "strategyCodeEmptyCannotPublish": "Strategy code is empty. Please save your code before publishing.",
        "systemTemplateReadOnly": "System strategies are read-only. Clone to edit.",
        "templateAlreadyPublished": "Strategy already published",
        "templateCreated": "Strategy created",
        "templateDeleted": "Strategy deleted",
        "templateNotDraftUnknownPublishStatus": "Strategy is not a draft. Unknown publish status.",
        "templateNotPublishedCannotCreateSchedule": "Strategy is not published. Cannot create schedule.",
        "templatePublished": "Strategy published",
        "templateRepublished": "Strategy republished",
        "templateUpdated": "Strategy updated"
      },
      "status": {
        "draft": "Draft",
        "published": "Published"
      },
      "table": {
        "actions": "Actions",
        "createdAt": "Created at",
        "defaultHint": "Default",
        "description": "Description",
        "emptyUser": "No user templates yet. Click \"Create Template\" above to get started.",
        "loadingDefault": "Loading default templates...",
        "name": "Name",
        "status": "Status",
        "tags": "Tags",
        "updatedAt": "Updated at",
        "useCount": "Use count",
        "visibility": "Visibility"
      },
      "tabs": {
        "system": "System strategies",
        "user": "User strategies"
      },
      "visibility": {
        "private": "Private",
        "public": "Public"
      },
      "copySuffix": " (copy)",
      "defaultDraftName": "Draft strategy",
      "deleteConfirm": "Delete this strategy?",
      "scheduleName": "Schedule name: {{name}}",
      "title": "Templates"
    }
  }
} as const;
export default StrategyTemplates;
