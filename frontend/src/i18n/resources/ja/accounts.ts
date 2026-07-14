// Auto-generated from proto/ant/v1/i18n/accounts_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Accounts = {
  "accounts": {
    "analytics": {
      "monthlyAnalysis": {
        "bonus": {
          "chartHoldingTitle": "{{month}} 平均持仓时间",
          "chartPopularTitle": "{{month}} 货币热度",
          "chartRiskTitle": "Bonus: {{month}}のシンボル別プロフィットファクター。",
          "emptyCharts": "この月の取引なし",
          "legendBulls": "買い",
          "legendShortTerm": "売り",
          "popularityShare": "手数份额",
          "sliceOther": "その他"
        },
        "metrics": {
          "change": "変化",
          "lots": "ロット",
          "pips": "点",
          "profit": "損益"
        },
        "chartMainTitle": "月次リターン（{{metric}}）",
        "focusedValue": "{{period}} · {{metric}}：{{value}}",
        "title": "月次分析"
      },
      "monthlyDetail": {
        "fields": {
          "averageHours": "平均",
          "bestTrade": "最良取引",
          "maxHours": "最大",
          "medianHours": "中央値",
          "minHours": "最小",
          "netReturn": "純利益",
          "profitFactor": "PF",
          "totalTrades": "取引数",
          "winRate": "勝率",
          "worstTrade": "最悪取引"
        },
        "holdingTitle": "保有時間",
        "long": "買い",
        "metricsTitle": "月次指標",
        "popularityTitle": "通貨人気度",
        "riskRewardTitle": "報酬:リスク比率",
        "short": "売り",
        "symbolPnLTitle": "銘柄別損益"
      },
      "advancedTabs": {
        "daily": "日",
        "hourly": "時間足"
      },
      "chartPeriod": {
        "all": "全部",
        "day": "今日",
        "month": "今月",
        "week": "今週",
        "year": "今年"
      },
      "chartSeries": {
        "balance": "残高",
        "equity": "純資産",
        "profit": "損益",
        "tradeCount": "取引数"
      },
      "chartType": {
        "balance": "残高",
        "equity": "純資産",
        "profit": "損益"
      },
      "empty": {
        "dailyPnL": "日次損益データがありません",
        "equityCurve": "エクイティカーブのデータがありません",
        "hourly": "暂无时段分析数据",
        "monthlyProfit": "月次損益データがありません",
        "symbolDistribution": "銘柄分布データがありません"
      },
      "stats": {
        "avgDailyReturn": "平均日次リターン",
        "avgHolding": "平均保有時間",
        "avgLoss": "平均損失",
        "avgProfit": "平均利益",
        "calmar": "カルマーレシオ",
        "consecutiveWinsLosses": "連勝/連敗",
        "largestLoss": "最大損失",
        "largestWin": "最大利益",
        "maxDrawdown": "最大ドローダウン",
        "netDeposit": "净入金",
        "netProfit": "純利益",
        "profitFactor": "プロフィットファクター",
        "sharpe": "シャープレシオ",
        "sortino": "ソルティノレシオ",
        "totalDeposit": "総入金",
        "totalTrades": "取引回数",
        "totalWithdrawal": "総出金",
        "volatility": "ボラティリティ",
        "winRate": "勝率"
      },
      "timeDetail": {
        "balance": "残高",
        "lots": "ロット",
        "maxFloatingLossAmount": "最大含み損額",
        "maxFloatingLossRatio": "最大含み損比率",
        "maxFloatingProfitAmount": "最大含み益額",
        "maxFloatingProfitRatio": "最大浮动盈利比",
        "profitAmount": "損益額",
        "profitFactor": "プロフィットファクター",
        "trades": "取引数"
      },
      "advancedStatsTitle": "詳細統計",
      "dailyPnLTitle": "日次損益",
      "hourlyTitle": "時間帯分析",
      "monthlyProfitTitle": "月次損益",
      "symbolDistributionTitle": "銘柄分布"
    },
    "bind": {
      "actions": {
        "confirmBind": "連携を確定",
        "retryVerify": "重试",
        "search": "検索",
        "verifyAccount": "アカウントを確認"
      },
      "errorModal": {
        "title": "绑定失败"
      },
      "errors": {
        "brokerUnavailable": "サーバーエラーまたはパスワードが正しくありません",
        "connectionFailed": "ブローカーサーバーに接続できません。ネットワークを確認してください",
        "invalidCredentials": "口座が見つからないか、パスワードが無効です",
        "timeout": "连接超时，请稍后重试"
      },
      "fields": {
        "brokerName": "ブローカー名",
        "company": "会社名",
        "password": "パスワード",
        "platform": "プラットフォーム",
        "server": "サーバー",
        "tradingAccount": "取引口座"
      },
      "labels": {
        "serverCount": "{{count}} 台服务器"
      },
      "messages": {
        "bindFailed": "口座の連携に失敗しました",
        "bindSuccess": "口座を連携しました",
        "changeCredentials": "認証情報を変更",
        "enterBrokerName": "ブローカー名を入力してください",
        "enterPassword": "パスワードを入力してください",
        "enterTradingAccount": "取引口座を入力してください",
        "foundBrokers": "{{count}} 件のブローカーが見つかりました",
        "loginDigitsOnly": "交易账户只能包含数字",
        "noAccessHosts": "利用可能なアクセスホストがありません",
        "noBrokersFound": "一致するブローカーが見つかりません。名称を確認してください。",
        "searchFailed": "検索に失敗しました。しばらくしてから再試行してください。",
        "selectServer": "サーバーを選択してください",
        "verifyFailed": "アカウント確認に失敗しました"
      },
      "placeholders": {
        "brokerName": "ブローカー名を入力（例：XM、IC Markets）",
        "company": "会社を選択",
        "password": "输入密码",
        "server": "サーバーを選択",
        "tradingAccount": "取引口座を入力",
        "alias": "任意のカスタム名"
      },
      "step1": {
        "subtitle": "选择您的交易平台并搜索经纪商",
        "title": "プラットフォームとブローカーを選択"
      },
      "step2": {
        "subtitle": "输入您的交易账户和密码",
        "title": "口座情報を入力"
      },
      "step3": {
        "subtitle": "验证凭据并确认完成",
        "title": "連携内容を確認"
      },
      "summary": {
        "balance": "残高",
        "broker": "ブローカー",
        "currency": "货币",
        "equity": "純資産",
        "freeMargin": "余剰証拠金",
        "leverage": "レバレッジ",
        "margin": "証拠金",
        "password": "パスワード",
        "platform": "プラットフォーム",
        "server": "サーバー",
        "tradingAccount": "取引口座",
        "verified": "アカウント確認済み"
      },
      "passwordHint": "パスワードは HTTPS で送信され、バックエンドで Argon2id ハッシュとして保存されます（復元不可）。",
      "title": "MT 口座を連携"
    },
    "card": {
      "actions": {
        "details": "详情",
        "orders": "注文",
        "positions": "保有ポジション"
      },
      "deleteConfirm": {
        "content": "此操作不可撤销",
        "title": "この口座を削除しますか？"
      },
      "fields": {
        "balance": "残高",
        "broker": "ブローカー",
        "equity": "純資産",
        "server": "サーバー"
      },
      "status": {
        "connected": "接続済み",
        "connecting": "接続中",
        "disabled": "無効",
        "disconnected": "切断",
        "error": "错误"
      }
    },
    "detail": {
      "accountType": {
        "demo": "模拟",
        "real": "リアル"
      },
      "actions": {
        "deleteAccount": "アカウント削除",
        "deleteConfirm": "確認して削除",
        "deletePasswordHint": "確認のため、MT取引パスワードまたは読み取り専用パスワードを入力してください：",
        "deletePasswordPlaceholder": "MT取引/読み取り専用パスワード",
        "deletePasswordWrong": "取引パスワードが正しくありません。現在のMTパスワードを入力してください。",
        "deleteWarning": "この操作は元に戻せません。取引記録、分析データなど、すべてのアカウントデータが完全に削除されます。",
        "disableAccount": "口座を無効化",
        "enableAccount": "口座を有効化",
        "syncHistory": "同步历史"
      },
      "balanceRecord": {
        "deposit": "💰 入金",
        "depositIconText": "💰 入金",
        "withdraw": "💸 出金",
        "withdrawIconText": "💸 出金"
      },
      "cards": {
        "balance": "残高",
        "credit": "授信",
        "equity": "純資産",
        "floatingProfit": "含み損益",
        "marginFree": "余剰証拠金",
        "marginLevel": "証拠金維持率",
        "marginUsed": "使用証拠金"
      },
      "messages": {
        "fetchAccountFailed": "口座情報の取得に失敗しました。しばらくしてから再試行してください。",
        "syncHistoryFailed": "同步订单历史失败，请确保账户已连接到 MT 服务器。",
        "syncHistorySuccess": "注文履歴の同期に成功しました"
      },
      "mode": {
        "investor": "投資家モード",
        "trader": "交易员模式"
      },
      "orderTypes": {
        "buyLimit": "買い指値",
        "buyStop": "買い逆指値",
        "sellLimit": "売り指値",
        "sellStop": "卖出止损"
      },
      "status": {
        "connected": "接続済み",
        "connecting": "接続中",
        "disabled": "無効",
        "disconnected": "切断",
        "error": "错误"
      },
      "syncHistory": {
        "content": "過去1年分の注文履歴を MT サーバーから同期しますか？時間がかかる場合があります。",
        "ok": "同步",
        "title": "注文履歴を同期"
      },
      "connected": "接続済み",
      "lastConnected": "{{time}}",
      "leverage": "レバレッジ {{leverage}} 倍"
    },
    "disabled": {
      "confirmDelete": {
        "content": "此操作不可撤销",
        "title": "この口座を削除しますか？"
      },
      "mobile": {
        "balanceLabel": "残高: ",
        "equityLabel": "净值: "
      },
      "table": {
        "account": "口座",
        "actions": "操作",
        "balance": "残高",
        "broker": "ブローカー",
        "equity": "純資産",
        "type": "タイプ"
      },
      "title": "無効な口座"
    },
    "edit": {
      "fields": {
        "oldPassword": "当前密码",
        "password": "新しいパスワード",
        "server": "サーバー",
        "tradingAccount": "取引口座"
      },
      "messages": {
        "enterOldPassword": "現在のパスワードを入力してください",
        "enterPassword": "新しいパスワードを入力してください",
        "passwordSaved": "密码已保存",
        "passwordVerifyFailed": "パスワード変更に失敗しました"
      },
      "placeholders": {
        "newPassword": "新しいパスワードを入力",
        "oldPassword": "输入当前密码"
      },
      "title": "口座編集"
    },
    "report": {
      "periods": {
        "month": "今月",
        "quarter": "今四半期",
        "week": "今週",
        "year": "今年"
      },
      "sections": {
        "findings": "主な発見",
        "recommendations": "改善提案",
        "summary": "総評"
      },
      "aiAnalysis": "AI分析",
      "direction": "売買分析",
      "directionLong": "買い",
      "directionShort": "売り",
      "drawdownEvents": "ドローダウンイベント",
      "drawdownOverlay": "資産曲線 + ドローダウン",
      "generate": "レポート生成",
      "goToAISettings": "AI設定へ →",
      "recovered": "回復済み",
      "symbolPnL": "銘柄別損益",
      "title": "取引レポート",
      "titleShort": "レポート",
      "tradeDistribution": "損益分布",
      "winRateTrend": "月次勝率トレンド"
    },
    "tradeTabs": {
      "pagination": {
        "total": "共 {{total}} 条"
      },
      "table": {
        "closePrice": "決済価格",
        "closeTime": "平仓时间",
        "currentPrice": "現在値",
        "openPrice": "建値",
        "openTime": "建玉時間",
        "orderId": "注文ID",
        "pendingPrice": "指値/逆指値",
        "pendingTime": "注文時間",
        "profit": "損益",
        "side": "売買",
        "symbol": "銘柄",
        "type": "タイプ",
        "volume": "数量"
      },
      "emptyHistory": "注文履歴がありません",
      "emptyPositions": "保有ポジションがありません",
      "historyWithCount": "履歴（{{count}}）",
      "pendingWithCount": "未決注文（{{count}}）",
      "positionsWithCount": "保有ポジション（{{count}}）",
      "syncHistory": "履歴同期"
    },
    "empty": {
      "subtitle": "点击下方按钮绑定您的 MT4/MT5 交易账户",
      "title": "連携済み口座がありません"
    },
    "legend": {
      "connected": "接続済み",
      "connecting": "接続中",
      "disabled": "無効",
      "disconnectedOrError": "切断/エラー",
      "title": "凡例:"
    },
    "messages": {
      "connectFailed": "接続に失敗しました",
      "connectSuccess": "接続しました",
      "connectingMtServer": "MT サーバーに接続中",
      "createFailed": "口座の作成に失敗しました",
      "createdSuccess": "口座を作成しました",
      "deleteFailed": "削除に失敗しました",
      "deleted": "口座を削除しました",
      "disableFailed": "口座の無効化に失敗しました",
      "disabledSuccess": "口座を無効化しました",
      "disconnectFailed": "切断に失敗しました",
      "enableFailed": "启用账户失败",
      "enabledSuccess": "口座を有効化しました",
      "fetchAccountFailed": "口座情報の取得に失敗しました",
      "fetchListFailed": "口座一覧の取得に失敗しました"
    },
    "bindNew": "口座を連携",
    "subtitle": "MT4/MT5 口座を管理します",
    "title": "口座"
  }
} as const;
export default Accounts;
