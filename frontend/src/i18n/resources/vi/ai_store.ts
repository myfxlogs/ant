// Auto-generated from proto/ant/v1/i18n/ai_store_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiStore = {
  "ai": {
    "store": {
      "context": {
        "outputRules": {
          "noImport": "- Không xuất bất kỳ câu lệnh import nào",
          "validateFirst": "- Mã phải vượt qua xác thực trước tiên",
          "wrapPython": "- 如果你输出策略代码，请输出完整代码，并用 ```python 包裹"
        },
        "outputTitle": "Yêu cầu đầu ra:",
        "userPrefsTitle": "Tùy chọn người dùng (vui lòng tuân thủ càng nhiều càng tốt):"
      },
      "strategyRules": {
        "rules": {
          "mustDefineEntry": "- Chiến lược phải định nghĩa biến signal hoặc hàm run(context) (ưu tiên run(context))",
          "noDunderAccess": "- Không được truy cập thuộc tính dunder (obj.__xxx__)",
          "noDunderName": "- Không được dùng tên dunder (__xxx__)",
          "noGlobal": "- Không được dùng global / nonlocal",
          "noImport": "- Không được phép import / from ... import ..."
        },
        "allowedGlobals": "Toàn cục/mô-đun được phép: np, math, datetime, calculate_rsi (không import).",
        "title": "Khi viết mã chiến lược Python AntTrader, bạn phải tuân thủ nghiêm ngặt các quy tắc xác thực sau:"
      },
      "conversations": {
        "newConversationTitle": "Hội thoại mới"
      },
      "messages": {
        "clearedLocalOnly": "Đã xóa (chỉ cục bộ)",
        "createConversationFailed": "Tạo hội thoại thất bại",
        "deleteConversationFailed": "Xóa hội thoại thất bại",
        "generateReportFailed": "Tạo báo cáo thất bại",
        "generateReportSuccess": "Báo cáo đã được tạo thành công",
        "getReportsFailed": "Lấy báo cáo thất bại",
        "loadConversationFailed": "Tải hội thoại thất bại",
        "sendFailedInline": "Gửi tin nhắn thất bại",
        "sendFailedToast": "Gửi tin nhắn thất bại"
      },
      "prefs": {
        "rememberPrefix": "/remember ",
        "rememberedToast": "Đã lưu tùy chọn",
        "savedReply": "Đã lưu tùy chọn của bạn."
      }
    }
  }
} as const;
export default AiStore;
