const accounts = {
  accounts: {
    title: '我的帳戶',
    subtitle: '管理您的 MT4/MT5 交易帳戶',
    bindNew: '綁定新帳戶',
    bind: {
      title: '綁定 MT 帳戶',
      errorModal: {
        title: 'Binding failed'
      },
      step1: {
        title: '選擇平台與經紀商',
        subtitle: 'Select your trading platform and search for your broker'
      },
      step2: {
        title: '輸入帳戶資訊',
        subtitle: 'Enter your trading account and password'
      },
      step3: {
        title: '確認綁定',
        subtitle: 'Verify credentials and confirm to complete'
      },
      fields: {
        platform: '交易平台',
        brokerName: '經紀商名稱',
        company: '選擇公司',
        server: '伺服器',
        tradingAccount: '交易帳號',
        password: '密碼'
      },
      placeholders: {
        brokerName: '輸入經紀商名稱，如：XMGlobal、ICMarkets',
        company: '請選擇經紀商公司',
        server: '請選擇伺服器',
        tradingAccount: '輸入交易帳號',
        password: 'Enter password'
      },
      labels: {
        serverCount: '{{count}} servers'
      },
      actions: {
        search: '搜尋',
        verifyAccount: '驗證帳戶',
        confirmBind: '確認綁定',
        retryVerify: 'Retry'
      },
      passwordHint: '密碼將透過 HTTPS 加密傳輸，後端使用 Argon2id 雜湊儲存不可回逆',
      summary: {
        broker: '經紀商',
        server: '伺服器',
        platform: '交易平台',
        tradingAccount: '交易帳號',
        password: '密碼',
        verified: '帳戶已驗證',
        balance: '餘額',
        equity: '淨值',
        margin: '保證金',
        freeMargin: '可用保證金',
        leverage: '槓桿',
        currency: 'Currency'
      },
      messages: {
        enterBrokerName: '請輸入經紀商名稱',
        foundBrokers: '找到 {{count}} 個經紀商',
        noBrokersFound: '未找到匹配的經紀商，請檢查名稱',
        searchFailed: '搜尋失敗，請稍後重試',
        selectServer: '請選擇伺服器',
        enterTradingAccount: '請輸入交易帳號',
        enterPassword: '請輸入密碼',
        noAccessHosts: '無可用伺服器位址',
        verifyFailed: '帳戶驗證失敗',
        bindSuccess: '帳戶綁定成功',
        bindFailed: '帳戶綁定失敗',
        loginDigitsOnly: 'Trading account must contain only digits'
      },
      errors: {
        brokerUnavailable: '連線伺服器錯誤或密碼不正確',
        invalidCredentials: '帳號或密碼錯誤，未找到該交易帳戶',
        connectionFailed: '無法連線到經紀商伺服器，請檢查網路',
        timeout: 'Connection timed out, please try again later'
      }
    },
    empty: {
      title: '暫無綁定帳戶',
      subtitle: 'Click the button below to bind your MT4/MT5 trading account'
    },
    legend: {
      title: '圖例:',
      connected: '已連接',
      connecting: '連線中',
      disconnectedOrError: '已斷線/錯誤',
      disabled: '已停用'
    },
    messages: {
      disabledSuccess: '帳戶停用成功',
      connectingMtServer: '正在連線 MT 伺服器',
      enabledSuccess: '帳戶啟用成功',
      fetchListFailed: '取得帳戶列表失敗',
      fetchAccountFailed: '取得帳戶資訊失敗',
      createdSuccess: '帳戶建立成功',
      createFailed: '建立帳戶失敗',
      connectSuccess: '連線成功',
      connectFailed: '連線失敗',
      disconnectFailed: '斷開連線失敗',
      disableFailed: '停用帳戶失敗',
      deleted: '帳戶已刪除',
      deleteFailed: '刪除失敗',
      enableFailed: 'Failed to enable account'
    },
    analytics: {
      monthlyAnalysis: {
        title: '月度分析',
        chartMainTitle: '每月收益（{{metric}}）',
        metrics: {
          change: '變化',
          profit: '盈虧',
          lots: '手數',
          pips: 'Pips'
        },
        focusedValue: '{{period}} · {{metric}}：{{value}}',
        bonus: {
          chartRiskTitle: 'Bonus：{{month}} 各品種風險報酬比（盈利因子）。',
          chartPopularTitle: `{{month}}'s currency popularity.`,
          chartHoldingTitle: `{{month}}'s average holding time.`,
          legendBulls: '買入側',
          legendShortTerm: '賣出側',
          sliceOther: '其他',
          emptyCharts: '該月無成交',
          popularityShare: 'Lot volume share'
        }
      },
      monthlyDetail: {
        metricsTitle: '月度指標',
        symbolPnLTitle: '品種盈虧',
        holdingTitle: '持倉時長',
        riskRewardTitle: '獎勵:風險比率',
        popularityTitle: '貨幣流行度',
        long: '做多',
        short: '做空',
        fields: {
          netReturn: '淨收益',
          totalTrades: '總筆數',
          winRate: '勝率',
          profitFactor: '盈虧比',
          bestTrade: '最優單筆',
          worstTrade: '最差單筆',
          averageHours: '平均',
          medianHours: '中位',
          maxHours: '最長',
          minHours: '最短',
        },
      },
      chartType: {
        equity: '淨值',
        balance: '餘額',
        profit: '盈虧'
      },
      chartPeriod: {
        day: '今日',
        week: '本週',
        month: '本月',
        year: '今年',
        all: 'All'
      },
      chartSeries: {
        equity: '淨值',
        balance: '餘額',
        profit: '盈虧',
        tradeCount: '交易次數'
      },
      empty: {
        equityCurve: '暫無淨值曲線資料',
        monthlyProfit: '暫無月度盈虧資料',
        symbolDistribution: '暫無品種資料',
        dailyPnL: '暫無每日盈虧資料',
        hourly: 'No time-of-day analysis data'
      },
      monthlyProfitTitle: '月度盈虧',
      advancedStatsTitle: '進階統計',
      symbolDistributionTitle: '品種分布',
      dailyPnLTitle: '📅 每日盈虧',
      hourlyTitle: '⏰ 時段分析',
      advancedTabs: {
        hourly: '每小時',
        daily: 'Daily'
      },
      timeDetail: {
        lots: '手數',
        trades: '交易次數',
        profitAmount: '利潤金額',
        balance: '餘額',
        profitFactor: '盈虧比',
        maxFloatingLossAmount: '最大浮動虧損金額',
        maxFloatingLossRatio: '最大浮動虧損比例',
        maxFloatingProfitAmount: '最大浮動盈利金額',
        maxFloatingProfitRatio: 'Max floating profit ratio'
      },
      stats: {
        winRate: '勝率',
        profitFactor: '盈虧比',
        maxDrawdown: '最大回撤',
        totalTrades: '總交易數',
        avgProfit: '平均盈利',
        avgLoss: '平均虧損',
        avgHolding: '平均持倉',
        consecutiveWinsLosses: '連勝/連敗',
        sharpe: '夏普比率',
        sortino: '索提諾',
        calmar: '卡爾馬',
        largestWin: '最大盈利',
        largestLoss: '最大虧損',
        avgDailyReturn: '日均收益',
        volatility: '波動率',
        netProfit: '淨利潤',
        totalDeposit: '入金',
        totalWithdrawal: '出金',
        netDeposit: 'Net deposit'
      }
    },
    card: {
      status: {
        disabled: '已停用',
        connected: '已連接',
        connecting: '連線中',
        disconnected: '已斷線',
        error: 'Error'
      },
      fields: {
        balance: '餘額',
        equity: '淨值',
        broker: '經紀商',
        server: '伺服器'
      },
      actions: {
        positions: '持倉',
        orders: '訂單',
        details: 'Details'
      },
      deleteConfirm: {
        title: '確定刪除此帳戶？',
        content: 'This action cannot be undone'
      }
    },
    disabled: {
      title: '已停用的帳戶',
      table: {
        account: '帳號',
        type: '類型',
        broker: '經紀商',
        balance: '餘額',
        equity: '淨值',
        actions: 'Actions'
      },
      confirmDelete: {
        title: '確定刪除此帳戶？',
        content: 'This action cannot be undone'
      },
      mobile: {
        balanceLabel: '餘額: ',
        equityLabel: 'Equity: '
      }
    },
    tradeTabs: {
      positionsWithCount: '持倉訂單 ({{count}})',
      pendingWithCount: '掛單 ({{count}})',
      historyWithCount: '歷史訂單 ({{count}})',
      emptyPositions: '暫無持倉',
      emptyHistory: '暫無歷史訂單',
      syncHistory: '同步历史',
      table: {
        orderId: '訂單號',
        symbol: '品種',
        side: '方向',
        type: '類型',
        volume: '手數',
        openPrice: '開倉價',
        currentPrice: '當前價',
        pendingPrice: '掛單價格',
        closePrice: '平倉價',
        profit: '盈虧',
        openTime: '開倉時間',
        pendingTime: '掛單時間',
        closeTime: 'Close time'
      },
      pagination: {
        total: '{{total}} total'
      }
    },
    edit: {
      title: '編輯帳戶',
      fields: {
        tradingAccount: '交易帳號',
        server: '伺服器',
        password: '新密碼',
        oldPassword: 'Current password'
      },
      placeholders: {
        newPassword: '輸入新密碼',
        oldPassword: 'Enter current password'
      },
      messages: {
        enterPassword: '請輸入新密碼',
        enterOldPassword: '請輸入當前密碼',
        passwordVerifyFailed: '密碼修改失敗',
        passwordSaved: 'Password saved'
      }
    },
    detail: {
      messages: {
        fetchAccountFailed: '獲取帳戶資訊失敗，請稍後重試',
        syncHistorySuccess: '同步歷史訂單成功',
        syncHistoryFailed: 'Failed to sync order history. Please ensure the account is connected to the MT server.'
      },
      orderTypes: {
        buyLimit: '買入限價',
        sellLimit: '賣出限價',
        buyStop: '買入止損',
        sellStop: 'Sell stop'
      },
      balanceRecord: {
        deposit: '💰 入金',
        withdraw: '💸 出金',
        depositIconText: '💰 入金',
        withdrawIconText: '💸 出金'
      },
      syncHistory: {
        title: '同步歷史訂單',
        content: '確定要從MT伺服器同步過去一年的歷史訂單嗎？這可能需要一些時間。',
        ok: 'Sync'
      },
      actions: {
        enableAccount: '啟用帳戶',
        disableAccount: '停用帳戶',
        deleteAccount: '刪除帳戶',
        deleteConfirm: '驗證並刪除',
        deleteWarning: '此操作不可逆轉。所有帳戶資料（交易記錄、分析等）將被永久刪除。',
        deletePasswordHint: '請輸入MT交易密碼或唯讀密碼以驗證：',
        deletePasswordPlaceholder: 'MT交易密碼/唯讀密碼',
        syncHistory: 'Sync history'
      },
      status: {
        disabled: '已停用',
        connected: '已連接',
        connecting: '連線中',
        disconnected: '已斷線',
        error: 'Error'
      },
      accountType: {
        real: '真實',
        demo: 'Demo'
      },
      mode: {
        investor: '投資者模式',
        trader: 'Trader mode'
      },
      connected: '已連接',
        lastConnected: '{{time}}',
        leverage: '槓桿 {{leverage}}x',
      cards: {
        balance: '餘額',
        equity: '淨值',
        floatingProfit: '浮動盈虧',
        marginUsed: '已用保證金',
        marginFree: '可用保證金',
        marginLevel: '保證金比例',
        credit: 'Credit'
      }
    },
    report: {
      title: '交易報告',
      titleShort: '報告',
      generate: '生成報告',
      goToAISettings: '前往 AI 設定 →',
      aiAnalysis: 'AI 分析',
      symbolPnL: '品種盈虧',
      direction: '多空分析',
      directionLong: '做多',
      directionShort: '做空',
      tradeDistribution: '盈虧分佈',
      drawdownOverlay: '權益曲線 + 回撤',
      drawdownEvents: '回撤事件',
      recovered: '已恢復',
      winRateTrend: '月度勝率趨勢',
      periods: {
        week: '本週',
        month: '本月',
        quarter: '本季度',
        year: '今年',
      },
      sections: {
        summary: '總體評價',
        findings: '關鍵發現',
        recommendations: '改進建議',
      },
    },
  }
} as const;

export default accounts;
