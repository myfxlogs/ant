// Auto-generated from proto/ant/v1/i18n/strategy_schedules_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const StrategySchedules = {
  "strategy": {
    "schedules": {
      "editModal": {
        "advanced": {
          "triggerModeOptions": {
            "hf": "高頻度シグナルストリーム",
            "stable": "安定（ローソク/周期）"
          },
          "fixedIntervalSeconds": "固定間隔(秒)",
          "fixedIntervalSecondsExtra": "任意。固定間隔で実行（時間足追従しない）。例：60 は60秒ごと",
          "hfCooldownMs": "高頻度クールダウン(ms)",
          "hfCooldownMsExtra": "デバウンス：評価/発注の最小間隔",
          "parametersJson": "パラメータ(JSONオブジェクト)",
          "parametersJsonExtra": "ストラテジーのJSONパラメータ",
          "stableOverrideIntervalSeconds": "安定モード上書き間隔(秒)",
          "stableOverrideIntervalSecondsExtra": "任意。安定モードの間隔を上書き",
          "timeframe": "時間足",
          "timeframeExtra": "ローソク/指標計算に使用",
          "title": "詳細設定",
          "triggerMode": "トリガーモード",
          "triggerModeExtra": "安定：ローソク/周期（ノイズ少・遅延あり）；高頻度：クオート（速い・デバウンス必要）"
        },
        "autoName": {
          "strategy": "ストラテジー"
        },
        "fields": {
          "account": "口座",
          "cronExpression": "Cron 式",
          "cronExtra": "標準の5項：分 時 日 月 週。例：*/5 * * * *；0 9 * * 1-5",
          "enableExtra": "作成後にスケジュール有効化",
          "intervalSeconds": "間隔(秒)",
          "intervalSecondsExtra": "時間足に自動追従。変更不要",
          "lot": "ロット",
          "lotExtra": "数量。0.01 からの開始を推奨",
          "name": "名称",
          "runFrequency": "実行頻度",
          "symbol": "銘柄",
          "template": "テンプレート",
          "templateExtra": "「戦略管理」に保存されたテンプレート"
        },
        "placeholders": {
          "name": "例：EURUSD M5 朝の戦略",
          "selectAccountFirst": "先に口座を選択",
          "symbol": "銘柄を選択"
        },
        "runFrequencyExtra": {
          "byTimeframe": "タイムフレーム実行",
          "cron": "高度：Cron で実行時間を精密に制御"
        },
        "runFrequencyOptions": {
          "byTimeframe": "時間足でトリガー（推奨）",
          "cron": "Cron 表达式"
        },
        "title": {
          "create": "スケジュール作成",
          "edit": "スケジュール編集"
        },
        "validation": {
          "accountRequired": "口座を選択してください",
          "cronRequired": "cron を入力してください",
          "lotRequired": "ロットを入力してください",
          "nameRequired": "名称を入力してください",
          "runFrequencyRequired": "実行頻度を選択してください",
          "symbolRequired": "銘柄を選択してください",
          "templateRequired": "テンプレートを選択してください",
          "timeframeRequired": "時間足を選択してください",
          "triggerModeRequired": "トリガーモードが必要です"
        }
      },
      "health": {
        "fields": {
          "configKey": "設定キー",
          "failedRuns": "失敗回数",
          "grade": "ヘルス評価",
          "lastRunAt": "最終実行",
          "latestError": "最新エラー",
          "latestProfit": "最新損益",
          "latestTicket": "最新約定チケット",
          "rule": "判定基準",
          "successOverTotal": "成功 / 総数",
          "thresholds": "現在の閾値"
        },
        "grade": {
          "alert": "アラート",
          "healthy": "健全",
          "noSample": "サンプル不足",
          "pending": "未評価",
          "watch": "要注意"
        },
        "messages": {
          "clickRefresh": "更新をクリックしてヘルスデータ読込",
          "loadFailed": "ヘルスデータの読み込みに失敗しました"
        },
        "notes": {
          "alert": "成功率低下。ストラテジー/アカウント状態を今すぐ調査。",
          "healthy": "成功率が高く、失敗回数も許容範囲です。",
          "noSample": "評価に必要なサンプル不足（最低 {{minSampleSize}} 件）。",
          "pending": "まずヘルスチェックを実行してください。",
          "watch": "成功率は監視対象です（>= {{yellowSuccessRate}}%）。"
        },
        "runLogs": {
          "signalType": "シグナル（発注用）"
        },
        "sections": {
          "orders": "最近の注文記録",
          "runLogs": "最近の実行ログ"
        },
        "summaryBanner": "ヘルス評価: {{grade}}、サンプル {{totalRuns}} 件、成功率 {{successRate}}%",
        "thresholdsSummary": "min_sample_size={{minSampleSize}}、緑: 成功率>={{greenSuccessRate}}% かつ 失敗<={{greenMaxFailedRuns}}、黄: 成功率>={{yellowSuccessRate}}%",
        "title": "戦略ヘルスチェック {{name}}"
      },
      "triggerModal": {
        "actions": {
          "confirmOrder": "発注する",
          "rerun": "再実行"
        },
        "cards": {
          "logs": "実行ログ",
          "signal": "シグナル（発注用）"
        },
        "confirmOrder": {
          "ok": "確認",
          "title": "発注する"
        },
        "messages": {
          "signalNotOrderable": "シグナルは注文不可"
        },
        "summary": {
          "account": "口座",
          "scheduleName": "スケジュール名",
          "symbol": "銘柄",
          "timeframe": "時間足"
        },
        "emptyLogs": "(ログなし)",
        "emptySignal": "シグナルなし",
        "title": "今すぐ実行（即時発注）"
      },
      "actions": {
        "create": "スケジュール作成",
        "healthCheck": "ヘルスチェック",
        "logs": "実行ログ",
        "runNow": "今すぐ実行"
      },
      "deleteConfirm": {
        "title": "このスケジュールを削除しますか？"
      },
      "format": {
        "cron": "定时: {{expr}}",
        "interval": "{{s}}秒毎"
      },
      "messages": {
        "defaultTemplateNotFound": "デフォルトテンプレートが見つかりません。更新して再試行してください。",
        "executeFailed": "実行に失敗しました",
        "importDefaultTemplateFailedNoId": "デフォルトテンプレートの取り込みに失敗しました（IDがありません）",
        "noOrderableSignal": "発注可能なシグナルがありません",
        "orderFailed": "注文に失敗しました",
        "orderSubmitted": "注文を送信しました",
        "parametersParseFailed": "パラメータ解析に失敗しました",
        "signalHoldCannotOrder": "シグナルが保留/無操作のため発注できません",
        "strategyExecuteFailed": "戦略実行に失敗しました",
        "templateCodeEmptyCannotExecute": "テンプレートコードが空です。実行できません。",
        "volumeInvalid": "数量が不正です（> 0）"
      },
      "status": {
        "disabled": "無効",
        "running": "実行中"
      },
      "table": {
        "account": "口座",
        "actions": "操作",
        "lastRun": "最終実行",
        "name": "名称",
        "schedule": "スケジュール",
        "status": "状態",
        "template": "テンプレート",
        "tradeParams": "取引パラメータ"
      },
      "templateVisibility": {
        "private": "非公開",
        "public": "公開"
      },
      "validation": {
        "parametersMustBeJsonObject": "パラメータはJSONオブジェクトである必要があります"
      },
      "createSchedule": "スケジュール作成",
      "enableCount": "有効化回数",
      "nextRunAt": "次回実行",
      "title": "戦略スケジュール"
    }
  }
} as const;
export default StrategySchedules;
