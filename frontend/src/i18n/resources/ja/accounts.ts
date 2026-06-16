const accounts = {
  accounts: {
    title: '口座',
    subtitle: 'MT4/MT5 口座を管理します',
    bindNew: '口座を連携',
    bind: {
      title: 'MT 口座を連携',
      errorModal: {
        title: '連携に失敗しました'
      },
      step1: {
        title: 'プラットフォームとブローカーを選択',
        subtitle: '取引プラットフォームを選び、ブローカー名で検索します'
      },
      step2: {
        title: '口座情報を入力',
        subtitle: '取引口座とパスワードを入力してください'
      },
      step3: {
        title: '連携内容を確認',
        subtitle: '以下の内容を確認してください'
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
        password: 'パスワードを入力'
      },
      labels: {
        serverCount: '{{count}} サーバー'
      },
      actions: {
        search: '検索',
        confirmBind: '連携を確定',
        verifyAccount: 'アカウントを確認',
        retryVerify: '再試行'
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
        freeMargin: '有効証拠金',
        leverage: 'レバレッジ',
        currency: '通貨'
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
        bindSuccess: '口座を連携しました',
        bindFailed: '口座の連携に失敗しました',
        loginDigitsOnly: '取引口座は数字のみ入力可能です',
        verifyFailed: 'アカウント確認に失敗しました'
      },
      errors: {
        brokerUnavailable: 'サーバーエラーまたはパスワードが正しくありません',
        invalidCredentials: '口座が見つからないか、パスワードが無効です',
        connectionFailed: 'ブローカーサーバーに接続できません。ネットワークを確認してください',
        timeout: '接続がタイムアウトしました。再試行してください'
      }
    },
    empty: {
      title: '連携済み口座がありません',
      subtitle: '下のボタンから MT4/MT5 口座を連携してください'
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
      enableFailed: '启用账户失败'
    },
    analytics: {
      monthlyAnalysis: {
        title: '月次分析',
        metrics: {
          change: '変化',
          profit: '利益',
          lots: 'ロット',
          pips: 'pips'
        },
        chartMainTitle: '月次リターン（{{metric}}）',
        focusedValue: '{{period}} · {{metric}}: {{value}}',
        bonus: {
          chartRiskTitle: 'Bonus: {{month}}のシンボル別プロフィットファクター。',
          chartPopularTitle: '{{month}}の通貨人気（ロット比率）。',
          chartHoldingTitle: '{{month}}の平均保有時間（買い/売り合計秒の積み上げ）。',
          legendBulls: '買い',
          legendShortTerm: '売り',
          sliceOther: 'その他',
          emptyCharts: 'この月の取引なし',
          popularityShare: 'ロット比率'
        }
      },
      monthlyDetail: {
        metricsTitle: '月次指標',
        symbolPnLTitle: '銘柄別損益',
        holdingTitle: '保有時間',
        riskRewardTitle: '報酬:リスク比率',
        popularityTitle: '通貨人気度',
        long: 'ロング',
        short: 'ショート',
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
        all: '全期間'
      },
      chartSeries: {
        equity: '純資産',
        balance: '残高',
        profit: '損益',
        tradeCount: '取引回数'
      },
      empty: {
        equityCurve: 'エクイティカーブのデータがありません',
        monthlyProfit: '月次損益データがありません',
        symbolDistribution: '銘柄分布データがありません',
        dailyPnL: '日次損益データがありません',
        hourly: '時間帯分析データがありません'
      },
      monthlyProfitTitle: '月次損益',
      advancedStatsTitle: '詳細統計',
      symbolDistributionTitle: '銘柄分布',
      dailyPnLTitle: '日次損益',
      hourlyTitle: '時間帯分析',
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
        netDeposit: '純入金'
      },
      advancedTabs: {
        hourly: '時間足',
        daily: '日足'
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
        maxFloatingProfitRatio: '最大含み益比率'
      }
    },
    card: {
      status: {
        disabled: '無効',
        connected: '接続済み',
        connecting: '接続中',
        disconnected: '切断',
        error: 'エラー'
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
        details: '詳細'
      },
      deleteConfirm: {
        title: 'この口座を削除しますか？',
        content: 'この操作は元に戻せません。'
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
        actions: '操作'
      },
      confirmDelete: {
        title: 'この口座を削除しますか？',
        content: 'この操作は元に戻せません。'
      },
      mobile: {
        balanceLabel: '残高: ',
        equityLabel: '純資産: '
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
        closeTime: '決済時間'
      },
      pagination: {
        total: '合計 {{total}} 件'
      }
    },
    edit: {
      title: '口座編集',
      fields: {
        tradingAccount: '取引口座',
        server: 'サーバー',
        password: '新しいパスワード',
        oldPassword: '現在のパスワード'
      },
      placeholders: {
        newPassword: '新しいパスワードを入力',
        oldPassword: '現在のパスワードを入力'
      },
      messages: {
        enterPassword: '新しいパスワードを入力してください',
        enterOldPassword: '現在のパスワードを入力してください',
        passwordVerifyFailed: 'パスワード変更に失敗しました',
        passwordSaved: 'パスワードを保存しました'
      }
    },
    detail: {
      messages: {
        fetchAccountFailed: '口座情報の取得に失敗しました。しばらくしてから再試行してください。',
        syncHistorySuccess: '注文履歴の同期に成功しました',
        syncHistoryFailed: '注文履歴の同期に失敗しました。口座が MT サーバーに接続されていることを確認してください。'
      },
      orderTypes: {
        buyLimit: '買い指値',
        sellLimit: '売り指値',
        buyStop: '買い逆指値',
        sellStop: '売り逆指値'
      },
      balanceRecord: {
        deposit: '入金',
        withdraw: '出金',
        depositIconText: '💰 入金',
        withdrawIconText: '💸 出金'
      },
      syncHistory: {
        title: '注文履歴を同期',
        content: '過去1年分の注文履歴を MT サーバーから同期しますか？時間がかかる場合があります。',
        ok: '同期する'
      },
      actions: {
        enableAccount: '口座を有効化',
        disableAccount: '口座を無効化',
        syncHistory: '履歴を同期',
        deleteAccount: 'アカウント削除',
        deleteConfirm: '確認して削除',
        deleteWarning: 'この操作は元に戻せません。取引記録、分析データなど、すべてのアカウントデータが完全に削除されます。',
        deletePasswordHint: '確認のため、MT取引パスワードまたは読み取り専用パスワードを入力してください：',
        deletePasswordPlaceholder: 'MT取引/読み取り専用パスワード'
      },
      status: {
        disabled: '無効',
        connected: '接続済み',
        connecting: '接続中',
        disconnected: '切断',
        error: 'エラー'
      },
      accountType: {
        real: 'リアル',
        demo: 'デモ'
      },
      mode: {
        investor: '投資家モード',
        trader: 'トレーダーモード'
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
        credit: 'クレジット'
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
