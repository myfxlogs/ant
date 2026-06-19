// Auto-generated from proto/ant/v1/i18n/ai_settings_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "actions": {
        "saveConfig": "Save config",
        "validateApiKey": "Validate API Key"
      },
      "agent": {
        "actions": {
          "add": "Add",
          "loadDefaults": "Load default 8 agents",
          "remove": "Delete",
          "restoreDefaults": "Restore defaults",
          "restoreDefaultsConfirmContent": "Will reset 8 system agents (style/signals/risk/macro/sentiment/portfolio/execution/code) to default identity definitions; your custom agents will be kept. This only modifies unsaved drafts; click \"Save\" to persist.",
          "restoreDefaultsConfirmTitle": "Restore system default identities?",
          "save": "Save"
        },
        "defaultName": "Custom Agent",
        "defaults": {
          "code": {
            "inputHint": "Example: target paradigm=trend following; indicators=EMA(fast)/EMA(slow)+ATR filter; params=fast,slow,atr_period,risk_per_trade."
          },
          "execution": {
            "inputHint": "Example: Order=long EURUSD 10 lots; current spread=0.6 pip; target completion in 5 mins; acceptable slippage=0.8 pip."
          },
          "executor": {
            "identity": "Trade execution optimization expert — minimizes slippage and execution costs."
          },
          "macro": {
            "inputHint": "Example: This week key events=US CPI (Thu 20:30), FOMC minutes (Wed next day 02:00); target symbol=XAUUSD."
          },
          "portfolio": {
            "inputHint": "Example: Existing strategies=trend-EURUSD, mean reversion-XAUUSD; total equity=50,000; target annual vol=12%."
          },
          "researcher": {
            "identity": "Macroeconomic and industry researcher — analyzes macro events and sector trends."
          },
          "risk": {
            "inputHint": "Example: Account equity=10,000; acceptable monthly drawdown=5%; risk per trade=0.5%; day trade cap=5; stop-loss=1.5×ATR."
          },
          "risk_manager": {
            "identity": "Strict risk control expert — designs position sizing, stop-loss, drawdown limits."
          },
          "sentiment": {
            "inputHint": "Example: Last week VIX rose from 14 to 22; non-commercial net long positions -18% WoW; news dominated by \"recession/rate cut\" keywords."
          },
          "signals": {
            "inputHint": "Example: Paradigm=trend following; Timeframe=H1; Available indicators=EMA/ATR/ADX; fast default 20, slow default 60."
          },
          "strategist": {
            "identity": "Senior quantitative strategy analyst — recommends strategy paradigms based on account/market conditions."
          },
          "style": {
            "inputHint": "Example: Account=EURUSD retail; Timeframe=H1; Goal=3% monthly return, max drawdown <10%; Preference=win rate over R/R."
          }
        },
        "fields": {
          "historicalBinding": "{{value}} (historical)",
          "identityPlaceholder": "Identity/persona description (will be appended to system prompt)",
          "inputHintPlaceholder": "Input hint (optional)",
          "modelProfileEmpty": "Please first enable at least one provider/model in \"AI Settings\"",
          "modelProfilePlaceholder": "Default (use current profile)",
          "namePlaceholder": "Agent name"
        },
        "messages": {
          "defaultsLoaded": "System default agent templates loaded, click \"Save\" to persist",
          "empty": "No custom agents yet, click \"Add\" to configure",
          "loading": "Loading...",
          "saveFailed": "Agent save failed",
          "saveSuccess": "Agent saved",
          "selectProfileFirst": "Please first select a config on the left"
        },
        "removeConfirmContent": "Are you sure you want to delete this agent?",
        "removeConfirmTitle": "Delete Agent",
        "title": "Agent Identity Definition",
        "types": {
          "code": "Code",
          "execution": "Execution",
          "executor": "Execution Advisor",
          "macro": "Macro",
          "portfolio": "Portfolio",
          "researcher": "Market Researcher",
          "risk": "Risk",
          "risk_manager": "Risk Manager",
          "sentiment": "Sentiment",
          "signals": "Signals/Indicators",
          "strategist": "Strategy Analyst",
          "style": "Style/Paradigm"
        }
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "Open DeepSeek platform: ",
          "step2": "Login/register, then go to API Keys page and create/copy API Key",
          "title": "How to get DeepSeek API Key"
        },
        "default": "Current provider: {{provider}}. Go to the provider\\\\",
        "modelSuggestionDeepSeek": "Model suggestion: select `deepseek-chat` in \"Model\" dropdown",
        "modelSuggestionZhipu": "Model suggestion: select `glm-4-flash` / `glm-4` in \"Model\" dropdown",
        "selectProviderHint": "After selecting an AI provider, how to apply API Key will be shown here.",
        "title": "API Key Application Guide",
        "zhipu": {
          "step1": "Open Zhipu platform: ",
          "step2": "Login/register, then go to console and create/copy API Key",
          "title": "How to get Zhipu API Key"
        }
      },
      "apiKeySavedAs": "Currently saved: {{masked}}",
      "defaultProfileName": "Default",
      "discoverErrors": {
        "baseUrlInvalid": "Invalid Base URL: use a full URL such as https://model.example.com or https://model.example.com/v1",
        "baseUrlRequired": "Please enter Base URL (model service address).",
        "endpoint404": "Model endpoint not found: ensure Base URL matches the OpenAI-compatible API (some need /v1).",
        "freeTierExhausted": "Free tier exhausted: disable free-tier-only in the provider console or use a paid key.",
        "generic": "Failed to list models. Check Base URL and API key.",
        "genericDetail": "Failed to list models: {{detail}}",
        "invalidModelsResponse": "Model service response is not compatible with /models.",
        "noModelsReturned": "No models returned: check account permissions or configuration.",
        "quotaForbidden403": "Call denied (quota): check billing and quota in the provider console.",
        "quotaOrRateLimit": "Quota or rate limit: the provider rejected the call. Check billing/rate limits or retry later.",
        "timeout": "Request timed out: check connectivity or retry later.",
        "unauthorized": "Auth failed: check API key/secret.",
        "unreachable": "Cannot reach model service: check Base URL, network, or gateway."
      },
      "errors": {
        "arrearage": "Provider response: account in arrears / insufficient balance or account status abnormal. Check balance, billing and account status in provider console, then retry.",
        "forbidden": "Provider response: access denied (403). Check key permissions, IP whitelist or account status.",
        "invalidModelId": "Provider response: model unavailable{{model}}. Please select from dropdown, or copy correct model id from provider console.",
        "timeout": "Connection timeout. Check if Base URL is accessible, network is smooth, or retry later.",
        "unauthorized": "Provider response: API Key invalid or unauthorized (401). Check if key is correct and has model permission."
      },
      "fields": {
        "apiKey": "API Key",
        "apiKeyConfigured": "Configured",
        "apiKeyReplaceHint": "Enter again to replace key",
        "availableModels": "Available models",
        "availableModelsEmpty": "Type model id then Enter to add",
        "availableModelsHint": "Multiple models can be enabled under one API Key; this list appears in /ai/agents dropdown. Default empty; pick from dropdown or type model id then Enter to add; only explicitly selected models are added, not all discovered ones.",
        "availableModelsPlaceholder": "Select or type model id then Enter (default empty)",
        "availableModelsTip": "Tip: deleting a model will not clear already-bound agents in /ai/agents, but removes it from dropdown suggestions.",
        "baseUrl": "Base URL",
        "baseUrlHint": "(Model service address)",
        "clear": "Clear",
        "defaultModel": "Default model",
        "deleteApiKey": "Delete key",
        "enabledOff": "Disabled → click to enable",
        "enabledOn": "Enabled → click to disable",
        "enabledStatus": "Enabled",
        "maxTokens": "Max Tokens",
        "model": "Model",
        "name": "Name",
        "provider": "AI Provider",
        "temperature": "Temperature",
        "timeoutSeconds": "Timeout (seconds)"
      },
      "inferenceParams": {
        "title": "Inference Parameters"
      },
      "messages": {
        "apiKeyValidated": "API Key validated",
        "deleted": "Deleted",
        "disabled": "Disabled",
        "enabled": "Enabled",
        "loadConfigFailed": "Failed to load AI config",
        "probeFailed": "Connection failed",
        "probeSuccess": "Connection success",
        "saveSuccess": "Config saved",
        "selectSavedProfileOrEnterKey": "Please first select a saved config, or enter API Key",
        "setCurrentSuccess": "Switched current config",
        "validateBeforeSave": "Please first click \"Validate API Key\", can save only after validation passes",
        "validateFailed": "Validation failed",
        "validateSuccess": "Validation success"
      },
      "pageTitle": "AI Assistant Settings",
      "placeholders": {
        "apiKey": "Enter API Key",
        "baseUrl": "e.g. https://api-inference.modelscope.cn/v1 or https://ark.cn-beijing.volces.com/api/v3",
        "modelManual": "Enter model name (copy model id from provider console)",
        "modelSelect": "Select model",
        "modelSelectOrType": "Select from dropdown or type model id",
        "name": "e.g. DeepSeek-LowCost",
        "provider": "Select AI provider",
        "providerFirst": "Please select AI provider first"
      },
      "primary": {
        "hint": "Used for intent clarification, code generation, AI assistant code modification panel in template editor, and any agent without separate model config.",
        "placeholder": "Select a provider · model as the fallback brain",
        "title": "Default Primary Model"
      },
      "profiles": {
        "actions": {
          "setCurrent": "Set current"
        },
        "current": "Current",
        "delete": {
          "content": "Are you sure you want to delete this config?",
          "title": "Delete config"
        }
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "Custom (OpenAI-compatible)",
        "deepseek": "DeepSeek",
        "doubao": "Doubao (Volcano Ark)",
        "emptyHint": "Please first configure API Key and available models in ",
        "emptyHintTail": ".",
        "emptyTitle": "No enabled providers yet",
        "enabledTitle": "Enabled providers",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "models",
        "moonshot": "Moonshot (Kimi)",
        "noModels": "No available models configured",
        "openai": "OpenAI",
        "openai_compatible": "Custom (OpenAI-compatible)",
        "openrouter": "OpenRouter",
        "qwen": "Tongyi Qianwen / DashScope",
        "siliconflow": "SiliconFlow",
        "zhipu": "Zhipu AI"
      },
      "sections": {
        "advanced": "Advanced Params",
        "advancedHint": "Adjust only if you understand the meaning; defaults fit most scenarios",
        "basic": "Basic Info",
        "connection": "Connection Config",
        "connectionApiKeyLink": "Go to apply / manage API Key for this provider"
      },
      "tabs": {
        "agents": "Agent Config",
        "config": "Model Config"
      },
      "validation": {
        "apiKeyRequired": "API Key cannot be empty",
        "baseUrlNoChatCompletionsSuffix": "Base URL should not end with /chat/completions (system will auto-append)",
        "baseUrlProtocol": "Base URL must start with http:// or https://",
        "baseUrlRequired": "Base URL cannot be empty",
        "modelFormat": "Invalid model format",
        "modelRequired": "Model cannot be empty",
        "nameRequired": "Name cannot be empty"
      }
    }
  }
} as const;
export default AiSettings;
