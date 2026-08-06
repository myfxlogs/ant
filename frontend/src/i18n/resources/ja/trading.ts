// Auto-generated from proto/ant/v1/i18n/trading_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Trading = {
  "trading": {
    "risk": {
      "errors": {
        "RISK_ACCOUNT_TRADE_DISABLED": {
          "action": "口座の状態と権限を確認してから再試行してください。",
          "title": "この口座では取引が無効化されています。"
        },
        "RISK_INTERNAL_RULE_UNAVAILABLE": {
          "action": "後で再試行。問題が続く場合はサポートに連絡してください。",
          "title": "リスクルールが一時的に利用できません。"
        },
        "RISK_MARGIN_INSUFFICIENT": {
          "action": "数量を減らすか、ポジションを決済するか、資金を追加してください。",
          "title": "この注文に必要な余剰証拠金が不足しています。"
        },
        "RISK_MARKET_SESSION_CLOSED": {
          "action": "次の取引セッションを待って再試行してください。",
          "title": "この銘柄の市場は休場中です。"
        },
        "RISK_MAX_OPEN_POSITIONS_EXCEEDED": {
          "action": "既存のポジションを決済するか、上限を引き上げてください。",
          "title": "最大保有ポジション数に達しています。"
        },
        "RISK_MAX_PENDING_ORDERS_EXCEEDED": {
          "action": "既存の未決注文を取消すか、上限を引き上げてください。",
          "title": "最大未決注文数に達しています。"
        },
        "RISK_ORDER_FROZEN_ZONE": {
          "action": "価格が凍結距離から離れるまで待ってから再試行してください。",
          "title": "凍結ゾーン内のため注文を変更できません。"
        },
        "RISK_ORDER_TYPE_UNSUPPORTED": {
          "action": "サポートされている注文タイプを選択して再試行してください。",
          "title": "この銘柄ではこの注文タイプはサポートされていません。"
        },
        "RISK_STOP_DISTANCE_TOO_CLOSE": {
          "action": "SL/TPの距離を広げて再試行してください。",
          "title": "損切りまたは利確が現在価格に近すぎます。"
        },
        "RISK_SYMBOL_TRADE_DISABLED": {
          "action": "取引可能な銘柄に切り替えるか、後で試してください。",
          "title": "この銘柄は現在取引できません。"
        },
        "RISK_VOLUME_INVALID": {
          "action": "最小/最大/ステップ要件に合わせて数量を調整してください。",
          "title": "注文数量が無効です。"
        },
        "unknown": {
          "action": "注文パラメータを確認して再試行してください。",
          "title": "取引リクエストが拒否されました。"
        }
      }
    },
    "autoTrade": {
      "confirm": {
        "disableConfirm": "無効化",
        "disableInfoDescription": "無効化すると自動取引は停止しますが、有効化済みの戦略は市場監視を継続する場合があります。",
        "disableInfoTitle": "自動取引を無効化",
        "disableQuestion": "自動取引を無効化しますか？",
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
    "riskConfig": {
      "confirm": {
        "confirmText": "保存",
        "description": "以下のリスク設定を確認してください：",
        "info": "保存後、すべての自動取引は新しいリスク制限に従います。",
        "title": "リスク設定の保存確認"
      },
      "fields": {
        "maxDailyLoss": "日次最大損失",
        "maxDrawdownPercent": "最大ドローダウン制限",
        "maxLotSize": "最大ロット",
        "maxPositions": "最大ポジション数",
        "maxRiskPercent": "1回あたり最大リスク",
        "trailingStopEnabled": "トレーリングストップ",
        "trailingStopPips": "トレーリングストップ (pips)"
      }
    },
    "strategyExecute": {
      "confirm": {
        "action": "方向",
        "buy": "買い",
        "confirmText": "実行",
        "sell": "売り",
        "strategyName": "戦略名",
        "symbol": "銘柄",
        "title": "取引実行の確認",
        "volume": "数量",
        "warningDescription": "この操作は即時に実取引を実行します。パラメータをよく確認してください。",
        "warningTitle": "取引実行確認"
      }
    },
    "messages": {
      "fetchOrderHistoryFailed": "注文履歴の取得に失敗しました",
      "fetchPendingOrdersFailed": "未決注文の取得に失敗しました",
      "fetchPositionsFailed": "保有ポジションの取得に失敗しました",
      "orderCloseFailed": "決済に失敗しました",
      "orderCloseSuccess": "決済しました",
      "orderModifyFailed": "注文の変更に失敗しました",
      "orderModifySuccess": "注文を変更しました",
      "orderSendFailed": "注文の送信に失敗しました",
      "orderSendSuccess": "注文を送信しました"
    },
    "account": "口座",
    "balance": "残高",
    "buy": "買い",
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
    "noAccount": "口座が選択されていません",
    "noOrders": "まだ注文はありません",
    "noPositions": "保有ポジションなし",
    "openPositionsTitle": "保有ポジション",
    "openTime": "エントリー時間",
    "orderHistory": "注文履歴",
    "ordersCount": "{{count}} 件の注文",
    "placeOrder": "注文する",
    "pnl": "損益",
    "positionEntryPrice": "エントリー価格",
    "positionLeverage": "レバレッジ",
    "positionLong": "ロング",
    "positionMarkPrice": "マーク価格",
    "positionShort": "ショート",
    "positionSide": "方向",
    "positionSize": "数量",
    "positionUnrealizedPnL": "含み損益",
    "positions": "ポジション",
    "price": "価格",
    "profit": "損益",
    "recentTrades": "最近の取引",
    "selectSymbol": "銘柄を選択",
    "sell": "売り",
    "side": "方向",
    "stop": "逆指値",
    "stopLoss": "損切り",
    "symbol": "銘柄",
    "takeProfit": "利確",
    "time": "時間",
    "title": "取引",
    "type": "タイプ",
    "volume": "数量",
    "platform": "プラットフォーム",
    "broker": "ブローカー",
    "server": "サーバー",
    "permission": "権限",
    "investor": "インベスター",
    "master": "マスター",
    "leverage": "レバレッジ"
  },
  "algo": {
    "actions": {
      "cancel": "取消",
      "start": "開始"
    },
    "dashboard": {
      "activeExecutions": "実行中",
      "noActive": "アクティブな実行はありません",
      "title": "アルゴダッシュボード"
    },
    "fields": {
      "account": "口座",
      "algo": "アルゴリズム",
      "limitPrice": "指値",
      "participationRate": "参加率",
      "side": "方向",
      "sliceInterval": "スライス間隔",
      "symbol": "銘柄",
      "timeRange": "期間",
      "urgency": "緊急度",
      "volume": "数量"
    },
    "info": {
      "description": "説明",
      "name": "名前"
    },
    "messages": {
      "started": "アルゴリズム開始"
    },
    "side": {
      "buy": "買い",
      "sell": "売り"
    },
    "submitForm": {
      "title": "アルゴリズム起動"
    },
    "table": {
      "actions": "操作",
      "algo": "アルゴリズム",
      "executionId": "実行ID",
      "progress": "進捗",
      "side": "方向",
      "state": "状態",
      "symbol": "銘柄",
      "volume": "数量"
    },
    "timePresets": {
      "EOD": "当日終了"
    },
    "twap": {
      "name": "TWAP (時間加重平均)",
      "description": "時間加重平均価格 — 大口注文を小さく分割し、時間軸で均等に執行します。"
    },
    "vwap": {
      "name": "VWAP (出来高加重平均)",
      "description": "出来高加重平均価格 — 過去の出来高分布に比例して注文を執行します。"
    },
    "pov": {
      "name": "POV (参加率)",
      "description": "参加率アルゴリズム — 市場出来高の一定割合で参加します。"
    },
    "shortfall": {
      "name": "Shortfall (最小ショートフォール)",
      "description": "インプリメンテーションショートフォール — 決定価格と執行価格の差を最小化します。"
    },
    "label": {
      "twap": "TWAP (時間加重平均)",
      "vwap": "VWAP (出来高加重平均)",
      "pov": "POV (参加率アルゴリズム)",
      "shortfall": "Shortfall (最小ショートフォール)"
    }
  }
} as const;
export default Trading;
