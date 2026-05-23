const analytics = {
  analytics: {
    summary: {
      title: '分析サマリー',
      periods: {
        today: '今日',
        week: '今週',
        month: '今月',
        year: '今年',
        all: '全期間',
      },
      direction: {
        buy: '買い',
        sell: '売り',
      },
      profit: {
        win: '勝ち',
        loss: '負け',
      },
      yearOption: '{{year}}',
      placeholders: {
        selectAccount: '口座を選択',
      },
      sections: {
        equityCurve: 'エクイティカーブ',
        monthlyStats: '月次統計',
      },
      labels: {
        pnl: '損益',
      },
      metrics: {
        netProfit: '純利益',
        equity: '有効証拠金',
        balance: '残高',
        equityValue: '有効証拠金額',
      },
      cards: {
        symbolPnlCompare: '銘柄別損益比較',
        symbolTradeShare: '銘柄別取引比率',
        directionShare: '方向比率',
        pnlShare: '損益比率',
        tradeStats: '取引統計',
        riskMetrics: 'リスク指標',
        economicCalendar: '経済カレンダー',
      },
      tradeStats: {
        totalTrades: '取引回数',
        wins: '勝ち',
        losses: '負け',
        winRate: '勝率',
        profitFactor: 'プロフィットファクター',
        avgHolding: '平均保有時間',
        maxConsecutiveWins: '最大連勝数',
        maxConsecutiveLosses: '最大連敗数',
        maxHolding: '最大保有時間',
        avgVolume: '平均ロット',
        avgProfit: '平均利益',
        avgLoss: '平均損失',
      },
      risk: {
        maxDrawdown: '最大ドローダウン',
        maxDrawdownPct: '最大ドローダウン(%)',
        sharpe: 'シャープレシオ',
        sortino: 'ソルティノレシオ',
        volatility: 'ボラティリティ',
        var95: 'VaR 95%',
      },
      economicCalendar: {
        loading: '経済カレンダーを読み込み中...',
        empty: '利用可能な経済イベントはありません。',
        actual: '実績',
        previous: '前回',
        estimate: '予想',
        keyIndicatorsTitle: '主要マクロ経済指標',
        indicators: {
          CPI: 'インフレ率（CPI）',
          UNRATE: '失業率',
          FEDFUNDS: 'フェデラルファンド金利',
          GDP: '実質GDP',
        },
      },
    },
  },
} as const;

export default analytics;
