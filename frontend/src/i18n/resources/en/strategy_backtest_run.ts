// Auto-generated from proto/ant/v1/i18n/strategy_backtest_run_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyBacktestRun = {
  "strategy": {
    "backtestRun": {
      "actions": {
        "cancel": "Cancel"
      },
      "fields": {
        "error": "Error",
        "maxDrawdown": "Max Drawdown",
        "sharpe": "Sharpe",
        "status": "Status"
      },
      "hints": {
        "canceling": "Canceling backtest",
        "queued": "Backtest is queued",
        "running": "Backtest is running"
      },
      "metrics": {
        "annualReturn": "Annual return",
        "equityCurvePoints": "Equity curve points",
        "maxDrawdown": "Max drawdown",
        "sharpe": "Sharpe ratio",
        "totalReturn": "Total return",
        "totalTrades": "Total trades",
        "winRate": "Win rate"
      },
      "status": {
        "canceled": "Canceled",
        "canceling": "Canceling",
        "completed": "Completed",
        "ended": "Ended",
        "failed": "Failed",
        "queued": "Queued",
        "running": "Running"
      },
      "title": "Backtest run",
      "trades": {
        "closePrice": "Close price",
        "closeTime": "Close time",
        "commission": "Commission",
        "empty": "No trades recorded",
        "loadFailed": "Failed to load order details",
        "openPrice": "Open price",
        "openTime": "Open time",
        "pnl": "P&L",
        "reason": "Close reason",
        "reasons": {
          "end_of_test": "End of test",
          "expired": "Expired",
          "margin_call": "Margin call",
          "signal": "Signal",
          "sl": "Stop loss",
          "tp": "Take profit"
        },
        "side": "Side",
        "sideBuy": "Buy",
        "sideSell": "Sell",
        "summary": "{{count}} trades · {{wins}} wins / {{losses}} losses · net P&L {{pnl}}",
        "ticket": "Ticket",
        "title": "Order details",
        "volume": "Volume"
      }
    }
  }
} as const;
export default StrategyBacktestRun;
