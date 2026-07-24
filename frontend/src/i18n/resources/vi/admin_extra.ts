// Auto-generated supplementary keys for admin
const AdminExtra = {
  "admin": {
    "aiGateway": {
      "errors": {
        "loadProviders": "Tải nhà cung cấp thất bại",
        "toggleFailed": "Chuyển đổi thất bại",
        "loadModels": "Tải mô hình thất bại"
      },
      "addProviderPending": "Tính năng thêm nhà cung cấp đang chờ backend",
      "title": "Quản lý AI Gateway",
      "description": "Quản lý nhà cung cấp AI, mô hình và giá. Người dùng chọn từ mô hình khả dụng, tính phí theo token từ ví.",
      "addProvider": "Thêm nhà cung cấp",
      "columns": {
        "baseUrl": "URL cơ sở",
        "apiKey": "API Key"
      },
      "configured": "Chưa cấu hình",
      "editProvider": "Thêm nhà cung cấp",
      "providerId": "Nhập ID nhà cung cấp",
      "providerIdPlaceholder": "deepseek / openai / qwen ...",
      "displayName": "Tên hiển thị",
      "displayNamePlaceholder": "DeepSeek",
      "baseUrl": "Nhập URL cơ sở",
      "apiKeyLabel": "API key, mã hóa khi lưu",
      "apiKeyEditPlaceholder": "Để trống để giữ nguyên",
      "editModel": "Thêm mô hình",
      "modelName": "Tên mô hình",
      "priceInput": "Giá nhập ($/1M)",
      "priceOutput": "Giá xuất ($/1M)",
      "addModel": "Thêm mô hình",
      "confirmDeleteModel": "Xóa mô hình này?",
      "noModels": "Không có mô hình"
    },
    "account": {
      "errors": {
        "loadFailed": "Tải tài khoản thất bại",
        "freezeFailed": "Đóng băng thất bại",
        "unfreezeFailed": "Giải đóng băng thất bại"
      },
      "frozen": "Tài khoản đã đóng băng",
      "unfrozen": "Tài khoản đã giải đóng băng",
      "columns": {
        "createdAt": "Thời gian tạo"
      },
      "confirmFreeze": "Đóng băng tài khoản này?",
      "title": "Quản lý tài khoản",
      "searchPlaceholder": "Tìm kiếm tài khoản",
      "detail": "Chi tiết tài khoản",
      "auditLogs": "Nhật ký kiểm toán"
    },
    "settings": {
      "saveSuccess": "Lưu thành công",
      "saveFailed": "Lưu thất bại",
      "deleteFailed": "Xóa thất bại",
      "actionFailed": "Thao tác thất bại",
      "columns": {
        "key": "Khóa cài đặt"
      },
      "confirmDelete": "Xác nhận xóa?",
      "title": "Cài đặt quản lý Agent",
      "addSetting": "Thêm cài đặt",
      "permissionRules": "Quy tắc quyền (permission.rule.N)",
      "permissionFormat": "Định dạng:",
      "permissionExample": "Ví dụ:",
      "permissionAddRule": "Thêm quy tắc: tạo khóa cài đặt",
      "addManagedSetting": "Thêm cài đặt quản lý",
      "settingKey": "Khóa cài đặt",
      "keyPlaceholder": "Ví dụ: allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "Ví dụ: claude-sonnet-5,deepseek-v4"
    },
    "autogen": {
      "approved": "Tác vụ đã duyệt và xuất bản",
      "rejected": "Tác vụ đã từ chối",
      "enqueued": "{{count}} tác vụ đã thêm vào hàng đợi",
      "confirmApprove": "Duyệt và xuất bản?",
      "confirmReject": "Từ chối tác vụ này?",
      "title": "Tác vụ tạo chiến lược AI",
      "allStatus": "Tất cả trạng thái",
      "triggerBatch": "Kích hoạt tạo hàng loạt",
      "symbols": "Ký hiệu (phẩy ngăn cách)",
      "timeframes": "Khung thời gian (phẩy ngăn cách)",
      "strategyTypes": "Loại chiến lược (phẩy ngăn cách)"
    },
    "billing": {
      "columns": {
        "autoRenew": "Tự động gia hạn",
        "periodStart": "Bắt đầu kỳ",
        "periodEnd": "Kết thúc kỳ",
        "createdAt": "Thời gian tạo",
        "balanceBefore": "Số dư trước",
        "balanceAfter": "Số dư sau"
      },
      "title": "Quản lý thanh toán",
      "monthlyRevenue": "Doanh thu tháng",
      "totalRevenue": "Tổng doanh thu",
      "activeSubs": "Đăng ký hoạt động",
      "planRevenue": "Chi tiết doanh thu gói",
      "filterByPlan": "Lọc theo gói",
      "filterByStatus": "Lọc theo trạng thái",
      "walletTransactions": "Giao dịch ví",
      "filterByType": "Lọc theo loại",
      "txPlatformFee": "Phí nền tảng"
    },
    "coupon": {
      "loadFailed": "Tải mã giảm giá thất bại",
      "fillRequired": "Vui lòng điền các trường bắt buộc",
      "created": "Đã tạo mã giảm giá",
      "createFailed": "Tạo mã giảm giá thất bại",
      "disabled": "Đã vô hiệu hóa mã giảm giá",
      "disableFailed": "Vô hiệu hóa thất bại",
      "colMinPurchase": "Mua tối thiểu",
      "create": "Tạo mã giảm giá",
      "createTitle": "Tạo mã giảm giá",
      "codePlaceholder": "Mã giảm giá (vd SUMMER20)",
      "valuePlaceholder": "Giá trị giảm (vd 20=20% hoặc 50=¥50)",
      "minPurchasePlaceholder": "Số tiền mua tối thiểu (0=không giới hạn)",
      "maxUsesPlaceholder": "Số lần dùng tối đa (0=không giới hạn)",
      "expiresPlaceholder": "Hết hạn (ISO 8601, trống=không hết hạn)"
    },
    "dashboard": {
      "errors": {
        "loadFailed": "Tải dữ liệu dashboard thất bại"
      },
      "title": "Dashboard quản trị",
      "totalUsers": "Tổng người dùng",
      "activeUsers": "Người dùng hoạt động",
      "verifiedUsers": "Người dùng đã xác minh",
      "mtAccounts": "Tài khoản MT",
      "onlineAccounts": "Tài khoản online",
      "todayTrades": "Giao dịch hôm nay",
      "todayProfit": "Lợi nhuận hôm nay",
      "activeSubs": "Đăng ký hoạt động",
      "monthlyRevenue": "Doanh thu tháng",
      "totalRevenue": "Tổng doanh thu",
      "marketStrategies": "Chiến lược thị trường",
      "marketSales": "Doanh số thị trường",
      "marketRevenue": "Doanh thu thị trường",
      "recentLogs": "Nhật ký gần đây"
    },
    "logs": {
      "modules": {
        "userManagement": "Quản lý người dùng",
        "accountManagement": "Quản lý tài khoản",
        "systemConfig": "Cấu hình hệ thống"
      },
      "columns": {
        "actionType": "Loại thao tác",
        "ip": "Địa chỉ IP"
      },
      "errors": {
        "loadFailed": "Tải nhật ký thất bại"
      },
      "title": "Nhật ký thao tác",
      "filterModule": "Lọc theo module",
      "filterAction": "Lọc theo thao tác"
    },
    "depositAddresses": {
      "importFailed": "Nhập thất bại",
      "user": "ID người dùng",
      "received": "USDT nhận",
      "assignedAt": "Thời gian gán",
      "importHint": "Sử dụng công cụ hdgen trên máy offline để tạo deposit_addresses.bin, sau đó tải lên đây.",
      "all": "Tất cả trạng thái",
      "import": "Nhập địa chỉ",
      "availablePool": "Khả dụng trong pool",
      "total": "Tổng địa chỉ"
    },
    "deposit": {
      "table": {
        "user": "ID người dùng",
        "amount": "Số USDT",
        "txHash": "Tx Hash"
      },
      "title": "Quản lý nạp tiền"
    },
    "analytics": {
      "platformRev": "DT nền tảng",
      "providerRev": "DT nhà cung cấp",
      "activeBuyers": "Người mua hoạt động",
      "refundRate": "Tỷ lệ hoàn tiền",
      "newSubs": "Người đăng ký mới",
      "totalStrategies": "Tổng chiến lược",
      "newStrategies": "Chiến lược mới",
      "topByRevenue": "Chiến lược DT cao nhất",
      "topBySubs": "Chiến lược đăng ký nhiều nhất",
      "topProvidersRev": "Nhà cung cấp DT cao nhất",
      "topProvidersStrat": "Nhà cung cấp nhiều chiến lược nhất"
    },
    "marketplace": {
      "loadFailed": "Tải chiến lược thất bại",
      "featureSuccess": "Đã đặt chiến lược nổi bật",
      "featureFailed": "Đặt nổi bật thất bại",
      "unfeatureSuccess": "Đã bỏ nổi bật",
      "unfeatureFailed": "Bỏ nổi bật thất bại",
      "unfeature": "Bỏ nổi bật",
      "filterStatus": "Tất cả trạng thái",
      "searchPlaceholder": "Tìm theo tiêu đề...",
      "featureTitle": "Chiến lược nổi bật",
      "featureDesc": "Đặt ưu tiên hiển thị nổi bật. Cao hơn = nổi bật hơn."
    },
    "refund": {
      "loadFailed": "Tải yêu cầu hoàn tiền thất bại",
      "approved": "Hoàn tiền đã duyệt và thực hiện",
      "rejected": "Yêu cầu hoàn tiền bị từ chối",
      "processFailed": "Xử lý hoàn tiền thất bại",
      "approve": "Duyệt & Thực hiện",
      "filterStatus": "Tất cả trạng thái",
      "approveTitle": "Duyệt hoàn tiền",
      "rejectTitle": "Từ chối hoàn tiền",
      "reviewNotePlaceholder": "Ghi chú duyệt (tùy chọn khi từ chối, nên có khi duyệt)..."
    },
    "sidebar": {
      "shareManagement": "Phân tích chia sẻ"
    },
    "walletCalculator": {
      "title": "Máy tính Token ↔ USD",
      "selectModel": "Chọn mô hình (cơ sở giá)",
      "usdAmount": "Số tiền USD",
      "fillResult": "Điền kết quả"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "Chưa chọn người dùng"
      },
      "messages": {
        "adjustSuccess": "Điều chỉnh số dư thành công",
        "adjustFailed": "Điều chỉnh thất bại"
      },
      "columns": {
        "walletNumber": "Số ví",
        "balanceAfter": "Số dư sau"
      },
      "title": "Quản lý ví",
      "tabWallets": "Ví người dùng",
      "userList": "Danh sách người dùng",
      "searchPlaceholder": "Tìm ví/email/nickname",
      "noMatch": "Không có người dùng",
      "walletDetail": "Chi tiết ví",
      "adjustBalance": "Điều chỉnh số dư",
      "tabDepositAddresses": "Địa chỉ nạp tiền"
    },
    "config": {
      "apiKey": "API Key"
    },
    "userManagement": {
      "form": {
        "accountNumber": "Số tài khoản",
        "accountNumberInvalid": "5-6 chữ số, không bắt đầu bằng 0, không chứa 4 hoặc 7"
      },
      "messages": {
        "loadUsersFailed": "Tải người dùng thất bại"
      }
    }
  }
} as const;
export default AdminExtra;
