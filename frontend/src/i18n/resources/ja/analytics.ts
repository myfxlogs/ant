// Auto-generated from proto/ant/v1/i18n/analytics_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Analytics = {
  "analytics": {
    "summary": {
      "cards": {
        "directionShare": "方向比率",
        "economicCalendar": "経済カレンダー",
        "pnlShare": "損益比率",
        "riskMetrics": "リスク指標",
        "symbolPnlCompare": "銘柄別損益比較",
        "symbolTradeShare": "銘柄別取引比率",
        "tradeStats": "取引統計"
      },
      "direction": {
        "buy": "買い",
        "sell": "売り"
      },
      "economicCalendar": {
        "actual": "実績",
        "empty": "利用可能な経済イベントはありません。",
        "estimate": "予想",
        "indicators": {
          "CPI": "インフレ率（CPI）",
          "FEDFUNDS": "フェデラルファンド金利",
          "GDP": "実質GDP",
          "UNRATE": "失業率"
        },
        "keyIndicatorsTitle": "主要マクロ経済指標",
        "loading": "経済カレンダーを読み込み中...",
        "previous": "前回"
      },
      "labels": {
        "pnl": "損益"
      },
      "metrics": {
        "balance": "残高",
        "equity": "有効証拠金",
        "equityValue": "有効証拠金額",
        "netProfit": "純利益"
      },
      "periods": {
        "all": "全期間",
        "month": "今月",
        "today": "今日",
        "week": "今週",
        "year": "今年"
      },
      "placeholders": {
        "selectAccount": "口座を選択"
      },
      "profit": {
        "loss": "負け",
        "win": "勝ち"
      },
      "risk": {
        "maxDrawdown": "最大ドローダウン",
        "maxDrawdownPct": "最大ドローダウン(%)",
        "sharpe": "シャープレシオ",
        "sortino": "ソルティノレシオ",
        "var95": "VaR 95%",
        "volatility": "ボラティリティ"
      },
      "sections": {
        "equityCurve": "エクイティカーブ",
        "monthlyStats": "月次統計"
      },
      "title": "分析サマリー",
      "tradeStats": {
        "avgHolding": "平均保有時間",
        "avgLoss": "平均損失",
        "avgProfit": "平均利益",
        "avgVolume": "平均ロット",
        "losses": "負け",
        "maxConsecutiveLosses": "最大連敗数",
        "maxConsecutiveWins": "最大連勝数",
        "maxHolding": "最大保有時間",
        "profitFactor": "プロフィットファクター",
        "totalTrades": "取引回数",
        "winRate": "勝率",
        "wins": "勝ち"
      },
      "yearOption": "{{year}}年"
    }
  }
} as const;
export default Analytics;
