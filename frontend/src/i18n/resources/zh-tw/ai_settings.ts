// Auto-generated from proto/ant/v1/i18n/ai_settings_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "agent": {
        "defaults": {
          "code": {
            "inputHint": "示例：目標正規化=趨勢跟隨；指標=EMA(fast)/EMA(slow)+ATR 過濾；引數=fast,slow,atr_period,risk_per_trade。"
          },
          "execution": {
            "inputHint": "示例：訂單=做多 EURUSD 10 手；當前點差=0.6 pip；目標 5 分鐘內完成；可接受滑點=0.8 pip。"
          },
          "executor": {
            "identity": "交易執行最佳化專家 — 最小化滑點和執行成本。"
          },
          "macro": {
            "inputHint": "示例：本週關鍵事件=美國 CPI（週四 20:30）、FOMC 紀要（週三次日 02:00）；目標品種=XAUUSD。"
          },
          "portfolio": {
            "inputHint": "示例：已有策略=趨勢-EURUSD、均值迴歸-XAUUSD；總權益=50,000；目標年化波動=12%。"
          },
          "researcher": {
            "identity": "宏觀經濟和行業研究員 — 分析宏觀事件和行業趨勢。"
          },
          "risk": {
            "inputHint": "示例：賬戶權益=10,000；可接受月回撤=5%；單筆風險=0.5%；日內交易上限=5；止損= 1.5×ATR。"
          },
          "risk_manager": {
            "identity": "嚴格的風險控制專家 — 設計倉位管理、止損、回撤限制。"
          },
          "sentiment": {
            "inputHint": "示例：近 1 周 VIX 從 14 升至 22；非商業淨多持倉環比 -18%；新聞以\"衰退/降息\"關鍵詞為主。"
          },
          "signals": {
            "inputHint": "示例：正規化=趨勢跟隨；週期=H1；可用指標=EMA/ATR/ADX；引數 fast 預設 20、slow 預設 60。"
          },
          "strategist": {
            "identity": "資深量化交易策略師 — 根據賬戶和市場狀況推薦策略正規化。"
          },
          "style": {
            "inputHint": "示例：賬戶=EURUSD 零售；週期=H1；目標=月均收益 3%、最大回撤 <10%；偏好=勝率優先於盈虧比。"
          }
        },
        "actions": {
          "add": "新增",
          "loadDefaults": "載入預設 8 個 Agent",
          "remove": "刪除",
          "restoreDefaults": "恢復預設",
          "restoreDefaultsConfirmContent": "將把 8 個系統 Agent（風格/訊號/風控/宏觀/情緒/組合/執行/程式碼）重置為預設身份定義，你自行新增的 Agent 會被保留。該操作僅修改未儲存的草稿，點選\"儲存\"後才會落庫。",
          "restoreDefaultsConfirmTitle": "恢復系統預設身份？",
          "save": "儲存"
        },
        "fields": {
          "historicalBinding": "{{value}}（歷史繫結）",
          "identityPlaceholder": "身份/人設描述（會拼接到 system prompt）",
          "inputHintPlaceholder": "輸入提示（可選）",
          "modelProfileEmpty": "請先在「AI 設定」啟用至少一個 provider/模型",
          "modelProfilePlaceholder": "預設（沿用當前配置檔）",
          "namePlaceholder": "Agent 名稱"
        },
        "messages": {
          "defaultsLoaded": "已載入系統預設 Agent 模板，請點選\"儲存\"落庫",
          "empty": "暫無自定義 Agent，點選\"新增\"開始配置",
          "loading": "載入中...",
          "saveFailed": "Agent 儲存失敗",
          "saveSuccess": "Agent 已儲存",
          "selectProfileFirst": "請先在左側選擇一個配置"
        },
        "types": {
          "code": "程式碼",
          "execution": "執行",
          "executor": "執行顧問",
          "macro": "宏觀",
          "portfolio": "組合",
          "researcher": "市場研究員",
          "risk": "風控",
          "risk_manager": "風控經理",
          "sentiment": "情緒",
          "signals": "訊號/指標",
          "strategist": "策略分析師",
          "style": "風格/正規化"
        },
        "defaultName": "自定義 Agent",
        "removeConfirmContent": "確定要刪除該 Agent 嗎？",
        "removeConfirmTitle": "刪除 Agent",
        "title": "Agent 身份定義"
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "開啟 DeepSeek 平臺：",
          "step2": "登入/註冊後在 API Keys 頁面建立並複製 API Key",
          "title": "如何獲取 DeepSeek API Key"
        },
        "zhipu": {
          "step1": "開啟智譜開放平臺：",
          "step2": "登入/註冊後進入控制檯，建立並複製 API Key",
          "title": "如何獲取智譜 API Key"
        },
        "default": "當前提供商：{{provider}}。前往該提供商\\\\\\\\\\\\\\\\",
        "modelSuggestionDeepSeek": "模型建議: 在\"模型\"下拉中選擇 `deepseek-chat`",
        "modelSuggestionZhipu": "模型建議: 在\"模型\"下拉中選擇 `glm-4-flash` / `glm-4`",
        "selectProviderHint": "選擇一個 AI 提供商後，會在這裡顯示如何申請 API Key。",
        "title": "申請 API Key 指引"
      },
      "profiles": {
        "actions": {
          "setCurrent": "設為當前"
        },
        "delete": {
          "content": "確定要刪除該配置嗎？",
          "title": "刪除配置"
        },
        "current": "當前"
      },
      "actions": {
        "saveConfig": "儲存配置",
        "validateApiKey": "驗證 API Key"
      },
      "discoverErrors": {
        "baseUrlInvalid": "Base URL 格式無效：請填寫完整地址，例如 https://model.example.com 或 https://model.example.com/v1",
        "baseUrlRequired": "請先填寫 Base URL（模型服務地址）。",
        "endpoint404": "模型端點不存在：請確認 Base URL 與服務協議匹配（部分服務需要 /v1）。",
        "freeTierExhausted": "免費額度已耗盡：請在廠商控制檯關閉「僅使用免費檔」或更換付費 Key。",
        "generic": "拉取模型失敗，請檢查 Base URL 與金鑰配置。",
        "genericDetail": "拉取模型失敗：{{detail}}",
        "invalidModelsResponse": "模型服務返回格式不相容 /models 協議。",
        "noModelsReturned": "模型服務未返回可用模型，請檢查賬號許可權或服務配置。",
        "quotaForbidden403": "呼叫被拒（配額受限）：請檢查廠商控制檯的計費/配額狀態。",
        "quotaOrRateLimit": "配額受限或被限流：廠商已拒絕呼叫，請檢查計費/速率限制或稍後重試。",
        "timeout": "請求超時：請檢查網路連通性或稍後重試。",
        "unauthorized": "鑑權失敗：請檢查 API Key/Secret 是否正確。",
        "unreachable": "無法連線到模型服務：請檢查 Base URL、網路或閘道器。"
      },
      "errors": {
        "arrearage": "服務商返回：賬號欠費/餘額不足或賬戶狀態異常。請到服務商控制檯檢查餘額、賬單與賬戶狀態後重試。",
        "forbidden": "服務商返回：訪問被拒絕（403）。請檢查 Key 許可權、IP 白名單或賬號狀態。",
        "invalidModelId": "服務商返回：模型不可用{{model}}。請從下拉選單選擇，或到服務商控制檯複製正確的 model id。",
        "timeout": "連線超時。請檢查 Base URL 是否可訪問、網路是否通暢，或稍後重試。",
        "unauthorized": "服務商返回：API Key 無效或未授權（401）。請檢查 Key 是否正確、是否有該模型許可權。"
      },
      "fields": {
        "apiKey": "API Key",
        "apiKeyConfigured": "已配置",
        "apiKeyReplaceHint": "如需更換金鑰，請重新輸入",
        "availableModels": "可用模型",
        "availableModelsEmpty": "直接輸入 model id 後回車即可加入",
        "availableModelsHint": "同一 API Key 下可同時啟用多個 model；這裡的清單會出現在 /ai/agents 的下拉里。預設空白，從下拉選擇或手動輸入 model id 後回車新增；只加入顯式選過的，不會自動併入全部已發現模型。",
        "availableModelsPlaceholder": "選擇或手動輸入 model id 後回車新增（預設空白）",
        "availableModelsTip": "提示：刪除某個模型不會立即清空 /ai/agents 中已繫結它的 Agent，但會將它從下拉建議中移除。",
        "baseUrl": "Base URL",
        "baseUrlHint": "（模型服務地址）",
        "clear": "清空",
        "defaultModel": "預設模型",
        "deleteApiKey": "刪除金鑰",
        "enabledOff": "已停用 → 點選啟用",
        "enabledOn": "已啟用 → 點選關閉",
        "enabledStatus": "已啟用",
        "maxTokens": "最大 Token 數",
        "model": "模型",
        "name": "名稱",
        "provider": "AI 提供商",
        "temperature": "溫度",
        "timeoutSeconds": "Timeout（秒）"
      },
      "inferenceParams": {
        "title": "推理引數"
      },
      "messages": {
        "apiKeyValidated": "API Key 驗證成功",
        "deleted": "已刪除",
        "disabled": "已停用",
        "enabled": "已啟用",
        "loadConfigFailed": "載入 AI 配置失敗",
        "probeFailed": "連線失敗",
        "probeSuccess": "連線成功",
        "saveSuccess": "配置儲存成功",
        "selectSavedProfileOrEnterKey": "請先選擇一個已儲存的配置，或輸入 API Key",
        "setCurrentSuccess": "已切換當前配置",
        "validateBeforeSave": "請先點選\"驗證 API Key\"，驗證透過後才能儲存",
        "validateFailed": "驗證失敗",
        "validateSuccess": "驗證成功"
      },
      "placeholders": {
        "apiKey": "輸入 API Key",
        "baseUrl": "例如：https://api-inference.modelscope.cn/v1 或 https://ark.cn-beijing.volces.com/api/v3",
        "modelManual": "請輸入模型名稱（建議從服務商控制檯複製 model id）",
        "modelSelect": "請選擇模型",
        "modelSelectOrType": "從下拉選擇，或直接輸入 model id",
        "name": "例如：DeepSeek-低成本",
        "provider": "選擇 AI 提供商",
        "providerFirst": "請先選擇 AI 提供商"
      },
      "primary": {
        "hint": "用於「澄清意圖」步驟、程式碼生成、模板編輯器中的「AI 助手 — 修改程式碼」面板，以及任何未單獨配置模型的 Agent。",
        "placeholder": "選擇一個 provider · model 作為兜底大腦",
        "title": "預設主模型"
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "自定義（OpenAI 相容）",
        "deepseek": "DeepSeek",
        "doubao": "豆包 Doubao (火山方舟)",
        "emptyHint": "請先在 ",
        "emptyHintTail": "。",
        "emptyTitle": "尚無啟用的廠商",
        "enabledTitle": "已啟用廠商",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "個模型",
        "moonshot": "月之暗面 Moonshot (Kimi)",
        "noModels": "未配置可用模型",
        "openai": "OpenAI",
        "openai_compatible": "自定義（OpenAI 相容）",
        "openrouter": "OpenRouter",
        "qwen": "通義千問 / DashScope",
        "siliconflow": "矽基流動 SiliconFlow",
        "zhipu": "智譜 AI"
      },
      "sections": {
        "advanced": "高階引數",
        "advancedHint": "僅在瞭解含義時調整；預設值已適配大多數場景",
        "basic": "基礎資訊",
        "connection": "連線配置",
        "connectionApiKeyLink": "前往申請 / 管理該廠商 API Key"
      },
      "tabs": {
        "agents": "Agent 配置",
        "config": "模型配置"
      },
      "validation": {
        "apiKeyRequired": "API Key 不能為空",
        "baseUrlNoChatCompletionsSuffix": "Base URL 不要以 /chat/completions 結尾（系統會自動拼接 /chat/completions）",
        "baseUrlProtocol": "Base URL 必須以 http:// 或 https:// 開頭",
        "baseUrlRequired": "Base URL 不能為空",
        "modelFormat": "模型格式不正確",
        "modelRequired": "模型不能為空",
        "nameRequired": "名稱不能為空"
      },
      "apiKeySavedAs": "當前已儲存：{{masked}}",
      "defaultProfileName": "預設",
      "pageTitle": "AI 助手設定"
    }
  }
} as const;
export default AiSettings;
