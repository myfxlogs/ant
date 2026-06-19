// Auto-generated from proto/ant/v1/i18n/strategy_schedules_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategySchedules = {
  "strategy": {
    "schedules": {
      "actions": {
        "create": "Create",
        "healthCheck": "Health check",
        "logs": "Logs",
        "runNow": "Run now"
      },
      "createSchedule": "Create Schedule",
      "deleteConfirm": {
        "title": "Delete this schedule?"
      },
      "editModal": {
        "advanced": {
          "fixedIntervalSeconds": "Fixed interval (seconds)",
          "fixedIntervalSecondsExtra": "Override default interval",
          "hfCooldownMs": "HF cooldown (ms)",
          "hfCooldownMsExtra": "Minimum interval between HF signals",
          "parametersJson": "Parameters (JSON)",
          "parametersJsonExtra": "JSON parameters for the strategy",
          "stableOverrideIntervalSeconds": "Stable override interval (seconds)",
          "stableOverrideIntervalSecondsExtra": "Override stable timeframe interval",
          "timeframe": "Timeframe",
          "timeframeExtra": "Select timeframe for execution",
          "title": "Advanced",
          "triggerMode": "Trigger mode",
          "triggerModeExtra": "Choose when to trigger signals",
          "triggerModeOptions": {
            "hf": "High-frequency signal stream",
            "stable": "Stable K-line"
          }
        },
        "autoName": {
          "strategy": "Strategy"
        },
        "fields": {
          "account": "Account",
          "cronExpression": "Cron expression",
          "cronExtra": "Use cron format to schedule runs",
          "enableExtra": "Enable schedule after creating",
          "intervalSeconds": "Interval (seconds)",
          "intervalSecondsExtra": "Run every N seconds",
          "lot": "Lots",
          "lotExtra": "Lots per trade",
          "name": "Name",
          "runFrequency": "Run frequency",
          "symbol": "Symbol",
          "template": "Template",
          "templateExtra": "Select a template to run"
        },
        "placeholders": {
          "name": "Enter schedule name",
          "selectAccountFirst": "Select an account first",
          "symbol": "Select a symbol"
        },
        "runFrequencyExtra": {
          "byTimeframe": "Run by timeframe",
          "cron": "Run by cron expression"
        },
        "runFrequencyOptions": {
          "byTimeframe": "By timeframe",
          "cron": "Cron"
        },
        "title": {
          "create": "Create schedule",
          "edit": "Edit schedule"
        },
        "validation": {
          "accountRequired": "Account is required",
          "cronRequired": "Cron expression is required",
          "lotRequired": "Lots is required",
          "nameRequired": "Name is required",
          "runFrequencyRequired": "Run frequency is required",
          "symbolRequired": "Symbol is required",
          "templateRequired": "Template is required",
          "timeframeRequired": "Timeframe is required",
          "triggerModeRequired": "Trigger mode is required"
        }
      },
      "enableCount": "Enable count",
      "format": {
        "cron": "cron: {{expr}}",
        "interval": "every {{s}}s"
      },
      "health": {
        "fields": {
          "configKey": "Config key",
          "failedRuns": "Failed runs",
          "grade": "Health grade",
          "lastRunAt": "Last run",
          "latestError": "Latest error",
          "latestProfit": "Latest profit",
          "latestTicket": "Latest filled ticket",
          "rule": "Rule",
          "successOverTotal": "Success / Total",
          "thresholds": "Current thresholds"
        },
        "grade": {
          "alert": "Alert",
          "healthy": "Healthy",
          "noSample": "No sample",
          "pending": "Pending",
          "watch": "Watch"
        },
        "messages": {
          "clickRefresh": "Click refresh to load health data",
          "loadFailed": "Failed to load health data"
        },
        "notes": {
          "alert": "Low success rate. Investigate strategy/account conditions now.",
          "healthy": "High success rate and controlled failures.",
          "noSample": "Not enough samples to evaluate (minimum {{minSampleSize}}).",
          "pending": "Run health check first.",
          "watch": "Success rate is acceptable but should be monitored (>= {{yellowSuccessRate}}%)."
        },
        "runLogs": {
          "signalType": "Signal"
        },
        "sections": {
          "orders": "Recent order records",
          "runLogs": "Recent execution logs"
        },
        "summaryBanner": "Grade: {{grade}}; samples: {{totalRuns}}, success rate: {{successRate}}%",
        "thresholdsSummary": "min_sample_size={{minSampleSize}}, green: success>={{greenSuccessRate}}% & failed<={{greenMaxFailedRuns}}, yellow: success>={{yellowSuccessRate}}%",
        "title": "Strategy health check {{name}}"
      },
      "messages": {
        "defaultTemplateNotFound": "Default template not found",
        "executeFailed": "Execution failed",
        "importDefaultTemplateFailedNoId": "Failed to import default template (missing id)",
        "noOrderableSignal": "No orderable signal",
        "orderFailed": "Order failed",
        "orderSubmitted": "Order submitted",
        "parametersParseFailed": "Failed to parse parameters",
        "signalHoldCannotOrder": "Signal is HOLD. Cannot place order.",
        "strategyExecuteFailed": "Strategy execution failed",
        "templateCodeEmptyCannotExecute": "Template code is empty. Cannot execute.",
        "volumeInvalid": "Invalid volume"
      },
      "nextRunAt": "Next run at",
      "status": {
        "disabled": "Disabled",
        "running": "Running"
      },
      "table": {
        "account": "Account",
        "actions": "Actions",
        "lastRun": "Last run",
        "name": "Name",
        "schedule": "Schedule",
        "status": "Status",
        "template": "Template",
        "tradeParams": "Trade params"
      },
      "templateVisibility": {
        "private": "Private",
        "public": "Public"
      },
      "title": "Schedules",
      "triggerModal": {
        "actions": {
          "confirmOrder": "Confirm order",
          "rerun": "Re-run"
        },
        "cards": {
          "logs": "Logs",
          "signal": "Signal"
        },
        "confirmOrder": {
          "ok": "Confirm",
          "title": "Confirm order"
        },
        "emptyLogs": "No logs",
        "emptySignal": "No signal",
        "messages": {
          "signalNotOrderable": "Signal is not orderable"
        },
        "summary": {
          "account": "Account",
          "scheduleName": "Schedule name",
          "symbol": "Symbol",
          "timeframe": "Timeframe"
        },
        "title": "Trigger schedule"
      },
      "validation": {
        "parametersMustBeJsonObject": "Parameters must be a JSON object"
      }
    }
  }
} as const;
export default StrategySchedules;
