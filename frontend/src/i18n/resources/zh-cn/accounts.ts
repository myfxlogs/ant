const accounts = {
  accounts: {
    title: '我的账户',
    subtitle: '管理您的 MT4/MT5 交易账户',
    bindNew: '绑定新账户',
    bind: {
      title: '绑定 MT 账户',
      errorModal: {
        title: 'Binding failed'
      },
      step1: {
        title: '选择平台和经纪商',
        subtitle: 'Select your trading platform and search for your broker'
      },
      step2: {
        title: '输入账户信息',
        subtitle: 'Enter your trading account and password'
      },
      step3: {
        title: '验证并确认',
        subtitle: 'Verify credentials and confirm to complete'
      },
      fields: {
        platform: '交易平台',
        brokerName: '经纪商名称',
        company: '选择公司',
        server: '服务器',
        tradingAccount: '交易账号',
        password: '密码'
      },
      placeholders: {
        brokerName: '输入经纪商名称，如：XMGlobal、ICMarkets',
        company: '请选择经纪商公司',
        server: '请选择服务器',
        tradingAccount: '输入交易账号',
        password: 'Enter password'
      },
      labels: {
        serverCount: '{{count}} servers'
      },
      actions: {
        search: '搜索',
        verifyAccount: '验证账户',
        confirmBind: '确认绑定',
        retryVerify: 'Retry'
      },
      passwordHint: '密码将通过 HTTPS 加密传输，后端使用 Argon2id 哈希存储不可回逆',
      summary: {
        broker: '经纪商',
        server: '服务器',
        platform: '交易平台',
        tradingAccount: '交易账号',
        password: '密码',
        verified: '账户验证通过',
        balance: '余额',
        equity: '净值',
        margin: '已用保证金',
        freeMargin: '可用保证金',
        leverage: '杠杆',
        currency: 'Currency'
      },
      messages: {
        enterBrokerName: '请输入经纪商名称',
        foundBrokers: '找到 {{count}} 个经纪商',
        noBrokersFound: '未找到匹配的经纪商，请检查名称',
        searchFailed: '搜索失败，请稍后重试',
        selectServer: '请选择服务器',
        enterTradingAccount: '请输入交易账号',
        enterPassword: '请输入密码',
        noAccessHosts: '无可用服务器地址',
        verifyFailed: '账户验证失败',
        bindSuccess: '账户绑定成功',
        bindFailed: '账户绑定失败',
        loginDigitsOnly: 'Trading account must contain only digits'
      },
      errors: {
        brokerUnavailable: '连接服务器错误或者密码不正确',
        invalidCredentials: '账号或密码错误，未找到该交易账户',
        connectionFailed: '无法连接到经纪商服务器，请检查网络',
        timeout: 'Connection timed out, please try again later'
      }
    },
    empty: {
      title: '暂无绑定账户',
      subtitle: 'Click the button below to bind your MT4/MT5 trading account'
    },
    legend: {
      title: '图例:',
      connected: '已连接',
      connecting: '连接中',
      disconnectedOrError: '已断开/错误',
      disabled: '已停用'
    },
    messages: {
      disabledSuccess: '账户停用成功',
      connectingMtServer: '正在连接 MT 服务器',
      enabledSuccess: '账户启用成功',
      fetchListFailed: '获取账户列表失败',
      fetchAccountFailed: '获取账户信息失败',
      createdSuccess: '账户创建成功',
      createFailed: '创建账户失败',
      connectSuccess: '连接成功',
      connectFailed: '连接失败',
      disconnectFailed: '断开连接失败',
      disableFailed: '停用账户失败',
      deleted: '账户已删除',
      deleteFailed: '删除失败',
      enableFailed: 'Failed to enable account'
    },
    analytics: {
      monthlyAnalysis: {
        title: '月度分析',
        chartMainTitle: '每月收益（{{metric}}）',
        metrics: {
          change: '变化',
          profit: '盈亏',
          lots: '手数',
          pips: 'Pips'
        },
        focusedValue: '{{period}} · {{metric}}：{{value}}',
        bonus: {
          chartRiskTitle: 'Bonus：{{month}} 各品种风险回报比（盈利因子）。',
          chartPopularTitle: `{{month}}'s currency popularity.`,
          chartHoldingTitle: `{{month}}'s average holding time.`,
          legendBulls: '买入侧',
          legendShortTerm: '卖出侧',
          sliceOther: '其他',
          emptyCharts: '该月无成交',
          popularityShare: 'Lot volume share'
        }
      },
      monthlyDetail: {
        metricsTitle: '月度指标',
        symbolPnLTitle: '品种盈亏',
        holdingTitle: '持仓时长',
        riskRewardTitle: '奖励:风险比率',
        popularityTitle: '货币流行度',
        long: '做多',
        short: '做空',
        fields: {
          netReturn: '净收益',
          totalTrades: '总笔数',
          winRate: '胜率',
          profitFactor: '盈亏比',
          bestTrade: '最优单笔',
          worstTrade: '最差单笔',
          averageHours: '平均',
          medianHours: '中位',
          maxHours: '最长',
          minHours: '最短',
        },
      },
      chartType: {
        equity: '净值',
        balance: '余额',
        profit: '盈亏'
      },
      chartPeriod: {
        day: '今日',
        week: '本周',
        month: '本月',
        year: '本年',
        all: 'All'
      },
      chartSeries: {
        equity: '净值',
        balance: '余额',
        profit: '盈亏',
        tradeCount: '次数'
      },
      empty: {
        equityCurve: '暂无净值曲线数据',
        monthlyProfit: '暂无月度盈亏数据',
        symbolDistribution: '暂无品种数据',
        dailyPnL: '暂无每日盈亏数据',
        hourly: 'No time-of-day analysis data'
      },
      monthlyProfitTitle: '月度盈亏',
      advancedStatsTitle: '高级统计',
      symbolDistributionTitle: '品种分布',
      dailyPnLTitle: '📅 每日盈亏',
      hourlyTitle: '⏰ 时段分析',
      advancedTabs: {
        hourly: '按小时',
        daily: 'Daily'
      },
      timeDetail: {
        lots: '手数',
        trades: '次数',
        profitAmount: '盈亏金额',
        balance: '余额',
        profitFactor: '盈亏比',
        maxFloatingLossAmount: '最大浮亏金额',
        maxFloatingLossRatio: '最大浮亏比例',
        maxFloatingProfitAmount: '最大浮盈金额',
        maxFloatingProfitRatio: 'Max floating profit ratio'
      },
      stats: {
        winRate: '胜率',
        profitFactor: '盈亏比',
        maxDrawdown: '最大回撤',
        totalTrades: '总交易数',
        avgProfit: '平均盈利',
        avgLoss: '平均亏损',
        avgHolding: '平均持仓',
        consecutiveWinsLosses: '连胜/连败',
        sharpe: '夏普比率',
        sortino: '索提诺',
        calmar: '卡尔马',
        largestWin: '最大盈利',
        largestLoss: '最大亏损',
        avgDailyReturn: '日均收益',
        volatility: '波动率',
        netProfit: '净利润',
        totalDeposit: '入金',
        totalWithdrawal: '出金',
        netDeposit: 'Net deposit'
      }
    },
    card: {
      status: {
        disabled: '已停用',
        connected: '已连接',
        connecting: '连接中',
        disconnected: '已断开',
        error: 'Error'
      },
      fields: {
        balance: '余额',
        equity: '净值',
        broker: '经纪商',
        server: '服务器'
      },
      actions: {
        positions: '持仓',
        orders: '订单',
        details: 'Details'
      },
      deleteConfirm: {
        title: '确定删除此账户？',
        content: 'This action cannot be undone'
      }
    },
    disabled: {
      title: '已停用的账户',
      table: {
        account: '账号',
        type: '类型',
        broker: '经纪商',
        balance: '余额',
        equity: '净值',
        actions: 'Actions'
      },
      confirmDelete: {
        title: '确定删除此账户？',
        content: 'This action cannot be undone'
      },
      mobile: {
        balanceLabel: '余额: ',
        equityLabel: 'Equity: '
      }
    },
    tradeTabs: {
      positionsWithCount: '持仓订单 ({{count}})',
      pendingWithCount: '挂单 ({{count}})',
      historyWithCount: '历史订单 ({{count}})',
      emptyPositions: '暂无持仓',
      emptyHistory: '暂无历史订单',
      syncHistory: '同步历史',
      table: {
        orderId: '订单号',
        symbol: '品种',
        side: '方向',
        type: '类型',
        volume: '手数',
        openPrice: '开仓价',
        currentPrice: '当前价',
        pendingPrice: '挂单价格',
        closePrice: '平仓价',
        profit: '盈亏',
        openTime: '开仓时间',
        pendingTime: '挂单时间',
        closeTime: 'Close time'
      },
      pagination: {
        total: '{{total}} total'
      }
    },
    edit: {
      title: '编辑账户',
      fields: {
        tradingAccount: '交易账号',
        server: '服务器',
        password: '新密码',
        oldPassword: 'Current password'
      },
      placeholders: {
        newPassword: '输入新密码',
        oldPassword: 'Enter current password'
      },
      messages: {
        enterPassword: '请输入新密码',
        enterOldPassword: '请输入当前密码',
        passwordVerifyFailed: '密码修改失败',
        passwordSaved: 'Password saved'
      }
    },
    detail: {
      messages: {
        fetchAccountFailed: '获取账户信息失败，请稍后重试',
        syncHistorySuccess: '同步历史订单成功',
        syncHistoryFailed: 'Failed to sync order history. Please ensure the account is connected to the MT server.'
      },
      orderTypes: {
        buyLimit: '买入限价',
        sellLimit: '卖出限价',
        buyStop: '买入止损',
        sellStop: 'Sell stop'
      },
      balanceRecord: {
        deposit: '💰 入金',
        withdraw: '💸 出金',
        depositIconText: '💰 入金',
        withdrawIconText: '💸 出金'
      },
      syncHistory: {
        title: '同步历史订单',
        content: '确定要从MT服务器同步过去一年的历史订单吗？这可能需要一些时间。',
        ok: 'Sync'
      },
      actions: {
        enableAccount: '启用账户',
        disableAccount: '停用账户',
        deleteAccount: '删除账户',
        deleteConfirm: '验证并删除',
        deleteWarning: '此操作不可撤销。账户所有数据（交易记录、分析数据等）将被永久删除。',
        deletePasswordHint: '请输入该账户的 MT 交易密码或只读密码进行验证：',
        deletePasswordPlaceholder: 'MT 交易密码 / 只读密码',
        syncHistory: 'Sync history'
      },
      status: {
        disabled: '已停用',
        connected: '已连接',
        connecting: '连接中',
        disconnected: '已断开',
        error: 'Error'
      },
      accountType: {
        real: '真实',
        demo: 'Demo'
      },
      mode: {
        investor: '投资者模式',
        trader: 'Trader mode'
      },
      connected: '已连接',
        lastConnected: '{{time}}',
        leverage: '杠杆 {{leverage}}x',
      cards: {
        balance: '余额',
        equity: '净值',
        floatingProfit: '浮动盈亏',
        marginUsed: '已用保证金',
        marginFree: '可用保证金',
        marginLevel: '保证金比例',
        credit: 'Credit'
      }
    },
    report: {
      title: '交易报告',
      titleShort: '报告',
      generate: '生成报告',
      goToAISettings: '前往 AI 设置 →',
      aiAnalysis: 'AI 分析',
      symbolPnL: '品种盈亏',
      direction: '多空分析',
      directionLong: '做多',
      directionShort: '做空',
      tradeDistribution: '盈亏分布',
      drawdownOverlay: '权益曲线 + 回撤',
      drawdownEvents: '回撤事件',
      recovered: '已恢复',
      winRateTrend: '月度胜率趋势',
      periods: {
        week: '本周',
        month: '本月',
        quarter: '本季度',
        year: '今年',
      },
      sections: {
        summary: '总体评价',
        findings: '关键发现',
        recommendations: '改进建议',
      },
    },
  }
} as const;

export default accounts;
