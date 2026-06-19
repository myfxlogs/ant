// Auto-generated from proto/ant/v1/i18n/ai_core_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "agentPrompts": {
      "code": {
        "title": "Code generation agent"
      },
      "risk": {
        "title": "Risk control and execution constraints"
      },
      "signals": {
        "title": "Signal and indicator design"
      },
      "style": {
        "title": "Market condition / style recommendation"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "No code block found (\\\\`\\\\`\\\\`...\\\\`\\\\`\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "None (save template first, will auto-calculate after backtest completes)",
        "loading": "Calculating...",
        "reasons": "Reasons",
        "reliable": "Reliable",
        "title": "Strategy Risk Score",
        "unknown": "unknown",
        "unreliable": "Unreliable",
        "warnings": "Warnings"
      },
      "chart": {
        "title": "Equity Curve"
      },
      "level": {
        "excellent": "Excellent",
        "fair": "Fair",
        "good": "Good",
        "poor": "Poor"
      },
      "metrics": {
        "annualReturn": "Annual Return",
        "equityPoints": "Equity points",
        "maxDrawdown": "Max Drawdown",
        "sharpe": "Sharpe",
        "totalReturn": "Total Return",
        "totalTrades": "Total Trades",
        "winRate": "Win Rate"
      },
      "recommendation": {
        "cautious": "Cautious for live: try small capital / manual confirmation for a while first.",
        "loading": "Risk assessment in progress, please wait for completion before going live.",
        "notRecommended": "Not recommended for direct live: high risk or unreliable, optimize before trying.",
        "recommended": "Recommended for live: risk controllable, metrics healthy."
      },
      "score": {
        "empty": "No score yet (wait for backtest or no metrics)",
        "title": "Overall Score (heuristic)"
      },
      "stateLabel": "State",
      "status": {
        "cancelRequested": "Cancelling",
        "canceled": "Cancelled",
        "failed": "Failed",
        "pending": "Queued",
        "running": "Running",
        "succeeded": "Success"
      },
      "title": "Backtest Scorecard"
    },
    "chatBox": {
      "collapse": "Collapse",
      "emptyDescription": "Start a conversation with the AI assistant",
      "expandAll": "Expand all",
      "thinking": "Thinking...",
      "truncated": "Content too long, truncated"
    },
    "client": {
      "errors": {
        "contentBlocked": "The provider safety filter blocked the response. Rephrase the prompt and try again.",
        "contextTooLong": "The request exceeds the model context window. Shorten the conversation/input or pick a model with a larger context.",
        "edgeGatewayTimeout": "The edge gateway timed out (often HTTP 524 on Cloudflare): the browser never received the app response, which is common for long-running operations. Try again; if the issue persists, raise proxy/origin timeouts with ops.",
        "forbidden": "The provider refused the request (403). Check key permissions, IP allowlist, and account status.",
        "gatewayForbidden403": "Gateway forbidden (403).",
        "gatewayRateLimited429": "Gateway rate limited (429).",
        "gatewayTimeoutOrUnreachable": "Gateway timeout or unreachable.",
        "gatewayUnauthorized401": "Gateway unauthorized (401).",
        "insufficientBalance": "The provider reported an empty balance / overdue payment. Top up the account in the provider console and retry.",
        "invalidModelId": "Model unavailable{{model}} – it may be wrong, deprecated, or outside your tier. Pick another from the dropdown or copy the canonical id from the provider console.",
        "networkUnreachable": "Gateway timed out or is unreachable. Check the Base URL, network connectivity, or try again later.",
        "providerInternalError": "The provider returned a server-side error (5xx). Wait a moment or switch to another provider.",
        "rateLimited": "The provider is rate-limiting your requests. Please wait a moment and try again.",
        "regionNotSupported": "The selected provider is not available in your region/country. Switch to a different provider.",
        "requestFailed": "Request failed. Please try again.",
        "unauthorized": "The provider rejected the API key (401). Check the key value and that it has access to the selected model."
      }
    },
    "consensus": {
      "actions": {
        "refresh": "Refresh"
      },
      "fields": {
        "account": "Account",
        "symbol": "Symbol",
        "timeframe": "Timeframe"
      },
      "panel": {
        "decision": "Decision",
        "overallScore": "Overall",
        "technicalScore": "Technical",
        "title": "Objective Score"
      },
      "signals": {
        "ma": {
          "trend": "MA Trend"
        },
        "macd": {
          "flag": "Signal",
          "hist": "Histogram",
          "signalLine": "Signal Line",
          "trend": "Pattern",
          "value": "MACD"
        },
        "rsi": {
          "flag": "Signal",
          "value": "RSI"
        }
      },
      "title": "Consensus & Discussion"
    },
    "conversation": {
      "defaultTitle": "New Conversation"
    },
    "gate": {
      "allPassed": "All 6 gates passed — strategy eligible for PromoteToLive evaluation",
      "backtestGrossReturn": "Backtest Gross Return",
      "backtestNetReturn": "Backtest Net Return",
      "dailyReturns": "Daily Returns (comma or newline separated)",
      "descriptions": {
        "compliance": "DSL expression non-empty validation",
        "correlation": "Signal correlation check with existing strategies",
        "deflated_sharpe": "Lopez de Prado Deflated Sharpe Ratio",
        "lookahead": "Future function reference scan (close[t+N], ref negative offset)",
        "paper": "≥14 days paper trading validation",
        "walkforward": "Purged Walk-Forward cross-validation"
      },
      "details": "Details",
      "dslExpression": "DSL Expression",
      "evaluating": "Evaluating...",
      "fail": "FAIL",
      "failed": "Failed: {{gate}}",
      "gateProgress": "Gate Evaluation Progress",
      "labels": {
        "compliance": "Compliance",
        "correlation": "Correlation",
        "deflated_sharpe": "Deflated Sharpe",
        "lookahead": "Look-Ahead Bias",
        "paper": "Paper Trading",
        "walkforward": "Walk-Forward"
      },
      "noData": "no data",
      "numAttempts": "Strategy Attempts",
      "paperDays": "Paper Days",
      "paperMetrics": "Paper Trading Metrics",
      "paperNetPnL": "Paper Net P&L",
      "paperNetReturn": "Paper Net Return",
      "paperTradeCount": "Paper Trade Count",
      "pass": "PASS",
      "pipelineDesc": "6-stage Gate pipeline: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation",
      "pipelineResult": "Pipeline Result",
      "retry": "Retry",
      "runHint": "Run a backtest first, then click \"Run Gate\" to evaluate strategy quality.",
      "runPipeline": "Run Gate Pipeline",
      "selectRun": "Select backtest run...",
      "skipped": "SKIPPED",
      "status": {
        "evaluating": "Evaluating..."
      },
      "strategyParams": "Strategy Parameters",
      "title": "AI Gate Progress",
      "unknown": "unknown"
    },
    "gateway": {
      "balance": "Balance",
      "modelPlaceholder": "Choose an AI model",
      "monthlyCost": "Cost this month",
      "monthlyTokens": "Tokens this month",
      "noModels": "No models available",
      "selectModel": "Select Model",
      "title": "AI Gateway",
      "usageByFeature": "Usage by Feature",
      "useGateway": "Platform Models",
      "useGatewayDesc": "Wallet billing · Pay per token",
      "useOwnKey": "My API Key",
      "useOwnKeyDesc": "Direct billing · Self-managed",
      "useOwnKeyHint": "Use your own API key to pay the provider directly. Select a provider card below to configure."
    },
    "reports": {
      "tradeAnalysis": {
        "riskAssessmentPrefix": "Risk Assessment:",
        "title": "AI Trade Analysis Report"
      }
    },
    "requireConfig": {
      "actions": {
        "goSettings": "Go to Settings"
      },
      "description": "Please go to Settings first to configure the AI provider, model, and API key, then use the strategy wizard or chat.",
      "title": "No LLM configured yet"
    },
    "riskEval": {
      "failed": "Risk evaluation failed"
    },
    "signalCard": {
      "actions": {
        "cancel": "Cancel",
        "confirm": "Confirm",
        "executeTrade": "Execute Trade"
      },
      "confirmCancel": {
        "title": "Are you sure you want to cancel this signal?"
      },
      "confirmExecute": {
        "description": "Will place the order immediately",
        "title": "Are you sure you want to execute this trade signal?"
      },
      "labels": {
        "analysisReason": "Analysis Reason",
        "confidence": "Confidence",
        "price": "Price",
        "stopLoss": "Stop Loss",
        "takeProfit": "Take Profit",
        "volume": "Lots"
      },
      "status": {
        "cancelled": "Cancelled",
        "confirmed": "Confirmed",
        "executed": "Executed",
        "pending": "Pending"
      }
    },
    "strategyCard": {
      "actionType": {
        "alert": "Alert",
        "buy": "Buy",
        "closeLong": "Close Long",
        "closeShort": "Close Short",
        "sell": "Sell"
      },
      "actions": {
        "start": "Start",
        "stop": "Stop"
      },
      "confirmDelete": {
        "description": "Cannot be recovered after deletion",
        "title": "Are you sure you want to delete this strategy?"
      },
      "labels": {
        "lastTriggeredAt": "Last triggered: {{time}}",
        "triggeredCount": "Triggered {{count}} times"
      },
      "sections": {
        "actions": "Actions",
        "conditions": "Trigger Conditions"
      },
      "status": {
        "active": "Active",
        "inactive": "Inactive",
        "paused": "Paused"
      },
      "tooltips": {
        "createdAt": "Created at",
        "lastTriggeredAt": "Last triggered"
      }
    },
    "systemAI": {
      "cardState": {
        "enabled": "Enabled",
        "noKey": "Not configured",
        "noModel": "Select model",
        "readyDisabled": "Ready · Disabled"
      },
      "cardTags": {
        "current": "Current",
        "enabledButUnavailable": "Enabled but unavailable",
        "hasKey": "Key configured",
        "noKey": "No key",
        "noModels": "No models configured"
      },
      "customProvider": {
        "deleted": "Custom provider deleted",
        "fillNameFirst": "Fill in name first",
        "nameHint": "A unique name to identify this provider",
        "nameLabel": "Provider Name",
        "namePlaceholder": "My Custom Provider",
        "nameRequired": "Provider name is required"
      },
      "emptyConfigs": "No AI Provider configured (system will auto-create default provider on startup)",
      "fields": {
        "apiKeyHint": "Will be auto-encrypted on save, no manual submission needed",
        "apiKeyPastePlaceholder": "Paste API Key, will auto-pre-save",
        "autoFetching": "Auto fetching",
        "baseUrlCustomHint": "Enter OpenAI-compatible endpoint, e.g. https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "e.g. https://model.example.com/v1",
        "baseUrlReadonlyHint": "Official address maintained by system, read-only",
        "baseUrlReadonlyPlaceholder": "Official address (read-only)",
        "enabledHint": "Disabled providers will not be routed",
        "httpWarning": "Currently HTTP, HTTPS recommended for production",
        "maxTokensHint": "Max tokens per response",
        "primaryFor": "Primary For",
        "primaryForHint": "For internal routing: chat / embedding / summarizer / reasoning",
        "temperatureHint": "Higher = more creative, lower = more stable",
        "timeoutHint": "Max wait time per request"
      },
      "messages": {
        "autoDiscoveredModels": "Auto-discovered {{count}} model(s) (for suggestion only)",
        "autoValidatedModels": "Auto-validated: {{count}} model(s) found",
        "configSaveFailed": "Config save failed",
        "configSaved": "Config saved",
        "deleteSecretFailed": "Delete secret failed",
        "loadConfigFailed": "Failed to load configs",
        "secretAutoSaveFailed": "Secret auto-save failed",
        "secretDeletedConfigReset": "Secret deleted, provider config reset to defaults",
        "secretSavedAutoDiscover": "Secret saved, auto-discovering models...",
        "toggleEnabledFailed": "Toggle enabled status failed",
        "validationFailedNeedApiKey": "Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.",
        "validationPassedModels": "Validation passed: {{count}} model(s) found"
      },
      "pageSubtitle": "Configure the AI brain – select providers, manage API keys and available models, and set the default primary model for the whole site.",
      "pageTitle": "AI Assistant Settings",
      "section1": {
        "subtitle": "Cards show each provider's configuration and readiness; click to select",
        "title": "Select Model Provider"
      },
      "status": {
        "checkUrl": "Check Base URL",
        "checkUrlDesc": "API Key ready, but address seems invalid",
        "configReady": "Config ready",
        "configReadyDesc": "Add available models to auto-check connectivity",
        "connectionFailed": "Connection error, check prompts above",
        "error": "Error exists",
        "needKey": "Complete key configuration",
        "needKeyDesc": "Fill API Key to auto-discover model list",
        "noProvider": "No provider selected yet",
        "noProviderDesc": "Pick a model provider from the cards below to start configuration",
        "notEnabled": "Connected, not enabled",
        "notEnabledDesc": "Toggle \"Enabled\" to activate",
        "ready": "Ready",
        "readyDesc": "Enabled and connected"
      },
      "statusBar": {
        "checking": "Checking connectivity…",
        "connected": "Connected",
        "disabled": "Disabled",
        "enabled": "Enabled",
        "keyReady": "Key ready"
      },
      "taglines": {
        "anthropic": "Claude series",
        "deepseek": "DeepSeek · High cost-performance",
        "moonshot": "Kimi · Long context",
        "openai": "GPT series · Official",
        "openai_compatible": "Any compatible endpoint",
        "qwen": "Alibaba Cloud · Chinese optimized",
        "zhipu": "Tsinghua · General"
      }
    },
    "tabs": {
      "agentSettings": "Agent Settings",
      "gate": "AI Gate",
      "settings": "Settings"
    },
    "workflowRuns": {
      "defaultTitle": "AI Workflow",
      "hints": {
        "selectToViewDetail": "Select a run from the left to view details"
      },
      "messages": {
        "loadDetailFailed": "Failed to load details",
        "loadListFailed": "Failed to load run list"
      },
      "title": "AI Workflow"
    }
  }
} as const;
export default AiCore;
