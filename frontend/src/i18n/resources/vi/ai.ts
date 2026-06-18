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
        title: 'Trạng thái thị trường / phong cách',
        prompt: `You are a senior quantitative strategy analyst. Based on the following information, recommend a strategy paradigm: trend / mean reversion / short-term, and explain the reasoning, applicable conditions and inapplicable scenarios.

Output requirements: use Markdown, must include:
1) Reasoning process: how you derive from data/constraints/objectives (bullet points)
2) Conclusion: main recommendation (only one primary paradigm) + alternative + applicable/inapplicable conditions
3) Risk alerts: at least 3

{{baseInfo}}`
      },
      signals: {
        title: 'Tín hiệu & chỉ báo',
        prompt: `You are a quantitative factor and signal engineer. Without relying on external data (unless the user provides macro event tables), design actionable trading signals.

Requirements: clearly define entry/exit/filter conditions, preferably parameterized, avoid overfitting.

Output requirements: use Markdown, must include:
1) Reasoning process: why choose these indicators/thresholds/filter conditions (bullet points)
2) Conclusion: executable rule list (entry/exit/filter), with parameter suggestions (default/range)
3) Boundaries and risks: at least 3 (e.g.: range-bound/gap/high volatility/news events)

{{baseInfo}}`
      },
      risk: {
        title: 'Rủi ro & ràng buộc thực thi',
        prompt: `You are a trading risk and execution expert. Based on the following information, design position management, stop-loss/take-profit, max drawdown control, cooldown period/trade frequency limits, etc.

Output requirements: use Markdown, must include:
1) Reasoning process: why these controls match objectives/constraints (bullet points)
2) Conclusion: hard constraints + default parameters (suggested/range) + actions after trigger
3) Failure modes: at least 3 (e.g.: consecutive losses, slippage widening, spread anomalies)

{{baseInfo}}`
      },
      code: {
        title: 'Sinh mã',
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
      title: 'Đồng thuận & Thảo luận',
      actions: {
        refresh: 'Refresh'
      },
      fields: {
        account: 'Tài khoản',
        symbol: 'Mã chứng khoán',
        timeframe: 'Timeframe'
      },
      panel: {
        title: 'Điểm mục tiêu',
        decision: 'Quyết định',
        overallScore: 'Tổng thể',
        technicalScore: 'Technical'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'Tín hiệu'
        },
        macd: {
          value: 'MACD',
          signalLine: 'Đường tín hiệu',
          hist: 'Biểu đồ cột',
          flag: 'Tín hiệu',
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
      emptyDescription: 'Bắt đầu trò chuyện với trợ lý AI',
      thinking: 'Đang suy nghĩ...',
      truncated: 'Nội dung quá dài và đã bị cắt bớt',
      expandAll: 'Mở rộng tất cả',
      collapse: 'Collapse'
    },
    reports: {
      tradeAnalysis: {
        title: 'Báo cáo phân tích giao dịch AI',
        riskAssessmentPrefix: 'Risk Assessment:'
      }
    },
    signalCard: {
      status: {
        pending: 'Chờ xác nhận',
        confirmed: 'Đã xác nhận',
        executed: 'Đã thực thi',
        cancelled: 'Cancelled'
      },
      labels: {
        price: 'Giá',
        volume: 'Khối lượng',
        confidence: 'Độ tin cậy',
        stopLoss: 'Cắt lỗ',
        takeProfit: 'Chốt lời',
        analysisReason: 'Analysis Reason'
      },
      actions: {
        confirm: 'Xác nhận',
        cancel: 'Hủy',
        executeTrade: 'Execute Trade'
      },
      confirmCancel: {
        title: 'Are you sure you want to cancel this signal?'
      },
      confirmExecute: {
        title: 'Thực thi tín hiệu giao dịch này?',
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
        active: 'Đang chạy',
        inactive: 'Đã dừng',
        paused: 'Paused'
      },
      actionType: {
        buy: 'Mua',
        sell: 'Bán',
        closeLong: 'Đóng long',
        closeShort: 'Đóng short',
        alert: 'Alert'
      },
      labels: {
        triggeredCount: 'Kích hoạt {{count}} lần',
        lastTriggeredAt: 'Last triggered: {{time}}'
      },
      sections: {
        conditions: 'Điều kiện kích hoạt',
        actions: 'Actions'
      },
      tooltips: {
        createdAt: 'Thời gian tạo',
        lastTriggeredAt: 'Last triggered'
      },
      actions: {
        start: 'Bắt đầu',
        stop: 'Stop'
      },
      confirmDelete: {
        title: 'Xóa chiến lược này?',
        description: 'Cannot be recovered after deletion'
      }
    },
    requireConfig: {
      title: 'Chưa cấu hình AI',
      description: 'Vui lòng cấu hình nhà cung cấp, mô hình và API key trong Cài đặt trước khi dùng trình hướng dẫn hoặc trò chuyện.',
      actions: {
        goSettings: 'Go to Settings'
      }
    },
    riskEval: {
      failed: 'Risk evaluation failed'
    },
    workflowRuns: {
      title: 'Quy trình AI',
      defaultTitle: 'Quy trình AI',
      hints: {
        selectToViewDetail: 'Select a run from the left to view details'
      },
      messages: {
        loadListFailed: 'Tải danh sách lần chạy thất bại',
        loadDetailFailed: 'Failed to load details'
      }
    },
    backtestScoreCard: {
      title: 'Thẻ điểm backtest',
      stateLabel: 'Trạng thái',
      status: {
        succeeded: 'Thành công',
        running: 'Đang chạy',
        pending: 'Đang chờ',
        failed: 'Thất bại',
        cancelRequested: 'Đang hủy',
        canceled: 'Cancelled'
      },
      recommendation: {
        loading: 'Đang đánh giá rủi ro. Vui lòng chờ xong trước khi triển khai.',
        recommended: 'Khuyến nghị triển khai: rủi ro trong mức kiểm soát, chỉ số nhìn chung tốt.',
        cautious: 'Triển khai thận trọng: bắt đầu nhỏ / theo dõi thủ công một thời gian.',
        notRecommended: 'Not recommended for direct live: high risk or unreliable, optimize before trying.'
      },
      backendRiskScore: {
        title: 'Điểm rủi ro chiến lược',
        loading: 'Đang tính...',
        unknown: 'không xác định',
        reliable: 'Đáng tin cậy',
        unreliable: 'Không đáng tin cậy',
        reasons: 'Lý do',
        warnings: 'Cảnh báo',
        empty: 'None (save template first, will auto-calculate after backtest completes)'
      },
      score: {
        empty: 'Chưa có điểm (đợi backtest hoàn tất hoặc thiếu metrics)',
        title: 'Overall Score (heuristic)'
      },
      level: {
        excellent: 'Xuất sắc',
        good: 'Tốt',
        fair: 'Trung bình',
        poor: 'Poor'
      },
      metrics: {
        totalReturn: 'Tổng lợi nhuận',
        annualReturn: 'Lợi nhuận năm',
        maxDrawdown: 'Sụt giảm tối đa',
        sharpe: 'Sharpe',
        winRate: 'Tỷ lệ thắng',
        totalTrades: 'Số lệnh',
        equityPoints: 'Equity points'
      },
      chart: {
        title: 'Equity Curve'
      }
    },
    systemAI: {
      taglines: {
        openai: 'Dòng GPT · Chính thức',
        anthropic: 'Dòng Claude',
        deepseek: 'DeepSeek · Hiệu quả chi phí',
        moonshot: 'Kimi · Ngữ cảnh dài',
        qwen: 'Alibaba Cloud · Tối ưu tiếng Trung',
        zhipu: 'Hệ Thanh Hoa · Đa dụng',
        openai_compatible: 'Any compatible endpoint'
      },
      pageTitle: 'Cài đặt trợ lý AI',
      pageSubtitle: 'Cấu hình bộ não AI — chọn nhà cung cấp, quản lý API key và model khả dụng, và chỉ định "Mô hình chính mặc định" dùng cho toàn hệ thống.',
      emptyConfigs: 'Chưa có cấu hình AI Provider (hệ thống sẽ tạo provider mặc định khi khởi động).',
      section1: {
        title: 'Chọn nhà cung cấp model',
        subtitle: `Cards show each provider's configuration and readiness; click to select`
      },
      statusBar: {
        enabled: 'Đã bật',
        disabled: 'Chưa bật',
        keyReady: 'Key sẵn sàng',
        checking: 'Đang kiểm tra kết nối…',
        connected: 'Connected'
      },
      status: {
        noProvider: 'Chưa chọn nhà cung cấp',
        noProviderDesc: 'Chọn một nhà cung cấp trong các thẻ bên dưới để bắt đầu',
        error: 'Có lỗi',
        ready: 'Sẵn sàng',
        readyDesc: 'đã bật và kết nối bình thường',
        notEnabled: 'Đã kết nối nhưng chưa bật',
        notEnabledDesc: 'Bật công tắc Enable để đưa vào sử dụng',
        configReady: 'Đã sẵn sàng cấu hình',
        configReadyDesc: 'Thêm model khả dụng để hệ thống tự kiểm tra kết nối',
        checkUrl: 'Kiểm tra Base URL',
        checkUrlDesc: 'Đã có API key nhưng URL có vẻ không hợp lệ',
        needKey: 'Cần cấu hình API key',
        needKeyDesc: 'Nhập API key, hệ thống sẽ tự phát hiện danh sách model',
        connectionFailed: 'Connection error, check prompts above'
      },
      cardState: {
        noKey: 'Chưa cấu hình',
        noModel: 'Chọn model',
        enabled: 'Đã bật',
        readyDisabled: 'Ready · Disabled'
      },
      cardTags: {
        current: 'Hiện tại',
        hasKey: 'Đã có key',
        noKey: 'Chưa có key',
        noModels: 'Chưa cấu hình model khả dụng',
        enabledButUnavailable: 'Enabled but unavailable'
      },
      fields: {
        autoFetching: 'Đang tự lấy danh sách…',
        baseUrlCustomHint: 'Nhập endpoint tương thích OpenAI, ví dụ https://model.example.com/v1',
        baseUrlReadonlyHint: 'Địa chỉ chính thức do hệ thống quản lý, không sửa được',
        baseUrlCustomPlaceholder: 'Ví dụ: https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: 'Địa chỉ chính thức (chỉ đọc)',
        httpWarning: 'Đang dùng HTTP — môi trường sản xuất nên dùng HTTPS',
        apiKeyHint: 'Sau khi nhập sẽ được mã hoá và lưu tự động, không cần submit thủ công',
        apiKeyPastePlaceholder: 'Dán API key — sẽ được lưu trước tự động',
        enabledHint: 'Tắt thì nhà cung cấp này sẽ không được hệ thống sử dụng',
        temperatureHint: 'Cao hơn = đa dạng hơn, thấp hơn = ổn định hơn',
        timeoutHint: 'Thời gian chờ tối đa cho mỗi request',
        maxTokensHint: 'Token tối đa cho mỗi phản hồi',
        primaryFor: 'Mục đích chính (Primary For)',
        primaryForHint: 'For internal routing: chat / embedding / summarizer / reasoning'
      },
      messages: {
        loadConfigFailed: 'Tải cấu hình thất bại',
        secretSavedAutoDiscover: 'Đã lưu khóa bí mật, đang tự động khám phá mô hình...',
        secretAutoSaveFailed: 'Tự động lưu khóa bí mật thất bại',
        autoDiscoveredModels: 'Đã tự động khám phá {{count}} mô hình (chỉ để gợi ý)',
        autoValidatedModels: 'Đã tự động xác thực: tìm thấy {{count}} mô hình',
        configSaved: 'Đã lưu cấu hình',
        configSaveFailed: 'Lưu cấu hình thất bại',
        toggleEnabledFailed: 'Thay đổi trạng thái bật/tắt thất bại',
        secretDeletedConfigReset: 'Đã xóa khóa bí mật, cấu hình nhà cung cấp đặt lại về mặc định',
        deleteSecretFailed: 'Xóa khóa bí mật thất bại',
        validationPassedModels: 'Xác thực thành công: tìm thấy {{count}} mô hình',
        validationFailedNeedApiKey: 'Validation failed: this provider typically requires an API Key. Please fill and save the key first, then retry.'
      },
      customProvider: {
        deleted: 'Đã xóa nhà cung cấp tùy chỉnh',
        fillNameFirst: 'Vui lòng điền tên trước',
        nameHint: '用于识别此提供商的唯一名称',
        nameLabel: 'Tên Nhà Cung Cấp',
        namePlaceholder: 'Nhà Cung Cấp Của Tôi',
        nameRequired: 'Provider name is required'
      }
    },
    tabs: {
      settings: 'Cài đặt',
      agentSettings: 'Thiết lập chuyên gia',
      gate: 'AI Gate'
    },
    gate: {
      title: 'Tiến trình Cổng AI',
      pipelineDesc: 'Quy trình 6 cổng: Tuân thủ -> Nhìn về phía trước -> Walk-Forward -> Deflated Sharpe -> Giấy -> Tương quan',
      labels: {
        compliance: 'Tuân thủ',
        lookahead: 'Độ chệch nhìn về phía trước',
        walkforward: 'Walk-Forward',
        deflated_sharpe: 'Deflated Sharpe',
        paper: 'Giao dịch giấy',
        correlation: 'Correlation'
      },
      descriptions: {
        compliance: 'Xác thực biểu thức DSL không rỗng',
        lookahead: 'Quét tham chiếu hàm tương lai (close[t+N], ref offset âm)',
        walkforward: 'Xác thực chéo Walk-Forward đã thanh lọc',
        deflated_sharpe: 'Tỷ lệ Sharpe đã giảm phát của Lopez de Prado',
        paper: 'Xác thực giao dịch giấy >=14 ngày',
        correlation: 'Signal correlation check with existing strategies'
      },
      status: {
        evaluating: 'Đang đánh giá...'
      },
      skipped: 'SKIPPED',
      noData: 'không có dữ liệu',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: 'không xác định',
      selectRun: 'Chọn lần chạy kiểm thử lùi...',
      strategyParams: 'Tham số chiến lược',
      dslExpression: 'Biểu thức DSL',
      dailyReturns: 'Lợi nhuận hàng ngày (phân cách bằng dấu phẩy hoặc xuống dòng)',
      numAttempts: 'Số lần thử chiến lược',
      paperMetrics: 'Chỉ số giao dịch giấy',
      paperDays: 'Ngày giao dịch giấy',
      paperNetPnL: 'Lãi/Lỗ ròng giấy',
      paperNetReturn: 'Lợi nhuận ròng giấy',
      paperTradeCount: 'Số giao dịch giấy',
      backtestNetReturn: 'Lợi nhuận ròng backtest',
      backtestGrossReturn: 'Lợi nhuận gộp backtest',
      runPipeline: 'Chạy quy trình Cổng',
      retry: 'Thử lại',
      gateProgress: 'Tiến trình đánh giá Cổng',
      pipelineResult: 'Kết quả quy trình',
      allPassed: 'Cả 6 cổng đều vượt qua — chiến lược đủ điều kiện đánh giá PromoteToLive',
      failed: 'Thất bại: {{gate}}',
      details: 'Chi tiết',
      evaluating: 'Đang đánh giá...',
      runHint: 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.'
    },
    gateway: {
      title: 'AI Gateway',
      useGateway: 'AI Gateway',
      useGatewayDesc: 'Trừ ví · Tính theo token',
      useOwnKey: 'API Key của tôi',
      useOwnKeyDesc: 'Thanh toán trực tiếp · Tự quản lý',
      useOwnKeyHint: 'Sử dụng API Key của bạn để thanh toán trực tiếp cho nhà cung cấp. Chọn thẻ nhà cung cấp bên dưới để cấu hình.',
      selectModel: 'Chọn mô hình',
      modelPlaceholder: 'Chọn mô hình AI',
      noModels: 'Không có mô hình nào',
      balance: 'Số dư ví',
      monthlyTokens: 'Token tháng này',
      monthlyCost: 'Chi phí tháng này',
      usageByFeature: 'Sử dụng theo tính năng',
    }
  }
} as const;

export default aiCore;
