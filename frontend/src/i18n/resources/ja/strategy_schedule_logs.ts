// Auto-generated from proto/ant/v1/i18n/strategy_schedule_logs_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyScheduleLogs = {
  "strategy": {
    "scheduleLogs": {
      "action": {
        "cleanup": "クリーンアップ",
        "register": "登録",
        "restart": "再起動",
        "start": "启动",
        "stop": "停止"
      },
      "execStatus": {
        "completed": "完了",
        "failed": "失败",
        "pending": "未評価",
        "running": "実行中",
        "skipped": "スキップ",
        "stopped": "停止",
        "success": "成功"
      },
      "execTable": {
        "action": "操作",
        "durationMs": "所要(ms)",
        "error": "エラー",
        "execute": "実行",
        "status": "状態",
        "time": "時間"
      },
      "messages": {
        "missingScheduleId": "スケジュールID不足"
      },
      "operationStatus": {
        "failed": "失败",
        "running": "実行中",
        "success": "成功"
      },
      "orderSide": {
        "buy": "成行買い",
        "buyLimit": "指値買い",
        "buyStop": "買いストップ",
        "buyStopLimit": "買いストップリミット",
        "close": "決済",
        "sell": "成行売り",
        "sellLimit": "指値売り",
        "sellStop": "逆指値売り",
        "sellStopLimit": "売りストップリミット"
      },
      "ordersTable": {
        "closePrice": "決済値",
        "lots": "ロット",
        "magic": "Magic",
        "openPrice": "建値",
        "profit": "損益",
        "side": "方向",
        "symbol": "銘柄",
        "ticket": "チケット",
        "time": "時間"
      },
      "status": {
        "failed": "失败",
        "success": "成功"
      },
      "summary": {
        "enableCount": "有効化回数",
        "lastError": "最終エラー",
        "lastRun": "最終実行",
        "name": "名称",
        "status": "状態",
        "trade": "取引"
      },
      "tabs": {
        "exec": "実行履歴",
        "execLogs": "执行日志",
        "orderLogs": "注文ログ",
        "orders": "取引履歴"
      },
      "scheduleIdLabel": "スケジュールID:",
      "title": "記録",
      "titleWithName": "記録 - {{name}}"
    }
  }
} as const;
export default StrategyScheduleLogs;
