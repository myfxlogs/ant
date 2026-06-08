const base = {
  app: {
    name: 'AntTrader'
  },
  auth: {
    fields: {
      email: 'Email',
      password: 'Mật khẩu',
      confirmPassword: 'Xác nhận mật khẩu'
    },
    messages: {
      loginSuccess: 'Đăng nhập thành công',
      loginFailed: 'Đăng nhập thất bại. Vui lòng kiểm tra email và mật khẩu.',
      registerSuccess: 'Đăng ký thành công. Vui lòng đăng nhập.',
      registerFailed: 'Đăng ký thất bại. Vui lòng thử lại sau.',
      logoutSuccess: 'Đã đăng xuất',
      fetchMeFailed: 'Không thể tải thông tin người dùng'
    },
    validation: {
      emailRequired: 'Vui lòng nhập email',
      emailInvalid: 'Vui lòng nhập email hợp lệ',
      passwordRequired: 'Vui lòng nhập mật khẩu',
      passwordMin8: 'Mật khẩu phải có ít nhất 8 ký tự',
      confirmPasswordRequired: 'Vui lòng xác nhận mật khẩu',
      passwordMismatch: 'Mật khẩu không khớp'
    },
    login: {
      subtitle: 'Đây là bản thử nghiệm và không chịu trách nhiệm',
      rememberMe: 'Ghi nhớ đăng nhập',
      forgotPassword: 'Quên mật khẩu?',
      signingIn: 'Đang đăng nhập...',
      login: 'Đăng nhập',
      noAccount: 'Chưa có tài khoản?',
      registerNow: 'Đăng ký ngay',
      agreePrefix: 'Bằng việc đăng nhập, bạn đồng ý với',
      terms: 'Điều khoản dịch vụ',
      and: 'và',
      privacy: 'Chính sách quyền riêng tư'
    },
    register: {
      subtitle: 'Tạo tài khoản mới',
      signingUp: 'Đang đăng ký...',
      register: 'Đăng ký',
      haveAccount: 'Đã có tài khoản?',
      loginNow: 'Đăng nhập ngay',
      agreePrefix: 'Bằng việc đăng ký, bạn đồng ý với',
      terms: 'Điều khoản dịch vụ',
      and: 'và',
      privacy: 'Chính sách quyền riêng tư'
    }
  },
  common: {
    refresh: 'Làm mới',
    create: 'Tạo mới',
    back: 'Quay lại',
    updated: 'Đã cập nhật',
    created: 'Đã tạo',
    enabled: 'Đã bật',
    disabled: 'Đã tắt',
    deleted: 'Đã xóa',
    deleteFailed: 'Xóa thất bại',
    loadingFailed: 'Tải thất bại',
    none: 'Không có',
    close: 'Đóng',
    operationFailed: 'Thao tác thất bại',
    pleaseWait: 'Vui lòng chờ...',
    next: 'Tiếp theo',
    previous: 'Quay lại',
    gotIt: 'Đã hiểu',
    loading: 'Đang tải...',
    unknown: 'Không rõ',
    enable: 'Bật',
    disable: 'Tắt',
    edit: 'Chỉnh sửa',
    delete: 'Xóa',
    confirm: 'Xác nhận',
    cancel: 'Hủy',
    save: 'Lưu',
    send: 'Gửi',
    saveFailed: 'Lưu thất bại',
    showDetails: 'Xem chi tiết',
    hideDetails: 'Ẩn chi tiết',
    translate: 'Dịch',
    viewOriginal: 'Xem nguyên văn',
    viewTranslation: 'Xem bản dịch',
    copy: 'Sao chép',
    copied: 'Đã sao chép',
    copyFailed: 'Sao chép thất bại',
    totalItems: 'Tổng {{total}} mục',
    time: {
      minute: 'phút',
      hour: 'giờ',
      day: 'ngày',
      lessThanMinute: '< 1 phút'
    },
    required: 'Bắt buộc',
    noData: 'Không có dữ liệu'
  },
  language: {
    simplifiedChinese: '简体中文',
    traditionalChinese: '繁體中文',
    english: 'English',
    japanese: '日本語',
    vietnamese: 'Tiếng Việt'
  },
  menu: {
    strategyWorkspace: 'Không gian chiến lược',
    dashboard: 'Bảng điều khiển',
    accounts: 'Quản lý tài khoản',
    aiAssistant: 'Trợ lý AI',
    strategies: 'Quản lý chiến lược',
    trading: 'Giao dịch',
    market: 'Thị trường',
    analytics: 'Phân tích',
    marketplace: 'Thị trường',
    experiments: 'Thí nghiệm',
    marketRegime: 'Chế độ thị trường',
    assets: 'Tài sản',
    schedules: 'Lịch chạy chiến lược',
    indicatorCatalog: 'Danh mục chỉ báo',
    logs: 'Nhật ký hệ thống',
    assetAnalysis: 'Phân tích AI'
  },
  market: {
    searchPlaceholder: 'Tìm kiếm mã (VD: EURUSD, XAUUSD)',
    selectAccount: 'Chọn tài khoản giao dịch',
    watchlist: 'Danh sách theo dõi',
    popularSymbols: 'Mã phổ biến',
    noSymbolSelected: 'Chọn một mã để xem dữ liệu thị trường',
    bid: 'Giá mua',
    ask: 'Giá bán',
    spread: 'Chênh lệch',
    mid: 'Giá trung bình'
  },
  topbar: {
    systemOk: 'Hệ thống đang hoạt động bình thường',
    profile: 'Hồ sơ',
    settings: 'Cài đặt',
    switchToAdmin: 'Chuyển sang quản trị',
    logout: 'Đăng xuất',
    user: 'Người dùng'
  },
  profile: {
    title: 'Hồ sơ',
    nickname: 'Biệt danh',
    role: 'Vai trò',
    status: 'Trạng thái',
    lastLogin: 'Đăng nhập cuối',
    registered: 'Đã đăng ký'
  },
  notifications: {
    title: 'Thông báo',
    empty: 'Không có thông báo',
    types: {
      trade: 'Giao dịch',
      signal: 'Tín hiệu',
      risk_alert: 'Rủi ro',
      strategy_execution: 'Chiến lược',
      system: 'Hệ thống'
    },
    tabs: {
      all: 'Tất cả ({{count}})',
      unread: 'Chưa đọc ({{count}})'
    },
    actions: {
      markAllAsRead: 'Đánh dấu đã đọc',
      clearAll: 'Xóa',
      clearAllConfirm: 'Xóa tất cả thông báo?'
    },
    all: 'Tất cả',
    unread: 'Chưa đọc',
    markAllRead: 'Đánh dấu tất cả đã đọc',
    clearAll: 'Xóa tất cả',
    confirmClearAll: 'Xóa tất cả thông báo?',
    stream: {
      strategyExecution: {
        title: 'Thực thi Chiến lược',
        completed: '{{symbol}} {{action}} đã hoàn thành',
        failed: 'Thực thi thất bại: {{error}}'
      },
      riskAlert: {
        title: 'Cảnh báo Rủi ro',
        fallback: 'Loại cảnh báo: {{alertType}}'
      },
      strategySignal: {
        title: 'Tín hiệu Chiến lược',
        message: '{{symbol}} đã kích hoạt {{signalType}}'
      },
      autoTrading: {
        title: 'Giao dịch Tự động',
        fallback: 'Sự kiện giao dịch tự động đã kích hoạt'
      }
    }
  },
  errors: {
    not_authenticated: 'Chưa đăng nhập',
    invalid_credentials: 'Thông tin đăng nhập không hợp lệ',
    user_not_found: 'Không tìm thấy người dùng',
    email_already_registered: 'Email đã được đăng ký',
    account_not_found: 'Không tìm thấy tài khoản',
    access_denied: 'Từ chối truy cập',
    account_connection_failed: 'Không thể kết nối đến máy chủ giao dịch',
    account_connected: 'Kết nối thành công',
    schedule_service_not_available: 'Dịch vụ lịch biểu không khả dụng',
    auto_trading_enabled: 'Đã bật giao dịch tự động',
    auto_trading_disabled: 'Đã tắt giao dịch tự động',
    connection_failed: {
      title: 'Kết nối thất bại',
      content: 'Không thể kết nối đến dịch vụ backend. Vui lòng kiểm tra máy chủ đã chạy hay chưa.'
    },
    ai: {
      not_configured: 'AI chưa được cấu hình. Vui lòng bật và cấu hình trong AI Settings trước.',
      config_service_not_initialized: 'Dịch vụ cấu hình AI chưa được khởi tạo',
      config_valid: 'Cấu hình AI hợp lệ',
      no_trade_data_available: 'Không có dữ liệu giao dịch',
      provider_returned_empty_message: 'Nhà cung cấp AI trả về thông điệp rỗng',
      provider_required: 'Vui lòng chọn nhà cung cấp',
      invalid_provider: 'Nhà cung cấp không hợp lệ',
      api_key_required: 'API key là bắt buộc',
      base_url_required: 'Base URL là bắt buộc',
      invalid_base_url: 'Base URL không hợp lệ',
      base_url_scheme_invalid: 'Base URL phải bắt đầu bằng http:// hoặc https://',
      base_url_should_not_end_with_chat_completions: 'Base URL không được kết thúc bằng /chat/completions',
      failed_to_create_request: 'Không thể tạo request',
      request_failed: 'Yêu cầu API thất bại',
      probe_ok: 'OK',
      probe_ok_no_models: 'OK (không trả về models)',
      free_tier_exhausted: 'Đã hết hạn mức miễn phí của AI. Vui lòng tắt “use free tier only” trong trang quản trị nhà cung cấp hoặc chuyển sang khóa trả phí.',
      rate_limited: 'AI bị giới hạn tốc độ hoặc hết hạn mức (429/resource exhausted). Vui lòng thử lại sau.',
      forbidden_quota: 'AI thiếu hạn mức/quyền truy cập (403). Vui lòng kiểm tra hạn mức của API key hoặc trạng thái thanh toán.'
    },
    wizard: {
      title: 'Trình hướng dẫn chiến lược AI',
      subtitle: 'Mỗi bước một trang, bạn có thể tiến/lùi',
      currentModel: 'Mô hình hiện tại: {{model}}',
      steps: {
        setup: 'Thiết lập',
        generate: 'Tạo chiến lược',
        publishCode: 'Triển khai - Mã',
        publishBacktest: 'Triển khai - Backtest',
        publishLaunch: 'Triển khai - Khởi chạy'
      },
      actions: {
        prev: 'Trước',
        next: 'Tiếp',
        cancel: 'Hủy'
      },
      agents: {
        styleTitle: 'Trạng thái thị trường / phong cách',
        signalsTitle: 'Tín hiệu & chỉ báo',
        riskTitle: 'Rủi ro & ràng buộc thực thi',
        codeTitle: 'Sinh mã'
      },
      template: {
        defaultName: 'Chiến lược AI {{title}}',
        defaultDescription: 'Tạo bởi trình hướng dẫn AI'
      },
      schedule: {
        defaultName: 'Lịch AI {{symbol}} {{timeframe}}'
      },
      prompts: {
        dataSpec: {
          dataset: 'Sử dụng dataset đã đóng băng datasetId={{datasetId}}',
          klineRange: 'Sử dụng phạm vi nến lịch sử from={{from}} to={{to}}'
        },
        base: {
          account: 'Tài khoản: {{accountId}}',
          symbol: 'Mã: {{symbol}}',
          timeframe: 'Khung thời gian: {{timeframe}}',
          data: 'Dữ liệu: {{dataSpec}}',
          constraints: 'Ràng buộc: max drawdown={{maxDrawdownPct}}% rủi ro/lệnh={{riskPerTradePct}}% tối đa lệnh/ngày={{maxTradesPerDay}}',
          params: `Tham số (định nghĩa + giá trị hiện tại; có trong context["params"] khi chạy):
{{params}}`,
          empty: '(trống)',
          macroEnabled: `Sự kiện vĩ mô (người dùng cung cấp):
{{text}}`,
          macroDisabled: 'Sự kiện vĩ mô: không dùng',
          userIntent: `Mục tiêu (ngôn ngữ tự nhiên):
{{intent}}`
        },
        upstream: {
          style: `[Kết luận trạng thái thị trường / phong cách]
{{text}}`,
          signals: `[Kết luận tín hiệu & chỉ báo]
{{text}}`,
          risk: `[Kết luận rủi ro & ràng buộc]
{{text}}`,
          sectionTitle: '[Kết luận agent phía trên (nguyên văn)]'
        },
        summary: {
          intro: 'Bạn là trợ lý giải thích chiến lược định lượng. Hãy giải thích ý tưởng cốt lõi của đoạn mã chiến lược AntTrader Python dưới đây bằng các gạch đầu dòng ngắn gọn (tối đa 12 dòng) để giúp người dùng đánh giá có đúng kỳ vọng hay không.',
          mustIncludeTitle: 'Bắt buộc gồm:',
          mustInclude1: '1) Loại/kiểu chiến lược (trend/mean-reversion/breakout/momentum/grid... nếu không chắc hãy ghi “Không rõ”)',
          mustInclude2: '2) Điều kiện vào lệnh chính (2-4 ý)',
          mustInclude3: '3) Điều kiện thoát/SL/TP/ràng buộc rủi ro chính (2-4 ý)',
          mustInclude4: '4) 1 bối cảnh phù hợp và 1 bối cảnh không phù hợp',
          userIntent: `Kỳ vọng người dùng (ngôn ngữ tự nhiên):
{{intent}}`,
          codeTitle: 'Mã:'
        }
      },
      messages: {
        generateCodeFirst: 'Vui lòng tạo mã chiến lược trước',
        validateCodeFirst: 'Vui lòng nhấn “Xác thực mã” trước',
        codeInvalidFixAndContinue: 'Xác thực mã thất bại. Hãy sửa trước khi tiếp tục',
        startBacktestFirst: 'Vui lòng bắt đầu backtest trước',
        backtestNotDoneWait: 'Backtest chưa xong. Hãy chờ đến khi trạng thái thành “Succeeded/Failed/Canceled”',
        confirmScoreFirst: 'Vui lòng xác nhận kết quả trong popup điểm số trước',
        fillRequiredWithFields: 'Vui lòng điền các trường bắt buộc: {{fields}}',
        fillRequired: 'Vui lòng điền các trường bắt buộc',
        watchBacktestRunFailed: 'watchBacktestRun thất bại',
        createDraftFailed: 'Không thể tạo bản nháp',
        loadAccountsFailed: 'Không thể tải tài khoản',
        loadSymbolsFailed: 'Không thể tải mã',
        loadDatasetFailed: 'Không thể tải dataset',
        datasetFrozenCreated: 'Đã tạo dataset đóng băng',
        freezeDatasetFailed: 'Không thể đóng băng dataset',
        inputIntentFirst: 'Vui lòng nhập mục tiêu/ý tưởng chiến lược trước',
        aiRequestTimeout: 'Hết thời gian yêu cầu AI (> {{seconds}}s)',
        modelReturnedEmpty: 'Mô hình trả về rỗng',
        noPythonCodeBlock: 'Agent code không xuất \`\`\`python code block\`\`\`. Vui lòng kiểm tra kết quả',
        agentFailed: '{{title}} thất bại',
        userAborted: 'Người dùng đã hủy',
        chatAborted: 'Đã hủy trò chuyện với mô hình',
        noCodeToValidate: 'Không có mã để xác thực',
        validateOk: 'Xác thực thành công',
        validateFailed: 'Xác thực thất bại',
        validateError: 'Lỗi xác thực',
        noCodeToBacktest: 'Không có mã để backtest',
        backtestCreated: 'Đã tạo backtest',
        createBacktestFailed: 'Không thể tạo backtest',
        draftNotCreated: 'Chưa tạo bản nháp',
        draftSaved: 'Đã lưu bản nháp',
        saveFailed: 'Lưu thất bại',
        publishedNoId: 'Đã triển khai nhưng không nhận được id (vui lòng kiểm tra trong quản lý chiến lược)',
        templatePublished: 'Đã triển khai template',
        publishFailed: 'Triển khai thất bại',
        publishTemplateFirst: 'Vui lòng triển khai template trước',
        scheduleCreatedAndEnabled: 'Đã tạo và bật lịch',
        scheduleCreated: 'Đã tạo lịch',
        createScheduleFailed: 'Không thể tạo lịch',
        scheduleAlreadyExists: 'Đã tồn tại lịch với cùng template+mã+khung thời gian cho tài khoản này. Vui lòng không tạo trùng.'
      }
    },
    translate_failed: 'Dịch thất bại'
  },
  marketplace: {
    title: 'Thị trường Chiến lược',
    subtitle: 'Khám phá, đánh giá và đăng ký chiến lược cộng đồng',
    publish: 'Xuất bản Chiến lược',
    tabs: {
      marketplace: 'Thị trường',
      subscriptions: 'Đăng ký của tôi'
    },
    searchPlaceholder: 'Tìm kiếm chiến lược...',
    filterByClass: 'Lọc theo loại tài sản',
    sort: {
      newest: 'Mới nhất',
      popular: 'Phổ biến nhất',
      performance: 'Hiệu suất tốt nhất'
    },
    empty: 'Chưa có chiến lược nào được xuất bản',
    noSubscriptions: 'Chưa có đăng ký nào',
    card: {
      subscribe: 'Đăng ký',
      subscribed: 'Đã đăng ký',
      unsubscribe: 'Hủy đăng ký',
      unsubscribeHint: 'Nhấp để hủy đăng ký',
      details: 'Chi tiết',
      subscribers: 'Người đăng ký',
      winRate: 'Tỷ lệ thắng',
      by: 'bởi'
    },
    assetClass: {
      forex: 'Ngoại hối',
      crypto: 'Tiền điện tử',
      commodity: 'Hàng hóa',
      index: 'Chỉ số',
      stock: 'Cổ phiếu',
      other: 'Khác'
    },
    risk: {
      low: 'Rủi ro thấp',
      medium: 'Rủi ro trung bình',
      high: 'Rủi ro cao'
    },
    messages: {
      loginFirst: 'Vui lòng đăng nhập trước',
      subscribed: 'Đăng ký thành công',
      subscribeFailed: 'Đăng ký thất bại',
      unsubscribed: 'Đã hủy đăng ký',
      unsubscribeFailed: 'Hủy đăng ký thất bại',
      rated: 'Đã gửi đánh giá',
      rateFailed: 'Đánh giá thất bại',
      commentPosted: 'Đã đăng bình luận',
      commentFailed: 'Bình luận thất bại'
    },
    detail: {
      comments: 'Bình luận',
      noComments: 'Chưa có bình luận. Hãy là người đầu tiên!',
      commentPlaceholder: 'Viết bình luận... (Shift+Enter để xuống dòng)'
    }
  },
  admin: {
    sidebar: {
      dashboard: 'Dashboard',
      userManagement: 'User Management',
      accountManagement: 'Account Management',
      tradingMonitor: 'Trading Monitor',
      operationLogs: 'Operation Logs',
      systemConfig: 'System Config',
      jurisdiction: 'Jurisdiction Gate'
    },
    header: {
      adminMode: 'Admin Mode',
      adminPanel: 'Admin Panel',
      backToUser: 'Back to User',
      logout: 'Logout',
      admin: 'Admin'
    },
    config: {
      title: 'System Configuration',
      editConfig: 'Edit Config: {{key}}',
      configItem: 'Config Item',
      value: 'Value',
      description: 'Description',
      status: 'Status',
      toggle: 'Toggle',
      updatedAt: 'Updated At',
      on: 'On',
      off: 'Off',
      maxAccountsPerUser: 'Max Accounts Per User',
      aiProviderCatalog: 'AI Model Provider Catalog',
      econAIConfig: 'Economic Calendar Translation AI Config',
      strategyHealthConfig: 'Strategy Health Grading Config',
      provider: 'Provider',
      modelName: 'Model Name',
      enableToggle: 'Enable',
      baseUrlLabel: 'Base URL (optional, custom/OpenAI compatible only)',
      formatJson: 'Format JSON',
      fillTemplate: 'Fill Example',
      thresholdInfo: 'Threshold Field Description',
      thresholdDesc: 'green_success_rate: green success rate threshold; green_max_failed_runs: max failed runs for green; yellow_success_rate: yellow success rate threshold; min_sample_size: minimum sample size.',
      validation: {
        jsonEmpty: 'JSON cannot be empty',
        jsonInvalid: 'Invalid JSON format',
        greenSuccessRateRange: 'green_success_rate must be between 0 and 100',
        yellowSuccessRateRange: 'yellow_success_rate must be between 0 and 100',
        yellowNotGreaterThanGreen: 'yellow_success_rate cannot be greater than green_success_rate',
        greenMaxFailedRunsNonNegative: 'green_max_failed_runs must be >= 0',
        minSampleSizeNonNegative: 'min_sample_size must be >= 0',
        apiKeyRequired: 'API Key cannot be empty',
        modelRequired: 'Model name cannot be empty'
      },
      messages: {
        loadFailed: 'Failed to load configs',
        updated: 'Config updated',
        updateFailed: 'Update failed',
        enabled: 'Config enabled',
        disabled: 'Config disabled',
        operationFailed: 'Operation failed'
      },
      placeholders: {
        json: 'Enter JSON',
        apiKey: 'Enter API Key',
        model: 'e.g. glm-4-flash / deepseek-chat / gpt-4o-mini',
        baseUrl: 'e.g. https://api.openai.com or self-hosted gateway',
        configValue: 'Enter config value',
        description: 'Enter description'
      },
      providerOptions: {
        zhipu: 'Zhipu',
        deepseek: 'DeepSeek',
        custom: 'Custom / OpenAI Compatible'
      }
    },
    trading: {
      title: 'Trading Monitor',
      loadFailed: 'Failed to load trading statistics',
      platform: 'Platform',
      accounts: 'Accounts',
      orders: 'Orders',
      volume: 'Volume',
      byPlatform: 'By Platform',
      profitStats: 'P&L Statistics',
      totalUsers: 'Total Users',
      activeUsers: 'Active Users',
      totalAccounts: 'Total Accounts',
      connectedAccounts: 'Connected Accounts',
      totalOrders: 'Total Orders',
      closedOrders: 'Closed Orders',
      totalVolume: 'Total Volume',
      netProfit: 'Net P&L',
      totalProfit: 'Total Profit',
      totalLoss: 'Total Loss',
      pendingOrders: 'Pending Orders'
    },
    dashboard: {
      title: 'Admin Dashboard',
      loadFailed: 'Failed to load dashboard data',
      totalUsers: 'Total Users',
      activeUsers: 'Active Users',
      mtAccounts: 'MT Accounts',
      onlineAccounts: 'Online Accounts',
      todayTrades: 'Today Trades',
      todayProfit: 'Today P&L',
      recentLogs: 'Recent Operation Logs',
      logs: {
        time: 'Time',
        module: 'Module',
        actionType: 'Action Type',
        target: 'Target',
        status: 'Status',
        success: 'Success',
        failed: 'Failed',
        moduleMap: {
          userManagement: 'User Management',
          accountManagement: 'Account Management',
          trading: 'Trading',
          systemConfig: 'System Config'
        }
      },
      riskMetrics: {
        title: 'Risk Control Metrics (Real-time)',
        riskValidateTotal: 'Risk Validated Total',
        riskValidatePass: 'Risk Validated Pass',
        riskValidateReject: 'Risk Validated Reject',
        riskValidateError: 'Risk Validated Error',
        orderSendSuccess: 'Order Sent Success',
        orderSendFailed: 'Order Sent Failed',
        orderCloseSuccess: 'Order Closed Success',
        orderCloseFailed: 'Order Closed Failed'
      },
      riskWindow: {
        title: 'Risk Control Window Metrics (1h / 24h / 72h)',
        validateTotal: '{{window}} Validated Total',
        validatePass: '{{window}} Pass',
        validateReject: '{{window}} Reject',
        validateError: '{{window}} Error',
        orderSendSuccess: '{{window}} Order Sent',
        orderSendFailed: '{{window}} Order Failed',
        orderCloseSuccess: '{{window}} Close Success',
        orderCloseFailed: '{{window}} Close Failed',
        rejectRiskCodesHeader: 'Top N Reject Risk Codes ({{window}})',
        rejectCount: 'Reject Count',
        noRejectData: 'No reject data for current window',
        noData: 'No window metrics data'
      }
    },
    jurisdiction: {
      title: 'Jurisdiction Gate',
      sanctionedCountriesTab: 'Sanctioned Countries',
      kycStatusTab: 'User KYC Status',
      sanctionedCountries: 'Sanctioned Countries',
      userKYCStatus: 'User KYC Status',
      addCountry: 'Add Country',
      addSanctionedCountry: 'Add Sanctioned Country',
      countryCode: 'Country Code',
      countryLabel: 'Label',
      addedBy: 'Added By',
      actions: 'Actions',
      userEmail: 'Email',
      kycStatus: 'KYC Status',
      country: 'Country',
      sanctioned: 'Sanctioned',
      disclaimer: 'Disclaimer',
      questionnaire: 'Questionnaire',
      override: 'Override',
      setKYC: 'Set KYC',
      setKYCStatus: 'Set KYC Status',
      grantOverride: 'Grant Override',
      revokeOverride: 'Revoke Override',
      filterByKYCStatus: 'Filter by KYC status',
      unverified: 'Unverified',
      pending: 'Pending',
      verified: 'Verified',
      rejected: 'Rejected',
      emptySanctions: 'No sanctioned countries configured',
      emptyKYC: 'No users match the selected KYC filter',
      messages: {
        countryAdded: 'Sanctioned country added',
        countryAddFailed: 'Failed to add sanctioned country',
        countryRemoved: 'Sanctioned country removed',
        countryRemoveFailed: 'Failed to remove sanctioned country',
        kycUpdated: 'KYC status updated',
        kycUpdateFailed: 'Failed to update KYC status',
        overrideUpdated: 'Sanctioned override updated',
        overrideUpdateFailed: 'Failed to update sanctioned override'
      }
    }
  },
  symbolDetection: {
    label: 'Biểu tượng được Phát hiện',
    loading: 'Đang phân tích…',
    noSymbols: 'Không phát hiện biểu tượng giao dịch. Thử bao gồm tên biểu tượng cụ thể (ví dụ: "Bitcoin", "EURUSD", "Vàng").',
    unresolvedTooltip: 'Chưa liên kết tài khoản giao dịch, không thể phân giải',
    resolvedTooltip: 'môi giới: {{broker}} | chế độ: {{mode}}',
    tradeMode: {
      disabled: 'Đã tắt',
      longOnly: 'Chỉ Mua',
      shortOnly: 'Chỉ Bán',
      longShort: 'Cả Mua & Bán',
      unknown: 'Không xác định({{mode}})'
    }
  }
} as const;

export default base;
