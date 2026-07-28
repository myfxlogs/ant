// Auto-generated from proto/ant/v1/i18n/ai_settings_ja.textproto
// DO NOT EDIT MANUALLY — run: npx tsx scripts/i18n-build.ts
const AiSettings = {
  "ai": {
    "settings": {
      "agent": {
        "defaults": {
          "code": {
            "inputHint": "例: パラダイム=トレンドフォロー、指標=EMA(fast)/EMA(slow)+ATRフィルタ"
          },
          "execution": {
            "inputHint": "例: 注文=EURUSDロング10ロット、スプレッド=0.6pip、目標5分、許容スリッページ=0.8pip"
          },
          "executor": {
            "identity": "取引執行最適化専門家 — スリッページと執行コストを最小化。"
          },
          "macro": {
            "inputHint": "例: 今週重要イベント=米CPI（木20:30）、FOMC議事録（水翌02:00）"
          },
          "portfolio": {
            "inputHint": "例: 既存戦略=トレンド-EURUSD、平均回帰-XAUUSD、総資産=50,000"
          },
          "researcher": {
            "identity": "マクロ経済・業界リサーチャー — マクロイベントとセクター動向を分析。"
          },
          "risk": {
            "inputHint": "例: 資産=10,000、許容DD=5%、取引リスク=0.5%、最大取引=5"
          },
          "risk_manager": {
            "identity": "厳格なリスク管理専門家 — ポジションサイジング、損切り、ドローダウン制限を設計。"
          },
          "sentiment": {
            "inputHint": "例: VIX 14→22、非商業純ロング-18%、ニュース「リセッション/利下げ」"
          },
          "signals": {
            "inputHint": "例: パラダイム=トレンドフォロー、時間枠=H1、指標=EMA/ATR/ADX"
          },
          "strategist": {
            "identity": "シニア定量ストラテジーアナリスト — 口座・市場状況に基づき戦略パラダイムを推奨。"
          },
          "style": {
            "inputHint": "例: アカウント=EURUSD小売、時間枠=H1、目標=月収益3%、DD<10%"
          }
        },
        "actions": {
          "add": "追加",
          "loadDefaults": "デフォルト8エージェント読込",
          "remove": "削除",
          "restoreDefaults": "デフォルト復元",
          "restoreDefaultsConfirmContent": "8システムエージェントをデフォルトにリセット。カスタムは保持。",
          "restoreDefaultsConfirmTitle": "システムデフォルト復元？",
          "save": "保存"
        },
        "fields": {
          "historicalBinding": "{{value}}（過去）",
          "identityPlaceholder": "身分/人物設定記述",
          "inputHintPlaceholder": "入力ヒント",
          "modelProfileEmpty": "先に「AI設定」でプロバイダー/モデルを有効化",
          "modelProfilePlaceholder": "デフォルト",
          "namePlaceholder": "エージェント名"
        },
        "messages": {
          "defaultsLoaded": "デフォルトテンプレート読込。「保存」で確定。",
          "empty": "カスタムエージェントなし。「追加」で設定。",
          "loading": "読込中…",
          "saveFailed": "保存失敗",
          "saveSuccess": "エージェント保存",
          "selectProfileFirst": "先に左で設定選択"
        },
        "types": {
          "code": "コード",
          "execution": "実行",
          "executor": "執行アドバイザー",
          "macro": "マクロ",
          "portfolio": "ポートフォリオ",
          "researcher": "マーケットリサーチャー",
          "risk": "リスク",
          "risk_manager": "リスクマネージャー",
          "sentiment": "センチメント",
          "signals": "シグナル/指標",
          "strategist": "ストラテジーアナリスト",
          "style": "スタイル/パラダイム"
        },
        "defaultName": "カスタムエージェント",
        "removeConfirmContent": "削除しますか？",
        "removeConfirmTitle": "エージェント削除",
        "title": "エージェント身分定義"
      },
      "apiKeyGuide": {
        "deepseek": {
          "step1": "DeepSeekプラットフォーム:",
          "step2": "ログイン/登録後、APIキー管理画面で作成。",
          "title": "DeepSeek API キー取得方法"
        },
        "zhipu": {
          "step1": "Zhipuプラットフォーム:",
          "step2": "ログイン/登録後、コンソールでAPI キー作成。",
          "title": "Zhipu API キー取得方法"
        },
        "default": "Current provider: {{provider}}. Go to the provider\\\\\\\\\\\\\\\\",
        "modelSuggestionDeepSeek": "モデル提案: deepseek-chat",
        "modelSuggestionZhipu": "モデル提案: glm-4-flash / glm-4",
        "selectProviderHint": "プロバイダー選択後、申請方法を表示。",
        "title": "API キー申請ガイド"
      },
      "profiles": {
        "actions": {
          "setCurrent": "現在に設定"
        },
        "delete": {
          "content": "削除しますか？",
          "title": "設定削除"
        },
        "current": "現在"
      },
      "actions": {
        "saveConfig": "設定保存",
        "validateApiKey": "API キー検証"
      },
      "discoverErrors": {
        "baseUrlInvalid": "Base URL が無効です。例: https://model.example.com または https://model.example.com/v1",
        "baseUrlRequired": "先に Base URL（モデルサービスURL）を入力してください。",
        "endpoint404": "エンドポイントが見つかりません: Base URL とプロトコル（/v1 など）を確認してください。",
        "freeTierExhausted": "無料枠を使い切りました。コンソールで設定を確認するか有料キーに切り替えてください。",
        "generic": "モデル一覧の取得に失敗しました。Base URL と API キーを確認してください。",
        "genericDetail": "モデル一覧の取得に失敗: {{detail}}",
        "invalidModelsResponse": "/models 互換ではない応答形式です。",
        "noModelsReturned": "利用可能なモデルがありません。権限と設定を確認してください。",
        "quotaForbidden403": "呼び出し拒否（クォータ）: コンソールの課金/クォータを確認してください。",
        "quotaOrRateLimit": "クォータまたはレート制限です。課金/制限を確認するか後で再試行してください。",
        "timeout": "タイムアウト: ネットワークを確認するか後で再試行してください。",
        "unauthorized": "認証失敗: API キー/シークレットを確認してください。",
        "unreachable": "モデルサービスに接続できません: Base URL・ネットワーク・ゲートウェイを確認してください。"
      },
      "errors": {
        "arrearage": "残高不足。コンソールで確認。",
        "forbidden": "アクセス拒否（403）。",
        "invalidModelId": "モデル利用不可{{model}}。",
        "timeout": "接続タイムアウト。",
        "unauthorized": "API キー無効（401）。"
      },
      "fields": {
        "apiKey": "API キー",
        "apiKeyConfigured": "設定済",
        "apiKeyReplaceHint": "交換する場合は再入力",
        "availableModels": "利用可能モデル",
        "availableModelsEmpty": "モデル IDを入力",
        "availableModelsHint": "複数モデルを有効化可能。",
        "availableModelsPlaceholder": "選択または入力",
        "availableModelsTip": "削除してもバインド済みエージェントは残ります。",
        "baseUrl": "基底 URL",
        "baseUrlHint": "（モデルサービスアドレス）",
        "clear": "クリア",
        "defaultModel": "デフォルトモデル",
        "deleteApiKey": "キー削除",
        "enabledOff": "無効→有効化",
        "enabledOn": "有効→無効化",
        "enabledStatus": "有効化",
        "maxTokens": "最大 トークン 数",
        "model": "モデル",
        "name": "名前",
        "provider": "AIプロバイダー",
        "temperature": "温度",
        "timeoutSeconds": "タイムアウト秒数"
      },
      "inferenceParams": {
        "title": "推論パラメータ"
      },
      "messages": {
        "apiKeyValidated": "API キー検証成功",
        "deleted": "削除済",
        "disabled": "無効化",
        "enabled": "有効化",
        "loadConfigFailed": "AI設定読込失敗",
        "probeFailed": "接続失敗",
        "probeSuccess": "接続成功",
        "saveSuccess": "設定保存成功",
        "selectSavedProfileOrEnterKey": "保存済設定選択またはKey入力",
        "setCurrentSuccess": "現在設定切替",
        "validateBeforeSave": "先に「API キー検証」をクリック",
        "validateFailed": "検証失敗",
        "validateSuccess": "検証成功"
      },
      "placeholders": {
        "apiKey": "API キー入力",
        "baseUrl": "例: https://api.example.com/v1",
        "modelManual": "モデル名入力",
        "modelSelect": "モデル選択",
        "modelSelectOrType": "選択または入力",
        "name": "例: DeepSeek-低コスト",
        "provider": "AIプロバイダー選択",
        "providerFirst": "先にプロバイダー選択"
      },
      "primary": {
        "hint": "意図明確化、コード生成などに使用。",
        "placeholder": "プロバイダー・モデルを選択してください",
        "title": "デフォルト主モデル"
      },
      "providers": {
        "anthropic": "Anthropic Claude",
        "custom": "カスタム（OpenAI互換）",
        "deepseek": "DeepSeek",
        "doubao": "Doubao",
        "emptyHint": "请先在 ",
        "emptyHintTail": "。",
        "emptyTitle": "有効プロバイダーなし",
        "enabledTitle": "有効プロバイダー",
        "groq": "Groq",
        "mistral": "Mistral",
        "modelsUnit": "モデル",
        "moonshot": "月之暗面（Kimi）",
        "noModels": "モデル未設定",
        "openai": "OpenAI",
        "openai_compatible": "カスタム（OpenAI互換）",
        "openrouter": "OpenRouter",
        "qwen": "Tongyi Qianwen",
        "siliconflow": "硅基流动 SiliconFlow",
        "zhipu": "智譜AI"
      },
      "sections": {
        "advanced": "高度パラメータ",
        "advancedHint": "意味を理解して調整。",
        "basic": "基本情報",
        "connection": "接続設定",
        "connectionApiKeyLink": "API キー申請/管理へ"
      },
      "tabs": {
        "agents": "エージェント設定",
        "config": "モデル設定"
      },
      "validation": {
        "apiKeyRequired": "API キー必須",
        "baseUrlNoChatCompletionsSuffix": "/chat/completionsで終了不可",
        "baseUrlProtocol": "http://またはhttps://で開始",
        "baseUrlRequired": "基底 URL必須",
        "modelFormat": "モデル形式不正",
        "modelRequired": "モデル必須",
        "nameRequired": "名前必須"
      },
      "apiKeySavedAs": "現在保存済: {{masked}}",
      "defaultProfileName": "デフォルト",
      "pageTitle": "AI アシスタント設定"
    }
  }
} as const;
export default AiSettings;
