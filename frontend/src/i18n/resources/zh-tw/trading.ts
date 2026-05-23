const trading = {
  trading: {
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: '當前帳戶被禁止交易。',
          action: '請檢查帳戶狀態與權限後重試。',
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: '當前商品暫不可交易。',
          action: '請切換可交易商品或稍後再試。',
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: '當前商品處於休市時段。',
          action: '請等待下一個交易時段後重試。',
        },
        RISK_VOLUME_INVALID: {
          title: '下單手數不合法。',
          action: '請依最小值/最大值/步長規則調整手數。',
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: '當前商品不支援該訂單類型。',
          action: '請改用支援的訂單類型後重試。',
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: '停損或停利距離當前價格過近。',
          action: '請增加停損/停利距離後重試。',
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: '訂單處於凍結區，當前不可修改。',
          action: '請等待價格離開凍結區後再重試。',
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: '可用保證金不足，無法下單。',
          action: '請降低手數、先平部分持倉或補充資金。',
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: '已達到最大持倉數量限制。',
          action: '請先平掉部分持倉或提高持倉上限。',
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: '已達到最大掛單數量限制。',
          action: '請先取消部分掛單或提高掛單上限。',
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: '風控規則暫不可用。',
          action: '請稍後重試，若持續失敗請聯絡支援。',
        },
        unknown: {
          title: '交易請求被拒絕。',
          action: '請檢查訂單參數後重試。',
        },
      },
    },
    messages: {
      fetchPositionsFailed: '取得持倉失敗',
      orderSendSuccess: '下單成功',
      orderSendFailed: '下單失敗',
      orderModifySuccess: '修改訂單成功',
      orderModifyFailed: '修改訂單失敗',
      orderCloseSuccess: '平倉成功',
      orderCloseFailed: '平倉失敗',
      fetchPendingOrdersFailed: '取得掛單失敗',
      fetchOrderHistoryFailed: '取得歷史訂單失敗',
    },
    autoTrade: {
      confirm: {
        enableTitle: '開啟自動交易',
        disableTitle: '關閉自動交易',
        enableConfirm: '確認開啟',
        disableConfirm: '確認關閉',
        enableRiskTitle: '風險提示',
        enableRiskDescription: '開啟自動交易後，系統將依策略自動執行交易。請確認你已充分了解相關風險。',
        enableQuestion: '確認開啟自動交易功能？',
        enableBullet1: '系統將自動執行符合策略條件的交易',
        enableBullet2: '請確認風險配置已正確設定',
        enableBullet3: '建議先在模擬帳戶測試',
        disableInfoTitle: '關閉自動交易',
        disableInfoDescription: '關閉後，系統將停止自動交易，但已啟用的策略仍可能繼續監控市場。',
        disableQuestion: '確認關閉自動交易功能？',
      },
    },
    riskConfig: {
      fields: {
        maxRiskPercent: '單筆最大風險',
        maxDailyLoss: '每日最大虧損',
        maxDrawdownPercent: '最大回撤限制',
        maxPositions: '最大持倉數量',
        maxLotSize: '最大手數',
        trailingStopEnabled: '移動止損',
        trailingStopPips: '移動止損點數',
      },
      confirm: {
        title: '確認保存風險配置',
        confirmText: '確認保存',
        description: '請確認以下風險配置：',
        info: '配置生效後，所有自動交易將遵循新的風險限制。',
      },
    },
    strategyExecute: {
      confirm: {
        title: '確認執行交易',
        confirmText: '確認執行',
        warningTitle: '交易執行確認',
        warningDescription: '此操作將立即執行真實交易，請仔細核對交易參數。',
        strategyName: '策略名稱',
        symbol: '交易品種',
        action: '交易方向',
        buy: '買入',
        sell: '賣出',
        volume: '交易手數',
      },
    },
  },
} as const;

export default trading;
