// Auto-generated from proto/ant/v1/i18n/analytics_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Analytics = {
  "analytics": {
    "summary": {
      "economicCalendar": {
        "indicators": {
          "CPI": "通胀率（CPI）",
          "FEDFUNDS": "联邦基金利率",
          "GDP": "实际 GDP",
          "UNRATE": "失业率"
        },
        "actual": "实际值",
        "empty": "暂无经济事件数据。",
        "estimate": "预期值",
        "keyIndicatorsTitle": "关键宏观指标",
        "loading": "正在加载经济日历...",
        "previous": "前值"
      },
      "cards": {
        "directionShare": "买卖方向占比",
        "economicCalendar": "经济日历",
        "pnlShare": "盈亏占比",
        "riskMetrics": "风险指标",
        "symbolPnlCompare": "品种盈亏对比",
        "symbolTradeShare": "品种交易占比",
        "tradeStats": "交易统计"
      },
      "direction": {
        "buy": "买入",
        "sell": "卖出"
      },
      "labels": {
        "pnl": "盈亏"
      },
      "metrics": {
        "balance": "余额",
        "equity": "当前持仓",
        "equityValue": "净值",
        "netProfit": "总盈亏"
      },
      "periods": {
        "all": "全部",
        "month": "本月",
        "today": "今日",
        "week": "本周",
        "year": "本年"
      },
      "placeholders": {
        "selectAccount": "选择账户"
      },
      "profit": {
        "loss": "亏损",
        "win": "盈利"
      },
      "risk": {
        "maxDrawdown": "最大回撤",
        "maxDrawdownPct": "回撤比例",
        "sharpe": "夏普比率",
        "sortino": "索提诺比率",
        "var95": "VaR 95%",
        "volatility": "波动率"
      },
      "sections": {
        "equityCurve": "资金曲线",
        "monthlyStats": "月度统计"
      },
      "tradeStats": {
        "avgHolding": "平均持仓",
        "avgLoss": "平均亏损",
        "avgProfit": "平均盈利",
        "avgVolume": "平均手数",
        "losses": "亏损",
        "maxConsecutiveLosses": "连续亏损最多",
        "maxConsecutiveWins": "连续盈利最多",
        "maxHolding": "最长持仓",
        "profitFactor": "盈亏比",
        "totalTrades": "总交易",
        "winRate": "胜率",
        "wins": "盈利"
      },
      "title": "分析",
      "yearOption": "{{year}}年"
    }
  }
} as const;
export default Analytics;
