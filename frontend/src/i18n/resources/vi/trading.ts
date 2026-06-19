// Auto-generated from proto/ant/v1/i18n/trading_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "algo": {
    "actions": {
      "cancel": "Cancel",
      "start": "Bắt Đầu"
    },
    "dashboard": {
      "activeExecutions": "Đang Thực Thi",
      "noActive": "No active algo executions",
      "title": "Bảng Thuật Toán"
    },
    "fields": {
      "account": "Tài Khoản",
      "algo": "Thuật Toán",
      "limitPrice": "限价",
      "participationRate": "Participation Rate",
      "side": "Hướng",
      "sliceInterval": "切片间隔",
      "symbol": "Mã",
      "timeRange": "时间范围",
      "urgency": "紧急度",
      "volume": "Khối Lượng"
    },
    "info": {
      "description": "Description",
      "name": "Tên"
    },
    "messages": {
      "started": "Algo started"
    },
    "side": {
      "buy": "Mua",
      "sell": "Bán"
    },
    "submitForm": {
      "title": "Launch Algo"
    },
    "table": {
      "actions": "Actions",
      "algo": "Thuật Toán",
      "executionId": "执行ID",
      "progress": "Tiến Độ",
      "side": "Hướng",
      "state": "Trạng Thái",
      "symbol": "Mã",
      "volume": "Khối Lượng"
    },
    "timePresets": {
      "EOD": "End of Day"
    }
  },
  "trading": {
    "account": "Tài Khoản",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "Xác nhận tắt",
        "disableInfoDescription": "Sau khi tắt, hệ thống sẽ dừng giao dịch tự động, nhưng các chiến lược đang bật vẫn có thể tiếp tục theo dõi thị trường.",
        "disableInfoTitle": "Tắt giao dịch tự động",
        "disableQuestion": "Are you sure you want to disable auto trading?",
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
    "messages": {
      "fetchOrderHistoryFailed": "Failed to load order history",
      "fetchPendingOrdersFailed": "Không thể tải lệnh chờ",
      "fetchPositionsFailed": "Không thể tải vị thế",
      "orderCloseFailed": "Đóng lệnh thất bại",
      "orderCloseSuccess": "Đóng lệnh thành công",
      "orderModifyFailed": "Sửa lệnh thất bại",
      "orderModifySuccess": "Sửa lệnh thành công",
      "orderSendFailed": "Đặt lệnh thất bại",
      "orderSendSuccess": "Đặt lệnh thành công"
    },
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
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "Check account status and permissions, then try again.",
          "title": "Giao dịch đã bị tắt cho tài khoản này."
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "Retry later; contact support if the issue persists.",
          "title": "Quy tắc rủi ro tạm thời chưa khả dụng."
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "Reduce volume, close positions, or add funds.",
          "title": "Không đủ ký quỹ khả dụng để đặt lệnh này."
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "Wait for the next trading session and retry.",
          "title": "Thị trường của mã này đang đóng cửa."
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "Close existing positions or raise the limit.",
          "title": "Đã đạt giới hạn số vị thế mở tối đa."
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "Cancel existing pending orders or raise the limit.",
          "title": "Đã đạt giới hạn số lệnh chờ tối đa."
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "Wait until price moves away from freeze distance, then retry.",
          "title": "Không thể sửa lệnh trong vùng đóng băng."
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "Choose a supported order type and retry.",
          "title": "Loại lệnh này không được hỗ trợ cho mã đã chọn."
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "Increase SL/TP distance and retry.",
          "title": "Stop-loss hoặc take-profit quá gần giá thị trường."
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "Switch to a tradable symbol or try later.",
          "title": "Mã này hiện không thể giao dịch."
        },
        "RISK_VOLUME_INVALID": {
          "action": "Adjust volume to match min/max/step requirements.",
          "title": "Khối lượng lệnh không hợp lệ."
        },
        "unknown": {
          "action": "Please review order parameters and try again.",
          "title": "Yêu cầu giao dịch đã bị từ chối."
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "Lưu",
        "description": "Vui lòng xác nhận cấu hình rủi ro sau:",
        "info": "After saving, all auto trading will follow the new risk limits.",
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
    "selectSymbol": "Select a symbol",
    "sell": "Bán",
    "side": "Hướng",
    "stop": "Dừng",
    "stopLoss": "Cắt lỗ",
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
    "symbol": "Mã",
    "takeProfit": "Chốt lời",
    "time": "Thời gian",
    "title": "Giao dịch",
    "type": "Loại",
    "volume": "Khối Lượng"
  }
} as const;
export default Trading;
