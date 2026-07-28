// Auto-generated from proto/ant/v1/i18n/strategy_backtest_run_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyBacktestRun = {
  "strategy": {
    "backtestRun": {
      "trades": {
        "reasons": {
          "end_of_test": "测试结束",
          "expired": "已過期",
          "margin_call": "保证金不足",
          "signal": "信號(用於下單)",
          "sl": "止損",
          "tp": "止盈"
        },
        "closePrice": "平倉價",
        "closeTime": "平仓時間",
        "commission": "手續費",
        "empty": "无交易紀錄",
        "loadFailed": "載入訂單明細失敗",
        "openPrice": "開倉價",
        "openTime": "开仓時間",
        "pnl": "盈亏",
        "reason": "平仓原因",
        "side": "方向",
        "sideBuy": "买入",
        "sideSell": "卖出",
        "summary": "{{count}} 笔交易 · {{wins}} 胜 / {{losses}} 负 · 净盈亏 {{pnl}}",
        "ticket": "訂單號",
        "title": "訂單明細",
        "volume": "手數"
      },
      "actions": {
        "cancel": "取消"
      },
      "fields": {
        "error": "錯誤",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普比率",
        "status": "狀態"
      },
      "hints": {
        "canceling": "正在取消回測",
        "queued": "回測正在排队",
        "running": "回測執行中"
      },
      "metrics": {
        "annualReturn": "年化收益",
        "equityCurvePoints": "淨值曲線資料点",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普比率",
        "totalReturn": "總收益",
        "totalTrades": "交易次數",
        "winRate": "勝率"
      },
      "status": {
        "canceled": "已取消",
        "canceling": "取消中",
        "completed": "已完成",
        "ended": "已结束",
        "failed": "失敗",
        "queued": "佇列中",
        "running": "運行中"
      },
      "title": "回測執行"
    }
  }
} as const;
export default StrategyBacktestRun;
