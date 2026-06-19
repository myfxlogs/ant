// Auto-generated from proto/ant/v1/i18n/ai_wizard_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiWizard = {
  "ai": {
    "wizard": {
      "actions": {
        "cancel": "Cancel",
        "next": "Next",
        "prev": "Previous"
      },
      "agents": {
        "codeTitle": "Code generation",
        "riskTitle": "Risk control and execution constraints",
        "signalsTitle": "Signal and indicator design",
        "styleTitle": "Market condition / style recommendation"
      },
      "currentModel": "Current model: {{model}}",
      "generate": {
        "actions": {
          "abort": "Abort",
          "goValidate": "Go to validate",
          "hide": "Hide",
          "regenerateSummary": "Regenerate summary",
          "rerun": "Regenerate",
          "runAgents": "Run multiple experts + generate code"
        },
        "cards": {
          "resultsTitle": "Multiple experts\\\\"
        },
        "hints": {
          "afterGenerated": "After generation, proceed to validate/backtest/launch"
        },
        "labels": {
          "elapsed": "Elapsed"
        },
        "modals": {
          "final": {
            "title": "Code generated. Recommended to click \"Validate code\" to confirm it passes."
          }
        },
        "sections": {
          "output": "Model output",
          "prompt": "Prompt sent to model",
          "spec": "Spec"
        },
        "status": {
          "done": "Done",
          "error": "Error",
          "idle": "Waiting",
          "inProgress": "In progress",
          "running": {
            "code": "Code generation in progress",
            "generic": "{{title}} in progress",
            "risk": "Risk control in progress",
            "signals": "Signal design in progress",
            "style": "Market condition/style recommending"
          }
        }
      },
      "messages": {
        "agentFailed": "{{title}} failed",
        "aiRequestTimeout": "AI request timeout (>{{seconds}}s)",
        "backtestCreated": "Backtest task created",
        "backtestNotDoneWait": "Backtest not finished, wait until scorecard status becomes success/failed/cancelled before continuing",
        "chatAborted": "Chat with model aborted",
        "codeInvalidFixAndContinue": "Code validation failed, please fix before continuing",
        "confirmScoreFirst": "Please confirm score result in the score popup first",
        "createBacktestFailed": "Create backtest failed",
        "createDraftFailed": "Create draft failed",
        "createScheduleFailed": "Create schedule failed",
        "datasetFrozenCreated": "Frozen dataset created",
        "draftNotCreated": "Draft not created",
        "draftSaved": "Draft saved",
        "fillRequired": "Please fill required fields first",
        "fillRequiredWithFields": "Please fill required fields first: {{fields}}",
        "freezeDatasetFailed": "Freeze dataset failed",
        "generateCodeFirst": "Please generate strategy code first",
        "inputIntentFirst": "Please enter strategy goal/idea first",
        "loadAccountsFailed": "Load accounts failed",
        "loadDatasetFailed": "Load dataset failed",
        "loadSymbolsFailed": "Load symbols failed",
        "modelReturnedEmpty": "Model returned empty",
        "noCodeToBacktest": "No code to backtest",
        "noCodeToValidate": "No code to validate",
        "noPythonCodeBlock": "Code agent did not output ```python block```, check result",
        "publishFailed": "Publish failed",
        "publishTemplateFirst": "Please publish template first",
        "publishedNoId": "Published but no returned id (please check in strategy management)",
        "saveFailed": "Save failed",
        "scheduleAlreadyExists": "Schedule with same template+symbol+timeframe already exists for this account; do not duplicate.",
        "scheduleCreated": "Schedule created",
        "scheduleCreatedAndEnabled": "Schedule created and enabled",
        "startBacktestFirst": "Please click \"Backtest (async task)\" to start backtest first",
        "templatePublished": "Template published",
        "userAborted": "User aborted",
        "validateCodeFirst": "Please click \"Validate code\" first",
        "validateError": "Validation error",
        "validateFailed": "Validation failed",
        "validateOk": "Validation passed",
        "watchBacktestRunFailed": "watchBacktestRun failed"
      },
      "prompts": {
        "base": {
          "account": "Account: {{accountId}}",
          "constraints": "Constraints: Max drawdown={{maxDrawdownPct}}% Risk per trade={{riskPerTradePct}}% Max trades per day={{maxTradesPerDay}}",
          "data": "Data: {{dataSpec}}",
          "empty": "(empty)",
          "macroDisabled": "Macro events: not used",
          "macroEnabled": "Macro events (user-provided):\\\\n{{text}}",
          "params": "Parameters (defs+current values; injected into context[\"params\"] at runtime):\\\\n{{params}}",
          "symbol": "Symbol: {{symbol}}",
          "timeframe": "Timeframe: {{timeframe}}",
          "userIntent": "User strategy goal (natural language):\\\\n{{intent}}"
        },
        "dataSpec": {
          "dataset": "Use frozen dataset datasetId={{datasetId}}",
          "klineRange": "Use historical K-line range from={{from}} to={{to}}"
        },
        "summary": {
          "codeTitle": "Code:",
          "intro": "You are a quantitative strategy explanation assistant. Concisely explain (bullet points, max 12 lines) the core idea of this AntTrader Python strategy code to help users judge if it matches expectations.",
          "mustInclude1": "1) Strategy type/paradigm (trend/mean/breakout/momentum/grid/etc.; state \"uncertain\" if unclear)",
          "mustInclude2": "2) Main entry conditions (2-4 bullet points)",
          "mustInclude3": "3) Main exit/SL/TP/risk controls (2-4 bullet points)",
          "mustInclude4": "4) Applicable / inapplicable scenario each 1 line",
          "mustIncludeTitle": "Must include:",
          "userIntent": "User expectation (natural language):\\\\n{{intent}}"
        },
        "upstream": {
          "risk": "【Risk control conclusion】\\\\n{{text}}",
          "sectionTitle": "【Upstream agent conclusions (as provided)】",
          "signals": "【Signal design conclusion】\\\\n{{text}}",
          "style": "【Market condition/style conclusion】\\\\n{{text}}"
        }
      },
      "publish": {
        "actions": {
          "publishTemplate": "Publish template",
          "startBacktest": "Backtest (async task)",
          "validateCode": "Validate code"
        },
        "cards": {
          "codeTitle": "1) Strategy code (editable)",
          "launchTitle": "3) Launch schedule",
          "scoreCardTitle": "2) Backtest scorecard"
        },
        "messages": {
          "validateFailed": "validate failed",
          "validateOk": "validate passed"
        },
        "placeholders": {
          "codeEditable": "AI-generated code will be filled here; you can also edit manually."
        }
      },
      "publishBacktest": {
        "actions": {
          "close": "Close",
          "confirm": "Confirm",
          "inProgress": "In progress",
          "retry": "Retry",
          "runInBackground": "Run in background",
          "startBacktest": "Start backtest",
          "succeeded": "Success"
        },
        "cards": {
          "backtestTitle": "Backtest",
          "scoreCardTitle": "Scorecard"
        },
        "draftName": "Backtest {{datetime}} {{symbol}} {{timeframe}}",
        "draftNameShort": "Backtest {{symbol}} {{timeframe}}",
        "labels": {
          "confirmed": "Confirmed",
          "elapsed": "Elapsed",
          "overallScore": "Overall score",
          "scoringProgress": "Scoring progress",
          "status": "Status"
        },
        "modals": {
          "score": {
            "title": "Score confirmation"
          },
          "status": {
            "title": "Backtest in progress"
          }
        }
      },
      "schedule": {
        "defaultName": "AI Schedule {{symbol}} {{timeframe}}"
      },
      "setup": {
        "actions": {
          "deleteCurrentDataset": "Delete current dataset",
          "freezeFromCurrentRange": "Freeze from current range",
          "refreshDataset": "Refresh"
        },
        "cards": {
          "constraintsAndGoalTitle": "Constraints & Goals",
          "hardConstraintsTitle": "Hard Constraints",
          "hintsTitle": "Hints",
          "tradeAndDataTitle": "Trading & Data"
        },
        "dataModes": {
          "dataset": "Frozen dataset",
          "klineRange": "Historical K-line range"
        },
        "hints": {
          "nextWillGenerateCode": "Next step will start generating strategy code.",
          "tradeDataNextStep": "Click \"Next\" after filling to proceed to constraints & goals."
        },
        "labels": {
          "account": "Account",
          "backtestRange": "Backtest Range",
          "dataset": "Frozen Dataset",
          "historicalData": "Historical Data",
          "intent": "Strategy Goal / Idea",
          "macroEvents": "Macro Events",
          "macroModule": "Macro Module",
          "maxDrawdownPct": "Max Drawdown (%)",
          "maxTradesPerDay": "Max Trades per Day",
          "riskPerTradePct": "Risk per Trade (%)",
          "symbol": "Symbol",
          "timeframe": "Timeframe"
        },
        "macro": {
          "off": "Off",
          "on": "On"
        },
        "messages": {
          "datasetDeleted": "Dataset deleted"
        },
        "modals": {
          "deleteDataset": {
            "content": "Are you sure you want to delete the selected frozen dataset?",
            "ok": "Delete",
            "title": "Delete dataset"
          }
        },
        "placeholders": {
          "intentExample": "Example: breakout trend following; avoid high volatility; prefer higher win rate...",
          "macroExample": "Example:\\\\n2024-01-03 21:15 FOMC minutes\\\\n2024-01-05 20:30 NFP",
          "selectAccount": "Select account",
          "selectFrozenDataset": "Select frozen dataset",
          "selectSymbol": "Select symbol",
          "selectTimeframe": "Select timeframe"
        },
        "validations": {
          "enterIntent": "Please enter strategy goal/idea",
          "selectAccount": "Please select account",
          "selectDataset": "Please select dataset",
          "selectSymbol": "Please select symbol",
          "selectTimeframe": "Please select timeframe"
        }
      },
      "steps": {
        "generate": "Generate Strategy",
        "publishBacktest": "Backtest & Live - Backtest",
        "publishCode": "Backtest & Live - Code",
        "publishLaunch": "Backtest & Live - Launch",
        "setup": "Basic Info"
      },
      "strategyParams": {
        "actions": {
          "addParam": "Add parameter",
          "delete": "Delete",
          "exportJson": "Export JSON",
          "importJson": "Import JSON"
        },
        "empty": "No parameters yet. You can add params like fast/slow/risk_per_trade to make the strategy more templated.",
        "hints": {
          "intro": "These parameters will:",
          "line1": "1) Save to template parameters",
          "line2": "2) Write to schedule.parameters when creating a schedule (map<string,string>)",
          "line3Prefix": "3) Backend will inject them into Python strategy"
        },
        "labels": {
          "default": "default",
          "description": "description",
          "label": "label",
          "max": "max",
          "min": "min",
          "name": "name",
          "options": "options (for select, comma-separated)",
          "step": "step",
          "type": "type",
          "value": "value (schedule current value)"
        },
        "messages": {
          "copied": "Copied",
          "copyFailed": "Copy failed",
          "importFormatInvalid": "Import format error: must be array or { \"paramDefs\": [...] }",
          "importMissingName": "Import failed: parameter missing name",
          "imported": "Imported {{count}} parameters",
          "jsonParseFailed": "JSON parse failed"
        },
        "modals": {
          "copyAndClose": "Copy and close",
          "exportTitle": "Export parameters JSON",
          "importOk": "Import",
          "importTitle": "Import parameters JSON"
        },
        "paramCardTitle": "Param #{{index}}",
        "placeholders": {
          "defaultExample": "e.g. 10",
          "description": "Description",
          "importJson": "Paste parameter JSON (array or {\"paramDefs\": [...]})",
          "label": "Display name",
          "nameExample": "e.g. fast",
          "optionsExample": "e.g. low,medium,high",
          "value": "Empty uses default"
        },
        "title": "Strategy Parameters (optional)",
        "types": {
          "bool": "Boolean",
          "number": "Number",
          "select": "Select",
          "string": "String"
        },
        "validations": {
          "nameRequired": "name required",
          "typeRequired": "type required"
        }
      },
      "subtitle": "One step per page, can go forward/backward",
      "template": {
        "defaultDescription": "AI wizard generated",
        "defaultName": "AI Strategy {{title}}"
      },
      "title": "AI Strategy Wizard"
    }
  }
} as const;
export default AiWizard;
