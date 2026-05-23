const analytics = {
  analytics: {
    summary: {
      title: '分析',
      placeholders: {
        selectAccount: '选择账户',
      },
      periods: {
        today: '今日',
        week: '本周',
        month: '本月',
        year: '本年',
        all: '全部',
      },
      sections: {
        equityCurve: '资金曲线',
        monthlyStats: '月度统计',
      },
      labels: {
        pnl: '盈亏',
      },
      metrics: {
        netProfit: '总盈亏',
        equity: '当前持仓',
        balance: '余额',
        equityValue: '净值',
      },
      cards: {
        symbolPnlCompare: '品种盈亏对比',
        symbolTradeShare: '品种交易占比',
        directionShare: '买卖方向占比',
        pnlShare: '盈亏占比',
        tradeStats: '交易统计',
        riskMetrics: '风险指标',
        economicCalendar: '经济日历',
      },
      tradeStats: {
        totalTrades: '总交易',
        wins: '盈利',
        losses: '亏损',
        winRate: '胜率',
        profitFactor: '盈亏比',
        avgHolding: '平均持仓',
        maxConsecutiveWins: '连续盈利最多',
        maxConsecutiveLosses: '连续亏损最多',
        maxHolding: '最长持仓',
        avgVolume: '平均手数',
        avgProfit: '平均盈利',
        avgLoss: '平均亏损',
      },
      risk: {
        maxDrawdown: '最大回撤',
        maxDrawdownPct: '回撤比例',
        sharpe: '夏普比率',
        sortino: '索提诺比率',
        volatility: '波动率',
        var95: 'VaR 95%',
      },
      direction: {
        buy: '买入',
        sell: '卖出',
      },
      profit: {
        win: '盈利',
        loss: '亏损',
      },
      yearOption: '{{year}}年',
      economicCalendar: {
        loading: '正在加载经济日历...',
        empty: '暂无经济事件数据。',
        actual: '实际值',
        previous: '前值',
        estimate: '预期值',
        keyIndicatorsTitle: '关键宏观指标',
        indicators: {
          CPI: '通胀率（CPI）',
          UNRATE: '失业率',
          FEDFUNDS: '联邦基金利率',
          GDP: '实际 GDP',
        },
      },
    },
  },
} as const;

export default analytics;
