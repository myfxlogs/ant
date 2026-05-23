const trading = {
  trading: {
    risk: {
      errors: {
        RISK_ACCOUNT_TRADE_DISABLED: {
          title: 'この口座では取引が無効化されています。',
          action: '口座状態と権限を確認してから再試行してください。',
        },
        RISK_SYMBOL_TRADE_DISABLED: {
          title: 'この銘柄は現在取引できません。',
          action: '取引可能な銘柄に切り替えるか、後で再試行してください。',
        },
        RISK_MARKET_SESSION_CLOSED: {
          title: 'この銘柄の市場は休場中です。',
          action: '次の取引時間まで待って再試行してください。',
        },
        RISK_VOLUME_INVALID: {
          title: '注文数量が無効です。',
          action: '最小値/最大値/ステップに合わせて数量を調整してください。',
        },
        RISK_ORDER_TYPE_UNSUPPORTED: {
          title: 'この銘柄ではこの注文タイプはサポートされていません。',
          action: 'サポートされている注文タイプを選んで再試行してください。',
        },
        RISK_STOP_DISTANCE_TOO_CLOSE: {
          title: '損切りまたは利確が現在価格に近すぎます。',
          action: 'SL/TP の距離を広げて再試行してください。',
        },
        RISK_ORDER_FROZEN_ZONE: {
          title: '凍結ゾーン内のため注文を変更できません。',
          action: '価格が凍結距離から離れてから再試行してください。',
        },
        RISK_MARGIN_INSUFFICIENT: {
          title: 'この注文に必要な余剰証拠金が不足しています。',
          action: '数量を減らす、ポジションを決済する、または資金を追加してください。',
        },
        RISK_MAX_OPEN_POSITIONS_EXCEEDED: {
          title: '最大保有ポジション数に達しています。',
          action: '既存ポジションを決済するか上限を引き上げてください。',
        },
        RISK_MAX_PENDING_ORDERS_EXCEEDED: {
          title: '最大未決注文数に達しています。',
          action: '既存の未決注文を取り消すか上限を引き上げてください。',
        },
        RISK_INTERNAL_RULE_UNAVAILABLE: {
          title: 'リスクルールが一時的に利用できません。',
          action: 'しばらくして再試行し、解消しない場合はサポートへ連絡してください。',
        },
        unknown: {
          title: '取引リクエストが拒否されました。',
          action: '注文パラメータを確認して再試行してください。',
        },
      },
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
      fetchOrderHistoryFailed: '注文履歴の取得に失敗しました',
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
        disableQuestion: '自動取引を無効化しますか？',
      },
    },
    riskConfig: {
      fields: {
        maxRiskPercent: '1回あたり最大リスク',
        maxDailyLoss: '日次最大損失',
        maxDrawdownPercent: '最大ドローダウン制限',
        maxPositions: '最大ポジション数',
        maxLotSize: '最大ロット',
        trailingStopEnabled: 'トレーリングストップ',
        trailingStopPips: 'トレーリング幅（ピップ）',
      },
      confirm: {
        title: 'リスク設定の保存確認',
        confirmText: '保存',
        description: '以下のリスク設定を確認してください：',
        info: '反映後、自動取引は新しいリスク制限に従います。',
      },
    },
    strategyExecute: {
      confirm: {
        title: '取引実行の確認',
        confirmText: '実行',
        warningTitle: '取引実行確認',
        warningDescription: 'この操作は即時に実取引を実行します。パラメータをよく確認してください。',
        strategyName: '戦略名',
        symbol: '銘柄',
        action: '売買',
        buy: '買い',
        sell: '売り',
        volume: '数量',
      },
    },
    chatBox: {
      emptyDescription: 'AI アシスタントと会話を開始',
      thinking: '考え中...',
      truncated: '内容が長すぎるため切り詰めました',
      expandAll: 'すべて展開',
      collapse: '折りたたむ',
    },
    conversation: {
      defaultTitle: '新しい会話',
    },
    reports: {
      tradeAnalysis: {
        title: 'AI 取引分析レポート',
        riskAssessmentPrefix: 'リスク評価:',
      },
    },
    signalCard: {
      status: {
        pending: '確認待ち',
        confirmed: '確認済み',
        executed: '実行済み',
        cancelled: 'キャンセル済み',
      },
      labels: {
        price: '価格',
        volume: '数量',
        confidence: '信頼度',
        stopLoss: '損切り',
        takeProfit: '利確',
        analysisReason: '分析理由',
      },
      actions: {
        confirm: '確認',
        cancel: 'キャンセル',
        executeTrade: '取引を実行',
      },
      confirmCancel: {
        title: 'このシグナルをキャンセルしますか？',
      },
      confirmExecute: {
        title: 'この取引シグナルを実行しますか？',
        description: '直ちに取引注文を発注します',
      },
    },
  },
} as const;

export default trading;
