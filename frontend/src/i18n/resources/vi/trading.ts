const trading = {
  trading: {
    title: 'Giao dịch',
    account: 'Tài Khoản',
    balance: 'Số dư',
    equity: 'Vốn chủ sở hữu',
    margin: 'Ký quỹ',
    freeMargin: 'Ký quỹ tự do',
    marginLevel: 'Mức ký quỹ',
    noAccount: 'Chưa chọn tài khoản',
    placeOrder: 'Đặt lệnh',
    symbol: 'Mã',
    type: 'Loại',
    volume: 'Khối Lượng',
    price: 'Giá',
    stopLoss: 'Cắt lỗ',
    takeProfit: 'Chốt lời',
    side: 'Hướng',
    buy: 'Mua',
    sell: 'Bán',
    market: 'Thị trường',
    limit: 'Giới hạn',
    stop: 'Dừng',
    positions: 'Vị thế',
    noPositions: 'Không có vị thế mở',
    closePosition: 'Đóng',
    closePositionConfirm: 'Đóng vị thế này?',
    openTime: 'Thời gian mở',
    orderHistory: 'Lịch sử lệnh',
    noOrders: 'Chưa có lệnh nào',
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: 'Giao dịch đã bị tắt cho tài khoản này.',
          action: 'Check account status and permissions, then try again.'
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: 'Mã này hiện không thể giao dịch.',
          action: 'Switch to a tradable symbol or try later.'
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: 'Thị trường của mã này đang đóng cửa.',
          action: 'Wait for the next trading session and retry.'
        },
        RISK_VOLUME_INVALID: {
          title: 'Khối lượng lệnh không hợp lệ.',
          action: 'Adjust volume to match min/max/step requirements.'
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: 'Loại lệnh này không được hỗ trợ cho mã đã chọn.',
          action: 'Choose a supported order type and retry.'
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: 'Stop-loss hoặc take-profit quá gần giá thị trường.',
          action: 'Increase SL/TP distance and retry.'
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: 'Không thể sửa lệnh trong vùng đóng băng.',
          action: 'Wait until price moves away from freeze distance, then retry.'
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: 'Không đủ ký quỹ khả dụng để đặt lệnh này.',
          action: 'Reduce volume, close positions, or add funds.'
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: 'Đã đạt giới hạn số vị thế mở tối đa.',
          action: 'Close existing positions or raise the limit.'
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: 'Đã đạt giới hạn số lệnh chờ tối đa.',
          action: 'Cancel existing pending orders or raise the limit.'
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: 'Quy tắc rủi ro tạm thời chưa khả dụng.',
          action: 'Retry later; contact support if the issue persists.'
        },
        unknown: {
          title: 'Yêu cầu giao dịch đã bị từ chối.',
          action: 'Please review order parameters and try again.'
        }
      }
    },
    messages: {
      fetchPositionsFailed: 'Không thể tải vị thế',
      orderSendSuccess: 'Đặt lệnh thành công',
      orderSendFailed: 'Đặt lệnh thất bại',
      orderModifySuccess: 'Sửa lệnh thành công',
      orderModifyFailed: 'Sửa lệnh thất bại',
      orderCloseSuccess: 'Đóng lệnh thành công',
      orderCloseFailed: 'Đóng lệnh thất bại',
      fetchPendingOrdersFailed: 'Không thể tải lệnh chờ',
      fetchOrderHistoryFailed: 'Failed to load order history'
    },
    riskConfig: {
      fields: {
        maxRiskPercent: 'Rủi ro tối đa mỗi lệnh',
        maxDailyLoss: 'Lỗ tối đa mỗi ngày',
        maxDrawdownPercent: 'Giới hạn drawdown',
        maxPositions: 'Số vị thế tối đa',
        maxLotSize: 'Khối lượng tối đa',
        trailingStopEnabled: 'Trailing stop',
        trailingStopPips: 'Trailing Stop (pips)'
      },
      confirm: {
        title: 'Xác nhận lưu cấu hình rủi ro',
        confirmText: 'Lưu',
        description: 'Vui lòng xác nhận cấu hình rủi ro sau:',
        info: 'After saving, all auto trading will follow the new risk limits.'
      }
    },
    strategyExecute: {
      confirm: {
        title: 'Xác nhận thực thi giao dịch',
        confirmText: 'Thực thi',
        warningTitle: 'Xác nhận thực thi giao dịch',
        warningDescription: 'Thao tác này sẽ thực hiện giao dịch thật ngay lập tức. Vui lòng kiểm tra kỹ tham số.',
        strategyName: 'Chiến lược',
        symbol: 'Mã',
        action: 'Hướng',
        buy: 'Mua',
        sell: 'Bán',
        volume: 'Khối Lượng'
      }
    },
    autoTrade: {
      confirm: {
        enableTitle: 'Bật giao dịch tự động',
        disableTitle: 'Tắt giao dịch tự động',
        enableConfirm: 'Xác nhận bật',
        disableConfirm: 'Xác nhận tắt',
        enableRiskTitle: 'Cảnh báo rủi ro',
        enableRiskDescription: 'Khi bật giao dịch tự động, hệ thống sẽ tự động thực hiện giao dịch theo chiến lược. Vui lòng chắc chắn bạn hiểu rõ các rủi ro.',
        enableQuestion: 'Bật tính năng giao dịch tự động?',
        enableBullet1: 'Hệ thống sẽ tự động thực hiện giao dịch khi điều kiện chiến lược thỏa mãn',
        enableBullet2: 'Hãy đảm bảo cấu hình rủi ro đã được thiết lập đúng',
        enableBullet3: 'Nên thử nghiệm trước trên tài khoản demo',
        disableInfoTitle: 'Tắt giao dịch tự động',
        disableInfoDescription: 'Sau khi tắt, hệ thống sẽ dừng giao dịch tự động, nhưng các chiến lược đang bật vẫn có thể tiếp tục theo dõi thị trường.',
        disableQuestion: 'Are you sure you want to disable auto trading?'
      }
    },
    pnl: 'Lãi/Lỗ',
    profit: 'Lợi nhuận',
    time: 'Thời gian',
    ordersCount: '{{count}} lệnh',
    markPrice: 'Giá thị trường',
    positionSide: 'Hướng',
    positionSize: 'Khối lượng',
    positionEntryPrice: 'Giá vào lệnh',
    positionMarkPrice: 'Giá thị trường',
    positionLeverage: 'Đòn bẩy',
    positionUnrealizedPnL: 'Lãi/Lỗ thả nổi',
    positionLong: 'LONG',
    positionShort: 'SHORT',
    openPositionsTitle: 'Vị thế mở',
    closePositionTitle: 'Đóng vị thế',
    recentTrades: 'Giao dịch gần đây',
    selectSymbol: 'Select a symbol'
  },
  algo: {
    submitForm: {
      title: 'Launch Algo'
    },
    actions: {
      start: 'Bắt Đầu',
      cancel: 'Cancel'
    },
    fields: {
      algo: 'Thuật Toán',
      symbol: 'Mã',
      side: 'Hướng',
      volume: 'Khối Lượng',
      limitPrice: '限价',
      account: 'Tài Khoản',
      timeRange: '时间范围',
      urgency: '紧急度',
      sliceInterval: '切片间隔',
      participationRate: 'Participation Rate'
    },
    side: {
      buy: 'Mua',
      sell: 'Bán'
    },
    info: {
      name: 'Tên',
      description: 'Description'
    },
    messages: {
      started: 'Algo started'
    },
    timePresets: {
      '1h': '1 Hour',
      '4h': '4 Hours',
      EOD: 'End of Day'
    },
    dashboard: {
      title: 'Bảng Thuật Toán',
      activeExecutions: 'Đang Thực Thi',
      noActive: 'No active algo executions'
    },
    table: {
      executionId: '执行ID',
      algo: 'Thuật Toán',
      symbol: 'Mã',
      side: 'Hướng',
      volume: 'Khối Lượng',
      progress: 'Tiến Độ',
      state: 'Trạng Thái',
      actions: 'Actions'
    }
  }
} as const;

export default trading;
