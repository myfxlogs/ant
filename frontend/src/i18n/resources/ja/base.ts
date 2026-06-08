const base = {
  app: {
    name: 'AntTrader'
  },
  auth: {
    fields: {
      email: 'メール',
      password: 'パスワード',
      confirmPassword: 'パスワード確認'
    },
    messages: {
      loginSuccess: 'ログインしました',
      loginFailed: 'ログインに失敗しました。メールアドレスとパスワードを確認してください。',
      registerSuccess: '登録が完了しました。ログインしてください。',
      registerFailed: '登録に失敗しました。しばらくしてから再試行してください。',
      logoutSuccess: 'ログアウトしました',
      fetchMeFailed: 'ユーザー情報の取得に失敗しました'
    },
    validation: {
      emailRequired: 'メールアドレスを入力してください',
      emailInvalid: '有効なメールアドレスを入力してください',
      passwordRequired: 'パスワードを入力してください',
      passwordMin8: 'パスワードは8文字以上である必要があります',
      confirmPasswordRequired: 'パスワードを確認してください',
      passwordMismatch: 'パスワードが一致しません'
    },
    login: {
      subtitle: '本サービスはテストであり責任を負いません',
      rememberMe: 'ログイン状態を保持',
      forgotPassword: 'パスワードをお忘れですか？',
      signingIn: 'ログイン中...',
      login: 'ログイン',
      noAccount: 'アカウントをお持ちでないですか？',
      registerNow: '新規登録',
      agreePrefix: 'ログインすることで、以下に同意したものとみなされます：',
      terms: '利用規約',
      and: 'および',
      privacy: 'プライバシーポリシー'
    },
    register: {
      subtitle: '新規アカウント作成',
      signingUp: '登録中...',
      register: '登録',
      haveAccount: 'すでにアカウントをお持ちですか？',
      loginNow: 'ログイン',
      agreePrefix: '登録することで、以下に同意したものとみなされます：',
      terms: '利用規約',
      and: 'および',
      privacy: 'プライバシーポリシー'
    },
    forgotPassword: {
      title: 'パスワードリセット',
      hint: '管理者またはサポートに連絡してパスワードをリセットしてください。',
      backToLogin: 'ログインに戻る'
    }
  },
  common: {
    refresh: '更新',
    create: '新規',
    back: '戻る',
    updated: '更新しました',
    created: '作成しました',
    enabled: '有効化しました',
    disabled: '無効化しました',
    deleted: '削除しました',
    deleteFailed: '削除に失敗しました',
    loadingFailed: '読み込みに失敗しました',
    none: 'なし',
    close: '閉じる',
    operationFailed: '操作に失敗しました',
    pleaseWait: 'しばらくお待ちください...',
    next: '次へ',
    previous: '戻る',
    gotIt: '了解',
    loading: '読み込み中...',
    unknown: '不明',
    enable: '有効化',
    disable: '無効化',
    edit: '編集',
    delete: '削除',
    confirm: '確定',
    cancel: 'キャンセル',
    save: '保存',
    send: '送信',
    saveFailed: '保存に失敗しました',
    showDetails: '詳細を表示',
    hideDetails: '詳細を隠す',
    translate: '翻訳',
    viewOriginal: '原文を見る',
    viewTranslation: '翻訳を見る',
    copy: 'コピー',
    copied: 'コピーしました',
    copyFailed: 'コピーに失敗しました',
    noData: 'データがありません',
    totalItems: '合計 {{total}} 件',
    time: {
      minute: '分',
      hour: '時間',
      day: '日',
      lessThanMinute: '< 1分'
    },
    required: '必須'
  },
  language: {
    simplifiedChinese: '简体中文',
    traditionalChinese: '繁體中文',
    english: 'English',
    japanese: '日本語',
    vietnamese: 'Tiếng Việt'
  },
  menu: {
    strategyWorkspace: '戦略ワークスペース',
    dashboard: 'ダッシュボード',
    accounts: '口座管理',
    aiAssistant: 'AIアシスタント',
    strategies: '戦略管理',
    trading: '取引',
    market: 'マーケット',
    analytics: '分析',
    marketplace: 'マーケットプレイス',
    experiments: '戦略実験',
    marketRegime: 'マーケットレジーム',
    assets: '戦略資産',
    schedules: '戦略スケジュール',
    indicatorCatalog: 'インジケーターカタログ',
    logs: 'システムログ',
    assetAnalysis: 'AI分析'
  },
  market: {
    searchPlaceholder: '銘柄を検索（例: EURUSD, XAUUSD）',
    selectAccount: '取引口座を選択',
    watchlist: 'ウォッチリスト',
    popularSymbols: '人気銘柄',
    noSymbolSelected: '銘柄を選択してマーケットデータを表示',
    bid: '買値',
    ask: '売値',
    spread: 'スプレッド',
    mid: '仲値'
  },
  topbar: {
    systemOk: 'システムは正常に稼働中',
    profile: 'プロフィール',
    settings: '設定',
    switchToAdmin: '管理画面へ切替',
    logout: 'ログアウト',
    user: 'ユーザー'
  },
  profile: {
    title: 'プロフィール',
    nickname: 'ニックネーム',
    role: 'ロール',
    status: 'ステータス',
    lastLogin: '最終ログイン',
    registered: '登録日時'
  },
  notifications: {
    title: '通知',
    empty: '通知はありません',
    tabs: {
      all: 'すべて',
      unread: '未読'
    },
    types: {
      trade: '取引',
      signal: 'シグナル',
      risk_alert: 'リスク',
      strategy_execution: '戦略',
      system: 'システム'
    },
    actions: {
      markAllAsRead: 'すべて既読',
      clearAll: 'クリア',
      clearAllConfirm: 'すべての通知を削除しますか？'
    },
    stream: {
      strategyExecution: {
        title: '戦略実行',
        completed: '実行完了',
        failed: '実行失敗'
      },
      riskAlert: {
        title: 'リスクアラート',
        fallback: 'リスクアラートが発生しました'
      },
      strategySignal: {
        title: '戦略シグナル',
        message: '新しい戦略シグナルを受信しました'
      },
      autoTrading: {
        title: '自動取引',
        fallback: '自動取引ステータスが更新されました'
      }
    },
    all: 'すべて',
    unread: '未読',
    markAllRead: 'すべて既読にする',
    clearAll: 'すべて消去',
    confirmClearAll: 'すべての通知を消去しますか？'
  },
  errors: {
    not_authenticated: '認証されていません',
    invalid_credentials: '認証情報が正しくありません',
    user_not_found: 'ユーザーが見つかりません',
    email_already_registered: 'このメールアドレスは既に登録されています',
    account_not_found: '口座が見つかりません',
    access_denied: 'アクセスが拒否されました',
    account_connection_failed: '取引サーバーへの接続に失敗しました',
    account_connected: '接続しました',
    schedule_service_not_available: 'スケジュールサービスは利用できません',
    auto_trading_enabled: '自動売買を有効にしました',
    auto_trading_disabled: '自動売買を無効にしました',
    connection_failed: {
      title: '接続に失敗しました',
      content: 'バックエンドに接続できません。サーバーが起動しているか確認してください。'
    },
    ai: {
      not_configured: 'AI が設定されていません。先に AI 設定で有効化・設定してください。',
      config_service_not_initialized: 'AI 設定サービスが初期化されていません',
      config_valid: 'AI 設定は有効です',
      no_trade_data_available: '利用可能な取引データがありません',
      provider_returned_empty_message: 'AI プロバイダが空のメッセージを返しました',
      provider_required: 'プロバイダを選択してください',
      invalid_provider: '無効なプロバイダです',
      api_key_required: 'API Key は必須です',
      base_url_required: 'Base URL は必須です',
      invalid_base_url: 'Base URL が無効です',
      base_url_scheme_invalid: 'Base URL は http:// または https:// で始まる必要があります',
      base_url_should_not_end_with_chat_completions: 'Base URL は /chat/completions で終わらないようにしてください',
      failed_to_create_request: 'リクエストの作成に失敗しました',
      request_failed: 'API リクエストに失敗しました',
      probe_ok: 'OK',
      probe_ok_no_models: 'OK（model が返されませんでした）',
      free_tier_exhausted: 'AI の無料枠が上限に達しました。プロバイダー管理画面で「無料枠のみ使用」を無効化するか、有料キーに切り替えてください。',
      rate_limited: 'AI サービスがレート制限/クォータ不足（429/資源枯渇）。しばらく待つか、利用可能な API Key/model に切り替えてください。',
      forbidden_quota: 'AI サービスのクォータ/権限が不足しています（403）。API Key の残高や課金状態を確認してください。'
    },
    translate_failed: '翻訳に失敗しました'
  },
  marketplace: {
    title: 'ストラテジーマーケットプレイス',
    subtitle: 'コミュニティストラテジーの発見、評価、購読',
    publish: 'ストラテジーを公開',
    tabs: {
      marketplace: 'マーケットプレイス',
      subscriptions: 'マイ購読'
    },
    searchPlaceholder: 'ストラテジーを検索...',
    filterByClass: '資産クラスで絞り込み',
    sort: {
      newest: '最新順',
      popular: '人気順',
      performance: 'パフォーマンス順'
    },
    empty: 'まだ公開されたストラテジーはありません',
    noSubscriptions: 'まだ購読していません',
    card: {
      subscribe: '購読',
      subscribed: '購読中',
      unsubscribe: '購読解除',
      unsubscribeHint: 'クリックして購読解除',
      details: '詳細',
      subscribers: '購読者',
      winRate: '勝率',
      by: 'by'
    },
    assetClass: {
      forex: '外国為替',
      crypto: '暗号資産',
      commodity: '商品',
      index: '指数',
      stock: '株式',
      other: 'その他'
    },
    risk: {
      low: '低リスク',
      medium: '中リスク',
      high: '高リスク'
    },
    messages: {
      loginFirst: '先にログインしてください',
      subscribed: '購読しました',
      subscribeFailed: '購読に失敗しました',
      unsubscribed: '購読を解除しました',
      unsubscribeFailed: '購読解除に失敗しました',
      rated: '評価を送信しました',
      rateFailed: '評価に失敗しました',
      commentPosted: 'コメントを投稿しました',
      commentFailed: 'コメント投稿に失敗しました'
    },
    detail: {
      comments: 'コメント',
      noComments: 'まだコメントはありません。最初に投稿してみませんか？',
      commentPlaceholder: 'コメントを入力...（Shift+Enterで改行）'
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
    label: '検出された銘柄',
    loading: '解析中...',
    noSymbols: '取引銘柄が検出されませんでした。具体的な銘柄名を含めてみてください（例：「Bitcoin」「EURUSD」「Gold」）',
    unresolvedTooltip: '取引口座が未バインドのため、解決できません',
    resolvedTooltip: 'ブローカー：{{broker}} | モード：{{mode}}',
    tradeMode: {
      disabled: '無効',
      longOnly: '買いのみ',
      shortOnly: '売りのみ',
      longShort: '両建て可',
      unknown: '不明（{{mode}}）'
    }
  }
} as const;

export default base;
