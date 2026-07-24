// Auto-generated from proto/ant/v1/i18n/trading_zh-cn.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "trading": {
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "檢查賬戶狀態和許可權後重試。",
          "title": "當前賬戶被禁止交易。"
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "稍後重試；如問題持續請聯絡客服。",
          "title": "風控規則暫不可用。"
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "減少手數、平倉或充值。",
          "title": "可用保證金不足，無法下單。"
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "等待下一個交易時段後重試。",
          "title": "當前品種處於休市時段。"
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "平掉現有持倉或提高上限。",
          "title": "已達到最大持倉數量限制。"
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "取消現有掛單或提高上限。",
          "title": "已達到最大掛單數量限制。"
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "等待價格離開凍結區域後重試。",
          "title": "訂單處於凍結區，當前不可修改。"
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "選擇支援的訂單型別後重試。",
          "title": "當前品種不支援該訂單型別。"
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "增加止損/止盈距離後重試。",
          "title": "止損或止盈距離當前價格過近。"
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "切換到可交易品種或稍後重試。",
          "title": "當前品種暫不可交易。"
        },
        "RISK_VOLUME_INVALID": {
          "action": "調整手數以匹配最小/最大/步長要求。",
          "title": "下單手數不合法。"
        },
        "unknown": {
          "action": "請檢查訂單引數後重試。",
          "title": "交易請求被拒絕。"
        }
      }
    },
    "autoTrade": {
      "confirm": {
        "disableConfirm": "確認關閉",
        "disableInfoDescription": "關閉後，系統將停止自動執行交易，但已開啟的策略仍會繼續監控市場。",
        "disableInfoTitle": "關閉自動交易",
        "disableQuestion": "確定要關閉自動交易？",
        "disableTitle": "關閉自動交易",
        "enableBullet1": "系統將自動執行符合策略條件的交易",
        "enableBullet2": "請確保風險配置已正確設定",
        "enableBullet3": "建議先在模擬賬戶測試",
        "enableConfirm": "確認開啟",
        "enableQuestion": "確認開啟自動交易功能？",
        "enableRiskDescription": "開啟自動交易後，系統將根據策略自動執行交易操作。請確保您已充分了解相關風險。",
        "enableRiskTitle": "風險提示",
        "enableTitle": "開啟自動交易"
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "確認儲存",
        "description": "請確認以下風險配置：",
        "info": "儲存後，所有自動交易將遵循新的風險限額。",
        "title": "確認儲存風險配置"
      },
      "fields": {
        "maxDailyLoss": "每日最大虧損",
        "maxDrawdownPercent": "最大回撤限制",
        "maxLotSize": "最大手數",
        "maxPositions": "最大持倉數量",
        "maxRiskPercent": "單筆最大風險",
        "trailingStopEnabled": "移動止損",
        "trailingStopPips": "移動止損 (點)"
      }
    },
    "strategyExecute": {
      "confirm": {
        "action": "方向",
        "buy": "買入",
        "confirmText": "執行",
        "sell": "賣出",
        "strategyName": "策略名稱",
        "symbol": "品種",
        "title": "確認交易執行",
        "volume": "數量",
        "warningDescription": "此操作將立即執行真實交易，請仔細核對交易引數。",
        "warningTitle": "交易執行確認"
      }
    },
    "messages": {
      "fetchOrderHistoryFailed": "載入訂單歷史失敗",
      "fetchPendingOrdersFailed": "獲取掛單失敗",
      "fetchPositionsFailed": "獲取持倉失敗",
      "orderCloseFailed": "平倉失敗",
      "orderCloseSuccess": "平倉成功",
      "orderModifyFailed": "修改訂單失敗",
      "orderModifySuccess": "修改訂單成功",
      "orderSendFailed": "下單失敗",
      "orderSendSuccess": "下單成功"
    },
    "account": "賬戶",
    "balance": "餘額",
    "buy": "買入",
    "closePosition": "平倉",
    "closePositionConfirm": "確認平倉？",
    "closePositionTitle": "平倉",
    "equity": "淨值",
    "freeMargin": "可用保證金",
    "limit": "限價",
    "margin": "已用保證金",
    "marginLevel": "保證金比例",
    "markPrice": "標記價",
    "market": "市價",
    "noAccount": "未選擇賬戶",
    "noOrders": "暫無訂單",
    "noPositions": "暫無持倉",
    "openPositionsTitle": "持倉",
    "openTime": "開倉時間",
    "orderHistory": "歷史訂單",
    "ordersCount": "{{count}} 條訂單",
    "placeOrder": "下單",
    "pnl": "盈虧",
    "positionEntryPrice": "入場價",
    "positionLeverage": "槓桿",
    "positionLong": "多頭",
    "positionMarkPrice": "標記價",
    "positionShort": "空頭",
    "positionSide": "方向",
    "positionSize": "數量",
    "positionUnrealizedPnL": "未實現盈虧",
    "positions": "持倉",
    "price": "價格",
    "profit": "盈虧",
    "recentTrades": "最近交易",
    "selectSymbol": "選擇品種",
    "sell": "賣出",
    "side": "方向",
    "stop": "止損",
    "stopLoss": "止損",
    "symbol": "品種",
    "takeProfit": "止盈",
    "time": "時間",
    "title": "交易",
    "type": "型別",
    "volume": "數量",
    "platform": "平臺",
    "broker": "經紀商",
    "server": "伺服器",
    "permission": "許可權",
    "investor": "觀察者",
    "master": "交易者",
    "leverage": "槓桿"
  },
  "algo": {
    "actions": {
      "cancel": "取消",
      "start": "啟動"
    },
    "dashboard": {
      "activeExecutions": "執行中",
      "noActive": "暫無活躍執行",
      "title": "演算法面板"
    },
    "fields": {
      "account": "賬戶",
      "algo": "演算法",
      "limitPrice": "限價",
      "participationRate": "參與率",
      "side": "方向",
      "sliceInterval": "切片間隔",
      "symbol": "品種",
      "timeRange": "時間範圍",
      "urgency": "緊急度",
      "volume": "數量"
    },
    "info": {
      "description": "描述",
      "name": "名稱"
    },
    "messages": {
      "started": "演算法已啟動"
    },
    "side": {
      "buy": "買入",
      "sell": "賣出"
    },
    "submitForm": {
      "title": "啟動演算法"
    },
    "table": {
      "actions": "操作",
      "algo": "演算法",
      "executionId": "執行ID",
      "progress": "進度",
      "side": "方向",
      "state": "狀態",
      "symbol": "品種",
      "volume": "數量"
    },
    "timePresets": {
      "EOD": "當日結束"
    },
    "twap": {
      "name": "TWAP (時間加權均價)",
      "description": "時間加權平均價格 — 將大單拆分為小塊，在時間維度上均勻分佈執行。"
    },
    "vwap": {
      "name": "VWAP (成交量加權均價)",
      "description": "成交量加權平均價格 — 根據歷史成交量分佈按比例執行訂單。"
    },
    "pov": {
      "name": "POV (參與率)",
      "description": "參與率演算法 — 以固定比例參與市場成交量。"
    },
    "shortfall": {
      "name": "Shortfall (最小缺口)",
      "description": "實現缺口最小化 — 最小化決策價格與執行價格之間的差異。"
    },
    "label": {
      "twap": "TWAP (時間加權均價)",
      "vwap": "VWAP (成交量加權均價)",
      "pov": "POV (參與率演算法)",
      "shortfall": "Shortfall (最小缺口)"
    }
  }
} as const;
export default Trading;
