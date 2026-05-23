const analytics = {
  analytics: {
    summary: {
      title: 'Phân tích',
      placeholders: {
        selectAccount: 'Chọn tài khoản',
      },
      periods: {
        today: 'Hôm nay',
        week: 'Tuần này',
        month: 'Tháng này',
        year: 'Năm nay',
        all: 'Tất cả',
      },
      sections: {
        equityCurve: 'Đường vốn',
        monthlyStats: 'Thống kê theo tháng',
      },
      labels: {
        pnl: 'Lãi/Lỗ',
      },
      metrics: {
        netProfit: 'Tổng lãi/lỗ',
        equity: 'Vốn',
        balance: 'Số dư',
        equityValue: 'Vốn',
      },
      cards: {
        symbolPnlCompare: 'Lãi/Lỗ theo mã',
        symbolTradeShare: 'Tỷ lệ giao dịch theo mã',
        directionShare: 'Tỷ lệ mua/bán',
        pnlShare: 'Tỷ lệ lãi/lỗ',
        tradeStats: 'Thống kê giao dịch',
        riskMetrics: 'Chỉ số rủi ro',
      },
      tradeStats: {
        totalTrades: 'Tổng số lệnh',
        wins: 'Lãi',
        losses: 'Lỗ',
        winRate: 'Tỷ lệ thắng',
        profitFactor: 'Hệ số lợi nhuận',
        avgHolding: 'Thời gian giữ TB',
        maxConsecutiveWins: 'Chuỗi lãi dài nhất',
        maxConsecutiveLosses: 'Chuỗi lỗ dài nhất',
        maxHolding: 'Thời gian giữ dài nhất',
        avgVolume: 'Khối lượng TB',
        avgProfit: 'Lãi TB',
        avgLoss: 'Lỗ TB',
      },
      risk: {
        maxDrawdown: 'Drawdown tối đa',
        maxDrawdownPct: 'Tỷ lệ drawdown',
        sharpe: 'Sharpe',
        sortino: 'Sortino',
        volatility: 'Biến động',
      },
      direction: {
        buy: 'Mua',
        sell: 'Bán',
      },
      profit: {
        win: 'Lãi',
        loss: 'Lỗ',
      },
      yearOption: '{{year}}',
    },
  },
} as const;

export default analytics;
