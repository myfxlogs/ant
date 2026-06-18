const aiCore = {
  ai: {
    client: {
      errors: {
        requestFailed: 'Request failed. Please try again.',
        insufficientBalance: 'The provider reported an empty balance / overdue payment. Top up the account in the provider console and retry.',
        rateLimited: 'The provider is rate-limiting your requests. Please wait a moment and try again.',
        unauthorized: 'The provider rejected the API key (401). Check the key value and that it has access to the selected model.',
        forbidden: 'The provider refused the request (403). Check key permissions, IP allowlist, and account status.',
        invalidModelId: 'Model unavailable{{model}} – it may be wrong, deprecated, or outside your tier. Pick another from the dropdown or copy the canonical id from the provider console.',
        contextTooLong: 'The request exceeds the model context window. Shorten the conversation/input or pick a model with a larger context.',
        contentBlocked: 'The provider safety filter blocked the response. Rephrase the prompt and try again.',
        regionNotSupported: 'The selected provider is not available in your region/country. Switch to a different provider.',
        providerInternalError: 'The provider returned a server-side error (5xx). Wait a moment or switch to another provider.',
        edgeGatewayTimeout: 'The edge gateway timed out (often HTTP 524 on Cloudflare): the browser never received the app response, which is common for long-running operations. Try again; if the issue persists, raise proxy/origin timeouts with ops.',
        networkUnreachable: 'Gateway timed out or is unreachable. Check the Base URL, network connectivity, or try again later.',
        gatewayTimeoutOrUnreachable: 'Gateway timeout or unreachable.',
        gatewayUnauthorized401: 'Gateway unauthorized (401).',
        gatewayForbidden403: 'Gateway forbidden (403).',
        gatewayRateLimited429: 'Gateway rate limited (429).'
      }
    },
    agentPrompts: {
      style: {
        title: '市场状态/风格推荐',
        prompt: `You are a senior quantitative strategy analyst. Based on the following information, recommend a strategy paradigm: trend / mean reversion / short-term, and explain the reasoning, applicable conditions and inapplicable scenarios.

Output requirements: use Markdown, must include:
1) Reasoning process: how you derive from data/constraints/objectives (bullet points)
2) Conclusion: main recommendation (only one primary paradigm) + alternative + applicable/inapplicable conditions
3) Risk alerts: at least 3

{{baseInfo}}`
      },
      signals: {
        title: '信号与指标设计',
        prompt: `You are a quantitative factor and signal engineer. Without relying on external data (unless the user provides macro event tables), design actionable trading signals.

Requirements: clearly define entry/exit/filter conditions, preferably parameterized, avoid overfitting.

Output requirements: use Markdown, must include:
1) Reasoning process: why choose these indicators/thresholds/filter conditions (bullet points)
2) Conclusion: executable rule list (entry/exit/filter), with parameter suggestions (default/range)
3) Boundaries and risks: at least 3 (e.g.: range-bound/gap/high volatility/news events)

{{baseInfo}}`
      },
      risk: {
        title: '风控与执行约束',
        prompt: `You are a trading risk and execution expert. Based on the following information, design position management, stop-loss/take-profit, max drawdown control, cooldown period/trade frequency limits, etc.

Output requirements: use Markdown, must include:
1) Reasoning process: why these controls match objectives/constraints (bullet points)
2) Conclusion: hard constraints + default parameters (suggested/range) + actions after trigger
3) Failure modes: at least 3 (e.g.: consecutive losses, slippage widening, spread anomalies)

{{baseInfo}}`
      },
      code: {
        title: '代码生成 Agent',
        prompt: `You are an AntTrader Python strategy code engineer. Generate runnable AntTrader Python strategy code that:
- Passes validate checks (no import, no dunder, sandbox constraints)
- Uses platform APIs like on_tick / on_kline (no custom network/file access)
- run() must receive exactly one parameter: context (must be named context; no run(ctx), run(context, data), etc.)
- run(context) returns a dict with at least: signal(buy/sell/hold), symbol, confidence(0~1), risk_level(low/medium/high), reason
- Read parameters from context["params"] (from schedule injection); use defaults if missing
- Use upstream signal design and risk controls (provide reasonable defaults if not provided)
- Output full code wrapped in \`\`\`python
- Strict output: only one \`\`\`python block\`\`\`, no explanation text
- Code block must be pure Python: no Markdown symbols, no Chinese punctuation, no nested code fences

[Mandatory entry template (do not change function name/param count/param name)]
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: implement signal/risk logic here
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

[Note: upstream analysis conclusions – apply to code (provide reasonable defaults if missing)]`
      }
    },
    consensus: {
      title: '共识与对话',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: '账号',
        symbol: '品种',
        timeframe: 'Timeframe'
      },
      panel: {
        title: '客观评分',
        decision: '决策',
        overallScore: '总体分',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: '信号'
        },
        macd: {
          value: 'MACD',
          signalLine: '信号线',
          hist: '柱体',
          flag: '信号',
          trend: 'Pattern'
        },
        ma: {
          trend: 'MA Trend'
        }
      }
    },
    conversation: {
      defaultTitle: 'New Conversation'
    },
    chatBox: {
      emptyDescription: '开始与AI助手对话',
      thinking: '思考中...',
      truncated: '内容过长，已截断',
      expandAll: '展开全部',
      collapse: 'Collapse'
    },
    reports: {
      tradeAnalysis: {
        title: 'AI交易分析报告',
        riskAssessmentPrefix: 'Risk Assessment:'
      }
    },
    signalCard: {
      status: {
        pending: '待确认',
        confirmed: '已确认',
        executed: '已执行',
        cancelled: 'Cancelled'
      },
      labels: {
        price: '价格',
        volume: '手数',
        confidence: '信心度',
        stopLoss: '止损',
        takeProfit: '止盈',
        analysisReason: 'Analysis Reason'
      },
      actions: {
        confirm: '确认',
        cancel: '取消',
        executeTrade: 'Execute Trade'
      },
      confirmCancel: {
        title: 'Are you sure you want to cancel this signal?'
      },
      confirmExecute: {
        title: '确定要执行这个交易信号吗?',
        description: 'Will place the order immediately'
      }
    },
    assistant: {
      messages: {
        noCodeBlockFound: 'No code block found (\`\`\`...\`\`\`)'
      }
    },
    strategyCard: {
      status: {
        active: '运行中',
        inactive: '已停止',
        paused: 'Paused'
      },
      actionType: {
        buy: '买入',
        sell: '卖出',
        closeLong: '平多',
        closeShort: '平空',
        alert: 'Alert'
      },
      labels: {
        triggeredCount: '触发 {{count}} 次',
        lastTriggeredAt: 'Last triggered: {{time}}'
      },
      sections: {
        conditions: '触发条件',
        actions: 'Actions'
      },
      tooltips: {
        createdAt: '创建时间',
        lastTriggeredAt: 'Last triggered'
      },
      actions: {
        start: '启动',
        stop: 'Stop'
      },
      confirmDelete: {
        title: '确定要删除这个策略吗?',
        description: 'Cannot be recovered after deletion'
      }
    },
    requireConfig: {
      title: '尚未配置大模型',
      description: '请先到设置页配置 AI 提供商、模型与 API Key，然后再使用策略向导或聊天。',
      actions: {
        goSettings: 'Go to Settings'
      }
    },
    riskEval: {
      failed: 'Risk evaluation failed'
    },
    workflowRuns: {
      title: 'AI 工作流',
      defaultTitle: 'AI 工作流',
      hints: {
        selectToViewDetail: 'Select a run from the left to view details'
      },
      messages: {
        loadListFailed: '加载运行记录失败',
        loadDetailFailed: 'Failed to load details'
      }
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
        canceled: 'Cancelled'
      },
      recommendation: {
        loading: '风险评估计算中，建议先等待完成再上线。',
        recommended: '推荐上线：风险可控，指标整体健康。',
        cautious: '谨慎上线：建议先小资金/手动确认运行一段时间。',
        notRecommended: 'Not recommended for direct live: high risk or unreliable, optimize before trying.'
      },
      backendRiskScore: {
        title: '策略风险评分',
        loading: '计算中...',
        unknown: '未知',
        reliable: '可靠',
        unreliable: '不可靠',
        reasons: '原因',
        warnings: '警告',
        empty: 'None (save template first, will auto-calculate after backtest completes)'
      },
      score: {
        empty: '暂无评分（等待回测完成或无 metrics）',
        title: 'Overall Score (heuristic)'
      },
      level: {
        excellent: '优秀',
        good: '良好',
        fair: '一般',
        poor: 'Poor'
      },
      metrics: {
        totalReturn: '总收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普',
        winRate: '胜率',
        totalTrades: '交易次数',
        equityPoints: 'Equity points'
      },
      chart: {
        title: 'Equity Curve'
      }
    },
    systemAI: {
      taglines: {
        openai: 'GPT 系列 · 官方',
        anthropic: 'Claude 系列',
        deepseek: '深度求索 · 高性价比',
        moonshot: 'Kimi · 长上下文',
        qwen: '阿里云 · 中文优化',
        zhipu: '清华系 · 通用',
        openai_compatible: 'Any compatible endpoint'
      },
      pageTitle: 'AI 助手设置',
      pageSubtitle: '配置 AI 大脑 — 选择模型厂商、管理 API 密钥与可用模型，并指定全站兜底使用的「默认主模型」。',
      emptyConfigs: '暂无 AI Provider 配置（系统启动时会自动创建默认 Provider）',
      section1: {
        title: '选择模型厂商',
        subtitle: `Cards show each provider's configuration and readiness; click to select`
      },
      statusBar: {
        enabled: '已启用',
        disabled: '未启用',
        keyReady: '密钥就绪',
        checking: '连通性检测中…',
        connected: 'Connected'
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
        connectionFailed: 'Connection error, check prompts above'
      },
      cardState: {
        noKey: '未配置',
        noModel: '待选模型',
        enabled: '已启用',
        readyDisabled: 'Ready · Disabled'
      },
      cardTags: {
        current: '当前',
        hasKey: '已配密钥',
        noKey: '未配密钥',
        noModels: '未配置可用模型',
        enabledButUnavailable: 'Enabled but unavailable'
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
        primaryForHint: 'For internal routing: chat / embedding / summarizer / reasoning'
      },
      messages: {
        loadConfigFailed: '加载配置失败',
        secretSavedAutoDiscover: '密钥已保存，正在自动发现模型...',
        secretAutoSaveFailed: '密钥自动保存失败',
        autoDiscoveredModels: '已自动发现 {{count}} 个模型（仅作选择建议）',
        autoValidatedModels: '已自动验证：发现 {{count}} 个模型',
        configSaved: '配置已保存',
        configSaveFailed: '配置保存失败',
        toggleEnabledFailed: '更新启用状态失败',
        secretDeletedConfigReset: '密钥已删除，厂商配置已恢复默认初始化',
        deleteSecretFailed: '删除密钥失败',
        validationPassedModels: '验证通过：发现 {{count}} 个模型',
        validationFailedNeedApiKey: 'Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.'
      },
      customProvider: {
        deleted: '自定义提供商已删除',
        fillNameFirst: '请先填写名称',
        nameHint: '用于识别此提供商的唯一名称',
        nameLabel: '提供商名称',
        namePlaceholder: '我的自定义提供商',
        nameRequired: 'Provider name is required'
      }
    },
    tabs: {
      settings: '设置',
      agentSettings: '专家设置',
      gate: 'AI Gate'
    },
    gate: {
      title: 'AI Gate 进度面板',
      pipelineDesc: '6 级 Gate 管道: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation',
      labels: {
        compliance: '合规检查',
        lookahead: '前视偏差',
        walkforward: 'Walk-Forward',
        deflated_sharpe: 'Deflated Sharpe',
        paper: '模拟交易',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'DSL 表达式非空验证',
        lookahead: '扫描未来函数引用 (close[t+N], ref 负偏移)',
        walkforward: 'Purged Walk-Forward 交叉验证',
        deflated_sharpe: 'Lopez de Prado 紧缩夏普比率',
        paper: '≥14 天模拟交易验证',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: '评估中...'
      },
      skipped: '已跳过',
      noData: '无数据',
      pass: '通过',
      fail: '失败',
      unknown: '未知',
      selectRun: '选择回测运行...',
      strategyParams: '策略参数',
      dslExpression: 'DSL 表达式',
      dailyReturns: '日收益率 (逗号或换行分隔)',
      numAttempts: '策略尝试次数',
      paperMetrics: '模拟交易指标',
      paperDays: '模拟天数',
      paperNetPnL: '模拟 Net P&L',
      paperNetReturn: '模拟净收益',
      paperTradeCount: '模拟交易数',
      backtestNetReturn: '回测净收益',
      backtestGrossReturn: '回测毛收益',
      runPipeline: '运行 Gate 管道',
      retry: '重试',
      gateProgress: 'Gate 评估进度',
      pipelineResult: '管道结果',
      allPassed: '所有 6 个 Gate 通过，策略可进入 PromoteToLive 评估',
      failed: '未通过: {{gate}}',
      details: '详细结果',
      evaluating: '评估中...',
      runHint: 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.'
    },
    gateway: {
      title: 'AI 网关',
      useGateway: 'AI 网关',
      useGatewayDesc: '扣钱包余额 · 按 Token 计费',
      useOwnKey: '我的 API Key',
      useOwnKeyDesc: '直付厂商 · 自行管理',
      useOwnKeyHint: '使用你自己的 API Key，直接向所选厂商付费。在下方选择厂商卡片进行配置。',
      selectModel: '选择模型',
      modelPlaceholder: '选择 AI 模型',
      noModels: '暂无可用模型',
      balance: '钱包余额',
      monthlyTokens: '本月 Token',
      monthlyCost: '本月费用',
      usageByFeature: '按功能用量',
    }
  }
} as const;

export default aiCore;
