// Auto-generated from proto/ant/v1/i18n/ai_core_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiCore = {
  "ai": {
    "agentPrompts": {
      "code": {
        "title": "代码生成 Agent"
      },
      "risk": {
        "title": "风控与执行约束"
      },
      "signals": {
        "title": "信号与指标设计"
      },
      "style": {
        "title": "市场状态/风格推荐"
      }
    },
    "assistant": {
      "messages": {
        "noCodeBlockFound": "No code block found (\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`...\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\`)"
      }
    },
    "backtestScoreCard": {
      "backendRiskScore": {
        "empty": "无（先保存模板，回测完成后自动计算）",
        "loading": "计算中...",
        "reasons": "原因",
        "reliable": "可靠",
        "title": "策略风险评分",
        "unknown": "未知",
        "unreliable": "不可靠",
        "warnings": "警告"
      },
      "chart": {
        "title": "净值曲线"
      },
      "level": {
        "excellent": "优秀",
        "fair": "一般",
        "good": "良好",
        "poor": "差"
      },
      "metrics": {
        "annualReturn": "年化收益",
        "equityPoints": "净值点数",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普",
        "totalReturn": "总收益",
        "totalTrades": "交易次数",
        "winRate": "胜率"
      },
      "recommendation": {
        "cautious": "谨慎上线：建议先小资金/手动确认运行一段时间。",
        "loading": "风险评估计算中，建议先等待完成再上线。",
        "notRecommended": "Not recommended for direct live: high risk or unreliable, optimize before trying.",
        "recommended": "推荐上线：风险可控，指标整体健康。"
      },
      "score": {
        "empty": "暂无评分（等待回测完成或无 metrics）",
        "title": "综合评分（启发式）"
      },
      "stateLabel": "状态",
      "status": {
        "cancelRequested": "取消中",
        "canceled": "已取消",
        "failed": "失败",
        "pending": "排队中",
        "running": "运行中",
        "succeeded": "成功"
      },
      "title": "回测评分卡"
    },
    "chatBox": {
      "collapse": "收起",
      "emptyDescription": "开始与AI助手对话",
      "expandAll": "展开全部",
      "thinking": "思考中...",
      "truncated": "内容过长，已截断"
    },
    "client": {
      "errors": {
        "contentBlocked": "服务商安全过滤器阻止了响应。请改写提示词后重试。",
        "contextTooLong": "请求超出模型上下文窗口。请缩短对话/输入，或选择上下文更大的模型。",
        "edgeGatewayTimeout": "边缘网关超时 (通常为 Cloudflare HTTP 524)：浏览器未收到应用响应，长时运行操作常见。请重试；如持续出现，请联系运维提高代理/源站超时。",
        "forbidden": "服务商拒绝了请求 (403)。请检查 Key 权限、IP 白名单和账户状态。",
        "gatewayForbidden403": "网关被禁止 (403)。",
        "gatewayRateLimited429": "网关速率受限 (429)。",
        "gatewayTimeoutOrUnreachable": "网关超时或不可达。",
        "gatewayUnauthorized401": "网关未授权 (401)。",
        "insufficientBalance": "服务商报告余额为空/欠费。请在服务商控制台充值后重试。",
        "invalidModelId": "模型不可用{{model}} — 可能输入错误、已弃用或超出您的套餐。请从下拉列表中选择其他模型，或从服务商控制台复制标准 ID。",
        "networkUnreachable": "网关超时或不可达。请检查 Base URL、网络连接，或稍后重试。",
        "providerInternalError": "服务商返回服务器错误 (5xx)。请稍候或切换到其他服务商。",
        "rateLimited": "服务商正在限制您的请求频率，请稍候再试。",
        "regionNotSupported": "所选服务商在您所在地区/国家不可用，请切换到其他服务商。",
        "requestFailed": "请求失败，请重试。",
        "unauthorized": "服务商拒绝了 API Key (401)。请检查 Key 值及其是否有权访问所选模型。"
      }
    },
    "consensus": {
      "actions": {
        "refresh": "刷新"
      },
      "fields": {
        "account": "账号",
        "symbol": "品种",
        "timeframe": "周期"
      },
      "panel": {
        "decision": "决策",
        "overallScore": "总体分",
        "technicalScore": "技术面",
        "title": "客观评分"
      },
      "signals": {
        "ma": {
          "trend": "均线趋势"
        },
        "macd": {
          "flag": "信号",
          "hist": "柱体",
          "signalLine": "信号线",
          "trend": "形态",
          "value": "MACD"
        },
        "rsi": {
          "flag": "信号",
          "value": "RSI"
        }
      },
      "title": "共识与对话"
    },
    "conversation": {
      "defaultTitle": "新对话"
    },
    "gate": {
      "allPassed": "所有 6 个 Gate 通过，策略可进入 PromoteToLive 评估",
      "backtestGrossReturn": "回测毛收益",
      "backtestNetReturn": "回测净收益",
      "dailyReturns": "日收益率 (逗号或换行分隔)",
      "descriptions": {
        "compliance": "DSL 表达式非空验证",
        "correlation": "与现有策略的信号相关性检查",
        "deflated_sharpe": "Lopez de Prado 紧缩夏普比率",
        "lookahead": "扫描未来函数引用 (close[t+N], ref 负偏移)",
        "paper": "≥14 天模拟交易验证",
        "walkforward": "Purged Walk-Forward 交叉验证"
      },
      "details": "详细结果",
      "dslExpression": "DSL 表达式",
      "evaluating": "评估中...",
      "fail": "失败",
      "failed": "未通过: {{gate}}",
      "gateProgress": "Gate 评估进度",
      "labels": {
        "compliance": "合规检查",
        "correlation": "相关性",
        "deflated_sharpe": "通缩夏普比率",
        "lookahead": "前视偏差",
        "paper": "模拟交易",
        "walkforward": "前向分析"
      },
      "noData": "无数据",
      "numAttempts": "策略尝试次数",
      "paperDays": "模拟天数",
      "paperMetrics": "模拟交易指标",
      "paperNetPnL": "模拟 Net P&L",
      "paperNetReturn": "模拟净收益",
      "paperTradeCount": "模拟交易数",
      "pass": "通过",
      "pipelineDesc": "6 级 Gate 管道: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation",
      "pipelineResult": "管道结果",
      "retry": "重试",
      "runHint": "请先运行回测，然后点击\"运行质量门\"评估策略质量。",
      "runPipeline": "运行 Gate 管道",
      "selectRun": "选择回测运行...",
      "skipped": "已跳过",
      "status": {
        "evaluating": "评估中..."
      },
      "strategyParams": "策略参数",
      "title": "AI Gate 进度面板",
      "unknown": "未知"
    },
    "gateway": {
      "balance": "钱包余额",
      "modelPlaceholder": "选择 AI 模型",
      "monthlyCost": "本月费用",
      "monthlyTokens": "本月 Token",
      "noModels": "暂无可用模型",
      "selectModel": "选择模型",
      "title": "AI 网关",
      "usageByFeature": "按功能用量",
      "useGateway": "AI 网关",
      "useGatewayDesc": "扣钱包余额 · 按 Token 计费",
      "useOwnKey": "我的 API Key",
      "useOwnKeyDesc": "直付厂商 · 自行管理",
      "useOwnKeyHint": "使用你自己的 API Key，直接向所选厂商付费。在下方选择厂商卡片进行配置。"
    },
    "reports": {
      "tradeAnalysis": {
        "riskAssessmentPrefix": "风险评估:",
        "title": "AI交易分析报告"
      }
    },
    "requireConfig": {
      "actions": {
        "goSettings": "前往设置"
      },
      "description": "请先到设置页配置 AI 提供商、模型与 API Key，然后再使用策略向导或聊天。",
      "title": "尚未配置大模型"
    },
    "riskEval": {
      "failed": "风险评估失败"
    },
    "signalCard": {
      "actions": {
        "cancel": "取消",
        "confirm": "确认",
        "executeTrade": "执行交易"
      },
      "confirmCancel": {
        "title": "确定要取消此信号？"
      },
      "confirmExecute": {
        "description": "将立即下单",
        "title": "确定要执行这个交易信号吗?"
      },
      "labels": {
        "analysisReason": "分析理由",
        "confidence": "信心度",
        "price": "价格",
        "stopLoss": "止损",
        "takeProfit": "止盈",
        "volume": "手数"
      },
      "status": {
        "cancelled": "已取消",
        "confirmed": "已确认",
        "executed": "已执行",
        "pending": "待确认"
      }
    },
    "strategyCard": {
      "actionType": {
        "alert": "警报",
        "buy": "买入",
        "closeLong": "平多",
        "closeShort": "平空",
        "sell": "卖出"
      },
      "actions": {
        "start": "启动",
        "stop": "停止"
      },
      "confirmDelete": {
        "description": "删除后无法恢复",
        "title": "确定要删除这个策略吗?"
      },
      "labels": {
        "lastTriggeredAt": "最近触发: {{time}}",
        "triggeredCount": "触发 {{count}} 次"
      },
      "sections": {
        "actions": "操作",
        "conditions": "触发条件"
      },
      "status": {
        "active": "运行中",
        "inactive": "已停止",
        "paused": "已暂停"
      },
      "tooltips": {
        "createdAt": "创建时间",
        "lastTriggeredAt": "最近触发"
      }
    },
    "systemAI": {
      "cardState": {
        "enabled": "已启用",
        "noKey": "未配置",
        "noModel": "待选模型",
        "readyDisabled": "就绪 · 已禁用"
      },
      "cardTags": {
        "current": "当前",
        "enabledButUnavailable": "已启用但不可用",
        "hasKey": "已配密钥",
        "noKey": "未配密钥",
        "noModels": "未配置可用模型"
      },
      "customProvider": {
        "deleted": "自定义提供商已删除",
        "fillNameFirst": "请先填写名称",
        "nameHint": "用于识别此提供商的唯一名称",
        "nameLabel": "提供商名称",
        "namePlaceholder": "我的自定义提供商",
        "nameRequired": "服务商名称不能为空"
      },
      "emptyConfigs": "暂无 AI Provider 配置（系统启动时会自动创建默认 Provider）",
      "fields": {
        "apiKeyHint": "输入后将自动加密保存，无需手动提交",
        "apiKeyPastePlaceholder": "粘贴 API Key，将自动预保存",
        "autoFetching": "自动拉取中",
        "baseUrlCustomHint": "输入 OpenAI 兼容端点，例如 https://model.example.com/v1",
        "baseUrlCustomPlaceholder": "例如: https://model.example.com/v1",
        "baseUrlReadonlyHint": "官方地址由系统维护，不可修改",
        "baseUrlReadonlyPlaceholder": "官方地址（只读）",
        "enabledHint": "关闭后该厂商不参与系统路由",
        "httpWarning": "当前为 HTTP，生产环境建议使用 HTTPS",
        "maxTokensHint": "单次响应最大 token 数",
        "primaryFor": "主要用途（Primary For）",
        "primaryForHint": "用于内部分发：对话/嵌入/摘要/推理",
        "temperatureHint": "越高越发散，越低越稳定",
        "timeoutHint": "单次请求最长等待时间"
      },
      "messages": {
        "autoDiscoveredModels": "已自动发现 {{count}} 个模型（仅作选择建议）",
        "autoValidatedModels": "已自动验证：发现 {{count}} 个模型",
        "configSaveFailed": "配置保存失败",
        "configSaved": "配置已保存",
        "deleteSecretFailed": "删除密钥失败",
        "loadConfigFailed": "加载配置失败",
        "secretAutoSaveFailed": "密钥自动保存失败",
        "secretDeletedConfigReset": "密钥已删除，厂商配置已恢复默认初始化",
        "secretSavedAutoDiscover": "密钥已保存，正在自动发现模型...",
        "toggleEnabledFailed": "更新启用状态失败",
        "validationFailedNeedApiKey": "验证失败：此服务商通常需要 API Key，请先填写并保存 Key 后重试。",
        "validationPassedModels": "验证通过：发现 {{count}} 个模型"
      },
      "pageSubtitle": "配置 AI 大脑 — 选择模型厂商、管理 API 密钥与可用模型，并指定全站兜底使用的「默认主模型」。",
      "pageTitle": "AI 助手设置",
      "section1": {
        "subtitle": "Cards show each provider's configuration and readiness; click to select",
        "title": "选择模型厂商"
      },
      "status": {
        "checkUrl": "请检查 Base URL",
        "checkUrlDesc": "API Key 已就绪，但地址似乎无效",
        "configReady": "配置已就绪",
        "configReadyDesc": "添加可用模型后系统将自动完成连通性检测",
        "connectionFailed": "连接错误，请检查上方提示",
        "error": "存在异常",
        "needKey": "请完成密钥配置",
        "needKeyDesc": "填写 API Key 后将自动发现模型列表",
        "noProvider": "尚未选择厂商",
        "noProviderDesc": "请从下方卡片挑选一个模型厂商开始配置",
        "notEnabled": "连接正常，尚未启用",
        "notEnabledDesc": "打开「启用」开关即可投入使用",
        "ready": "运行就绪",
        "readyDesc": "已启用并连接正常"
      },
      "statusBar": {
        "checking": "连通性检测中…",
        "connected": "已连接",
        "disabled": "未启用",
        "enabled": "已启用",
        "keyReady": "密钥就绪"
      },
      "taglines": {
        "anthropic": "Claude 系列",
        "deepseek": "深度求索 · 高性价比",
        "moonshot": "Kimi · 长上下文",
        "openai": "GPT 系列 · 官方",
        "openai_compatible": "任意兼容端点",
        "qwen": "阿里云 · 中文优化",
        "zhipu": "清华系 · 通用"
      }
    },
    "tabs": {
      "agentSettings": "专家设置",
      "gate": "AI 质量门",
      "settings": "设置"
    },
    "workflowRuns": {
      "defaultTitle": "AI 工作流",
      "hints": {
        "selectToViewDetail": "从左侧选择运行记录查看详情"
      },
      "messages": {
        "loadDetailFailed": "加载详情失败",
        "loadListFailed": "加载运行记录失败"
      },
      "title": "AI 工作流"
    }
  }
} as const;
export default AiCore;
