// Auto-generated from proto/ant/v1/i18n/accounts_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Accounts = {
  "accounts": {
    "analytics": {
      "monthlyAnalysis": {
        "bonus": {
          "chartHoldingTitle": "{{month}} 平均持仓时间",
          "chartPopularTitle": "{{month}} 货币热度",
          "chartRiskTitle": "Bonus: Ty so loi nhuan (profit factor) theo symbol — {{month}}.",
          "emptyCharts": "Khong co lenh trong thang",
          "legendBulls": "Mua",
          "legendShortTerm": "Ban",
          "popularityShare": "手数份额",
          "sliceOther": "Khac"
        },
        "metrics": {
          "change": "Thay doi",
          "lots": "Lot",
          "pips": "点",
          "profit": "Lợi nhuận"
        },
        "chartMainTitle": "Loi nhuan theo thang ({{metric}})",
        "focusedValue": "{{period}} · {{metric}}：{{value}}",
        "title": "Phan tich theo thang"
      },
      "monthlyDetail": {
        "fields": {
          "averageHours": "TB",
          "bestTrade": "Lệnh tốt nhất",
          "maxHours": "Tối đa",
          "medianHours": "Trung vị",
          "minHours": "Tối thiểu",
          "netReturn": "Lợi nhuận ròng",
          "profitFactor": "Hệ số lợi nhuận",
          "totalTrades": "Tổng lệnh",
          "winRate": "Tỷ lệ thắng",
          "worstTrade": "Lệnh tệ nhất"
        },
        "holdingTitle": "Thời gian giữ lệnh",
        "long": "Mua",
        "metricsTitle": "Chỉ số tháng",
        "popularityTitle": "Mức độ phổ biến",
        "riskRewardTitle": "Tỷ lệ R:R",
        "short": "Bán",
        "symbolPnLTitle": "Lợi nhuận theo cặp"
      },
      "advancedTabs": {
        "daily": "日",
        "hourly": "Theo giờ"
      },
      "chartPeriod": {
        "all": "全部",
        "day": "Hôm nay",
        "month": "Tháng này",
        "week": "Tuần này",
        "year": "Năm nay"
      },
      "chartSeries": {
        "balance": "Số dư",
        "equity": "Vốn",
        "profit": "Lợi nhuận",
        "tradeCount": "Giao dịch"
      },
      "chartType": {
        "balance": "Số dư",
        "equity": "Vốn",
        "profit": "Lợi nhuận"
      },
      "empty": {
        "dailyPnL": "Không có dữ liệu lãi/lỗ theo ngày",
        "equityCurve": "Không có dữ liệu đường vốn",
        "hourly": "暂无时段分析数据",
        "monthlyProfit": "Không có dữ liệu lãi/lỗ theo tháng",
        "symbolDistribution": "Không có dữ liệu phân bổ mã"
      },
      "stats": {
        "avgDailyReturn": "Lợi nhuận ngày TB",
        "avgHolding": "Thời gian giữ TB",
        "avgLoss": "Lỗ trung bình",
        "avgProfit": "Lãi trung bình",
        "calmar": "Calmar",
        "consecutiveWinsLosses": "Chuỗi thắng/thua",
        "largestLoss": "Lỗ lớn nhất",
        "largestWin": "Lãi lớn nhất",
        "maxDrawdown": "Drawdown tối đa",
        "netDeposit": "净入金",
        "netProfit": "Lợi nhuận ròng",
        "profitFactor": "Hệ số lợi nhuận",
        "sharpe": "Sharpe",
        "sortino": "Sortino",
        "totalDeposit": "Nạp",
        "totalTrades": "Tổng số lệnh",
        "totalWithdrawal": "Rút",
        "volatility": "Biến động",
        "winRate": "Tỷ lệ thắng"
      },
      "timeDetail": {
        "balance": "Số dư",
        "lots": "Lot",
        "maxFloatingLossAmount": "Lỗ thả nổi tối đa",
        "maxFloatingLossRatio": "Tỷ lệ lỗ thả nổi tối đa",
        "maxFloatingProfitAmount": "Lợi nhuận thả nổi tối đa",
        "maxFloatingProfitRatio": "最大浮动盈利比",
        "profitAmount": "Lợi nhuận",
        "profitFactor": "Hệ số lợi nhuận",
        "trades": "Giao dịch"
      },
      "advancedStatsTitle": "Thống kê nâng cao",
      "dailyPnLTitle": "📅 Lãi/lỗ theo ngày",
      "hourlyTitle": "⏰ Phân tích theo giờ",
      "monthlyProfitTitle": "Lãi/lỗ theo tháng",
      "symbolDistributionTitle": "Phân bổ mã"
    },
    "bind": {
      "actions": {
        "confirmBind": "Xác nhận",
        "retryVerify": "重试",
        "search": "Tìm kiếm",
        "verifyAccount": "Xác minh tài khoản"
      },
      "errorModal": {
        "title": "绑定失败"
      },
      "errors": {
        "brokerUnavailable": "Lỗi máy chủ hoặc mật khẩu không đúng",
        "connectionFailed": "Không thể kết nối đến máy chủ môi giới, vui lòng kiểm tra mạng",
        "invalidCredentials": "Không tìm thấy tài khoản hoặc mật khẩu không đúng",
        "timeout": "连接超时，请稍后重试"
      },
      "fields": {
        "brokerName": "Tên môi giới",
        "company": "Công ty",
        "password": "Mật khẩu",
        "platform": "Nền tảng",
        "server": "Máy chủ",
        "tradingAccount": "Tài khoản giao dịch"
      },
      "labels": {
        "serverCount": "{{count}} 台服务器"
      },
      "messages": {
        "bindFailed": "Liên kết tài khoản thất bại",
        "bindSuccess": "Liên kết tài khoản thành công",
        "enterBrokerName": "Vui lòng nhập tên môi giới",
        "enterPassword": "Vui lòng nhập mật khẩu",
        "enterTradingAccount": "Vui lòng nhập tài khoản giao dịch",
        "foundBrokers": "Tìm thấy {{count}} nhà môi giới",
        "loginDigitsOnly": "交易账户只能包含数字",
        "noAccessHosts": "Không có máy chủ khả dụng",
        "noBrokersFound": "Không tìm thấy nhà môi giới phù hợp. Vui lòng kiểm tra tên.",
        "searchFailed": "Tìm kiếm thất bại. Vui lòng thử lại sau.",
        "selectServer": "Vui lòng chọn máy chủ",
        "verifyFailed": "Xác minh tài khoản thất bại"
      },
      "placeholders": {
        "brokerName": "Nhập tên môi giới, ví dụ: XM, IC Markets",
        "company": "Chọn công ty môi giới",
        "password": "输入密码",
        "server": "Chọn máy chủ",
        "tradingAccount": "Nhập tài khoản giao dịch"
      },
      "step1": {
        "subtitle": "选择您的交易平台并搜索经纪商",
        "title": "Chọn nền tảng và nhà môi giới"
      },
      "step2": {
        "subtitle": "输入您的交易账户和密码",
        "title": "Nhập thông tin tài khoản"
      },
      "step3": {
        "subtitle": "验证凭据并确认完成",
        "title": "Xác nhận liên kết"
      },
      "summary": {
        "balance": "Số dư",
        "broker": "Nhà môi giới",
        "currency": "货币",
        "equity": "Vốn",
        "freeMargin": "Ký quỹ khả dụng",
        "leverage": "Đòn bẩy",
        "margin": "Ký quỹ",
        "password": "Mật khẩu",
        "platform": "Nền tảng",
        "server": "Máy chủ",
        "tradingAccount": "Tài khoản giao dịch",
        "verified": "Tài khoản đã xác minh"
      },
      "passwordHint": "Mật khẩu được truyền qua HTTPS. Backend lưu dưới dạng băm Argon2id không thể đảo ngược.",
      "title": "Liên kết tài khoản MT"
    },
    "card": {
      "actions": {
        "details": "详情",
        "orders": "Lệnh",
        "positions": "Vị thế"
      },
      "deleteConfirm": {
        "content": "此操作不可撤销",
        "title": "Xóa tài khoản này?"
      },
      "fields": {
        "balance": "Số dư",
        "broker": "Nhà môi giới",
        "equity": "Vốn",
        "server": "Máy chủ"
      },
      "status": {
        "connected": "Đã kết nối",
        "connecting": "Đang kết nối",
        "disabled": "Đã tắt",
        "disconnected": "Mất kết nối",
        "error": "错误"
      }
    },
    "detail": {
      "accountType": {
        "demo": "模拟",
        "real": "Thực"
      },
      "actions": {
        "deleteAccount": "Xóa tài khoản",
        "deleteConfirm": "Xác minh & Xóa",
        "deletePasswordHint": "Nhập mật khẩu giao dịch MT hoặc mật khẩu chỉ-đọc để xác minh:",
        "deletePasswordPlaceholder": "Mật khẩu giao dịch / chỉ-đọc MT",
        "deleteWarning": "Hành động này không thể hoàn tác. Toàn bộ dữ liệu tài khoản (lịch sử giao dịch, phân tích, v.v.) sẽ bị xóa vĩnh viễn.",
        "disableAccount": "Tắt tài khoản",
        "enableAccount": "Bật tài khoản",
        "syncHistory": "同步历史"
      },
      "balanceRecord": {
        "deposit": "💰 Nạp tiền",
        "depositIconText": "💰 Nạp tiền",
        "withdraw": "💸 Rút tiền",
        "withdrawIconText": "💸 Rút tiền"
      },
      "cards": {
        "balance": "Số dư",
        "credit": "授信",
        "equity": "Vốn",
        "floatingProfit": "Lãi/lỗ thả nổi",
        "marginFree": "Ký quỹ khả dụng",
        "marginLevel": "Tỷ lệ ký quỹ",
        "marginUsed": "Ký quỹ đã dùng"
      },
      "messages": {
        "fetchAccountFailed": "Không thể tải thông tin tài khoản. Vui lòng thử lại sau.",
        "syncHistoryFailed": "同步订单历史失败，请确保账户已连接到 MT 服务器。",
        "syncHistorySuccess": "Đồng bộ lịch sử lệnh thành công"
      },
      "mode": {
        "investor": "Chế độ nhà đầu tư",
        "trader": "交易员模式"
      },
      "orderTypes": {
        "buyLimit": "Mua giới hạn",
        "buyStop": "Mua dừng",
        "sellLimit": "Bán giới hạn",
        "sellStop": "卖出止损"
      },
      "status": {
        "connected": "Đã kết nối",
        "connecting": "Đang kết nối",
        "disabled": "Đã tắt",
        "disconnected": "Mất kết nối",
        "error": "错误"
      },
      "syncHistory": {
        "content": "Đồng bộ lịch sử lệnh 1 năm gần nhất từ máy chủ MT? Việc này có thể mất một chút thời gian.",
        "ok": "同步",
        "title": "Đồng bộ lịch sử lệnh"
      },
      "connected": "Đã kết nối",
      "lastConnected": "{{time}}",
      "leverage": "Đòn bẩy {{leverage}}x"
    },
    "disabled": {
      "confirmDelete": {
        "content": "此操作不可撤销",
        "title": "Xóa tài khoản này?"
      },
      "mobile": {
        "balanceLabel": "Số dư: ",
        "equityLabel": "净值: "
      },
      "table": {
        "account": "Tài khoản",
        "actions": "操作",
        "balance": "Số dư",
        "broker": "Nhà môi giới",
        "equity": "Vốn",
        "type": "Loại"
      },
      "title": "Tài khoản đã tắt"
    },
    "edit": {
      "fields": {
        "oldPassword": "当前密码",
        "password": "Mật khẩu mới",
        "server": "Máy chủ",
        "tradingAccount": "Tài khoản giao dịch"
      },
      "messages": {
        "enterOldPassword": "Vui lòng nhập mật khẩu hiện tại",
        "enterPassword": "Vui lòng nhập mật khẩu mới",
        "passwordSaved": "密码已保存",
        "passwordVerifyFailed": "Thay đổi mật khẩu thất bại"
      },
      "placeholders": {
        "newPassword": "Nhập mật khẩu mới",
        "oldPassword": "输入当前密码"
      },
      "title": "Chỉnh sửa tài khoản"
    },
    "report": {
      "periods": {
        "month": "Tháng này",
        "quarter": "Quý này",
        "week": "Tuần này",
        "year": "Năm nay"
      },
      "sections": {
        "findings": "Phát hiện chính",
        "recommendations": "Khuyến nghị",
        "summary": "Tổng quan"
      },
      "aiAnalysis": "Phân tích AI",
      "direction": "Phân tích hướng",
      "directionLong": "Mua",
      "directionShort": "Bán",
      "drawdownEvents": "Sự kiện drawdown",
      "drawdownOverlay": "Đường vốn + Drawdown",
      "generate": "Tạo báo cáo",
      "goToAISettings": "Đi tới Cài đặt AI →",
      "recovered": "Đã phục hồi",
      "symbolPnL": "Lãi/lỗ theo mã",
      "title": "Báo cáo giao dịch",
      "titleShort": "Báo cáo",
      "tradeDistribution": "Phân phối lãi/lỗ",
      "winRateTrend": "Xu hướng tỷ lệ thắng"
    },
    "tradeTabs": {
      "pagination": {
        "total": "共 {{total}} 条"
      },
      "table": {
        "closePrice": "Giá đóng",
        "closeTime": "平仓时间",
        "currentPrice": "Giá hiện tại",
        "openPrice": "Giá mở",
        "openTime": "Thời gian mở",
        "orderId": "Mã lệnh",
        "pendingPrice": "Giá đặt",
        "pendingTime": "Thời gian đặt",
        "profit": "Lợi nhuận",
        "side": "Hướng",
        "symbol": "Mã",
        "type": "Loại",
        "volume": "Khối lượng"
      },
      "emptyHistory": "Chưa có lịch sử lệnh",
      "emptyPositions": "Chưa có vị thế",
      "historyWithCount": "Lịch sử ({{count}})",
      "pendingWithCount": "Lệnh chờ ({{count}})",
      "positionsWithCount": "Vị thế ({{count}})",
      "syncHistory": "Đồng bộ lịch sử"
    },
    "empty": {
      "subtitle": "点击下方按钮绑定您的 MT4/MT5 交易账户",
      "title": "Chưa có tài khoản"
    },
    "legend": {
      "connected": "Đã kết nối",
      "connecting": "Đang kết nối",
      "disabled": "Đã tắt",
      "disconnectedOrError": "Mất kết nối/Lỗi",
      "title": "Chú giải:"
    },
    "messages": {
      "connectFailed": "Kết nối thất bại",
      "connectSuccess": "Kết nối thành công",
      "connectingMtServer": "Đang kết nối đến máy chủ MT",
      "createFailed": "Tạo tài khoản thất bại",
      "createdSuccess": "Tạo tài khoản thành công",
      "deleteFailed": "Xóa thất bại",
      "deleted": "Đã xóa",
      "disableFailed": "Vô hiệu hóa tài khoản thất bại",
      "disabledSuccess": "Vô hiệu hóa tài khoản thành công",
      "disconnectFailed": "Ngắt kết nối thất bại",
      "enableFailed": "启用账户失败",
      "enabledSuccess": "Kích hoạt tài khoản thành công",
      "fetchAccountFailed": "Không thể tải thông tin tài khoản",
      "fetchListFailed": "Không thể tải danh sách tài khoản"
    },
    "bindNew": "Liên kết tài khoản mới",
    "subtitle": "Quản lý tài khoản MT4/MT5",
    "title": "Tài khoản"
  }
} as const;
export default Accounts;
