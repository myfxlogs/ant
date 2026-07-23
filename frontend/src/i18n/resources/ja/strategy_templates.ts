// Auto-generated from proto/ant/v1/i18n/strategy_templates_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategyTemplates = {
  "strategy": {
    "templates": {
      "scheduleLaunch": {
        "form": {
          "scheduleTypes": {
            "hfQuote": "高频报价",
            "interval": "定时执行",
            "klineClose": "K線クローズ"
          },
          "account": "口座",
          "accountPlaceholder": "选择账户",
          "defaultVolume": "默认手数",
          "defaultVolumeTip": "每个信号的默认下单量",
          "enableAfterCreate": "创建后立即启用",
          "hfCooldownMs": "高频冷却(毫秒)",
          "hfCooldownMsTip": "报价驱动执行间的冷却时间",
          "intervalMs": "间隔(毫秒)",
          "intervalMsTip": "非高频模式最小1000ms",
          "investorTag": "投资者(只读)",
          "maxDrawdownPct": "最大回撤%",
          "maxDrawdownPctTip": "回撤超过此阈值自动停止",
          "maxPositions": "最大持仓数",
          "maxPositionsTip": "同时持有的最大仓位数量",
          "riskSection": "风控设置",
          "scheduleName": "计划名称",
          "scheduleNameMax": "最多64字符",
          "scheduleNamePlaceholder": "例：EURUSD M5 朝の戦略",
          "scheduleType": "计划类型",
          "stopLossOffset": "止损偏移",
          "stopLossOffsetTip": "距入场价的止损距离(点)",
          "strategyParamsSection": "策略参数",
          "symbol": "銘柄",
          "symbolPlaceholder": "选择品种",
          "symbolPlaceholderEmpty": "未配置品种",
          "takeProfitOffset": "止盈偏移",
          "takeProfitOffsetTip": "距入场价的止盈距离(点)",
          "timeframe": "時間足"
        },
        "actions": {
          "addAccount": "添加账户",
          "create": "スケジュール作成",
          "createAndEnable": "作成して有効化",
          "createScheduleNoEnable": "スケジュール作成",
          "publishTemplate": "テンプレートを公開",
          "updateTradingPassword": "更新交易密码"
        },
        "metrics": {
          "annualReturn": "年率リターン",
          "maxDrawdown": "最大ドローダウン",
          "sharpe": "シャープレシオ",
          "totalReturn": "総リターン",
          "totalTrades": "取引回数",
          "winRate": "勝率"
        },
        "backtestRunningHint": "バックテスト実行中です。しばらくお待ちください。",
        "errorInvestorAccount": "无法使用投资者账户启动计划。请更新交易密码以启用交易。",
        "investorWarningBody": "此账户为投资者(只读)模式，需要交易权限才能启动计划。",
        "investorWarningTitle": "投资者账户",
        "keyMetrics": "主要指標",
        "launchSection": "スケジュール起動",
        "newPasswordPlaceholder": "新しい取引パスワードを入力",
        "noAccountBody": "启动计划前需要先绑定MT账户。",
        "noAccountTitle": "无账户",
        "noRun": "バックテスト実行がありません",
        "score": "スコア",
        "title": "スケジュール起動",
        "tradePermissionOk": "交易权限验证通过",
        "updatePasswordFailed": "更新交易密码失败",
        "updatePasswordHint": "输入此账户的交易密码以启用交易。",
        "updatePasswordOk": "交易密码已更新",
        "updatePasswordStillInvestor": "密码更新成功但账户仍为投资者模式，请联系客服。",
        "updatePasswordTitle": "更新交易密码",
        "verifyingPermission": "验证交易权限中..."
      },
      "backtest": {
        "fields": {
          "account": "口座",
          "extraSymbols": "追加銘柄 (複数選択)",
          "initialCapital": "初期資金",
          "range": "範囲",
          "symbol": "銘柄",
          "timeframe": "時間足",
          "title": "タイトル"
        },
        "parameters": {
          "title": "策略参数"
        },
        "placeholders": {
          "account": "口座を選択",
          "extraSymbols": "オプション。ペア/ローテーション戦略に有用",
          "range": "期間選択",
          "symbol": "銘柄を選択"
        },
        "quickRange": {
          "custom": "カスタム"
        },
        "tooltips": {
          "extraSymbols": "追加のK線取得銘柄（同アカウント、同タイムフレーム）。context[\"closes_by_symbol\"]でアクセス可能。"
        },
        "validation": {
          "accountRequired": "口座を選択してください",
          "initialCapitalRequired": "初期資金が必要です",
          "rangeRequired": "期間が必要です",
          "symbolRequired": "銘柄を選択してください",
          "timeframeRequired": "時間足を選択してください"
        },
        "accountDisabledSuffix": "（無効）",
        "modalTitleWithName": "バックテスト: {{name}}",
        "title": "バックテスト"
      },
      "backtestRuns": {
        "actions": {
          "createSchedule": "スケジュール作成",
          "launchSchedule": "スコア表示",
          "view": "表示"
        },
        "status": {
          "canceled": "キャンセル済",
          "canceling": "キャンセル中",
          "completed": "完了",
          "failed": "失败",
          "queued": "待機中",
          "running": "実行中"
        },
        "table": {
          "actions": "操作",
          "createdAt": "作成日時",
          "status": "状態",
          "symbol": "銘柄",
          "timeframe": "時間足",
          "title": "タイトル"
        },
        "batchDelete": "{{count}}件削除",
        "batchDeleteConfirm": "{{count}}件のバックテストレポートを削除しますか？",
        "batchDeleteSuccess": "{{count}}件のバックテストレポートを削除しました",
        "deleteConfirm": "この実行を削除しますか？",
        "empty": "バックテスト実行なし",
        "title": "バックテスト実行履歴"
      },
      "codeModal": {
        "actions": {
          "copy": "コピー"
        },
        "title": "戦略コード"
      },
      "editTemplateModal": {
        "actions": {
          "validateCode": "コード検証"
        },
        "fields": {
          "code": "戦略コード",
          "description": "説明",
          "name": "名称",
          "publicShare": "公開"
        },
        "placeholders": {
          "codeSample": "Python 戦略コードを入力...",
          "description": "任意：説明",
          "name": "例：移動平均クロス戦略"
        },
        "title": {
          "create": "テンプレート作成",
          "edit": "テンプレート編集"
        },
        "validation": {
          "codeRequired": "コードが必要です",
          "nameRequired": "名称を入力してください"
        }
      },
      "actions": {
        "backtest": "バックテスト",
        "copy": "コピー",
        "create": "テンプレート作成",
        "createTemplate": "テンプレート作成",
        "delete": "削除",
        "edit": "編集",
        "launchSchedule": "スケジュール起動",
        "viewCode": "コード表示"
      },
      "badges": {
        "preset": "プリセット"
      },
      "messages": {
        "backtestCancelFailed": "バックテストキャンセル失敗",
        "backtestCancelRequested": "バックテストキャンセル依頼済",
        "backtestRangeInvalid": "無効なバックテスト期間",
        "backtestReportDeleted": "バックテストレポート削除済",
        "backtestReportNotFound": "バックテストレポート未発見",
        "backtestRunNoPublishedTemplate": "バックテスト実行に公開テンプレートがありません",
        "backtestRunningCannotPublish": "バックテスト実行中のため公開不可。",
        "backtestSubmitFailed": "バックテスト提出失敗",
        "backtestSubmitted": "バックテスト提出済",
        "cannotPublishAndCreateDraftFailed": "公開不可。下書き作成失敗。",
        "codeCopied": "コードコピー済",
        "codeValidationFailed": "コード検証失敗",
        "codeValidationNotPassed": "コード検証未通過",
        "codeValidationPassed": "コード検証通過",
        "copyFailed": "コピーに失敗しました。手動でコピーしてください",
        "createScheduleFailed": "スケジュール作成失敗",
        "deepLinkNavigate": "外部リンクからテンプレートと最新実行情報を開きました",
        "enterStrategyCode": "戦略コードを入力してください",
        "fetchTemplateListFailed": "テンプレート一覧読込失敗",
        "missingDraftIdCannotPublish": "下書きID不足のため公開不可。",
        "missingScheduleInfo": "スケジュール情報不足",
        "publishFailed": "公開失敗",
        "publishedButNoTemplateId": "公開されましたが、テンプレートIDが不足しています。",
        "readStrategyCodeFailed": "ストラテジーコード読込失敗",
        "readTemplateStatusFailed": "テンプレート状態の読込失敗",
        "republishedButNoTemplateId": "再公開されましたが、テンプレートIDが不足しています。",
        "scheduleCreated": "スケジュール作成完了",
        "scheduleCreatedAndEnabled": "スケジュール作成・有効化",
        "selectBacktestRange": "バックテスト期間を選択してください",
        "strategyCodeEmptyCannotBacktest": "ストラテジーコードが空です。バックテスト不可。",
        "strategyCodeEmptyCannotPublish": "ストラテジーコードが空です。公開前に保存してください。",
        "systemTemplateReadOnly": "システムテンプレートは読み取り専用です。編集するには複製してください。",
        "templateAlreadyPublished": "テンプレートは既に公開済みです",
        "templateCreated": "テンプレート作成済",
        "templateDeleted": "テンプレート削除済",
        "templateNotDraftUnknownPublishStatus": "テンプレートは下書きではありません。公開状態不明。",
        "templateNotPublishedCannotCreateSchedule": "テンプレート未公開。スケジュール作成不可。",
        "templatePublished": "テンプレート公開済",
        "templateRepublished": "テンプレート再公開済",
        "templateUpdated": "テンプレート更新済"
      },
      "status": {
        "draft": "下書き",
        "published": "公開済み"
      },
      "table": {
        "actions": "操作",
        "createdAt": "作成日時",
        "defaultHint": "デフォルト",
        "description": "説明",
        "emptyUser": "まだユーザーテンプレートがありません。上の「テンプレート作成」をクリックしてください。",
        "loadingDefault": "デフォルトテンプレートを読み込み中...",
        "name": "名称",
        "status": "状態",
        "tags": "タグ",
        "updatedAt": "更新日時",
        "useCount": "使用回数",
        "visibility": "公開範囲"
      },
      "tabs": {
        "system": "システムテンプレート",
        "user": "ユーザーテンプレート"
      },
      "visibility": {
        "private": "非公開",
        "public": "公開"
      },
      "gallery": {
        "title": "ストラテジー",
        "aiGenerate": "AI生成",
        "searchPlaceholder": "ストラテジーを検索...",
        "filterAll": "すべて",
        "filterMine": "マイ",
        "filterSystem": "システム",
        "sortRecent": "新着",
        "sortReturn": "リターン",
        "sortRisk": "リスク",
        "sortUsage": "使用量",
        "empty": "ストラテジーが見つかりません",
        "system": "システム",
        "shared": "共有",
        "deploy": "デプロイ",
        "fork": "フォーク",
        "publish": "公開",
        "unpublish": "非公開にする",
        "unpublishSuccess": "非公開にしました",
        "unpublishFailed": "非公開化に失敗しました",
        "deleteFailed": "削除に失敗しました"
      },
      "detail": {
        "notFound": "ストラテジーが見つかりません",
        "overview": "概要",
        "noDescription": "説明なし",
        "equityCurve": "エクイティカーブ",
        "tradeStats": "取引統計",
        "profitFactor": "プロフィットファクター",
        "parameters": "パラメータ"
      },
      "copySuffix": "（コピー）",
      "defaultDraftName": "下書きテンプレート",
      "deleteConfirm": "このテンプレートを削除しますか？",
      "scheduleName": "{{symbol}} {{timeframe}} {{name}}",
      "title": "戦略テンプレート"
    }
  }
} as const;
export default StrategyTemplates;
