// Auto-generated from proto/ant/v1/i18n/base_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "Quản Lý Tài Khoản",
          "systemConfig": "Cấu Hình Hệ Thống",
          "trading": "Giao Dịch",
          "userManagement": "Quản Lý Người Dùng"
        },
        "actionType": "Hành Động",
        "failed": "Thất Bại",
        "module": "Mô-đun",
        "status": "Trạng Thái",
        "success": "Thành Công",
        "target": "Mục Tiêu",
        "time": "Thời Gian"
      },
      "riskMetrics": {
        "orderCloseFailed": "平仓失败",
        "orderCloseSuccess": "平仓成功",
        "orderSendFailed": "下单失败",
        "orderSendSuccess": "下单成功",
        "riskValidateError": "错误",
        "riskValidatePass": "通过",
        "riskValidateReject": "拒绝",
        "riskValidateTotal": "总验证数",
        "title": "风控指标"
      },
      "riskWindow": {
        "noData": "暂无窗口指标数据",
        "noRejectData": "本时段无拒绝记录",
        "orderCloseFailed": "平仓失败",
        "orderCloseSuccess": "平仓成功",
        "orderSendFailed": "下单失败",
        "orderSendSuccess": "下单成功",
        "rejectCount": "拒绝次数",
        "rejectRiskCodesHeader": "风控代码",
        "title": "风控窗口",
        "validateError": "错误",
        "validatePass": "通过",
        "validateReject": "拒绝",
        "validateTotal": "总计"
      },
      "errors": {
        "loadFailed": "Tải dữ liệu bảng điều khiển thất bại"
      },
      "activeUsers": "Người Dùng Hoạt Động",
      "loadFailed": "Tải dữ liệu bảng điều khiển thất bại",
      "mtAccounts": "TK MT",
      "onlineAccounts": "Trực Tuyến",
      "recentLogs": "Nhật Ký Gần Đây",
      "title": "Bảng Điều Khiển Quản Trị",
      "todayProfit": "Lợi Nhuận Hôm Nay",
      "todayTrades": "Giao Dịch Hôm Nay",
      "totalUsers": "Tổng Người Dùng",
      "verifiedUsers": "Người dùng đã xác minh",
      "activeSubs": "Gói đang hoạt động",
      "monthlyRevenue": "Doanh thu hàng tháng",
      "totalRevenue": "Tổng doanh thu",
      "marketStrategies": "Chiến lược thị trường",
      "marketSales": "Doanh số thị trường",
      "marketRevenue": "Doanh thu thị trường",
      "validateTotal": "Tổng xác thực",
      "validatePass": "Xác thực đạt",
      "validateReject": "Xác thực từ chối",
      "validateError": "Lỗi xác thực",
      "orderSendSuccess": "Gửi lệnh thành công",
      "orderSendFailed": "Gửi lệnh thất bại",
      "orderCloseSuccess": "Đóng lệnh thành công",
      "orderCloseFailed": "Đóng lệnh thất bại",
      "rejectCount": "Số lần từ chối"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "Ngày Tạo",
          "email": "用户邮箱",
          "id": "ID",
          "lastLogin": "Đăng Nhập Cuối",
          "mtAccountCount": "TK MT",
          "nickname": "Biệt Danh",
          "role": "Vai Trò",
          "status": "Trạng Thái"
        },
        "title": "Chi Tiết Người Dùng"
      },
      "form": {
        "placeholders": {
          "email": "Nhập email",
          "nickname": "Nhập biệt danh",
          "password": "输入密码"
        },
        "accountNumber": "Số Tài Khoản",
        "accountNumberInvalid": "5-6 chữ số, không có số 0 ở đầu, không có 4 hoặc 7",
        "email": "用户邮箱",
        "nickname": "Biệt Danh",
        "password": "Mật Khẩu",
        "role": "Vai Trò",
        "status": "Trạng Thái",
        "accountNumberPlaceholder": "e.g. 123568"
      },
      "passwordForm": {
        "placeholders": {
          "confirmPassword": "再次输入新密码",
          "newPassword": "Nhập mật khẩu mới"
        },
        "validation": {
          "confirmPasswordRequired": "Vui lòng xác nhận mật khẩu mới",
          "newPasswordRequired": "Vui lòng nhập mật khẩu mới",
          "passwordMin8": "Mật khẩu phải có ít nhất 8 ký tự",
          "passwordMismatch": "Mật khẩu không khớp",
          "passwordMustContainLettersAndNumbers": "密码必须包含字母和数字"
        },
        "confirmPassword": "Xác Nhận Mật Khẩu",
        "newPassword": "Mật Khẩu Mới",
        "submit": "Cập Nhật Mật Khẩu"
      },
      "actions": {
        "changePassword": "修改密码",
        "details": "Chi Tiết",
        "disable": "Vô Hiệu",
        "enable": "Kích Hoạt"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "Xóa {{count}} người dùng? Hành động này không thể hoàn tác.",
        "batchDeletePartial": "Đã xóa {{deleted}}, {{failed}} thất bại",
        "batchDeleteSuccess": "Đã xóa {{count}} người dùng",
        "title": "Xóa người dùng này? Hành động này không thể hoàn tác."
      },
      "filters": {
        "rolePlaceholder": "Lọc theo vai trò",
        "searchPlaceholder": "Tìm theo email hoặc biệt danh",
        "statusPlaceholder": "按状态筛选"
      },
      "messages": {
        "newPasswordIs": "新密码为: {{password}}",
        "passwordUpdateFailed": "Cập nhật mật khẩu thất bại",
        "passwordUpdatedSuccess": "Đã cập nhật mật khẩu",
        "userCreateFailed": "Tạo người dùng thất bại",
        "userCreatedSuccess": "Đã tạo người dùng",
        "userDeleteFailed": "Xóa người dùng thất bại",
        "userDeletedSuccess": "Đã xóa người dùng",
        "userDisabled": "Đã vô hiệu người dùng",
        "userEnabled": "Đã kích hoạt người dùng",
        "userUpdateFailed": "Cập nhật người dùng thất bại",
        "userUpdatedSuccess": "Đã cập nhật người dùng",
        "loadUsersFailed": "Không tải được người dùng"
      },
      "modals": {
        "createTitle": "Tạo Người Dùng",
        "editTitle": "Sửa Người Dùng",
        "passwordTitle": "修改密码"
      },
      "pagination": {
        "total": "共 {{total}} 位用户"
      },
      "roles": {
        "audit": "审计",
        "customerService": "CSKH",
        "operation": "Vận Hành",
        "superAdmin": "Quản Trị Viên",
        "user": "Người Dùng"
      },
      "status": {
        "active": "Hoạt Động",
        "suspended": "已停用"
      },
      "table": {
        "actions": "Thao Tác",
        "createdAt": "Ngày Tạo",
        "email": "用户邮箱",
        "id": "ID",
        "mtAccountCount": "TK MT",
        "nickname": "Biệt Danh",
        "role": "Vai Trò",
        "status": "Trạng Thái"
      },
      "addUser": "Thêm Người Dùng",
      "title": "Quản Lý Người Dùng"
    },
    "config": {
      "messages": {
        "disabled": "已禁用",
        "enabled": "已启用",
        "loadFailed": "加载配置失败",
        "operationFailed": "操作失败",
        "updateFailed": "更新配置失败",
        "updated": "配置已更新"
      },
      "placeholders": {
        "apiKey": "输入API Key",
        "baseUrl": "输入Base URL",
        "configValue": "输入配置值",
        "description": "输入描述",
        "json": "输入JSON",
        "model": "输入模型名称"
      },
      "providerOptions": {
        "custom": "自定义 / OpenAI 兼容",
        "deepseek": "DeepSeek",
        "zhipu": "智谱AI"
      },
      "validation": {
        "apiKeyRequired": "API Key不能为空",
        "greenMaxFailedRunsNonNegative": "绿色最大失败次数需≥0",
        "greenSuccessRateRange": "绿色成功率需在0-100之间",
        "jsonEmpty": "JSON不能为空",
        "jsonInvalid": "JSON格式无效",
        "minSampleSizeNonNegative": "最小样本量需≥0",
        "modelRequired": "模型名称不能为空",
        "yellowNotGreaterThanGreen": "黄色阈值不能超过绿色阈值",
        "yellowSuccessRateRange": "黄色成功率需在0-100之间"
      },
      "aiProviderCatalog": "AI提供商目录",
      "baseUrlLabel": "Base URL",
      "configItem": "配置项",
      "description": "Mô Tả",
      "econAIConfig": "经济日历AI配置",
      "editConfig": "编辑配置: {{key}}",
      "enableToggle": "Kích Hoạt",
      "fillTemplate": "填充模板",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每用户最大账户数",
      "modelName": "模型名称",
      "off": "关",
      "on": "开",
      "provider": "提供商",
      "status": "Trạng Thái",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "阈值描述",
      "thresholdInfo": "阈值说明",
      "title": "系统配置",
      "toggle": "切换",
      "updatedAt": "更新时间",
      "value": "值",
      "apiKey": "API Key"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "添加国家失败",
        "countryAdded": "国家已添加",
        "countryRemoveFailed": "移除国家失败",
        "countryRemoved": "国家已移除",
        "kycUpdateFailed": "更新KYC状态失败",
        "kycUpdated": "KYC状态已更新",
        "overrideUpdateFailed": "更新制裁豁免失败",
        "overrideUpdated": "豁免状态已更新"
      },
      "actions": "Thao Tác",
      "addCountry": "Thêm Quốc Gia",
      "addSanctionedCountry": "添加制裁国家",
      "addedBy": "Người Thêm",
      "confirmGrantOverride": "确认授予该用户豁免权限？",
      "confirmRevokeOverride": "确认撤销该用户的豁免权限？",
      "country": "Quốc Gia",
      "countryCode": "Mã Quốc Gia",
      "countryLabel": "Quốc Gia",
      "disclaimer": "免责声明",
      "emptyKYC": "Không có hồ sơ KYC",
      "emptySanctions": "Không có quốc gia bị cấm vận",
      "filterByKYCStatus": "按KYC状态筛选",
      "grantOverride": "Cấp Ghi Đè",
      "kycStatus": "Trạng Thái KYC",
      "kycStatusTab": "用户KYC状态",
      "override": "Ghi Đè",
      "overrideWarning": "此用户来自受制裁国家，授予豁免将允许交易。",
      "pending": "Đang Chờ",
      "questionnaire": "问卷",
      "rejected": "Đã Từ Chối",
      "revokeOverride": "Thu Hồi Ghi Đè",
      "sanctioned": "Đã Cấm Vận",
      "sanctionedCountries": "Quốc Gia Bị Cấm Vận",
      "sanctionedCountriesTab": "Quốc Gia Bị Cấm Vận",
      "setKYC": "Đặt KYC",
      "setKYCStatus": "Đặt Trạng Thái KYC",
      "title": "Kiểm Soát Quyền Hạn",
      "unverified": "Chưa Xác Minh",
      "userEmail": "用户邮箱",
      "userKYCStatus": "用户KYC状态",
      "verified": "Đã Xác Minh"
    },
    "aiGateway": {
      "errors": {
        "loadProviders": "Không thể tải nhà cung cấp",
        "toggleFailed": "Chuyển đổi thất bại",
        "loadModels": "Không thể tải mô hình"
      },
      "columns": {
        "baseUrl": "URL cơ sở",
        "apiKey": "API Key"
      },
      "addProviderPending": "Tính năng thêm nhà cung cấp đang chờ backend hỗ trợ",
      "title": "Quản lý AI Gateway",
      "description": "Quản lý nhà cung cấp AI, mô hình và giá. Người dùng chọn mô hình có sẵn, tính phí token từ ví.",
      "addProvider": "Thêm nhà cung cấp",
      "provider": "Nhà cung cấp",
      "configured": "Đã cấu hình",
      "notConfigured": "Chưa cấu hình",
      "models": "Mô hình",
      "editProvider": "Sửa nhà cung cấp",
      "providerId": "ID nhà cung cấp",
      "providerIdRequired": "Vui lòng nhập Provider ID",
      "displayName": "Tên hiển thị",
      "displayNameRequired": "Vui lòng nhập tên hiển thị",
      "baseUrl": "URL cơ sở",
      "baseUrlRequired": "Vui lòng nhập Base URL",
      "apiKeyLabel": "API Key",
      "apiKeyEditHint": "Để trống để giữ khóa hiện tại",
      "apiKeyHint": "API key, được mã hóa khi lưu trữ",
      "apiKeyEditPlaceholder": "Để trống để giữ nguyên",
      "editModel": "Sửa mô hình",
      "addModel": "Thêm mô hình",
      "modelName": "Tên mô hình",
      "modelNameRequired": "Vui lòng nhập tên mô hình",
      "priceInput": "Giá đầu vào ($/1M)",
      "priceOutput": "Giá đầu ra ($/1M)",
      "confirmDeleteModel": "Xóa mô hình này?",
      "noModels": "Không có mô hình",
      "noModelsDiscovered": "No models discovered. Check API key and base URL.",
      "discoverFailed": "Failed to discover models",
      "discover": "Discover",
      "displayNamePlaceholder": "DeepSeek Chat"
    },
    "account": {
      "errors": {
        "loadFailed": "Không thể tải tài khoản",
        "freezeFailed": "Đóng băng thất bại",
        "unfreezeFailed": "Mở khóa thất bại"
      },
      "columns": {
        "id": "ID",
        "user": "Người dùng",
        "login": "Tên đăng nhập",
        "type": "Loại",
        "broker": "Môi giới",
        "status": "Trạng thái",
        "balance": "Số dư",
        "createdAt": "Thời gian tạo",
        "action": "Hành động",
        "server": "Máy chủ",
        "equity": "Vốn",
        "margin": "Ký quỹ",
        "time": "Thời gian",
        "detail": "Chi tiết"
      },
      "frozen": "Tài khoản bị đóng băng",
      "unfrozen": "Tài khoản đã được mở khóa",
      "detail": "Chi tiết",
      "unfreeze": "Mở khóa",
      "confirmFreeze": "Đóng băng tài khoản này?",
      "freeze": "Đóng băng",
      "title": "Quản lý tài khoản",
      "searchPlaceholder": "Tìm kiếm tài khoản",
      "status": "Trạng thái",
      "online": "Trực tuyến",
      "offline": "Ngoại tuyến",
      "auditLogs": "Nhật ký kiểm toán"
    },
    "settings": {
      "columns": {
        "key": "Khóa cài đặt",
        "value": "Giá trị",
        "action": "Hành động"
      },
      "saveSuccess": "Lưu thành công",
      "saveFailed": "Lưu thất bại",
      "deleted": "Đã xóa",
      "deleteFailed": "Xóa thất bại",
      "actionFailed": "Hành động thất bại",
      "confirmDelete": "Xác nhận xóa?",
      "title": "Cài đặt quản lý Agent",
      "addSetting": "Thêm cài đặt",
      "permissionRules": "Quy tắc quyền (permission.rule.N)",
      "permissionFormat": "Định dạng:",
      "permissionExample": "Ví dụ:",
      "permissionAddRule": "Thêm quy tắc: tạo cài đặt với khóa",
      "addManagedSetting": "Thêm cài đặt được quản lý",
      "settingKey": "Khóa cài đặt",
      "keyPlaceholder": "VD: allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "VD: claude-sonnet-5,deepseek-v4"
    },
    "billing": {
      "columns": {
        "user": "Người dùng",
        "plan": "Gói",
        "status": "Trạng thái",
        "cycle": "Chu kỳ",
        "price": "Giá",
        "autoRenew": "Tự động gia hạn",
        "periodStart": "Bắt đầu kỳ",
        "periodEnd": "Kết thúc kỳ",
        "createdAt": "Ngày tạo",
        "type": "Loại",
        "amount": "Số tiền",
        "balanceBefore": "Số dư trước",
        "balanceAfter": "Số dư sau",
        "description": "Mô tả",
        "time": "Thời gian"
      },
      "title": "Quản lý thanh toán",
      "monthlyRevenue": "Doanh thu hàng tháng",
      "totalRevenue": "Tổng doanh thu",
      "activeSubs": "Đăng ký đang hoạt động",
      "txRecords": "Giao dịch",
      "planRevenue": "Chi tiết doanh thu gói",
      "activeCount": "Đang hoạt động",
      "subscriptions": "Đăng ký",
      "filterByPlan": "Lọc theo gói",
      "planFree": "Miễn phí",
      "planPro": "Pro",
      "planEnterprise": "Doanh nghiệp",
      "filterByStatus": "Lọc theo trạng thái",
      "statusActive": "Đang hoạt động",
      "statusCancelled": "Đã hủy",
      "statusExpired": "Đã hết hạn",
      "walletTransactions": "Giao dịch ví",
      "filterByType": "Lọc theo loại",
      "txPurchase": "Mua",
      "txSale": "Bán",
      "txPlatformFee": "Phí nền tảng",
      "txDeposit": "Nạp tiền",
      "txWithdrawal": "Rút tiền"
    },
    "logs": {
      "columns": {
        "time": "Thời gian",
        "module": "Mô-đun",
        "actionType": "Loại hành động",
        "target": "Mục tiêu",
        "status": "Trạng thái",
        "ip": "Địa chỉ IP",
        "action": "Hành động",
        "details": "Chi tiết"
      },
      "modules": {
        "userManagement": "Quản lý người dùng",
        "accountManagement": "Quản lý tài khoản",
        "trading": "Giao dịch",
        "systemConfig": "Cấu hình hệ thống"
      },
      "errors": {
        "loadFailed": "Tải nhật ký thất bại"
      },
      "actions": {
        "create": "Tạo",
        "update": "Cập nhật",
        "delete": "Xóa",
        "disable": "Vô hiệu hóa",
        "enable": "Kích hoạt",
        "freeze": "Đóng băng",
        "unfreeze": "Bỏ đóng băng"
      },
      "title": "Nhật ký hoạt động",
      "filterModule": "Lọc theo mô-đun",
      "filterAction": "Lọc theo hành động"
    },
    "deposit": {
      "table": {
        "user": "Người dùng",
        "amount": "Số tiền USDT",
        "amountUsd": "Tín dụng USD",
        "txHash": "Tx Hash",
        "status": "Trạng thái",
        "reviewNote": "Ghi chú duyệt",
        "time": "Thời gian",
        "action": "Hành động",
        "block": "Block",
        "confirmations": "Confirmations"
      },
      "approved": "Nạp tiền đã được phê duyệt và ví đã được ghi có.",
      "approveFailed": "Phê duyệt nạp tiền thất bại.",
      "rejected": "Nạp tiền đã bị từ chối.",
      "rejectFailed": "Từ chối nạp tiền thất bại.",
      "approve": "Phê duyệt",
      "reject": "Từ chối",
      "title": "Quản lý nạp tiền",
      "allStatuses": "Tất cả trạng thái",
      "statusPending": "Đang chờ",
      "statusApproved": "Đã phê duyệt",
      "statusRejected": "Đã từ chối",
      "approveTitle": "Phê duyệt nạp tiền",
      "rejectTitle": "Từ chối nạp tiền",
      "reviewNoteLabel": "Ghi chú duyệt (tùy chọn)",
      "reviewNotePlaceholder": "Thêm ghi chú cho lần duyệt này...",
      "approveWarning": "Phê duyệt sẽ cộng tiền vào ví người dùng ngay lập tức."
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
        "walletNumber": "Số Ví",
        "email": "Email",
        "nickname": "Biệt danh",
        "type": "Loại",
        "amount": "Số tiền",
        "balanceAfter": "Số dư sau",
        "description": "Mô tả",
        "time": "Thời gian",
        "balance": "Số dư",
        "frozen": "Đã đóng băng",
        "currency": "Tiền tệ"
      },
      "accountNumber": "Số TK",
      "add": "Thêm",
      "adjustBalance": "Điều Chỉnh Số Dư",
      "adjustFailed": "Điều chỉnh thất bại",
      "adjustSuccess": "Đã điều chỉnh số dư",
      "deduct": "Trừ",
      "noUsers": "Không tìm thấy người dùng",
      "reason": "Lý do điều chỉnh...",
      "searchPlaceholder": "Tìm theo email hoặc số tài khoản...",
      "title": "Quản Lý Ví",
      "walletFor": "Ví của",
      "unassigned": "Chưa gán",
      "userList": "Danh sách người dùng",
      "noMatch": "Không có người dùng khớp",
      "walletDetail": "Chi tiết ví",
      "transactions": "Giao dịch",
      "adjustReason": "Lý do",
      "tabWallets": "User Wallets",
      "tabDepositAddresses": "Deposit Addresses"
    },
    "header": {
      "admin": "管理",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "Đăng xuất"
    },
    "sidebar": {
      "accountManagement": "Quản Lý Tài Khoản",
      "agentSettings": "Cài Đặt Agent",
      "aiGateway": "Cổng AI",
      "billing": "Quản Lý Thanh Toán",
      "dashboard": "Bảng điều khiển",
      "deposits": "Quản Lý Nạp Tiền",
      "jurisdiction": "Kiểm Soát Quyền Hạn",
      "monitoring": "Giám Sát & Cảnh Báo",
      "operationLogs": "Nhật Ký Thao Tác",
      "shareManagement": "Phân Tích Chia Sẻ",
      "sre": "Điều Khiển SRE",
      "strategies": "Quản Lý Chiến Lược",
      "systemConfig": "Cấu Hình Hệ Thống",
      "tradingMonitor": "Giám Sát Giao Dịch",
      "userManagement": "Quản Lý Người Dùng",
      "walletManagement": "Quản Lý Ví",
      "sweep": "Quản lý Sweep",
      "autogenTasks": "Tác vụ AI Gen",
      "marketplace": "Marketplace",
      "refunds": "Hoàn tiền",
      "analytics": "Phân tích",
      "coupons": "Quản lý mã giảm giá"
    },
    "trading": {
      "accounts": "Tài Khoản",
      "activeUsers": "Người Dùng Hoạt Động",
      "byPlatform": "按平台",
      "closedOrders": "Đã Đóng",
      "connectedAccounts": "Đã Kết Nối",
      "loadFailed": "加载交易统计失败",
      "netProfit": "Lợi Nhuận Ròng",
      "orders": "Lệnh",
      "pendingOrders": "挂单",
      "platform": "Nền Tảng",
      "profitStats": "Thống Kê Lợi Nhuận",
      "title": "Giám Sát Giao Dịch",
      "totalAccounts": "Tổng Tài Khoản",
      "totalLoss": "Tổng Thua Lỗ",
      "totalOrders": "Tổng Lệnh",
      "totalProfit": "Tổng Lợi Nhuận",
      "totalUsers": "Tổng Người Dùng",
      "totalVolume": "Tổng Khối Lượng",
      "volume": "Khối Lượng"
    },
    "walletCalculator": {
      "title": "Máy tính Token ↔ USD",
      "selectModel": "Chọn mô hình (cơ sở định giá)",
      "usdAmount": "Số tiền USD",
      "tokenAmount": "Số lượng Token",
      "fillResult": "Điền kết quả"
    },
    "autogen": {
      "loadFailed": "Failed to load tasks",
      "approved": "Task approved and published",
      "approveFailed": "Approve failed",
      "rejected": "Task rejected",
      "rejectFailed": "Reject failed",
      "triggerFailed": "Trigger failed",
      "symbol": "Symbol",
      "timeframe": "TF",
      "strategyType": "Type",
      "status": "Status",
      "quality": "Quality",
      "error": "Error",
      "actions": "Actions",
      "confirmApprove": "Approve and publish?",
      "approve": "Approve",
      "confirmReject": "Reject this task?",
      "reject": "Reject",
      "title": "AI Strategy Generation Tasks",
      "allStatus": "All Status",
      "refresh": "Refresh",
      "triggerBatch": "Trigger Batch",
      "enqueue": "Enqueue",
      "symbols": "Symbols (comma-separated)",
      "timeframes": "Timeframes (comma-separated)",
      "strategyTypes": "Strategy Types (comma-separated)"
    },
    "coupon": {
      "loadFailed": "Failed to load coupons",
      "fillRequired": "Please fill required fields",
      "created": "Coupon created",
      "createFailed": "Failed to create coupon",
      "disabled": "Coupon disabled",
      "disableFailed": "Failed to disable coupon",
      "colCode": "Code",
      "colType": "Type",
      "colValue": "Value",
      "colMinPurchase": "Min Purchase",
      "colUsage": "Usage",
      "colExpires": "Expires",
      "colStatus": "Status",
      "colActions": "Actions",
      "disable": "Disable",
      "create": "Create Coupon",
      "createTitle": "Create Coupon",
      "codePlaceholder": "Coupon code (e.g. SUMMER20)",
      "valuePlaceholder": "Discount value (e.g. 20 for 20% or 50 for $50)",
      "minPurchasePlaceholder": "Minimum purchase amount (0 = none)",
      "maxUsesPlaceholder": "Max uses (0 = unlimited)",
      "expiresPlaceholder": "Expires at (ISO 8601, empty = never)"
    },
    "depositAddresses": {
      "importFailed": "Import failed",
      "address": "Address",
      "user": "User ID",
      "index": "Index",
      "status": "Status",
      "received": "Received USDT",
      "network": "Network",
      "assignedAt": "Assigned At",
      "importHint": "Use hdgen tool on an offline machine to generate deposit_addresses.bin, then upload it here.",
      "all": "All Status",
      "import": "Import Addresses",
      "availablePool": "Available in Pool",
      "total": "Total Addresses"
    },
    "analytics": {
      "name": "Name",
      "value": "Value",
      "platformRev": "Platform Rev",
      "providerRev": "Provider Rev",
      "activeBuyers": "Active Buyers",
      "refundRate": "Refund Rate",
      "totalTx": "Transactions",
      "newSubs": "New Subscribers",
      "totalStrategies": "Total Strategies",
      "newStrategies": "New Strategies",
      "topByRevenue": "Top Strategies by Revenue",
      "topBySubs": "Top Strategies by Subscribers",
      "topProvidersRev": "Top Providers by Revenue",
      "topProvidersStrat": "Top Providers by Strategies"
    },
    "marketplace": {
      "loadFailed": "Failed to load strategies",
      "featureSuccess": "Strategy featured",
      "featureFailed": "Failed to feature strategy",
      "unfeatureSuccess": "Removed featured",
      "unfeatureFailed": "Failed to unfeature",
      "colTitle": "Title",
      "colPublisher": "Publisher",
      "colStatus": "Status",
      "colPrice": "Price",
      "colSales": "Sales",
      "colRevenue": "Revenue",
      "colFeatured": "Featured",
      "colActions": "Actions",
      "feature": "Feature",
      "unfeature": "Remove featured",
      "filterStatus": "All statuses",
      "searchPlaceholder": "Search by title...",
      "featureTitle": "Feature Strategy",
      "featureDesc": "Set priority for featured placement. Higher = more prominent."
    },
    "refund": {
      "loadFailed": "Failed to load refund requests",
      "approved": "Refund approved and executed",
      "rejected": "Refund request rejected",
      "processFailed": "Failed to process refund",
      "colUser": "User",
      "colStrategy": "Strategy",
      "colAmount": "Amount",
      "colReason": "Reason",
      "colStatus": "Status",
      "colDate": "Date",
      "colActions": "Actions",
      "approve": "Approve & Execute",
      "reject": "Reject",
      "filterStatus": "All statuses",
      "approveTitle": "Approve Refund",
      "rejectTitle": "Reject Refund",
      "reviewNotePlaceholder": "Review note (optional for reject, recommended for approve)..."
    }
  },
  "strategy": {
    "backtest": {
      "diagnostic": {
        "suggestion": {
          "iCustom": "iCustom (custom indicator) is not supported — replace with a built-in indicator (iMA/iRSI/iMACD etc.) or implement the logic manually",
          "dll": "DLL imports are not supported — remove external DLL calls and use built-in MQL functions"
        },
        "invariant": "Invariant Violation",
        "defenseA": "Structural Validation",
        "lookahead": "Lookahead Bias",
        "statistical": "Statistical Hint",
        "unknown": "Diagnostic",
        "coverage": "Coverage",
        "compatible": "compatible",
        "unsupported": "unsupported",
        "fatal": "Critical Issues",
        "suggestionLabel": "建议",
        "warning": "Risk Warnings",
        "silenceHint": "Acknowledge as intentional — hide this warning",
        "allSilenced": "All warnings acknowledged as intentional",
        "info": "Quality Hints",
        "aiFix": "AI Fix",
        "noCode": "No strategy code to fix",
        "aiNoResult": "AI returned no code",
        "aiFailed": "AI fix failed",
        "fixApplied": "Fix applied — re-running backtest",
        "fixAppliedCompileWarn": "Fix applied but compile has warnings",
        "applyFailed": "Failed to apply fix",
        "saveFirst": "Please save the strategy first to apply AI fixes",
        "diffPreview": "AI Fix Preview",
        "apply": "Apply & Re-run",
        "diffHint": "Review the AI-generated code below. Apply to create a new version and re-run backtest."
      },
      "canceled": "Backtest bị hủy",
      "lotSize": "Khối lượng lô",
      "strategyParameters": "Tham số chiến lược",
      "autoGate": "Auto Gate Evaluation",
      "publishable": "Publishable",
      "notPublishable": "Not Publishable",
      "cancelFailed": "Cancel failed"
    },
    "templates": {
      "scheduleLaunch": {
        "metrics": {
          "winRate": "Win Rate",
          "maxDrawdown": "Max Drawdown",
          "sharpe": "Sharpe Ratio"
        }
      },
      "gallery": {
        "title": "Strategies",
        "system": "System",
        "shared": "Shared",
        "forkEdit": "Fork & Edit",
        "aiGenerate": "AI Generate",
        "searchPlaceholder": "Search strategies...",
        "filterAll": "All",
        "filterMine": "Mine",
        "filterSystem": "System",
        "sortRecent": "Recent",
        "sortReturn": "Return",
        "sortRisk": "Risk",
        "sortUsage": "Usage",
        "empty": "No strategies found",
        "forkSuccess": "Forked to new strategy",
        "forkFailed": "Fork failed",
        "unpublishSuccess": "Unpublished",
        "unpublishFailed": "Unpublish failed",
        "deleteFailed": "Delete failed",
        "deploy": "Deploy",
        "publish": "Publish",
        "unpublish": "Unpublish",
        "fork": "Fork"
      },
      "actions": {
        "deploy": "Deploy",
        "create": "New Strategy",
        "delete": "Delete"
      },
      "detail": {
        "profitFactor": "Profit Factor",
        "notFound": "Strategy not found",
        "openInWorkspace": "Open in Workspace",
        "overview": "Overview",
        "noDescription": "No description",
        "equityCurve": "Equity Curve",
        "tradeStats": "Trade Statistics",
        "parameters": "Parameters"
      },
      "table": {
        "useCount": "Use Count",
        "createdAt": "Created",
        "visibility": "Visibility",
        "status": "Status"
      },
      "visibility": {
        "public": "Public",
        "private": "Private"
      },
      "codeModal": {
        "title": "Code"
      },
      "messages": {
        "fetchTemplateListFailed": "Failed to load strategies",
        "publishFailed": "Publish failed",
        "templateDeleted": "Deleted"
      },
      "title": "Mẫu Chiến lược",
      "saveCurrent": "Lưu Chiến lược Hiện tại",
      "lines": "dòng",
      "chatEdit": "Chỉnh sửa Chat",
      "source": "Nguồn",
      "rename": "Đổi tên",
      "confirmDelete": "Xóa chiến lược này?",
      "noTemplates": "Không có mẫu chiến lược đã lưu",
      "sourceCode": "Mã nguồn Chiến lược",
      "copyAll": "Sao chép tất cả",
      "deleteConfirm": "Delete this strategy?",
      "loadFailed": "Failed to load templates",
      "loadOneFailed": "Failed to load template"
    },
    "workspace": {
      "chartIndicators": {
        "overlay": "Chỉ báo chồng (biểu đồ chính)",
        "subPane": "Chỉ báo khung phụ"
      },
      "sidebar": {
        "noRuns": "No backtest runs yet",
        "batchDeleteRunsConfirm": "Delete selected runs?",
        "trades": "trades",
        "deleteRunConfirm": "Delete this backtest run?",
        "viewAll": "View all",
        "noStrategies": "No strategies yet",
        "batchDeleteConfirm": "Delete selected strategies?",
        "deleteStrategyConfirm": "Delete this strategy?",
        "title": "Workspace",
        "myStrategies": "My Strategies",
        "backtestHistory": "Backtest History",
        "newStrategy": "New Strategy"
      },
      "tour": {
        "ai": "AI Assistant",
        "aiDesc": "Ask AI to generate, optimize, or debug your strategy. Applied code appears in the editor instantly.",
        "code": "Code Editor",
        "codeDesc": "Write or paste your MQL strategy code here. You can also import .mq4/.mq5 files from the Import MQL tab.",
        "backtest": "Backtest",
        "backtestDesc": "Run backtests with configurable parameters. View equity curve, trade statistics, and risk metrics.",
        "save": "Save & Publish",
        "saveDesc": "Save your strategy as a template, publish to marketplace, or deploy to a live schedule."
      },
      "importMql": "Import MQL"
    },
    "tuning": {
      "searchMethod": {
        "grid": "Lưới",
        "random": "Ngẫu nhiên"
      },
      "noParams": {
        "title": "No tunable parameters detected",
        "desc": "Add @param annotations to your strategy code to enable Smart Tuning. Example: // @param fastPeriod 14 range=5:30:5"
      },
      "strategyName": "Strategy",
      "totalTrades": "Trades",
      "disabledHint": "Need strategy code and symbol. Select a strategy from the sidebar or run a backtest first.",
      "noDimsHint": "Enable at least one parameter dimension below.",
      "qualityGate": "Gate",
      "failed": "Tuning failed"
    },
    "schedules": {
      "status": {
        "enabled": "Đã bật",
        "running": "Đang chạy",
        "idle": "Chờ",
        "disabled": "Đã tắt"
      },
      "actions": {
        "runNow": "Chạy ngay"
      },
      "deleteConfirm": {
        "title": "Xóa lịch trình này?"
      },
      "table": {
        "schedule": "Lịch trình"
      }
    },
    "chat": {
      "executionPlan": "Kế hoạch thực hiện",
      "codeGenerated": "Đã tạo mã. Sử dụng các nút bên dưới để chạy đánh giá chiến lược và backtest.",
      "entry": "Entry:",
      "exit": "Exit:",
      "risk": "Risk:",
      "indicators": "Indicators:"
    },
    "aiChat": {
      "historyTab": "Lịch sử",
      "strategiesTab": "Chiến lược",
      "codeLoaded": "Strategy code in context",
      "noContext": "No strategy loaded — describe what you want"
    },
    "live": {
      "stopSuccess": "Chiến lược đã dừng",
      "stopFailed": "Dừng thất bại",
      "runId": "ID Lần chạy",
      "account": "Tài khoản",
      "symbol": "Mã giao dịch",
      "timeframe": "Khung thời gian",
      "mode": "Chế độ",
      "signals": "Tín hiệu",
      "errors": "Lỗi",
      "startedAt": "Bắt đầu",
      "watchSignals": "Xem Tín hiệu",
      "confirmStop": "Dừng chiến lược này?",
      "status": "Trạng thái",
      "totalSignals": "Tổng Tín hiệu",
      "stoppedAt": "Thời gian dừng",
      "error": "Lỗi",
      "title": "Giám sát Chiến lược Trực tiếp",
      "activeTab": "Các lần chạy Đang hoạt động",
      "noActive": "Không có chiến lược đang hoạt động",
      "historyTab": "Lịch sử Chạy",
      "noRuns": "Không có lần chạy chiến lược nào",
      "schedulesTab": "Lịch trình",
      "time": "Thời gian",
      "signalType": "Loại",
      "volume": "Khối lượng",
      "price": "Giá",
      "sl": "SL",
      "tp": "TP",
      "reason": "Lý do",
      "signalLog": "Nhật ký tín hiệu",
      "waitingSignals": "Đang chờ tín hiệu...",
      "myStrategies": "Chiến lược của tôi",
      "temporaryRuns": "Chạy tạm thời",
      "positions": "Vị thế",
      "noPositions": "Không có vị thế",
      "config": "Cấu hình",
      "parameters": "Tham số",
      "runStarted": "Đã bắt đầu chạy",
      "runStartFailed": "Khởi động thất bại",
      "strategyName": "Chiến lược",
      "stale": "lỗi thời",
      "lastSignal": "Tín hiệu gần nhất",
      "pnl": "Lợi nhuận",
      "unknownError": "Lỗi không xác định",
      "logs": "Nhật ký",
      "health": "Sức khỏe",
      "streamDisconnected": "Kết nối bị gián đoạn, đang kết nối lại…",
      "goSchedules": "Đến lịch trình",
      "positionClosed": "Vị thế đã đóng",
      "closeFailed": "Đóng vị thế thất bại"
    },
    "schedule": {
      "maxPositionsPlaceholder": "Không giới hạn"
    },
    "ai": {
      "reviseHint": "Viết mã trước, sau đó yêu cầu AI cải thiện.",
      "explainHint": "Viết mã để xem giải thích từ AI.",
      "settingsHint": "Cấu hình nhà cung cấp AI và mô hình"
    },
    "validate": {
      "running": "Đang xác thực...",
      "errors": "Lỗi",
      "warnings": "Cảnh báo",
      "fixWithAI": "Gửi lỗi cho AI Sửa đổi",
      "parameters": "Tham số",
      "hints": "Gợi ý",
      "allClear": "Tất cả kiểm tra đã thông qua — không tìm thấy vấn đề",
      "passed": "Xác nhận thành công — Chức năng Lưu đã được mở khóa",
      "autoFixFailed": "Auto-fix failed",
      "failed": "Validation failed"
    },
    "importEA": {
      "writeTab": "Mã Chiến lược",
      "importTab": "Nhập EA",
      "codeTooShort": "Vui lòng dán mã nguồn EA/chỉ báo đầy đủ.",
      "pastePlaceholder": "Dán mã EA MQL4/MQL5...",
      "migration": "Nhập chiến lược",
      "aiTranslate": "AI Dịch",
      "bridge": "Cầu nối điểm mù",
      "analyze": "Phân tích cấu trúc chiến lược",
      "confirmImport": "Xác nhận nhập",
      "tryAI": "Bổ sung bản dịch AI",
      "apply": "Áp dụng vào Trình soạn thảo",
      "importSuccess": "'Mã nguồn MQL đã được nhập, nhấp vào「Apply to Editor」để ghi vào trình soạn thảo'",
      "hint": "Dán mã MQL4/MQL5 và nhấp Phân tích",
      "translate": "Dịch sang Go",
      "translating": "AI đang dịch...",
      "bridgeBtn": "Dịch cầu nối điểm mù",
      "bridgeSuccess": "Cầu nối thành công",
      "bridgeFailedTag": "Cầu nối thất bại",
      "bridging": "AI đang bắc cầu điểm mù...",
      "bridgeFailedMsg": "Agent không thể tự động bắc cầu tất cả điểm mù",
      "noBridgeNeeded": "Độ phủ 100%, không cần bridge",
      "bridgeHint": "Dán mã EA MQL4/MQL5, AI sẽ tự động chuyển vùng mù thành tập con Python"
    },
    "version": {
      "loadFailed": "Không tải được các phiên bản",
      "rollbackFailed": "Khôi phục thất bại",
      "loadVersionFailed": "Không thể tải phiên bản",
      "loadDiffFailed": "Không thể tải sự khác biệt (diff)",
      "colVersion": "Phiên bản",
      "colSummary": "Tóm tắt thay đổi",
      "colLang": "Ngôn ngữ",
      "colHash": "Hash",
      "colDate": "Ngày",
      "colActions": "Hành động",
      "title": "Lịch sử phiên bản",
      "diff": "Khác biệt",
      "empty": "Chưa có lịch sử phiên bản",
      "history": "Lịch sử phiên bản",
      "rollbackSuccess": "Đã khôi phục về phiên bản {{n}}",
      "rollbackConfirm": "Khôi phục về v{{n}}?",
      "diffTitle": "Khác biệt: v{{from}} → v{{to}}",
      "viewTitle": "Phiên bản {{n}}",
      "diffFrom": "Từ",
      "diffTo": "Đến"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "Hành Động",
        "price": "Giá",
        "profit": "Lãi/Lỗ",
        "symbol": "Mã",
        "ticket": "单号",
        "time": "Thời Gian",
        "volume": "Khối Lượng"
      },
      "empty": "Chưa có nhật ký giao dịch",
      "title": "Nhật Ký Giao Dịch Gần Đây"
    },
    "messages": {
      "loadFailed": "Tải dữ liệu giao dịch tự động thất bại",
      "toggleFailed": "切换自动交易失败"
    },
    "settings": {
      "maxDailyLoss": "Lỗ Tối Đa Hàng Ngày",
      "maxDailyLossHint": "Tự động tắt giao dịch nếu lỗ hàng ngày vượt quá mức này",
      "maxDrawdownPercent": "Sụt Giảm Tối Đa %",
      "maxDrawdownPercentHint": "Tự động tắt giao dịch nếu drawdown vượt quá mức này",
      "maxLotSize": "Lot Tối Đa",
      "maxLotSizeHint": "Khối lượng tối đa mỗi giao dịch (lots)",
      "maxPositions": "Vị Thế Tối Đa",
      "maxPositionsHint": "Số vị thế mở tối đa",
      "maxRiskPercent": "Rủi Ro Tối Đa %",
      "maxRiskPercentHint": "Phần trăm số dư để rủi ro mỗi giao dịch",
      "saveFailed": "保存设置失败",
      "saveSuccess": "Đã lưu cài đặt",
      "title": "Cài Đặt Rủi Ro Toàn Cục"
    },
    "status": {
      "activeStrategies": "Chiến Lược Đang Hoạt Động",
      "disabled": "Giao Dịch Tự Động Đã Tắt",
      "enabled": "Giao Dịch Tự Động Đã Bật",
      "todayExecutions": "Lệnh hôm nay",
      "todayProfit": "Lợi nhuận hôm nay"
    },
    "title": "Giao Dịch Tự Động"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "自动交易事件触发",
        "title": "Giao Dịch Tự Động"
      },
      "riskAlert": {
        "fallback": "警报类型: {{alertType}}",
        "title": "Cảnh báo Rủi ro"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} đã hoàn thành",
        "failed": "执行失败: {{error}}",
        "title": "Thực thi Chiến lược"
      },
      "strategySignal": {
        "message": "{{symbol}} kích hoạt {{signalType}}",
        "title": "Tín hiệu Chiến lược"
      }
    },
    "actions": {
      "clearAll": "Xóa",
      "clearAllConfirm": "Xóa tất cả thông báo?",
      "markAllAsRead": "Đánh dấu đã đọc"
    },
    "tabs": {
      "all": "Tất cả ({{count}})",
      "unread": "未读 ({{count}})"
    },
    "types": {
      "risk_alert": "Cảnh báo Rủi ro",
      "signal": "Tín hiệu",
      "strategy_execution": "Chiến lược",
      "system": "系统",
      "trade": "Giao dịch"
    },
    "prefs": {
      "saveFailed": "Failed to save preferences",
      "newStrategy": "New strategy published",
      "priceChange": "Strategy price changed",
      "subExpiring": "Subscription expiring soon",
      "performance": "Strategy performance anomaly",
      "newRating": "New rating or comment received",
      "title": "Notification Preferences"
    },
    "all": "Tất cả",
    "clearAll": "Xóa",
    "confirmClearAll": "Xóa tất cả thông báo?",
    "empty": "Không có thông báo",
    "markAllRead": "Đánh dấu đã đọc",
    "title": "Thông báo",
    "unread": "Chưa đọc"
  },
  "wallet": {
    "deposit": {
      "table": {
        "amount": "Số Tiền USDT",
        "amountUsd": "USD Nhận",
        "status": "Trạng Thái",
        "time": "Thời Gian",
        "txHash": "Mã GD",
        "confirmations": "Xác nhận"
      },
      "address": "Địa Chỉ Nhận",
      "addressCopied": "Đã sao chép địa chỉ vào clipboard",
      "amountLabel": "Số Tiền USDT",
      "button": "Nạp Tiền Mới",
      "copy": "Sao Chép",
      "exchangeRate": "Tỷ Giá",
      "failed": "Gửi yêu cầu nạp tiền thất bại.",
      "history": "Lịch Sử Nạp",
      "modalTitle": "Gửi Yêu Cầu Nạp Tiền",
      "network": "Mạng",
      "notConfigured": "Nạp USDT chưa được cấu hình. Vui lòng liên hệ hỗ trợ.",
      "notice": "Chỉ gửi USDT qua mạng được chỉ định. Gửi token khác hoặc sử dụng mạng khác có thể gây mất vĩnh viễn.",
      "submit": "Gửi",
      "success": "Yêu cầu nạp tiền đã được gửi. Tiền nạp của bạn sẽ được xác nhận tự động.",
      "title": "Nạp Tiền",
      "txHashLabel": "Mã giao dịch (tùy chọn)",
      "willCredit": "Sẽ nhận"
    },
    "table": {
      "amount": "Số Tiền",
      "balanceAfter": "Số Dư Sau",
      "description": "Mô Tả",
      "time": "Thời Gian",
      "type": "Loại"
    },
    "txType": {
      "adjustment": "Điều Chỉnh",
      "deposit": "Nạp Tiền",
      "fee": "Phí",
      "reversal": "Hoàn Tác",
      "withdrawal": "Rút Tiền"
    },
    "passkey": {
      "title": "Quản lý Passkey",
      "add": "Thêm Passkey",
      "name": "Tên",
      "credentialId": "ID Chứng chỉ",
      "signCount": "Số lần ký",
      "createdAt": "Thời gian tạo",
      "confirmRemove": "Xóa passkey này?",
      "register": "Đăng ký",
      "registered": "Đã đăng ký passkey thành công",
      "registerFailed": "Đăng ký thất bại",
      "registerHint": "Nhập tên cho passkey này, sau đó nhấn Đăng ký để bắt đầu quy trình WebAuthn.",
      "namePlaceholder": "VD: YubiKey của tôi",
      "removed": "Đã xóa passkey"
    },
    "withdraw": {
      "title": "Rút tiền",
      "new": "Rút tiền mới",
      "submit": "Gửi",
      "available": "Số dư khả dụng",
      "amount": "Số tiền",
      "amountLabel": "Số tiền rút (USDT)",
      "amountRequired": "Vui lòng nhập số tiền",
      "destAddress": "Địa chỉ đích",
      "destLabel": "Địa chỉ TRC20 đích",
      "destRequired": "Vui lòng nhập địa chỉ đích",
      "whitelist": "Danh sách trắng (nhấp để điền)",
      "status": "Trạng thái",
      "txHash": "Mã giao dịch",
      "time": "Thời gian",
      "cancelled": "Đã hủy rút tiền",
      "confirmCancel": "Hủy lệnh rút tiền này?",
      "success": "Đã gửi yêu cầu rút tiền thành công",
      "failed": "Rút tiền thất bại",
      "noBalance": "Không có số dư khả dụng để rút",
      "warning": "Rút tiền cần xác thực passkey. Vui lòng đảm bảo địa chỉ đích chính xác — giao dịch blockchain không thể hoàn tác."
    },
    "whitelist": {
      "title": "Quản lý danh sách trắng",
      "add": "Thêm địa chỉ",
      "added": "Đã thêm địa chỉ danh sách trắng",
      "removed": "Đã xóa địa chỉ danh sách trắng",
      "label": "Nhãn",
      "address": "Địa chỉ",
      "status": "Trạng thái",
      "confirmedAt": "Thời gian xác nhận",
      "confirmRemove": "Xóa địa chỉ danh sách trắng này?",
      "addressLabel": "Địa chỉ TRC20",
      "addressRequired": "Vui lòng nhập địa chỉ",
      "labelLabel": "Nhãn (tùy chọn)",
      "labelPlaceholder": "VD: Ví Binance của tôi"
    },
    "accountNumber": "Số TK",
    "balance": "Số Dư",
    "currency": "Tiền Tệ",
    "frozen": "Đóng Băng",
    "frozenBalance": "Đóng Băng",
    "history": "Lịch Sử",
    "title": "Ví Của Tôi",
    "transactions": "Giao Dịch"
  },
  "accounts": {
    "bind": {
      "fields": {
        "alias": "Bí danh Tài khoản"
      },
      "placeholders": {
        "alias": "Tên tùy chỉnh (tùy chọn)"
      },
      "messages": {
        "changeCredentials": "Thay đổi thông tin đăng nhập"
      }
    },
    "messages": {
      "shareLinkCopied": "Đã sao chép liên kết chia sẻ",
      "shareLinkFailed": "Tạo liên kết chia sẻ thất bại"
    },
    "status": {
      "circuit_open": "Circuit Open",
      "circuit_half_open": "Circuit Testing"
    }
  },
  "sre": {
    "breakers": {
      "columns": {
        "strategyId": "ID Chiến lược",
        "state": "Trạng thái",
        "totalPnl": "Tổng P&L",
        "lossPercent": "Tỷ lệ lỗ %",
        "tradeCount": "Số giao dịch",
        "trippedAt": "Ngắt lúc",
        "tripReason": "Lý do ngắt"
      },
      "title": "Bộ ngắt mạch chiến lược",
      "stateClosed": "Bình thường",
      "stateOpen": "Ngắt",
      "stateHalfOpen": "Bán mở (đang thăm dò)",
      "confirmReset": "Đặt lại bộ ngắt này?",
      "description": "Tổng quan trạng thái bộ ngắt chiến lược — tự động phát hiện thua lỗ bất thường và ngắt",
      "noBreakers": "Không có bộ ngắt nào được đăng ký"
    },
    "canary": {
      "columns": {
        "strategyId": "ID chiến lược",
        "versionTag": "Tag phiên bản",
        "accounts": "Tài khoản Canary",
        "startAt": "Bắt đầu lúc",
        "days": "Ngày",
        "status": "Trạng thái"
      },
      "promoted": "Đã phát hành",
      "canarying": "Đang Canary",
      "confirmDelete": "Xóa cấu hình canary này?",
      "title": "Cấu hình Canary",
      "description": "Phiên bản chiến lược mới chạy trên một số tài khoản trong N ngày trước khi triển khai cho tất cả",
      "newCanary": "Canary mới",
      "noCanaries": "Không có cấu hình canary nào",
      "newCanaryTitle": "Canary mới",
      "accountIdsLabel": "ID tài khoản Canary (phân cách bằng dấu phẩy hoặc xuống dòng)",
      "durationDays": "Số ngày Canary",
      "accountIdsPlaceholder": "account-1, account-2"
    },
    "killSwitch": {
      "description": "Dừng tất cả giao dịch chỉ bằng một cú nhấp — yêu cầu xác nhận KILL; có thể hoàn tác trong 5 phút",
      "engaged": "Công tắc khẩn cấp đã kích hoạt — tất cả giao dịch đã dừng",
      "disarmed": "Công tắc khẩn cấp đã tắt — giao dịch bình thường",
      "status": "Trạng thái",
      "reason": "Lý do",
      "operator": "Người vận hành",
      "engagedAt": "Kích hoạt lúc",
      "undo": "Hoàn tác công tắc khẩn cấp",
      "disengage": "Tắt công tắc khẩn cấp",
      "engage": "Kích hoạt công tắc khẩn cấp",
      "confirmTitle": "Kích hoạt Công tắc Khẩn cấp — Xác nhận",
      "confirmEngage": "Xác nhận kích hoạt",
      "confirmWarning": "Hành động này sẽ ngay lập tức dừng mọi hoạt động giao dịch cho tất cả tài khoản, bao gồm các lệnh đang chờ và đã gửi. Nhập lý do và gõ KILL để xác nhận.",
      "reasonLabel": "Lý do (bắt buộc)",
      "reasonPlaceholder": "v.d.: Phát hiện biến động thị trường bất thường, dừng khẩn cấp tất cả giao dịch",
      "typeKill": "Gõ KILL để xác nhận",
      "typeKillPlaceholder": "Gõ KILL (chữ hoa)",
      "undoWindow": "Cửa sổ hoàn tác: còn {{minutes}}m {{seconds}}s",
      "title": "Kill Switch"
    }
  },
  "marketplace": {
    "publish": {
      "priceModel": {
        "free": "Miễn phí",
        "subscription": "Đăng ký hàng tháng",
        "once": "Mua một lần",
        "label": "Giá cả"
      },
      "assetClass": {
        "label": "Loại tài sản"
      },
      "riskLevel": {
        "label": "Mức độ rủi ro"
      },
      "trialDays": {
        "7": "7 days",
        "14": "14 days",
        "30": "30 days"
      },
      "return": "Lợi nhuận",
      "winRate": "Tỷ lệ thắng",
      "trades": "Giao dịch",
      "title": "Xuất bản lên Thị trường",
      "titleLabel": "Tiêu đề",
      "titlePlaceholder": "ví dụ: Chiến lược Golden Cross",
      "descriptionLabel": "Mô tả",
      "descriptionPlaceholder": "Mô tả logic chiến lược, quy tắc vào/ra lệnh...",
      "priceAmount": "Số tiền",
      "tags": "Thẻ",
      "tagsPlaceholder": "Nhập và nhấn Enter để thêm thẻ",
      "codeSnippet": "Xem trước Chiến lược (công khai)",
      "codeSnippetPlaceholder": "'Tùy chọn: chia sẻ đoạn mã hoặc ý tưởng cấp cao về chiến lược (hiển thị cho tất cả)'",
      "includeBacktestSnapshot": "Bao gồm kết quả backtest gần nhất",
      "trialDaysLabel": "Trial Period",
      "trialDaysPlaceholder": "Select or enter custom days",
      "trialDaysCustom": "Custom days"
    },
    "author": {
      "avgRating": "Đánh Giá TB",
      "empty": "Chưa có chiến lược nào được xuất bản. Vào Thư Viện Chiến Lược để xuất bản.",
      "published": "Đã Xuất Bản",
      "myStrategies": "Chiến lược Đã Phát hành",
      "publishNew": "Phát hành Chiến lược Mới",
      "monthlyRevenue": "Doanh thu Hàng tháng",
      "totalRevenue": "Tổng Doanh thu",
      "goToLibrary": "Đến Thư viện Chiến lược"
    },
    "card": {
      "by": "của",
      "free": "Miễn Phí",
      "owned": "Ngày Mua",
      "subscribers": "Người Đăng Ký",
      "winRate": "Tỷ Lệ Thắng",
      "yourStrategy": "Chiến lược của Bạn"
    },
    "detail": {
      "assetClass": "Loại Tài Sản",
      "author": "Tác Giả",
      "commentPlaceholder": "Viết bình luận...",
      "comments": "Bình Luận",
      "description": "Mô Tả",
      "getFree": "Nhận Miễn Phí",
      "rentPrice": "¥{{amount}} / tháng",
      "subscribers": "Người Đăng Ký",
      "yourRating": "Đánh Giá Của Bạn",
      "runBacktest": "Chạy Backtest"
    },
    "messages": {
      "commentFailed": "Bình luận thất bại",
      "commentPosted": "Đã đăng bình luận",
      "loginFirst": "Vui lòng đăng nhập trước",
      "paymentComingSoon": "Thanh toán sắp ra mắt",
      "rateFailed": "Đánh giá thất bại",
      "rated": "Đã gửi đánh giá",
      "subscribeFailed": "Thất Bại",
      "subscribed": "Đã thêm vào mục đã mua",
      "published": "Chiến lược đã được xuất bản lên thị trường!",
      "publishFailed": "Xuất bản chiến lược thất bại"
    },
    "payment": {
      "alreadyPurchased": "Bạn đã sở hữu chiến lược này.",
      "balanceAfter": "Số dư sau khi mua",
      "cancel": "Hủy",
      "confirm": "Xác Nhận Mua",
      "depositPrompt": "Vui lòng nạp tiền để tiếp tục.",
      "goToDeposit": "Nạp Tiền",
      "insufficientBalance": "Số dư không đủ",
      "oneTimePurchase": "¥{{amount}} mua đứt",
      "price": "Giá",
      "purchaseFailed": "Mua thất bại. Vui lòng thử lại.",
      "purchaseSuccess": "Mua thành công! Chiến lược đã được thêm vào thư viện.",
      "purchasing": "Đang xử lý...",
      "strategyName": "Chiến Lược",
      "title": "Xác Nhận Mua",
      "walletBalance": "Số Dư Của Bạn",
      "deployGuide": "Your strategy is ready to deploy.",
      "goDeploy": "Deploy Now"
    },
    "purchases": {
      "empty": "Chưa có giao dịch mua nào. Duyệt chợ để tìm chiến lược.",
      "status": "Trạng Thái",
      "strategy": "Chiến Lược",
      "runBacktest": "Chạy Backtest"
    },
    "sort": {
      "newest": "Mới Nhất",
      "performance": "Hiệu Suất Tốt Nhất",
      "popular": "Phổ Biến Nhất",
      "priceAsc": "Giá: Thấp đến Cao",
      "priceDesc": "Giá: Cao đến Thấp",
      "rating": "Đánh Giá Cao Nhất",
      "score": "Điểm Tổng Hợp"
    },
    "tabs": {
      "author": "Trung Tâm Tác Giả",
      "marketplace": "Chợ",
      "purchases": "Đã Mua",
      "subscriptions": "Đăng Ký"
    },
    "backtest": {
      "title": "Backtest Chiến lược",
      "capital": "Vốn",
      "commission": "Hoa hồng",
      "leverage": "Đòn bẩy",
      "completed": "Đã hoàn thành",
      "totalReturn": "Tổng Lợi nhuận",
      "maxDrawdown": "Sụt giảm Tối đa",
      "sharpe": "Sharpe",
      "winRate": "Tỷ lệ Thắng",
      "totalTrades": "Tổng số Giao dịch",
      "equityCurve": "Đường cong Vốn",
      "protected": "Mã chiến lược được bảo vệ. Backtest chạy trên máy chủ của chúng tôi.",
      "run": "Chạy Backtest",
      "idle": "Thiết lập tham số và chạy backtest"
    },
    "live": {
      "loadError": "Failed to load live performance data"
    },
    "optimization": {
      "decayScore": "Decay Score",
      "trigger": "Trigger",
      "sharpeDecline": "Sharpe Decline",
      "winRateDecline": "Win Rate Decline",
      "returnDelta": "Return Delta"
    },
    "empty": "Chưa có chiến lược nào được xuất bản",
    "filterByClass": "Lọc theo loại tài sản",
    "noSubscriptions": "Chưa có đăng ký nào",
    "searchPlaceholder": "Tìm kiếm chiến lược...",
    "subtitle": "Khám phá, mua và sử dụng chiến lược cộng đồng",
    "title": "Chợ Chiến Lược"
  },
  "schedule": {
    "launch": {
      "noAccount": {
        "bindButton": "Bind MT Account"
      }
    }
  },
  "onboarding": {
    "step1": {
      "title": "Liên kết tài khoản",
      "desc": "Liên kết tài khoản giao dịch MT4/MT5 để bắt đầu.",
      "action": "Liên kết tài khoản"
    },
    "step2": {
      "title": "Tạo chiến lược đầu tiên",
      "desc": "Sử dụng AI để tạo chiến lược giao dịch từ ngôn ngữ tự nhiên.",
      "action": "Mở không gian làm việc"
    },
    "step3": {
      "title": "Nâng cấp gói",
      "desc": "Mở khóa thêm AI tokens, chiến lược và giao dịch trực tiếp.",
      "action": "Xem các gói"
    },
    "subtitle": "Bắt đầu chỉ với 3 bước đơn giản",
    "dismiss": "Đã hiểu, bỏ qua"
  },
  "auth": {
    "fields": {
      "confirmPassword": "确认密码",
      "email": "用户邮箱",
      "password": "Mật Khẩu",
      "login": "Email/Tài khoản"
    },
    "forgotPassword": {
      "backToLogin": "返回登录",
      "hint": "Vui lòng liên hệ quản trị viên hoặc hỗ trợ để đặt lại mật khẩu.",
      "title": "Đặt lại Mật khẩu",
      "emailSent": "If the email exists, a reset link has been sent.",
      "mtVerified": "Identity verified. Redirecting to password reset.",
      "mtFailed": "MT credential verification failed.",
      "emailTab": "Email",
      "sendResetLink": "Send Reset Link",
      "mtTab": "MT Verify",
      "mtLogin": "MT Account Number",
      "mtLoginPlaceholder": "e.g. 12345678",
      "mtPassword": "MT Password",
      "mtPasswordPlaceholder": "MT trading password",
      "mtHint": "Enter your bound MT account credentials to verify your identity. Server and platform are detected automatically.",
      "verifyAndReset": "Verify & Reset Password",
      "adminTab": "Admin",
      "adminHint": "Please contact your administrator or support to reset your password."
    },
    "login": {
      "forgotPassword": "Quên mật khẩu?",
      "login": "Đăng nhập ngay",
      "noAccount": "没有账户？",
      "registerNow": "Đăng ký ngay",
      "rememberMe": "Ghi nhớ đăng nhập",
      "signingIn": "Đang đăng nhập...",
      "subtitle": "Đây là bản thử nghiệm và không chịu trách nhiệm"
    },
    "messages": {
      "fetchMeFailed": "加载用户信息失败",
      "loginFailed": "Đăng nhập thất bại. Vui lòng kiểm tra email và mật khẩu.",
      "loginSuccess": "Đăng nhập thành công",
      "logoutSuccess": "Đã đăng xuất",
      "registerFailed": "Đăng ký thất bại. Vui lòng thử lại sau.",
      "registerSuccess": "Đăng ký thành công. Vui lòng đăng nhập."
    },
    "register": {
      "haveAccount": "Đã có tài khoản?",
      "loginNow": "Đăng nhập ngay",
      "register": "Đăng ký",
      "signingUp": "Đang đăng ký...",
      "subtitle": "Tạo tài khoản mới"
    },
    "validation": {
      "confirmPasswordRequired": "Vui lòng xác nhận mật khẩu",
      "emailInvalid": "Vui lòng nhập email hợp lệ",
      "emailRequired": "Vui lòng nhập email",
      "passwordMin8": "Mật khẩu phải có ít nhất 8 ký tự",
      "passwordMismatch": "Mật khẩu không khớp",
      "passwordRequired": "Vui lòng nhập mật khẩu",
      "loginRequired": "Vui lòng nhập email hoặc số tài khoản của bạn"
    },
    "resetPassword": {
      "mismatch": "Passwords do not match.",
      "invalidToken": "Invalid or missing reset token.",
      "success": "Password has been reset. Please log in with your new password.",
      "failed": "Failed to reset password.",
      "title": "Set New Password",
      "newPassword": "New Password",
      "confirmRequired": "Please confirm your password",
      "confirmPassword": "Confirm Password",
      "submit": "Reset Password"
    }
  },
  "common": {
    "months": {
      "jan": "1月",
      "jul": "7月"
    },
    "time": {
      "day": "{{n}}ngày",
      "hour": "{{n}}giờ",
      "lessThanMinute": "<1分钟",
      "minute": "{{n}}ph"
    },
    "active": "Hoạt Động",
    "back": "Quay lại",
    "cancel": "Hủy",
    "clear": "Xóa",
    "close": "Đóng",
    "comingSoon": "Sắp Ra Mắt",
    "confirm": "Xác nhận",
    "copied": "Đã sao chép",
    "copy": "Sao chép",
    "copyFailed": "Sao chép thất bại",
    "create": "Tạo mới",
    "created": "Đã tạo",
    "currentPosition": "📊 Vị thế hiện tại",
    "delete": "Xóa",
    "deleteFailed": "Xóa thất bại",
    "deleteSelected": "Xóa {{count}} mục đã chọn",
    "deleted": "Đã xóa",
    "disable": "Vô Hiệu",
    "disabled": "Đã Tắt",
    "edit": "Chỉnh sửa",
    "enable": "Kích Hoạt",
    "enabled": "Đã bật",
    "error": "Lỗi",
    "gotIt": "Đã hiểu",
    "hideDetails": "Ẩn chi tiết",
    "inactive": "Không Hoạt Động",
    "indicatorSettings": "Cài đặt {{name}}",
    "lineColor": "Màu đường",
    "loading": "Đang tải...",
    "loadingFailed": "Tải thất bại",
    "next": "Tiếp theo",
    "no": "否",
    "noData": "Không có dữ liệu",
    "noOpenPositionsForSymbol": "Không có vị thế mở cho {{symbol}}",
    "none": "Không có",
    "ok": "Đồng ý",
    "operationFailed": "操作失败",
    "pageError": "Lỗi trang",
    "pageUnderDevelopment": "此页面开发中",
    "pleaseWait": "Vui lòng chờ...",
    "previous": "Quay lại",
    "refresh": "Làm mới",
    "remove": "Xóa",
    "required": "Bắt buộc",
    "retry": "Thử lại",
    "save": "Lưu",
    "saveFailed": "Lưu thất bại",
    "saveSuccess": "Đã Lưu",
    "searching": "Đang tìm...",
    "selectSymbolToViewChart": "Chọn mã để xem biểu đồ",
    "send": "Gửi",
    "showDetails": "Xem chi tiết",
    "totalItems": "Tổng {{count}} mục",
    "translate": "Dịch",
    "unexpectedError": "Đã xảy ra lỗi không mong muốn",
    "unknown": "Không Xác Định",
    "updated": "Đã cập nhật",
    "viewOriginal": "Xem nguyên văn",
    "viewTranslation": "Xem bản dịch",
    "yes": "Có",
    "you": "Bạn",
    "unsaved": "Chưa lưu",
    "saved": "Đã lưu",
    "unknownError": "Lỗi không xác định",
    "duplicateName": "Tên đã tồn tại",
    "step1Label": "Sàn giao dịch",
    "step2Label": "Thông tin đăng nhập",
    "step3Label": "Xác nhận",
    "unit": "đơn vị",
    "action": "Hành động",
    "on": "Bật",
    "off": "Tắt",
    "true": "Đúng",
    "false": "Sai",
    "success": "Thành công",
    "failed": "Thất bại",
    "reset": "Đặt lại",
    "status": "Trạng thái",
    "message": "Tin nhắn",
    "openPrice": "Giá mở",
    "currentPrice": "Giá hiện tại",
    "saving": "Đang lưu...",
    "selected": "selected"
  },
  "errors": {
    "ai": {
      "api_key_required": "API key là bắt buộc",
      "base_url_required": "Base URL là bắt buộc",
      "base_url_scheme_invalid": "Base URL phải bắt đầu bằng http:// hoặc https://",
      "base_url_should_not_end_with_chat_completions": "Base URL không được kết thúc bằng /chat/completions",
      "config_service_not_initialized": "Dịch vụ cấu hình AI chưa được khởi tạo",
      "config_valid": "Cấu hình AI hợp lệ",
      "failed_to_create_request": "Không thể tạo request",
      "forbidden_quota": "配额超限",
      "free_tier_exhausted": "Đã hết hạn mức miễn phí của AI. Vui lòng tắt “use free tier only” trong trang quản trị nhà cung cấp hoặc chuyển sang khóa trả phí.",
      "invalid_base_url": "Base URL không hợp lệ",
      "invalid_provider": "Nhà cung cấp không hợp lệ",
      "no_trade_data_available": "Không có dữ liệu giao dịch",
      "not_configured": "AI chưa được cấu hình. Vui lòng bật và cấu hình trong AI Settings trước.",
      "probe_ok": "OK",
      "probe_ok_no_models": "OK (không trả về models)",
      "provider_required": "Vui lòng chọn nhà cung cấp",
      "provider_returned_empty_message": "Nhà cung cấp AI trả về thông điệp rỗng",
      "rate_limited": "AI bị giới hạn tốc độ hoặc hết hạn mức (429/resource exhausted). Vui lòng thử lại sau.",
      "request_failed": "Yêu cầu API thất bại",
      "insufficient_balance_title": "Số dư không đủ",
      "insufficient_balance": "Số dư ví AI của bạn không đủ. Vui lòng nạp thêm trước khi tiếp tục."
    },
    "connection_failed": {
      "content": "无法连接到服务器，请检查网络后重试。",
      "title": "Kết nối thất bại"
    },
    "access_denied": "Từ chối truy cập",
    "account_connected": "Kết nối thành công",
    "account_connection_failed": "Không thể kết nối đến máy chủ giao dịch",
    "account_not_found": "Không tìm thấy tài khoản",
    "auto_trading_disabled": "Đã tắt giao dịch tự động",
    "auto_trading_enabled": "Đã bật giao dịch tự động",
    "email_already_registered": "Email đã được đăng ký",
    "invalid_credentials": "Thông tin đăng nhập không hợp lệ",
    "not_authenticated": "Chưa đăng nhập",
    "schedule_service_not_available": "Dịch vụ lịch biểu không khả dụng",
    "translate_failed": "Dịch thất bại",
    "user_not_found": "Không tìm thấy người dùng"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "Đã Tắt",
      "longOnly": "Chỉ Mua",
      "longShort": "Mua & Bán",
      "shortOnly": "Chỉ Bán",
      "unknown": "Không Xác Định"
    },
    "label": "Biểu tượng được Phát hiện",
    "loading": "Đang phân tích…",
    "noSymbols": "Không phát hiện biểu tượng giao dịch. Thử bao gồm tên biểu tượng cụ thể (ví dụ: \"Bitcoin\", \"EURUSD\", \"Vàng\").",
    "resolvedTooltip": "môi giới: {{broker}} | chế độ: {{mode}}",
    "unresolvedTooltip": "Chưa liên kết tài khoản giao dịch, không thể phân giải"
  },
  "subscription": {
    "feature": {
      "aiTokens": "{{count}} AI token/tháng",
      "strategies": "{{count}} chiến lược",
      "backtests": "{{count}} backtest/ngày",
      "liveStrategies": "{{count}} chiến lược live",
      "symbols": "{{count}} cặp/chiến lược",
      "unlimitedAccounts": "Unlimited MT accounts"
    },
    "title": "Gói Đăng Ký",
    "subscribeSuccess": "Đăng ký kích hoạt thành công!",
    "charged": "Đã tính phí: {{amount}}, Số dư: {{balance}}",
    "insufficientBalance": "Số dư ví không đủ. Vui lòng nạp tiền trước.",
    "subscribeFailed": "Đăng ký thất bại. Vui lòng thử lại.",
    "cancelSuccess": "Tự động gia hạn đã hủy. Đăng ký của bạn vẫn hiệu lực đến hết kỳ hiện tại.",
    "cancelFailed": "Hủy thất bại. Vui lòng thử lại.",
    "changeSuccess": "Đổi gói thành công!",
    "changeFailed": "Đổi gói thất bại. Vui lòng thử lại.",
    "billingCycle": "Thanh Toán",
    "autoRenew": "Tự động gia hạn",
    "period": "Kỳ hiện tại",
    "cancelAutoRenew": "Hủy tự động gia hạn",
    "usageTitle": "Sử Dụng Tháng Này",
    "aiTokens": "AI Token",
    "activeStrategies": "Chiến Lược Hoạt Động",
    "runtimeMinutes": "Thời Gian Chạy (phút)",
    "walletBalance": "Số Dư Ví",
    "month": "tháng",
    "year": "năm",
    "freeForever": "Miễn Phí Mãi Mãi",
    "currentPlan": "Gói Hiện Tại",
    "choosePlan": "Chọn Gói",
    "noPlans": "Không có gói nào",
    "changePlanTitle": "Đổi Gói",
    "subscribeTitle": "Đăng Ký Gói",
    "selectBillingCycle": "Chu Kỳ Thanh Toán",
    "monthly": "Hàng Tháng",
    "yearly": "Hàng Năm",
    "chargeNotice": "Gói trả phí sẽ được trừ từ ví. Gói miễn phí không tính phí.",
    "unbindSuccess": "Account unbound successfully.",
    "unbindFailed": "Failed to unbind account.",
    "accountLogin": "Login",
    "accountBroker": "Broker",
    "accountServer": "Server",
    "accountType": "Type",
    "accountStatus": "Status",
    "boundAt": "Bound At",
    "unbindConfirm": "Unbind this account? Active schedules on it will be stopped.",
    "unbind": "Unbind",
    "boundAccountsCount": "Bound Accounts",
    "noBoundAccounts": "No bound accounts yet. Schedule a strategy to auto-bind an account.",
    "aiTokensRemaining": "AI Tokens Remaining",
    "boundAccountsTitle": "Bound MT Accounts"
  },
  "agent": {
    "analysis": {
      "title": "Phân tích Backtest",
      "sharpe": "Sharpe",
      "drawdown": "DD",
      "winrate": "Tỷ lệ thắng",
      "consistency": "Tính nhất quán",
      "risk_adj": "Lợi nhuận điều chỉnh rủi ro",
      "overfitting": "Rủi ro quá khớp",
      "observations": "Quan sát chính",
      "suggestions": "Đề xuất cải thiện",
      "detailed": "Phân tích chi tiết"
    },
    "semantic_diff": {
      "title": "Thay đổi Chiến lược",
      "effect": "Tác động"
    },
    "profile": {
      "title": "Hồ sơ Chiến lược",
      "timeframe": "Khung thời gian",
      "regime": "Chế độ thị trường",
      "indicators": "Chỉ báo",
      "entry": "Điểm vào",
      "exit": "Điểm thoát",
      "risk": "Quản lý rủi ro",
      "coverage": "Mức bao phủ",
      "strengths": "Điểm mạnh",
      "weaknesses": "Điểm yếu",
      "blind_spots": "Điểm mù"
    }
  },
  "importAnalysis": {
    "execution": {
      "onBar": "Thực thi khi nến đóng",
      "onTick": "Thực thi theo tick",
      "onInitGrid": "Khởi tạo lưới"
    },
    "sizing": {
      "fixed": "Khối lượng cố định",
      "martingale": "Martingale",
      "percentBalance": "% số dư"
    },
    "analyzing": "Đang phân tích cấu trúc chiến lược...",
    "tradeLogicComplete": "Logic giao dịch đã được nhận diện đầy đủ",
    "guiNoiseDesc": "Các điểm mù sau là các tính năng hiển thị biểu đồ/nút được bỏ qua khi chạy phía máy chủ và không ảnh hưởng đến kết quả giao dịch. An toàn để nhập.",
    "cannotImport": "Không thể tự động nhập",
    "incompleteCoverage": "Phạm vi logic giao dịch chưa đầy đủ",
    "goodCoverage": "Phạm vi nhập khẩu tốt",
    "goodCoverageDesc": "Logic chính của chiến lược đã được nhận diện. An toàn để nhập. Kiểm tra danh sách tham số trước khi sử dụng.",
    "coverageTitle": "Phạm vi Nhập",
    "location": "Vị trí",
    "handling": "Xử lý",
    "userActionRequired": "Cần thao tác",
    "noBlindSpots": "Không cần xác nhận logic",
    "noBlindSpotsDesc": "Tất cả logic chiến lược đã được nhận dạng tự động. An toàn để nhập.",
    "emptyAnalysisDesc": "No strategy logic was recognized. The source code may be incomplete or use a different language."
  },
  "dashboard": {
    "quickActions": {
      "aiStrategy": "Chiến lược AI"
    },
    "noAccountsDesc": "Bind your first MT4/MT5 account to start monitoring and trading."
  },
  "logs": {
    "triggerSource": {
      "manual": "Thủ công",
      "strategy": "Chiến lược",
      "recovery": "Phục hồi"
    },
    "result": {
      "pass": "PASS",
      "reject": "REJECT"
    }
  },
  "app": {
    "name": "AlphaForge"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "Tiếng Trung giản thể",
    "traditionalChinese": "Tiếng Trung phồn thể",
    "vietnamese": "Tiếng Việt"
  },
  "market": {
    "allSymbols": "Tất cả mã",
    "ask": "Giá bán",
    "bid": "Giá mua",
    "common": "Phổ biến",
    "emptyWatchlist": "Danh sách trống",
    "loadingSymbols": "Đang tải...",
    "mid": "Giá trung bình",
    "mtSessionLost": "⚠ MT mất phiên — đang kết nối lại…",
    "noSymbolSelected": "Chọn một mã để xem dữ liệu thị trường",
    "noSymbolsFound": "Không tìm thấy mã nào",
    "popularSymbols": "Mã phổ biến",
    "searchPlaceholder": "Tìm kiếm mã (VD: EURUSD, XAUUSD)",
    "searchSymbol": "搜索品种...",
    "selectAccount": "Chọn tài khoản giao dịch",
    "selectSymbol": "Chọn mã giao dịch",
    "spread": "Chênh lệch",
    "watchlist": "Danh sách theo dõi"
  },
  "menu": {
    "accounts": "Tài Khoản",
    "aiAssistant": "Trợ lý AI",
    "algoDashboard": "Thuật toán",
    "analytics": "Phân tích",
    "assetAnalysis": "Phân tích AI",
    "assets": "Tài sản",
    "autoTrading": "Giao Dịch Tự Động",
    "dashboard": "Bảng điều khiển",
    "devGroup": "Phát triển",
    "experiments": "Thí nghiệm",
    "indicatorCatalog": "Danh mục chỉ báo",
    "logs": "Nhật ký hệ thống",
    "market": "Chợ",
    "marketRegime": "Chế độ thị trường",
    "marketTools": "Công cụ thị trường",
    "marketplace": "Thị trường",
    "opsGroup": "Vận hành",
    "schedules": "Lịch chạy chiến lược",
    "strategies": "Quản lý chiến lược",
    "strategy": "Chiến Lược",
    "strategyLibrary": "Thư viện chiến lược",
    "strategyLive": "Giám Sát Live",
    "strategyWorkspace": "Không gian chiến lược",
    "subscription": "Đăng Ký",
    "trading": "Giao Dịch",
    "wallet": "Ví"
  },
  "profile": {
    "lastLogin": "Đăng Nhập Cuối",
    "nickname": "Biệt Danh",
    "registered": "已注册",
    "role": "Vai Trò",
    "status": "Trạng Thái",
    "title": "Hồ sơ"
  },
  "share": {
    "actions": "Thao Tác",
    "createNew": "Tạo Liên Kết Chia Sẻ Mới",
    "createdAt": "Đã tạo",
    "deleteConfirm": "删除此分享链接？",
    "empty": "Chưa có liên kết chia sẻ",
    "expires": "Hết Hạn",
    "positions": "持仓",
    "showPositions": "显示持仓",
    "title": "Quản Lý Chia Sẻ",
    "token": "Liên Kết Chia Sẻ",
    "userId": "Người Dùng",
    "views": "Lượt Xem"
  },
  "sharePage": {
    "avgHolding": "Thời Gian Giữ TB",
    "avgLoss": "Lỗ TB",
    "avgWin": "Lãi TB",
    "bestTrade": "Lệnh Tốt Nhất",
    "bySymbol": "Hiệu Suất Theo Mã",
    "closeTime": "Đóng",
    "count": "Số Lệnh",
    "disclaimer": "Hiệu suất trong quá khứ không đảm bảo kết quả tương lai.",
    "equityCurve": "Đường Cong Vốn",
    "expired": "Liên kết chia sẻ này đã hết hạn",
    "footer": "Được tạo bởi AlphaForge",
    "language": "Ngôn ngữ",
    "loadFailed": "Không tải được dữ liệu chia sẻ",
    "losingTrades": "Lệnh Thua",
    "maxDrawdown": "Sụt Giảm Tối Đa",
    "netProfit": "Lợi Nhuận Ròng",
    "noPositions": "暂无持仓",
    "noTrades": "Chưa có giao dịch",
    "notFound": "Không tìm thấy",
    "openPrice": "开仓价",
    "positions": "当前持仓",
    "positionsLocked": "创建者未开放持仓查看",
    "profit": "Lợi Nhuận",
    "profitFactor": "Hệ Số Lợi Nhuận",
    "sharpeRatio": "Tỷ Số Sharpe",
    "side": "Lệnh",
    "subtitle": "Kết quả giao dịch thực tế",
    "symbol": "Mã",
    "title": "Hiệu Suất Giao Dịch",
    "totalReturn": "Lợi Nhuận Ròng",
    "totalTrades": "Tổng Số Lệnh",
    "totalVolume": "Tổng Khối Lượng",
    "tradeRecords": "Lịch Sử Giao Dịch",
    "volume": "Khối Lượng",
    "winRate": "Tỷ Lệ Thắng",
    "winningTrades": "Lệnh Thắng",
    "worstTrade": "Lệnh Tệ Nhất",
    "countUnit": "giao dịch"
  },
  "topbar": {
    "logout": "Đăng xuất",
    "profile": "Hồ sơ",
    "settings": "Cài đặt",
    "switchToAdmin": "Chuyển sang quản trị",
    "systemOk": "Hệ thống đang hoạt động bình thường",
    "user": "Người Dùng"
  },
  "theme": {
    "switchToDark": "Chuyển sang chế độ tối",
    "switchToLight": "Chuyển sang chế độ sáng"
  },
  "monitoring": {
    "unknown": "Không xác định",
    "healthy": "OK",
    "title": "Giám sát hệ thống",
    "sseConnected": "SSE đã kết nối",
    "disconnected": "Đã ngắt kết nối",
    "streamError": "Lỗi luồng",
    "waitingData": "Đang chờ dữ liệu...",
    "serviceHealth": "Tình trạng dịch vụ",
    "uptime": "Thời gian hoạt động",
    "database": "Cơ sở dữ liệu",
    "diskUsage": "Dung lượng đĩa",
    "goRuntime": "Go Runtime",
    "goroutines": "Goroutines",
    "gcCount": "Số lần GC",
    "gcPauseAvg": "Thời gian dừng GC TB",
    "stackUsage": "Sử dụng Stack",
    "heapMemory": "Bộ nhớ Heap",
    "dbPool": "Pool kết nối DB",
    "totalConns": "Tổng số",
    "idle": "Nhàn rỗi",
    "acquired": "Đang dùng",
    "mdGateway": "MD Gateway",
    "spillFiles": "Spill Files",
    "droppedBars": "Nến bị bỏ lỡ",
    "droppedSignals": "Tín hiệu bị bỏ lỡ",
    "consumerLag": "Độ trễ Consumer",
    "staleAccounts": "Tài khoản trễ",
    "deadAccounts": "Tài khoản chết",
    "avgGapSec": "Khoảng cách TB (s)",
    "maxGapSec": "Khoảng cách tối đa (s)",
    "dlq": "Hàng đợi thư chết (DLQ)",
    "parseErrors": "Lỗi phân tích",
    "bidGtAsk": "Giá mua > Giá bán",
    "nonPositive": "Không dương",
    "pushInterval": "Khoảng thời gian đẩy: 5s",
    "lastUpdate": "Cập nhật lần cuối"
  },
  "analytics": {
    "pnl": "P&L:"
  },
  "landing": {
    "brokersTitle": "Compatible with 30+ MT4/MT5 Brokers",
    "brokersDesc": "IC Markets, Pepperstone, XM, Exness, OANDA, FXTM, FBS, OctaFX, HotForex, Alpari, RoboForex and more. Connect your existing broker account in seconds.",
    "brokersLink": "View all supported brokers"
  }
} as const;
export default Base;
