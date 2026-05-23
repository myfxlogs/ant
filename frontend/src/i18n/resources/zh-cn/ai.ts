const aiCore = {
  ai: {
    agentPrompts: {
      style: {
        title: '市场状态/风格推荐',
        prompt:
          '你是资深量化投研分析师。请基于以下信息，推荐策略范式：趋势/均值回复/短线，并说明理由、适用条件与不适用场景。\n\n输出要求：用 Markdown，必须包含：\n1) 推理过程：你如何从数据/约束/目标推导（分点）\n2) 结论：主推荐（只能选一个主范式）+ 备选（可选）+ 适用/不适用条件\n3) 风险提示：至少 3 条\n\n{{baseInfo}}',
      },
      signals: {
        title: '信号与指标设计',
        prompt:
          '你是量化因子与信号工程师。请在不依赖外部数据（除非用户提供宏观事件表）的前提下，设计可实现的交易信号。\n\n要求：明确入场/出场/过滤条件，尽量参数化，避免过拟合。\n\n输出要求：用 Markdown，必须包含：\n1) 推理过程：为何选择这些指标/阈值/过滤条件（分点）\n2) 结论：可执行的规则清单（入场/出场/过滤），并给出参数建议（含默认值/范围）\n3) 边界与风险：至少 3 条（例如：震荡/跳空/高波动/消息面等）\n\n{{baseInfo}}',
      },
      risk: {
        title: '风控与执行约束',
        prompt:
          '你是交易风控与执行专家。请根据以下信息，设计仓位管理、止损止盈、最大回撤控制、冷却期/交易频率限制等规则。\n\n输出要求：用 Markdown，必须包含：\n1) 推理过程：为何这些风控能匹配目标/约束（分点）\n2) 结论：硬约束（必须遵守）+ 默认参数（建议值/范围）+ 触发后的动作\n3) 失败模式：至少 3 条（例如：连续亏损、滑点扩大、点差异常等）\n\n{{baseInfo}}',
      },
      code: {
        title: '代码生成 Agent',
        prompt:
          '你是 AntTrader Python 策略代码工程师。请生成一份可运行的 AntTrader Python 策略代码，要求：\n- 必须通过 validate 校验（禁止 import、禁止 dunder、遵守沙箱约束）\n- 使用 on_tick / on_kline 等平台提供的 API（不要自定义网络/文件访问）\n- run 必须且只能接收一个参数：context（参数名必须是 context；不允许 run(ctx)、run(context, data) 等）\n- run(context) 返回一个 dict，至少包含：signal(buy/sell/hold)、symbol、confidence(0~1)、risk_level(low/medium/high)、reason\n- 必须从 context["params"] 读取参数（它是一个 dict，来自调度参数注入）；参数缺失时使用参数表里的 default 值\n- 使用上文的信号设计与风控建议（如果未提供，也请给出合理默认）\n- 直接输出完整代码，并用 ```python 包裹\n- 严格输出：只允许输出 1 个 ```python 代码块```，除此之外不要输出任何解释文字\n- 代码块内必须是纯 Python 代码：禁止出现 Markdown 符号（例如 "- ", "* ", "###"）、禁止出现中文全角标点、禁止出现三引号代码围栏 ```\n\n【必须照抄的入口模板（不要改函数名/参数个数/参数名）】\n```python\ndef run(context):\n    params = context.get("params") or {}\n    symbol = context.get("symbol") or params.get("symbol") or ""\n    # TODO: 在这里实现信号/风控逻辑\n    return {\n        "signal": "hold",\n        "symbol": symbol,\n        "confidence": 0.5,\n        "risk_level": "low",\n        "reason": "",\n    }\n```\n\n{{baseInfo}}\n\n【附：上游分析（若有）】\n请你将市场/信号/风控三个分析结论落到代码中（如果上游结论未提供，也请给出合理默认）。',
      },
    },
    consensus: {
      title: '共识与对话',
      actions: { refresh: '刷新' },
      fields: {
        account: '账号',
        symbol: '品种',
        timeframe: '周期',
      },
      panel: {
        title: '客观评分',
        decision: '决策',
        overallScore: '总体分',
        technicalScore: '技术面分',
      },
      signals: {
        rsi: { value: 'RSI', flag: '信号' },
        macd: { value: 'MACD', signalLine: '信号线', hist: '柱体', flag: '信号', trend: '形态' },
        ma: { trend: '均线趋势' },
      },
    },
    conversation: {
      defaultTitle: '新对话',
    },
    chatBox: {
      emptyDescription: '开始与AI助手对话',
      thinking: '思考中...',
      truncated: '内容过长，已截断',
      expandAll: '展开全部',
      collapse: '收起',
    },
    reports: {
      tradeAnalysis: {
        title: 'AI交易分析报告',
        riskAssessmentPrefix: '风险评估:',
      },
    },
    signalCard: {
      status: {
        pending: '待确认',
        confirmed: '已确认',
        executed: '已执行',
        cancelled: '已取消',
      },
      labels: {
        price: '价格',
        volume: '手数',
        confidence: '信心度',
        stopLoss: '止损',
        takeProfit: '止盈',
        analysisReason: '分析理由',
      },
      actions: {
        confirm: '确认',
        cancel: '取消',
        executeTrade: '执行交易',
      },
      confirmCancel: {
        title: '确定要取消这个信号吗?',
      },
      confirmExecute: {
        title: '确定要执行这个交易信号吗?',
        description: '将立即下单交易',
      },
    },
    assistant: {
      messages: {
        noCodeBlockFound: '未找到代码块（```...```）',
      },
    },
    strategyCard: {
      status: {
        active: '运行中',
        inactive: '已停止',
        paused: '已暂停',
      },
      actionType: {
        buy: '买入',
        sell: '卖出',
        closeLong: '平多',
        closeShort: '平空',
        alert: '提醒',
      },
      labels: {
        triggeredCount: '触发 {{count}} 次',
        lastTriggeredAt: '最后触发: {{time}}',
      },
      sections: {
        conditions: '触发条件',
        actions: '执行动作',
      },
      tooltips: {
        createdAt: '创建时间',
        lastTriggeredAt: '最后触发',
      },
      actions: {
        start: '启动',
        stop: '停止',
      },
      confirmDelete: {
        title: '确定要删除这个策略吗?',
        description: '删除后将无法恢复',
      },
    },
    requireConfig: {
      title: '尚未配置大模型',
      description: '请先到设置页配置 AI 提供商、模型与 API Key，然后再使用策略向导或聊天。',
      actions: {
        goSettings: '去设置',
      },
    },
    riskEval: {
      failed: '风险评估失败',
    },
    workflowRuns: {
      title: 'AI 工作流',
      defaultTitle: 'AI 工作流',
      hints: {
        selectToViewDetail: '选择左侧运行记录查看详情',
      },
      messages: {
        loadListFailed: '加载运行记录失败',
        loadDetailFailed: '加载详情失败',
      },
    },
    client: {
      errors: {
        requestFailed: '请求失败，请重试。',
        insufficientBalance: '模型厂商返回：账户余额不足或欠费。请到厂商控制台充值后再试。',
        rateLimited: '模型厂商限流（请求过于频繁），请稍后重试。',
        unauthorized: '模型厂商返回 401 未授权：请检查 API Key 是否正确、是否有该模型权限。',
        forbidden: '模型厂商返回 403 拒绝访问：请检查 Key 权限、IP 白名单或账号状态。',
        invalidModelId: '模型不可用{{model}}：可能不存在、已下线或不在你的权限内。请重新选择或到厂商控制台复制正确的 model id。',
        contextTooLong: '请求超出模型最大上下文长度，请缩短对话历史/输入或换更大上下文窗口的模型。',
        contentBlocked: '内容被厂商安全策略拦截。请调整提问措辞后重试。',
        regionNotSupported: '当前地区/国家不被该厂商支持。请更换地区或选择其他厂商。',
        providerInternalError: '模型厂商服务暂时不可用（5xx）。请稍后重试，或切换到其他厂商。',
        edgeGatewayTimeout:
          '站点前置网关超时（常见为 Cloudflare 524 等）：请求在到达应用前被中断，长耗时的「生成代码」等步骤更容易触发。请在辩论页代码步骤使用「重新尝试生成代码」，或先返回上一步再进入生成代码；仍失败时需运维调大网关/源站超时。',
        networkUnreachable: '模型网关连接超时或不可达。请检查 Base URL 是否可访问、网络是否通畅，或稍后重试。',
        gatewayTimeoutOrUnreachable: '模型网关连接超时/不可达。请检查 AI 设置中的 Base URL 是否可访问、网络是否通畅，或稍后重试。',
        gatewayUnauthorized401: '模型厂商返回 401 未授权：请检查 API Key 是否正确、是否有模型权限。',
        gatewayForbidden403: '模型厂商返回 403 拒绝访问：请检查 Key 权限、IP 白名单或账号状态。',
        gatewayRateLimited429: '模型厂商请求过于频繁（429），请稍后重试。',
      },
    },
    backtestScoreCard: {
      title: '回测评分卡',
      stateLabel: '状态',
      status: {
        succeeded: '成功',
        running: '运行中',
        pending: '排队中',
        failed: '失败',
        cancelRequested: '取消中',
        canceled: '已取消',
      },
      recommendation: {
        loading: '风险评估计算中，建议先等待完成再上线。',
        recommended: '推荐上线：风险可控，指标整体健康。',
        cautious: '谨慎上线：建议先小资金/手动确认运行一段时间。',
        notRecommended: '不推荐直接上线：风险较高或不可靠，建议优化后再尝试。',
      },
      backendRiskScore: {
        title: '后端风险评分',
        loading: '计算中...',
        unknown: 'unknown',
        reliable: '可靠',
        unreliable: '不可靠',
        reasons: '原因',
        warnings: '警告',
        empty: '暂无（请先保存模板，且回测完成后自动计算）',
      },
      score: {
        empty: '暂无评分（等待回测完成或无 metrics）',
        title: '综合评分（前端启发式）',
      },
      level: {
        excellent: '优秀',
        good: '良好',
        fair: '一般',
        poor: '较差',
      },
      metrics: {
        totalReturn: '总收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普',
        winRate: '胜率',
        totalTrades: '交易次数',
        equityPoints: 'equity 点数',
      },
      chart: {
        title: '资金曲线（equity）',
      },
    },
    systemAI: {
      taglines: {
        openai: 'GPT 系列 · 官方',
        anthropic: 'Claude 系列',
        deepseek: '深度求索 · 高性价比',
        moonshot: 'Kimi · 长上下文',
        qwen: '阿里云 · 中文优化',
        zhipu: '清华系 · 通用',
        openai_compatible: '任意兼容端点',
      },
      pageTitle: 'AI 助手设置',
      pageSubtitle: '配置 AI 大脑 — 选择模型厂商、管理 API 密钥与可用模型，并指定全站兜底使用的「默认主模型」。',
      emptyConfigs: '暂无 AI Provider 配置（系统启动时会自动创建默认 Provider）',
      section1: {
        title: '选择模型厂商',
        subtitle: '卡片直接展示每个厂商的配置与就绪状态，点击选择',
      },
      statusBar: {
        enabled: '已启用',
        disabled: '未启用',
        keyReady: '密钥就绪',
        checking: '连通性检测中…',
        connected: '连接正常',
      },
      status: {
        noProvider: '尚未选择厂商',
        noProviderDesc: '请从下方卡片挑选一个模型厂商开始配置',
        error: '存在异常',
        ready: '运行就绪',
        readyDesc: '已启用并连接正常',
        notEnabled: '连接正常，尚未启用',
        notEnabledDesc: '打开「启用」开关即可投入使用',
        configReady: '配置已就绪',
        configReadyDesc: '添加可用模型后系统将自动完成连通性检测',
        checkUrl: '请检查 Base URL',
        checkUrlDesc: 'API Key 已就绪，但地址似乎无效',
        needKey: '请完成密钥配置',
        needKeyDesc: '填写 API Key 后将自动发现模型列表',
        connectionFailed: '连接异常，请检查上方提示',
      },
      cardState: {
        noKey: '未配置',
        noModel: '待选模型',
        enabled: '已启用',
        readyDisabled: '已就绪 · 未启用',
      },
      cardTags: {
        current: '当前',
        hasKey: '已配密钥',
        noKey: '未配密钥',
        noModels: '未配置可用模型',
        enabledButUnavailable: '启用但不可用',
      },
      fields: {
        autoFetching: '自动拉取中',
        baseUrlCustomHint: '输入 OpenAI 兼容端点，例如 https://model.example.com/v1',
        baseUrlReadonlyHint: '官方地址由系统维护，不可修改',
        baseUrlCustomPlaceholder: '例如: https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: '官方地址（只读）',
        httpWarning: '当前为 HTTP，生产环境建议使用 HTTPS',
        apiKeyHint: '输入后将自动加密保存，无需手动提交',
        apiKeyPastePlaceholder: '粘贴 API Key，将自动预保存',
        enabledHint: '关闭后该厂商不参与系统路由',
        temperatureHint: '越高越发散，越低越稳定',
        timeoutHint: '单次请求最长等待时间',
        maxTokensHint: '单次响应最大 token 数',
        primaryFor: '主要用途（Primary For）',
        primaryForHint: '仅用于服务内部路由：chat / embedding / summarizer / reasoning',
      },
    },
    tabs: {
      debate: '专家讨论',
      settings: '设置',
      agentSettings: '专家设置',
    },
  },
} as const;

export default aiCore;
