// Auto-generated from proto/ant/v1/i18n/trading_zh-tw.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "algo": {
    "actions": {
      "cancel": "Cancel",
      "start": "啟動"
    },
    "dashboard": {
      "activeExecutions": "執行中",
      "noActive": "No active algo executions",
      "title": "演算法面板"
    },
    "fields": {
      "account": "账戶",
      "algo": "演算法",
      "limitPrice": "限價",
      "participationRate": "Participation Rate",
      "side": "方向",
      "sliceInterval": "切片間隔",
      "symbol": "商品",
      "timeRange": "時間範圍",
      "urgency": "緊急度",
      "volume": "數量"
    },
    "info": {
      "description": "Description",
      "name": "名稱"
    },
    "messages": {
      "started": "Algo started"
    },
    "side": {
      "buy": "買入",
      "sell": "賣出"
    },
    "submitForm": {
      "title": "Launch Algo"
    },
    "table": {
      "actions": "Actions",
      "algo": "演算法",
      "executionId": "執行ID",
      "progress": "進度",
      "side": "方向",
      "state": "狀態",
      "symbol": "商品",
      "volume": "數量"
    },
    "timePresets": {
      "EOD": "End of Day"
    }
  },
  "trading": {
    "account": "账戶",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "確認關閉",
        "disableInfoDescription": "關閉後，系統將停止自動交易，但已啟用的策略仍可能繼續監控市場。",
        "disableInfoTitle": "關閉自動交易",
        "disableQuestion": "Are you sure you want to disable auto trading?",
        "disableTitle": "關閉自動交易",
        "enableBullet1": "系統將自動執行符合策略條件的交易",
        "enableBullet2": "請確認風險配置已正確設定",
        "enableBullet3": "建議先在模擬帳戶測試",
        "enableConfirm": "確認開啟",
        "enableQuestion": "確認開啟自動交易功能？",
        "enableRiskDescription": "開啟自動交易後，系統將依策略自動執行交易。請確認你已充分了解相關風險。",
        "enableRiskTitle": "風險提示",
        "enableTitle": "開啟自動交易"
      }
    },
    "balance": "餘額",
    "buy": "買入",
    "closePosition": "平倉",
    "closePositionConfirm": "確定平倉此持倉？",
    "closePositionTitle": "平倉",
    "equity": "淨值",
    "freeMargin": "可用保證金",
    "limit": "限價",
    "margin": "保證金",
    "marginLevel": "保證金比例",
    "markPrice": "標記價格",
    "market": "市價",
    "messages": {
      "fetchOrderHistoryFailed": "Failed to load order history",
      "fetchPendingOrdersFailed": "取得掛單失敗",
      "fetchPositionsFailed": "取得持倉失敗",
      "orderCloseFailed": "平倉失敗",
      "orderCloseSuccess": "平倉成功",
      "orderModifyFailed": "修改訂單失敗",
      "orderModifySuccess": "修改訂單成功",
      "orderSendFailed": "下單失敗",
      "orderSendSuccess": "下單成功"
    },
    "noAccount": "未選擇帳戶",
    "noOrders": "暫無訂單",
    "noPositions": "暫無持倉",
    "openPositionsTitle": "持倉",
    "openTime": "開倉時間",
    "orderHistory": "訂單歷史",
    "ordersCount": "{{count}} 筆訂單",
    "placeOrder": "下單",
    "pnl": "盈虧",
    "positionEntryPrice": "進場價格",
    "positionLeverage": "槓桿",
    "positionLong": "做多",
    "positionMarkPrice": "標記價格",
    "positionShort": "做空",
    "positionSide": "方向",
    "positionSize": "數量",
    "positionUnrealizedPnL": "未實現盈虧",
    "positions": "持倉",
    "price": "價格",
    "profit": "盈虧",
    "recentTrades": "近期交易",
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "Check account status and permissions, then try again.",
          "title": "當前帳戶被禁止交易。"
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "Retry later; contact support if the issue persists.",
          "title": "風控規則暫不可用。"
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "Reduce volume, close positions, or add funds.",
          "title": "可用保證金不足，無法下單。"
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "Wait for the next trading session and retry.",
          "title": "當前商品處於休市時段。"
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "Close existing positions or raise the limit.",
          "title": "已達到最大持倉數量限制。"
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "Cancel existing pending orders or raise the limit.",
          "title": "已達到最大掛單數量限制。"
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "Wait until price moves away from freeze distance, then retry.",
          "title": "訂單處於凍結區，當前不可修改。"
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "Choose a supported order type and retry.",
          "title": "當前商品不支援該訂單類型。"
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "Increase SL/TP distance and retry.",
          "title": "停損或停利距離當前價格過近。"
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "Switch to a tradable symbol or try later.",
          "title": "當前商品暫不可交易。"
        },
        "RISK_VOLUME_INVALID": {
          "action": "Adjust volume to match min/max/step requirements.",
          "title": "下單手數不合法。"
        },
        "unknown": {
          "action": "Please review order parameters and try again.",
          "title": "交易請求被拒絕。"
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "確認保存",
        "description": "請確認以下風險配置：",
        "info": "After saving, all auto trading will follow the new risk limits.",
        "title": "確認保存風險配置"
      },
      "fields": {
        "maxDailyLoss": "每日最大虧損",
        "maxDrawdownPercent": "最大回撤限制",
        "maxLotSize": "最大手數",
        "maxPositions": "最大持倉數量",
        "maxRiskPercent": "單筆最大風險",
        "trailingStopEnabled": "移動止損",
        "trailingStopPips": "Trailing Stop (pips)"
      }
    },
    "selectSymbol": "Select a symbol",
    "sell": "賣出",
    "side": "方向",
    "stop": "止損",
    "stopLoss": "止損",
    "strategyExecute": {
      "confirm": {
        "action": "方向",
        "buy": "買入",
        "confirmText": "確認執行",
        "sell": "賣出",
        "strategyName": "策略名稱",
        "symbol": "商品",
        "title": "確認執行交易",
        "volume": "數量",
        "warningDescription": "此操作將立即執行真實交易，請仔細核對交易參數。",
        "warningTitle": "交易執行確認"
      }
    },
    "symbol": "商品",
    "takeProfit": "止盈",
    "time": "時間",
    "title": "交易",
    "type": "類型",
    "volume": "數量"
  }
} as const;
export default Trading;
