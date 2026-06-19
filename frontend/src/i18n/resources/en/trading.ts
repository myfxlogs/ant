// Auto-generated from proto/ant/v1/i18n/trading_en.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "trading": {
    "account": "Account",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "Disable",
        "disableInfoDescription": "Auto trading will stop placing new orders.",
        "disableInfoTitle": "Disable auto trading",
        "disableQuestion": "Are you sure you want to disable auto trading?",
        "disableTitle": "Disable auto trading",
        "enableBullet1": "Orders will be executed automatically.",
        "enableBullet2": "Market volatility can cause losses.",
        "enableBullet3": "You can disable auto trading at any time.",
        "enableConfirm": "Enable",
        "enableQuestion": "Are you sure you want to enable auto trading?",
        "enableRiskDescription": "Auto trading will place orders automatically. Please ensure you understand the risks.",
        "enableRiskTitle": "Risk notice",
        "enableTitle": "Enable auto trading"
      }
    },
    "balance": "Balance",
    "buy": "Buy",
    "closePosition": "Close",
    "closePositionConfirm": "Close this position?",
    "closePositionTitle": "Close Position",
    "equity": "Equity",
    "freeMargin": "Free Margin",
    "limit": "Limit",
    "margin": "Margin",
    "marginLevel": "Margin Level",
    "markPrice": "Mark Price",
    "market": "Market",
    "messages": {
      "fetchOrderHistoryFailed": "Failed to load order history",
      "fetchPendingOrdersFailed": "Failed to load pending orders",
      "fetchPositionsFailed": "Failed to load positions",
      "orderCloseFailed": "Failed to close position",
      "orderCloseSuccess": "Position closed successfully",
      "orderModifyFailed": "Failed to update order",
      "orderModifySuccess": "Order updated successfully",
      "orderSendFailed": "Failed to place order",
      "orderSendSuccess": "Order placed successfully"
    },
    "noAccount": "No account selected",
    "noOrders": "No orders yet",
    "noPositions": "No open positions",
    "openPositionsTitle": "Open Positions",
    "openTime": "Open Time",
    "orderHistory": "Order History",
    "ordersCount": "{{count}} orders",
    "placeOrder": "Place Order",
    "pnl": "P&L",
    "positionEntryPrice": "Entry Price",
    "positionLeverage": "Leverage",
    "positionLong": "LONG",
    "positionMarkPrice": "Mark Price",
    "positionShort": "SHORT",
    "positionSide": "Side",
    "positionSize": "Size",
    "positionUnrealizedPnL": "Unrealized PnL",
    "positions": "Positions",
    "price": "Price",
    "profit": "Profit",
    "recentTrades": "Recent Trades",
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "Check account status and permissions, then try again.",
          "title": "Trading is disabled for this account."
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "Retry later; contact support if the issue persists.",
          "title": "Risk rules are temporarily unavailable."
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "Reduce volume, close positions, or add funds.",
          "title": "Insufficient free margin to place this order."
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "Wait for the next trading session and retry.",
          "title": "Market is closed for this symbol."
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "Close existing positions or raise the limit.",
          "title": "Maximum open positions limit reached."
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "Cancel existing pending orders or raise the limit.",
          "title": "Maximum pending orders limit reached."
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "Wait until price moves away from freeze distance, then retry.",
          "title": "Order cannot be modified in the freeze zone."
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "Choose a supported order type and retry.",
          "title": "This order type is not supported for the symbol."
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "Increase SL/TP distance and retry.",
          "title": "Stop-loss or take-profit is too close to market price."
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "Switch to a tradable symbol or try later.",
          "title": "This symbol is currently not tradable."
        },
        "RISK_VOLUME_INVALID": {
          "action": "Adjust volume to match min/max/step requirements.",
          "title": "Order volume is invalid."
        },
        "unknown": {
          "action": "Please review order parameters and try again.",
          "title": "Trade request was rejected."
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "Save",
        "description": "Please confirm the following risk settings:",
        "info": "After saving, all auto trading will follow the new risk limits.",
        "title": "Confirm Risk Settings"
      },
      "fields": {
        "maxDailyLoss": "Max Daily Loss",
        "maxDrawdownPercent": "Max Drawdown Limit",
        "maxLotSize": "Max Lot Size",
        "maxPositions": "Max Open Positions",
        "maxRiskPercent": "Max Risk per Trade",
        "trailingStopEnabled": "Trailing Stop",
        "trailingStopPips": "Trailing Stop (pips)"
      }
    },
    "selectSymbol": "Select a symbol",
    "sell": "Sell",
    "side": "Side",
    "stop": "Stop",
    "stopLoss": "Stop Loss",
    "strategyExecute": {
      "confirm": {
        "action": "Side",
        "buy": "Buy",
        "confirmText": "Execute",
        "sell": "Sell",
        "strategyName": "Strategy",
        "symbol": "Symbol",
        "title": "Confirm Trade Execution",
        "volume": "Volume",
        "warningDescription": "This action will place a real trade immediately. Please verify all parameters.",
        "warningTitle": "Trade execution confirmation"
      }
    },
    "symbol": "Symbol",
    "takeProfit": "Take Profit",
    "time": "Time",
    "title": "Trading",
    "type": "Type",
    "volume": "Volume"
  }
} as const;
export default Trading;
