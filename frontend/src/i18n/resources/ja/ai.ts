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
        gatewayRateLimited429: 'Gateway rate limited (429).'
      }
    },
    agentPrompts: {
      style: {
        title: '相場環境・スタイル推奨',
        prompt: `You are a senior quantitative strategy analyst. Based on the following information, recommend a strategy paradigm: trend / mean reversion / short-term, and explain the reasoning, applicable conditions and inapplicable scenarios.

Output requirements: use Markdown, must include:
1) Reasoning process: how you derive from data/constraints/objectives (bullet points)
2) Conclusion: main recommendation (only one primary paradigm) + alternative + applicable/inapplicable conditions
3) Risk alerts: at least 3

{{baseInfo}}`
      },
      signals: {
        title: 'シグナルとインジケーター設計',
        prompt: `You are a quantitative factor and signal engineer. Without relying on external data (unless the user provides macro event tables), design actionable trading signals.

Requirements: clearly define entry/exit/filter conditions, preferably parameterized, avoid overfitting.

Output requirements: use Markdown, must include:
1) Reasoning process: why choose these indicators/thresholds/filter conditions (bullet points)
2) Conclusion: executable rule list (entry/exit/filter), with parameter suggestions (default/range)
3) Boundaries and risks: at least 3 (e.g.: range-bound/gap/high volatility/news events)

{{baseInfo}}`
      },
      risk: {
        title: 'リスク管理と執行制約',
        prompt: `You are a trading risk and execution expert. Based on the following information, design position management, stop-loss/take-profit, max drawdown control, cooldown period/trade frequency limits, etc.

Output requirements: use Markdown, must include:
1) Reasoning process: why these controls match objectives/constraints (bullet points)
2) Conclusion: hard constraints + default parameters (suggested/range) + actions after trigger
3) Failure modes: at least 3 (e.g.: consecutive losses, slippage widening, spread anomalies)

{{baseInfo}}`
      },
      code: {
        title: 'コード生成エージェント',
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
      title: 'コンセンサスと議論',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: '口座',
        symbol: '銘柄',
        timeframe: 'Timeframe'
      },
      panel: {
        title: '目標スコア',
        decision: '判定',
        overallScore: '総合',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'シグナル'
        },
        macd: {
          value: 'MACD',
          signalLine: 'シグナル線',
          hist: 'ヒストグラム',
          flag: 'シグナル',
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
        title: 'AI取引分析レポート',
        riskAssessmentPrefix: 'Risk Assessment:'
      }
    },
    signalCard: {
      status: {
        pending: '保留中',
        confirmed: '確認済み',
        executed: '執行済み',
        cancelled: 'Cancelled'
      },
      labels: {
        price: '価格',
        volume: 'ロット',
        confidence: '確信度',
        stopLoss: 'ストップロス',
        takeProfit: 'テイクプロフィット',
        analysisReason: 'Analysis Reason'
      },
      actions: {
        confirm: '確認',
        cancel: 'キャンセル',
        executeTrade: 'Execute Trade'
      },
      confirmCancel: {
        title: 'Are you sure you want to cancel this signal?'
      },
      confirmExecute: {
        title: 'この取引シグナルを執行してもよろしいですか？',
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
        active: '稼働中',
        inactive: '停止中',
        paused: 'Paused'
      },
      actionType: {
        buy: '買い',
        sell: '売り',
        closeLong: '買い決済',
        closeShort: '売り決済',
        alert: 'Alert'
      },
      labels: {
        triggeredCount: '{{count}}回トリガー',
        lastTriggeredAt: 'Last triggered: {{time}}'
      },
      sections: {
        conditions: 'トリガー条件',
        actions: 'Actions'
      },
      tooltips: {
        createdAt: '作成日時',
        lastTriggeredAt: 'Last triggered'
      },
      actions: {
        start: '開始',
        stop: 'Stop'
      },
      confirmDelete: {
        title: 'このストラテジーを削除してもよろしいですか？',
        description: 'Cannot be recovered after deletion'
      }
    },
    requireConfig: {
      title: 'LLMがまだ設定されていません',
      description: '先に設定画面でAIプロバイダー、モデル、APIキーを設定してください。その後、ストラテジーウィザードまたはチャットをご利用いただけます。',
      actions: {
        goSettings: 'Go to Settings'
      }
    },
    riskEval: {
      failed: 'Risk evaluation failed'
    },
    workflowRuns: {
      title: 'AIワークフロー',
      defaultTitle: 'AIワークフロー',
      hints: {
        selectToViewDetail: 'Select a run from the left to view details'
      },
      messages: {
        loadListFailed: '実行一覧の読み込みに失敗しました',
        loadDetailFailed: 'Failed to load details'
      }
    },
    backtestScoreCard: {
      title: 'バックテストスコアカード',
      stateLabel: '状態',
      status: {
        succeeded: '成功',
        running: '実行中',
        pending: 'キュー待ち',
        failed: '失敗',
        cancelRequested: 'キャンセル中',
        canceled: 'Cancelled'
      },
      recommendation: {
        loading: 'リスク評価中です。完了するまでお待ちください。',
        recommended: '本番推奨：リスク管理可能、指標良好。',
        cautious: '本番注意：まずは少額または手動確認でしばらく運用してください。',
        notRecommended: 'Not recommended for direct live: high risk or unreliable, optimize before trying.'
      },
      backendRiskScore: {
        title: 'ストラテジーリスクスコア',
        loading: '計算中...',
        unknown: '不明',
        reliable: '信頼できる',
        unreliable: '信頼できない',
        reasons: '理由',
        warnings: '警告',
        empty: 'None (save template first, will auto-calculate after backtest completes)'
      },
      score: {
        empty: 'スコア未評価（バックテスト待ちまたは指標なし）',
        title: 'Overall Score (heuristic)'
      },
      level: {
        excellent: '優秀',
        good: '良好',
        fair: '普通',
        poor: 'Poor'
      },
      metrics: {
        totalReturn: '総収益率',
        annualReturn: '年換算収益率',
        maxDrawdown: '最大ドローダウン',
        sharpe: 'シャープレシオ',
        winRate: '勝率',
        totalTrades: '総取引数',
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
      title: 'AIゲート進捗',
      pipelineDesc: '6段階ゲートパイプライン：コンプライアンス → 先読みバイアス → ウォークフォワード → 収縮シャープ → ペーパー → 相関',
      labels: {
        compliance: 'コンプライアンス',
        lookahead: '先読みバイアス',
        walkforward: 'ウォークフォワード',
        deflated_sharpe: '収縮シャープレシオ',
        paper: 'ペーパートレーディング',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'DSL式の空チェック',
        lookahead: '未来関数参照のスキャン（close[t+N]、ref負のオフセット）',
        walkforward: 'パージド・ウォークフォワード交差検証',
        deflated_sharpe: 'Lopez de Prado 収縮シャープレシオ',
        paper: '14日以上のペーパートレーディング検証',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: '评估中...'
      },
      skipped: 'SKIPPED',
      noData: 'データがありません',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: '不明',
      selectRun: 'バックテスト実行を選択...',
      strategyParams: 'ストラテジーパラメーター',
      dslExpression: 'DSL式',
      dailyReturns: '日次リターン（カンマまたは改行区切り）',
      numAttempts: 'ストラテジー試行回数',
      paperMetrics: 'ペーパートレーディング指標',
      paperDays: 'ペーパー日数',
      paperNetPnL: 'ペーパー純損益',
      paperNetReturn: 'ペーパー純収益率',
      paperTradeCount: 'ペーパー取引回数',
      backtestNetReturn: 'バックテスト純収益率',
      backtestGrossReturn: 'バックテスト総収益率',
      runPipeline: 'ゲートパイプライン実行',
      retry: '再試行',
      gateProgress: 'ゲート評価進捗',
      pipelineResult: 'パイプライン結果',
      allPassed: '全6ゲートを通過 — ストラテジーは本番昇格評価の対象です',
      failed: '不合格：{{gate}}',
      details: '詳細',
      evaluating: '评估中...',
      runHint: 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.'
    },
    gateway: {
      title: 'AI ゲートウェイ',
      useGateway: 'AI ゲートウェイ',
      useGatewayDesc: 'ウォレット課金 · トークン単位',
      useOwnKey: '自分の API Key',
      useOwnKeyDesc: '直接課金 · 自己管理',
      useOwnKeyHint: '自分の API Key を使用してプロバイダーに直接支払います。下のプロバイダーカードを選択して設定してください。',
      selectModel: 'モデル選択',
      modelPlaceholder: 'AI モデルを選択',
      noModels: '利用可能なモデルがありません',
      balance: 'ウォレット残高',
      monthlyTokens: '今月のトークン',
      monthlyCost: '今月の費用',
      usageByFeature: '機能別使用量',
    }
  }
} as const;

export default aiCore;
