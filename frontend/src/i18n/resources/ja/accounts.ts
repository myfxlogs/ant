const accounts = {
  accounts: {
    title: '口座',
    subtitle: 'MT4/MT5 口座を管理します',
    bindNew: '口座を連携',
    bind: {
      title: 'MT 口座を連携',
      errorModal: {
        title: 'Binding failed'
      },
      step1: {
        title: 'プラットフォームとブローカーを選択',
        subtitle: 'Select your trading platform and search for your broker'
      },
      step2: {
        title: '口座情報を入力',
        subtitle: 'Enter your trading account and password'
      },
      step3: {
        title: '連携内容を確認',
        subtitle: 'Verify credentials and confirm to complete'
      },
      fields: {
        platform: 'プラットフォーム',
        brokerName: 'ブローカー名',
        company: '会社名',
        server: 'サーバー',
        tradingAccount: '取引口座',
        password: 'パスワード'
      },
      placeholders: {
        brokerName: 'ブローカー名を入力（例：XM、IC Markets）',
        company: '会社を選択',
        server: 'サーバーを選択',
        tradingAccount: '取引口座を入力',
        password: 'Enter password'
      },
      labels: {
        serverCount: '{{count}} servers'
      },
      actions: {
        search: '検索',
        verifyAccount: 'アカウントを確認',
        confirmBind: '連携を確定',
        retryVerify: 'Retry'
      },
      passwordHint: 'パスワードは HTTPS で送信され、バックエンドで Argon2id ハッシュとして保存されます（復元不可）。',
      summary: {
        broker: 'ブローカー',
        server: 'サーバー',
        platform: 'プラットフォーム',
        tradingAccount: '取引口座',
        password: 'パスワード',
        verified: 'アカウント確認済み',
        balance: '残高',
        equity: '純資産',
        margin: '証拠金',
        freeMargin: '余剰証拠金',
        leverage: 'レバレッジ',
        currency: 'Currency'
      },
      messages: {
        enterBrokerName: 'ブローカー名を入力してください',
        foundBrokers: '{{count}} 件のブローカーが見つかりました',
        noBrokersFound: '一致するブローカーが見つかりません。名称を確認してください。',
        searchFailed: '検索に失敗しました。しばらくしてから再試行してください。',
        selectServer: 'サーバーを選択してください',
        enterTradingAccount: '取引口座を入力してください',
        enterPassword: 'パスワードを入力してください',
        noAccessHosts: '利用可能なアクセスホストがありません',
        verifyFailed: 'アカウント確認に失敗しました',
        bindSuccess: '口座を連携しました',
        bindFailed: '口座の連携に失敗しました',
        loginDigitsOnly: 'Trading account must contain only digits'
      },
      errors: {
        brokerUnavailable: 'サーバーエラーまたはパスワードが正しくありません',
        invalidCredentials: '口座が見つからないか、パスワードが無効です',
        connectionFailed: 'ブローカーサーバーに接続できません。ネットワークを確認してください',
        timeout: 'Connection timed out, please try again later'
      }
    },
    empty: {
      title: '連携済み口座がありません',
      subtitle: 'Click the button below to bind your MT4/MT5 trading account'
    },
    legend: {
      title: '凡例:',
      connected: '接続済み',
      connecting: '接続中',
      disconnectedOrError: '切断/エラー',
      disabled: '無効'
    },
    messages: {
      disabledSuccess: '口座を無効化しました',
      connectingMtServer: 'MT サーバーに接続中',
      enabledSuccess: '口座を有効化しました',
      fetchListFailed: '口座一覧の取得に失敗しました',
      fetchAccountFailed: '口座情報の取得に失敗しました',
      createdSuccess: '口座を作成しました',
      createFailed: '口座の作成に失敗しました',
      connectSuccess: '接続しました',
      connectFailed: '接続に失敗しました',
      disconnectFailed: '切断に失敗しました',
      disableFailed: '口座の無効化に失敗しました',
      deleted: '口座を削除しました',
      deleteFailed: '削除に失敗しました',
      enableFailed: 'Failed to enable account'
    },
    analytics: {
      monthlyAnalysis: {
        title: '月次分析',
        chartMainTitle: '月次リターン（{{metric}}）',
        metrics: {
          change: '変化',
          profit: '損益',
          lots: 'ロット',
          pips: 'Pips'
        },
        focusedValue: '{{period}} · {{metric}}: {{value}}',
        bonus: {
          chartRiskTitle: 'Bonus: {{month}}のシンボル別プロフィットファクター。',
          chartPopularTitle: `{{month}}'s currency popularity.`,
          chartHoldingTitle: `{{month}}'s average holding time.`,
          legendBulls: '買い',
          legendShortTerm: '売り',
          sliceOther: 'その他',
          emptyCharts: 'この月の取引なし',
          popularityShare: 'Lot volume share'
        }
      },
      monthlyDetail: {
        metricsTitle: '月次指標',
        symbolPnLTitle: '銘柄別損益',
        holdingTitle: '保有時間',
        riskRewardTitle: '報酬:リスク比率',
        popularityTitle: '通貨人気度',
        long: '買い',
        short: '売り',
        fields: {
          netReturn: '純利益',
          totalTrades: '取引数',
          winRate: '勝率',
          profitFactor: 'PF',
          bestTrade: '最良取引',
          worstTrade: '最悪取引',
          averageHours: '平均',
          medianHours: '中央値',
          maxHours: '最大',
          minHours: '最小',
        },
      },
      chartType: {
        equity: '純資産',
        balance: '残高',
        profit: '損益'
      },
      chartPeriod: {
        day: '今日',
        week: '今週',
        month: '今月',
        year: '今年',
        all: 'All'
      },
      chartSeries: {
        equity: '純資産',
        balance: '残高',
        profit: '損益',
        tradeCount: '取引数'
      },
      empty: {
        equityCurve: 'エクイティカーブのデータがありません',
        monthlyProfit: '月次損益データがありません',
        symbolDistribution: '銘柄分布データがありません',
        dailyPnL: '日次損益データがありません',
        hourly: 'No time-of-day analysis data'
      },
      monthlyProfitTitle: '月次損益',
      advancedStatsTitle: '詳細統計',
      symbolDistributionTitle: '銘柄分布',
      dailyPnLTitle: '日次損益',
      hourlyTitle: '時間帯分析',
      advancedTabs: {
        hourly: '時間足',
        daily: 'Daily'
      },
      timeDetail: {
        lots: 'ロット',
        trades: '取引数',
        profitAmount: '損益額',
        balance: '残高',
        profitFactor: 'プロフィットファクター',
        maxFloatingLossAmount: '最大含み損額',
        maxFloatingLossRatio: '最大含み損比率',
        maxFloatingProfitAmount: '最大含み益額',
        maxFloatingProfitRatio: 'Max floating profit ratio'
      },
      stats: {
        winRate: '勝率',
        profitFactor: 'プロフィットファクター',
        maxDrawdown: '最大ドローダウン',
        totalTrades: '取引回数',
        avgProfit: '平均利益',
        avgLoss: '平均損失',
        avgHolding: '平均保有時間',
        consecutiveWinsLosses: '連勝/連敗',
        sharpe: 'シャープレシオ',
        sortino: 'ソルティノレシオ',
        calmar: 'カルマーレシオ',
        largestWin: '最大利益',
        largestLoss: '最大損失',
        avgDailyReturn: '平均日次リターン',
        volatility: 'ボラティリティ',
        netProfit: '純利益',
        totalDeposit: '総入金',
        totalWithdrawal: '総出金',
        netDeposit: 'Net deposit'
      }
    },
    card: {
      status: {
        disabled: '無効',
        connected: '接続済み',
        connecting: '接続中',
        disconnected: '切断',
        error: 'Error'
      },
      fields: {
        balance: '残高',
        equity: '純資産',
        broker: 'ブローカー',
        server: 'サーバー'
      },
      actions: {
        positions: '保有ポジション',
        orders: '注文',
        details: 'Details'
      },
      deleteConfirm: {
        title: 'この口座を削除しますか？',
        content: 'This action cannot be undone'
      }
    },
    disabled: {
      title: '無効な口座',
      table: {
        account: '口座',
        type: 'タイプ',
        broker: 'ブローカー',
        balance: '残高',
        equity: '純資産',
        actions: 'Actions'
      },
      confirmDelete: {
        title: 'この口座を削除しますか？',
        content: 'This action cannot be undone'
      },
      mobile: {
        balanceLabel: '残高: ',
        equityLabel: 'Equity: '
      }
    },
    tradeTabs: {
      positionsWithCount: '保有ポジション（{{count}}）',
      pendingWithCount: '未決注文（{{count}}）',
      historyWithCount: '履歴（{{count}}）',
      emptyPositions: '保有ポジションがありません',
      emptyHistory: '注文履歴がありません',
      syncHistory: '履歴同期',
      table: {
        orderId: '注文ID',
        symbol: '銘柄',
        side: '売買',
        type: 'タイプ',
        volume: '数量',
        openPrice: '建値',
        currentPrice: '現在値',
        pendingPrice: '指値/逆指値',
        closePrice: '決済価格',
        profit: '損益',
        openTime: '建玉時間',
        pendingTime: '注文時間',
        closeTime: 'Close time'
      },
      pagination: {
        total: '{{total}} total'
      }
    },
    edit: {
      title: '口座編集',
      fields: {
        tradingAccount: '取引口座',
        server: 'サーバー',
        password: '新しいパスワード',
        oldPassword: 'Current password'
      },
      placeholders: {
        newPassword: '新しいパスワードを入力',
        oldPassword: 'Enter current password'
      },
      messages: {
        enterPassword: '新しいパスワードを入力してください',
        enterOldPassword: '現在のパスワードを入力してください',
        passwordVerifyFailed: 'パスワード変更に失敗しました',
        passwordSaved: 'Password saved'
      }
    },
    detail: {
      messages: {
        fetchAccountFailed: '口座情報の取得に失敗しました。しばらくしてから再試行してください。',
        syncHistorySuccess: '注文履歴の同期に成功しました',
        syncHistoryFailed: 'Failed to sync order history. Please ensure the account is connected to the MT server.'
      },
      orderTypes: {
        buyLimit: '買い指値',
        sellLimit: '売り指値',
        buyStop: '買い逆指値',
        sellStop: 'Sell stop'
      },
      balanceRecord: {
        deposit: '💰 入金',
        withdraw: '💸 出金',
        depositIconText: '💰 入金',
        withdrawIconText: '💸 出金'
      },
      syncHistory: {
        title: '注文履歴を同期',
        content: '過去1年分の注文履歴を MT サーバーから同期しますか？時間がかかる場合があります。',
        ok: 'Sync'
      },
      actions: {
        enableAccount: '口座を有効化',
        disableAccount: '口座を無効化',
        deleteAccount: 'アカウント削除',
        deleteConfirm: '確認して削除',
        deleteWarning: 'この操作は元に戻せません。取引記録、分析データなど、すべてのアカウントデータが完全に削除されます。',
        deletePasswordHint: '確認のため、MT取引パスワードまたは読み取り専用パスワードを入力してください：',
        deletePasswordPlaceholder: 'MT取引/読み取り専用パスワード',
        syncHistory: 'Sync history'
      },
      status: {
        disabled: '無効',
        connected: '接続済み',
        connecting: '接続中',
        disconnected: '切断',
        error: 'Error'
      },
      accountType: {
        real: 'リアル',
        demo: 'Demo'
      },
      mode: {
        investor: '投資家モード',
        trader: 'Trader mode'
      },
      connected: '接続済み',
        lastConnected: '{{time}}',
        leverage: 'レバレッジ {{leverage}} 倍',
      cards: {
        balance: '残高',
        equity: '純資産',
        floatingProfit: '含み損益',
        marginUsed: '使用証拠金',
        marginFree: '余剰証拠金',
        marginLevel: '証拠金維持率',
        credit: 'Credit'
      }
    },
    report: {
      title: '取引レポート',
      titleShort: 'レポート',
      generate: 'レポート生成',
      goToAISettings: 'AI設定へ →',
      aiAnalysis: 'AI分析',
      symbolPnL: '銘柄別損益',
      direction: '売買分析',
      directionLong: '買い',
      directionShort: '売り',
      tradeDistribution: '損益分布',
      drawdownOverlay: '資産曲線 + ドローダウン',
      drawdownEvents: 'ドローダウンイベント',
      recovered: '回復済み',
      winRateTrend: '月次勝率トレンド',
      periods: {
        week: '今週',
        month: '今月',
        quarter: '今四半期',
        year: '今年',
      },
      sections: {
        summary: '総評',
        findings: '主な発見',
        recommendations: '改善提案',
      },
    },
  }
} as const;

export default accounts;
