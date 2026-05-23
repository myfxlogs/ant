const analytics = {
  analytics: {
    summary: {
      title: 'Analytics Summary',
      periods: {
        today: 'Today',
        week: 'This week',
        month: 'This month',
        year: 'This year',
        all: 'All',
      },
      direction: {
        buy: 'Buy',
        sell: 'Sell',
      },
      profit: {
        win: 'Win',
        loss: 'Loss',
      },
      yearOption: '{{year}}',
      placeholders: {
        selectAccount: 'Select an account',
      },
      sections: {
        equityCurve: 'Equity curve',
        monthlyStats: 'Monthly stats',
      },
      labels: {
        pnl: 'P/L',
      },
      metrics: {
        netProfit: 'Net profit',
        equity: 'Equity',
        balance: 'Balance',
        equityValue: 'Equity value',
      },
      cards: {
        symbolPnlCompare: 'Symbol P/L comparison',
        symbolTradeShare: 'Symbol trade share',
        directionShare: 'Direction share',
        pnlShare: 'P/L share',
        tradeStats: 'Trade stats',
        riskMetrics: 'Risk metrics',
        economicCalendar: 'Economic calendar',
      },
      tradeStats: {
        totalTrades: 'Total trades',
        wins: 'Wins',
        losses: 'Losses',
        winRate: 'Win rate',
        profitFactor: 'Profit factor',
        avgHolding: 'Avg holding',
        maxConsecutiveWins: 'Max consecutive wins',
        maxConsecutiveLosses: 'Max consecutive losses',
        maxHolding: 'Max holding',
        avgVolume: 'Avg volume',
        avgProfit: 'Avg profit',
        avgLoss: 'Avg loss',
      },
      risk: {
        maxDrawdown: 'Max drawdown',
        maxDrawdownPct: 'Max drawdown (%)',
        sharpe: 'Sharpe',
        sortino: 'Sortino',
        volatility: 'Volatility',
        var95: 'Value at Risk (95%)',
      },
      economicCalendar: {
        loading: 'Loading economic calendar...',
        empty: 'No economic events available.',
        actual: 'Actual',
        previous: 'Previous',
        estimate: 'Estimate',
        keyIndicatorsTitle: 'Key macro indicators',
        indicators: {
          CPI: 'Inflation (CPI)',
          UNRATE: 'Unemployment rate',
          FEDFUNDS: 'Fed funds rate',
          GDP: 'Real GDP',
        },
      },
    },
  },
} as const;

export default analytics;
