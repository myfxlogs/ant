// Auto-generated supplementary keys for admin
const AdminExtra = {
  "admin": {
    "aiGateway": {
      "errors": {
        "loadProviders": "プロバイダーの読み込みに失敗",
        "toggleFailed": "切り替え失敗",
        "loadModels": "モデルの読み込みに失敗"
      },
      "addProviderPending": "プロバイダー追加機能はバックエンド対応待ち",
      "title": "AIゲートウェイ管理",
      "description": "AIプロバイダー、モデル、価格を管理。ユーザーは利用可能なモデルから選択し、トークン単位でウォレットから課金。",
      "addProvider": "プロバイダー追加",
      "columns": {
        "baseUrl": "ベースURL",
        "apiKey": "APIキー"
      },
      "configured": "未設定",
      "editProvider": "プロバイダー追加",
      "providerId": "プロバイダーIDを入力",
      "providerIdPlaceholder": "deepseek / openai / qwen ...",
      "displayName": "表示名",
      "displayNamePlaceholder": "DeepSeek",
      "baseUrl": "ベースURLを入力",
      "apiKeyLabel": "APIキー、保存時に暗号化",
      "apiKeyEditPlaceholder": "空欄で変更なし",
      "editModel": "モデル追加",
      "modelName": "モデル名",
      "priceInput": "入力価格（$/1M）",
      "priceOutput": "出力価格（$/1M）",
      "addModel": "モデル追加",
      "confirmDeleteModel": "このモデルを削除？",
      "noModels": "モデルなし"
    },
    "account": {
      "errors": {
        "loadFailed": "アカウントの読み込みに失敗",
        "freezeFailed": "凍結失敗",
        "unfreezeFailed": "解凍失敗"
      },
      "frozen": "アカウントが凍結されました",
      "unfrozen": "アカウントが解凍されました",
      "columns": {
        "createdAt": "作成日時"
      },
      "confirmFreeze": "このアカウントを凍結？",
      "title": "アカウント管理",
      "searchPlaceholder": "アカウントを検索",
      "detail": "アカウント詳細",
      "auditLogs": "監査ログ"
    },
    "settings": {
      "saveSuccess": "保存成功",
      "saveFailed": "保存失敗",
      "deleteFailed": "削除失敗",
      "actionFailed": "操作失敗",
      "columns": {
        "key": "設定キー"
      },
      "confirmDelete": "削除確認？",
      "title": "Agent管理設定",
      "addSetting": "設定追加",
      "permissionRules": "権限ルール (permission.rule.N)",
      "permissionFormat": "形式：",
      "permissionExample": "例：",
      "permissionAddRule": "ルール追加：設定キーを作成",
      "addManagedSetting": "管理設定追加",
      "settingKey": "設定キー",
      "keyPlaceholder": "例：allowed_models, disable_live_trading, permission.rule.1",
      "valuePlaceholder": "例：claude-sonnet-5,deepseek-v4"
    },
    "autogen": {
      "approved": "タスクが承認・公開されました",
      "rejected": "タスクが拒否されました",
      "enqueued": "{{count}}件のタスクをキューに追加",
      "confirmApprove": "承認して公開？",
      "confirmReject": "このタスクを拒否？",
      "title": "AI戦略生成タスク",
      "allStatus": "全ステータス",
      "triggerBatch": "バッチ生成をトリガー",
      "symbols": "シンボル（カンマ区切り）",
      "timeframes": "タイムフレーム（カンマ区切り）",
      "strategyTypes": "戦略タイプ（カンマ区切り）"
    },
    "billing": {
      "columns": {
        "autoRenew": "自動更新",
        "periodStart": "期間開始",
        "periodEnd": "期間終了",
        "createdAt": "作成日時",
        "balanceBefore": "取引前残高",
        "balanceAfter": "取引後残高"
      },
      "title": "請求管理",
      "monthlyRevenue": "月次収益",
      "totalRevenue": "総収益",
      "activeSubs": "アクティブサブスク",
      "planRevenue": "プラン収益明細",
      "filterByPlan": "プランで絞り込み",
      "filterByStatus": "ステータスで絞り込み",
      "walletTransactions": "ウォレット取引",
      "filterByType": "タイプで絞り込み",
      "txPlatformFee": "プラットフォーム手数料"
    },
    "coupon": {
      "loadFailed": "クーポンの読み込みに失敗",
      "fillRequired": "必須項目を入力してください",
      "created": "クーポン作成済み",
      "createFailed": "クーポン作成失敗",
      "disabled": "クーポン無効化済み",
      "disableFailed": "無効化失敗",
      "colMinPurchase": "最低購入額",
      "create": "クーポン作成",
      "createTitle": "クーポン作成",
      "codePlaceholder": "クーポンコード（例 SUMMER20）",
      "valuePlaceholder": "割引値（例 20=20% or 50=¥50）",
      "minPurchasePlaceholder": "最低購入金額（0=制限なし）",
      "maxUsesPlaceholder": "最大使用回数（0=無制限）",
      "expiresPlaceholder": "有効期限（ISO 8601、空=期限なし）"
    },
    "dashboard": {
      "errors": {
        "loadFailed": "ダッシュボードデータの読み込みに失敗"
      },
      "title": "管理ダッシュボード",
      "totalUsers": "総ユーザー数",
      "activeUsers": "アクティブユーザー",
      "verifiedUsers": "認証済みユーザー",
      "mtAccounts": "MTアカウント",
      "onlineAccounts": "オンラインアカウント",
      "todayTrades": "本日取引",
      "todayProfit": "本日損益",
      "activeSubs": "アクティブサブスク",
      "monthlyRevenue": "月次収益",
      "totalRevenue": "総収益",
      "marketStrategies": "市場戦略",
      "marketSales": "市場販売",
      "marketRevenue": "市場収益",
      "recentLogs": "最近ログ"
    },
    "logs": {
      "modules": {
        "userManagement": "ユーザー管理",
        "accountManagement": "アカウント管理",
        "systemConfig": "システム設定"
      },
      "columns": {
        "actionType": "操作タイプ",
        "ip": "IPアドレス"
      },
      "errors": {
        "loadFailed": "ログの読み込みに失敗"
      },
      "title": "操作ログ",
      "filterModule": "モジュールで絞り込み",
      "filterAction": "操作で絞り込み"
    },
    "depositAddresses": {
      "importFailed": "インポート失敗",
      "user": "ユーザーID",
      "received": "受取USDT",
      "assignedAt": "割当日時",
      "importHint": "オフラインマシンでhdgenツールを使用してdeposit_addresses.binを生成し、ここにアップロード。",
      "all": "全ステータス",
      "import": "アドレスインポート",
      "availablePool": "プール利用可能",
      "total": "総アドレス数"
    },
    "deposit": {
      "table": {
        "user": "ユーザーID",
        "amount": "USDT金額",
        "txHash": "取引ハッシュ"
      },
      "title": "入金管理"
    },
    "analytics": {
      "platformRev": "プラットフォーム収益",
      "providerRev": "プロバイダー収益",
      "activeBuyers": "アクティブ購入者",
      "refundRate": "返金率",
      "newSubs": "新規サブスク",
      "totalStrategies": "総戦略数",
      "newStrategies": "新規戦略",
      "topByRevenue": "収益トップ戦略",
      "topBySubs": "サブスクトップ戦略",
      "topProvidersRev": "収益トッププロバイダー",
      "topProvidersStrat": "戦略数トッププロバイダー"
    },
    "marketplace": {
      "loadFailed": "戦略の読み込みに失敗",
      "featureSuccess": "戦略をおすすめに設定",
      "featureFailed": "おすすめ設定失敗",
      "unfeatureSuccess": "おすすめ解除",
      "unfeatureFailed": "おすすめ解除失敗",
      "unfeature": "おすすめ解除",
      "filterStatus": "全ステータス",
      "searchPlaceholder": "タイトルで検索...",
      "featureTitle": "おすすめ戦略",
      "featureDesc": "おすすめ表示優先度を設定。高いほど目立つ。"
    },
    "refund": {
      "loadFailed": "返金リクエストの読み込みに失敗",
      "approved": "返金承認・実行済み",
      "rejected": "返金リクエスト拒否",
      "processFailed": "返金処理失敗",
      "approve": "承認・実行",
      "filterStatus": "全ステータス",
      "approveTitle": "返金承認",
      "rejectTitle": "返金拒否",
      "reviewNotePlaceholder": "審査メモ（拒否時任意、承認時推奨）..."
    },
    "sidebar": {
      "shareManagement": "シェア分析"
    },
    "walletCalculator": {
      "title": "Token ↔ USD 計算機",
      "selectModel": "モデル選択（価格基準）",
      "usdAmount": "USD金額",
      "fillResult": "結果入力"
    },
    "wallet": {
      "errors": {
        "noUserSelected": "ユーザー未選択"
      },
      "messages": {
        "adjustSuccess": "残高調整成功",
        "adjustFailed": "調整失敗"
      },
      "columns": {
        "walletNumber": "ウォレット番号",
        "balanceAfter": "取引後残高"
      },
      "title": "ウォレット管理",
      "tabWallets": "ユーザーウォレット",
      "userList": "ユーザーリスト",
      "searchPlaceholder": "ウォレット/メール/ニックネーム検索",
      "noMatch": "ユーザーなし",
      "walletDetail": "ウォレット詳細",
      "adjustBalance": "残高調整",
      "tabDepositAddresses": "入金アドレス"
    },
    "config": {
      "apiKey": "APIキー"
    },
    "userManagement": {
      "form": {
        "accountNumber": "アカウント番号",
        "accountNumberInvalid": "5-6桁、先頭0不可、4と7不可"
      },
      "messages": {
        "loadUsersFailed": "ユーザーの読み込みに失敗"
      }
    }
  }
} as const;
export default AdminExtra;
