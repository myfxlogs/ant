// Auto-generated from proto/ant/v1/i18n/strategy_templates_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Templates = {
  "strategy": {
    "templates": {
      "actions": {
        "backtest": "Backtest",
        "copy": "Copy",
        "create": "New Template",
        "createTemplate": "Create template",
        "delete": "Delete",
        "edit": "Edit",
        "launchSchedule": "Launch schedule",
        "viewCode": "View code"
      },
      "backtest": {
        "accountDisabledSuffix": " (disabled)",
        "fields": {
          "account": "Account",
          "extraSymbols": "Extra symbols (multi-select)",
          "initialCapital": "Initial capital",
          "range": "Range",
          "symbol": "Symbol",
          "timeframe": "Timeframe",
          "title": "Title"
        },
        "modalTitleWithName": "Backtest: {{name}}",
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
        "title": "Backtest",
        "tooltips": {
          "extraSymbols": "Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context[\"closes_by_symbol\"]."
        },
        "validation": {
          "accountRequired": "Account is required",
          "initialCapitalRequired": "Initial capital is required",
          "rangeRequired": "Range is required",
          "symbolRequired": "Symbol is required",
          "timeframeRequired": "Timeframe is required"
        }
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "Create schedule",
          "launchSchedule": "View score",
          "view": "View"
        },
        "batchDelete": "Delete {{count}}",
        "batchDeleteConfirm": "Delete {{count}} backtest report(s)?",
        "batchDeleteSuccess": "{{count}} backtest report(s) deleted",
        "deleteConfirm": "Delete this run?",
        "empty": "No backtest runs",
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
        "title": "Backtest runs"
      },
      "badges": {
        "preset": "Preset"
      },
      "codeModal": {
        "actions": {
          "copy": "Copy"
        },
        "title": "Strategy code"
      },
      "copySuffix": " (copy)",
      "defaultDraftName": "Draft template",
      "deleteConfirm": "Delete this template?",
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
          "create": "Create template",
          "edit": "Edit template"
        },
        "validation": {
          "codeRequired": "Code is required",
          "nameRequired": "Name is required"
        }
      },
      "messages": {
        "backtestCancelFailed": "Failed to cancel backtest",
        "backtestCancelRequested": "Backtest cancel requested",
        "backtestRangeInvalid": "Invalid backtest range",
        "backtestReportDeleted": "Backtest report deleted",
        "backtestReportNotFound": "Backtest report not found",
        "backtestRunNoPublishedTemplate": "Backtest run has no published template",
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
        "deepLinkNavigate": "Opened template and latest run details from external link",
        "enterStrategyCode": "Please enter strategy code",
        "fetchTemplateListFailed": "Failed to load template list",
        "missingDraftIdCannotPublish": "Missing draft id. Cannot publish.",
        "missingScheduleInfo": "Missing schedule info",
        "publishFailed": "Publish failed",
        "publishedButNoTemplateId": "Published, but template id is missing.",
        "readStrategyCodeFailed": "Failed to read strategy code",
        "readTemplateStatusFailed": "Failed to read template status",
        "republishedButNoTemplateId": "Republished, but template id is missing.",
        "scheduleCreated": "Schedule created",
        "scheduleCreatedAndEnabled": "Schedule created and enabled",
        "selectBacktestRange": "Please select backtest range",
        "strategyCodeEmptyCannotBacktest": "Strategy code is empty. Cannot backtest.",
        "strategyCodeEmptyCannotPublish": "Strategy code is empty. Please save your code before publishing.",
        "systemTemplateReadOnly": "System templates are read-only. Clone to edit.",
        "templateAlreadyPublished": "Template already published",
        "templateCreated": "Template created",
        "templateDeleted": "Template deleted",
        "templateNotDraftUnknownPublishStatus": "Template is not a draft. Unknown publish status.",
        "templateNotPublishedCannotCreateSchedule": "Template is not published. Cannot create schedule.",
        "templatePublished": "Template published",
        "templateRepublished": "Template republished",
        "templateUpdated": "Template updated"
      },
      "scheduleLaunch": {
        "actions": {
          "addAccount": "Add Account",
          "create": "Create Schedule",
          "createAndEnable": "Create & enable",
          "createScheduleNoEnable": "Create schedule",
          "publishTemplate": "Publish template",
          "updateTradingPassword": "Update Trading Password"
        },
        "backtestRunningHint": "Backtest is running. Please wait.",
        "errorInvestorAccount": "Cannot launch schedule with investor-only account. Update trading password to enable trading.",
        "form": {
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
          "scheduleTypes": {
            "hfQuote": "High-Freq Quote",
            "interval": "Interval",
            "klineClose": "K-line Close"
          },
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
        "investorWarningBody": "This account is in investor (read-only) mode. You need trading permission to launch schedules.",
        "investorWarningTitle": "Investor Account",
        "keyMetrics": "Key metrics",
        "launchSection": "Launch schedule",
        "metrics": {
          "annualReturn": "Annual return",
          "maxDrawdown": "Max drawdown",
          "sharpe": "Sharpe ratio",
          "totalReturn": "Total return",
          "totalTrades": "Total trades",
          "winRate": "Win rate"
        },
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
      "scheduleName": "Schedule name: {{name}}",
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
        "system": "System templates",
        "user": "User templates"
      },
      "title": "Templates",
      "visibility": {
        "private": "Private",
        "public": "Public"
      }
    }
  }
} as const;
export default Templates;
