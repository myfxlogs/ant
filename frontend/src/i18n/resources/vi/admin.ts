// Auto-generated from proto/ant/v1/i18n/admin_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Admin = {
  "admin": {
    "strategy": {
      "actions": {
        "archive": "Lưu trữ",
        "archiveConfirm": "Lưu trữ chiến lược này?",
        "code": "Mã",
        "disable": "Vô hiệu",
        "disableConfirm": "Dừng tất cả lịch trình?",
        "enable": "Kích hoạt",
        "flag": "Cờ",
        "publish": "Xuất Bản",
        "unflag": "Bỏ cờ",
        "unpublish": "Hủy xuất bản"
      },
      "all": {
        "allActive": "Tất cả Hoạt động",
        "archived": "Đã lưu trữ",
        "disabled": "Đã vô hiệu",
        "flagFilter": "Bộ lọc Cờ",
        "flagged": "Đã gắn cờ",
        "searchPlaceholder": "Tìm theo tên...",
        "total": "Tổng {{count}}"
      },
      "columns": {
        "actions": "Thao tác",
        "code": "Mã",
        "description": "Mô tả",
        "flag": "Cờ",
        "name": "Tên",
        "no": "否",
        "owner": "Chủ sở hữu",
        "preset": "Cài sẵn",
        "public": "Công khai",
        "schedules": "Lịch trình",
        "status": "Trạng thái",
        "system": "— Hệ thống —",
        "tags": "Thẻ",
        "tagsPlaceholder": "xu-hướng, MA",
        "type": "Loại",
        "user": "Người dùng",
        "uses": "Lượt dùng",
        "yes": "Có"
      },
      "messages": {
        "archiveFailed": "Lưu trữ thất bại",
        "archiveSuccess": "Đã lưu trữ",
        "deleteFailed": "Xóa thất bại",
        "disableFailed": "Vô hiệu thất bại",
        "disableSuccess": "Đã vô hiệu — tất cả lịch trình đã dừng",
        "enableFailed": "Kích hoạt thất bại",
        "enableSuccess": "Đã kích hoạt",
        "flagFailed": "Gắn cờ thất bại",
        "flagSuccess": "Đã gắn cờ chiến lược",
        "loadPresetFailed": "Không thể tải chiến lược cài sẵn",
        "loadStrategiesFailed": "Không thể tải danh sách chiến lược",
        "presetCreated": "Đã tạo cài sẵn",
        "presetDeleted": "Đã xóa cài sẵn",
        "presetUpdated": "Đã cập nhật cài sẵn",
        "publishFailed": "发布失败",
        "publishSuccess": "已发布",
        "saveFailed": "Lưu thất bại",
        "unflagFailed": "Bỏ cờ thất bại",
        "unflagSuccess": "Đã bỏ cờ",
        "unpublishFailed": "取消发布失败",
        "unpublishSuccess": "已取消发布"
      },
      "preset": {
        "add": "Thêm Cài sẵn",
        "create": "Tạo Cài sẵn",
        "deleteConfirm": "Xóa cài sẵn này?",
        "edit": "Sửa Cài sẵn"
      },
      "tabs": {
        "allStrategies": "Tất cả Chiến lược",
        "preset": "Chiến lược Cài sẵn"
      },
      "title": "Quản lý Chiến lược"
    },
    "sweep": {
      "aboveThreshold": "Vượt ngưỡng",
      "address": "Địa chỉ",
      "addressId": "ID địa chỉ",
      "batchExport": "Xuất hàng loạt",
      "batchExportSuccess": "Xuất hàng loạt hoàn tất",
      "builtAt": "Thời gian xây dựng",
      "bundleId": "ID gói",
      "bundleStatus": "Trạng thái",
      "dashboard": "Bảng điều khiển",
      "derivationIndex": "Chỉ mục dẫn xuất",
      "export": "Xuất",
      "exportSuccess": "Xuất hoàn tất",
      "import": "Nhập",
      "importHint": "Tải lên gói sweep đã ký (.bin) để nhập và phát.",
      "importSuccess": "Nhập hoàn tất",
      "importTitle": "Nhập gói đã ký",
      "pendingBundles": "Gói đang chờ",
      "pendingSignBundles": "Gói chờ ký",
      "sweepStatus": "Trạng thái sweep",
      "threshold": "Ngưỡng",
      "title": "Quản lý Sweep",
      "totalUnswept": "Tổng chưa sweep",
      "undelegate": "Hủy ủy quyền",
      "undelegateSuccess": "Đã xuất gói hủy ủy quyền",
      "unswept": "Chưa sweep",
      "uploadHint": "Nhấp hoặc kéo tệp để tải lên",
      "uploadXpub": "Tải lên XPUB",
      "xpubFpNotSet": "Chưa xác minh dấu vân tay",
      "xpubFpVerified": "Đã xác minh dấu vân tay",
      "xpubHint": "Tải lên tệp XPUB để dẫn xuất địa chỉ nạp tiền. Dấu vân tay sẽ được xác minh khi nhập.",
      "xpubImported": "Đã nhập XPUB",
      "xpubTitle": "Quản lý XPUB"
    }
  }
} as const;
export default Admin;
