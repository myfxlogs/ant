// Auto-generated from proto/ant/v1/i18n/ai_settings_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "agent": {
        "defaults": {
          "code": {
            "inputHint": "示例：目标范式=趋势跟随；指标=EMA(fast)/EMA(slow)+ATR 过滤；参数=fast,slow,atr_period,risk_per_trade。"
          },
          "execution": {
            "inputHint": "示例：订单=做多 EURUSD 10 手；当前点差=0.6 pip；目标 5 分钟内完成；可接受滑点=0.8 pip。"
          },
          "executor": {
            "identity": "交易执行优化专家 — 最小化滑点和执行成本。"
          },
          "macro": {
            "inputHint": "示例：本周关键事件=美国 CPI（周四 20:30）、FOMC 纪要（周三次日 02:00）；目标品种=XAUUSD。"
          },
          "portfolio": {
            "inputHint": "示例：已有策略=趋势-EURUSD、均值回归-XAUUSD；总权益=50,000；目标年化波动=12%。"
          },
          "researcher": {
            "identity": "宏观经济和行业研究员 — 分析宏观事件和行业趋势。"
          },
          "risk": {
            "inputHint": "示例：账户权益=10,000；可接受月回撤=5%；单笔风险=0.5%；日内交易上限=5；止损= 1.5×ATR。"
          },
          "risk_manager": {
            "identity": "严格的风险控制专家 — 设计仓位管理、止损、回撤限制。"
          },
          "sentiment": {
            "inputHint": "示例：近 1 周 VIX 从 14 升至 22；非商业净多持仓环比 -18%；新闻以\"衰退/降息\"关键词为主。"
          },
          "signals": {
            "inputHint": "示例：范式=趋势跟随；周期=H1；可用指标=EMA/ATR/ADX；参数 fast 默认 20、slow 默认 60。"
          },
          "strategist": {
            "identity": "资深量化交易策略师 — 根据账户和市场状况推荐策略范式。"
          },
          "style": {
            "inputHint": "示例：账户=EURUSD 零售；周期=H1；目标=月均收益 3%、最大回撤 <10%；偏好=胜率优先于盈亏比。"
          }
        },
        "actions": {
          "add": "新增",
          "loadDefaults": "加载默认 8 个 Agent",
          "remove": "删除",
          "restoreDefaults": "恢复默认",
          "restoreDefaultsConfirmContent": "将把 8 个系统 Agent（风格/信号/风控/宏观/情绪/组合/执行/代码）重置为默认身份定义，你自行新增的 Agent 会被保留。该操作仅修改未保存的草稿，点击\"保存\"后才会落库。",
          "restoreDefaultsConfirmTitle": "恢复系统默认身份？",
          "save": "保存"
        },
        "fields": {
          "historicalBinding": "{{value}}（历史绑定）",
          "identityPlaceholder": "身份/人设描述（会拼接到 system prompt）",
          "inputHintPlaceholder": "输入提示（可选）",
          "modelProfileEmpty": "请先在「AI 设置」启用至少一个 provider/模型",
          "modelProfilePlaceholder": "默认（沿用当前配置档）",
          "namePlaceholder": "Agent 名称"
        },
        "messages": {
          "defaultsLoaded": "已载入系统默认 Agent 模板，请点击\"保存\"落库",
          "empty": "暂无自定义 Agent，点击\"新增\"开始配置",
          "loading": "加载中...",
          "saveFailed": "Agent 保存失败",
          "saveSuccess": "Agent 已保存",
          "selectProfileFirst": "请先在左侧选择一个配置"
        },
        "types": {
          "code": "代码",
          "execution": "执行",
          "executor": "执行顾问",
          "macro": "宏观",
          "portfolio": "组合",
          "researcher": "市场研究员",
          "risk": "风控",
          "risk_manager": "风控经理",
          "sentiment": "情绪",
          "signals": "信号/指标",
          "strategist": "策略分析师",
          "style": "风格/范式"
        },
        "defaultName": "自定义 Agent",
        "removeConfirmContent": "确定要删除该 Agent 吗？",
        "removeConfirmTitle": "删除 Agent",
        "title": "Agent 身份定义"
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "打开 DeepSeek 平台：",
          "step2": "登录/注册后在 API Keys 页面创建并复制 API Key",
          "title": "如何获取 DeepSeek API Key"
        },
        "zhipu": {
          "step1": "打开智谱开放平台：",
          "step2": "登录/注册后进入控制台，创建并复制 API Key",
          "title": "如何获取智谱 API Key"
        },
        "default": "Current provider: {{provider}}. Go to the provider\\\\\\\\\\\\\\\\",
        "modelSuggestionDeepSeek": "模型建议: 在\"模型\"下拉中选择 `deepseek-chat`",
        "modelSuggestionZhipu": "模型建议: 在\"模型\"下拉中选择 `glm-4-flash` / `glm-4`",
        "selectProviderHint": "选择一个 AI 提供商后，会在这里显示如何申请 API Key。",
        "title": "申请 API Key 指引"
      },
      "profiles": {
        "actions": {
          "setCurrent": "设为当前"
        },
        "delete": {
          "content": "确定要删除该配置吗？",
          "title": "删除配置"
        },
        "current": "当前"
      },
      "actions": {
        "saveConfig": "保存配置",
        "validateApiKey": "验证 API Key"
      },
      "discoverErrors": {
        "baseUrlInvalid": "Base URL 格式无效：请填写完整地址，例如 https://model.example.com 或 https://model.example.com/v1",
        "baseUrlRequired": "请先填写 Base URL（模型服务地址）。",
        "endpoint404": "模型端点不存在：请确认 Base URL 与服务协议匹配（部分服务需要 /v1）。",
        "freeTierExhausted": "免费额度已耗尽：请在厂商控制台关闭「仅使用免费档」或更换付费 Key。",
        "generic": "拉取模型失败，请检查 Base URL 与密钥配置。",
        "genericDetail": "拉取模型失败：{{detail}}",
        "invalidModelsResponse": "模型服务返回格式不兼容 /models 协议。",
        "noModelsReturned": "模型服务未返回可用模型，请检查账号权限或服务配置。",
        "quotaForbidden403": "调用被拒（配额受限）：请检查厂商控制台的计费/配额状态。",
        "quotaOrRateLimit": "配额受限或被限流：厂商已拒绝调用，请检查计费/速率限制或稍后重试。",
        "timeout": "请求超时：请检查网络连通性或稍后重试。",
        "unauthorized": "鉴权失败：请检查 API Key/Secret 是否正确。",
        "unreachable": "无法连接到模型服务：请检查 Base URL、网络或网关。"
      },
      "errors": {
        "arrearage": "服务商返回：账号欠费/余额不足或账户状态异常。请到服务商控制台检查余额、账单与账户状态后重试。",
        "forbidden": "服务商返回：访问被拒绝（403）。请检查 Key 权限、IP 白名单或账号状态。",
        "invalidModelId": "服务商返回：模型不可用{{model}}。请从下拉列表选择，或到服务商控制台复制正确的 model id。",
        "timeout": "连接超时。请检查 Base URL 是否可访问、网络是否通畅，或稍后重试。",
        "unauthorized": "服务商返回：API Key 无效或未授权（401）。请检查 Key 是否正确、是否有该模型权限。"
      },
      "fields": {
        "apiKey": "API Key",
        "apiKeyConfigured": "已配置",
        "apiKeyReplaceHint": "如需更换密钥，请重新输入",
        "availableModels": "可用模型",
        "availableModelsEmpty": "直接输入 model id 后回车即可加入",
        "availableModelsHint": "同一 API Key 下可同时启用多个 model；这里的清单会出现在 /ai/agents 的下拉里。默认空白，从下拉选择或手动输入 model id 后回车添加；只加入显式选过的，不会自动并入全部已发现模型。",
        "availableModelsPlaceholder": "选择或手动输入 model id 后回车添加（默认空白）",
        "availableModelsTip": "提示：删除某个模型不会立即清空 /ai/agents 中已绑定它的 Agent，但会将它从下拉建议中移除。",
        "baseUrl": "Base URL",
        "baseUrlHint": "（模型服务地址）",
        "clear": "清空",
        "defaultModel": "默认模型",
        "deleteApiKey": "删除密钥",
        "enabledOff": "已停用 → 点击启用",
        "enabledOn": "已启用 → 点击关闭",
        "enabledStatus": "已启用",
        "maxTokens": "最大 Token 数",
        "model": "模型",
        "name": "名称",
        "provider": "AI 提供商",
        "temperature": "温度",
        "timeoutSeconds": "Timeout（秒）"
      },
      "inferenceParams": {
        "title": "推理参数"
      },
      "messages": {
        "apiKeyValidated": "API Key 验证成功",
        "deleted": "已删除",
        "disabled": "已停用",
        "enabled": "已启用",
        "loadConfigFailed": "加载 AI 配置失败",
        "probeFailed": "连接失败",
        "probeSuccess": "连接成功",
        "saveSuccess": "配置保存成功",
        "selectSavedProfileOrEnterKey": "请先选择一个已保存的配置，或输入 API Key",
        "setCurrentSuccess": "已切换当前配置",
        "validateBeforeSave": "请先点击\"验证 API Key\"，验证通过后才能保存",
        "validateFailed": "验证失败",
        "validateSuccess": "验证成功"
      },
      "placeholders": {
        "apiKey": "输入 API Key",
        "baseUrl": "例如：https://api-inference.modelscope.cn/v1 或 https://ark.cn-beijing.volces.com/api/v3",
        "modelManual": "请输入模型名称（建议从服务商控制台复制 model id）",
        "modelSelect": "请选择模型",
        "modelSelectOrType": "从下拉选择，或直接输入 model id",
        "name": "例如：DeepSeek-低成本",
        "provider": "选择 AI 提供商",
        "providerFirst": "请先选择 AI 提供商"
      },
      "primary": {
        "hint": "用于「澄清意图」步骤、代码生成、模板编辑器中的「AI 助手 — 修改代码」面板，以及任何未单独配置模型的 Agent。",
        "placeholder": "选择一个 provider · model 作为兜底大脑",
        "title": "默认主模型"
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "自定义（OpenAI 兼容）",
        "deepseek": "DeepSeek",
        "doubao": "豆包 Doubao (火山方舟)",
        "emptyHint": "请先在 ",
        "emptyHintTail": "。",
        "emptyTitle": "尚无启用的厂商",
        "enabledTitle": "已启用厂商",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "个模型",
        "moonshot": "月之暗面 Moonshot (Kimi)",
        "noModels": "未配置可用模型",
        "openai": "OpenAI",
        "openai_compatible": "自定义（OpenAI 兼容）",
        "openrouter": "OpenRouter",
        "qwen": "通义千问 / DashScope",
        "siliconflow": "硅基流动 SiliconFlow",
        "zhipu": "智谱 AI"
      },
      "sections": {
        "advanced": "高级参数",
        "advancedHint": "仅在了解含义时调整；默认值已适配大多数场景",
        "basic": "基础信息",
        "connection": "连接配置",
        "connectionApiKeyLink": "前往申请 / 管理该厂商 API Key"
      },
      "tabs": {
        "agents": "Agent 配置",
        "config": "模型配置"
      },
      "validation": {
        "apiKeyRequired": "API Key 不能为空",
        "baseUrlNoChatCompletionsSuffix": "Base URL 不要以 /chat/completions 结尾（系统会自动拼接 /chat/completions）",
        "baseUrlProtocol": "Base URL 必须以 http:// 或 https:// 开头",
        "baseUrlRequired": "Base URL 不能为空",
        "modelFormat": "模型格式不正确",
        "modelRequired": "模型不能为空",
        "nameRequired": "名称不能为空"
      },
      "apiKeySavedAs": "当前已保存：{{masked}}",
      "defaultProfileName": "默认",
      "pageTitle": "AI 助手设置"
    }
  }
} as const;
export default AiSettings;
