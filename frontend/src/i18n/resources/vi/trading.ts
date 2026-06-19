// Auto-generated from proto/ant/v1/i18n/trading_vi.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "algo": {
    "actions": {
      "cancel": "取消",
      "start": "Bắt Đầu"
    },
    "dashboard": {
      "activeExecutions": "Đang Thực Thi",
      "noActive": "暂无活跃执行",
      "title": "Bảng Thuật Toán"
    },
    "fields": {
      "account": "Tài Khoản",
      "algo": "Thuật Toán",
      "limitPrice": "限价",
      "participationRate": "参与率",
      "side": "Hướng",
      "sliceInterval": "切片间隔",
      "symbol": "Mã",
      "timeRange": "时间范围",
      "urgency": "紧急度",
      "volume": "Khối Lượng"
    },
    "info": {
      "description": "描述",
      "name": "Tên"
    },
    "messages": {
      "started": "算法已启动"
    },
    "side": {
      "buy": "Mua",
      "sell": "Bán"
    },
    "submitForm": {
      "title": "启动算法"
    },
    "table": {
      "actions": "操作",
      "algo": "Thuật Toán",
      "executionId": "执行ID",
      "progress": "Tiến Độ",
      "side": "Hướng",
      "state": "Trạng Thái",
      "symbol": "Mã",
      "volume": "Khối Lượng"
    },
    "timePresets": {
      "EOD": "当日结束"
    }
  },
  "trading": {
    "account": "Tài Khoản",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "Xác nhận tắt",
        "disableInfoDescription": "Sau khi tắt, hệ thống sẽ dừng giao dịch tự động, nhưng các chiến lược đang bật vẫn có thể tiếp tục theo dõi thị trường.",
        "disableInfoTitle": "Tắt giao dịch tự động",
        "disableQuestion": "确定要关闭自动交易？",
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
      "fetchOrderHistoryFailed": "加载订单历史失败",
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
          "action": "检查账户状态和权限后重试。",
          "title": "Giao dịch đã bị tắt cho tài khoản này."
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "稍后重试；如问题持续请联系客服。",
          "title": "Quy tắc rủi ro tạm thời chưa khả dụng."
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "减少手数、平仓或充值。",
          "title": "Không đủ ký quỹ khả dụng để đặt lệnh này."
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "等待下一个交易时段后重试。",
          "title": "Thị trường của mã này đang đóng cửa."
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "平掉现有持仓或提高上限。",
          "title": "Đã đạt giới hạn số vị thế mở tối đa."
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "取消现有挂单或提高上限。",
          "title": "Đã đạt giới hạn số lệnh chờ tối đa."
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "等待价格离开冻结区域后重试。",
          "title": "Không thể sửa lệnh trong vùng đóng băng."
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "选择支持的订单类型后重试。",
          "title": "Loại lệnh này không được hỗ trợ cho mã đã chọn."
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "增加止损/止盈距离后重试。",
          "title": "Stop-loss hoặc take-profit quá gần giá thị trường."
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "切换到可交易品种或稍后重试。",
          "title": "Mã này hiện không thể giao dịch."
        },
        "RISK_VOLUME_INVALID": {
          "action": "调整手数以匹配最小/最大/步长要求。",
          "title": "Khối lượng lệnh không hợp lệ."
        },
        "unknown": {
          "action": "请检查订单参数后重试。",
          "title": "Yêu cầu giao dịch đã bị từ chối."
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "Lưu",
        "description": "Vui lòng xác nhận cấu hình rủi ro sau:",
        "info": "保存后，所有自动交易将遵循新的风险限额。",
        "title": "Xác nhận lưu cấu hình rủi ro"
      },
      "fields": {
        "maxDailyLoss": "Lỗ tối đa mỗi ngày",
        "maxDrawdownPercent": "Giới hạn drawdown",
        "maxLotSize": "Khối lượng tối đa",
        "maxPositions": "Số vị thế tối đa",
        "maxRiskPercent": "Rủi ro tối đa mỗi lệnh",
        "trailingStopEnabled": "Trailing stop",
        "trailingStopPips": "移动止损 (点)"
      }
    },
    "selectSymbol": "选择品种",
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
