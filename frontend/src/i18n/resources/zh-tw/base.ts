const base = {
  app: {
    name: 'AntTrader'
  },
  auth: {
    fields: {
      email: '信箱',
      password: '密碼',
      confirmPassword: '確認密碼'
    },
    messages: {
      loginSuccess: '登入成功',
      loginFailed: '登入失敗，請檢查信箱與密碼',
      registerSuccess: '註冊成功，請登入',
      registerFailed: '註冊失敗，請稍後重試',
      logoutSuccess: '已登出',
      fetchMeFailed: '取得使用者資訊失敗'
    },
    validation: {
      emailRequired: '請輸入信箱',
      emailInvalid: '請輸入有效的信箱',
      passwordRequired: '請輸入密碼',
      passwordMin8: '密碼至少 8 碼',
      confirmPasswordRequired: '請確認密碼',
      passwordMismatch: '兩次輸入的密碼不一致'
    },
    login: {
      subtitle: '這是一個測試不具備責任能力',
      rememberMe: '記住我',
      forgotPassword: '忘記密碼？',
      signingIn: '登入中...',
      login: '登入',
      noAccount: '還沒有帳號？',
      registerNow: '立即註冊'
    },
    register: {
      subtitle: '建立新帳號',
      signingUp: '註冊中...',
      register: '註冊',
      haveAccount: '已有帳號？',
      loginNow: '立即登入'
    },
    forgotPassword: {
      title: '重設密碼',
      hint: '請聯繫管理員或支援人員重設密碼。',
      backToLogin: '返回登入'
    }
  },
  common: {
    refresh: '刷新',
    create: '新增',
    back: '返回',
    updated: '已更新',
    created: '已建立',
    enabled: '已啟用',
    disabled: '已停用',
    deleted: '已刪除',
    deleteFailed: '刪除失敗',
  deleteSelected: '刪除選中 ({{count}})',
    loadingFailed: '載入失敗',
    none: '無',
    close: '關閉',
    operationFailed: '操作失敗',
    pleaseWait: '請稍候...',
    next: '下一步',
    previous: '上一步',
    gotIt: '我知道了',
    loading: '載入中...',
    searching: '搜尋中...',
    unknown: '未知',
    enable: '啟用',
    disable: '停用',
    edit: '編輯',
    delete: '刪除',
    confirm: '確定',
    cancel: '取消',
    save: '保存',
    send: '發送',
    saveFailed: '保存失敗',
    showDetails: '查看詳情',
    hideDetails: '收起詳情',
    translate: '翻譯',
    viewOriginal: '查看原文',
    viewTranslation: '查看譯文',
    copy: '複製',
    copied: '已複製',
    copyFailed: '複製失敗',
    totalItems: '共 {{count}} 項',
    time: {
      minute: '{{n}}分',
      hour: '{{n}}時',
      day: '{{n}}天',
      lessThanMinute: '<1分'
    },
    required: '必填',
    noData: '尚無資料',
    ok: '確定',
    error: '錯誤',
    retry: '重試',
    pageError: '頁面錯誤',
    unexpectedError: '發生意外錯誤',
    lineColor: '線顏色',
    selectSymbolToViewChart: '選擇品種查看圖表',
    currentPosition: '📊 當前持倉',
    noOpenPositionsForSymbol: '{{symbol}} 暫無持倉',
    indicatorSettings: '{{name}} 設定',
    active: '啟用',
    inactive: '停用',
    clear: '清除',
    saveSuccess: '儲存成功',
    remove: '移除',
    yes: '是',
    no: '否',
    you: '你',
    comingSoon: '即將上線',
    pageUnderDevelopment: '此页面正在開发中'
  },
  language: {
    simplifiedChinese: '简体中文',
    traditionalChinese: '繁體中文',
    english: 'English',
    japanese: '日本語',
    vietnamese: 'Tiếng Việt'
  },
  menu: {
    strategyWorkspace: '策略工作台',
    dashboard: '儀表板',
    strategy: '策略',
    accounts: '帳戶管理',
    aiAssistant: 'AI助手',
    strategies: '策略管理',
    trading: '交易',
    wallet: '錢包',
    algoDashboard: '演算法看板',
    market: '行情',
    analytics: '分析',
    marketplace: '市場',
    experiments: '策略實驗',
    marketRegime: '市場狀態',
    assets: '策略資產',
    schedules: '策略調度',
    indicatorCatalog: '指標目錄',
    logs: '系統日誌',
    assetAnalysis: 'AI 分析',
    autoTrading: '自動交易',
    marketTools: '市場分析工具',
    devGroup: '策略開發',
    opsGroup: '策略營運',
  },
  market: {
    searchPlaceholder: '搜尋品種（如 EURUSD, XAUUSD）',
    selectAccount: '選擇交易賬戶',
    watchlist: '自選',
    popularSymbols: '熱門品種',
    noSymbolSelected: '選擇一個品種以檢視行情數據',
    bid: '買價',
    ask: '賣價',
    spread: '點差',
    mid: '中間價',
    allSymbols: '全部品種',
    common: '常用',
    selectSymbol: '選擇品種',
    noSymbolsFound: '找不到品種',
    loadingSymbols: '載入中...',
    emptyWatchlist: '暫無自選',
    searchSymbol: '搜尋商品...'
  },
  topbar: {
    systemOk: '系統正常運行',
    profile: '個人資訊',
    settings: '設定',
    switchToAdmin: '切換到管理',
    logout: '退出登入',
    user: '用戶'
  },
  profile: {
    title: '個人資訊',
    nickname: '暱稱',
    role: '角色',
    status: '狀態',
    lastLogin: '最後登入',
    registered: '註冊時間'
  },
  notifications: {
    title: '通知中心',
    empty: '暫無通知',
    types: {
      trade: '交易',
      signal: '訊號',
      risk_alert: '風險',
      strategy_execution: '策略',
      system: '系統'
    },
    tabs: {
      all: '全部 ({{count}})',
      unread: '未讀 ({{count}})'
    },
    actions: {
      markAllAsRead: '全部已讀',
      clearAll: '清空',
      clearAllConfirm: '確定清空所有通知？'
    },
    all: '全部',
    unread: '未讀',
    markAllRead: '全部已讀',
    clearAll: '清除全部',
    confirmClearAll: '確定清除所有通知？',
    stream: {
      strategyExecution: {
        title: '策略執行',
        completed: '{{symbol}} {{action}} 已完成',
        failed: '執行失敗：{{error}}'
      },
      riskAlert: {
        title: '風險警示',
        fallback: '警示類型：{{alertType}}'
      },
      strategySignal: {
        title: '策略訊號',
        message: '{{symbol}} 觸發 {{signalType}}'
      },
      autoTrading: {
        title: '自動交易',
        fallback: '自動交易事件已觸發'
      }
    }
  },
  errors: {
    not_authenticated: '未登入',
    invalid_credentials: '帳號或密碼錯誤',
    user_not_found: '使用者不存在',
    email_already_registered: '信箱已註冊',
    account_not_found: '帳戶不存在',
    access_denied: '無權限存取',
    account_connection_failed: '無法連線到交易伺服器',
    account_connected: '連線成功',
    schedule_service_not_available: '排程服務不可用',
    auto_trading_enabled: '自動交易已開啟',
    auto_trading_disabled: '自動交易已關閉',
    connection_failed: {
      title: '連線失敗',
      content: '無法連線到後端服務，請確認伺服器是否已啟動。'
    },
    ai: {
      not_configured: 'AI 未配置：請先到 AI 設定中啟用並配置。',
      config_service_not_initialized: 'AI 設定服務未初始化',
      config_valid: 'AI 設定有效',
      no_trade_data_available: '暫無可用交易資料',
      provider_returned_empty_message: 'AI 服務回傳空訊息',
      provider_required: '請先選擇服務商',
      invalid_provider: '服務商無效',
      api_key_required: 'API Key 不能為空',
      base_url_required: 'Base URL 不能為空',
      invalid_base_url: 'Base URL 無效',
      base_url_scheme_invalid: 'Base URL 必須以 http:// 或 https:// 開頭',
      base_url_should_not_end_with_chat_completions: 'Base URL 不應以 /chat/completions 結尾',
      failed_to_create_request: '建立請求失敗',
      request_failed: 'API 請求失敗',
      probe_ok: '正常',
      probe_ok_no_models: '正常（未回傳 models）',
      free_tier_exhausted: 'AI 模型免費額度已耗盡：請在模型供應商管理後台關閉「use free tier only」或更換付費 Key。',
      rate_limited: 'AI 服務觸發限流/額度不足（429/資源耗盡）。請稍後重試或更換可用的 API Key/模型配置。',
      forbidden_quota: 'AI 服務額度/權限不足（403）。請檢查 API Key 是否有可用額度或是否已啟用付費模式。',
      wizard: {
        title: 'AI 策略嚮導',
        subtitle: '每步一個頁面，可前進/後退',
        currentModel: '目前模型：{{model}}',
        steps: {
          setup: '基礎資訊',
          generate: '生成策略',
          publishCode: '回測上線-程式碼',
          publishBacktest: '回測上線-回測',
          publishLaunch: '回測上線-上線'
        },
        actions: {
          prev: '上一步',
          next: '下一步',
          cancel: '取消'
        },
        agents: {
          styleTitle: '市場狀態/風格推薦',
          signalsTitle: '訊號與指標設計',
          riskTitle: '風控與執行約束',
          codeTitle: '程式碼生成'
        },
        template: {
          defaultName: 'AI 策略 {{title}}',
          defaultDescription: 'AI 嚮導生成'
        },
        schedule: {
          defaultName: 'AI 調度 {{symbol}} {{timeframe}}'
        },
        prompts: {
          dataSpec: {
            dataset: '使用凍結資料集 datasetId={{datasetId}}',
            klineRange: '使用歷史K線範圍 from={{from}} to={{to}}'
          },
          base: {
            account: '帳號: {{accountId}}',
            symbol: '品種: {{symbol}}',
            timeframe: '週期: {{timeframe}}',
            data: '資料: {{dataSpec}}',
            constraints: '約束: 最大回撤={{maxDrawdownPct}}% 單筆風險={{riskPerTradePct}}% 日內最多交易={{maxTradesPerDay}} 次',
            params: `參數（定義+目前值；執行時在 context["params"] 中）：
{{params}}`,
            empty: '(空)',
            macroEnabled: `宏觀事件(使用者提供):
{{text}}`,
            macroDisabled: '宏觀事件: 不使用',
            userIntent: `使用者策略目標(自然語言):
{{intent}}`
          },
          upstream: {
            style: `【市場狀態/風格推薦 結論】
{{text}}`,
            signals: `【訊號與指標設計 結論】
{{text}}`,
            risk: `【風控與執行約束 結論】
{{text}}`,
            sectionTitle: '【上游 Agent 結論（原樣提供）】'
          },
          summary: {
            intro: '你是量化策略解釋助手。請用精簡中文（要點形式，最多 12 行）解釋下面這段 AntTrader Python 策略程式碼的核心思路，幫助使用者判斷是否符合預期。',
            mustIncludeTitle: '必須包含：',
            mustInclude1: '1) 策略類型/範式（趨勢/均值/突破/動量/網格等，若無法判斷則寫「無法確定」）',
            mustInclude2: '2) 主要入場條件（2~4 點）',
            mustInclude3: '3) 主要出場/止損止盈/風控約束（2~4 點）',
            mustInclude4: '4) 適用/不適用場景各 1 點',
            userIntent: `使用者預期（自然語言）：
{{intent}}`,
            codeTitle: '程式碼如下：'
          }
        },
        messages: {
          generateCodeFirst: '請先生成策略程式碼',
          validateCodeFirst: '請先點擊「驗證程式碼」',
          codeInvalidFixAndContinue: '程式碼驗證未通過，請修復後再繼續',
          startBacktestFirst: '請先點擊「回測（非同步任務）」啟動回測',
          backtestNotDoneWait: '回測尚未完成，請等待評分卡狀態變為「成功/失敗/已取消」後再繼續',
          confirmScoreFirst: '請先在評分彈窗中確認評分結果',
          fillRequiredWithFields: '請先補全必填項：{{fields}}',
          fillRequired: '請先補全必填項',
          watchBacktestRunFailed: 'watchBacktestRun 失敗',
          createDraftFailed: '建立草稿失敗',
          loadAccountsFailed: '載入帳號失敗',
          loadSymbolsFailed: '載入品種失敗',
          loadDatasetFailed: '載入 dataset 失敗',
          datasetFrozenCreated: '已凍結建立 dataset',
          freezeDatasetFailed: '凍結 dataset 失敗',
          inputIntentFirst: '請先輸入策略目標/想法',
          aiRequestTimeout: 'AI 請求逾時（>{{seconds}}s）',
          modelReturnedEmpty: '模型回傳為空',
          noPythonCodeBlock: '程式碼 Agent 未輸出 \`\`\`python 代碼塊\`\`\`，請在結果中檢查',
          agentFailed: '{{title}} 失敗',
          userAborted: '使用者已中止',
          chatAborted: '已中止與模型對話',
          noCodeToValidate: '暫無程式碼可驗證',
          validateOk: '驗證通過',
          validateFailed: '驗證未通過',
          validateError: '驗證失敗',
          noCodeToBacktest: '暫無程式碼可回測',
          backtestCreated: '回測任務已建立',
          createBacktestFailed: '建立回測失敗',
          draftNotCreated: '草稿未建立',
          draftSaved: '草稿已保存',
          saveFailed: '保存失敗',
          publishedNoId: '已發布，但未拿到回傳 id（請在策略管理中確認）',
          templatePublished: '模板已發布',
          publishFailed: '發布失敗',
          publishTemplateFirst: '請先發布模板',
          scheduleCreatedAndEnabled: '調度已建立並啟用',
          scheduleCreated: '調度已建立',
          createScheduleFailed: '建立調度失敗',
          scheduleAlreadyExists: '該帳號下已存在相同策略調度（模板+品種+週期相同），請勿重複建立。'
        }
      }
    },
    translate_failed: '翻譯失敗'
  },
  marketplace: {
    title: '策略市集',
    subtitle: '探索、評分與訂閱社群策略',
    publish: '發布策略',
    tabs: {
      marketplace: '策略市場',
      subscriptions: '我的訂閱'
    },
    searchPlaceholder: '搜尋策略...',
    filterByClass: '依資產類別篩選',
    sort: {
      newest: '最新',
      popular: '最熱門',
      performance: '最佳表現'
    },
    empty: '尚無已發布策略',
    noSubscriptions: '尚無訂閱',
    card: {
      subscribe: '訂閱',
      subscribed: '已訂閱',
      unsubscribe: '取消訂閱',
      unsubscribeHint: '點擊取消訂閱',
      details: '詳細',
      subscribers: '訂閱者',
      winRate: '勝率',
      by: '發布者'
    },
    assetClass: {
      forex: '外匯',
      crypto: '加密貨幣',
      commodity: '大宗商品',
      index: '指數',
      stock: '股票',
      other: '其他'
    },
    risk: {
      low: '低',
      medium: '中',
      high: '高'
    },
    messages: {
      loginFirst: '請先登入',
      subscribed: '訂閱成功',
      subscribeFailed: '訂閱失敗',
      unsubscribed: '已取消訂閱',
      unsubscribeFailed: '取消訂閱失敗',
      rated: '評分已提交',
      rateFailed: '评分失敗',
      commentPosted: '評論已發布',
      commentFailed: '评论失敗',
      publishFailed: '發布失敗',
      published: '發布成功'
    },
    detail: {
      comments: '評論',
      noComments: '尚無評論，快來搶頭香！',
      commentPlaceholder: '撰寫評論...（Shift+Enter 換行）'
    },
    publishModal: {
      symbolsPlaceholder: 'EURUSD, GBPUSD, XAUUSD',
      strategyId: '策略ID',
      title: '發布策略',
      titleField: '標題',
      titlePlaceholder: '輸入策略標題',
      description: '描述',
      assetClass: '资产類別',
      riskLevel: '風險等級',
      priceModel: '價格模式',
      priceAmount: '價格',
      symbols: '交易商品',
      tags: '標籤',
      timeframe: '時間週期',
      submit: '發布'
    },
    priceModel: {
      free: '免費',
      subscription: '訂閱制',
      performanceFee: '績效分成'
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
      configItem: '配置項目',
      value: '值',
      description: '描述',
      status: '狀態',
      toggle: '切换',
      updatedAt: '更新時間',
      on: '開',
      off: '關',
      maxAccountsPerUser: '每用戶最大账戶數',
      aiProviderCatalog: 'AI提供者目錄',
      econAIConfig: '經濟日曆AI配置',
      strategyHealthConfig: '策略健康度配置',
      provider: '提供者',
      modelName: '模型名稱',
      enableToggle: '啟用',
      baseUrlLabel: 'Base URL',
      formatJson: '格式化JSON',
      fillTemplate: '填入範本',
      thresholdInfo: '閾值說明',
      thresholdDesc: '閾值描述',
      validation: {
        jsonEmpty: 'JSON不能為空白',
        jsonInvalid: 'JSON格式無效',
        greenSuccessRateRange: '绿色成功率需在0-100之間',
        yellowSuccessRateRange: '黄色成功率需在0-100之間',
        yellowNotGreaterThanGreen: '黄色閾值不能超過绿色閾值',
        greenMaxFailedRunsNonNegative: '绿色最大失敗次數需≥0',
        minSampleSizeNonNegative: '最小样本量需≥0',
        apiKeyRequired: 'API Key不能為空白',
        modelRequired: '模型名稱不能為空白'
      },
      messages: {
        loadFailed: '加载配置失敗',
        updated: '配置已更新',
        updateFailed: '更新配置失敗',
        enabled: '已啟用',
        disabled: '已停用',
        operationFailed: '操作失敗'
      },
      placeholders: {
        json: '輸入JSON',
        apiKey: '輸入API Key',
        model: '輸入模型名稱',
        baseUrl: '輸入Base URL',
        configValue: '輸入配置值',
        description: '輸入描述'
      },
      providerOptions: {
        zhipu: '智谱AI',
        deepseek: 'DeepSeek',
        custom: '自訂'
      }
    },
    trading: {
      title: '交易監控',
      loadFailed: 'Failed to load trading statistics',
      platform: '平台',
      accounts: '账戶',
      orders: '訂單',
      volume: '交易量',
      byPlatform: '按平台',
      profitStats: '盈虧統計',
      totalUsers: '總用戶數',
      activeUsers: '活躍用戶',
      totalAccounts: '總账戶數',
      connectedAccounts: '已连接',
      totalOrders: '總訂單',
      closedOrders: '已平倉',
      totalVolume: '總交易量',
      netProfit: '淨利潤',
      totalProfit: '總盈利',
      totalLoss: '總虧損',
      pendingOrders: '掛單'
    },
    dashboard: {
      title: '管理儀表板',
      loadFailed: '加载儀表板數据失敗',
      totalUsers: '總用戶數',
      activeUsers: '活躍用戶',
      mtAccounts: 'MT账戶',
      onlineAccounts: '在線账戶',
      todayTrades: '今日交易',
      todayProfit: '今日盈虧',
      recentLogs: '最近日誌',
      logs: {
        time: '時間',
        module: '模組',
        actionType: '操作',
        target: '目標',
        status: '狀態',
        success: '成功',
        failed: '失敗',
        moduleMap: {
          userManagement: '用戶管理',
          accountManagement: '账戶管理',
          trading: '交易',
          systemConfig: '系统配置'
        }
      },
      riskMetrics: {
        title: '風控指標',
        riskValidateTotal: '總驗證數',
        riskValidatePass: '通過',
        riskValidateReject: '拒絕',
        riskValidateError: '錯誤',
        orderSendSuccess: '下單成功',
        orderSendFailed: '下單失敗',
        orderCloseSuccess: '平倉成功',
        orderCloseFailed: '平倉失敗'
      },
      riskWindow: {
        title: '風控窗口',
        validateTotal: '總计',
        validatePass: '通過',
        validateReject: '拒絕',
        validateError: '錯誤',
        orderSendSuccess: '下單成功',
        orderSendFailed: '下單失敗',
        orderCloseSuccess: '平倉成功',
        orderCloseFailed: '平倉失敗',
        rejectRiskCodesHeader: '風控代碼',
        rejectCount: '拒絕次數',
        noRejectData: '此時段无拒絕記錄',
        noData: '暫無風控數据'
      }
    },
    jurisdiction: {
      title: '管轄权管理',
      sanctionedCountriesTab: '制裁國家',
      kycStatusTab: 'KYC狀態',
      sanctionedCountries: '制裁國家',
      userKYCStatus: '用戶KYC狀態',
      addCountry: '新增國家',
      addSanctionedCountry: '新增制裁國家',
      countryCode: '國家代碼',
      countryLabel: '國家',
      addedBy: '新增人',
      actions: '操作',
      userEmail: '用戶電子郵件',
      kycStatus: 'KYC狀態',
      country: '國家',
      sanctioned: '已制裁',
      disclaimer: '免責聲明',
      questionnaire: '問卷',
      override: '豁免',
      setKYC: '設定KYC',
      setKYCStatus: '設定KYC狀態',
      grantOverride: '授予豁免',
      revokeOverride: '撤銷豁免',
      filterByKYCStatus: '按KYC狀態篩選',
      unverified: '未驗證',
      pending: '待審核',
      verified: '已驗證',
      rejected: '已拒絕',
      emptySanctions: '暫無制裁國家',
      emptyKYC: '暫無KYC記錄',
      messages: {
        countryAdded: '國家已新增',
        countryAddFailed: '新增國家失敗',
        countryRemoved: '國家已移除',
        countryRemoveFailed: '移除國家失敗',
        kycUpdated: 'KYC狀態已更新',
        kycUpdateFailed: '更新KYC狀態失敗',
        overrideUpdated: '豁免狀態已更新',
        overrideUpdateFailed: '更新豁免狀態失敗'
      },
      overrideWarning: '该用戶來自受制裁國家，授予豁免將允許交易。',
      confirmGrantOverride: '確認授予该用戶豁免權限？',
      confirmRevokeOverride: '確認撤銷该用戶的豁免權限？'
    },
    userManagement: {
      title: '用戶管理',
      addUser: '新建用戶',
      table: {
        id: 'ID',
        email: '電子郵件',
        nickname: '暱稱',
        role: '角色',
        status: '狀態',
        mtAccountCount: 'MT账戶數',
        createdAt: '建立時間',
        actions: '操作'
      },
      actions: {
        details: '詳細',
        enable: '啟用',
        disable: '停用',
        changePassword: '修改密碼'
      },
      filters: {
        searchPlaceholder: '搜尋電子郵件或暱稱',
        rolePlaceholder: '按角色篩選',
        statusPlaceholder: '按狀態篩選'
      },
      status: {
        active: '正常',
        suspended: '已停用'
      },
      roles: {
        user: '一般用戶',
        superAdmin: '超級管理員',
        operation: '營運',
        customerService: '客服',
        audit: '稽核'
      },
      pagination: {
        total: '共 {{total}} 個用戶'
      },
      deleteConfirm: {
        title: '確認刪除此用戶？此操作不可復原。',
        batchDeleteConfirm: '確認刪除 {{count}} 個用戶？此操作不可復原。',
        batchDeleteSuccess: '已刪除 {{count}} 個用戶',
        batchDeletePartial: '已刪除 {{deleted}} 個，{{failed}} 個失敗',
      },
      modals: {
        createTitle: '新建用戶',
        editTitle: '編輯用戶',
        passwordTitle: '修改密碼'
      },
      form: {
        email: '電子郵件',
        nickname: '暱稱',
        password: '密碼',
        role: '角色',
        status: '狀態',
        accountNumber: '錢包號',
        accountNumberInvalid: '5-6位數字，無前導零，不含4和7',
        placeholders: {
          email: '輸入電子郵件',
          nickname: '輸入暱稱',
          password: '輸入密碼'
        }
      },
      passwordForm: {
        newPassword: '新密碼',
        confirmPassword: '確認密碼',
        placeholders: {
          newPassword: '輸入新密碼',
          confirmPassword: '再次輸入新密碼'
        },
        submit: '更新密碼',
        validation: {
          newPasswordRequired: '请輸入新密碼',
          confirmPasswordRequired: '请確認新密碼',
          passwordMin8: '密碼至少8位元',
          passwordMismatch: '两次密碼不一致',
          passwordMustContainLettersAndNumbers: '密碼需包含字母和數字'
        }
      },
      messages: {
        userCreatedSuccess: '用戶建立成功',
        userCreateFailed: '建立用戶失敗',
        userUpdatedSuccess: '用戶更新成功',
        userUpdateFailed: '更新用戶失敗',
        userDeletedSuccess: '用戶已刪除',
        userDeleteFailed: '刪除用戶失敗',
        userEnabled: '用戶已啟用',
        userDisabled: '用戶已停用',
        passwordUpdatedSuccess: '密碼更新成功',
        passwordUpdateFailed: '密碼更新失敗',
        newPasswordIs: '新密碼为: {{password}}'
      },
      drawer: {
        title: '用戶詳細',
        labels: {
          id: 'ID',
          email: '電子郵件',
          nickname: '暱稱',
          role: '角色',
          status: '狀態',
          mtAccountCount: 'MT账戶數',
          createdAt: '建立時間',
          lastLogin: '最后登入'
        }
      }
    },
    wallet: {
      title: '錢包管理',
      searchPlaceholder: '搜尋郵箱或錢包號...',
      noUsers: '未找到用戶',
      walletFor: '錢包 -',
      accountNumber: '錢包號',
      adjustBalance: '調整餘額',
      adjustSuccess: '餘額已調整',
      adjustFailed: '調整失敗',
      add: '增加',
      deduct: '扣除',
      reason: '調整原因...',
    }
  },
  wallet: {
    title: '我的錢包',
    accountNumber: '錢包號',
    table: {
      type: '類型',
      amount: '金額',
      balanceAfter: '調整後餘額',
      description: '描述',
      time: '時間',
    },
    balance: '餘額',
    frozen: '凍結',
    frozenBalance: '凍結',
    currency: '幣種',
    transactions: '交易記錄',
    viewDetails: '查看詳情',
  },
  symbolDetection: {
    label: '偵測到的交易品種',
    loading: '解析中…',
    noSymbols: '未偵測到交易品種。請嘗試包含具體品種名稱（如「比特幣」、「EURUSD」、「黃金」）。',
    unresolvedTooltip: '尚未綁定交易帳戶，無法解析',
    resolvedTooltip: '券商：{{broker}} | 模式：{{mode}}',
    tradeMode: {
      disabled: '已停用',
      longOnly: '僅做多',
      shortOnly: '僅做空',
      longShort: '多空雙向',
      unknown: '未知'
    }
  },
  autoTrading: {
    title: '自動交易',
    status: {
      enabled: '自動交易已開啟',
      disabled: '自動交易已關閉',
      activeStrategies: '活躍策略',
      todayExecutions: '今日執行',
      todayProfit: '今日盈虧'
    },
    settings: {
      title: '全域風控設定',
      maxRiskPercent: '最大風險%',
      maxRiskPercentHint: '每筆交易風險佔餘額百分比',
      maxPositions: '最大持倉數',
      maxPositionsHint: '同時持有的最大倉位數量',
      maxLotSize: '最大手數',
      maxLotSizeHint: '每筆交易最大交易量(手)',
      maxDailyLoss: '每日最大虧損',
      maxDailyLossHint: '日虧損超過此值時自動停止交易',
      maxDrawdownPercent: '最大回撤%',
      maxDrawdownPercentHint: '回撤超過此值時自動停止交易',
      saveSuccess: '設定已儲存',
      saveFailed: '儲存設定失敗'
    },
    logs: {
      title: '最近交易日誌',
      empty: '暫無交易日誌',
      columns: {
        time: '時間',
        symbol: '商品',
        action: '操作',
        volume: '數量',
        price: '價格',
        profit: '盈虧',
        ticket: '單號'
      }
    },
    messages: {
      loadFailed: '載入自動交易資料失敗',
      toggleFailed: '切換自動交易失敗'
    }
  }
} as const;

export default base;
