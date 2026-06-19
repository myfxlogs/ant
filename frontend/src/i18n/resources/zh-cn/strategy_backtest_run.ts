// Auto-generated from proto/ant/v1/i18n/strategy_backtest_run_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyBacktestRun = {
  "strategy": {
    "backtestRun": {
      "actions": {
        "cancel": "取消"
      },
      "fields": {
        "error": "错误",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普比率",
        "status": "状态"
      },
      "hints": {
        "canceling": "正在取消回测",
        "queued": "回测正在排队",
        "running": "回测运行中"
      },
      "metrics": {
        "annualReturn": "年化收益",
        "equityCurvePoints": "净值曲线数据点",
        "maxDrawdown": "最大回撤",
        "sharpe": "夏普比率",
        "totalReturn": "总收益",
        "totalTrades": "交易次数",
        "winRate": "胜率"
      },
      "status": {
        "canceled": "已取消",
        "canceling": "取消中",
        "completed": "已完成",
        "ended": "已结束",
        "failed": "失败",
        "queued": "排队中",
        "running": "运行中"
      },
      "title": "回测运行",
      "trades": {
        "closePrice": "平仓价",
        "closeTime": "平仓时间",
        "commission": "手续费",
        "empty": "无交易记录",
        "loadFailed": "加载订单明细失败",
        "openPrice": "开仓价",
        "openTime": "开仓时间",
        "pnl": "盈亏",
        "reason": "平仓原因",
        "reasons": {
          "end_of_test": "测试结束",
          "expired": "已过期",
          "margin_call": "保证金不足",
          "signal": "信号(用于下单)",
          "sl": "止损",
          "tp": "止盈"
        },
        "side": "方向",
        "sideBuy": "买入",
        "sideSell": "卖出",
        "summary": "{{count}} 笔交易 · {{wins}} 胜 / {{losses}} 负 · 净盈亏 {{pnl}}",
        "ticket": "订单号",
        "title": "订单明细",
        "volume": "手数"
      }
    }
  }
} as const;
export default StrategyBacktestRun;
