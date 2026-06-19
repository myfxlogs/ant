// Auto-generated from proto/ant/v1/i18n/strategy_code_assist_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const CodeAssist = {
  "strategy": {
    "codeAssist": {
      "aiReviseTitle": "Trợ lý AI — sửa code",
      "applyAllSuggestions": "Áp Dụng Mặc Định Đề Xuất",
      "codeEmpty": "Chưa có code để sửa.",
      "codeUpdated": "Code đã cập nhật. Vui lòng chạy lại xác thực trước khi lưu.",
      "defaultLabel": "mặc định",
      "enterInstruction": "Vui lòng mô tả điều bạn muốn thay đổi.",
      "explain": "Giải Thích Code",
      "fillRequiredParams": "Vui lòng điền tham số bắt buộc: {{keys}}",
      "generatePlaceholder": "Mô tả yêu cầu chiến lược của bạn...",
      "noPython": "AI không trả về khối Python. Hãy thử diễn đạt lại.",
      "optionalParamsDesc": "Các tham số này đã có mặc định từ code. Để trống để dùng mặc định; giá trị nhập chỉ áp dụng cho lần chạy này và không sửa đổi chiến lược đã lưu.",
      "optionalParamsTitle": "Tham Số Tùy Chọn",
      "paramDescriptions": {
        "confidence": "Ngưỡng tin cậy tín hiệu (0-1). Tín hiệu dưới giá trị này bị bỏ qua.",
        "emaPeriod": "Số nến nhìn lại EMA (trung bình động hàm mũ).",
        "fastPeriod": "Chu kỳ nhanh (số nến). Dùng bởi MACD / dual-MA; nhỏ hơn là nhạy hơn.",
        "genericPercent": "Tham số phần trăm / tỷ lệ (VD: 1 nghĩa là 1%).",
        "genericPeriod": "Cửa sổ nhìn lại (số nến) dùng để tính chỉ báo.",
        "lotSize": "Kích thước lệnh (lots / khối lượng). Cỡ lớn hơn nghĩa là rủi ro cao hơn.",
        "maxLoss": "Lỗ tối đa mỗi giao dịch theo tỷ lệ vốn (0.01 = 1%).",
        "riskLevel": "Mức rủi ro (thấp / vừa / cao). Kiểm soát kích thước vị thế và độ rộng cắt lỗ/chốt lời.",
        "rsiPeriod": "Số nến nhìn lại RSI. Giá trị điển hình: 14.",
        "signalPeriod": "Chu kỳ tín hiệu (số nến). Độ dài làm mượt cho MACD DIF/DEA.",
        "slowPeriod": "Chu kỳ chậm (số nến). Dùng bởi MACD / dual-MA; lớn hơn là mượt hơn.",
        "smaPeriod": "Số nến nhìn lại SMA (trung bình động đơn giản).",
        "stopLoss": "Khoảng cách cắt lỗ (%). Đóng vị thế khi giá di chuyển xa đến mức này ngược hướng.",
        "takeProfit": "Khoảng cách chốt lời (%). Đóng vị thế khi giá di chuyển xa đến mức này theo hướng có lợi.",
        "threshold": "Ngưỡng kích hoạt tín hiệu. Ý nghĩa chính xác phụ thuộc vào logic chiến lược."
      },
      "required": "bắt buộc",
      "requiredParamsDesc": "Chiến lược đọc các tham số này nhưng chưa có mặc định. Hãy điền trước khi lưu.",
      "requiredParamsTitle": "Tham Số Bắt Buộc",
      "reviseInputPlaceholder": "VD: Thay SMA(20) bằng EMA(50) và thêm cắt lỗ 1%.",
      "reviseSend": "Gửi AI để sửa",
      "saveBlockedNotValidated": "Vui lòng nhấn \"Xác thực code\" trước. Lưu bị vô hiệu hóa cho đến khi xác thực qua.",
      "suggested": "đề xuất",
      "tabAI": "AI Sửa",
      "tabExplain": "Giải Thích Code"
    }
  }
} as const;
export default CodeAssist;
