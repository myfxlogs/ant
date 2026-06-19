// Auto-generated from proto/ant/v1/i18n/strategy_backtest_run_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const BacktestRun = {
  "strategy": {
    "backtestRun": {
      "actions": {
        "cancel": "Hủy"
      },
      "fields": {
        "error": "Lỗi",
        "maxDrawdown": "Sụt Giảm Tối Đa",
        "sharpe": "Sharpe",
        "status": "Trạng thái"
      },
      "hints": {
        "canceling": "Đang hủy backtest",
        "queued": "Backtest đang chờ",
        "running": "Backtest đang chạy"
      },
      "metrics": {
        "annualReturn": "Lợi nhuận năm",
        "equityCurvePoints": "Điểm đường vốn",
        "maxDrawdown": "Sụt giảm tối đa",
        "sharpe": "Sharpe",
        "totalReturn": "Tổng lợi nhuận",
        "totalTrades": "Số lệnh",
        "winRate": "Tỷ lệ thắng"
      },
      "status": {
        "canceled": "Đã Hủy",
        "canceling": "Đang Hủy",
        "completed": "Hoàn tất",
        "ended": "Đã Kết Thúc",
        "failed": "Thất Bại",
        "queued": "Đang Chờ",
        "running": "Đang chạy"
      },
      "title": "Chạy Backtest",
      "trades": {
        "closePrice": "Giá đóng",
        "closeTime": "Giờ Đóng",
        "commission": "Hoa Hồng",
        "empty": "Không có giao dịch",
        "loadFailed": "Tải chi tiết lệnh thất bại",
        "openPrice": "Giá mở",
        "openTime": "Giờ Mở",
        "pnl": "Lãi/Lỗ",
        "reason": "Lý Do Đóng",
        "reasons": {
          "end_of_test": "Kết Thúc Kiểm Tra",
          "expired": "Hết Hạn",
          "margin_call": "Gọi Ký Quỹ",
          "signal": "Tín hiệu (đặt lệnh)",
          "sl": "Cắt Lỗ",
          "tp": "Chốt Lời"
        },
        "side": "Hướng",
        "sideBuy": "Mua",
        "sideSell": "Bán",
        "summary": "{{count}} giao dịch · {{wins}} thắng / {{losses}} thua · lãi/lỗ ròng {{pnl}}",
        "ticket": "Mã lệnh",
        "title": "Chi Tiết Lệnh",
        "volume": "Khối Lượng"
      }
    }
  }
} as const;
export default BacktestRun;
