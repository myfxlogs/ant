// Auto-generated from proto/ant/v1/i18n/strategy_schedule_logs_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyScheduleLogs = {
  "strategy": {
    "scheduleLogs": {
      "action": {
        "cleanup": "Cleanup",
        "register": "Register",
        "restart": "Restart",
        "start": "Start",
        "stop": "Stop"
      },
      "execStatus": {
        "completed": "Completed",
        "failed": "Failed",
        "pending": "Pending",
        "running": "Running",
        "skipped": "Skipped",
        "stopped": "Stopped",
        "success": "Success"
      },
      "execTable": {
        "action": "Action",
        "durationMs": "Duration (ms)",
        "error": "Error",
        "execute": "Execute",
        "status": "Status",
        "time": "Time"
      },
      "messages": {
        "missingScheduleId": "Missing schedule ID"
      },
      "operationStatus": {
        "failed": "Failed",
        "running": "Running",
        "success": "Success"
      },
      "orderSide": {
        "buy": "Market buy",
        "buyLimit": "Buy limit",
        "buyStop": "Buy stop",
        "buyStopLimit": "Buy stop limit",
        "close": "Close",
        "sell": "Market sell",
        "sellLimit": "Sell limit",
        "sellStop": "Sell stop",
        "sellStopLimit": "Sell stop limit"
      },
      "ordersTable": {
        "closePrice": "Close price",
        "lots": "Lots",
        "magic": "Magic",
        "openPrice": "Open price",
        "profit": "P/L",
        "side": "Side",
        "symbol": "Symbol",
        "ticket": "Ticket",
        "time": "Time"
      },
      "status": {
        "failed": "Failed",
        "success": "Success"
      },
      "summary": {
        "enableCount": "Enabled count",
        "lastError": "Last error",
        "lastRun": "Last run",
        "name": "Name",
        "status": "Status",
        "trade": "Trade"
      },
      "tabs": {
        "exec": "Executions",
        "execLogs": "Execution Logs",
        "orderLogs": "Order Logs",
        "orders": "Orders"
      },
      "scheduleIdLabel": "Schedule ID:",
      "title": "Schedule logs",
      "titleWithName": "Schedule logs: {{name}}"
    }
  }
} as const;
export default StrategyScheduleLogs;
