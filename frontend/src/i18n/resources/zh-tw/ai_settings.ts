// Auto-generated from proto/ant/v1/i18n/ai_settings_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "agent": {
        "defaults": {
          "code": {
            "inputHint": "範例：目標範式=趨勢跟隨；指標=EMA(fast)/EMA(slow)+ATR 過濾"
          },
          "execution": {
            "inputHint": "範例：訂單=做多 EURUSD 10 手；目前點差=0.6 pip"
          },
          "executor": {
            "identity": "交易執行優化專家 — 最小化滑點和執行成本。"
          },
          "macro": {
            "inputHint": "範例：本週關鍵事件=美國 CPI（週四 20:30）"
          },
          "portfolio": {
            "inputHint": "範例：已有策略=趨勢-EURUSD、均值回歸-XAUUSD"
          },
          "researcher": {
            "identity": "宏觀經濟和行業研究員 — 分析宏觀事件和行業趨勢。"
          },
          "risk": {
            "inputHint": "範例：帳戶權益=10,000；可接受月回撤=5%；單筆風險=0.5%"
          },
          "risk_manager": {
            "identity": "嚴格的風險控制專家 — 設計倉位管理、止損、回撤限制。"
          },
          "sentiment": {
            "inputHint": "範例：近 1 週 VIX 從 14 升至 22"
          },
          "signals": {
            "inputHint": "範例：範式=趨勢跟隨；週期=H1；可用指標=EMA/ATR/ADX"
          },
          "strategist": {
            "identity": "資深量化交易策略師 — 根據帳戶和市場狀況推薦策略範式。"
          },
          "style": {
            "inputHint": "範例：帳戶=EURUSD 零售；週期=H1；目標=月均收益 3%、最大回撤 <10%"
          }
        },
        "actions": {
          "add": "新增",
          "loadDefaults": "載入預設 8 個 智能體",
          "remove": "刪除",
          "restoreDefaults": "恢復預設",
          "restoreDefaultsConfirmContent": "將把 8 個系統 智能體 重置為預設身份定義。",
          "restoreDefaultsConfirmTitle": "恢復系統預設身份？",
          "save": "儲存"
        },
        "fields": {
          "historicalBinding": "{{value}}（歷史）",
          "identityPlaceholder": "身份/人設描述",
          "inputHintPlaceholder": "輸入提示（可選）",
          "modelProfileEmpty": "請先在「AI 設定」啟用至少一個 廠商/模型",
          "modelProfilePlaceholder": "預設（沿用目前設定檔）",
          "namePlaceholder": "智能體 名稱"
        },
        "messages": {
          "defaultsLoaded": "已載入系統預設 智能體 模板",
          "empty": "暫無自定義 智能體，點選\"新增\"開始設定",
          "loading": "載入中...",
          "saveFailed": "智能體 儲存失敗",
          "saveSuccess": "智能體 已儲存",
          "selectProfileFirst": "請先在左側選擇一個設定"
        },
        "types": {
          "code": "程式碼",
          "execution": "執行",
          "executor": "執行顧問",
          "macro": "巨集觀",
          "portfolio": "組合",
          "researcher": "市場研究員",
          "risk": "風控",
          "risk_manager": "風控經理",
          "sentiment": "情緒",
          "signals": "訊號/指標",
          "strategist": "策略分析師",
          "style": "風格/範式"
        },
        "defaultName": "自定義 智能體",
        "removeConfirmContent": "確定要刪除該 智能體 嗎？",
        "removeConfirmTitle": "刪除 智能體",
        "title": "智能體 身份定義"
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "開啟 DeepSeek 平臺：",
          "step2": "登入/註冊後在 API 金鑰s 頁面建立並複製 API 金鑰",
          "title": "如何獲取 DeepSeek API 金鑰"
        },
        "zhipu": {
          "step1": "開啟智譜開放平臺：",
          "step2": "登入/註冊後進入控制檯，建立並複製 API 金鑰",
          "title": "如何獲取智譜 API 金鑰"
        },
        "default": "Current provider: {{provider}}. Go to the provider\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\",
        "modelSuggestionDeepSeek": "模型建議: 在\"模型\"下拉中選擇 `deepseek-chat`",
        "modelSuggestionZhipu": "模型建議: 在\"模型\"下拉中選擇 `glm-4-flash` / `glm-4`",
        "selectProviderHint": "選擇一個 AI 提供商後，會在這裡顯示如何申請 API 金鑰。",
        "title": "申請 API 金鑰 指引"
      },
      "profiles": {
        "actions": {
          "setCurrent": "設為目前"
        },
        "delete": {
          "content": "確定要刪除該設定嗎？",
          "title": "刪除設定"
        },
        "current": "目前"
      },
      "actions": {
        "saveConfig": "儲存設定",
        "validateApiKey": "驗證 API 金鑰"
      },
      "discoverErrors": {
        "baseUrlInvalid": "Base URL 格式無效：請填寫完整位址，例如 https://model.example.com 或 https://model.example.com/v1",
        "baseUrlRequired": "請先填寫 Base URL（模型服務位址）。",
        "endpoint404": "模型端點不存在：請確認 Base URL 與服務協定相符（部分服務需要 /v1）。",
        "freeTierExhausted": "免費額度已耗盡：請在廠商控制台關閉「僅使用免費檔」或更換付費 Key。",
        "generic": "拉取模型失敗，請檢查 Base URL 與金鑰設定。",
        "genericDetail": "拉取模型失敗：{{detail}}",
        "invalidModelsResponse": "模型服務回傳格式不相容 /models 協定。",
        "noModelsReturned": "模型服務未回傳可用模型，請檢查帳號權限或服務設定。",
        "quotaForbidden403": "呼叫被拒（配額受限）：請檢查廠商控制台的計費/配額狀態。",
        "quotaOrRateLimit": "配額受限或被限流：廠商已拒絕呼叫，請檢查計費/速率限制或稍後重試。",
        "timeout": "請求逾時：請檢查網路連通性或稍後重試。",
        "unauthorized": "鑑權失敗：請檢查 API Key/Secret 是否正確。",
        "unreachable": "無法連線到模型服務：請檢查 Base URL、網路或閘道。"
      },
      "errors": {
        "arrearage": "服務商返回：帳號欠費/餘額不足。請到服務商控制檯檢查。",
        "forbidden": "服務商返回：訪問被拒絕（403）。請檢查 Key 權限。",
        "invalidModelId": "服務商返回：模型不可用{{model}}。請從下拉列表選擇。",
        "timeout": "連線逾時。請檢查 基礎網址 是否可訪問。",
        "unauthorized": "服務商返回：API 金鑰無效（401）。請檢查 Key 是否正確。"
      },
      "fields": {
        "apiKey": "API 金鑰",
        "apiKeyConfigured": "已設定",
        "apiKeyReplaceHint": "如需更換金鑰，請重新輸入",
        "availableModels": "可用模型",
        "availableModelsEmpty": "直接輸入 模型 ID",
        "availableModelsHint": "同一 API 金鑰 下可同時啟用多個 模型。",
        "availableModelsPlaceholder": "選擇或手動輸入 模型 ID",
        "availableModelsTip": "提示：刪除某個模型不會立即清空已繫結的 智能體。",
        "baseUrl": "基礎網址",
        "baseUrlHint": "（模型服務位址）",
        "clear": "清空",
        "defaultModel": "預設模型",
        "deleteApiKey": "刪除金鑰",
        "enabledOff": "已停用 → 點選啟用",
        "enabledOn": "已啟用 → 點選關閉",
        "enabledStatus": "已啟用",
        "maxTokens": "最大 權杖 數",
        "model": "模型",
        "name": "名稱",
        "provider": "AI 提供商",
        "temperature": "溫度",
        "timeoutSeconds": "逾時秒數"
      },
      "inferenceParams": {
        "title": "推理參數"
      },
      "messages": {
        "apiKeyValidated": "API 金鑰 驗證成功",
        "deleted": "已刪除",
        "disabled": "已停用",
        "enabled": "已啟用",
        "loadConfigFailed": "載入 AI 設定失敗",
        "probeFailed": "連線失敗",
        "probeSuccess": "連線成功",
        "saveSuccess": "設定儲存成功",
        "selectSavedProfileOrEnterKey": "請先選擇一個已儲存的設定，或輸入 API 金鑰",
        "setCurrentSuccess": "已切換目前設定",
        "validateBeforeSave": "請先點選「驗證 API 金鑰」",
        "validateFailed": "驗證失敗",
        "validateSuccess": "驗證成功"
      },
      "placeholders": {
        "apiKey": "輸入 API 金鑰",
        "baseUrl": "例如：https://api.example.com/v1",
        "modelManual": "請輸入模型名稱",
        "modelSelect": "請選擇模型",
        "modelSelectOrType": "從下拉選擇，或直接輸入 模型 ID",
        "name": "例如：DeepSeek-低成本",
        "provider": "選擇 AI 提供商",
        "providerFirst": "請先選擇 AI 提供商"
      },
      "primary": {
        "hint": "用於「釐清意圖」步驟、程式碼生成等。",
        "placeholder": "選擇廠商 · 模型作為兜底大腦",
        "title": "預設主模型"
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "自定義（OpenAI 相容）",
        "deepseek": "DeepSeek",
        "doubao": "豆包 Doubao",
        "emptyHint": "請先在 ",
        "emptyHintTail": "。",
        "emptyTitle": "尚無啟用的廠商",
        "enabledTitle": "已啟用廠商",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "個模型",
        "moonshot": "月之暗面 Moonshot (Kimi)",
        "noModels": "未設定可用模型",
        "openai": "OpenAI",
        "openai_compatible": "自定義（OpenAI 相容）",
        "openrouter": "OpenRouter",
        "qwen": "通義千問 / DashScope",
        "siliconflow": "矽基流動 SiliconFlow",
        "zhipu": "智譜 AI"
      },
      "sections": {
        "advanced": "高階參數",
        "advancedHint": "僅在瞭解含義時調整；預設值已適配大多數場景",
        "basic": "基礎資訊",
        "connection": "連線設定",
        "connectionApiKeyLink": "前往申請 / 管理該廠商 API 金鑰"
      },
      "tabs": {
        "agents": "智能體 設定",
        "config": "模型設定"
      },
      "validation": {
        "apiKeyRequired": "API 金鑰 不能為空",
        "baseUrlNoChatCompletionsSuffix": "基礎網址 不要以 /chat/completions 結尾",
        "baseUrlProtocol": "基礎網址 必須以 http:// 或 https:// 開頭",
        "baseUrlRequired": "基礎網址 不能為空",
        "modelFormat": "模型格式不正確",
        "modelRequired": "模型不能為空",
        "nameRequired": "名稱不能為空"
      },
      "apiKeySavedAs": "目前已儲存：{{masked}}",
      "defaultProfileName": "預設",
      "pageTitle": "AI 助手設定"
    }
  }
} as const;
export default AiSettings;
