const analytics = {
  analytics: {
    summary: {
      title: '分析',
      placeholders: {
        selectAccount: '選擇帳戶',
      },
      periods: {
        today: '今日',
        week: '本週',
        month: '本月',
        year: '本年',
        all: '全部',
      },
      sections: {
        equityCurve: '資金曲線',
        monthlyStats: '月度統計',
      },
      labels: {
        pnl: '盈虧',
      },
      metrics: {
        netProfit: '總盈虧',
        equity: '當前持倉',
        balance: '餘額',
        equityValue: '淨值',
      },
      cards: {
        symbolPnlCompare: '品種盈虧對比',
        symbolTradeShare: '品種交易占比',
        directionShare: '買賣方向占比',
        pnlShare: '盈虧占比',
        tradeStats: '交易統計',
        riskMetrics: '風險指標',
      },
      tradeStats: {
        totalTrades: '總交易',
        wins: '盈利',
        losses: '虧損',
        winRate: '勝率',
        profitFactor: '盈虧比',
        avgHolding: '平均持倉',
        maxConsecutiveWins: '連續盈利最多',
        maxConsecutiveLosses: '連續虧損最多',
        maxHolding: '最長持倉',
        avgVolume: '平均手數',
        avgProfit: '平均盈利',
        avgLoss: '平均虧損',
      },
      risk: {
        maxDrawdown: '最大回撤',
        maxDrawdownPct: '回撤比例',
        sharpe: '夏普比率',
        sortino: '索提諾比率',
        volatility: '波動率',
      },
      direction: {
        buy: '買入',
        sell: '賣出',
      },
      profit: {
        win: '盈利',
        loss: '虧損',
      },
      yearOption: '{{year}}年',
    },
  },
} as const;

export default analytics;
