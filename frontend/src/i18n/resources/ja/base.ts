// Auto-generated from proto/ant/v1/i18n/base_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const Base = {
  "admin": {
    "dashboard": {
      "logs": {
        "moduleMap": {
          "accountManagement": "アカウント管理",
          "systemConfig": "システム設定",
          "trading": "取引",
          "userManagement": "ユーザー管理"
        },
        "actionType": "アクション",
        "failed": "失敗",
        "module": "モジュール",
        "status": "ステータス",
        "success": "成功",
        "target": "対象",
        "time": "時間"
      },
      "riskMetrics": {
        "orderCloseFailed": "平仓失败",
        "orderCloseSuccess": "決済成功",
        "orderSendFailed": "注文失敗",
        "orderSendSuccess": "注文成功",
        "riskValidateError": "エラー",
        "riskValidatePass": "通過",
        "riskValidateReject": "拒否",
        "riskValidateTotal": "検証総数",
        "title": "リスク検証指標"
      },
      "riskWindow": {
        "noData": "暂无窗口指标数据",
        "noRejectData": "この期間に拒否はありません",
        "orderCloseFailed": "決済失敗",
        "orderCloseSuccess": "決済OK",
        "orderSendFailed": "注文失敗",
        "orderSendSuccess": "注文OK",
        "rejectCount": "拒否数",
        "rejectRiskCodesHeader": "リスクコード",
        "title": "リスク管理ウィンドウ",
        "validateError": "エラー",
        "validatePass": "通過",
        "validateReject": "拒否",
        "validateTotal": "合計"
      },
      "activeUsers": "アクティブユーザー",
      "loadFailed": "ダッシュボードデータの読み込みに失敗しました",
      "mtAccounts": "MTアカウント",
      "onlineAccounts": "オンラインアカウント",
      "recentLogs": "最近のログ",
      "title": "管理ダッシュボード",
      "todayProfit": "本日の損益",
      "todayTrades": "本日の取引",
      "totalUsers": "総ユーザー数"
    },
    "userManagement": {
      "drawer": {
        "labels": {
          "createdAt": "作成日時",
          "email": "メール",
          "id": "ID",
          "lastLogin": "最終ログイン",
          "mtAccountCount": "MTアカウント",
          "nickname": "ニックネーム",
          "role": "役割",
          "status": "ステータス"
        },
        "title": "ユーザー詳細"
      },
      "form": {
        "placeholders": {
          "email": "メールを入力",
          "nickname": "ニックネームを入力",
          "password": "输入密码"
        },
        "accountNumber": "口座番号",
        "accountNumberInvalid": "5-6桁、先頭ゼロなし、4と7は不可",
        "email": "メール",
        "nickname": "ニックネーム",
        "password": "パスワード",
        "role": "役割",
        "status": "ステータス"
      },
      "passwordForm": {
        "placeholders": {
          "confirmPassword": "再次输入新密码",
          "newPassword": "新しいパスワードを入力"
        },
        "validation": {
          "confirmPasswordRequired": "パスワード確認が必要です",
          "newPasswordRequired": "新しいパスワードが必要です",
          "passwordMin8": "パスワードは8文字以上必要です",
          "passwordMismatch": "パスワードが一致しません",
          "passwordMustContainLettersAndNumbers": "密码必须包含字母和数字"
        },
        "confirmPassword": "パスワード確認",
        "newPassword": "新しいパスワード",
        "submit": "パスワード更新"
      },
      "actions": {
        "changePassword": "修改密码",
        "details": "詳細",
        "disable": "無効化",
        "enable": "有効化"
      },
      "deleteConfirm": {
        "batchDeleteConfirm": "{{count}}人のユーザーを削除しますか？この操作は元に戻せません。",
        "batchDeletePartial": "{{deleted}}人削除、{{failed}}人失敗",
        "batchDeleteSuccess": "{{count}}人のユーザーを削除しました",
        "title": "このユーザーを削除しますか？この操作は元に戻せません。"
      },
      "filters": {
        "rolePlaceholder": "役割でフィルター",
        "searchPlaceholder": "メールまたは名前で検索",
        "statusPlaceholder": "按状态筛选"
      },
      "messages": {
        "newPasswordIs": "新密码为: {{password}}",
        "passwordUpdateFailed": "パスワード更新に失敗しました",
        "passwordUpdatedSuccess": "パスワードを更新しました",
        "userCreateFailed": "ユーザー作成に失敗しました",
        "userCreatedSuccess": "ユーザーを作成しました",
        "userDeleteFailed": "ユーザー削除に失敗しました",
        "userDeletedSuccess": "ユーザーを削除しました",
        "userDisabled": "ユーザーを無効化しました",
        "userEnabled": "ユーザーを有効化しました",
        "userUpdateFailed": "ユーザー更新に失敗しました",
        "userUpdatedSuccess": "ユーザーを更新しました"
      },
      "modals": {
        "createTitle": "ユーザー作成",
        "editTitle": "ユーザー編集",
        "passwordTitle": "修改密码"
      },
      "pagination": {
        "total": "共 {{total}} 位用户"
      },
      "roles": {
        "audit": "审计",
        "customerService": "カスタマーサポート",
        "operation": "運用",
        "superAdmin": "スーパー管理者",
        "user": "ユーザー"
      },
      "status": {
        "active": "アクティブ",
        "suspended": "已停用"
      },
      "table": {
        "actions": "操作",
        "createdAt": "作成日時",
        "email": "メール",
        "id": "ID",
        "mtAccountCount": "MTアカウント",
        "nickname": "ニックネーム",
        "role": "役割",
        "status": "ステータス"
      },
      "addUser": "ユーザー追加",
      "title": "ユーザー管理"
    },
    "config": {
      "messages": {
        "disabled": "已禁用",
        "enabled": "已启用",
        "loadFailed": "加载配置失败",
        "operationFailed": "操作失败",
        "updateFailed": "更新配置失败",
        "updated": "配置已更新"
      },
      "placeholders": {
        "apiKey": "输入API Key",
        "baseUrl": "输入Base URL",
        "configValue": "输入配置值",
        "description": "输入描述",
        "json": "输入JSON",
        "model": "输入模型名称"
      },
      "providerOptions": {
        "custom": "自定义 / OpenAI 兼容",
        "deepseek": "DeepSeek",
        "zhipu": "智谱AI"
      },
      "validation": {
        "apiKeyRequired": "API Key不能为空",
        "greenMaxFailedRunsNonNegative": "绿色最大失败次数需≥0",
        "greenSuccessRateRange": "绿色成功率需在0-100之间",
        "jsonEmpty": "JSON不能为空",
        "jsonInvalid": "JSON格式无效",
        "minSampleSizeNonNegative": "最小样本量需≥0",
        "modelRequired": "模型名称不能为空",
        "yellowNotGreaterThanGreen": "黄色阈值不能超过绿色阈值",
        "yellowSuccessRateRange": "黄色成功率需在0-100之间"
      },
      "aiProviderCatalog": "AI提供商目录",
      "baseUrlLabel": "Base URL",
      "configItem": "配置项",
      "description": "説明",
      "econAIConfig": "经济日历AI配置",
      "editConfig": "编辑配置: {{key}}",
      "enableToggle": "有効化",
      "fillTemplate": "填充模板",
      "formatJson": "格式化JSON",
      "maxAccountsPerUser": "每用户最大账户数",
      "modelName": "模型名称",
      "off": "关",
      "on": "开",
      "provider": "提供商",
      "status": "ステータス",
      "strategyHealthConfig": "策略健康度配置",
      "thresholdDesc": "阈值描述",
      "thresholdInfo": "阈值说明",
      "title": "系统配置",
      "toggle": "切换",
      "updatedAt": "更新时间",
      "value": "值"
    },
    "jurisdiction": {
      "messages": {
        "countryAddFailed": "国の追加に失敗しました",
        "countryAdded": "国を追加しました",
        "countryRemoveFailed": "国の削除に失敗しました",
        "countryRemoved": "国を削除しました",
        "kycUpdateFailed": "KYCステータス更新に失敗しました",
        "kycUpdated": "KYCステータスを更新しました",
        "overrideUpdateFailed": "更新制裁豁免失败",
        "overrideUpdated": "上書き設定を更新しました"
      },
      "actions": "操作",
      "addCountry": "国を追加",
      "addSanctionedCountry": "制裁国を追加",
      "addedBy": "追加者",
      "confirmGrantOverride": "このユーザーに上書き許可を付与しますか？",
      "confirmRevokeOverride": "このユーザーの上書き許可を取り消しますか？",
      "country": "国",
      "countryCode": "国コード",
      "countryLabel": "国",
      "disclaimer": "免責事項",
      "emptyKYC": "KYCレコードがありません",
      "emptySanctions": "制裁国はありません",
      "filterByKYCStatus": "KYCステータスでフィルター",
      "grantOverride": "上書き許可",
      "kycStatus": "KYCステータス",
      "kycStatusTab": "ユーザーKYCステータス",
      "override": "上書き",
      "overrideWarning": "此用户来自受制裁国家，授予豁免将允许交易。",
      "pending": "保留中",
      "questionnaire": "アンケート",
      "rejected": "拒否",
      "revokeOverride": "上書き取消",
      "sanctioned": "制裁済み",
      "sanctionedCountries": "制裁対象国",
      "sanctionedCountriesTab": "制裁対象国",
      "setKYC": "KYC設定",
      "setKYCStatus": "KYCステータス設定",
      "title": "管轄権管理",
      "unverified": "未確認",
      "userEmail": "メール",
      "userKYCStatus": "ユーザーKYCステータス",
      "verified": "確認済み"
    },
    "header": {
      "admin": "管理",
      "adminMode": "管理员模式",
      "adminPanel": "管理后台",
      "backToUser": "返回用户端",
      "logout": "ログアウト"
    },
    "sidebar": {
      "accountManagement": "アカウント管理",
      "dashboard": "ダッシュボード",
      "jurisdiction": "管轄権管理",
      "operationLogs": "操作日志",
      "shareManagement": "分享分析",
      "systemConfig": "システム設定",
      "tradingMonitor": "取引監視",
      "userManagement": "ユーザー管理",
      "walletManagement": "钱包管理"
    },
    "trading": {
      "accounts": "アカウント",
      "activeUsers": "アクティブユーザー",
      "byPlatform": "プラットフォーム別",
      "closedOrders": "決済済み",
      "connectedAccounts": "接続済み",
      "loadFailed": "加载交易统计失败",
      "netProfit": "純利益",
      "orders": "注文",
      "pendingOrders": "挂单",
      "platform": "プラットフォーム",
      "profitStats": "損益統計",
      "title": "取引監視",
      "totalAccounts": "総アカウント数",
      "totalLoss": "総損失",
      "totalOrders": "総注文数",
      "totalProfit": "総利益",
      "totalUsers": "総ユーザー数",
      "totalVolume": "総取引量",
      "volume": "数量"
    },
    "wallet": {
      "accountNumber": "口座番号",
      "add": "追加",
      "adjustBalance": "残高調整",
      "adjustFailed": "調整に失敗しました",
      "adjustSuccess": "残高を調整しました",
      "deduct": "控除",
      "noUsers": "ユーザーが見つかりません",
      "reason": "調整理由...",
      "searchPlaceholder": "メールまたは口座番号で検索...",
      "title": "ウォレット管理",
      "walletFor": "ウォレット -"
    }
  },
  "autoTrading": {
    "logs": {
      "columns": {
        "action": "アクション",
        "price": "価格",
        "profit": "損益",
        "symbol": "銘柄",
        "ticket": "单号",
        "time": "時間",
        "volume": "数量"
      },
      "empty": "取引ログはまだありません",
      "title": "最近の取引ログ"
    },
    "messages": {
      "loadFailed": "自動取引データの読み込みに失敗しました",
      "toggleFailed": "切换自动交易失败"
    },
    "settings": {
      "maxDailyLoss": "最大日次損失",
      "maxDailyLossHint": "日次損失がこれを超えた場合、自動で取引を無効化",
      "maxDrawdownPercent": "最大ドローダウン%",
      "maxDrawdownPercentHint": "ドローダウンがこれを超えた場合、自動で取引を無効化",
      "maxLotSize": "最大ロットサイズ",
      "maxLotSizeHint": "1取引あたりの最大ボリューム（ロット）",
      "maxPositions": "最大ポジション数",
      "maxPositionsHint": "最大同時オープンポジション数",
      "maxRiskPercent": "最大リスク%",
      "maxRiskPercentHint": "1取引あたりのリスク許容額（残高の％）",
      "saveFailed": "保存设置失败",
      "saveSuccess": "設定を保存しました",
      "title": "グローバルリスク設定"
    },
    "status": {
      "activeStrategies": "アクティブ戦略",
      "disabled": "自動取引が無効です",
      "enabled": "自動取引が有効です",
      "todayExecutions": "Today's Executions",
      "todayProfit": "Today's Profit"
    },
    "title": "自動取引"
  },
  "notifications": {
    "stream": {
      "autoTrading": {
        "fallback": "自动交易事件触发",
        "title": "自動取引"
      },
      "riskAlert": {
        "fallback": "警报类型: {{alertType}}",
        "title": "リスクアラート"
      },
      "strategyExecution": {
        "completed": "{{symbol}} {{action}} が完了しました",
        "failed": "执行失败: {{error}}",
        "title": "戦略実行"
      },
      "strategySignal": {
        "message": "{{symbol}} triggered {{signalType}}",
        "title": "戦略シグナル"
      }
    },
    "actions": {
      "clearAll": "クリア",
      "clearAllConfirm": "すべての通知を削除しますか？",
      "markAllAsRead": "すべて既読"
    },
    "tabs": {
      "all": "すべて ({{count}})",
      "unread": "未读 ({{count}})"
    },
    "types": {
      "risk_alert": "リスクアラート",
      "signal": "シグナル",
      "strategy_execution": "戦略",
      "system": "系统",
      "trade": "取引"
    },
    "all": "すべて",
    "clearAll": "クリア",
    "confirmClearAll": "すべての通知を削除しますか？",
    "empty": "通知はありません",
    "markAllRead": "すべて既読",
    "title": "通知",
    "unread": "未読"
  },
  "auth": {
    "fields": {
      "confirmPassword": "确认密码",
      "email": "メール",
      "password": "パスワード"
    },
    "forgotPassword": {
      "backToLogin": "返回登录",
      "hint": "管理者またはサポートに連絡してパスワードをリセットしてください。",
      "title": "パスワードリセット"
    },
    "login": {
      "forgotPassword": "パスワードをお忘れですか？",
      "login": "ログイン",
      "noAccount": "没有账户？",
      "registerNow": "新規登録",
      "rememberMe": "ログイン状態を保持",
      "signingIn": "ログイン中...",
      "subtitle": "本サービスはテストであり責任を負いません"
    },
    "messages": {
      "fetchMeFailed": "加载用户信息失败",
      "loginFailed": "ログインに失敗しました。メールアドレスとパスワードを確認してください。",
      "loginSuccess": "ログインしました",
      "logoutSuccess": "ログアウトしました",
      "registerFailed": "登録に失敗しました。しばらくしてから再試行してください。",
      "registerSuccess": "登録が完了しました。ログインしてください。"
    },
    "register": {
      "haveAccount": "すでにアカウントをお持ちですか？",
      "loginNow": "ログイン",
      "register": "登録",
      "signingUp": "登録中...",
      "subtitle": "新規アカウント作成"
    },
    "validation": {
      "confirmPasswordRequired": "パスワードを確認してください",
      "emailInvalid": "有効なメールアドレスを入力してください",
      "emailRequired": "メールアドレスを入力してください",
      "passwordMin8": "パスワードは8文字以上必要です",
      "passwordMismatch": "パスワードが一致しません",
      "passwordRequired": "パスワードを入力してください"
    }
  },
  "common": {
    "months": {
      "jan": "1月",
      "jul": "7月"
    },
    "time": {
      "day": "{{n}}天",
      "hour": "{{n}}时",
      "lessThanMinute": "<1分钟",
      "minute": "{{n}}分"
    },
    "active": "アクティブ",
    "back": "戻る",
    "cancel": "キャンセル",
    "clear": "清除",
    "close": "決済",
    "comingSoon": "即将上线",
    "confirm": "確定",
    "copied": "コピーしました",
    "copy": "コピー",
    "copyFailed": "コピーに失敗しました",
    "create": "新規",
    "created": "作成しました",
    "currentPosition": "📊 現在のポジション",
    "delete": "削除",
    "deleteFailed": "削除に失敗しました",
    "deleteSelected": "選択した{{count}}件を削除",
    "deleted": "削除しました",
    "disable": "無効化",
    "disabled": "已禁用",
    "edit": "編集",
    "enable": "有効化",
    "enabled": "有効化しました",
    "error": "エラー",
    "gotIt": "了解",
    "hideDetails": "詳細を隠す",
    "inactive": "停用",
    "indicatorSettings": "{{name}} 設定",
    "lineColor": "ライン色",
    "loading": "読み込み中...",
    "loadingFailed": "読み込みに失敗しました",
    "next": "次へ",
    "no": "否",
    "noData": "データがありません",
    "noOpenPositionsForSymbol": "{{symbol}} のポジションはありません",
    "none": "なし",
    "ok": "OK",
    "operationFailed": "操作失败",
    "pageError": "ページエラー",
    "pageUnderDevelopment": "此页面开发中",
    "pleaseWait": "しばらくお待ちください...",
    "previous": "戻る",
    "refresh": "更新",
    "remove": "移除",
    "required": "必須",
    "retry": "リトライ",
    "readOnly": "読み取り専用",
    "save": "保存",
    "saveFailed": "保存に失敗しました",
    "saveSuccess": "保存成功",
    "saved": "保存済み",
    "searching": "検索中...",
    "selectSymbolToViewChart": "銘柄を選択してチャートを表示",
    "send": "送信",
    "showDetails": "詳細を表示",
    "totalItems": "共 {{count}} 项",
    "translate": "翻訳",
    "unexpectedError": "予期しないエラーが発生しました",
    "unknown": "未知",
    "unsaved": "未保存",
    "updated": "更新しました",
    "viewOriginal": "原文を見る",
    "viewTranslation": "翻訳を見る",
    "yes": "是",
    "you": "你"
  },
  "errors": {
    "ai": {
      "api_key_required": "API Key は必須です",
      "base_url_required": "Base URL は必須です",
      "base_url_scheme_invalid": "Base URL は http:// または https:// で始まる必要があります",
      "base_url_should_not_end_with_chat_completions": "Base URL は /chat/completions で終わらないようにしてください",
      "config_service_not_initialized": "AI 設定サービスが初期化されていません",
      "config_valid": "AI 設定は有効です",
      "failed_to_create_request": "リクエストの作成に失敗しました",
      "forbidden_quota": "配额超限",
      "free_tier_exhausted": "AI の無料枠が上限に達しました。プロバイダー管理画面で「無料枠のみ使用」を無効化するか、有料キーに切り替えてください。",
      "invalid_base_url": "Base URL が無効です",
      "invalid_provider": "無効なプロバイダです",
      "no_trade_data_available": "利用可能な取引データがありません",
      "not_configured": "AI が設定されていません。先に AI 設定で有効化・設定してください。",
      "probe_ok": "OK",
      "probe_ok_no_models": "OK（model が返されませんでした）",
      "provider_required": "プロバイダを選択してください",
      "provider_returned_empty_message": "AI プロバイダが空のメッセージを返しました",
      "rate_limited": "AI サービスがレート制限/クォータ不足（429/資源枯渇）。しばらく待つか、利用可能な API Key/model に切り替えてください。",
      "request_failed": "API リクエストに失敗しました"
    },
    "connection_failed": {
      "content": "无法连接到服务器，请检查网络后重试。",
      "title": "接続に失敗しました"
    },
    "access_denied": "アクセスが拒否されました",
    "account_connected": "接続しました",
    "account_connection_failed": "取引サーバーへの接続に失敗しました",
    "account_not_found": "口座が見つかりません",
    "auto_trading_disabled": "自動売買を無効にしました",
    "auto_trading_enabled": "自動売買を有効にしました",
    "email_already_registered": "このメールアドレスは既に登録されています",
    "invalid_credentials": "認証情報が正しくありません",
    "not_authenticated": "認証されていません",
    "schedule_service_not_available": "スケジュールサービスは利用できません",
    "translate_failed": "翻訳に失敗しました",
    "user_not_found": "ユーザーが見つかりません"
  },
  "marketplace": {
    "author": {
      "avgRating": "平均評価",
      "empty": "公開された戦略はまだありません。戦略ライブラリで公開してください。",
      "published": "公開済み"
    },
    "card": {
      "by": "by",
      "free": "無料",
      "owned": "購入日",
      "subscribers": "購読者",
      "winRate": "勝率"
    },
    "detail": {
      "assetClass": "資産クラス",
      "author": "作成者",
      "commentPlaceholder": "コメントを書く...",
      "comments": "コメント",
      "description": "説明",
      "getFree": "無料で入手",
      "rentPrice": "¥{{amount}} / 月",
      "subscribers": "購読者",
      "yourRating": "あなたの評価"
    },
    "messages": {
      "commentFailed": "コメントに失敗しました",
      "commentPosted": "コメントを投稿しました",
      "loginFirst": "先にログインしてください",
      "paymentComingSoon": "決済機能は近日公開",
      "rateFailed": "評価に失敗しました",
      "rated": "評価を送信しました",
      "subscribeFailed": "失敗",
      "subscribed": "購入に追加しました"
    },
    "payment": {
      "alreadyPurchased": "この戦略は既に購入済みです。",
      "balanceAfter": "購入後残高",
      "cancel": "キャンセル",
      "confirm": "購入確定",
      "depositPrompt": "続行するには入金してください。",
      "goToDeposit": "入金",
      "insufficientBalance": "残高不足",
      "oneTimePurchase": "¥{{amount}} 買い切り",
      "price": "価格",
      "purchaseFailed": "購入に失敗しました。もう一度お試しください。",
      "purchaseSuccess": "購入完了！戦略がライブラリに追加されました。",
      "purchasing": "処理中...",
      "strategyName": "戦略",
      "title": "購入確定",
      "walletBalance": "残高"
    },
    "purchases": {
      "empty": "購入履歴はまだありません。マーケットで戦略を見つけましょう。",
      "status": "ステータス",
      "strategy": "戦略"
    },
    "sort": {
      "newest": "新着順",
      "performance": "パフォーマンス順",
      "popular": "人気順",
      "priceAsc": "価格：安い順",
      "priceDesc": "価格：高い順",
      "rating": "評価順",
      "score": "総合スコア"
    },
    "tabs": {
      "author": "作成者センター",
      "marketplace": "マーケット",
      "purchases": "購入履歴",
      "subscriptions": "マイ購読"
    },
    "empty": "公開された戦略はまだありません",
    "filterByClass": "資産クラスで絞り込む",
    "noSubscriptions": "購読はまだありません",
    "publish": "戦略を公開",
    "searchPlaceholder": "戦略を検索...",
    "subtitle": "コミュニティ戦略を発見、購入、利用",
    "title": "ストラテジーマーケット"
  },
  "symbolDetection": {
    "tradeMode": {
      "disabled": "已禁用",
      "longOnly": "仅做多",
      "longShort": "多空双向",
      "shortOnly": "仅做空",
      "unknown": "未知"
    },
    "label": "検出された銘柄",
    "loading": "解析中...",
    "noSymbols": "取引銘柄が検出されませんでした。具体的な銘柄名を含めてみてください（例：「Bitcoin」「EURUSD」「Gold」）",
    "resolvedTooltip": "ブローカー：{{broker}} | モード：{{mode}}",
    "unresolvedTooltip": "取引口座が未バインドのため、解決できません"
  },
  "wallet": {
    "table": {
      "amount": "金額",
      "balanceAfter": "調整後残高",
      "description": "説明",
      "time": "時間",
      "type": "種類"
    },
    "txType": {
      "adjustment": "残高調整",
      "deposit": "入金",
      "fee": "手数料",
      "reversal": "取消",
      "withdrawal": "出金"
    },
    "accountNumber": "口座番号",
    "balance": "残高",
    "currency": "通貨",
    "deposit": "入金",
    "frozen": "凍結",
    "frozenBalance": "凍結",
    "history": "履歴",
    "title": "マイウォレット",
    "transactions": "取引履歴",
    "withdraw": "出金"
  },
  "app": {
    "name": "AntTrader"
  },
  "language": {
    "english": "English",
    "japanese": "日本語",
    "simplifiedChinese": "简体中文",
    "traditionalChinese": "繁體中文",
    "vietnamese": "Tiếng Việt"
  },
  "market": {
    "allSymbols": "全銘柄",
    "ask": "売値",
    "bid": "買値",
    "common": "共通",
    "emptyWatchlist": "暂无自选",
    "loadingSymbols": "読み込み中...",
    "mid": "仲値",
    "noSymbolSelected": "銘柄を選択してマーケットデータを表示",
    "noSymbolsFound": "銘柄が見つかりません",
    "popularSymbols": "人気銘柄",
    "searchPlaceholder": "銘柄を検索（例: EURUSD, XAUUSD）",
    "searchSymbol": "搜索品种...",
    "selectAccount": "取引口座を選択",
    "selectSymbol": "銘柄を選択",
    "spread": "スプレッド",
    "watchlist": "ウォッチリスト"
  },
  "menu": {
    "accounts": "アカウント",
    "aiAssistant": "AIアシスタント",
    "algoDashboard": "アルゴダッシュボード",
    "analytics": "分析",
    "assetAnalysis": "AI分析",
    "assets": "戦略資産",
    "autoTrading": "自動取引",
    "dashboard": "ダッシュボード",
    "devGroup": "戦略開発",
    "experiments": "戦略実験",
    "indicatorCatalog": "インジケーターカタログ",
    "logs": "システムログ",
    "market": "マーケット",
    "marketRegime": "マーケットレジーム",
    "marketTools": "マーケット分析ツール",
    "marketplace": "マーケットプレイス",
    "opsGroup": "戦略運用",
    "schedules": "戦略スケジュール",
    "strategies": "戦略管理",
    "strategy": "戦略",
    "strategyLibrary": "戦略ライブラリ",
    "strategyWorkspace": "戦略ワークスペース",
    "trading": "取引",
    "wallet": "ウォレット"
  },
  "profile": {
    "lastLogin": "最終ログイン",
    "nickname": "ニックネーム",
    "registered": "已注册",
    "role": "役割",
    "status": "ステータス",
    "title": "プロフィール"
  },
  "share": {
    "actions": "操作",
    "createNew": "新しい共有リンクを作成",
    "createdAt": "作成しました",
    "deleteConfirm": "删除此分享链接？",
    "empty": "共有リンクはまだありません",
    "expires": "有効期限",
    "positions": "持仓",
    "showPositions": "显示持仓",
    "title": "共有管理",
    "token": "共有リンク",
    "userId": "ユーザー",
    "views": "閲覧数"
  },
  "sharePage": {
    "avgHolding": "平均保有時間",
    "avgLoss": "平均損失",
    "avgWin": "平均利益",
    "bestTrade": "ベストトレード",
    "bySymbol": "銘柄別成績",
    "closeTime": "決済",
    "count": "取引数",
    "disclaimer": "過去の実績は将来の成果を保証するものではありません。",
    "equityCurve": "資産曲線",
    "expired": "この共有リンクは期限切れです",
    "footer": "AntTrader により生成",
    "language": "言語",
    "loadFailed": "共有データの読み込みに失敗しました",
    "losingTrades": "負けトレード数",
    "maxDrawdown": "最大ドローダウン",
    "netProfit": "純損益",
    "noPositions": "暂无持仓",
    "noTrades": "取引履歴がありません",
    "notFound": "見つかりません",
    "openPrice": "开仓价",
    "positions": "当前持仓",
    "positionsLocked": "创建者未开放持仓查看",
    "profit": "損益",
    "profitFactor": "プロフィットファクター",
    "sharpeRatio": "シャープレシオ",
    "side": "売買",
    "subtitle": "実際の取引成績",
    "symbol": "銘柄",
    "title": "取引パフォーマンス",
    "totalReturn": "純損益",
    "totalTrades": "総取引数",
    "totalVolume": "総取引量",
    "tradeRecords": "取引履歴",
    "volume": "数量",
    "winRate": "勝率",
    "winningTrades": "勝ちトレード数",
    "worstTrade": "ワーストトレード"
  },
  "topbar": {
    "logout": "ログアウト",
    "profile": "プロフィール",
    "settings": "設定",
    "switchToAdmin": "管理画面へ切替",
    "systemOk": "システムは正常に稼働中",
    "user": "ユーザー"
  }
} as const;
export default Base;
