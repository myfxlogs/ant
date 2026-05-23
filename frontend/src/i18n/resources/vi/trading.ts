const trading = {
  trading: {
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: 'Giao dịch đã bị tắt cho tài khoản này.',
          action: 'Kiểm tra trạng thái tài khoản và quyền truy cập rồi thử lại.',
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: 'Mã này hiện không thể giao dịch.',
          action: 'Chuyển sang mã có thể giao dịch hoặc thử lại sau.',
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: 'Thị trường của mã này đang đóng cửa.',
          action: 'Chờ phiên giao dịch tiếp theo rồi thử lại.',
        },
        RISK_VOLUME_INVALID: {
          title: 'Khối lượng lệnh không hợp lệ.',
          action: 'Điều chỉnh khối lượng theo giới hạn min/max/step.',
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: 'Loại lệnh này không được hỗ trợ cho mã đã chọn.',
          action: 'Chọn loại lệnh được hỗ trợ rồi thử lại.',
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: 'Stop-loss hoặc take-profit quá gần giá thị trường.',
          action: 'Tăng khoảng cách SL/TP rồi thử lại.',
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: 'Không thể sửa lệnh trong vùng đóng băng.',
          action: 'Chờ giá rời khỏi khoảng đóng băng rồi thử lại.',
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: 'Không đủ ký quỹ khả dụng để đặt lệnh này.',
          action: 'Giảm khối lượng, đóng bớt vị thế hoặc nạp thêm tiền.',
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: 'Đã đạt giới hạn số vị thế mở tối đa.',
          action: 'Đóng bớt vị thế hiện có hoặc tăng giới hạn.',
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: 'Đã đạt giới hạn số lệnh chờ tối đa.',
          action: 'Hủy bớt lệnh chờ hiện có hoặc tăng giới hạn.',
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: 'Quy tắc rủi ro tạm thời chưa khả dụng.',
          action: 'Thử lại sau; liên hệ hỗ trợ nếu lỗi vẫn còn.',
        },
        unknown: {
          title: 'Yêu cầu giao dịch đã bị từ chối.',
          action: 'Vui lòng kiểm tra lại tham số lệnh và thử lại.',
        },
      },
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
      fetchOrderHistoryFailed: 'Không thể tải lịch sử lệnh',
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
        disableQuestion: 'Tắt tính năng giao dịch tự động?',
      },
    },
    riskConfig: {
      fields: {
        maxRiskPercent: 'Rủi ro tối đa mỗi lệnh',
        maxDailyLoss: 'Lỗ tối đa mỗi ngày',
        maxDrawdownPercent: 'Giới hạn drawdown',
        maxPositions: 'Số vị thế tối đa',
        maxLotSize: 'Khối lượng tối đa',
        trailingStopEnabled: 'Trailing stop',
        trailingStopPips: 'Trailing stop (pips)',
      },
      confirm: {
        title: 'Xác nhận lưu cấu hình rủi ro',
        confirmText: 'Lưu',
        description: 'Vui lòng xác nhận cấu hình rủi ro sau:',
        info: 'Sau khi có hiệu lực, mọi giao dịch tự động sẽ tuân theo giới hạn rủi ro mới.',
      },
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
        volume: 'Khối lượng',
      },
    },
  },
} as const;

export default trading;
