// Auto-generated from proto/ant/v1/i18n/ai_wizard_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiWizard = {
  "ai": {
    "wizard": {
      "generate": {
        "modals": {
          "final": {
            "title": "コード生成完了。検証を推奨。"
          }
        },
        "status": {
          "running": {
            "code": "進行中",
            "generic": "{{title}}進行中",
            "risk": "進行中",
            "signals": "進行中",
            "style": "進行中"
          },
          "done": "完了",
          "error": "エラー",
          "idle": "待機中",
          "inProgress": "進行中"
        },
        "actions": {
          "abort": "中止",
          "goValidate": "検証へ",
          "hide": "隠す",
          "regenerateSummary": "要約再生成",
          "rerun": "再生成",
          "runAgents": "複数専門家分析 + コード生成"
        },
        "cards": {
          "resultsTitle": "複数エキスパート\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\"
        },
        "hints": {
          "afterGenerated": "生成後、検証/バックテスト/公開へ"
        },
        "labels": {
          "elapsed": "経過"
        },
        "sections": {
          "output": "出力",
          "prompt": "プロンプト",
          "spec": "仕様"
        }
      },
      "publishBacktest": {
        "modals": {
          "score": {
            "title": "スコア確認"
          },
          "status": {
            "title": "バックテスト進行中"
          }
        },
        "actions": {
          "close": "閉じる",
          "confirm": "確認",
          "inProgress": "進行中",
          "retry": "再試行",
          "runInBackground": "バックグラウンド実行",
          "startBacktest": "開始",
          "succeeded": "成功"
        },
        "cards": {
          "backtestTitle": "バックテスト",
          "scoreCardTitle": "スコアカード"
        },
        "labels": {
          "confirmed": "確認済",
          "elapsed": "経過",
          "overallScore": "総合スコア",
          "scoringProgress": "進捗",
          "status": "状態"
        },
        "draftName": "バックテスト {{datetime}} {{symbol}} {{timeframe}}",
        "draftNameShort": "バックテスト {{symbol}} {{timeframe}}"
      },
      "setup": {
        "modals": {
          "deleteDataset": {
            "content": "選択した凍結データセットを削除しますか？",
            "ok": "削除",
            "title": "データセット削除"
          }
        },
        "actions": {
          "deleteCurrentDataset": "現在データセット削除",
          "freezeFromCurrentRange": "現在範囲から凍結",
          "refreshDataset": "更新"
        },
        "cards": {
          "constraintsAndGoalTitle": "制約と目標",
          "hardConstraintsTitle": "ハード制約",
          "hintsTitle": "ヒント",
          "tradeAndDataTitle": "取引とデータ"
        },
        "dataModes": {
          "dataset": "データセット",
          "klineRange": "履歴K線"
        },
        "hints": {
          "nextWillGenerateCode": "次でコード生成開始。",
          "tradeDataNextStep": "完了後「次へ」。"
        },
        "labels": {
          "account": "アカウント",
          "backtestRange": "バックテスト範囲",
          "dataset": "データセット",
          "historicalData": "履歴データ",
          "intent": "戦略目標/アイデア",
          "macroEvents": "マクロイベント",
          "macroModule": "マクロモジュール",
          "maxDrawdownPct": "最大DD(%)",
          "maxTradesPerDay": "1日最大取引数",
          "riskPerTradePct": "取引リスク(%)",
          "symbol": "銘柄",
          "timeframe": "時間枠"
        },
        "macro": {
          "off": "OFF",
          "on": "开"
        },
        "messages": {
          "datasetDeleted": "データセット削除済"
        },
        "placeholders": {
          "intentExample": "例: トレンドフォロー、高ボラ回避、高勝率重視",
          "macroExample": "例：\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n2024-01-03 21:15 FOMC議事録\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n2024-01-05 20:30 非農業部門雇用者数",
          "selectAccount": "アカウント選択",
          "selectFrozenDataset": "データセット選択",
          "selectSymbol": "銘柄選択",
          "selectTimeframe": "時間枠選択"
        },
        "validations": {
          "enterIntent": "戦略目標を入力",
          "selectAccount": "アカウントを選択",
          "selectDataset": "データセットを選択",
          "selectSymbol": "銘柄を選択",
          "selectTimeframe": "時間枠を選択"
        }
      },
      "prompts": {
        "base": {
          "account": "アカウント: {{accountId}}",
          "constraints": "制約: DD={{maxDrawdownPct}}% リスク={{riskPerTradePct}}% 最大取引={{maxTradesPerDay}}",
          "data": "データ: {{dataSpec}}",
          "empty": "(空)",
          "macroDisabled": "マクロ: 未使用",
          "macroEnabled": "マクロイベント（ユーザー提供）：\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{text}}",
          "params": "パラメータ（定義+現在値; 実行時にcontext[\"params\"]に注入）：\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{params}}",
          "symbol": "銘柄: {{symbol}}",
          "timeframe": "時間枠: {{timeframe}}",
          "userIntent": "ユーザー戦略目標（自然言語）：\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "dataSpec": {
          "dataset": "データセット {{datasetId}}",
          "klineRange": "K線範囲 {{from}} - {{to}}"
        },
        "summary": {
          "codeTitle": "コード:",
          "intro": "戦略コードの核心を簡潔に説明してください。",
          "mustInclude1": "1) 戦略タイプ",
          "mustInclude2": "2) 主要エントリー条件",
          "mustInclude3": "3) 主要エグジット/リスク管理",
          "mustInclude4": "4) 適用/非適用シナリオ",
          "mustIncludeTitle": "必須:",
          "userIntent": "ユーザー期待（自然言語）：\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{intent}}"
        },
        "upstream": {
          "risk": "【リスク管理結論】\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{text}}",
          "sectionTitle": "【上流エージェント結論】",
          "signals": "【シグナル設計結論】\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{text}}",
          "style": "【市場状況/スタイル結論】\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\n{{text}}"
        }
      },
      "publish": {
        "actions": {
          "publishTemplate": "テンプレート公開",
          "startBacktest": "バックテスト",
          "validateCode": "コード検証"
        },
        "cards": {
          "codeTitle": "1) コード",
          "launchTitle": "3) 公開スケジュール",
          "scoreCardTitle": "2) スコアカード"
        },
        "messages": {
          "validateFailed": "検証不合格",
          "validateOk": "検証合格"
        },
        "placeholders": {
          "codeEditable": "AI生成コードがここに表示されます。"
        }
      },
      "strategyParams": {
        "actions": {
          "addParam": "追加",
          "delete": "削除",
          "exportJson": "JSON出力",
          "importJson": "JSON入力"
        },
        "hints": {
          "intro": "これらのパラメータは:",
          "line1": "1) テンプレートに保存",
          "line2": "2) スケジュールに書き込み",
          "line3Prefix": "3) Pythonに注入"
        },
        "labels": {
          "default": "デフォルト",
          "description": "説明",
          "label": "ラベル",
          "max": "最大",
          "min": "最小",
          "name": "名前",
          "options": "options",
          "step": "刻み幅",
          "type": "型",
          "value": "value"
        },
        "messages": {
          "copied": "コピー済",
          "copyFailed": "コピー失敗",
          "importFormatInvalid": "形式エラー",
          "importMissingName": "name欠落",
          "imported": "{{count}}個入力",
          "jsonParseFailed": "JSON解析失敗"
        },
        "modals": {
          "copyAndClose": "コピーして閉じる",
          "exportTitle": "JSON出力",
          "importOk": "入力",
          "importTitle": "JSON入力"
        },
        "placeholders": {
          "defaultExample": "例: 10",
          "description": "説明",
          "importJson": "JSON貼り付け",
          "label": "表示名",
          "nameExample": "例: fast",
          "optionsExample": "例: low,medium,high",
          "value": "空欄でdefault使用"
        },
        "types": {
          "bool": "真偽値",
          "number": "数値",
          "select": "選択",
          "string": "文字列"
        },
        "validations": {
          "nameRequired": "name必須",
          "typeRequired": "type必須"
        },
        "empty": "パラメータなし。",
        "paramCardTitle": "パラメータ #{{index}}",
        "title": "戦略パラメータ（任意）"
      },
      "actions": {
        "cancel": "キャンセル",
        "next": "次へ",
        "prev": "戻る"
      },
      "agents": {
        "codeTitle": "コード生成",
        "riskTitle": "リスク管理",
        "signalsTitle": "シグナル/指標",
        "styleTitle": "市場状況/スタイル"
      },
      "messages": {
        "agentFailed": "{{title}}失敗",
        "aiRequestTimeout": "AIリクエストがタイムアウトしました（>{{seconds}}秒）",
        "backtestCreated": "バックテスト作成",
        "backtestNotDoneWait": "バックテスト完了待ち",
        "chatAborted": "中止",
        "codeInvalidFixAndContinue": "コード検証失敗",
        "confirmScoreFirst": "まずスコア確認",
        "createBacktestFailed": "作成失敗",
        "createDraftFailed": "ドラフト作成失敗",
        "createScheduleFailed": "作成失敗",
        "datasetFrozenCreated": "データセット凍結作成",
        "draftNotCreated": "ドラフト未作成",
        "draftSaved": "ドラフト保存",
        "fillRequired": "必須項目入力",
        "fillRequiredWithFields": "必須項目入力: {{fields}}",
        "freezeDatasetFailed": "凍結失敗",
        "generateCodeFirst": "まずコード生成",
        "inputIntentFirst": "まず戦略目標入力",
        "loadAccountsFailed": "アカウント読込失敗",
        "loadDatasetFailed": "データセット読込失敗",
        "loadSymbolsFailed": "銘柄読込失敗",
        "modelReturnedEmpty": "空返答",
        "noCodeToBacktest": "バックテストするコードなし",
        "noCodeToValidate": "検証するコードなし",
        "noPythonCodeBlock": "コードブロックなし",
        "publishFailed": "公開失敗",
        "publishTemplateFirst": "まずテンプレート公開",
        "publishedNoId": "公開済（ID未取得）",
        "saveFailed": "保存失敗",
        "scheduleAlreadyExists": "同じスケジュールが既に存在",
        "scheduleCreated": "スケジュール作成",
        "scheduleCreatedAndEnabled": "スケジュール作成・有効化",
        "startBacktestFirst": "まずバックテスト開始",
        "templatePublished": "テンプレート公開",
        "userAborted": "中止",
        "validateCodeFirst": "まずコード検証",
        "validateError": "検証エラー",
        "validateFailed": "検証不合格",
        "validateOk": "検証合格",
        "watchBacktestRunFailed": "watch失敗"
      },
      "schedule": {
        "defaultName": "AIスケジュール {{symbol}} {{timeframe}}"
      },
      "steps": {
        "analyze": "分析",
        "backtest": "バックテスト",
        "compliance": "コンプライアンス",
        "generate": "戦略生成",
        "plan": "計画",
        "publishBacktest": "バックテスト・公開-バックテスト",
        "publishCode": "バックテスト・公開-コード",
        "publishLaunch": "バックテスト・公開-公開",
        "setup": "基本情報"
      },
      "template": {
        "defaultDescription": "AI生成",
        "defaultName": "AI戦略 {{title}}"
      },
      "currentModel": "現在のモデル: {{model}}",
      "subtitle": "1ページ1ステップ",
      "title": "AI 戦略ウィザード"
    }
  }
} as const;
export default AiWizard;
