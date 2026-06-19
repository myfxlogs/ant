// Auto-generated from proto/ant/v1/i18n/trading_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "trading": {
    "account": "账户",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "确认关闭",
        "disableInfoDescription": "关闭后，系统将停止自动执行交易，但已开启的策略仍会继续监控市场。",
        "disableInfoTitle": "关闭自动交易",
        "disableQuestion": "确定要禁用自动交易？",
        "disableTitle": "关闭自动交易",
        "enableBullet1": "系统将自动执行符合策略条件的交易",
        "enableBullet2": "请确保风险配置已正确设置",
        "enableBullet3": "建议先在模拟账户测试",
        "enableConfirm": "确认开启",
        "enableQuestion": "确认开启自动交易功能？",
        "enableRiskDescription": "开启自动交易后，系统将根据策略自动执行交易操作。请确保您已充分了解相关风险。",
        "enableRiskTitle": "风险提示",
        "enableTitle": "开启自动交易"
      }
    },
    "balance": "余额",
    "buy": "买入",
    "closePosition": "平仓",
    "closePositionConfirm": "确认平仓？",
    "closePositionTitle": "平仓",
    "equity": "净值",
    "freeMargin": "可用保证金",
    "limit": "限价",
    "margin": "已用保证金",
    "marginLevel": "保证金比例",
    "markPrice": "标记价",
    "market": "市价",
    "messages": {
      "fetchOrderHistoryFailed": "加载订单历史失败",
      "fetchPendingOrdersFailed": "获取挂单失败",
      "fetchPositionsFailed": "获取持仓失败",
      "orderCloseFailed": "平仓失败",
      "orderCloseSuccess": "平仓成功",
      "orderModifyFailed": "修改订单失败",
      "orderModifySuccess": "修改订单成功",
      "orderSendFailed": "下单失败",
      "orderSendSuccess": "下单成功"
    },
    "noAccount": "未选择账户",
    "noOrders": "暂无订单",
    "noPositions": "暂无持仓",
    "openPositionsTitle": "持仓",
    "openTime": "开仓时间",
    "orderHistory": "历史订单",
    "ordersCount": "{{count}} 条订单",
    "placeOrder": "下单",
    "pnl": "盈亏",
    "positionEntryPrice": "入场价",
    "positionLeverage": "杠杆",
    "positionLong": "多头",
    "positionMarkPrice": "标记价",
    "positionShort": "空头",
    "positionSide": "方向",
    "positionSize": "数量",
    "positionUnrealizedPnL": "未实现盈亏",
    "positions": "持仓",
    "price": "价格",
    "profit": "盈亏",
    "recentTrades": "最近交易",
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "请检查账户状态和权限后重试。",
          "title": "当前账户被禁止交易。"
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "稍后重试；如问题持续请联系客服。",
          "title": "风控规则暂不可用。"
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "请减少手数、平仓或追加资金。",
          "title": "可用保证金不足，无法下单。"
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "请等待下一个交易时段后重试。",
          "title": "当前品种处于休市时段。"
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "平掉现有持仓或提高上限。",
          "title": "已达到最大持仓数量限制。"
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "取消现有挂单或提高上限。",
          "title": "已达到最大挂单数量限制。"
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "请等待价格离开冻结区域后重试。",
          "title": "订单处于冻结区，当前不可修改。"
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "请选择支持的订单类型后重试。",
          "title": "当前品种不支持该订单类型。"
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "请增加止损/止盈距离后重试。",
          "title": "止损或止盈距离当前价格过近。"
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "请切换到可交易品种或稍后重试。",
          "title": "当前品种暂不可交易。"
        },
        "RISK_VOLUME_INVALID": {
          "action": "请调整手数以符合最小/最大/步长要求。",
          "title": "下单手数不合法。"
        },
        "unknown": {
          "action": "请检查订单参数后重试。",
          "title": "交易请求被拒绝。"
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "确认保存",
        "description": "请确认以下风险配置：",
        "info": "保存后，所有自动交易将遵循新的风险限额。",
        "title": "确认保存风险配置"
      },
      "fields": {
        "maxDailyLoss": "每日最大亏损",
        "maxDrawdownPercent": "最大回撤限制",
        "maxLotSize": "最大手数",
        "maxPositions": "最大持仓数量",
        "maxRiskPercent": "单笔最大风险",
        "trailingStopEnabled": "移动止损",
        "trailingStopPips": "移动止损 (点)"
      }
    },
    "selectSymbol": "选择品种",
    "sell": "卖出",
    "side": "方向",
    "stop": "止损",
    "stopLoss": "止损",
    "strategyExecute": {
      "confirm": {
        "action": "方向",
        "buy": "买入",
        "confirmText": "执行",
        "sell": "卖出",
        "strategyName": "策略名称",
        "symbol": "品种",
        "title": "确认交易执行",
        "volume": "数量",
        "warningDescription": "此操作将立即执行真实交易，请仔细核对交易参数。",
        "warningTitle": "交易执行确认"
      }
    },
    "symbol": "品种",
    "takeProfit": "止盈",
    "time": "时间",
    "title": "交易",
    "type": "类型",
    "volume": "数量"
  }
} as const;
export default Trading;
