const trading = {
  trading: {
    title: '交易',
    account: '账戶',
    balance: '餘額',
    equity: '淨值',
    margin: '保證金',
    freeMargin: '可用保證金',
    marginLevel: '保證金比例',
    noAccount: '未選擇帳戶',
    placeOrder: '下單',
    symbol: '商品',
    type: '類型',
    volume: '數量',
    price: '價格',
    stopLoss: '止損',
    takeProfit: '止盈',
    side: '方向',
    buy: '買入',
    sell: '賣出',
    market: '市價',
    limit: '限價',
    stop: '止損',
    positions: '持倉',
    noPositions: '暫無持倉',
    closePosition: '平倉',
    closePositionConfirm: '確定平倉此持倉？',
    openTime: '開倉時間',
    orderHistory: '訂單歷史',
    noOrders: '暫無訂單',
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: '當前帳戶被禁止交易。',
          action: 'Check account status and permissions, then try again.'
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: '當前商品暫不可交易。',
          action: 'Switch to a tradable symbol or try later.'
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: '當前商品處於休市時段。',
          action: 'Wait for the next trading session and retry.'
        },
        RISK_VOLUME_INVALID: {
          title: '下單手數不合法。',
          action: 'Adjust volume to match min/max/step requirements.'
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: '當前商品不支援該訂單類型。',
          action: 'Choose a supported order type and retry.'
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: '停損或停利距離當前價格過近。',
          action: 'Increase SL/TP distance and retry.'
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: '訂單處於凍結區，當前不可修改。',
          action: 'Wait until price moves away from freeze distance, then retry.'
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: '可用保證金不足，無法下單。',
          action: 'Reduce volume, close positions, or add funds.'
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: '已達到最大持倉數量限制。',
          action: 'Close existing positions or raise the limit.'
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: '已達到最大掛單數量限制。',
          action: 'Cancel existing pending orders or raise the limit.'
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: '風控規則暫不可用。',
          action: 'Retry later; contact support if the issue persists.'
        },
        unknown: {
          title: '交易請求被拒絕。',
          action: 'Please review order parameters and try again.'
        }
      }
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
      fetchOrderHistoryFailed: 'Failed to load order history'
    },
    riskConfig: {
      fields: {
        maxRiskPercent: '單筆最大風險',
        maxDailyLoss: '每日最大虧損',
        maxDrawdownPercent: '最大回撤限制',
        maxPositions: '最大持倉數量',
        maxLotSize: '最大手數',
        trailingStopEnabled: '移動止損',
        trailingStopPips: 'Trailing Stop (pips)'
      },
      confirm: {
        title: '確認保存風險配置',
        confirmText: '確認保存',
        description: '請確認以下風險配置：',
        info: 'After saving, all auto trading will follow the new risk limits.'
      }
    },
    strategyExecute: {
      confirm: {
        title: '確認執行交易',
        confirmText: '確認執行',
        warningTitle: '交易執行確認',
        warningDescription: '此操作將立即執行真實交易，請仔細核對交易參數。',
        strategyName: '策略名稱',
        symbol: '商品',
        action: '方向',
        buy: '買入',
        sell: '賣出',
        volume: '數量'
      }
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
        disableQuestion: 'Are you sure you want to disable auto trading?'
      }
    },
    pnl: '盈虧',
    profit: '盈虧',
    time: '時間',
    ordersCount: '{{count}} 筆訂單',
    markPrice: '標記價格',
    positionSide: '方向',
    positionSize: '數量',
    positionEntryPrice: '進場價格',
    positionMarkPrice: '標記價格',
    positionLeverage: '槓桿',
    positionUnrealizedPnL: '未實現盈虧',
    positionLong: '做多',
    positionShort: '做空',
    openPositionsTitle: '持倉',
    closePositionTitle: '平倉',
    recentTrades: '近期交易',
    selectSymbol: 'Select a symbol'
  },
  algo: {
    submitForm: {
      title: 'Launch Algo'
    },
    actions: {
      start: '啟動',
      cancel: 'Cancel'
    },
    fields: {
      algo: '演算法',
      symbol: '商品',
      side: '方向',
      volume: '數量',
      limitPrice: '限價',
      account: '账戶',
      timeRange: '時間範圍',
      urgency: '緊急度',
      sliceInterval: '切片間隔',
      participationRate: 'Participation Rate'
    },
    side: {
      buy: '買入',
      sell: '賣出'
    },
    info: {
      name: '名稱',
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
      title: '演算法面板',
      activeExecutions: '執行中',
      noActive: 'No active algo executions'
    },
    table: {
      executionId: '執行ID',
      algo: '演算法',
      symbol: '商品',
      side: '方向',
      volume: '數量',
      progress: '進度',
      state: '狀態',
      actions: 'Actions'
    }
  }
} as const;

export default trading;
