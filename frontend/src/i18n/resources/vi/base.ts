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
      "activeUsers": "Người Dùng Hoạt Động",
      "loadFailed": "Tải dữ liệu bảng điều khiển thất bại",
      "mtAccounts": "TK MT",
      "onlineAccounts": "Trực Tuyến",
      "recentLogs": "Nhật Ký Gần Đây",
      "title": "Bảng Điều Khiển Quản Trị",
      "todayProfit": "Lợi Nhuận Hôm Nay",
      "todayTrades": "Giao Dịch Hôm Nay",
      "totalUsers": "Tổng Người Dùng"
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
        "status": "Trạng Thái"
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
        "userUpdatedSuccess": "Đã cập nhật người dùng"
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
      "value": "值"
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
    "header": {
      "admin": "管理",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "Đăng xuất"
    },
    "sidebar": {
      "accountManagement": "Quản Lý Tài Khoản",
      "dashboard": "Bảng điều khiển",
      "jurisdiction": "Kiểm Soát Quyền Hạn",
      "operationLogs": "操作日志",
      "shareManagement": "分享分析",
      "systemConfig": "Cấu Hình Hệ Thống",
      "tradingMonitor": "Giám Sát Giao Dịch",
      "userManagement": "Quản Lý Người Dùng",
      "walletManagement": "钱包管理"
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
    "wallet": {
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
      "walletFor": "Ví của"
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
      "todayExecutions": "Today's Executions",
      "todayProfit": "Today's Profit"
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
        "message": "{{symbol}} triggered {{signalType}}",
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
    "all": "Tất cả",
    "clearAll": "Xóa",
    "confirmClearAll": "Xóa tất cả thông báo?",
    "empty": "Không có thông báo",
    "markAllRead": "Đánh dấu đã đọc",
    "title": "Thông báo",
    "unread": "Chưa đọc"
  },
  "auth": {
    "fields": {
      "confirmPassword": "确认密码",
      "email": "用户邮箱",
      "password": "Mật Khẩu"
    },
    "forgotPassword": {
      "backToLogin": "返回登录",
      "hint": "Vui lòng liên hệ quản trị viên hoặc hỗ trợ để đặt lại mật khẩu.",
      "title": "Đặt lại Mật khẩu"
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
      "passwordRequired": "Vui lòng nhập mật khẩu"
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
    "ok": "OK",
    "operationFailed": "操作失败",
    "pageError": "Lỗi trang",
    "pageUnderDevelopment": "此页面开发中",
    "pleaseWait": "Vui lòng chờ...",
    "previous": "Quay lại",
    "refresh": "Làm mới",
    "remove": "Xóa",
    "required": "Bắt buộc",
    "retry": "Thử lại",
    "readOnly": "Chỉ đọc",
    "save": "Lưu",
    "saveFailed": "Lưu thất bại",
    "saveSuccess": "Đã Lưu",
    "saved": "Đã lưu",
    "searching": "Đang tìm...",
    "selectSymbolToViewChart": "Chọn mã để xem biểu đồ",
    "send": "Gửi",
    "showDetails": "Xem chi tiết",
    "totalItems": "Tổng {{count}} mục",
    "translate": "Dịch",
    "unexpectedError": "Đã xảy ra lỗi không mong muốn",
    "unknown": "Không Xác Định",
    "unsaved": "Chưa lưu",
    "updated": "Đã cập nhật",
    "viewOriginal": "Xem nguyên văn",
    "viewTranslation": "Xem bản dịch",
    "yes": "Có",
    "you": "Bạn"
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
      "request_failed": "Yêu cầu API thất bại"
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
  "marketplace": {
    "author": {
      "avgRating": "Đánh Giá TB",
      "empty": "Chưa có chiến lược nào được xuất bản. Vào Thư Viện Chiến Lược để xuất bản.",
      "published": "Đã Xuất Bản"
    },
    "card": {
      "by": "by",
      "free": "Miễn Phí",
      "owned": "Ngày Mua",
      "subscribers": "Người Đăng Ký",
      "winRate": "Tỷ Lệ Thắng"
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
      "yourRating": "Đánh Giá Của Bạn"
    },
    "messages": {
      "commentFailed": "Bình luận thất bại",
      "commentPosted": "Đã đăng bình luận",
      "loginFirst": "Vui lòng đăng nhập trước",
      "paymentComingSoon": "Thanh toán sắp ra mắt",
      "rateFailed": "Đánh giá thất bại",
      "rated": "Đã gửi đánh giá",
      "subscribeFailed": "Thất Bại",
      "subscribed": "Đã thêm vào mục đã mua"
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
      "walletBalance": "Số Dư Của Bạn"
    },
    "purchases": {
      "empty": "Chưa có giao dịch mua nào. Duyệt chợ để tìm chiến lược.",
      "status": "Trạng Thái",
      "strategy": "Chiến Lược"
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
    "empty": "Chưa có chiến lược nào được xuất bản",
    "filterByClass": "Lọc theo loại tài sản",
    "noSubscriptions": "Chưa có đăng ký nào",
    "publish": "Xuất Bản Chiến Lược",
    "searchPlaceholder": "Tìm kiếm chiến lược...",
    "subtitle": "Khám phá, mua và sử dụng chiến lược cộng đồng",
    "title": "Chợ Chiến Lược"
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
  "wallet": {
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
    "accountNumber": "Số TK",
    "balance": "Số Dư",
    "currency": "Tiền Tệ",
    "deposit": "Nạp Tiền",
    "frozen": "Đóng Băng",
    "frozenBalance": "Đóng Băng",
    "history": "Lịch Sử",
    "title": "Ví Của Tôi",
    "transactions": "Giao Dịch",
    "withdraw": "Rút Tiền"
  },
  "app": {
    "name": "AntTrader"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "简体中文",
    "traditionalChinese": "繁體中文",
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
    "strategyWorkspace": "Không gian chiến lược",
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
    "footer": "Được tạo bởi AntTrader",
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
    "worstTrade": "Lệnh Tệ Nhất"
  },
  "topbar": {
    "logout": "Đăng xuất",
    "profile": "Hồ sơ",
    "settings": "Cài đặt",
    "switchToAdmin": "Chuyển sang quản trị",
    "systemOk": "Hệ thống đang hoạt động bình thường",
    "user": "Người Dùng"
  }
} as const;
export default Base;
