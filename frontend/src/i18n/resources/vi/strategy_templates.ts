// Auto-generated from proto/ant/v1/i18n/strategy_templates_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTemplates = {
  "strategy": {
    "templates": {
      "scheduleLaunch": {
        "form": {
          "scheduleTypes": {
            "hfQuote": "高频报价",
            "interval": "定时执行",
            "klineClose": "K-line Đóng"
          },
          "account": "Tài khoản",
          "accountPlaceholder": "选择账户",
          "defaultVolume": "默认手数",
          "defaultVolumeTip": "每个信号的默认下单量",
          "enableAfterCreate": "创建后立即启用",
          "hfCooldownMs": "高频冷却(毫秒)",
          "hfCooldownMsTip": "报价驱动执行间的冷却时间",
          "intervalMs": "间隔(毫秒)",
          "intervalMsTip": "非高频模式最小1000ms",
          "investorTag": "投资者(只读)",
          "maxDrawdownPct": "最大回撤%",
          "maxDrawdownPctTip": "回撤超过此阈值自动停止",
          "maxPositions": "最大持仓数",
          "maxPositionsTip": "同时持有的最大仓位数量",
          "riskSection": "Kiểm Soát Rủi Ro",
          "scheduleName": "计划名称",
          "scheduleNameMax": "最多64字符",
          "scheduleNamePlaceholder": "VD: EURUSD M5 chiến lược buổi sáng",
          "scheduleType": "计划类型",
          "stopLossOffset": "止损偏移",
          "stopLossOffsetTip": "距入场价的止损距离(点)",
          "strategyParamsSection": "策略参数",
          "symbol": "Mã",
          "symbolPlaceholder": "选择品种",
          "symbolPlaceholderEmpty": "未配置品种",
          "takeProfitOffset": "止盈偏移",
          "takeProfitOffsetTip": "距入场价的止盈距离(点)",
          "timeframe": "Khung thời gian"
        },
        "actions": {
          "addAccount": "添加账户",
          "create": "Tạo lịch",
          "createAndEnable": "Tạo & bật",
          "createScheduleNoEnable": "Tạo lịch chạy",
          "publishTemplate": "Xuất bản template",
          "updateTradingPassword": "更新交易密码"
        },
        "metrics": {
          "annualReturn": "Lợi nhuận năm",
          "maxDrawdown": "Sụt giảm tối đa",
          "sharpe": "Sharpe",
          "totalReturn": "Tổng lợi nhuận",
          "totalTrades": "Số lệnh",
          "winRate": "Tỷ lệ thắng"
        },
        "backtestRunningHint": "Backtest đang chạy. Vui lòng đợi.",
        "errorInvestorAccount": "无法使用投资者账户启动计划。请更新交易密码以启用交易。",
        "investorWarningBody": "此账户为投资者(只读)模式，需要交易权限才能启动计划。",
        "investorWarningTitle": "投资者账户",
        "keyMetrics": "Chỉ số chính",
        "launchSection": "Khởi chạy lịch",
        "newPasswordPlaceholder": "Nhập mật khẩu giao dịch mới",
        "noAccountBody": "启动计划前需要先绑定MT账户。",
        "noAccountTitle": "无账户",
        "noRun": "Chưa có lần chạy backtest",
        "score": "Điểm",
        "title": "Khởi chạy lịch",
        "tradePermissionOk": "交易权限验证通过",
        "updatePasswordFailed": "更新交易密码失败",
        "updatePasswordHint": "输入此账户的交易密码以启用交易。",
        "updatePasswordOk": "交易密码已更新",
        "updatePasswordStillInvestor": "密码更新成功但账户仍为投资者模式，请联系客服。",
        "updatePasswordTitle": "更新交易密码",
        "verifyingPermission": "验证交易权限中..."
      },
      "backtest": {
        "fields": {
          "account": "Tài khoản",
          "extraSymbols": "Mã Bổ Sung (đa chọn)",
          "initialCapital": "Vốn Ban Đầu",
          "range": "Phạm vi",
          "symbol": "Mã",
          "timeframe": "Khung thời gian",
          "title": "Tiêu Đề"
        },
        "parameters": {
          "title": "策略参数"
        },
        "placeholders": {
          "account": "Chọn tài khoản",
          "extraSymbols": "Tùy chọn, hữu ích cho chiến lược cặp/xoay vòng",
          "range": "Chọn Khoảng",
          "symbol": "Chọn mã"
        },
        "quickRange": {
          "custom": "Tùy Chỉnh"
        },
        "tooltips": {
          "extraSymbols": "Mã bổ sung để lấy K-line (cùng tài khoản, cùng khung thời gian). Chiến lược truy cập qua context[\"closes_by_symbol\"]."
        },
        "validation": {
          "accountRequired": "Vui lòng chọn tài khoản",
          "initialCapitalRequired": "Vốn ban đầu là bắt buộc",
          "rangeRequired": "Khoảng thời gian là bắt buộc",
          "symbolRequired": "Vui lòng chọn mã",
          "timeframeRequired": "Vui lòng chọn timeframe"
        },
        "accountDisabledSuffix": " (đã tắt)",
        "modalTitleWithName": "回测: {{name}}",
        "title": "Kiểm thử lùi"
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "Tạo lịch chạy",
          "launchSchedule": "Xem Điểm",
          "view": "Xem"
        },
        "status": {
          "canceled": "Đã Hủy",
          "canceling": "Đang Hủy",
          "completed": "Hoàn tất",
          "failed": "Thất Bại",
          "queued": "Đang Chờ",
          "running": "Đang chạy"
        },
        "table": {
          "actions": "Thao tác",
          "createdAt": "Thời gian tạo",
          "status": "Trạng thái",
          "symbol": "Mã",
          "timeframe": "Khung thời gian",
          "title": "Tiêu Đề"
        },
        "batchDelete": "Xóa {{count}}",
        "batchDeleteConfirm": "Xóa {{count}} báo cáo backtest?",
        "batchDeleteSuccess": "Đã xóa {{count}} báo cáo backtest",
        "deleteConfirm": "Xóa lần chạy này?",
        "empty": "Không có lần backtest nào",
        "title": "Lịch Sử Backtest"
      },
      "codeModal": {
        "actions": {
          "copy": "Sao chép"
        },
        "title": "Mã chiến lược"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "Xác Thực Code"
        },
        "fields": {
          "code": "Code chiến lược",
          "description": "Mô tả",
          "name": "Tên",
          "publicShare": "Công khai"
        },
        "placeholders": {
          "codeSample": "Dán mã chiến lược Python...",
          "description": "Tùy chọn: mô tả",
          "name": "VD: Chiến lược cắt MA"
        },
        "title": {
          "create": "Tạo Mẫu",
          "edit": "Chỉnh sửa mẫu"
        },
        "validation": {
          "codeRequired": "Code là bắt buộc",
          "nameRequired": "Vui lòng nhập tên"
        }
      },
      "actions": {
        "backtest": "Kiểm thử lùi",
        "copy": "Sao chép",
        "create": "Tạo mẫu",
        "createTemplate": "Tạo Mẫu",
        "delete": "Xóa",
        "edit": "Chỉnh sửa",
        "launchSchedule": "Khởi chạy lịch",
        "viewCode": "Xem code"
      },
      "badges": {
        "preset": "Mặc định"
      },
      "messages": {
        "backtestCancelFailed": "Hủy backtest thất bại",
        "backtestCancelRequested": "Đã yêu cầu hủy backtest",
        "backtestRangeInvalid": "Khoảng backtest không hợp lệ",
        "backtestReportDeleted": "Báo cáo backtest đã xóa",
        "backtestReportNotFound": "Không tìm thấy báo cáo backtest",
        "backtestRunNoPublishedTemplate": "Lần chạy backtest không có mẫu đã xuất bản",
        "backtestRunningCannotPublish": "Backtest đang chạy. Không thể xuất bản lúc này.",
        "backtestSubmitFailed": "Nộp backtest thất bại",
        "backtestSubmitted": "Backtest đã nộp",
        "cannotPublishAndCreateDraftFailed": "Không thể xuất bản. Tạo bản nháp thất bại.",
        "codeCopied": "Đã sao chép code",
        "codeValidationFailed": "Xác thực code thất bại",
        "codeValidationNotPassed": "Xác thực code không đạt",
        "codeValidationPassed": "Xác thực code thành công",
        "copyFailed": "Sao chép thất bại, vui lòng sao chép thủ công",
        "createScheduleFailed": "Tạo lịch thất bại",
        "deepLinkNavigate": "Đã mở mẫu và chi tiết chạy mới nhất từ liên kết ngoài",
        "enterStrategyCode": "Vui lòng nhập mã chiến lược",
        "fetchTemplateListFailed": "Tải danh sách mẫu thất bại",
        "missingDraftIdCannotPublish": "Thiếu ID bản nháp. Không thể xuất bản.",
        "missingScheduleInfo": "Thiếu thông tin lịch",
        "publishFailed": "Xuất bản thất bại",
        "publishedButNoTemplateId": "Đã xuất bản nhưng thiếu ID mẫu.",
        "readStrategyCodeFailed": "Đọc code chiến lược thất bại",
        "readTemplateStatusFailed": "Đọc trạng thái mẫu thất bại",
        "republishedButNoTemplateId": "Đã tái xuất bản nhưng thiếu ID mẫu.",
        "scheduleCreated": "Lịch đã tạo",
        "scheduleCreatedAndEnabled": "Lịch đã tạo và kích hoạt",
        "selectBacktestRange": "Vui lòng chọn khoảng backtest",
        "strategyCodeEmptyCannotBacktest": "Code chiến lược trống. Không thể backtest.",
        "strategyCodeEmptyCannotPublish": "Code chiến lược trống. Vui lòng lưu code trước khi xuất bản.",
        "systemTemplateReadOnly": "Mẫu hệ thống chỉ đọc. Hãy sao chép để chỉnh sửa.",
        "templateAlreadyPublished": "Mẫu đã được xuất bản",
        "templateCreated": "Mẫu đã tạo",
        "templateDeleted": "Mẫu đã xóa",
        "templateNotDraftUnknownPublishStatus": "Mẫu không phải bản nháp. Không rõ trạng thái xuất bản.",
        "templateNotPublishedCannotCreateSchedule": "Mẫu chưa được xuất bản. Không thể tạo lịch.",
        "templatePublished": "Mẫu đã xuất bản",
        "templateRepublished": "Mẫu đã tái xuất bản",
        "templateUpdated": "Mẫu đã cập nhật"
      },
      "status": {
        "draft": "Nháp",
        "published": "Đã xuất bản"
      },
      "table": {
        "actions": "Thao tác",
        "createdAt": "Thời gian tạo",
        "defaultHint": "Mặc định",
        "description": "Mô tả",
        "emptyUser": "Chưa có mẫu người dùng. Nhấn \"Tạo Mẫu\" ở trên để bắt đầu.",
        "loadingDefault": "Đang tải mẫu mặc định...",
        "name": "Tên",
        "status": "Trạng thái",
        "tags": "Nhãn",
        "updatedAt": "Cập nhật lúc",
        "useCount": "Số lần dùng",
        "visibility": "Hiển thị"
      },
      "tabs": {
        "system": "Mẫu hệ thống",
        "user": "Mẫu Người Dùng"
      },
      "visibility": {
        "private": "Riêng tư",
        "public": "Công khai"
      },
      "copySuffix": " (Bản sao)",
      "defaultDraftName": "Mẫu Nháp",
      "deleteConfirm": "Xóa mẫu này?",
      "scheduleName": "{{symbol}} {{timeframe}} {{name}}",
      "title": "Mẫu chiến lược"
    }
  }
} as const;
export default StrategyTemplates;
