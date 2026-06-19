// Auto-generated from proto/ant/v1/i18n/strategy_schedule_logs_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const ScheduleLogs = {
  "strategy": {
    "scheduleLogs": {
      "action": {
        "restart": "Khởi Động Lại",
        "start": "Bắt Đầu",
        "stop": "Dừng"
      },
      "execStatus": {
        "completed": "Hoàn tất",
        "failed": "Thất Bại",
        "pending": "Chưa kiểm tra",
        "running": "Đang chạy",
        "skipped": "Đã Bỏ Qua"
      },
      "execTable": {
        "action": "Hành động",
        "durationMs": "Thời lượng (ms)",
        "error": "Lỗi",
        "execute": "Thực thi",
        "status": "Trạng thái",
        "time": "Thời gian"
      },
      "messages": {
        "missingScheduleId": "Thiếu ID lịch"
      },
      "operationStatus": {
        "failed": "Thất Bại",
        "running": "Đang chạy",
        "success": "Thành Công"
      },
      "orderSide": {
        "buy": "Mua thị trường",
        "buyLimit": "Mua giới hạn",
        "buyStop": "Mua Stop",
        "buyStopLimit": "Mua Stop Limit",
        "close": "Đóng lệnh",
        "sell": "Bán thị trường",
        "sellLimit": "Bán giới hạn",
        "sellStop": "Bán stop",
        "sellStopLimit": "Bán Stop Limit"
      },
      "ordersTable": {
        "closePrice": "Giá đóng",
        "lots": "Khối lượng (Lot)",
        "openPrice": "Giá mở",
        "profit": "Lãi/Lỗ",
        "side": "Hướng",
        "symbol": "Mã",
        "ticket": "Mã lệnh",
        "time": "Thời gian"
      },
      "scheduleIdLabel": "ID lịch chạy:",
      "status": {
        "failed": "Thất Bại",
        "success": "Thành Công"
      },
      "summary": {
        "enableCount": "Số lần bật",
        "lastError": "Lỗi Cuối",
        "lastRun": "Lần chạy gần nhất",
        "name": "Tên",
        "status": "Trạng thái",
        "trade": "Giao dịch"
      },
      "tabs": {
        "exec": "Lần chạy",
        "execLogs": "执行日志",
        "orderLogs": "Nhật Ký Lệnh",
        "orders": "Lệnh"
      },
      "title": "Nhật ký",
      "titleWithName": "Nhật ký - {{name}}"
    }
  }
} as const;
export default ScheduleLogs;
