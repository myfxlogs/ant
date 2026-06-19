// Auto-generated from proto/ant/v1/i18n/trading_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "algo": {
    "actions": {
      "cancel": "取消",
      "start": "启动"
    },
    "dashboard": {
      "activeExecutions": "执行中",
      "noActive": "暂无活跃执行",
      "title": "算法面板"
    },
    "fields": {
      "account": "账户",
      "algo": "算法",
      "limitPrice": "限价",
      "participationRate": "参与率",
      "side": "方向",
      "sliceInterval": "切片间隔",
      "symbol": "品种",
      "timeRange": "时间范围",
      "urgency": "紧急度",
      "volume": "数量"
    },
    "info": {
      "description": "描述",
      "name": "名称"
    },
    "messages": {
      "started": "算法已启动"
    },
    "side": {
      "buy": "买入",
      "sell": "卖出"
    },
    "submitForm": {
      "title": "启动算法"
    },
    "table": {
      "actions": "操作",
      "algo": "算法",
      "executionId": "执行ID",
      "progress": "进度",
      "side": "方向",
      "state": "状态",
      "symbol": "品种",
      "volume": "数量"
    },
    "timePresets": {
      "EOD": "当日结束"
    },
    "twap": "TWAP (时间加权均价)",
    "vwap": "VWAP (成交量加权均价)",
    "pov": "POV (参与率算法)",
    "shortfall": "Shortfall (最小缺口)"
  },
  "trading": {
    "account": "账户",
    "autoTrade": {
      "confirm": {
        "disableConfirm": "無効化",
        "disableInfoDescription": "無効化すると自動取引は停止しますが、有効化済みの戦略は市場監視を継続する場合があります。",
        "disableInfoTitle": "自動取引を無効化",
        "disableQuestion": "确定要关闭自动交易？",
        "disableTitle": "自動取引を無効化",
        "enableBullet1": "戦略条件に合致した取引が自動で実行されます",
        "enableBullet2": "リスク設定が正しいことを確認してください",
        "enableBullet3": "まずはデモ口座でのテストを推奨します",
        "enableConfirm": "有効化",
        "enableQuestion": "自動取引を有効化しますか？",
        "enableRiskDescription": "自動取引を有効化すると、戦略に基づいて自動で取引が実行されます。リスクを理解したうえで実行してください。",
        "enableRiskTitle": "リスク注意",
        "enableTitle": "自動取引を有効化"
      }
    },
    "balance": "残高",
    "buy": "买入",
    "closePosition": "決済",
    "closePositionConfirm": "このポジションを決済しますか？",
    "closePositionTitle": "ポジションを決済",
    "equity": "純資産",
    "freeMargin": "有効証拠金",
    "limit": "指値",
    "margin": "証拠金",
    "marginLevel": "証拠金維持率",
    "markPrice": "マーク価格",
    "market": "成行",
    "messages": {
      "fetchOrderHistoryFailed": "加载订单历史失败",
      "fetchPendingOrdersFailed": "未決注文の取得に失敗しました",
      "fetchPositionsFailed": "保有ポジションの取得に失敗しました",
      "orderCloseFailed": "決済に失敗しました",
      "orderCloseSuccess": "決済しました",
      "orderModifyFailed": "注文の変更に失敗しました",
      "orderModifySuccess": "注文を変更しました",
      "orderSendFailed": "注文の送信に失敗しました",
      "orderSendSuccess": "注文を送信しました"
    },
    "noAccount": "口座が選択されていません",
    "noOrders": "まだ注文はありません",
    "noPositions": "オープンポジションはありません",
    "openPositionsTitle": "オープンポジション",
    "openTime": "エントリー時間",
    "orderHistory": "注文履歴",
    "ordersCount": "{{count}} 件の注文",
    "placeOrder": "注文する",
    "pnl": "損益",
    "positionEntryPrice": "エントリー価格",
    "positionLeverage": "レバレッジ",
    "positionLong": "LONG",
    "positionMarkPrice": "マーク価格",
    "positionShort": "SHORT",
    "positionSide": "方向",
    "positionSize": "数量",
    "positionUnrealizedPnL": "含み損益",
    "positions": "ポジション",
    "price": "価格",
    "profit": "損益",
    "recentTrades": "最近の取引",
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "检查账户状态和权限后重试。",
          "title": "この口座では取引が無効化されています。"
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "稍后重试；如问题持续请联系客服。",
          "title": "リスクルールが一時的に利用できません。"
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "减少手数、平仓或充值。",
          "title": "この注文に必要な余剰証拠金が不足しています。"
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "等待下一个交易时段后重试。",
          "title": "この銘柄の市場は休場中です。"
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "平掉现有持仓或提高上限。",
          "title": "最大保有ポジション数に達しています。"
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "取消现有挂单或提高上限。",
          "title": "最大未決注文数に達しています。"
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "等待价格离开冻结区域后重试。",
          "title": "凍結ゾーン内のため注文を変更できません。"
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "选择支持的订单类型后重试。",
          "title": "この銘柄ではこの注文タイプはサポートされていません。"
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "增加止损/止盈距离后重试。",
          "title": "損切りまたは利確が現在価格に近すぎます。"
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "切换到可交易品种或稍后重试。",
          "title": "この銘柄は現在取引できません。"
        },
        "RISK_VOLUME_INVALID": {
          "action": "调整手数以匹配最小/最大/步长要求。",
          "title": "注文数量が無効です。"
        },
        "unknown": {
          "action": "请检查订单参数后重试。",
          "title": "取引リクエストが拒否されました。"
        }
      }
    },
    "riskConfig": {
      "confirm": {
        "confirmText": "保存",
        "description": "以下のリスク設定を確認してください：",
        "info": "保存后，所有自动交易将遵循新的风险限额。",
        "title": "リスク設定の保存確認"
      },
      "fields": {
        "maxDailyLoss": "日次最大損失",
        "maxDrawdownPercent": "最大ドローダウン制限",
        "maxLotSize": "最大ロット",
        "maxPositions": "最大ポジション数",
        "maxRiskPercent": "1回あたり最大リスク",
        "trailingStopEnabled": "トレーリングストップ",
        "trailingStopPips": "移动止损 (点)"
      }
    },
    "selectSymbol": "选择品种",
    "sell": "卖出",
    "side": "方向",
    "stop": "逆指値",
    "stopLoss": "ストップロス",
    "strategyExecute": {
      "confirm": {
        "action": "方向",
        "buy": "买入",
        "confirmText": "実行",
        "sell": "卖出",
        "strategyName": "戦略名",
        "symbol": "品种",
        "title": "取引実行の確認",
        "volume": "数量",
        "warningDescription": "この操作は即時に実取引を実行します。パラメータをよく確認してください。",
        "warningTitle": "取引実行確認"
      }
    },
    "symbol": "品种",
    "takeProfit": "テイクプロフィット",
    "time": "時間",
    "title": "取引",
    "type": "タイプ",
    "volume": "数量"
  }
} as const;
export default Trading;
