// Auto-generated from proto/ant/v1/i18n/strategy_schedule_logs_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const ScheduleLogs = {
  "strategy": {
    "scheduleLogs": {
      "action": {
        "restart": "重启",
        "start": "启动",
        "stop": "停止"
      },
      "execStatus": {
        "completed": "已完成",
        "failed": "失败",
        "pending": "待检测",
        "running": "运行中",
        "skipped": "已跳过"
      },
      "execTable": {
        "action": "操作",
        "durationMs": "耗时(ms)",
        "error": "错误",
        "execute": "执行",
        "status": "状态",
        "time": "时间"
      },
      "messages": {
        "missingScheduleId": "缺少调度 ID"
      },
      "operationStatus": {
        "failed": "失败",
        "running": "运行中",
        "success": "成功"
      },
      "orderSide": {
        "buy": "市价买入",
        "buyLimit": "限价买入",
        "buyStop": "突破买入",
        "buyStopLimit": "限价突破买",
        "close": "平仓",
        "sell": "市价卖出",
        "sellLimit": "限价卖出",
        "sellStop": "突破卖出",
        "sellStopLimit": "卖出止损限价"
      },
      "ordersTable": {
        "closePrice": "平仓价",
        "lots": "手数(Lot)",
        "openPrice": "开仓价",
        "profit": "盈亏",
        "side": "方向",
        "symbol": "品种",
        "ticket": "订单号",
        "time": "时间"
      },
      "scheduleIdLabel": "调度ID:",
      "status": {
        "failed": "失败",
        "success": "成功"
      },
      "summary": {
        "enableCount": "启用次数",
        "lastError": "最近错误",
        "lastRun": "最后运行时间",
        "name": "名称",
        "status": "状态",
        "trade": "交易"
      },
      "tabs": {
        "exec": "运行记录",
        "execLogs": "执行日志",
        "orderLogs": "订单日志",
        "orders": "交易记录"
      },
      "title": "记录",
      "titleWithName": "记录 - {{name}}"
    }
  }
} as const;
export default ScheduleLogs;
