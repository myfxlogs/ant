// Auto-generated from proto/ant/v1/i18n/strategy_schedule_logs_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyScheduleLogs = {
  "strategy": {
    "scheduleLogs": {
      "action": {
        "cleanup": "清理",
        "register": "註冊",
        "restart": "重启",
        "start": "啟動",
        "stop": "停止"
      },
      "execStatus": {
        "completed": "已完成",
        "failed": "失敗",
        "pending": "待檢測",
        "running": "運行中",
        "skipped": "已略過",
        "stopped": "已停止",
        "success": "成功"
      },
      "execTable": {
        "action": "操作",
        "durationMs": "耗時(ms)",
        "error": "錯誤",
        "execute": "執行",
        "status": "狀態",
        "time": "時間"
      },
      "messages": {
        "missingScheduleId": "缺少排程 ID"
      },
      "operationStatus": {
        "failed": "失敗",
        "running": "運行中",
        "success": "成功"
      },
      "orderSide": {
        "buy": "市價買入",
        "buyLimit": "限價買入",
        "buyStop": "突破買入",
        "buyStopLimit": "限價突破買",
        "close": "平倉",
        "sell": "市價賣出",
        "sellLimit": "限價賣出",
        "sellStop": "突破賣出",
        "sellStopLimit": "卖出止損限价"
      },
      "ordersTable": {
        "closePrice": "平倉價",
        "lots": "手數(Lot)",
        "openPrice": "開倉價",
        "profit": "盈虧",
        "side": "方向",
        "symbol": "品種",
        "ticket": "訂單號",
        "time": "時間"
      },
      "status": {
        "failed": "失敗",
        "success": "成功"
      },
      "summary": {
        "enableCount": "啟用次數",
        "lastError": "最近錯誤",
        "lastRun": "最後運行時間",
        "name": "名稱",
        "status": "狀態",
        "trade": "交易"
      },
      "tabs": {
        "exec": "運行記錄",
        "execLogs": "执行日誌",
        "orderLogs": "訂單日志",
        "orders": "交易記錄"
      },
      "scheduleIdLabel": "排程ID:",
      "title": "記錄",
      "titleWithName": "記錄 - {{name}}"
    }
  }
} as const;
export default StrategyScheduleLogs;
