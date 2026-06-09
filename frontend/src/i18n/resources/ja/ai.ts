const aiCore = {
  ai: {
    client: {
      errors: {
        requestFailed: 'リクエストに失敗しました。もう一度お試しください。',
        insufficientBalance: 'プロバイダーからの返答：残高不足または支払い延滞。プロバイダーコンソールで残高・請求を確認し、再度お試しください。',
        rateLimited: 'プロバイダーによるレート制限（リクエストが頻繁すぎます）。しばらくしてから再度お試しください。',
        unauthorized: 'プロバイダーから 401 未認証：API キー が正しいか、モデル権限があるかを確認してください。',
        forbidden: 'プロバイダーから 403 アクセス拒否：Key の権限、IP ホワイトリスト、アカウント状態を確認してください。',
        invalidModelId: 'モデル利用不可{{model}}：存在しない、廃止された、または権限外の可能性があります。ドロップダウンから選ぶか、プロバイダーコンソールから正しい モデル ID をコピーしてください。',
        contextTooLong: 'リクエストがモデルの最大コンテキスト長を超えています。会話履歴・入力を短縮するか、より大きなコンテキストウィンドウのモデルを選んでください。',
        contentBlocked: '内容がプロバイダーのセーフティポリシーによりブロックされました。質問の言い回しを調整して再度お試しください。',
        regionNotSupported: '現在の地域・国はこのプロバイダーではサポートされていません。地域を変更するか、別のプロバイダーを選択してください。',
        providerInternalError: 'プロバイダーサービスが一時的に利用不可（5xx）。しばらくしてから再度お試しください、または別のプロバイダーに切り替えてください。',
        edgeGatewayTimeout: 'フロントの前段ゲートウェイがタイムアウトしました（Cloudflare の HTTP 524 など）。長時間の「コード生成」で起きやすいです。ディベートのコード画面で「コード生成を再試行」するか、一つ戻ってから再度進めてください。改善しない場合はプロキシ／オリジンのタイムアウト延長が必要です。',
        networkUnreachable: 'モデルゲートウェイへの接続がタイムアウトまたは到達不可。AI 設定の 基底 URL がアクセス可能か、ネットワークが正常かを確認し、またはしばらくしてから再度お試しください。',
        gatewayTimeoutOrUnreachable: 'ゲートウェイ接続タイムアウト/到達不可。',
        gatewayUnauthorized401: 'ゲートウェイ未認証（401）。',
        gatewayForbidden403: 'ゲートウェイアクセス拒否（403）。',
        gatewayRateLimited429: 'ゲートウェイレート制限（429）。'
      }
    },
    chatBox: {
      emptyDescription: 'AI アシスタントと会話を開始',
      thinking: '考え中...',
      truncated: '内容が長すぎるため切り詰めました',
      expandAll: 'すべて展開',
      collapse: '折りたたむ'
    },
    systemAI: {
      taglines: {
        openai: 'GPT シリーズ · 公式',
        anthropic: 'Claude シリーズ',
        deepseek: 'DeepSeek · 高コストパフォーマンス',
        moonshot: 'Kimi · 長文コンテキスト',
        qwen: 'Alibaba Cloud · 中国語最適化',
        zhipu: '清華系 · 汎用',
        openai_compatible: '任意の互換エンドポイント'
      },
      pageTitle: 'AI アシスタント設定',
      pageSubtitle: 'AI の中枢を設定します — モデルプロバイダー、API キー、利用可能モデルを管理し、サイト全体の既定主モデルを指定します。',
      emptyConfigs: 'AI プロバイダー 設定がありません（システム起動時に既定 プロバイダー が自動作成されます）',
      section1: {
        title: 'モデルプロバイダーを選択',
        subtitle: '各プロバイダーの設定と準備状態をカードで表示します。クリックして選択してください'
      },
      statusBar: {
        enabled: '有効',
        disabled: '無効',
        keyReady: 'キー準備完了',
        checking: '接続確認中…',
        connected: '接続正常'
      },
      status: {
        noprovider: 'プロバイダー未選択',
        noProviderDesc: '下のカードからモデルプロバイダーを選んで設定を開始してください',
        error: '異常があります',
        ready: '利用準備完了',
        readyDesc: '有効化済みで接続正常です',
        notEnabled: '接続正常、まだ無効です',
        notEnabledDesc: '「有効」スイッチをオンにすると使用できます',
        configReady: '設定準備完了',
        configReadyDesc: '利用可能モデルを追加すると接続確認が自動実行されます',
        checkUrl: '基底 URL を確認してください',
        checkUrlDesc: 'API キーは準備済みですが、アドレスが無効の可能性があります',
        needKey: 'キー設定を完了してください',
        needKeyDesc: 'API キーを入力するとモデル一覧を自動検出します',
        connectionFailed: '接続異常です。上の提示を確認してください',
        noProvider: 'プロバイダーが未選択です'
      },
      cardState: {
        noKey: '未設定',
        noModel: 'モデル選択待ち',
        enabled: '有効',
        readyDisabled: '準備完了 · 無効'
      },
      cardTags: {
        current: '現在',
        hasKey: 'キー設定済み',
        noKey: 'キー未設定',
        noModels: '利用可能モデル未設定',
        enabledButUnavailable: '有効だが利用不可'
      },
      fields: {
        autoFetching: '自動取得中',
        baseUrlCustomHint: 'OpenAI 互換エンドポイントを入力してください。例：https://model.example.com/v1',
        baseUrlReadonlyHint: '公式アドレスはシステム管理のため変更できません',
        baseUrlCustomPlaceholder: '例：https://model.example.com/v1',
        baseUrlReadonlyPlaceholder: '公式アドレス（読み取り専用）',
        httpWarning: '現在 HTTP です。本番環境では HTTPS を推奨します',
        apiKeyHint: '入力後は自動で暗号化保存されます。手動送信は不要です',
        apiKeyPastePlaceholder: 'API キーを貼り付けると自動で仮保存されます',
        enabledHint: '無効にするとこのプロバイダーはシステムルーティングに参加しません',
        temperatureHint: '高いほど発散的、低いほど安定します',
        timeoutHint: '1 回のリクエストの最大待機時間',
        maxTokensHint: '1 回の応答の最大トークン数',
        primaryFor: '主な用途',
        primaryForHint: 'サービス内部ルーティング専用：チャット / 埋め込み / 要約 / 推論'
      },
      messages: {
        loadConfigFailed: '設定の読み込みに失敗しました',
        secretSavedAutoDiscover: 'シークレットを保存しました。モデルを自動検出中...',
        secretAutoSaveFailed: 'シークレットの自動保存に失敗しました',
        autoDiscoveredModels: '{{count}}個のモデルを自動検出しました（参考用）',
        autoValidatedModels: '自動検証完了：{{count}}個のモデルが見つかりました',
        configSaved: '設定を保存しました',
        configSaveFailed: '設定の保存に失敗しました',
        toggleEnabledFailed: '有効状態の切り替えに失敗しました',
        secretDeletedConfigReset: 'シークレットを削除しました。プロバイダー設定をデフォルトにリセットしました',
        deleteSecretFailed: 'シークレットの削除に失敗しました',
        validationPassedModels: '検証完了：{{count}}個のモデルが見つかりました',
        validationFailedNeedApiKey: '検証に失敗しました：このプロバイダーは通常APIキーが必要です。キーを入力して保存した後、再試行してください。'
      },
      customProvider: {
        deleted: '自定义提供商已删除',
        fillNameFirst: '请先填写名称',
        nameHint: '用于识别此提供商的唯一名称',
        nameLabel: '提供商名称',
        namePlaceholder: '我的自定义提供商',
        nameRequired: '提供商名称不能为空'
      }
    },
    tabs: {
      settings: '設定',
      agentSettings: 'エージェント設定',
      gate: 'AIゲート'
    },
    agentPrompts: {
      style: {
        title: '相場環境・スタイル推奨',
        prompt: `あなたはシニア定量ストラテジーアナリストです。以下の情報に基づき、トレンド/平均回帰/短期売買の戦略パラダイムを推奨し、その理由、適用条件、不適用シナリオを説明してください。

出力要件：Markdownを使用し、以下を含めてください：
1) 推論プロセス：データ/制約/目標からどのように導き出したか（箇条書き）
2) 結論：主推奨（1つの主要パラダイムのみ）+ 代替案 + 適用/不適用条件
3) リスクアラート：最低3つ

{{baseInfo}}`
      },
      signals: {
        title: 'シグナルとインジケーター設計',
        prompt: `あなたは定量ファクター・シグナルエンジニアです。外部データに依存せず（ユーザーがマクロイベントテーブルを提供する場合を除く）、実用的な取引シグナルを設計してください。

要件：エントリー/エグジット/フィルター条件を明確に定義し、可能な限りパラメーター化し、オーバーフィッティングを避けてください。

出力要件：Markdownを使用し、以下を含めてください：
1) 推論プロセス：なぜこれらのインジケーター/しきい値/フィルター条件を選択したか（箇条書き）
2) 結論：実行可能なルール一覧（エントリー/エグジット/フィルター）、パラメーター提案（デフォルト/範囲）
3) 限界とリスク：最低3つ（例：レンジ相場/ギャップ/高ボラティリティ/ニュースイベント）

{{baseInfo}}`
      },
      risk: {
        title: 'リスク管理と執行制約',
        prompt: `あなたは取引リスクと執行の専門家です。以下の情報に基づき、ポジション管理、ストップロス/テイクプロフィット、最大ドローダウン管理、クールダウン期間/取引頻度制限などを設計してください。

出力要件：Markdownを使用し、以下を含めてください：
1) 推論プロセス：なぜこれらの制御が目標/制約に適合するか（箇条書き）
2) 結論：ハード制約 + デフォルトパラメーター（推奨値/範囲）+ トリガー後のアクション
3) 障害モード：最低3つ（例：連続損失、スリッページ拡大、スプレッド異常）

{{baseInfo}}`
      },
      code: {
        title: 'コード生成エージェント',
        prompt: `あなたはAntTraderのPythonストラテジーコードエンジニアです。以下の条件を満たす実行可能なAntTrader Pythonストラテジーコードを生成してください：
- バリデーションチェックに合格（import不可、dunder不可、サンドボックス制約）
- on_tick/on_klineなどのプラットフォームAPIを使用（カスタムネットワーク/ファイルアクセス不可）
- run()は引数を1つだけ受け取る：context（名前はcontextでなければなりません。run(ctx)、run(context, data)などは不可）
- run(context)は以下を含むdictを返す：signal(buy/sell/hold)、symbol、confidence(0~1)、risk_level(low/medium/high)、reason
- context["params"]からパラメーターを読み取り（スケジュール注入から）、欠落している場合はデフォルト値を使用
- 上流のシグナル設計とリスク制御を使用（提供されない場合は適切なデフォルト値を設定）
- 完全なコードを \`\`\`python でラップして出力
- 厳密な出力：\`\`\`python ブロック1つのみ、説明文なし
- コードブロックは純粋なPython：Markdown記号不可、中国語の句読点不可、ネストされたコードフェンス不可

[必須テンプレート（関数名/パラメーター数/パラメーター名は変更不可）]
\`\`\`python
def run(context):
    params = context.get("params") or {}
    symbol = context.get("symbol") or params.get("symbol") or ""
    # TODO: シグナル/リスクロジックをここに実装
    return {
        "signal": "hold",
        "symbol": symbol,
        "confidence": 0.5,
        "risk_level": "low",
        "reason": "",
    }
\`\`\`

{{baseInfo}}

[注記：上流分析の結論 – コードに適用（欠落している場合は適切なデフォルト値を設定）]`
      }
    },
    consensus: {
      title: 'コンセンサスと議論',
      actions: {
        refresh: '更新'
      },
      fields: {
        account: '口座',
        symbol: '銘柄',
        timeframe: '時間足'
      },
      panel: {
        title: '目標スコア',
        decision: '判定',
        overallScore: '総合',
        technicalScore: 'テクニカル'
      },
      signals: {
        rsi: {
          value: 'RSI',
          flag: 'シグナル'
        },
        macd: {
          value: 'MACD',
          signalLine: 'シグナル線',
          hist: 'ヒストグラム',
          flag: 'シグナル',
          trend: 'パターン'
        },
        ma: {
          trend: 'MAトレンド'
        }
      }
    },
    conversation: {
      defaultTitle: '新しい会話'
    },
    reports: {
      tradeAnalysis: {
        title: 'AI取引分析レポート',
        riskAssessmentPrefix: 'リスク評価：'
      }
    },
    signalCard: {
      status: {
        pending: '保留中',
        confirmed: '確認済み',
        executed: '執行済み',
        cancelled: 'キャンセル済み'
      },
      labels: {
        price: '価格',
        volume: 'ロット',
        confidence: '確信度',
        stopLoss: 'ストップロス',
        takeProfit: 'テイクプロフィット',
        analysisReason: '分析理由'
      },
      actions: {
        confirm: '確認',
        cancel: 'キャンセル',
        executeTrade: '取引を執行'
      },
      confirmCancel: {
        title: 'このシグナルをキャンセルしてもよろしいですか？'
      },
      confirmExecute: {
        title: 'この取引シグナルを執行してもよろしいですか？',
        description: 'すぐに注文を発注します'
      }
    },
    assistant: {
      messages: {
        noCodeBlockFound: 'コードブロックが見つかりませんでした（\`\`\`...\`\`\`）'
      }
    },
    strategyCard: {
      status: {
        active: '稼働中',
        inactive: '停止中',
        paused: '一時停止中'
      },
      actionType: {
        buy: '買い',
        sell: '売り',
        closeLong: '買い決済',
        closeShort: '売り決済',
        alert: 'アラート'
      },
      labels: {
        triggeredCount: '{{count}}回トリガー',
        lastTriggeredAt: '最終トリガー：{{time}}'
      },
      sections: {
        conditions: 'トリガー条件',
        actions: 'アクション'
      },
      tooltips: {
        createdAt: '作成日時',
        lastTriggeredAt: '最終トリガー日時'
      },
      actions: {
        start: '開始',
        stop: '停止'
      },
      confirmDelete: {
        title: 'このストラテジーを削除してもよろしいですか？',
        description: '削除後は復元できません'
      }
    },
    requireConfig: {
      title: 'LLMがまだ設定されていません',
      description: '先に設定画面でAIプロバイダー、モデル、APIキーを設定してください。その後、ストラテジーウィザードまたはチャットをご利用いただけます。',
      actions: {
        goSettings: '設定へ'
      }
    },
    riskEval: {
      failed: 'リスク評価に失敗しました'
    },
    workflowRuns: {
      title: 'AIワークフロー',
      defaultTitle: 'AIワークフロー',
      hints: {
        selectToViewDetail: '左側から実行を選択して詳細を表示'
      },
      messages: {
        loadListFailed: '実行一覧の読み込みに失敗しました',
        loadDetailFailed: '詳細の読み込みに失敗しました'
      }
    },
    backtestScoreCard: {
      title: 'バックテストスコアカード',
      stateLabel: '状態',
      status: {
        succeeded: '成功',
        running: '実行中',
        pending: 'キュー待ち',
        failed: '失敗',
        cancelRequested: 'キャンセル中',
        canceled: 'キャンセル済み'
      },
      recommendation: {
        loading: 'リスク評価中です。完了するまでお待ちください。',
        recommended: '本番推奨：リスク管理可能、指標良好。',
        cautious: '本番注意：まずは少額または手動確認でしばらく運用してください。',
        notRecommended: '直接の本番運用非推奨：高リスクまたは信頼性に欠けます。最適化後に再試行してください。'
      },
      backendRiskScore: {
        title: 'バックエンドリスクスコア',
        loading: '計算中...',
        unknown: '不明',
        reliable: '信頼できる',
        unreliable: '信頼できない',
        reasons: '理由',
        warnings: '警告',
        empty: 'なし（先にテンプレートを保存してください。バックテスト完了後に自動計算されます）'
      },
      score: {
        empty: 'スコア未評価（バックテスト待ちまたは指標なし）',
        title: '総合スコア（ヒューリスティック）'
      },
      level: {
        excellent: '優秀',
        good: '良好',
        fair: '普通',
        poor: '劣る'
      },
      metrics: {
        totalReturn: '総収益率',
        annualReturn: '年換算収益率',
        maxDrawdown: '最大ドローダウン',
        sharpe: 'シャープレシオ',
        winRate: '勝率',
        totalTrades: '総取引数',
        equityPoints: 'エクイティポイント'
      },
      chart: {
        title: 'エクイティカーブ'
      }
    },
    gate: {
      title: 'AIゲート進捗',
      pipelineDesc: '6段階ゲートパイプライン：コンプライアンス → 先読みバイアス → ウォークフォワード → 収縮シャープ → ペーパー → 相関',
      labels: {
        compliance: 'コンプライアンス',
        lookahead: '先読みバイアス',
        walkforward: 'ウォークフォワード',
        deflated_sharpe: '収縮シャープレシオ',
        paper: 'ペーパートレーディング',
        correlation: '相関'
      },
      descriptions: {
        compliance: 'DSL式の空チェック',
        lookahead: '未来関数参照のスキャン（close[t+N]、ref負のオフセット）',
        walkforward: 'パージド・ウォークフォワード交差検証',
        deflated_sharpe: 'Lopez de Prado 収縮シャープレシオ',
        paper: '14日以上のペーパートレーディング検証',
        correlation: '既存ストラテジーとのシグナル相関チェック'
      },
      status: {
        evaluating: '評価中...'
      },
      strategyParams: 'ストラテジーパラメーター',
      dslExpression: 'DSL式',
      dailyReturns: '日次リターン（カンマまたは改行区切り）',
      numAttempts: 'ストラテジー試行回数',
      paperMetrics: 'ペーパートレーディング指標',
      paperDays: 'ペーパー日数',
      paperNetPnL: 'ペーパー純損益',
      paperNetReturn: 'ペーパー純収益率',
      paperTradeCount: 'ペーパー取引回数',
      backtestNetReturn: 'バックテスト純収益率',
      backtestGrossReturn: 'バックテスト総収益率',
      runPipeline: 'ゲートパイプライン実行',
      retry: '再試行',
      gateProgress: 'ゲート評価進捗',
      pipelineResult: 'パイプライン結果',
      allPassed: '全6ゲートを通過 — ストラテジーは本番昇格評価の対象です',
      failed: '不合格：{{gate}}',
      details: '詳細',
      skipped: 'SKIPPED',
      noData: 'データがありません',
      pass: 'PASS',
      fail: 'FAIL',
      unknown: '不明',
      selectRun: 'バックテスト実行を選択...',
      evaluating: '评估中...',
      runHint: '先运行回测，然后点击"运行Gate"评估策略质量。'
    }
  }
} as const;

export default aiCore;
