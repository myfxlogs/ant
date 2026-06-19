// Auto-generated from proto/ant/v1/i18n/strategy_code_editor_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyCodeEditor = {
  "strategy": {
    "codeEditor": {
      "actions": {
        "copy": "Sao chép",
        "preview": "Xem trước tín hiệu",
        "saveAsTemplate": "Lưu thành template",
        "sendToAI": "Gửi AI để sửa",
        "sendToAIFixTitlePreview": "Sửa Lỗi Xem Trước",
        "sendToAIFixTitleValidate": "Xác thực thất bại / có cảnh báo",
        "validate": "Xác thực mã"
      },
      "aiPrompt": {
        "currentCodeTitle": "[Mã hiện tại]",
        "fenceEnd": "```",
        "intro": "Vui lòng chỉnh sửa mã chiến lược theo thông tin bên dưới để vượt qua xác thực và chạy xem trước thành công.",
        "outputTitle": "[Đầu ra]",
        "outro": "Chỉ trả về code đã sửa trong ```python```.",
        "problem": "[Vấn đề] {{title}}",
        "pythonFenceStart": "```python"
      },
      "cards": {
        "previewResult": "Kết Quả Xem Trước",
        "validationResult": "Kết quả xác thực"
      },
      "hints": {
        "previewInfo": "Xem trước sẽ chạy với dữ liệu mẫu."
      },
      "labels": {
        "account": "Tài khoản",
        "code": "Mã chiến lược",
        "disabledSuffix": " (đã tắt)",
        "symbol": "Mã",
        "timeframe": "Khung thời gian"
      },
      "messages": {
        "copied": "Đã sao chép",
        "copyFailed": "Sao chép thất bại, vui lòng sao chép thủ công",
        "enterCode": "Vui lòng nhập mã chiến lược",
        "execFailed": "Thực thi thất bại",
        "previewFailed": "Xem trước thất bại",
        "previewOk": "Xem trước hoàn tất",
        "previewSuccess": "Xem trước thành công",
        "savedAsTemplate": "Đã lưu thành template",
        "selectAccount": "Vui lòng chọn tài khoản",
        "validateError": "Lỗi xác thực",
        "validateFailed": "Xác thực thất bại",
        "validateOk": "Xác thực thành công"
      },
      "placeholders": {
        "code": "Dán mã chiến lược Python...",
        "loadingSymbols": "Đang tải danh sách mã...",
        "noSymbols": "Không có mã khả dụng",
        "selectAccount": "Chọn tài khoản",
        "selectAccountFirst": "Chọn tài khoản trước",
        "selectSymbol": "Chọn mã"
      },
      "title": "Trình soạn thảo chiến lược"
    }
  }
} as const;
export default StrategyCodeEditor;
