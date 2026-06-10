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
      registerNow: '新規登録'
    },
    register: {
      subtitle: '新規アカウント作成',
      signingUp: '登録中...',
      register: '登録',
      haveAccount: 'すでにアカウントをお持ちですか？',
      loginNow: 'ログイン'
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
  deleteSelected: '選択した{{count}}件を削除',
    loadingFailed: '読み込みに失敗しました',
    none: 'なし',
    close: '閉じる',
    operationFailed: '操作に失敗しました',
    pleaseWait: 'しばらくお待ちください...',
    next: '次へ',
    previous: '戻る',
    gotIt: '了解',
    loading: '読み込み中...',
    searching: '検索中...',
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
    totalItems: '共 {{count}} 项',
    time: {
      minute: '{{n}}分',
      hour: '{{n}}时',
      day: '{{n}}天',
      lessThanMinute: '<1分'
    },
    required: '必須',
    ok: 'OK',
    error: 'エラー',
    retry: 'リトライ',
    pageError: 'ページエラー',
    unexpectedError: '予期しないエラーが発生しました',
    lineColor: 'ライン色',
    selectSymbolToViewChart: '銘柄を選択してチャートを表示',
    currentPosition: '📊 現在のポジション',
    noOpenPositionsForSymbol: '{{symbol}} のポジションはありません',
    indicatorSettings: '{{name}} 設定',
    active: '启用',
    inactive: '停用',
    clear: '清除',
    saveSuccess: '保存成功',
    remove: '移除',
    yes: '是',
    no: '否',
    you: '你',
    comingSoon: '即将上线',
    pageUnderDevelopment: '此页面正在开发中'
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
    strategy: '戦略',
    accounts: '口座管理',
    aiAssistant: 'AIアシスタント',
    strategies: '戦略管理',
    trading: '取引',
    wallet: 'ウォレット',
    algoDashboard: 'アルゴダッシュボード',
    market: 'マーケット',
    analytics: '分析',
    marketplace: 'マーケットプレイス',
    experiments: '戦略実験',
    marketRegime: 'マーケットレジーム',
    assets: '戦略資産',
    schedules: '戦略スケジュール',
    indicatorCatalog: 'インジケーターカタログ',
    logs: 'システムログ',
    assetAnalysis: 'AI分析',
    autoTrading: '自動取引',
    marketTools: 'マーケット分析ツール',
    devGroup: '戦略開発',
    opsGroup: '戦略運用',
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
    mid: '仲値',
    allSymbols: '全銘柄',
    common: '共通',
    selectSymbol: '銘柄を選択',
    noSymbolsFound: '銘柄が見つかりません',
    loadingSymbols: '読み込み中...',
    emptyWatchlist: '暂无自选',
    searchSymbol: '搜索品种...'
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
      all: 'すべて ({{count}})',
      unread: '未読 ({{count}})'
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
        completed: '{{symbol}} {{action}} が完了しました',
        failed: '実行失敗: {{error}}'
      },
      riskAlert: {
        title: 'リスクアラート',
        fallback: 'アラートタイプ: {{alertType}}'
      },
      strategySignal: {
        title: '戦略シグナル',
        message: '{{symbol}} が {{signalType}} をトリガーしました'
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
      marketplace: '策略市场',
      subscriptions: '我的订阅'
    },
    searchPlaceholder: 'ストラテジーを検索...',
    filterByClass: '資産クラスで絞り込み',
    sort: {
      newest: '最新',
      popular: '最热门',
      performance: '最佳表现'
    },
    empty: 'まだ公開されたストラテジーはありません',
    noSubscriptions: 'まだ購読していません',
    card: {
      subscribe: '購読',
      subscribed: '已订阅',
      unsubscribe: '取消订阅',
      unsubscribeHint: 'クリックして購読解除',
      details: '详情',
      subscribers: '購読者',
      winRate: '胜率',
      by: '发布者'
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
      low: '低',
      medium: '中',
      high: '高'
    },
    messages: {
      loginFirst: '先にログインしてください',
      subscribed: '購読しました',
      subscribeFailed: '订阅失败',
      unsubscribed: '購読を解除しました',
      unsubscribeFailed: '取消订阅失败',
      rated: '評価を送信しました',
      rateFailed: '评分失败',
      commentPosted: 'コメントを投稿しました',
      commentFailed: '评论失败',
      publishFailed: '发布失败',
      published: '发布成功'
    },
    detail: {
      comments: 'コメント',
      noComments: 'まだコメントはありません。最初に投稿してみませんか？',
      commentPlaceholder: 'コメントを入力...（Shift+Enterで改行）'
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
      free: '免费',
      subscription: '订阅制',
      performanceFee: '绩效分成'
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
      title: '取引監視',
      loadFailed: 'Failed to load trading statistics',
      platform: 'プラットフォーム',
      accounts: 'アカウント',
      orders: '注文',
      volume: '取引量',
      byPlatform: 'プラットフォーム別',
      profitStats: '損益統計',
      totalUsers: '総ユーザー数',
      activeUsers: 'アクティブユーザー',
      totalAccounts: '総アカウント数',
      connectedAccounts: '接続済み',
      totalOrders: '総注文数',
      closedOrders: '決済済み',
      totalVolume: '総取引量',
      netProfit: '純利益',
      totalProfit: '総利益',
      totalLoss: '総損失',
      pendingOrders: '保留中注文'
    },
    dashboard: {
      title: '管理ダッシュボード',
      loadFailed: 'ダッシュボードデータの読み込みに失敗しました',
      totalUsers: '総ユーザー数',
      activeUsers: 'アクティブユーザー',
      mtAccounts: 'MTアカウント',
      onlineAccounts: 'オンラインアカウント',
      todayTrades: '本日の取引',
      todayProfit: '本日の損益',
      recentLogs: '最近のログ',
      logs: {
        time: '時間',
        module: 'モジュール',
        actionType: 'アクション',
        target: '対象',
        status: 'ステータス',
        success: '成功',
        failed: '失敗',
        moduleMap: {
          userManagement: 'ユーザー管理',
          accountManagement: 'アカウント管理',
          trading: '取引',
          systemConfig: 'システム設定'
        }
      },
      riskMetrics: {
        title: 'リスク検証指標',
        riskValidateTotal: '検証総数',
        riskValidatePass: '通過',
        riskValidateReject: '拒否',
        riskValidateError: 'エラー',
        orderSendSuccess: '注文成功',
        orderSendFailed: '注文失敗',
        orderCloseSuccess: '決済成功',
        orderCloseFailed: '決済失敗'
      },
      riskWindow: {
        title: 'リスク管理ウィンドウ',
        validateTotal: '合計',
        validatePass: '通過',
        validateReject: '拒否',
        validateError: 'エラー',
        orderSendSuccess: '注文OK',
        orderSendFailed: '注文失敗',
        orderCloseSuccess: '決済OK',
        orderCloseFailed: '決済失敗',
        rejectRiskCodesHeader: 'リスクコード',
        rejectCount: '拒否数',
        noRejectData: 'この期間に拒否はありません',
        noData: 'リスクデータなし'
      }
    },
    jurisdiction: {
      title: '管轄権管理',
      sanctionedCountriesTab: '制裁対象国',
      kycStatusTab: 'KYCステータス',
      sanctionedCountries: '制裁対象国',
      userKYCStatus: 'ユーザーKYCステータス',
      addCountry: '国を追加',
      addSanctionedCountry: '制裁国を追加',
      countryCode: '国コード',
      countryLabel: '国',
      addedBy: '追加者',
      actions: '操作',
      userEmail: 'ユーザーメール',
      kycStatus: 'KYCステータス',
      country: '国',
      sanctioned: '制裁済み',
      disclaimer: '免責事項',
      questionnaire: 'アンケート',
      override: '上書き',
      setKYC: 'KYC設定',
      setKYCStatus: 'KYCステータス設定',
      grantOverride: '上書き許可',
      revokeOverride: '上書き取消',
      filterByKYCStatus: 'KYCステータスでフィルター',
      unverified: '未確認',
      pending: '保留中',
      verified: '確認済み',
      rejected: '拒否',
      emptySanctions: '制裁国はありません',
      emptyKYC: 'KYCレコードがありません',
      messages: {
        countryAdded: '国を追加しました',
        countryAddFailed: '国の追加に失敗しました',
        countryRemoved: '国を削除しました',
        countryRemoveFailed: '国の削除に失敗しました',
        kycUpdated: 'KYCステータスを更新しました',
        kycUpdateFailed: 'KYCステータス更新に失敗しました',
        overrideUpdated: '上書き設定を更新しました',
        overrideUpdateFailed: '上書き設定更新に失敗しました'
      },
      overrideWarning: 'このユーザーは制裁対象国からのアクセスです。上書き許可で取引が可能になります。',
      confirmGrantOverride: 'このユーザーに上書き許可を付与しますか？',
      confirmRevokeOverride: 'このユーザーの上書き許可を取り消しますか？'
    },
    userManagement: {
      title: 'ユーザー管理',
      addUser: 'ユーザー追加',
      table: {
        id: 'ID',
        email: 'メール',
        nickname: 'ニックネーム',
        role: '役割',
        status: 'ステータス',
        mtAccountCount: 'MTアカウント',
        createdAt: '作成日時',
        actions: '操作'
      },
      actions: {
        details: '詳細',
        enable: '有効化',
        disable: '無効化',
        changePassword: 'パスワード変更'
      },
      filters: {
        searchPlaceholder: 'メールまたは名前で検索',
        rolePlaceholder: '役割でフィルター',
        statusPlaceholder: 'ステータスでフィルター'
      },
      status: {
        active: 'アクティブ',
        suspended: '停止中'
      },
      roles: {
        user: 'ユーザー',
        superAdmin: 'スーパー管理者',
        operation: '運用',
        customerService: 'カスタマーサポート',
        audit: '監査'
      },
      pagination: {
        total: '合計 {{total}} ユーザー'
      },
      deleteConfirm: {
        title: 'このユーザーを削除しますか？この操作は元に戻せません。',
        batchDeleteConfirm: '{{count}}人のユーザーを削除しますか？この操作は元に戻せません。',
        batchDeleteSuccess: '{{count}}人のユーザーを削除しました',
        batchDeletePartial: '{{deleted}}人削除、{{failed}}人失敗',
      },
      modals: {
        createTitle: 'ユーザー作成',
        editTitle: 'ユーザー編集',
        passwordTitle: 'パスワード変更'
      },
      form: {
        email: 'メール',
        nickname: 'ニックネーム',
        password: 'パスワード',
        role: '役割',
        status: 'ステータス',
        accountNumber: '口座番号',
        accountNumberInvalid: '5-6桁、先頭ゼロなし、4と7は不可',
        placeholders: {
          email: 'メールを入力',
          nickname: 'ニックネームを入力',
          password: 'パスワードを入力'
        }
      },
      passwordForm: {
        newPassword: '新しいパスワード',
        confirmPassword: 'パスワード確認',
        placeholders: {
          newPassword: '新しいパスワードを入力',
          confirmPassword: '新しいパスワードを再入力'
        },
        submit: 'パスワード更新',
        validation: {
          newPasswordRequired: '新しいパスワードが必要です',
          confirmPasswordRequired: 'パスワード確認が必要です',
          passwordMin8: 'パスワードは8文字以上必要です',
          passwordMismatch: 'パスワードが一致しません',
          passwordMustContainLettersAndNumbers: 'パスワードには英字と数字の両方を含める必要があります'
        }
      },
      messages: {
        userCreatedSuccess: 'ユーザーを作成しました',
        userCreateFailed: 'ユーザー作成に失敗しました',
        userUpdatedSuccess: 'ユーザーを更新しました',
        userUpdateFailed: 'ユーザー更新に失敗しました',
        userDeletedSuccess: 'ユーザーを削除しました',
        userDeleteFailed: 'ユーザー削除に失敗しました',
        userEnabled: 'ユーザーを有効化しました',
        userDisabled: 'ユーザーを無効化しました',
        passwordUpdatedSuccess: 'パスワードを更新しました',
        passwordUpdateFailed: 'パスワード更新に失敗しました',
        newPasswordIs: '新しいパスワード: {{password}}'
      },
      drawer: {
        title: 'ユーザー詳細',
        labels: {
          id: 'ID',
          email: 'メール',
          nickname: 'ニックネーム',
          role: '役割',
          status: 'ステータス',
          mtAccountCount: 'MTアカウント',
          createdAt: '作成日時',
          lastLogin: '最終ログイン'
        }
      }
    },
    wallet: {
      title: 'ウォレット管理',
      searchPlaceholder: 'メールまたは口座番号で検索...',
      noUsers: 'ユーザーが見つかりません',
      walletFor: 'ウォレット -',
      accountNumber: '口座番号',
      adjustBalance: '残高調整',
      adjustSuccess: '残高を調整しました',
      adjustFailed: '調整に失敗しました',
      add: '追加',
      deduct: '控除',
      reason: '調整理由...',
    }
  },
  wallet: {
    title: 'マイウォレット',
    accountNumber: '口座番号',
    table: {
      type: '種類',
      amount: '金額',
      balanceAfter: '調整後残高',
      description: '説明',
      time: '時間',
    },
    balance: '残高',
    frozen: '凍結',
    frozenBalance: '凍結',
    currency: '通貨',
    transactions: '取引履歴',
    deposit: '入金',
    withdraw: '出金',
    history: '履歴',
  },
  symbolDetection: {
    label: '検出された銘柄',
    loading: '解析中...',
    noSymbols: '取引銘柄が検出されませんでした。具体的な銘柄名を含めてみてください（例：「Bitcoin」「EURUSD」「Gold」）',
    unresolvedTooltip: '取引口座が未バインドのため、解決できません',
    resolvedTooltip: 'ブローカー：{{broker}} | モード：{{mode}}',
    tradeMode: {
      disabled: '已禁用',
      longOnly: '仅做多',
      shortOnly: '仅做空',
      longShort: '多空双向',
      unknown: '未知'
    }
  },
  autoTrading: {
    title: '自動取引',
    status: {
      enabled: '自動取引が有効です',
      disabled: '自動取引が無効です',
      activeStrategies: 'アクティブ戦略',
      todayExecutions: '本日の実行',
      todayProfit: '本日の損益'
    },
    settings: {
      title: 'グローバルリスク設定',
      maxRiskPercent: '最大リスク%',
      maxPositions: '最大ポジション数',
      maxLotSize: '最大ロットサイズ',
      maxDailyLoss: '最大日次損失',
      maxDrawdownPercent: '最大ドローダウン%',
      saveSuccess: '設定を保存しました',
      saveFailed: '設定の保存に失敗しました'
    },
    logs: {
      title: '最近の取引ログ',
      empty: '取引ログはまだありません',
      columns: {
        time: '時間',
        symbol: '銘柄',
        action: 'アクション',
        volume: '数量',
        price: '価格',
        profit: '損益',
        ticket: 'チケット'
      }
    },
    messages: {
      loadFailed: '自動取引データの読み込みに失敗しました',
      toggleFailed: '自動取引の切り替えに失敗しました'
    }
  }
} as const;

export default base;
