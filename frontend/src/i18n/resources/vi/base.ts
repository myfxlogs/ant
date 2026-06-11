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
      registerNow: 'Đăng ký ngay'
    },
    register: {
      subtitle: 'Tạo tài khoản mới',
      signingUp: 'Đang đăng ký...',
      register: 'Đăng ký',
      haveAccount: 'Đã có tài khoản?',
      loginNow: 'Đăng nhập ngay'
    },
    forgotPassword: {
      title: 'Đặt lại Mật khẩu',
      hint: 'Vui lòng liên hệ quản trị viên hoặc hỗ trợ để đặt lại mật khẩu.',
      backToLogin: 'Quay lại Đăng nhập'
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
  deleteSelected: 'Xóa {{count}} mục đã chọn',
    loadingFailed: 'Tải thất bại',
    none: 'Không có',
    close: 'Đóng',
    operationFailed: 'Thao tác thất bại',
    pleaseWait: 'Vui lòng chờ...',
    next: 'Tiếp theo',
    previous: 'Quay lại',
    gotIt: 'Đã hiểu',
    loading: 'Đang tải...',
    searching: 'Đang tìm...',
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
    totalItems: 'Tổng {{count}} mục',
    time: {
      minute: '{{n}}ph',
      hour: '{{n}}giờ',
      day: '{{n}}ngày',
      lessThanMinute: '<1ph'
    },
    required: 'Bắt buộc',
    noData: 'Không có dữ liệu',
    ok: 'OK',
    error: 'Lỗi',
    retry: 'Thử lại',
    pageError: 'Lỗi trang',
    unexpectedError: 'Đã xảy ra lỗi không mong muốn',
    lineColor: 'Màu đường',
    selectSymbolToViewChart: 'Chọn mã để xem biểu đồ',
    currentPosition: '📊 Vị thế hiện tại',
    noOpenPositionsForSymbol: 'Không có vị thế mở cho {{symbol}}',
    indicatorSettings: 'Cài đặt {{name}}',
    active: 'Hoạt Động',
    inactive: 'Không Hoạt Động',
    clear: 'Xóa',
    saveSuccess: 'Đã Lưu',
    remove: 'Xóa',
    yes: 'Có',
    no: 'Không',
    you: 'Bạn',
    comingSoon: 'Sắp Ra Mắt',
    pageUnderDevelopment: 'Trang Đang Phát Triển'
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
    strategyLibrary: 'Thư viện chiến lược',
    dashboard: 'Bảng điều khiển',
    strategy: 'Chiến lược',
    accounts: 'Quản lý tài khoản',
    aiAssistant: 'Trợ lý AI',
    strategies: 'Quản lý chiến lược',
    trading: 'Giao dịch',
    wallet: 'Ví',
    algoDashboard: 'Thuật toán',
    market: 'Thị trường',
    analytics: 'Phân tích',
    marketplace: 'Thị trường',
    experiments: 'Thí nghiệm',
    marketRegime: 'Chế độ thị trường',
    assets: 'Tài sản',
    schedules: 'Lịch chạy chiến lược',
    indicatorCatalog: 'Danh mục chỉ báo',
    logs: 'Nhật ký hệ thống',
    assetAnalysis: 'Phân tích AI',
    autoTrading: 'Giao Dịch Tự Động',
    marketTools: 'Công cụ thị trường',
    devGroup: 'Phát triển',
    opsGroup: 'Vận hành',
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
    mid: 'Giá trung bình',
    allSymbols: 'Tất cả mã',
    common: 'Phổ biến',
    selectSymbol: 'Chọn mã giao dịch',
    noSymbolsFound: 'Không tìm thấy mã nào',
    loadingSymbols: 'Đang tải...',
    emptyWatchlist: 'Danh sách trống',
    searchSymbol: 'Tìm mã...'
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
      marketplace: 'Thị Trường',
      subscriptions: 'Đăng Ký Của Tôi'
    },
    searchPlaceholder: 'Tìm kiếm chiến lược...',
    filterByClass: 'Lọc theo loại tài sản',
    sort: {
      newest: 'Mới Nhất',
      popular: 'Phổ Biến Nhất',
      performance: 'Hiệu Suất Tốt Nhất'
    },
    empty: 'Chưa có chiến lược nào được xuất bản',
    noSubscriptions: 'Chưa có đăng ký nào',
    card: {
      subscribe: 'Đăng ký',
      subscribed: 'Đã Đăng Ký',
      unsubscribe: 'Hủy Đăng Ký',
      unsubscribeHint: 'Nhấp để hủy đăng ký',
      details: 'Chi Tiết',
      subscribers: 'Người đăng ký',
      winRate: 'Tỷ Lệ Thắng',
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
      low: 'Thấp',
      medium: 'Trung Bình',
      high: 'Cao'
    },
    messages: {
      loginFirst: 'Vui lòng đăng nhập trước',
      subscribed: 'Đăng ký thành công',
      subscribeFailed: 'Đăng ký thất bại',
      unsubscribed: 'Đã hủy đăng ký',
      unsubscribeFailed: 'Hủy đăng ký thất bại',
      rated: 'Đã gửi đánh giá',
      rateFailed: '评分失败',
      commentPosted: 'Đã đăng bình luận',
      commentFailed: '评论失败',
      publishFailed: 'Đăng thất bại',
      published: 'Đã đăng thành công'
    },
    detail: {
      comments: 'Bình luận',
      noComments: 'Chưa có bình luận. Hãy là người đầu tiên!',
      commentPlaceholder: 'Viết bình luận... (Shift+Enter để xuống dòng)'
    },
    publishModal: {
      symbolsPlaceholder: 'EURUSD, GBPUSD, XAUUSD',
      strategyId: '策略ID',
      title: '发布策略',
      titleField: '标题',
      titlePlaceholder: '输入策略标题',
      description: '描述',
      assetClass: '资产类别',
      riskLevel: '风险等级',
      priceModel: '价格模式',
      priceAmount: '价格',
      symbols: '交易品种',
      tags: '标签',
      timeframe: '时间周期',
      submit: '发布'
    },
    priceModel: {
      free: 'Miễn Phí',
      subscription: 'Đăng Ký',
      performanceFee: 'Phí Hiệu Suất'
    }
  },
  admin: {
    sidebar: {
      dashboard: 'Dashboard',
      userManagement: 'User Management',
      walletManagement: 'Wallets',
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
      configItem: '配置项',
      value: '值',
      description: '描述',
      status: '状态',
      toggle: '切换',
      updatedAt: '更新时间',
      on: '开',
      off: '关',
      maxAccountsPerUser: '每用户最大账户数',
      aiProviderCatalog: 'AI提供商目录',
      econAIConfig: '经济日历AI配置',
      strategyHealthConfig: '策略健康度配置',
      provider: '提供商',
      modelName: '模型名称',
      enableToggle: '启用',
      baseUrlLabel: 'Base URL',
      formatJson: '格式化JSON',
      fillTemplate: '填充模板',
      thresholdInfo: '阈值说明',
      thresholdDesc: '阈值描述',
      validation: {
        jsonEmpty: 'JSON不能为空',
        jsonInvalid: 'JSON格式无效',
        greenSuccessRateRange: '绿色成功率需在0-100之间',
        yellowSuccessRateRange: '黄色成功率需在0-100之间',
        yellowNotGreaterThanGreen: '黄色阈值不能超过绿色阈值',
        greenMaxFailedRunsNonNegative: '绿色最大失败次数需≥0',
        minSampleSizeNonNegative: '最小样本量需≥0',
        apiKeyRequired: 'API Key不能为空',
        modelRequired: '模型名称不能为空'
      },
      messages: {
        loadFailed: '加载配置失败',
        updated: '配置已更新',
        updateFailed: '更新配置失败',
        enabled: '已启用',
        disabled: '已禁用',
        operationFailed: '操作失败'
      },
      placeholders: {
        json: '输入JSON',
        apiKey: '输入API Key',
        model: '输入模型名称',
        baseUrl: '输入Base URL',
        configValue: '输入配置值',
        description: '输入描述'
      },
      providerOptions: {
        zhipu: '智谱AI',
        deepseek: 'DeepSeek',
        custom: '自定义'
      }
    },
    trading: {
      title: 'Giám Sát Giao Dịch',
      loadFailed: 'Failed to load trading statistics',
      platform: 'Nền Tảng',
      accounts: 'Tài Khoản',
      orders: 'Lệnh',
      volume: 'Khối Lượng',
      byPlatform: '按平台',
      profitStats: 'Thống Kê Lợi Nhuận',
      totalUsers: 'Tổng Người Dùng',
      activeUsers: 'Người Dùng Hoạt Động',
      totalAccounts: 'Tổng Tài Khoản',
      connectedAccounts: 'Đã Kết Nối',
      totalOrders: 'Tổng Lệnh',
      closedOrders: 'Đã Đóng',
      totalVolume: 'Tổng Khối Lượng',
      netProfit: 'Lợi Nhuận Ròng',
      totalProfit: 'Tổng Lợi Nhuận',
      totalLoss: 'Tổng Thua Lỗ',
      pendingOrders: 'Lệnh Chờ'
    },
    dashboard: {
      title: 'Bảng Điều Khiển Quản Trị',
      loadFailed: 'Tải dữ liệu bảng điều khiển thất bại',
      totalUsers: 'Tổng Người Dùng',
      activeUsers: 'Người Dùng Hoạt Động',
      mtAccounts: 'Tài Khoản MT',
      onlineAccounts: 'Trực Tuyến',
      todayTrades: 'Giao Dịch Hôm Nay',
      todayProfit: 'Lợi Nhuận Hôm Nay',
      recentLogs: 'Nhật Ký Gần Đây',
      logs: {
        time: 'Thời Gian',
        module: 'Mô-đun',
        actionType: 'Hành Động',
        target: 'Mục Tiêu',
        status: 'Trạng Thái',
        success: 'Thành Công',
        failed: 'Thất Bại',
        moduleMap: {
          userManagement: 'Quản Lý Người Dùng',
          accountManagement: 'Quản Lý Tài Khoản',
          trading: 'Giao Dịch',
          systemConfig: 'Cấu Hình Hệ Thống'
        }
      },
      riskMetrics: {
        title: '风控指标',
        riskValidateTotal: '总验证数',
        riskValidatePass: '通过',
        riskValidateReject: '拒绝',
        riskValidateError: '错误',
        orderSendSuccess: '下单成功',
        orderSendFailed: '下单失败',
        orderCloseSuccess: '平仓成功',
        orderCloseFailed: '平仓失败'
      },
      riskWindow: {
        title: '风控窗口',
        validateTotal: '总计',
        validatePass: '通过',
        validateReject: '拒绝',
        validateError: '错误',
        orderSendSuccess: '下单成功',
        orderSendFailed: '下单失败',
        orderCloseSuccess: '平仓成功',
        orderCloseFailed: '平仓失败',
        rejectRiskCodesHeader: '风控代码',
        rejectCount: '拒绝次数',
        noRejectData: '本时段无拒绝记录',
        noData: '暂无风控数据'
      }
    },
    jurisdiction: {
      title: 'Kiểm Soát Quyền Hạn',
      sanctionedCountriesTab: '制裁国家',
      kycStatusTab: 'KYC状态',
      sanctionedCountries: 'Quốc Gia Bị Cấm Vận',
      userKYCStatus: '用户KYC状态',
      addCountry: 'Thêm Quốc Gia',
      addSanctionedCountry: '添加制裁国家',
      countryCode: 'Mã Quốc Gia',
      countryLabel: 'Quốc Gia',
      addedBy: 'Người Thêm',
      actions: 'Thao Tác',
      userEmail: '用户邮箱',
      kycStatus: 'Trạng Thái KYC',
      country: 'Quốc Gia',
      sanctioned: 'Đã Cấm Vận',
      disclaimer: '免责声明',
      questionnaire: '问卷',
      override: 'Ghi Đè',
      setKYC: 'Đặt KYC',
      setKYCStatus: 'Đặt Trạng Thái KYC',
      grantOverride: 'Cấp Ghi Đè',
      revokeOverride: 'Thu Hồi Ghi Đè',
      filterByKYCStatus: '按KYC状态筛选',
      unverified: 'Chưa Xác Minh',
      pending: 'Đang Chờ',
      verified: 'Đã Xác Minh',
      rejected: 'Đã Từ Chối',
      emptySanctions: 'Không có quốc gia bị cấm vận',
      emptyKYC: 'Không có hồ sơ KYC',
      messages: {
        countryAdded: '国家已添加',
        countryAddFailed: '添加国家失败',
        countryRemoved: '国家已移除',
        countryRemoveFailed: '移除国家失败',
        kycUpdated: 'KYC状态已更新',
        kycUpdateFailed: '更新KYC状态失败',
        overrideUpdated: '豁免状态已更新',
        overrideUpdateFailed: '更新豁免状态失败'
      },
      overrideWarning: '该用户来自受制裁国家，授予豁免将允许交易。',
      confirmGrantOverride: '确认授予该用户豁免权限？',
      confirmRevokeOverride: '确认撤销该用户的豁免权限？'
    },
    userManagement: {
      title: 'Quản Lý Người Dùng',
      addUser: 'Thêm Người Dùng',
      table: {
        id: 'ID',
        email: 'Email',
        nickname: 'Biệt Danh',
        role: 'Vai Trò',
        status: 'Trạng Thái',
        mtAccountCount: 'TK MT',
        createdAt: 'Ngày Tạo',
        actions: 'Thao Tác'
      },
      actions: {
        details: 'Chi Tiết',
        enable: 'Kích Hoạt',
        disable: 'Vô Hiệu',
        changePassword: 'Đổi Mật Khẩu'
      },
      filters: {
        searchPlaceholder: 'Tìm theo email hoặc biệt danh',
        rolePlaceholder: 'Lọc theo vai trò',
        statusPlaceholder: 'Lọc theo trạng thái'
      },
      status: {
        active: 'Hoạt Động',
        suspended: 'Đã Khóa'
      },
      roles: {
        user: 'Người Dùng',
        superAdmin: 'Quản Trị Viên',
        operation: 'Vận Hành',
        customerService: 'CSKH',
        audit: 'Kiểm Toán'
      },
      pagination: {
        total: 'Tổng {{total}} người dùng'
      },
      deleteConfirm: {
        title: 'Xóa người dùng này? Hành động này không thể hoàn tác.',
        batchDeleteConfirm: 'Xóa {{count}} người dùng? Hành động này không thể hoàn tác.',
        batchDeleteSuccess: 'Đã xóa {{count}} người dùng',
        batchDeletePartial: 'Đã xóa {{deleted}}, {{failed}} thất bại',
      },
      modals: {
        createTitle: 'Tạo Người Dùng',
        editTitle: 'Sửa Người Dùng',
        passwordTitle: 'Đổi Mật Khẩu'
      },
      form: {
        email: 'Email',
        nickname: 'Biệt Danh',
        password: 'Mật Khẩu',
        role: 'Vai Trò',
        status: 'Trạng Thái',
        accountNumber: 'Số Tài Khoản',
        accountNumberInvalid: '5-6 chữ số, không có số 0 ở đầu, không có 4 hoặc 7',
        placeholders: {
          email: 'Nhập email',
          nickname: 'Nhập biệt danh',
          password: 'Nhập mật khẩu'
        }
      },
      passwordForm: {
        newPassword: 'Mật Khẩu Mới',
        confirmPassword: 'Xác Nhận Mật Khẩu',
        placeholders: {
          newPassword: 'Nhập mật khẩu mới',
          confirmPassword: 'Nhập lại mật khẩu mới'
        },
        submit: 'Cập Nhật Mật Khẩu',
        validation: {
          newPasswordRequired: 'Vui lòng nhập mật khẩu mới',
          confirmPasswordRequired: 'Vui lòng xác nhận mật khẩu mới',
          passwordMin8: 'Mật khẩu phải có ít nhất 8 ký tự',
          passwordMismatch: 'Mật khẩu không khớp',
          passwordMustContainLettersAndNumbers: 'Mật khẩu phải chứa cả chữ và số'
        }
      },
      messages: {
        userCreatedSuccess: 'Đã tạo người dùng',
        userCreateFailed: 'Tạo người dùng thất bại',
        userUpdatedSuccess: 'Đã cập nhật người dùng',
        userUpdateFailed: 'Cập nhật người dùng thất bại',
        userDeletedSuccess: 'Đã xóa người dùng',
        userDeleteFailed: 'Xóa người dùng thất bại',
        userEnabled: 'Đã kích hoạt người dùng',
        userDisabled: 'Đã vô hiệu người dùng',
        passwordUpdatedSuccess: 'Đã cập nhật mật khẩu',
        passwordUpdateFailed: 'Cập nhật mật khẩu thất bại',
        newPasswordIs: 'Mật khẩu mới là: {{password}}'
      },
      drawer: {
        title: 'Chi Tiết Người Dùng',
        labels: {
          id: 'ID',
          email: 'Email',
          nickname: 'Biệt Danh',
          role: 'Vai Trò',
          status: 'Trạng Thái',
          mtAccountCount: 'TK MT',
          createdAt: 'Ngày Tạo',
          lastLogin: 'Đăng Nhập Cuối'
        }
      }
    },
    wallet: {
      title: 'Quản Lý Ví',
      searchPlaceholder: 'Tìm theo email hoặc số tài khoản...',
      noUsers: 'Không tìm thấy người dùng',
      walletFor: 'Ví của',
      accountNumber: 'Số TK',
      adjustBalance: 'Điều Chỉnh Số Dư',
      adjustSuccess: 'Đã điều chỉnh số dư',
      adjustFailed: 'Điều chỉnh thất bại',
      add: 'Thêm',
      deduct: 'Trừ',
      reason: 'Lý do điều chỉnh...',
    }
  },
  wallet: {
    title: 'Ví Của Tôi',
    accountNumber: 'Số TK',
    table: {
      type: 'Loại',
      amount: 'Số Tiền',
      balanceAfter: 'Số Dư Sau',
      description: 'Mô Tả',
      time: 'Thời Gian',
    },
    balance: 'Số Dư',
    frozen: 'Đóng Băng',
    frozenBalance: 'Đóng Băng',
    currency: 'Tiền Tệ',
    transactions: 'Giao Dịch',
    deposit: 'Nạp Tiền',
    withdraw: 'Rút Tiền',
    history: 'Lịch Sử',
    txType: {
      deposit: 'Nạp Tiền',
      withdrawal: 'Rút Tiền',
      adjustment: 'Điều Chỉnh',
      fee: 'Phí',
      reversal: 'Hoàn Tác',
    },
  },
  symbolDetection: {
    label: 'Biểu tượng được Phát hiện',
    loading: 'Đang phân tích…',
    noSymbols: 'Không phát hiện biểu tượng giao dịch. Thử bao gồm tên biểu tượng cụ thể (ví dụ: "Bitcoin", "EURUSD", "Vàng").',
    unresolvedTooltip: 'Chưa liên kết tài khoản giao dịch, không thể phân giải',
    resolvedTooltip: 'môi giới: {{broker}} | chế độ: {{mode}}',
    tradeMode: {
      disabled: 'Đã Tắt',
      longOnly: 'Chỉ Mua',
      shortOnly: 'Chỉ Bán',
      longShort: 'Mua & Bán',
      unknown: 'Không Xác Định'
    }
  },
  autoTrading: {
    title: 'Giao Dịch Tự Động',
    status: {
      enabled: 'Giao Dịch Tự Động Đã Bật',
      disabled: 'Giao Dịch Tự Động Đã Tắt',
      activeStrategies: 'Chiến Lược Đang Hoạt Động',
      todayExecutions: 'Thực Thi Hôm Nay',
      todayProfit: 'Lợi Nhuận Hôm Nay'
    },
    settings: {
      title: 'Cài Đặt Rủi Ro Toàn Cục',
      maxRiskPercent: 'Rủi Ro Tối Đa %',
      maxPositions: 'Vị Thế Tối Đa',
      maxLotSize: 'Lot Tối Đa',
      maxDailyLoss: 'Lỗ Tối Đa Hàng Ngày',
      maxDrawdownPercent: 'Sụt Giảm Tối Đa %',
      saveSuccess: 'Đã lưu cài đặt',
      saveFailed: 'Lưu cài đặt thất bại'
    },
    logs: {
      title: 'Nhật Ký Giao Dịch Gần Đây',
      empty: 'Chưa có nhật ký giao dịch',
      columns: {
        time: 'Thời Gian',
        symbol: 'Mã',
        action: 'Hành Động',
        volume: 'Khối Lượng',
        price: 'Giá',
        profit: 'Lãi/Lỗ',
        ticket: 'Ticket'
      }
    },
    messages: {
      loadFailed: 'Tải dữ liệu giao dịch tự động thất bại',
      toggleFailed: 'Chuyển đổi giao dịch tự động thất bại'
    }
  }
} as const;

export default base;
