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
        title: 'Market condition / style recommendation',
        prompt: `You are a senior quantitative strategy analyst. Based on the following information, recommend a strategy paradigm: trend / mean reversion / short-term, and explain the reasoning, applicable conditions and inapplicable scenarios.

Output requirements: use Markdown, must include:
1) Reasoning process: how you derive from data/constraints/objectives (bullet points)
2) Conclusion: main recommendation (only one primary paradigm) + alternative + applicable/inapplicable conditions
3) Risk alerts: at least 3

{{baseInfo}}`
      },
      signals: {
        title: 'Signal and indicator design',
        prompt: `You are a quantitative factor and signal engineer. Without relying on external data (unless the user provides macro event tables), design actionable trading signals.

Requirements: clearly define entry/exit/filter conditions, preferably parameterized, avoid overfitting.

Output requirements: use Markdown, must include:
1) Reasoning process: why choose these indicators/thresholds/filter conditions (bullet points)
2) Conclusion: executable rule list (entry/exit/filter), with parameter suggestions (default/range)
3) Boundaries and risks: at least 3 (e.g.: range-bound/gap/high volatility/news events)

{{baseInfo}}`
      },
      risk: {
        title: 'Risk control and execution constraints',
        prompt: `You are a trading risk and execution expert. Based on the following information, design position management, stop-loss/take-profit, max drawdown control, cooldown period/trade frequency limits, etc.

Output requirements: use Markdown, must include:
1) Reasoning process: why these controls match objectives/constraints (bullet points)
2) Conclusion: hard constraints + default parameters (suggested/range) + actions after trigger
3) Failure modes: at least 3 (e.g.: consecutive losses, slippage widening, spread anomalies)

{{baseInfo}}`
      },
      code: {
        title: 'Code generation agent',
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
      title: 'Consensus & Discussion',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: 'Account',
        symbol: 'Symbol',
        timeframe: 'Timeframe'
      },
      panel: {
        title: 'Objective Score',
        decision: 'Decision',
        overallScore: 'Overall',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'Signal'
        },
        macd: {
          value: 'MACD',
          signalLine: 'Signal Line',
          hist: 'Histogram',
          flag: 'Signal',
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
      emptyDescription: 'Start a conversation with the AI assistant',
      thinking: 'Thinking...',
      truncated: 'Content too long, truncated',
      expandAll: 'Expand all',
      collapse: 'Collapse'
    },
    reports: {
      tradeAnalysis: {
        title: 'AI Trade Analysis Report',
        riskAssessmentPrefix: 'Risk Assessment:'
      }
    },
    signalCard: {
      status: {
        pending: 'Pending',
        confirmed: 'Confirmed',
        executed: 'Executed',
        cancelled: 'Cancelled'
      },
      labels: {
        price: 'Price',
        volume: 'Lots',
        confidence: 'Confidence',
        stopLoss: 'Stop Loss',
        takeProfit: 'Take Profit',
        analysisReason: 'Analysis Reason'
      },
      actions: {
        confirm: 'Confirm',
        cancel: 'Cancel',
        executeTrade: 'Execute Trade'
      },
      confirmCancel: {
        title: 'Are you sure you want to cancel this signal?'
      },
      confirmExecute: {
        title: 'Are you sure you want to execute this trade signal?',
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
        active: 'Active',
        inactive: 'Inactive',
        paused: 'Paused'
      },
      actionType: {
        buy: 'Buy',
        sell: 'Sell',
        closeLong: 'Close Long',
        closeShort: 'Close Short',
        alert: 'Alert'
      },
      labels: {
        triggeredCount: 'Triggered {{count}} times',
        lastTriggeredAt: 'Last triggered: {{time}}'
      },
      sections: {
        conditions: 'Trigger Conditions',
        actions: 'Actions'
      },
      tooltips: {
        createdAt: 'Created at',
        lastTriggeredAt: 'Last triggered'
      },
      actions: {
        start: 'Start',
        stop: 'Stop'
      },
      confirmDelete: {
        title: 'Are you sure you want to delete this strategy?',
        description: 'Cannot be recovered after deletion'
      }
    },
    requireConfig: {
      title: 'No LLM configured yet',
      description: 'Please go to Settings first to configure the AI provider, model, and API key, then use the strategy wizard or chat.',
      actions: {
        goSettings: 'Go to Settings'
      }
    },
    riskEval: {
      failed: 'Risk evaluation failed'
    },
    workflowRuns: {
      title: 'AI Workflow',
      defaultTitle: 'AI Workflow',
      hints: {
        selectToViewDetail: 'Select a run from the left to view details'
      },
      messages: {
        loadListFailed: 'Failed to load run list',
        loadDetailFailed: 'Failed to load details'
      }
    },
    backtestScoreCard: {
      title: 'Backtest Scorecard',
      stateLabel: 'State',
      status: {
        succeeded: 'Success',
        running: 'Running',
        pending: 'Queued',
        failed: 'Failed',
        cancelRequested: 'Cancelling',
        canceled: 'Cancelled'
      },
      recommendation: {
        loading: 'Risk assessment in progress, please wait for completion before going live.',
        recommended: 'Recommended for live: risk controllable, metrics healthy.',
        cautious: 'Cautious for live: try small capital / manual confirmation for a while first.',
        notRecommended: 'Not recommended for direct live: high risk or unreliable, optimize before trying.'
      },
      backendRiskScore: {
        title: 'Backend Risk Score',
        loading: 'Calculating...',
        unknown: 'unknown',
        reliable: 'Reliable',
        unreliable: 'Unreliable',
        reasons: 'Reasons',
        warnings: 'Warnings',
        empty: 'None (save template first, will auto-calculate after backtest completes)'
      },
      score: {
        empty: 'No score yet (wait for backtest or no metrics)',
        title: 'Overall Score (heuristic)'
      },
      level: {
        excellent: 'Excellent',
        good: 'Good',
        fair: 'Fair',
        poor: 'Poor'
      },
      metrics: {
        totalReturn: 'Total Return',
        annualReturn: 'Annual Return',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe',
        winRate: 'Win Rate',
        totalTrades: 'Total Trades',
        equityPoints: 'Equity points'
      },
      chart: {
        title: 'Equity Curve'
      }
    },
    systemAI: {
      taglines: {
        openai: 'GPT series · Official',
        anthropic: 'Claude series',
        deepseek: 'DeepSeek · High cost-performance',
        moonshot: 'Kimi · Long context',
        qwen: 'Alibaba Cloud · Chinese optimized',
        zhipu: 'Tsinghua · General',
        openai_compatible: 'Any compatible endpoint'
      },
      pageTitle: 'AI Assistant Settings',
      pageSubtitle: 'Configure the AI brain – select providers, manage API keys and available models, and set the default primary model for the whole site.',
      emptyConfigs: 'No AI Provider configured (system will auto-create default provider on startup)',
      section1: {
        title: 'Select Model Provider',
        subtitle: `Cards show each provider's configuration and readiness; click to select`
      },
      statusBar: {
        enabled: 'Enabled',
        disabled: 'Disabled',
        keyReady: 'Key ready',
        checking: 'Checking connectivity…',
        connected: 'Connected'
      },
      status: {
        noProvider: 'No provider selected yet',
        noProviderDesc: 'Pick a model provider from the cards below to start configuration',
        error: 'Error exists',
        ready: 'Ready',
        readyDesc: 'Enabled and connected',
        notEnabled: 'Connected, not enabled',
        notEnabledDesc: 'Toggle "Enabled" to activate',
        configReady: 'Config ready',
        configReadyDesc: 'Add available models to auto-check connectivity',
        checkUrl: 'Check Base URL',
        checkUrlDesc: 'API Key ready, but address seems invalid',
        needKey: 'Complete key configuration',
        needKeyDesc: 'Fill API Key to auto-discover model list',
        connectionFailed: 'Connection error, check prompts above'
      },
      cardState: {
        noKey: 'Not configured',
        noModel: 'Select model',
        enabled: 'Enabled',
        readyDisabled: 'Ready · Disabled'
      },
      cardTags: {
        current: 'Current',
        hasKey: 'Key configured',
        noKey: 'No key',
        noModels: 'No models configured',
        enabledButUnavailable: 'Enabled but unavailable'
      },
      fields: {
        autoFetching: 'Auto fetching',
        baseUrlCustomHint: 'Enter OpenAI-compatible endpoint, e.g. https://model.example.com/v1',
        baseUrlReadonlyHint: 'Official address maintained by system, read-only',
        baseUrlCustomPlaceholder: 'e.g. https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: 'Official address (read-only)',
        httpWarning: 'Currently HTTP, HTTPS recommended for production',
        apiKeyHint: 'Will be auto-encrypted on save, no manual submission needed',
        apiKeyPastePlaceholder: 'Paste API Key, will auto-pre-save',
        enabledHint: 'Disabled providers will not be routed',
        temperatureHint: 'Higher = more creative, lower = more stable',
        timeoutHint: 'Max wait time per request',
        maxTokensHint: 'Max tokens per response',
        primaryFor: 'Primary For',
        primaryForHint: 'For internal routing: chat / embedding / summarizer / reasoning'
      },
      messages: {
        loadConfigFailed: 'Failed to load configs',
        secretSavedAutoDiscover: 'Secret saved, auto-discovering models...',
        secretAutoSaveFailed: 'Secret auto-save failed',
        autoDiscoveredModels: 'Auto-discovered {{count}} model(s) (for suggestion only)',
        autoValidatedModels: 'Auto-validated: {{count}} model(s) found',
        configSaved: 'Config saved',
        configSaveFailed: 'Config save failed',
        toggleEnabledFailed: 'Toggle enabled status failed',
        secretDeletedConfigReset: 'Secret deleted, provider config reset to defaults',
        deleteSecretFailed: 'Delete secret failed',
        validationPassedModels: 'Validation passed: {{count}} model(s) found',
        validationFailedNeedApiKey: 'Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.'
      },
      customProvider: {
        deleted: 'Custom provider deleted',
        fillNameFirst: 'Fill in name first',
        nameHint: 'A unique name to identify this provider',
        nameLabel: 'Provider Name',
        namePlaceholder: 'My Custom Provider',
        nameRequired: 'Provider name is required'
      }
    },
    tabs: {
      settings: 'Settings',
      agentSettings: 'Agent Settings',
      gate: 'AI Gate'
    },
    gate: {
      title: 'AI Gate Progress',
      pipelineDesc: '6-stage Gate pipeline: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation',
      labels: {
        compliance: 'Compliance',
        lookahead: 'Look-Ahead Bias',
        walkforward: 'Walk-Forward',
        deflated_sharpe: 'Deflated Sharpe',
        paper: 'Paper Trading',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'DSL expression non-empty validation',
        lookahead: 'Future function reference scan (close[t+N], ref negative offset)',
        walkforward: 'Purged Walk-Forward cross-validation',
        deflated_sharpe: 'Lopez de Prado Deflated Sharpe Ratio',
        paper: '≥14 days paper trading validation',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: 'Evaluating...'
      },
      skipped: 'SKIPPED',
      noData: 'no data',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: 'unknown',
      selectRun: 'Select backtest run...',
      strategyParams: 'Strategy Parameters',
      dslExpression: 'DSL Expression',
      dailyReturns: 'Daily Returns (comma or newline separated)',
      numAttempts: 'Strategy Attempts',
      paperMetrics: 'Paper Trading Metrics',
      paperDays: 'Paper Days',
      paperNetPnL: 'Paper Net P&L',
      paperNetReturn: 'Paper Net Return',
      paperTradeCount: 'Paper Trade Count',
      backtestNetReturn: 'Backtest Net Return',
      backtestGrossReturn: 'Backtest Gross Return',
      runPipeline: 'Run Gate Pipeline',
      retry: 'Retry',
      gateProgress: 'Gate Evaluation Progress',
      pipelineResult: 'Pipeline Result',
      allPassed: 'All 6 gates passed — strategy eligible for PromoteToLive evaluation',
      failed: 'Failed: {{gate}}',
      details: 'Details',
      evaluating: 'Evaluating...',
      runHint: 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.'
    }
  }
} as const;

export default aiCore;
