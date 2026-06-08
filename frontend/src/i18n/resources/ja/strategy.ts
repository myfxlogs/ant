const strategy = {
  strategy: {
    validation: {
      passed: 'コード検証に成功しました',
      notPassed: 'コード検証に失敗しました',
      riskEval: {
        title: 'Risk Assessment',
        riskHigh: 'Risk level: high',
        riskUnreliable: 'Risk assessment: unreliable (isReliable=false)',
        riskLoading: 'Backend risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: '戦略エディタ',
      labels: {
        account: '口座',
        symbol: '銘柄',
        timeframe: '時間足',
        code: '戦略コード',
        disabledSuffix: '（無効）'
      },
      actions: {
        copy: 'コピー',
        validate: 'コード検証',
        preview: 'シグナルプレビュー',
        saveAsTemplate: 'テンプレートとして保存',
        sendToAI: 'AI に修正を依頼',
        sendToAIFixTitleValidate: '検証失敗 / 警告あり',
        sendToAIFixTitlePreview: 'プレビュー失敗 / 改善が必要'
      },
      placeholders: {
        selectAccount: '口座を選択',
        selectAccountFirst: '先に口座を選択してください',
        loadingSymbols: '銘柄を読み込み中...',
        selectSymbol: '銘柄を選択',
        noSymbols: '銘柄一覧を取得できませんでした',
        code: 'Python 戦略コードを入力...'
      },
      cards: {
        validationResult: '検証結果',
        previewResult: 'プレビュー結果'
      },
      hints: {
        previewInfo: 'プレビューは直近 N 本のローソク足を使用します（既定 500、設定: strategy.preview_bars）。バックテストは直近 N か月（既定 3、設定: strategy.backtest_window_months）。'
      },
      messages: {
        enterCode: '戦略コードを入力してください',
        selectAccount: '口座を選択してください',
        validateOk: '検証に成功しました',
        validateFailed: '検証に失敗しました',
        validateError: '検証エラー',
        previewOk: 'プレビューが完了しました',
        previewSuccess: 'プレビューに成功しました',
        previewFailed: 'プレビューに失敗しました',
        execFailed: '実行に失敗しました',
        savedAsTemplate: 'テンプレートとして保存しました',
        copied: 'コピーしました',
        copyFailed: 'コピーに失敗しました。手動でコピーしてください'
      },
      aiPrompt: {
        intro: '以下の情報に基づいて戦略コードを修正し、検証に通り、プレビュー実行が成功するようにしてください。',
        problem: '【問題】{{title}}',
        currentCodeTitle: '【現在のコード】',
        outputTitle: '【出力】',
        outro: '修正後の完全なコードを \`\`\`python で囲って出力し、変更点を説明してください。',
        pythonFenceStart: '\`\`\`python',
        fenceEnd: '\`\`\`'
      }
    },
    schedules: {
      title: '戦略スケジュール',
      actions: {
        create: 'スケジュール作成',
        logs: 'ログ',
        healthCheck: 'ヘルスチェック',
        runNow: '今すぐ実行'
      },
      health: {
        title: '戦略ヘルスチェック {{name}}',
        summaryBanner: 'ヘルス評価: {{grade}}、サンプル {{totalRuns}} 件、成功率 {{successRate}}%',
        grade: {
          pending: '未評価',
          noSample: 'サンプル不足',
          healthy: '健全',
          watch: '要注意',
          alert: '警告'
        },
        notes: {
          pending: 'まずヘルスチェックを実行してください。',
          noSample: '評価に必要なサンプル不足（最低 {{minSampleSize}} 件）。',
          healthy: '成功率が高く、失敗回数も許容範囲です。',
          watch: '成功率は監視対象です（>= {{yellowSuccessRate}}%）。',
          alert: '成功率が低いです。戦略と口座状態をすぐ確認してください。'
        },
        fields: {
          grade: 'ヘルス評価',
          rule: '判定基準',
          thresholds: '現在の閾値',
          configKey: '設定キー',
          lastRunAt: '最終実行',
          latestTicket: '最新約定チケット',
          successOverTotal: '成功 / 総数',
          failedRuns: '失敗回数',
          latestProfit: '最新損益',
          latestError: '最新エラー'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}、緑: 成功率>={{greenSuccessRate}}% かつ 失敗<={{greenMaxFailedRuns}}、黄: 成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近の実行ログ',
          orders: '最近の約定履歴'
        },
        runLogs: {
          signalType: 'シグナル'
        },
        messages: {
          loadFailed: 'ヘルスデータの読み込みに失敗しました',
          clickRefresh: '更新を押してヘルスデータを読み込んでください'
        }
      },
      editModal: {
        title: {
          create: 'スケジュール作成',
          edit: 'スケジュール編集'
        },
        autoName: {
          strategy: '戦略'
        },
        fields: {
          template: 'テンプレート',
          templateExtra: '「戦略管理」に保存されたテンプレート',
          account: '口座',
          name: '名称',
          symbol: '銘柄',
          lot: 'ロット',
          lotExtra: '数量。0.01 からの開始を推奨',
          runFrequency: '実行頻度',
          cronExpression: 'Cron 式',
          cronExtra: '標準の5項：分 時 日 月 週。例：*/5 * * * *；0 9 * * 1-5',
          intervalSeconds: '間隔(秒)',
          intervalSecondsExtra: '時間足に自動追従。変更不要',
          enableExtra: 'EAのように：有効化すると手動で停止するまで継続実行'
        },
        placeholders: {
          name: '例：EURUSD M5 朝の戦略',
          selectAccountFirst: '先に口座を選択',
          symbol: '銘柄を選択'
        },
        validation: {
          templateRequired: 'テンプレートを選択してください',
          accountRequired: '口座を選択してください',
          nameRequired: '名称を入力してください',
          symbolRequired: '銘柄を選択してください',
          lotRequired: 'ロットを入力してください',
          runFrequencyRequired: '実行頻度を選択してください',
          cronRequired: 'cron を入力してください',
          timeframeRequired: '時間足を選択してください',
          triggerModeRequired: 'トリガーモードを選択してください'
        },
        runFrequencyExtra: {
          cron: '高度：Cron で実行時間を精密に制御',
          byTimeframe: '既定：時間足に従ってトリガー（EAに近い挙動）'
        },
        runFrequencyOptions: {
          byTimeframe: '時間足でトリガー（推奨）',
          cron: 'Cron（高度）'
        },
        advanced: {
          title: '詳細設定',
          fixedIntervalSeconds: '固定間隔(秒)',
          fixedIntervalSecondsExtra: '任意。固定間隔で実行（時間足追従しない）。例：60 は60秒ごと',
          timeframe: '時間足',
          timeframeExtra: 'ローソク/指標計算に使用',
          triggerMode: 'トリガーモード',
          triggerModeExtra: '安定：ローソク/周期（ノイズ少・遅延あり）；高頻度：クオート（速い・デバウンス必要）',
          triggerModeOptions: {
            stable: '安定（ローソク/周期）',
            hf: '高頻度（クオート/tick）'
          },
          stableOverrideIntervalSeconds: '安定モード上書き間隔(秒)',
          stableOverrideIntervalSecondsExtra: '任意。安定モードの間隔を上書き',
          hfCooldownMs: '高頻度クールダウン(ms)',
          hfCooldownMsExtra: 'デバウンス：評価/発注の最小間隔',
          parametersJson: 'パラメータ(JSONオブジェクト)',
          parametersJsonExtra: '戦略コードへパラメータ渡し（文字列）。例：{ "fast": 10, "slow": 20, "risk": "low" }'
        }
      },
      triggerModal: {
        title: '今すぐ実行（即時発注）',
        actions: {
          rerun: '再実行',
          confirmOrder: '発注する'
        },
        confirmOrder: {
          title: '発注しますか？',
          ok: '確定'
        },
        summary: {
          scheduleName: 'スケジュール名',
          account: '口座',
          symbol: '銘柄',
          timeframe: '時間足'
        },
        messages: {
          signalNotOrderable: 'このシグナルは発注できません（買い/売り かつ 数量 > 0 が必要）'
        },
        cards: {
          logs: '実行ログ',
          signal: 'シグナル（発注用）'
        },
        emptyLogs: '(ログなし)',
        emptySignal: '(シグナルなし)'
      },
      table: {
        name: '名称',
        template: 'テンプレート',
        account: '口座',
        tradeParams: '取引パラメータ',
        schedule: 'スケジュール',
        status: '状態',
        lastRun: '最終実行',
        actions: '操作'
      },
      templateVisibility: {
        public: '公開',
        private: '非公開'
      },
      status: {
        running: '実行中',
        disabled: '停止'
      },
      nextRunAt: '次回実行',
      enableCount: '有効化回数',
      deleteConfirm: {
        title: 'このスケジュールを削除しますか？'
      },
      validation: {
        parametersMustBeJsonObject: 'パラメータは JSON オブジェクトである必要があります'
      },
      messages: {
        parametersParseFailed: 'パラメータ解析に失敗しました',
        defaultTemplateNotFound: 'デフォルトテンプレートが見つかりません。更新して再試行してください。',
        importDefaultTemplateFailedNoId: 'デフォルトテンプレートの取り込みに失敗しました（IDがありません）',
        templateCodeEmptyCannotExecute: 'テンプレートコードが空です。実行できません。',
        executeFailed: '実行に失敗しました',
        strategyExecuteFailed: '戦略実行に失敗しました',
        noOrderableSignal: '発注可能なシグナルがありません',
        signalHoldCannotOrder: 'シグナルが保留/無操作のため発注できません',
        volumeInvalid: '数量が不正です（> 0）',
        orderSubmitted: '注文を送信しました',
        orderFailed: '注文に失敗しました'
      }
    },
    scheduleLogs: {
      title: '記録',
      titleWithName: '記録 - {{name}}',
      tabs: {
        exec: '実行履歴',
        orders: '取引履歴'
      },
      messages: {
        missingScheduleId: 'scheduleId がありません'
      },
      summary: {
        name: '名称',
        status: '状態',
        trade: '取引',
        enableCount: '有効化回数',
        lastRun: '最終実行',
        lastError: '最新エラー'
      },
      execStatus: {
        pending: '待機',
        running: '実行中',
        completed: '完了',
        failed: '失敗',
        skipped: 'スキップ'
      },
      operationStatus: {
        success: '成功',
        failed: '失敗',
        running: '実行中'
      },
      execTable: {
        time: '時間',
        action: '操作',
        status: '状態',
        durationMs: '所要(ms)',
        error: 'エラー',
        execute: '実行'
      },
      ordersTable: {
        time: '時間',
        side: '方向',
        symbol: '銘柄',
        lots: '数量',
        openPrice: '建値',
        closePrice: '決済値',
        profit: '損益',
        ticket: 'チケット'
      },
      orderSide: {
        buy: '成行買い',
        sell: '成行売り',
        close: '決済',
        buyLimit: '指値買い',
        sellLimit: '指値売り',
        buyStop: '逆指値買い',
        sellStop: '逆指値売り',
        buyStopLimit: 'ストップリミット買い',
        sellStopLimit: 'ストップリミット売り'
      },
      scheduleIdLabel: 'スケジュールID:'
    },
    templates: {
      title: '戦略テンプレート',
      tabs: {
        system: 'システムテンプレート',
        user: 'ユーザーテンプレート'
      },
      copySuffix: '（コピー）',
      scheduleName: '{{symbol}} {{timeframe}} {{nowText}}',
      badges: {
        preset: 'プリセット'
      },
      scheduleLaunch: {
        title: 'スケジュール起動',
        noRun: 'バックテスト実行がありません',
        backtestRunningHint: 'バックテスト実行中です。しばらくお待ちください。',
        score: 'スコア',
        keyMetrics: '主要指標',
        launchSection: 'スケジュール起動',
        actions: {
          publishTemplate: 'テンプレートを公開',
          createScheduleNoEnable: 'スケジュール作成',
          createAndEnable: '作成して有効化'
        },
        metrics: {
          totalReturn: '総リターン',
          annualReturn: '年率リターン',
          maxDrawdown: '最大ドローダウン',
          sharpe: 'シャープレシオ',
          winRate: '勝率',
          totalTrades: '取引回数'
        }
      },
      visibility: {
        public: '公開',
        private: '非公開'
      },
      status: {
        draft: '下書き',
        published: '公開済み'
      },
      actions: {
        create: 'テンプレート作成',
        createTemplate: 'テンプレート作成',
        edit: '編集',
        delete: '削除',
        copy: 'コピー',
        viewCode: 'コード表示',
        backtest: 'バックテスト',
        launchSchedule: 'スケジュール起動'
      },
      table: {
        name: '名称',
        description: '説明',
        tags: 'タグ',
        visibility: '公開範囲',
        status: '状態',
        useCount: '使用回数',
        createdAt: '作成日時',
        updatedAt: '更新日時',
        actions: '操作',
        loadingDefault: 'Loading default templates...',
        defaultHint: 'Default',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      editTemplateModal: {
        title: {
          create: 'テンプレート作成',
          edit: 'テンプレート編集'
        },
        fields: {
          name: '名称',
          description: '説明',
          code: '戦略コード',
          publicShare: '公開共有'
        },
        placeholders: {
          name: '例：移動平均クロス戦略',
          description: '任意：説明',
          codeSample: `# 戦略コード例
# 利用可能: close, open, high, low, volume, symbol
# 戻り値: signal（dict）

import numpy as np

# 指標
maFast = np.mean(close[-10:])
maSlow = np.mean(close[-20:])

# シグナル
if maFast > maSlow:
    signal = '買い'
elif maFast < maSlow:
    signal = '売り'
else:
    signal = '保留'

# 結果
signal = {
    'signal': signal,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.7,
    'reason': f'fast={maFast:.5f}, slow={maSlow:.5f}'
}`
        },
        validation: {
          nameRequired: '名称を入力してください',
          codeRequired: '戦略コードを入力してください'
        },
        actions: {
          validateCode: 'コード検証'
        }
      },
      backtest: {
        title: 'バックテスト',
        fields: {
          title: 'タイトル',
          account: '口座',
          symbol: '銘柄',
          timeframe: '時間足',
          initialCapital: '初期資金',
          range: '期間',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        placeholders: {
          account: '口座を選択',
          symbol: '銘柄を選択',
          range: '期間を選択',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
        },
        validation: {
          accountRequired: '口座を選択してください',
          symbolRequired: '銘柄を選択してください',
          timeframeRequired: '時間足を選択してください',
          initialCapitalRequired: '初期資金を入力してください',
          rangeRequired: '期間を選択してください'
        },
        quickRange: {
          '1d': '1日',
          '3d': '3日',
          '1w': '1週間',
          '1y': '1年',
          custom: 'カスタム'
        },
        accountDisabledSuffix: '（無効）',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: 'Strategy Parameters'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
        }
      },
      messages: {
        fetchTemplateListFailed: 'テンプレート一覧の読み込みに失敗しました',
        enterStrategyCode: '戦略コードを入力してください',
        codeValidationPassed: 'コード検証に成功しました',
        codeValidationNotPassed: 'コード検証に通りませんでした',
        codeValidationFailed: 'コード検証に失敗しました',
        templateUpdated: 'テンプレートを更新しました',
        templateCreated: 'テンプレートを作成しました',
        templateDeleted: 'テンプレートを削除しました',
        readStrategyCodeFailed: '戦略コードの読み込みに失敗しました',
        strategyCodeEmptyCannotBacktest: '戦略コードが空です。バックテストできません。',
        selectBacktestRange: 'バックテスト期間を選択してください',
        backtestRangeInvalid: 'バックテスト期間が不正です',
        backtestSubmitted: 'バックテストを送信しました',
        backtestSubmitFailed: 'バックテストの送信に失敗しました',
        backtestCancelRequested: 'バックテストのキャンセルを要求しました',
        backtestCancelFailed: 'バックテストのキャンセルに失敗しました',
        backtestReportDeleted: 'バックテストレポートを削除しました',
        backtestReportNotFound: 'バックテストレポートが見つかりません',
        codeCopied: 'コードをコピーしました',
        copyFailed: 'コピーに失敗しました',
        missingScheduleInfo: 'スケジュール作成に必要な情報が不足しています',
        templateNotPublishedCannotCreateSchedule: 'テンプレートが公開されていないためスケジュールを作成できません',
        readTemplateStatusFailed: 'テンプレート状態の取得に失敗しました',
        scheduleCreated: 'スケジュールを作成しました',
        scheduleCreatedAndEnabled: 'スケジュールを作成して有効化しました',
        createScheduleFailed: 'スケジュール作成に失敗しました',
        deepLinkNavigate: 'Opened template and latest run details from external link',
        templatePublished: 'Template published',
        cannotPublishAndCreateDraftFailed: 'Unable to publish. Draft creation failed.',
        republishedButNoTemplateId: 'Republished, but template id is missing.',
        backtestRunningCannotPublish: 'Backtest is running. Cannot publish now.',
        missingDraftIdCannotPublish: 'Missing draft id. Cannot publish.',
        publishedButNoTemplateId: 'Published, but template id is missing.',
        templateRepublished: 'Template republished',
        templateAlreadyPublished: 'Template already published',
        templateNotDraftUnknownPublishStatus: 'Template is not a draft. Unknown publish status.',
        publishFailed: 'Publish failed',
        backtestRunNoPublishedTemplate: 'Backtest run has no published template'
      },
      backtestRuns: {
        title: 'バックテストレポート',
        empty: 'バックテスト記録がありません',
        deleteConfirm: 'このバックテストレポートを削除しますか？',
        status: {
          queued: '待機中',
          running: '実行中',
          completed: '完了',
          failed: '失敗',
          canceling: 'キャンセル中',
          canceled: 'キャンセル済み'
        },
        table: {
          title: 'タイトル',
          status: '状態',
          symbol: '銘柄',
          timeframe: '時間足',
          createdAt: '作成日時',
          actions: '操作'
        },
        actions: {
          view: '表示',
          launchSchedule: 'スケジュール起動',
          createSchedule: 'スケジュール作成'
        }
      },
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      codeModal: {
        title: 'Strategy code',
        actions: {
          copy: 'Copy'
        }
      }
    },
    defaultTemplates: {
      maCross: {
        name: '移動平均クロス',
        description: '短期MAが長期MAを上抜けで買い、下抜けで売り'
      },
      forceBuy: {
        name: '強制買い（テスト）',
        description: '発注パイプライン検証用：常に買い。context/params のロットを数量として使用'
      },
      rsi: {
        name: 'RSI 過熱（買われすぎ/売られすぎ）',
        description: 'RSI < 30 で買い、RSI > 70 で売り'
      },
      macd: {
        name: 'MACD クロス',
        description: 'MACD のゴールデンクロスで買い、デッドクロスで売り'
      }
    },
    codeAssist: {
      tabAI: 'AI 修正',
      tabExplain: 'コード解説',
      explain: 'コードを解説',
      requiredParamsTitle: '必須パラメータ',
      requiredParamsDesc: '戦略コードがこれらのパラメータを参照していますが、デフォルト値が設定されていません。保存前に入力してください。',
      optionalParamsTitle: '任意パラメータ',
      optionalParamsDesc: 'これらのパラメータはコード内にデフォルト値があります。空欄にするとデフォルトを使用し、値を入力した場合は今回の実行のみに適用され、保存済みの戦略は変更されません。',
      defaultLabel: 'デフォルト',
      paramDescriptions: {
        riskLevel: 'リスクレベル（low / medium / high）。建玉サイズと損切り・利確幅に影響します。',
        takeProfit: '利確幅（%）。価格が建値からこの割合だけ有利方向へ動くと利確します。',
        stopLoss: '損切り幅（%）。価格が建値からこの割合だけ不利方向へ動くと損切りします。',
        maxLoss: '1 取引あたりの最大損失（口座資金に対する比率、0.01 = 1%）。',
        confidence: 'シグナル信頼度のしきい値（0〜1）。これ未満のシグナルは無視されます。',
        threshold: 'シグナル発生のしきい値。具体的な意味はコードのロジックに依存します。',
        lotSize: '注文ロット / 数量。大きいほどリスクが高くなります。',
        fastPeriod: '短期周期（バー本数）。MACD / 二重移動平均で使用。小さいほど敏感です。',
        slowPeriod: '長期周期（バー本数）。MACD / 二重移動平均で使用。大きいほど滑らかです。',
        signalPeriod: 'シグナル周期（バー本数）。MACD の DIF/DEA を平滑化する期間です。',
        rsiPeriod: 'RSI の計算期間（バー本数）。一般的な値は 14。',
        emaPeriod: 'EMA（指数移動平均）の期間（バー本数）。',
        smaPeriod: 'SMA（単純移動平均）の期間（バー本数）。',
        genericPeriod: '指標計算に使う参照期間（バー本数）。',
        genericPercent: 'パーセンテージ / 比率系パラメータ（例：1 は 1%）。'
      },
      required: '必須',
      suggested: '推奨',
      applyAllSuggestions: '推奨値を一括入力',
      fillRequiredParams: '必須パラメータを入力してください：{{keys}}',
      aiReviseTitle: 'AI アシスタント — コード修正',
      reviseInputPlaceholder: '例：SMA(20) を EMA(50) に置き換え、1% の損切りを追加。',
      reviseSend: 'AI に送信',
      enterInstruction: '変更内容を入力してください。',
      codeEmpty: '修正できるコードがありません。',
      codeUpdated: 'コードが更新されました。保存前に再度コード検証を実行してください。',
      noPython: 'AI が Python コードブロックを返しませんでした。表現を変えて再試行してください。',
      saveBlockedNotValidated: 'まず「コード検証」を実行してください。検証に合格するまで保存は無効です。'
    },
    templateModal: {
      title: 'Save as template',
      fields: {
        name: 'Name',
        description: 'Description'
      },
      placeholders: {
        name: 'Enter template name',
        description: 'Enter description'
      }
    },
    backtestRun: {
      title: 'Backtest run',
      status: {
        queued: 'Queued',
        running: 'Running',
        completed: 'Completed',
        failed: 'Failed',
        canceling: 'Canceling',
        canceled: 'Canceled',
        ended: 'Ended'
      },
      actions: {
        cancel: 'Cancel'
      },
      hints: {
        queued: 'Backtest is queued',
        running: 'Backtest is running',
        canceling: 'Canceling backtest'
      },
      fields: {
        status: 'Status',
        error: 'Error'
      },
      metrics: {
        totalReturn: 'Total return',
        annualReturn: 'Annual return',
        maxDrawdown: 'Max drawdown',
        sharpe: 'Sharpe ratio',
        winRate: 'Win rate',
        totalTrades: 'Total trades',
        equityCurvePoints: 'Equity curve points'
      },
      trades: {
        title: 'Order details',
        empty: 'No trades recorded',
        loadFailed: 'Failed to load order details',
        ticket: 'Ticket',
        side: 'Side',
        sideBuy: 'Buy',
        sideSell: 'Sell',
        volume: 'Volume',
        openTime: 'Open time',
        openPrice: 'Open price',
        closeTime: 'Close time',
        closePrice: 'Close price',
        pnl: 'P&L',
        commission: 'Commission',
        reason: 'Close reason',
        reasons: {
          signal: 'Signal',
          sl: 'Stop loss',
          tp: 'Take profit',
          margin_call: 'Margin call',
          expired: 'Expired',
          end_of_test: 'End of test'
        },
        summary: '{{count}} trades · {{wins}} wins / {{losses}} losses · net P&L {{pnl}}'
      }
    },
    asset: {
      title: 'Strategy Assets',
      subtitle: 'Asset publishing, review status, and cloning are maintained by the backend. Cloned results are independent user templates.',
      submitAsset: 'Submit Asset',
      assetList: 'Asset List',
      name: 'Name',
      visibility: 'Visibility',
      reviewStatus: 'Review Status',
      cloneCount: 'Clones',
      version: 'Version',
      description: 'Description',
      actions: 'Actions',
      cloneAsDraft: 'Clone as Draft',
      sourceTemplate: 'Source Template',
      assetName: 'Asset Name',
      submit: 'Submit',
      messages: {
        loadFailed: 'Failed to load strategy assets',
        submitSuccess: 'Strategy asset submitted',
        submitFailed: 'Failed to submit strategy asset',
        cloneSuccess: 'Cloned as template: {{templateId}}',
        cloneFailed: 'Failed to clone strategy asset'
      },
      validation: {
        selectTemplate: 'Please select a source template',
        enterName: 'Please enter asset name'
      }
    },
    gen: {
      title: 'Strategy Generation',
      send: 'Generate Strategy',
      regenerate: 'Regenerate',
      reset: 'Start Over',
      template: 'Template',
      generating: 'Generating...',
      validating: 'Compliance Check',
      backtestStarted: 'Backtest Started',
      done: 'Done',
      backtestMsg: 'Backtest task created',
      clarifyTitle: 'A few details to confirm:',
      useDefaults: 'Continue with defaults',
      placeholder: 'Describe the trading strategy you want to create, e.g.: "Make a Bollinger Band mean-reversion strategy for EURUSD on 1H"',
      chat: {
        generate: '⚡ Generate',
        revise: '✏️ Revise',
        repair: '🔧 Repair',
        discuss: '💬 Discuss'
      },
      feedback: {
        heading: '📊 Backtest Results',
        placeholder: 'Provide feedback to iterate (e.g. "Too aggressive", "Add stop loss")'
      }
    },
    marketRegime: {
      title: 'Market Regime Detection',
      subtitle: 'Backend computes trend, volatility, and efficiency features from K-lines. Frontend only displays results.',
      ruleVersionAlert: 'Currently using rule-based detection model rule-v1. K-line authoritative source remains the backend Market/Kline service.',
      detectSuccess: 'Market regime detection completed',
      detectFailed: 'Market regime detection failed',
      form: {
        title: 'Detection Parameters',
        accountId: 'Account ID',
        accountIdRequired: 'Account ID is required',
        accountIdPlaceholder: 'MT account UUID',
        symbol: 'Symbol',
        symbolRequired: 'Symbol is required',
        symbolPlaceholder: 'EURUSD',
        timeframe: 'Timeframe',
        klineCount: 'K-line Count',
        submit: 'Start Detection'
      },
      result: {
        title: 'Detection Result',
        status: 'Status',
        confidence: 'Confidence',
        modelVersion: 'Model Version',
        strategyFamilies: 'Strategy Families',
        features: 'Features',
        recordId: 'Record ID'
      }
    },
    experiment: {
      title: 'Strategy Experiment',
      subtitle: 'Parameter experimentation, candidate scoring, and draft generation are handled by the backend. Frontend only submits and displays.',
      ruleVersionAlert: 'Current minimal loop: deterministic parameter experiment. Candidates only generate drafts and will not auto-publish, schedule, or trade.',
      jobEventStream: 'Job Event Stream',
      noEvents: 'No events',
      selectJobToView: 'Select an experiment with a Job to view events.',
      submitForm: {
        title: 'Submit Experiment',
        baseTemplate: 'Base Strategy Template',
        baseTemplateRequired: 'Please select a base strategy template',
        baseTemplatePlaceholder: 'Select template',
        parameterSpace: 'Parameter Space JSON',
        parameterSpaceRequired: 'Please enter parameter space JSON',
        searchMethod: 'Search Method',
        maxCandidates: 'Max Candidates',
        objective: 'Objective',
        submit: 'Submit Experiment'
      },
      list: {
        title: 'Experiment List',
        column: {
          status: 'Status',
          searchMethod: 'Search Method',
          maxCandidates: 'Max Candidates',
          objective: 'Objective',
          actions: 'Actions',
          viewCandidates: 'View Candidates'
        }
      },
      candidates: {
        title: 'Candidates',
        titleWithId: 'Candidates: {{id}}',
        column: {
          rank: 'Rank',
          grade: 'Grade',
          score: 'Score',
          parameters: 'Parameters',
          summary: 'Summary',
          recommendation: 'Recommendation',
          actions: 'Actions',
          viewCandidates: 'View Candidates',
          generateDraft: 'Generate Draft'
        }
      },
      messages: {
        loadTemplatesFailed: 'Failed to load strategy templates',
        loadExperimentsFailed: 'Failed to load experiment list',
        loadCandidatesFailed: 'Failed to load candidates',
        subscribeJobFailed: 'Failed to subscribe to experiment Job events',
        candidatesGenerated: 'Strategy experiment candidates generated',
        submitFailed: 'Failed to submit experiment. Please verify the parameter space is valid JSON.',
        draftGenerated: 'Draft template generated: {{templateId}}',
        promoteFailed: 'Failed to promote candidate to draft'
      }
    },
    workspace: {
      title: 'Strategy Workspace',
      account: 'Account',
      accountPlaceholder: 'Account ID',
      chartWindow: 'Chart',
      hideCode: 'Hide Code',
      showCode: 'Show Code',
      quickTrade: 'Quick Trade',
      quickTradeHint: 'Select a symbol first',
      tradePanelPlaceholder: 'Trade panel — coming soon',
      selectSymbolHint: 'Select a trading account and symbol to view chart',
      noAccounts: 'No available accounts',
      selectSymbol: 'Symbol',
      code: 'Strategy Code',
      codePlaceholder: '# Python strategy code...
def run(context):
    return {"signal": "hold"}',
      validate: 'Validate',
      validatePass: 'Validation passed',
      validateFailed: 'Validation failed',
      validateBeforeSave: 'Please validate code before saving',
      runBacktest: 'Run Backtest',
      save: 'Save',
      copy: 'Copy',
      copySuccess: 'Copied',
      copyFailed: 'Copy failed',
      saveSuccess: 'Saved',
      chart: 'K-line',
      backtest: 'Backtest',
      backtestRunning: 'Backtest running...',
      backtestCompleted: 'Completed',
      backtestError: 'Backtest failed',
      backtestEmpty: 'Run a backtest to see results',
      backtestTab: 'Backtest Results',
      tuningTab: 'Smart Tuning',
      execAssumptions: 'ℹ Execution Assumptions',
      execAssumptionsFields: {
        mode: 'Mode',
        timing: 'Timing',
        fillRule: 'Fill Rule',
        direction: 'Direction',
        commission: 'Commission',
        slippage: 'Slippage',
        leverage: 'Leverage',
        mtfFallback: 'MTF Fallback'
      },
      aiAssist: 'AI Assistant',
      ai: 'AI',
      runtimeMode: 'Runtime',
      saveFailed: 'Save failed',
      autoFix: {
        fixing: 'Fixing...',
        button: 'Auto Fix',
        askAI: 'Ask AI',
        dismiss: 'Dismiss',
        passed: 'Auto-fix passed in {{iterations}} iteration{{plural}}',
        failed: 'Auto-fix: {{remaining}} issue(s) remain after {{iterations}} iterations',
        fixed: 'Fixed ({{count}})',
        remaining: 'Remaining ({{count}})',
        newRegression: 'New regression ({{count}})',
        lineInfo: 'line {{line}}'
      },
      template: {
        title: 'Template',
        selectPlaceholder: 'Select a template...',
        load: 'Load',
        saveAs: 'Save As New',
        loaded: 'Loaded'
      }
    },
    codeQuality: {
      category: {
        FUTURE_DATA_LEAK: 'Future Data Leak',
        MISSING_PARAM: 'Missing Param',
        UNREAD_PARAM: 'Unread Param',
        NDARRAY_PANDAS_MISUSE: 'ndarray/pandas Misuse',
        NO_STOP_AND_TAKE_PROFIT: 'Missing Stop/Take Profit',
        NO_ENTRY_PCT: 'Missing Entry %'
      }
    },
    backtestParams: {
      title: 'Backtest',
      currentDraft: '📝 Current Draft',
      dateRange: 'Date Range',
      execution: 'Execution',
      capital: 'Capital',
      leverage: 'Leverage',
      commission: 'Commission',
      slippage: 'Slippage',
      trade: 'Trade',
      direction: 'Direction',
      long: '↑ Long',
      short: '↓ Short',
      both: 'Both',
      strictMode: 'Strict Mode',
      strictModeOn: 'ON',
      strictModeOff: 'OFF',
      strictModeOnDesc: 'Next-bar-open. Standard, conservative.',
      strictModeOffDesc: 'Same-bar-close + MTF 1m. Higher precision.',
      strictModeOnTooltip: 'ON: signals confirmed at bar close, executed next bar open',
      strictModeOffTooltip: 'OFF: same-bar close execution with 1m sub-resolution',
      vectorizedMode: 'Vectorized',
      eventDrivenMode: 'Run(context)',
      runtimeMode: 'Runtime',
      history: 'Backtest History',
      run: '▶ Run',
      settingsSave: 'Save as My Defaults',
      settingsLoad: 'Load My Defaults',
      settingsReset: 'Reset to Factory',
      defaultsSaved: 'Defaults saved',
      defaultsLoaded: 'Defaults loaded',
      defaultsReset: 'Reset to factory defaults',
      presets: {
        liveAligned: 'Live Aligned',
        exploration: 'Exploration'
      }
    },
    tuning: {
      optimizerMethod: 'Optimizer method',
      parameterDimensions: 'Parameter dimensions',
      enabledCombinations: '{{enabled}} enabled · {{combos}} combinations',
      hide: 'Hide',
      preview: 'Preview',
      previewTitle: 'Preview ({{shown}} of {{total}})',
      truncated: 'TRUNCATED',
      results: 'Results ({{count}})',
      rank: '#',
      grade: 'Grade',
      score: 'Score',
      parameters: 'Parameters',
      summary: 'Summary',
      oosScore: 'OOS Score',
      degradation: 'Degradation',
      overfit: 'Overfit',
      overfitWarning: '⚠ OVERFIT',
      apply: 'Apply',
      run: 'Run ({{count}})',
      tuning: 'Tuning…',
      requiresAI: 'Requires AI provider configured',
      switchToDE: 'Switch to DE',
      waiting: 'Waiting for experiment... (SSE auto-refresh)',
      gridWarning: 'Grid Search would test <b>{{count}}</b> combinations (budget: 48). Consider switching to <b>Differential Evolution</b> which handles large parameter spaces efficiently.',
      oosFootnote: 'OOS validation run on top-5 candidates (by IS score). Green degradation <20%, orange 20-40%, red >40%.',
      optimizer: {
        grid: 'Grid Search',
        random: 'Random Search',
        de: 'Differential Evolution',
        tpe: 'TPE (KDE)',
        ags: 'Annealed Gaussian',
        ai: 'AI Optimizer',
        gridDesc: 'Exhaustive Cartesian product. Best for ≤3 params.',
        randomDesc: 'Uniform random sampling. Good for exploration.',
        deDesc: 'rand/1/bin mutation. Converges fast on smooth landscapes.',
        tpeDesc: 'Tree-structured Parzen Estimator. KDE models good/bad distributions.',
        agsDesc: 'Gaussian jitter with sigma annealing. Lightweight alternative to TPE.',
        aiDesc: 'LLM multi-round proposal. Learns from previous results over 3 rounds.'
      }
    }
  },
  indicatorCatalog: {
    title: 'インジケーターカタログ',
    description: '戦略サンドボックスで利用可能なテクニカル指標とリスクパラメータ。戦略コードではこれらのヘルパーとパラメータキーのみを使用してください。',
    indicatorsTitle: 'テクニカル指標',
    riskSectionTitle: 'リスク管理パラメータ',
    riskParamsTitle: '共通リスクパラメータ',
    riskParamsDesc: '指標の選択に関わらず、すべての戦略はこれらのリスク管理パラメータを尊重する必要があります。',
    paramKey: 'キー',
    paramLabel: 'ラベル',
    paramType: '型',
    paramDefault: 'デフォルト',
    paramRange: '範囲',
    paramDescription: '説明',
    workspace: {
      title: '戦略ワークスペース',
      account: '口座',
      accountPlaceholder: '口座 ID',
      chartWindow: 'チャート',
      hideCode: 'コード非表示',
      showCode: 'コード表示',
      quickTrade: 'クイック取引',
      quickTradeHint: '銘柄を選択してください',
      tradePanelPlaceholder: '取引パネル — 近日公開',
      selectSymbolHint: '取引口座と銘柄を選択してください',
      noAccounts: '利用可能な口座がありません',
      selectSymbol: '銘柄',
      code: '戦略コード',
      codePlaceholder: '# Python 戦略コード...
def run(context):
    return {"signal": "hold"}',
      validate: '検証',
      validatePass: '検証通過',
      validateFailed: '検証失敗',
      validateBeforeSave: '保存前にコードを検証してください',
      runBacktest: 'バックテスト実行',
      save: '保存',
      copy: 'コピー',
      copySuccess: 'コピーしました',
      copyFailed: 'コピー失敗',
      saveSuccess: '保存しました',
      chart: 'K線',
      backtest: 'バックテスト',
      backtestRunning: '実行中...',
      backtestCompleted: '完了',
      backtestError: '失敗',
      backtestEmpty: 'バックテストを実行してください',
      aiAssist: 'AI アシスタント',
      ai: 'AI',
      template: {
        title: 'テンプレート',
        selectPlaceholder: 'テンプレート選択...',
        load: 'ロード',
        saveAs: '新規保存',
        loaded: 'ロード済'
      }
    }
  }
} as const;

export default strategy;
