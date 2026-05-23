const trading = {
  trading: {
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: '当前账户被禁止交易。',
          action: '请检查账户状态与权限后重试。',
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: '当前品种暂不可交易。',
          action: '请切换可交易品种或稍后再试。',
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: '当前品种处于休市时段。',
          action: '请等待下一个交易时段后重试。',
        },
        RISK_VOLUME_INVALID: {
          title: '下单手数不合法。',
          action: '请按最小值/最大值/步长规则调整手数。',
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: '当前品种不支持该订单类型。',
          action: '请改用支持的订单类型后重试。',
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: '止损或止盈距离当前价格过近。',
          action: '请增大止损/止盈距离后重试。',
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: '订单处于冻结区，当前不可修改。',
          action: '请等待价格离开冻结区后再重试。',
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: '可用保证金不足，无法下单。',
          action: '请降低手数、先平部分仓位或补充资金。',
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: '已达到最大持仓数量限制。',
          action: '请先平掉部分持仓或提高持仓上限。',
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: '已达到最大挂单数量限制。',
          action: '请先取消部分挂单或提高挂单上限。',
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: '风控规则暂不可用。',
          action: '请稍后重试，若持续失败请联系支持。',
        },
        unknown: {
          title: '交易请求被拒绝。',
          action: '请检查订单参数后重试。',
        },
      },
    },
    messages: {
      fetchPositionsFailed: '获取持仓失败',
      orderSendSuccess: '下单成功',
      orderSendFailed: '下单失败',
      orderModifySuccess: '修改订单成功',
      orderModifyFailed: '修改订单失败',
      orderCloseSuccess: '平仓成功',
      orderCloseFailed: '平仓失败',
      fetchPendingOrdersFailed: '获取挂单失败',
      fetchOrderHistoryFailed: '获取历史订单失败',
    },
    autoTrade: {
      confirm: {
        enableTitle: '开启自动交易',
        disableTitle: '关闭自动交易',
        enableConfirm: '确认开启',
        disableConfirm: '确认关闭',
        enableRiskTitle: '风险提示',
        enableRiskDescription: '开启自动交易后，系统将根据策略自动执行交易操作。请确保您已充分了解相关风险。',
        enableQuestion: '确认开启自动交易功能？',
        enableBullet1: '系统将自动执行符合策略条件的交易',
        enableBullet2: '请确保风险配置已正确设置',
        enableBullet3: '建议先在模拟账户测试',
        disableInfoTitle: '关闭自动交易',
        disableInfoDescription: '关闭后，系统将停止自动执行交易，但已开启的策略仍会继续监控市场。',
        disableQuestion: '确认关闭自动交易功能？',
      },
    },
    riskConfig: {
      fields: {
        maxRiskPercent: '单笔最大风险',
        maxDailyLoss: '每日最大亏损',
        maxDrawdownPercent: '最大回撤限制',
        maxPositions: '最大持仓数量',
        maxLotSize: '最大手数',
        trailingStopEnabled: '移动止损',
        trailingStopPips: '移动止损点数',
      },
      confirm: {
        title: '确认保存风险配置',
        confirmText: '确认保存',
        description: '请确认以下风险配置：',
        info: '配置生效后，所有自动交易将遵循新的风险限制。',
      },
    },
    strategyExecute: {
      confirm: {
        title: '确认交易执行',
        confirmText: '执行',
        warningTitle: '交易执行确认',
        warningDescription: '此操作将立即执行真实交易，请仔细核对交易参数。',
        strategyName: '策略名称',
        symbol: '交易品种',
        action: '交易方向',
        buy: '买入',
        sell: '卖出',
        volume: '交易手数',
      },
    },
  },
} as const;

export default trading;
