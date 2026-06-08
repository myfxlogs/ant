const aiCore = {
  ai: {
    client: {
      errors: {
        edgeGatewayTimeout: '網站最外層閘道逾時（常見為 Cloudflare 524）：請求未到應用程式即被中斷，「產生程式碼」等長步驟較易觸發。請在辯論程式碼步驟按「重新嘗試產生程式碼」，或先返回上一步再進入產生程式碼；仍失敗時需請維運調大閘道／來源站逾時。',
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
        networkUnreachable: 'Gateway timed out or is unreachable. Check the Base URL, network connectivity, or try again later.',
        gatewayTimeoutOrUnreachable: 'Gateway timeout or unreachable.',
        gatewayUnauthorized401: 'Gateway unauthorized (401).',
        gatewayForbidden403: 'Gateway forbidden (403).',
        gatewayRateLimited429: 'Gateway rate limited (429).'
      }
    },
    'agent提示詞s': {
      style: {
        title: '市場狀態/風格推薦',
        prompt: '你是資深量化投研分析師。請基於以下資訊，推薦策略範式：趨勢/均值回歸/短線，並說明理由、適用條件與不適用場景。

輸出要求：用 Markdown，必須包含：
1) 推理過程：你如何從資料/約束/目標推導（分點）
2) 結論：主推薦（只能選一個主範式）+ 備選（可選）+ 適用/不適用條件
3) 風險提示：至少 3 條

{{baseInfo}}'
      },
      signals: {
        title: '訊號與指標設計',
        prompt: '你是量化因子與訊號工程師。請在不依賴外部資料（除非使用者提供宏觀事件表）的前提下，設計可實作的交易訊號。

要求：明確入場/出場/過濾條件，盡量參數化，避免過度擬合。

輸出要求：用 Markdown，必須包含：
1) 推理過程：為何選擇這些指標/閾值/過濾條件（分點）
2) 結論：可執行的規則清單（入場/出場/過濾），並給出參數建議（含預設值/範圍）
3) 邊界與風險：至少 3 條（例如：震盪/跳空/高波動/消息面等）

{{baseInfo}}'
      },
      risk: {
        title: '風控與執行約束',
        prompt: '你是交易風控與執行專家。請根據以下資訊，設計倉位管理、止損止盈、最大回撤控制、冷卻期/交易頻率限制等規則。

輸出要求：用 Markdown，必須包含：
1) 推理過程：為何這些風控能匹配目標/約束（分點）
2) 結論：硬約束（必須遵守）+ 預設參數（建議值/範圍）+ 觸發後的動作
3) 失敗模式：至少 3 條（例如：連續虧損、滑點擴大、點差異常等）

{{baseInfo}}'
      },
      code: {
        title: '程式碼生成',
        prompt: '你是 AntTrader Python 策略程式碼工程師。請生成一份可執行的 AntTrader Python 策略程式碼，要求：
- 必須通過 validate 校驗（禁止 import、禁止 dunder、遵守沙箱約束）
- 使用 on_tick / on_kline 等平台提供的 API（不要自訂網路/檔案存取）
- run 必須且只能接收一個參數：context（參數名必須是 context；不允許 run(ctx)、run(context, data) 等）
- run(context) 回傳一個 dict，至少包含：signal(buy/sell/hold)、symbol、confidence(0~1)、risk_level(low/medium/high)、reason
- 必須從 context["params"] 讀取參數（它是由調度參數注入的 dict）；參數缺失時使用參數表中的 default 值
- 使用上文的訊號設計與風控建議（若未提供，也請給出合理預設）
- 直接輸出完整程式碼，並用 \`\`\`python 包裹
- 嚴格輸出：只允許輸出 1 個 \`\`\`python 程式碼塊\`\`\`，除此之外不要輸出任何解釋文字
- 程式碼塊內必須是純 Python 程式碼：禁止出現 Markdown 符號（例如 "- ", "* ", "###"）、禁止出現中文全形標點、禁止出現三個反引號圍欄 \`\`\`

【必須照抄的入口模板（不要改函式名/參數個數/參數名）】
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: 在這裡實作訊號/風控邏輯
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

【附：上游分析（若有）】
請你將市場/訊號/風控三個分析結論落到程式碼中（若上游結論未提供，也請給出合理預設）。'
      }
    },
    systemAI: {
      taglines: {
        openai: 'GPT 系列 · 官方',
        anthropic: 'Claude 系列',
        deepseek: 'DeepSeek · 高性價比',
        moonshot: 'Kimi · 長上下文',
        qwen: '阿里雲 · 中文最佳化',
        zhipu: '清華系 · 通用',
        openai_compatible: '任意相容端點'
      },
      pageTitle: 'AI 助手設定',
      pageSubtitle: '設定 AI 大腦 — 選擇模型廠商、管理 API 金鑰與可用模型，並指定全站兜底使用的「預設主模型」。',
      emptyConfigs: '暫無 AI 廠商 設定（系統啟動時會自動建立預設 廠商）',
      section1: {
        title: '選擇模型廠商',
        subtitle: '卡片直接展示每個廠商的設定與就緒狀態，點擊即可選擇'
      },
      statusBar: {
        enabled: '已啟用',
        disabled: '未啟用',
        keyReady: '金鑰就緒',
        checking: '連線檢測中…',
        connected: '連線正常'
      },
      status: {
        noprovider: '尚未選擇廠商',
        noProviderDesc: '請從下方卡片挑選一個模型廠商開始設定',
        error: '存在異常',
        ready: '運行就緒',
        readyDesc: '已啟用並連線正常',
        notEnabled: '連線正常，尚未啟用',
        notEnabledDesc: '打開「啟用」開關即可投入使用',
        configReady: '設定已就緒',
        configReadyDesc: '新增可用模型後系統將自動完成連線檢測',
        checkUrl: '請檢查 基礎網址',
        checkUrlDesc: 'API 金鑰已就緒，但地址似乎無效',
        needKey: '請完成金鑰設定',
        needKeyDesc: '填寫 API 金鑰後將自動發現模型清單',
        connectionFailed: '連線異常，請檢查上方提示',
        noProvider: 'No provider selected yet'
      },
      cardState: {
        noKey: '未設定',
        noModel: '待選模型',
        enabled: '已啟用',
        readyDisabled: '已就緒 · 未啟用'
      },
      cardTags: {
        current: '目前',
        hasKey: '已配金鑰',
        noKey: '未配金鑰',
        noModels: '未設定可用模型',
        enabledButUnavailable: '啟用但不可用'
      },
      fields: {
        autoFetching: '自動拉取中',
        baseUrlCustomHint: '輸入 OpenAI 相容端點，例如 https://model.example.com/v1',
        baseUrlReadonlyHint: '官方地址由系統維護，不可修改',
        baseUrlCustomPlaceholder: '例如: https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: '官方地址（唯讀）',
        httpWarning: '目前為 HTTP，生產環境建議使用 HTTPS',
        apiKeyHint: '輸入後將自動加密儲存，無需手動提交',
        apiKeyPastePlaceholder: '貼上 API 金鑰，將自動預儲存',
        enabledHint: '關閉後該廠商不參與系統路由',
        temperatureHint: '越高越發散，越低越穩定',
        timeoutHint: '單次請求最長等待時間',
        maxTokensHint: '單次回應最大權杖數',
        primaryFor: '主要用途',
        primaryForHint: '僅用於服務內部路由：對話 / 向量嵌入 / 摘要 / 推理'
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
      agentSettings: '專家設定',
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
    title: 'AI Gate 進度面板',
    pipelineDesc: '6 級 Gate 管道: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation',
    labels: {
      compliance: '合規檢查',
      lookahead: '前視偏差',
      walkforward: 'Walk-Forward',
      deflated_sharpe: 'Deflated Sharpe',
      paper: '模擬交易',
      correlation: '相關性'
    },
    descriptions: {
      compliance: 'DSL 表達式非空驗證',
      lookahead: '掃描未來函數引用',
      walkforward: 'Purged Walk-Forward 交叉驗證',
      deflated_sharpe: 'Lopez de Prado 緊縮夏普比率',
      paper: '≥14 天模擬交易驗證',
      correlation: '與現有策略信號相關性檢查'
    },
    status: {
      evaluating: '評估中...'
    },
    strategyParams: '策略參數',
    dslExpression: 'DSL 表達式',
    dailyReturns: '日收益率 (逗號或換行分隔)',
    numAttempts: '策略嘗試次數',
    paperMetrics: '模擬交易指標',
    paperDays: '模擬天數',
    paperNetPnL: '模擬 Net P&L',
    paperNetReturn: '模擬淨收益',
    paperTradeCount: '模擬交易數',
    backtestNetReturn: '回測淨收益',
    backtestGrossReturn: '回測毛收益',
    runPipeline: '運行 Gate 管道',
    retry: '重試',
    gateProgress: 'Gate 評估進度',
    pipelineResult: '管道結果',
    allPassed: '所有 6 個 Gate 通過，策略可進入 PromoteToLive 評估',
    failed: '未通過: {{gate}}',
    details: '詳細結果'
  }
} as const;

export default aiCore;
