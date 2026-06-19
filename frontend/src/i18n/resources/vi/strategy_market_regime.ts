// Auto-generated from proto/ant/v1/i18n/strategy_market_regime_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const MarketRegime = {
  "strategy": {
    "marketRegime": {
      "detectFailed": "Phát hiện chế độ thị trường thất bại",
      "detectSuccess": "Phát hiện chế độ thị trường hoàn tất",
      "form": {
        "accountId": "ID Tài Khoản",
        "accountIdPlaceholder": "UUID tài khoản MT",
        "accountIdRequired": "ID tài khoản là bắt buộc",
        "klineCount": "Số K-line",
        "submit": "Bắt Đầu Phát Hiện",
        "symbol": "Mã",
        "symbolPlaceholder": "EURUSD",
        "symbolRequired": "Vui lòng chọn mã",
        "timeframe": "Khung thời gian",
        "title": "Tham Số Phát Hiện"
      },
      "result": {
        "confidence": "Độ Tin Cậy",
        "features": "Đặc Trưng",
        "modelVersion": "Phiên Bản Mô Hình",
        "recordId": "ID Bản Ghi",
        "status": "Trạng thái",
        "strategyFamilies": "Họ Chiến Lược",
        "title": "Kết Quả Phát Hiện"
      },
      "ruleVersionAlert": "Hiện đang dùng mô hình phát hiện dựa trên quy tắc rule-v1, điều khiển bởi dữ liệu K-line thời gian thực.",
      "subtitle": "Phân tích hướng xu hướng, chế độ biến động và hiệu quả giá từ dữ liệu K-line lịch sử để phân loại điều kiện thị trường hiện tại.",
      "title": "Phát Hiện Chế Độ Thị Trường"
    }
  }
} as const;
export default MarketRegime;
