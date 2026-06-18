const trading = {
  trading: {
    title: '取引',
    account: '账户',
    balance: '残高',
    equity: '純資産',
    margin: '証拠金',
    freeMargin: '有効証拠金',
    marginLevel: '証拠金維持率',
    noAccount: '口座が選択されていません',
    placeOrder: '注文する',
    symbol: '品种',
    type: 'タイプ',
    volume: '数量',
    price: '価格',
    stopLoss: 'ストップロス',
    takeProfit: 'テイクプロフィット',
    side: '方向',
    buy: '买入',
    sell: '卖出',
    market: '成行',
    limit: '指値',
    stop: '逆指値',
    positions: 'ポジション',
    noPositions: 'オープンポジションはありません',
    closePosition: '決済',
    closePositionConfirm: 'このポジションを決済しますか？',
    openTime: 'エントリー時間',
    orderHistory: '注文履歴',
    noOrders: 'まだ注文はありません',
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: 'この口座では取引が無効化されています。',
          action: 'Check account status and permissions, then try again.'
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: 'この銘柄は現在取引できません。',
          action: 'Switch to a tradable symbol or try later.'
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: 'この銘柄の市場は休場中です。',
          action: 'Wait for the next trading session and retry.'
        },
        RISK_VOLUME_INVALID: {
          title: '注文数量が無効です。',
          action: 'Adjust volume to match min/max/step requirements.'
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: 'この銘柄ではこの注文タイプはサポートされていません。',
          action: 'Choose a supported order type and retry.'
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: '損切りまたは利確が現在価格に近すぎます。',
          action: 'Increase SL/TP distance and retry.'
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: '凍結ゾーン内のため注文を変更できません。',
          action: 'Wait until price moves away from freeze distance, then retry.'
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: 'この注文に必要な余剰証拠金が不足しています。',
          action: 'Reduce volume, close positions, or add funds.'
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: '最大保有ポジション数に達しています。',
          action: 'Close existing positions or raise the limit.'
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: '最大未決注文数に達しています。',
          action: 'Cancel existing pending orders or raise the limit.'
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: 'リスクルールが一時的に利用できません。',
          action: 'Retry later; contact support if the issue persists.'
        },
        unknown: {
          title: '取引リクエストが拒否されました。',
          action: 'Please review order parameters and try again.'
        }
      }
    },
    messages: {
      fetchPositionsFailed: '保有ポジションの取得に失敗しました',
      orderSendSuccess: '注文を送信しました',
      orderSendFailed: '注文の送信に失敗しました',
      orderModifySuccess: '注文を変更しました',
      orderModifyFailed: '注文の変更に失敗しました',
      orderCloseSuccess: '決済しました',
      orderCloseFailed: '決済に失敗しました',
      fetchPendingOrdersFailed: '未決注文の取得に失敗しました',
      fetchOrderHistoryFailed: 'Failed to load order history'
    },
    riskConfig: {
      fields: {
        maxRiskPercent: '1回あたり最大リスク',
        maxDailyLoss: '日次最大損失',
        maxDrawdownPercent: '最大ドローダウン制限',
        maxPositions: '最大ポジション数',
        maxLotSize: '最大ロット',
        trailingStopEnabled: 'トレーリングストップ',
        trailingStopPips: 'Trailing Stop (pips)'
      },
      confirm: {
        title: 'リスク設定の保存確認',
        confirmText: '保存',
        description: '以下のリスク設定を確認してください：',
        info: 'After saving, all auto trading will follow the new risk limits.'
      }
    },
    strategyExecute: {
      confirm: {
        title: '取引実行の確認',
        confirmText: '実行',
        warningTitle: '取引実行確認',
        warningDescription: 'この操作は即時に実取引を実行します。パラメータをよく確認してください。',
        strategyName: '戦略名',
        symbol: '品种',
        action: '方向',
        buy: '买入',
        sell: '卖出',
        volume: '数量'
      }
    },
    autoTrade: {
      confirm: {
        enableTitle: '自動取引を有効化',
        disableTitle: '自動取引を無効化',
        enableConfirm: '有効化',
        disableConfirm: '無効化',
        enableRiskTitle: 'リスク注意',
        enableRiskDescription: '自動取引を有効化すると、戦略に基づいて自動で取引が実行されます。リスクを理解したうえで実行してください。',
        enableQuestion: '自動取引を有効化しますか？',
        enableBullet1: '戦略条件に合致した取引が自動で実行されます',
        enableBullet2: 'リスク設定が正しいことを確認してください',
        enableBullet3: 'まずはデモ口座でのテストを推奨します',
        disableInfoTitle: '自動取引を無効化',
        disableInfoDescription: '無効化すると自動取引は停止しますが、有効化済みの戦略は市場監視を継続する場合があります。',
        disableQuestion: 'Are you sure you want to disable auto trading?'
      }
    },
    pnl: '損益',
    profit: '損益',
    time: '時間',
    ordersCount: '{{count}} 件の注文',
    markPrice: 'マーク価格',
    positionSide: '方向',
    positionSize: '数量',
    positionEntryPrice: 'エントリー価格',
    positionMarkPrice: 'マーク価格',
    positionLeverage: 'レバレッジ',
    positionUnrealizedPnL: '含み損益',
    positionLong: 'LONG',
    positionShort: 'SHORT',
    openPositionsTitle: 'オープンポジション',
    closePositionTitle: 'ポジションを決済',
    recentTrades: '最近の取引',
    selectSymbol: 'Select a symbol'
  },
  algo: {
    submitForm: {
      title: 'Launch Algo'
    },
    actions: {
      start: '启动',
      cancel: 'Cancel'
    },
    fields: {
      algo: '算法',
      symbol: '品种',
      side: '方向',
      volume: '数量',
      limitPrice: '限价',
      account: '账户',
      timeRange: '时间范围',
      urgency: '紧急度',
      sliceInterval: '切片间隔',
      participationRate: 'Participation Rate'
    },
    side: {
      buy: '买入',
      sell: '卖出'
    },
    info: {
      name: '名称',
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
      title: '算法面板',
      activeExecutions: '执行中',
      noActive: 'No active algo executions'
    },
    table: {
      executionId: '执行ID',
      algo: '算法',
      symbol: '品种',
      side: '方向',
      volume: '数量',
      progress: '进度',
      state: '状态',
      actions: 'Actions'
    }
  }
} as const;

export default trading;
