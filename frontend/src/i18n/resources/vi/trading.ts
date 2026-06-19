// Auto-generated from proto/ant/v1/i18n/trading_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "trading": {
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "Kiểm tra trạng thái và quyền của tài khoản, rồi thử lại.",
          "title": "Giao dịch đã bị tắt cho tài khoản này."
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "Thử lại sau; liên hệ hỗ trợ nếu vấn đề vẫn tiếp diễn.",
          "title": "Quy tắc rủi ro tạm thời chưa khả dụng."
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "Giảm khối lượng, đóng vị thế hoặc nạp thêm tiền.",
          "title": "Không đủ ký quỹ khả dụng để đặt lệnh này."
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "Chờ phiên giao dịch tiếp theo và thử lại.",
          "title": "Thị trường của mã này đang đóng cửa."
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "Đóng vị thế hiện có hoặc tăng giới hạn.",
          "title": "Đã đạt giới hạn số vị thế mở tối đa."
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "Hủy lệnh chờ hiện có hoặc tăng giới hạn.",
          "title": "Đã đạt giới hạn số lệnh chờ tối đa."
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "Chờ giá di chuyển ra khỏi vùng đóng băng, rồi thử lại.",
          "title": "Không thể sửa lệnh trong vùng đóng băng."
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "Chọn loại lệnh được hỗ trợ và thử lại.",
          "title": "Loại lệnh này không được hỗ trợ cho mã đã chọn."
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "Tăng khoảng cách SL/TP và thử lại.",
          "title": "Stop-loss hoặc take-profit quá gần giá thị trường."
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "Chuyển sang mã có thể giao dịch hoặc thử lại sau.",
          "title": "Mã này hiện không thể giao dịch."
        },
        "RISK_VOLUME_INVALID": {
          "action": "Điều chỉnh khối lượng để phù hợp với yêu cầu tối thiểu/tối đa/bước.",
          "title": "Khối lượng lệnh không hợp lệ."
        },
        "unknown": {
          "action": "Vui lòng kiểm tra tham số lệnh và thử lại.",
          "title": "Yêu cầu giao dịch đã bị từ chối."
        }
      }
    },
    "autoTrade": {
      "confirm": {
        "disableConfirm": "Xác nhận tắt",
        "disableInfoDescription": "Sau khi tắt, hệ thống sẽ dừng giao dịch tự động, nhưng các chiến lược đang bật vẫn có thể tiếp tục theo dõi thị trường.",
        "disableInfoTitle": "Tắt giao dịch tự động",
        "disableQuestion": "Bạn có chắc muốn tắt giao dịch tự động?",
        "disableTitle": "Tắt giao dịch tự động",
        "enableBullet1": "Hệ thống sẽ tự động thực hiện giao dịch khi điều kiện chiến lược thỏa mãn",
        "enableBullet2": "Hãy đảm bảo cấu hình rủi ro đã được thiết lập đúng",
        "enableBullet3": "Nên thử nghiệm trước trên tài khoản demo",
        "enableConfirm": "Xác nhận bật",
        "enableQuestion": "Bật tính năng giao dịch tự động?",
        "enableRiskDescription": "Khi bật giao dịch tự động, hệ thống sẽ tự động thực hiện giao dịch theo chiến lược. Vui lòng chắc chắn bạn hiểu rõ các rủi ro.",
        "enableRiskTitle": "Cảnh báo rủi ro",
        "enableTitle": "Bật giao dịch tự động"
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "Lưu",
        "description": "Vui lòng xác nhận cấu hình rủi ro sau:",
        "info": "Sau khi lưu, tất cả giao dịch tự động sẽ tuân theo giới hạn rủi ro mới.",
        "title": "Xác nhận lưu cấu hình rủi ro"
      },
      "fields": {
        "maxDailyLoss": "Lỗ tối đa mỗi ngày",
        "maxDrawdownPercent": "Giới hạn drawdown",
        "maxLotSize": "Khối lượng tối đa",
        "maxPositions": "Số vị thế tối đa",
        "maxRiskPercent": "Rủi ro tối đa mỗi lệnh",
        "trailingStopEnabled": "Trailing stop",
        "trailingStopPips": "Trailing Stop (pips)"
      }
    },
    "strategyExecute": {
      "confirm": {
        "action": "Hướng",
        "buy": "Mua",
        "confirmText": "Thực thi",
        "sell": "Bán",
        "strategyName": "Chiến lược",
        "symbol": "Mã",
        "title": "Xác nhận thực thi giao dịch",
        "volume": "Khối Lượng",
        "warningDescription": "Thao tác này sẽ thực hiện giao dịch thật ngay lập tức. Vui lòng kiểm tra kỹ tham số.",
        "warningTitle": "Xác nhận thực thi giao dịch"
      }
    },
    "messages": {
      "fetchOrderHistoryFailed": "Tải lịch sử lệnh thất bại",
      "fetchPendingOrdersFailed": "Không thể tải lệnh chờ",
      "fetchPositionsFailed": "Không thể tải vị thế",
      "orderCloseFailed": "Đóng lệnh thất bại",
      "orderCloseSuccess": "Đóng lệnh thành công",
      "orderModifyFailed": "Sửa lệnh thất bại",
      "orderModifySuccess": "Sửa lệnh thành công",
      "orderSendFailed": "Đặt lệnh thất bại",
      "orderSendSuccess": "Đặt lệnh thành công"
    },
    "account": "Tài Khoản",
    "balance": "Số dư",
    "buy": "Mua",
    "closePosition": "Đóng",
    "closePositionConfirm": "Đóng vị thế này?",
    "closePositionTitle": "Đóng vị thế",
    "equity": "Vốn chủ sở hữu",
    "freeMargin": "Ký quỹ tự do",
    "limit": "Giới hạn",
    "margin": "Ký quỹ",
    "marginLevel": "Mức ký quỹ",
    "markPrice": "Giá thị trường",
    "market": "Thị trường",
    "noAccount": "Chưa chọn tài khoản",
    "noOrders": "Chưa có lệnh nào",
    "noPositions": "Không có vị thế mở",
    "openPositionsTitle": "Vị thế mở",
    "openTime": "Thời gian mở",
    "orderHistory": "Lịch sử lệnh",
    "ordersCount": "{{count}} lệnh",
    "placeOrder": "Đặt lệnh",
    "pnl": "Lãi/Lỗ",
    "positionEntryPrice": "Giá vào lệnh",
    "positionLeverage": "Đòn bẩy",
    "positionLong": "LONG",
    "positionMarkPrice": "Giá thị trường",
    "positionShort": "SHORT",
    "positionSide": "Hướng",
    "positionSize": "Khối lượng",
    "positionUnrealizedPnL": "Lãi/Lỗ thả nổi",
    "positions": "Vị thế",
    "price": "Giá",
    "profit": "Lợi nhuận",
    "recentTrades": "Giao dịch gần đây",
    "selectSymbol": "Chọn mã",
    "sell": "Bán",
    "side": "Hướng",
    "stop": "Dừng",
    "stopLoss": "Cắt lỗ",
    "symbol": "Mã",
    "takeProfit": "Chốt lời",
    "time": "Thời gian",
    "title": "Giao dịch",
    "type": "Loại",
    "volume": "Khối Lượng"
  },
  "algo": {
    "actions": {
      "cancel": "Hủy",
      "start": "Bắt Đầu"
    },
    "dashboard": {
      "activeExecutions": "Đang Thực Thi",
      "noActive": "Không có lệnh nào đang chạy",
      "title": "Bảng Thuật Toán"
    },
    "fields": {
      "account": "Tài Khoản",
      "algo": "Thuật Toán",
      "limitPrice": "Giá Giới Hạn",
      "participationRate": "Tỷ Lệ Tham Gia",
      "side": "Hướng",
      "sliceInterval": "Khoảng Cắt Lệnh",
      "symbol": "Mã",
      "timeRange": "Khoảng Thời Gian",
      "urgency": "Độ Khẩn Cấp",
      "volume": "Khối Lượng"
    },
    "info": {
      "description": "Mô tả",
      "name": "Tên"
    },
    "messages": {
      "started": "Đã khởi động thuật toán"
    },
    "side": {
      "buy": "Mua",
      "sell": "Bán"
    },
    "submitForm": {
      "title": "Khởi Động Thuật Toán"
    },
    "table": {
      "actions": "Thao Tác",
      "algo": "Thuật Toán",
      "executionId": "Mã Thực Thi",
      "progress": "Tiến Độ",
      "side": "Hướng",
      "state": "Trạng Thái",
      "symbol": "Mã",
      "volume": "Khối Lượng"
    },
    "timePresets": {
      "EOD": "Cuối Ngày"
    },
    "twap": {
      "name": "TWAP (Giá Bình Quân Thời Gian)",
      "description": "Giá Trung Bình Theo Thời Gian — chia lệnh lớn thành các phần nhỏ phân bổ đều theo thời gian."
    },
    "vwap": {
      "name": "VWAP (Giá Bình Quân Khối Lượng)",
      "description": "Giá Trung Bình Theo Khối Lượng — thực hiện lệnh theo tỷ lệ phân phối khối lượng lịch sử."
    },
    "pov": {
      "name": "POV (Tỷ Lệ Tham Gia)",
      "description": "Phần Trăm Khối Lượng — tham gia với tỷ lệ cố định trên khối lượng thị trường."
    },
    "shortfall": {
      "name": "Shortfall (Giảm Khoảng Cách)",
      "description": "Giảm Thiểu Khoảng Cách — giảm thiểu chênh lệch giữa giá quyết định và giá thực thi."
    },
    "label": {
      "twap": "TWAP (Giá Bình Quân Thời Gian)",
      "vwap": "VWAP (Giá Bình Quân Khối Lượng)",
      "pov": "POV (Tỷ Lệ Tham Gia)",
      "shortfall": "Shortfall (Giảm Khoảng Cách)"
    }
  }
} as const;
export default Trading;
