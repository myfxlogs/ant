const aiCore = {
  ai: {
    client: {
      errors: {
        requestFailed: 'リクエストに失敗しました。もう一度お試しください。',
        insufficientBalance: 'プロバイダーからの返答：残高不足または支払い延滞。プロバイダーコンソールで残高・請求を確認し、再度お試しください。',
        rateLimited: 'プロバイダーによるレート制限（リクエストが頻繁すぎます）。しばらくしてから再度お試しください。',
        unauthorized: 'プロバイダーから 401 未認証：API キー が正しいか、モデル権限があるかを確認してください。',
        forbidden: 'プロバイダーから 403 アクセス拒否：Key の権限、IP ホワイトリスト、アカウント状態を確認してください。',
        invalidModelId: 'モデル利用不可{{model}}：存在しない、廃止された、または権限外の可能性があります。ドロップダウンから選ぶか、プロバイダーコンソールから正しい モデル ID をコピーしてください。',
        contextTooLong: 'リクエストがモデルの最大コンテキスト長を超えています。会話履歴・入力を短縮するか、より大きなコンテキストウィンドウのモデルを選んでください。',
        contentBlocked: '内容がプロバイダーのセーフティポリシーによりブロックされました。質問の言い回しを調整して再度お試しください。',
        regionNotSupported: '現在の地域・国はこのプロバイダーではサポートされていません。地域を変更するか、別のプロバイダーを選択してください。',
        providerInternalError: 'プロバイダーサービスが一時的に利用不可（5xx）。しばらくしてから再度お試しください、または別のプロバイダーに切り替えてください。',
        edgeGatewayTimeout: 'フロントの前段ゲートウェイがタイムアウトしました（Cloudflare の HTTP 524 など）。長時間の「コード生成」で起きやすいです。ディベートのコード画面で「コード生成を再試行」するか、一つ戻ってから再度進めてください。改善しない場合はプロキシ／オリジンのタイムアウト延長が必要です。',
        networkUnreachable: 'モデルゲートウェイへの接続がタイムアウトまたは到達不可。AI 設定の 基底 URL がアクセス可能か、ネットワークが正常かを確認し、またはしばらくしてから再度お試しください。',
        gatewayTimeoutOrUnreachable: 'ゲートウェイ接続タイムアウト/到達不可。',
        gatewayUnauthorized401: 'ゲートウェイ未認証（401）。',
        gatewayForbidden403: 'ゲートウェイアクセス拒否（403）。',
        gatewayRateLimited429: 'ゲートウェイレート制限（429）。'
      }
    },
    chatBox: {
      emptyDescription: 'AI アシスタントと会話を開始',
      thinking: '考え中...',
      truncated: '内容が長すぎるため切り詰めました',
      expandAll: 'すべて展開',
      collapse: '折りたたむ'
    },
    systemAI: {
      taglines: {
        openai: 'GPT シリーズ · 公式',
        anthropic: 'Claude シリーズ',
        deepseek: 'DeepSeek · 高コストパフォーマンス',
        moonshot: 'Kimi · 長文コンテキスト',
        qwen: 'Alibaba Cloud · 中国語最適化',
        zhipu: '清華系 · 汎用',
        openai_compatible: '任意の互換エンドポイント'
      },
      pageTitle: 'AI アシスタント設定',
      pageSubtitle: 'AI の中枢を設定します — モデルプロバイダー、API キー、利用可能モデルを管理し、サイト全体の既定主モデルを指定します。',
      emptyConfigs: 'AI プロバイダー 設定がありません（システム起動時に既定 プロバイダー が自動作成されます）',
      section1: {
        title: 'モデルプロバイダーを選択',
        subtitle: '各プロバイダーの設定と準備状態をカードで表示します。クリックして選択してください'
      },
      statusBar: {
        enabled: '有効',
        disabled: '無効',
        keyReady: 'キー準備完了',
        checking: '接続確認中…',
        connected: '接続正常'
      },
      status: {
        noprovider: 'プロバイダー未選択',
        noProviderDesc: '下のカードからモデルプロバイダーを選んで設定を開始してください',
        error: '異常があります',
        ready: '利用準備完了',
        readyDesc: '有効化済みで接続正常です',
        notEnabled: '接続正常、まだ無効です',
        notEnabledDesc: '「有効」スイッチをオンにすると使用できます',
        configReady: '設定準備完了',
        configReadyDesc: '利用可能モデルを追加すると接続確認が自動実行されます',
        checkUrl: '基底 URL を確認してください',
        checkUrlDesc: 'API キーは準備済みですが、アドレスが無効の可能性があります',
        needKey: 'キー設定を完了してください',
        needKeyDesc: 'API キーを入力するとモデル一覧を自動検出します',
        connectionFailed: '接続異常です。上の提示を確認してください',
        noProvider: 'No provider selected yet'
      },
      cardState: {
        noKey: '未設定',
        noModel: 'モデル選択待ち',
        enabled: '有効',
        readyDisabled: '準備完了 · 無効'
      },
      cardTags: {
        current: '現在',
        hasKey: 'キー設定済み',
        noKey: 'キー未設定',
        noModels: '利用可能モデル未設定',
        enabledButUnavailable: '有効だが利用不可'
      },
      fields: {
        autoFetching: '自動取得中',
        baseUrlCustomHint: 'OpenAI 互換エンドポイントを入力してください。例：https://model.example.com/v1',
        baseUrlReadonlyHint: '公式アドレスはシステム管理のため変更できません',
        baseUrlCustomPlaceholder: '例：https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: '公式アドレス（読み取り専用）',
        httpWarning: '現在 HTTP です。本番環境では HTTPS を推奨します',
        apiKeyHint: '入力後は自動で暗号化保存されます。手動送信は不要です',
        apiKeyPastePlaceholder: 'API キーを貼り付けると自動で仮保存されます',
        enabledHint: '無効にするとこのプロバイダーはシステムルーティングに参加しません',
        temperatureHint: '高いほど発散的、低いほど安定します',
        timeoutHint: '1 回のリクエストの最大待機時間',
        maxTokensHint: '1 回の応答の最大トークン数',
        primaryFor: '主な用途',
        primaryForHint: 'サービス内部ルーティング専用：チャット / 埋め込み / 要約 / 推論'
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
      }
    },
    tabs: {
      settings: '設定',
      agentSettings: 'エージェント設定',
      gate: 'AI Gate'
    },
    agentPrompts: {
      style: {
        title: 'Market condition / style recommendation',
        prompt: 'You are a senior quantitative strategy analyst. Based on the following information, recommend a strategy paradigm: trend / mean reversion / short-term, and explain the reasoning, applicable conditions and inapplicable scenarios.

Output requirements: use Markdown, must include:
1) Reasoning process: how you derive from data/constraints/objectives (bullet points)
2) Conclusion: main recommendation (only one primary paradigm) + alternative + applicable/inapplicable conditions
3) Risk alerts: at least 3

{{baseInfo}}'
      },
      signals: {
        title: 'Signal and indicator design',
        prompt: 'You are a quantitative factor and signal engineer. Without relying on external data (unless the user provides macro event tables), design actionable trading signals.

Requirements: clearly define entry/exit/filter conditions, preferably parameterized, avoid overfitting.

Output requirements: use Markdown, must include:
1) Reasoning process: why choose these indicators/thresholds/filter conditions (bullet points)
2) Conclusion: executable rule list (entry/exit/filter), with parameter suggestions (default/range)
3) Boundaries and risks: at least 3 (e.g.: range-bound/gap/high volatility/news events)

{{baseInfo}}'
      },
      risk: {
        title: 'Risk control and execution constraints',
        prompt: 'You are a trading risk and execution expert. Based on the following information, design position management, stop-loss/take-profit, max drawdown control, cooldown period/trade frequency limits, etc.

Output requirements: use Markdown, must include:
1) Reasoning process: why these controls match objectives/constraints (bullet points)
2) Conclusion: hard constraints + default parameters (suggested/range) + actions after trigger
3) Failure modes: at least 3 (e.g.: consecutive losses, slippage widening, spread anomalies)

{{baseInfo}}'
      },
      code: {
        title: 'Code generation agent',
        prompt: 'You are an AntTrader Python strategy code engineer. Generate runnable AntTrader Python strategy code that:
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

[Note: upstream analysis conclusions – apply to code (provide reasonable defaults if missing)]'
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
      details: 'Details'
    }
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
    details: 'Details'
  }
} as const;

export default aiCore;
