const aiCore = {
  ai: {
    client: {
      errors: {
        requestFailed: '請求失敗，請重試。',
        insufficientBalance: '供應商回報餘額不足/逾期付款。請在供應商控制台充值後重試。',
        rateLimited: '供應商正在對您的請求進行速率限制。請稍候再試。',
        unauthorized: '供應商拒絕API金鑰（401）。請檢查金鑰值及其對所選模型的存取權限。',
        forbidden: '供應商拒絕請求（403）。請檢查金鑰權限、IP白名單及帳戶狀態。',
        invalidModelId: '模型{{model}}不可用——可能名稱錯誤、已棄用或超出您的權限範圍。請從下拉選單中選擇其他模型，或從供應商控制台複製標準ID。',
        contextTooLong: '請求超過模型上下文視窗大小。請縮短對話/輸入內容，或選擇上下文更大的模型。',
        contentBlocked: '供應商安全過濾器已阻止回應。請重新措辭提示後重試。',
        regionNotSupported: '所選供應商在您的地區/國家不可用。請切換至其他供應商。',
        providerInternalError: '供應商返回伺服器端錯誤（5xx）。請稍候或切換至其他供應商。',
        edgeGatewayTimeout: '網站最外層閘道逾時（常見為 Cloudflare 524）：請求未到應用程式即被中斷，「產生程式碼」等長步驟較易觸發。請在辯論程式碼步驟按「重新嘗試產生程式碼」，或先返回上一步再進入產生程式碼；仍失敗時需請維運調大閘道／來源站逾時。',
        networkUnreachable: '閘道逾時或無法連線。請檢查基礎URL、網路連線，或稍後重試。',
        gatewayTimeoutOrUnreachable: '閘道逾時或無法連線。',
        gatewayUnauthorized401: '閘道未經授權（401）。',
        gatewayForbidden403: '閘道禁止存取（403）。',
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
      title: '共識與討論',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: '帳戶',
        symbol: '品種',
        timeframe: 'Timeframe'
      },
      panel: {
        title: '目標評分',
        decision: '決策',
        overallScore: '總體',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: '訊號'
        },
        macd: {
          value: 'MACD',
          signalLine: '訊號線',
          hist: '柱狀圖',
          flag: '訊號',
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
      emptyDescription: '開始與AI助手對話',
      thinking: '思考中...',
      truncated: '內容過長，已截斷',
      expandAll: '全部展開',
      collapse: 'Collapse'
    },
    reports: {
      tradeAnalysis: {
        title: 'AI交易分析報告',
        riskAssessmentPrefix: 'Risk Assessment:'
      }
    },
    signalCard: {
      status: {
        pending: '待處理',
        confirmed: '已確認',
        executed: '已執行',
        cancelled: 'Cancelled'
      },
      labels: {
        price: '價格',
        volume: '手數',
        confidence: '信心度',
        stopLoss: '止損',
        takeProfit: '止盈',
        analysisReason: 'Analysis Reason'
      },
      actions: {
        confirm: '確認',
        cancel: '取消',
        executeTrade: 'Execute Trade'
      },
      confirmCancel: {
        title: 'Are you sure you want to cancel this signal?'
      },
      confirmExecute: {
        title: '確定要執行此交易訊號嗎？',
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
        active: '啟用中',
        inactive: '未啟用',
        paused: 'Paused'
      },
      actionType: {
        buy: '買入',
        sell: '賣出',
        closeLong: '平多',
        closeShort: '平空',
        alert: 'Alert'
      },
      labels: {
        triggeredCount: '已觸發{{count}}次',
        lastTriggeredAt: 'Last triggered: {{time}}'
      },
      sections: {
        conditions: '觸發條件',
        actions: 'Actions'
      },
      tooltips: {
        createdAt: '建立時間',
        lastTriggeredAt: 'Last triggered'
      },
      actions: {
        start: '啟動',
        stop: 'Stop'
      },
      confirmDelete: {
        title: '確定要刪除此策略嗎？',
        description: 'Cannot be recovered after deletion'
      }
    },
    requireConfig: {
      title: '尚未配置LLM',
      description: '請先前往設定頁面配置AI供應商、模型及API金鑰，然後再使用策略精靈或聊天功能。',
      actions: {
        goSettings: 'Go to Settings'
      }
    },
    riskEval: {
      failed: 'Risk evaluation failed'
    },
    workflowRuns: {
      title: 'AI工作流程',
      defaultTitle: 'AI工作流程',
      hints: {
        selectToViewDetail: 'Select a run from the left to view details'
      },
      messages: {
        loadListFailed: '載入執行列表失敗',
        loadDetailFailed: 'Failed to load details'
      }
    },
    backtestScoreCard: {
      title: '回測評分卡',
      stateLabel: '狀態',
      status: {
        succeeded: '成功',
        running: '執行中',
        pending: '佇列中',
        failed: '失敗',
        cancelRequested: '取消中',
        canceled: 'Cancelled'
      },
      recommendation: {
        loading: '風險評估進行中，請等待完成後再上線。',
        recommended: '建議上線：風險可控，指標健康。',
        cautious: '謹慎上線：建議先小資金/手動驗證一段時間。',
        notRecommended: 'Not recommended for direct live: high risk or unreliable, optimize before trying.'
      },
      backendRiskScore: {
        title: '策略風險評分',
        loading: '計算中...',
        unknown: '未知',
        reliable: '可靠',
        unreliable: '不可靠',
        reasons: '原因',
        warnings: '警告',
        empty: 'None (save template first, will auto-calculate after backtest completes)'
      },
      score: {
        empty: '暫無評分（等待回測或無指標資料）',
        title: 'Overall Score (heuristic)'
      },
      level: {
        excellent: '優秀',
        good: '良好',
        fair: '一般',
        poor: 'Poor'
      },
      metrics: {
        totalReturn: '總收益率',
        annualReturn: '年化收益率',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '勝率',
        totalTrades: '總交易次數',
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
        deepseek: 'DeepSeek · 高性價比',
        moonshot: 'Kimi · 長上下文',
        qwen: '阿里雲 · 中文最佳化',
        zhipu: '清華系 · 通用',
        openai_compatible: 'Any compatible endpoint'
      },
      pageTitle: 'AI 助手設定',
      pageSubtitle: '設定 AI 大腦 — 選擇模型廠商、管理 API 金鑰與可用模型，並指定全站兜底使用的「預設主模型」。',
      emptyConfigs: '暫無 AI 廠商 設定（系統啟動時會自動建立預設 廠商）',
      section1: {
        title: '選擇模型廠商',
        subtitle: `Cards show each provider's configuration and readiness; click to select`
      },
      statusBar: {
        enabled: '已啟用',
        disabled: '未啟用',
        keyReady: '金鑰就緒',
        checking: '連線檢測中…',
        connected: 'Connected'
      },
      status: {
        noProvider: '尚未選擇供應商',
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
        connectionFailed: 'Connection error, check prompts above'
      },
      cardState: {
        noKey: '未設定',
        noModel: '待選模型',
        enabled: '已啟用',
        readyDisabled: 'Ready · Disabled'
      },
      cardTags: {
        current: '目前',
        hasKey: '已配金鑰',
        noKey: '未配金鑰',
        noModels: '未設定可用模型',
        enabledButUnavailable: 'Enabled but unavailable'
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
        primaryForHint: 'For internal routing: chat / embedding / summarizer / reasoning'
      },
      messages: {
        loadConfigFailed: '載入配置失敗',
        secretSavedAutoDiscover: '金鑰已儲存，正在自動探索模型...',
        secretAutoSaveFailed: '金鑰自動儲存失敗',
        autoDiscoveredModels: '自動探索到{{count}}個模型（僅供參考）',
        autoValidatedModels: '自動驗證：找到{{count}}個模型',
        configSaved: '配置已儲存',
        configSaveFailed: '配置儲存失敗',
        toggleEnabledFailed: '切換啟用狀態失敗',
        secretDeletedConfigReset: '金鑰已刪除，供應商配置已重設為預設值',
        deleteSecretFailed: '刪除金鑰失敗',
        validationPassedModels: '驗證通過：找到{{count}}個模型',
        validationFailedNeedApiKey: 'Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.'
      },
      customProvider: {
        deleted: '自訂提供者已刪除',
        fillNameFirst: '请先填写名稱',
        nameHint: '用于识别此提供者的唯一名稱',
        nameLabel: '提供者名稱',
        namePlaceholder: '我的自訂提供者',
        nameRequired: 'Provider name is required'
      }
    },
    tabs: {
      settings: '設定',
      agentSettings: '專家設定',
      gate: 'AI Gate'
    },
    gate: {
      title: 'AI閘道進度',
      pipelineDesc: '6階段閘道管線：合規性 → 前瞻偏差 → 前進式驗證 → 縮減夏普 → 紙交易 → 相關性',
      labels: {
        compliance: '合規性',
        lookahead: '前瞻偏差',
        walkforward: '前進式驗證',
        deflated_sharpe: '縮減夏普比率',
        paper: '紙交易',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'DSL表達式非空驗證',
        lookahead: '未來函數引用掃描（close[t+N]、ref負偏移）',
        walkforward: '淨化前進式交叉驗證',
        deflated_sharpe: 'Lopez de Prado縮減夏普比率',
        paper: '≥14天紙交易驗證',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: '評估中...'
      },
      skipped: 'SKIPPED',
      noData: '無數據',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: '未知',
      selectRun: '選擇回測運行...',
      strategyParams: '策略參數',
      dslExpression: 'DSL表達式',
      dailyReturns: '每日收益率（逗號或換行分隔）',
      numAttempts: '策略嘗試次數',
      paperMetrics: '紙交易指標',
      paperDays: '紙交易天數',
      paperNetPnL: '紙交易淨損益',
      paperNetReturn: '紙交易淨收益率',
      paperTradeCount: '紙交易次數',
      backtestNetReturn: '回測淨收益率',
      backtestGrossReturn: '回測總收益率',
      runPipeline: '執行閘道管線',
      retry: '重試',
      gateProgress: '閘道評估進度',
      pipelineResult: '管線結果',
      allPassed: '全部6個閘道已通過——策略符合推廣至上線評估條件',
      failed: '失敗：{{gate}}',
      details: '詳情',
      evaluating: '評估中...',
      runHint: 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.'
    },
    gateway: {
      title: 'AI 網關',
      useGateway: 'AI 網關',
      useGatewayDesc: '扣錢包餘額 · 按 Token 計費',
      useOwnKey: '我的 API Key',
      useOwnKeyDesc: '直付廠商 · 自行管理',
      useOwnKeyHint: '使用你自己的 API Key，直接向所選廠商付費。在下方選擇廠商卡片進行配置。',
      selectModel: '選擇模型',
      modelPlaceholder: '選擇 AI 模型',
      noModels: '暫無可用模型',
      balance: '錢包餘額',
      monthlyTokens: '本月 Token',
      monthlyCost: '本月費用',
      usageByFeature: '按功能用量',
    }
  }
} as const;

export default aiCore;
