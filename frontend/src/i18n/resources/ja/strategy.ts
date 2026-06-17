const strategy = {
  strategy: {
    validation: {
      passed: 'コード検証に成功しました',
      notPassed: 'コード検証に失敗しました',
      riskEval: {
        title: 'リスク評価',
        riskHigh: 'リスクレベル：高',
        riskUnreliable: 'リスク評価：信頼できない（isReliable=false）',
        riskLoading: 'リスク評価を計算中です'
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
        code: 'Python 戦略コードを入力...',
        codeSample: 'ここに戦略コードを貼り付け',
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
        outro: '修正後の完全なコードを ```python で囲って出力し、変更点を説明してください。',
        pythonFenceStart: '```python',
        fenceEnd: '```'
      }
    },
    schedules: {
      title: '戦略スケジュール',
      createSchedule: 'スケジュール作成',
      format: {
        interval: '{{s}}秒毎',
        cron: 'cron: {{expr}}',
      },
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
        orders: '取引履歴',
        execLogs: '执行日志',
        orderLogs: '订单日志'
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
      scheduleIdLabel: 'スケジュールID:',
      status: {
        success: '成功',
        failed: '失败'
      },
      action: {
        start: '启动',
        stop: '停止',
        restart: '重启'
      }
    },
    templates: {
      title: '戦略テンプレート',
      tabs: {
        system: 'システムテンプレート',
        user: 'ユーザーテンプレート'
      },
      copySuffix: '（コピー）',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
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
          createAndEnable: '作成して有効化',
          create: '创建计划',
          addAccount: '添加账户',
          updateTradingPassword: '更新交易密码'
        },
        metrics: {
          totalReturn: '総リターン',
          annualReturn: '年率リターン',
          maxDrawdown: '最大ドローダウン',
          sharpe: 'シャープレシオ',
          winRate: '勝率',
          totalTrades: '取引回数'
        },
        form: {
          account: '账户',
          accountPlaceholder: '选择账户',
          scheduleName: '计划名称',
          scheduleNamePlaceholder: '输入计划名称',
          scheduleNameMax: '最多64字符',
          scheduleType: '计划类型',
          scheduleTypes: {
            interval: '定时执行',
            hfQuote: '高频报价',
            klineClose: 'K线收盘'
          },
          intervalMs: '间隔(毫秒)',
          intervalMsTip: '非高频模式最小1000ms',
          hfCooldownMs: '高频冷却(毫秒)',
          hfCooldownMsTip: '报价驱动执行间的冷却时间',
          symbol: '品种',
          symbolPlaceholder: '选择品种',
          symbolPlaceholderEmpty: '未配置品种',
          timeframe: '时间周期',
          defaultVolume: '默认手数',
          defaultVolumeTip: '每个信号的默认下单量',
          enableAfterCreate: '创建后立即启用',
          riskSection: '风控设置',
          maxDrawdownPct: '最大回撤%',
          maxDrawdownPctTip: '回撤超过此阈值自动停止',
          maxPositions: '最大持仓数',
          maxPositionsTip: '同时持有的最大仓位数量',
          stopLossOffset: '止损偏移',
          stopLossOffsetTip: '距入场价的止损距离(点)',
          takeProfitOffset: '止盈偏移',
          takeProfitOffsetTip: '距入场价的止盈距离(点)',
          strategyParamsSection: '策略参数',
          investorTag: '投资者(只读)'
        },
        noAccountTitle: '无账户',
        noAccountBody: '启动计划前需要先绑定MT账户。',
        investorWarningTitle: '投资者账户',
        investorWarningBody: '此账户为投资者(只读)模式，需要交易权限才能启动计划。',
        errorInvestorAccount: '无法使用投资者账户启动计划。请更新交易密码以启用交易。',
        verifyingPermission: '验证交易权限中...',
        tradePermissionOk: '交易权限验证通过',
        updatePasswordTitle: '更新交易密码',
        updatePasswordHint: '输入此账户的交易密码以启用交易。',
        updatePasswordOk: '交易密码已更新',
        updatePasswordFailed: '更新交易密码失败',
        updatePasswordStillInvestor: '密码更新成功但账户仍为投资者模式，请联系客服。',
        newPasswordPlaceholder: '输入新交易密码'
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
        loadingDefault: 'デフォルトテンプレートを読み込み中...',
        defaultHint: 'デフォルト',
        emptyUser: 'まだユーザーテンプレートはありません。上の「テンプレート作成」をクリックして始めてください。'
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
          extraSymbols: '追加銘柄（複数選択可）'
        },
        placeholders: {
          account: '口座を選択',
          symbol: '銘柄を選択',
          range: '期間を選択',
          extraSymbols: 'オプション。ペア/ローテーション戦略に便利です'
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
        modalTitleWithName: 'バックテスト：{{name}}',
        parameters: {
          title: 'ストラテジーパラメーター'
        },
        tooltips: {
          extraSymbols: 'K線を取得する追加銘柄（同一口座、同一時間足）。ストラテジーはcontext["closes_by_symbol"]からアクセスできます。'
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
        deepLinkNavigate: '外部リンクからテンプレートと最新実行の詳細を開きました',
        templatePublished: 'テンプレートを公開しました',
        cannotPublishAndCreateDraftFailed: '公開できません。下書きの作成に失敗しました。',
        republishedButNoTemplateId: '再公開しましたが、テンプレートIDがありません。',
        backtestRunningCannotPublish: 'バックテスト実行中のため、公開できません。',
        missingDraftIdCannotPublish: '下書きIDがないため、公開できません。',
        publishedButNoTemplateId: '公開しましたが、テンプレートIDがありません。',
        templateRepublished: 'テンプレートを再公開しました',
        templateAlreadyPublished: 'テンプレートは既に公開済みです',
        templateNotDraftUnknownPublishStatus: 'テンプレートは下書きではありません。公開ステータスが不明です。',
        publishFailed: '公開に失敗しました',
        backtestRunNoPublishedTemplate: 'バックテスト実行に対応する公開テンプレートがありません',
        strategyCodeEmptyCannotPublish: '策略代码为空，请先保存代码再发布。',
        systemTemplateReadOnly: '系统模板为只读，请克隆后再编辑。'
      },
      backtestRuns: {
        title: 'バックテストレポート',
        empty: 'バックテスト記録がありません',
        deleteConfirm: 'このバックテストレポートを削除しますか？',
        batchDelete: '{{count}}件を削除',
        batchDeleteConfirm: '{{count}}件のバックテストレポートを削除しますか？',
        batchDeleteSuccess: '{{count}}件のバックテストレポートを削除しました',
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
      deleteConfirm: 'このテンプレートを削除しますか？',
      defaultDraftName: '下書きテンプレート',
      codeModal: {
        title: 'ストラテジーコード',
        actions: {
          copy: 'コピー'
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
      saveBlockedNotValidated: 'まず「コード検証」を実行してください。検証に合格するまで保存は無効です。',
      generatePlaceholder: '描述你的策略需求...'
    },
    templateModal: {
      title: 'テンプレートとして保存',
      fields: {
        name: '名前',
        description: '説明'
      },
      placeholders: {
        name: 'テンプレート名を入力',
        description: '説明を入力'
      }
    },
    backtestRun: {
      title: 'バックテスト実行',
      status: {
        queued: 'キュー待ち',
        running: '実行中',
        completed: '完了',
        failed: '失敗',
        canceling: 'キャンセル中',
        canceled: 'キャンセル済み',
        ended: '終了'
      },
      actions: {
        cancel: 'キャンセル'
      },
      hints: {
        queued: 'バックテストはキュー待ちです',
        running: 'バックテストを実行中です',
        canceling: 'バックテストをキャンセル中です'
      },
      fields: {
        status: 'ステータス',
        error: 'エラー',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率'
      },
      metrics: {
        totalReturn: '総収益率',
        annualReturn: '年換算収益率',
        maxDrawdown: '最大ドローダウン',
        sharpe: 'シャープレシオ',
        winRate: '勝率',
        totalTrades: '総取引数',
        equityCurvePoints: 'エクイティカーブポイント数'
      },
      trades: {
        title: '注文詳細',
        empty: '記録された取引はありません',
        loadFailed: '注文詳細の読み込みに失敗しました',
        ticket: 'チケット',
        side: '売買',
        sideBuy: '買い',
        sideSell: '売り',
        volume: '数量',
        openTime: 'エントリー時間',
        openPrice: 'エントリー価格',
        closeTime: '決済時間',
        closePrice: '決済価格',
        pnl: '損益',
        commission: '手数料',
        reason: '決済理由',
        reasons: {
          signal: 'シグナル',
          sl: 'ストップロス',
          tp: 'テイクプロフィット',
          margin_call: 'マージンコール',
          expired: '期限切れ',
          end_of_test: 'テスト終了'
        },
        summary: '{{count}}件の取引 · {{wins}}勝 / {{losses}}敗 · 純損益 {{pnl}}'
      }
    },
    asset: {
      title: 'ストラテジーアセット',
      subtitle: 'アセットの公開、レビューステータス、複製はシステムで管理されます。複製結果は独立したユーザーテンプレートです。',
      submitAsset: 'アセットを提出',
      assetList: 'アセット一覧',
      name: '名前',
      visibility: '公開設定',
      reviewStatus: 'レビューステータス',
      cloneCount: '複製数',
      version: 'バージョン',
      description: '説明',
      actions: '操作',
      cloneAsDraft: '下書きとして複製',
      sourceTemplate: '元テンプレート',
      assetName: 'アセット名',
      submit: '提出',
      messages: {
        loadFailed: 'ストラテジーアセットの読み込みに失敗しました',
        submitSuccess: 'ストラテジーアセットを提出しました',
        submitFailed: 'ストラテジーアセットの提出に失敗しました',
        cloneSuccess: 'テンプレートとして複製しました：{{templateId}}',
        cloneFailed: 'ストラテジーアセットの複製に失敗しました'
      },
      validation: {
        selectTemplate: '元テンプレートを選択してください',
        enterName: 'アセット名を入力してください'
      },
      empty: '暂无策略资产'
    },
    gen: {
      title: 'ストラテジー生成',
      send: 'ストラテジー生成',
      regenerate: '再生成',
      reset: '最初からやり直す',
      template: 'テンプレート',
      generating: '生成中...',
      validating: 'コンプライアンスチェック',
      backtestStarted: 'バックテスト開始',
      done: '完了',
      backtestMsg: 'バックテストタスクを作成しました',
      clarifyTitle: '確認事項：',
      useDefaults: 'デフォルトで続行',
      placeholder: '作成したい取引ストラテジーを説明してください。例：「EURUSDの1時間足でボリンジャーバンド平均回帰ストラテジーを作成」',
      chat: {
        generate: '⚡ 生成',
        revise: '✏️ 修正',
        repair: '🔧 修復',
        discuss: '💬 議論'
      },
      feedback: {
        heading: '📊 バックテスト結果',
        placeholder: 'フィードバックを入力して反復改善（例：「攻撃的すぎる」「ストップロスを追加」）'
      },
      metrics: {
        sharpe: 'シャープ',
        maxDrawdown: '最大DD',
        winRate: '勝率',
        trades: '取引',
        return: 'リターン'
      }
    },
    marketRegime: {
      title: '相場状態検出',
      subtitle: '過去のK線データからトレンド方向、ボラティリティ状態、価格効率性を分析し、現在の市場環境を分類します。',
      ruleVersionAlert: '現在はルールベース検出モデル rule-v1 を使用しており、リアルタイムK線市場データで駆動されています。',
      detectSuccess: '相場状態検出が完了しました',
      detectFailed: '相場状態検出に失敗しました',
      form: {
        title: '検出パラメーター',
        accountId: 'アカウントID',
        accountIdRequired: 'アカウントIDは必須です',
        accountIdPlaceholder: 'MTアカウントUUID',
        symbol: '銘柄',
        symbolRequired: '銘柄は必須です',
        symbolPlaceholder: 'EURUSD',
        timeframe: '時間足',
        klineCount: 'K線本数',
        submit: '検出開始'
      },
      result: {
        title: '検出結果',
        status: 'ステータス',
        confidence: '確信度',
        modelVersion: 'モデルバージョン',
        strategyFamilies: 'ストラテジーファミリー',
        features: '特徴量',
        recordId: 'レコードID'
      }
    },
    experiment: {
      title: 'ストラテジー実験',
      subtitle: 'パラメーターの組み合わせを送信すると、システムが自動的に実験を実行し、候補戦略をスコアしてドラフトを生成します。',
      ruleVersionAlert: '現在の最小ループ：決定論的パラメーター実験。候補は下書きを生成するのみで、自動公開、スケジュール、取引は行いません。',
      jobEventStream: 'Jobイベントストリーム',
      noEvents: 'イベントなし',
      selectJobToView: 'Jobを持つ実験を選択してイベントを表示してください。',
      submitForm: {
        title: '実験を送信',
        baseTemplate: 'ベースストラテジーテンプレート',
        baseTemplateRequired: 'ベースストラテジーテンプレートを選択してください',
        baseTemplatePlaceholder: 'テンプレートを選択',
        parameterSpace: 'パラメーター空間JSON',
        parameterSpaceRequired: 'パラメーター空間JSONを入力してください',
        searchMethod: '探索手法',
        maxCandidates: '最大候補数',
        objective: '目標',
        submit: '実験を送信'
      },
      list: {
        title: '実験一覧',
        column: {
          status: 'ステータス',
          searchMethod: '探索手法',
          maxCandidates: '最大候補数',
          objective: '目標',
          actions: '操作',
          viewCandidates: '候補を表示'
        }
      },
      candidates: {
        title: '候補',
        titleWithId: '候補：{{id}}',
        column: {
          rank: '順位',
          grade: 'グレード',
          score: 'スコア',
          parameters: 'パラメーター',
          summary: '概要',
          recommendation: '推奨',
          actions: '操作',
          viewCandidates: '候補を表示',
          generateDraft: '下書き生成'
        }
      },
      messages: {
        loadTemplatesFailed: 'ストラテジーテンプレートの読み込みに失敗しました',
        loadExperimentsFailed: '実験一覧の読み込みに失敗しました',
        loadCandidatesFailed: '候補の読み込みに失敗しました',
        subscribeJobFailed: '実験Jobイベントの購読に失敗しました',
        candidatesGenerated: 'ストラテジー実験の候補が生成されました',
        submitFailed: '実験の送信に失敗しました。パラメーター空間が有効なJSONであることを確認してください。',
        draftGenerated: '下書きテンプレートを生成しました：{{templateId}}',
        promoteFailed: '候補から下書きへの昇格に失敗しました'
      }
    },
    workspace: {
      title: 'ストラテジーワークスペース',
      account: '口座',
      accountPlaceholder: 'アカウントID',
      chartWindow: 'チャート',
      hideCode: 'コードを隠す',
      showCode: 'コードを表示',
      quickTrade: 'クイックトレード',
      quickTradeHint: '先に銘柄を選択してください',
      tradePanelPlaceholder: 'トレードパネル — 近日公開予定',
      selectSymbolHint: '取引口座と銘柄を選択してチャートを表示',
      noAccounts: '利用可能な口座がありません',
      selectSymbol: '銘柄',
      code: 'ストラテジーコード',
      codePlaceholder: `# Pythonストラテジーコード...
def run(context):
    return {"signal": "hold"}`,
      validate: '検証',
      validatePass: '検証に合格',
      validateFailed: '検証に失敗',
      validateBeforeSave: '保存する前にコードを検証してください',
      runBacktest: 'バックテスト実行',
      save: '保存',
      copy: 'コピー',
      copySuccess: 'コピーしました',
      copyFailed: 'コピーに失敗しました',
      saveSuccess: '保存しました',
      chart: 'K線',
      backtest: 'バックテスト',
      backtestRunning: 'バックテスト実行中...',
      backtestCompleted: '完了',
      backtestError: 'バックテスト失敗',
      backtestEmpty: 'バックテストを実行して結果を確認してください',
      backtestTab: 'バックテスト結果',
      tuningTab: 'スマートチューニング',
      execAssumptions: 'ℹ 執行前提条件',
      execAssumptionsFields: {
        mode: 'モード',
        timing: 'タイミング',
        fillRule: '約定ルール',
        direction: '方向',
        commission: '手数料',
        slippage: 'スリッページ',
        leverage: 'レバレッジ',
        mtfFallback: 'MTFフォールバック'
      },
      aiAssist: 'AIアシスタント',
      ai: 'AI',
      runtimeMode: 'ランタイム',
      saveFailed: '保存に失敗しました',
      autoFix: {
        fixing: '修正中...',
        button: '自動修正',
        askAI: 'AIに問い合わせ',
        dismiss: '閉じる',
        passed: '自動修正が{{iterations}}回の反復で成功しました{{plural}}',
        failed: '自動修正：{{iterations}}回の反復後も{{remaining}}件の問題が残っています',
        fixed: '修正済み（{{count}}）',
        remaining: '残り（{{count}}）',
        newRegression: '新規回帰（{{count}}）',
        lineInfo: '{{line}}行目'
      },
      template: {
        title: 'テンプレート',
        selectPlaceholder: 'テンプレートを選択...',
        load: '読み込み',
        saveAs: '新規保存',
        loaded: '読み込み済み'
      },
      watchlist: 'ウォッチリスト',
      selectAccount: '口座を選択',
      openPositions: 'オープンポジション ({{count}})',
      noOpenPositions: 'この口座にオープンポジションはありません',
      chartError: 'チャートエラー — 再読み込みしてください',
      smartTuning: 'スマートチューニング',
      quickTradeSection: {
        selectSymbol: 'まず銘柄を選択してください',
        validVolume: '有効な数量を入力してください',
        priceRequired: '指値・逆指値注文には価格が必要です',
        orderPlaced: '{{side}} 注文が成立しました',
        orderFailed: '注文失敗',
        amountLots: '数量（ロット）',
        marginMode: '証拠金モード',
        cross: 'クロス',
        isolated: 'アイソレート',
        mt4CrossOnly: 'MT4はクロス証拠金のみ対応'
      },
      chartTools: {
        streamActive: 'ライブバーストリーム アクティブ',
        streamUnavailable: 'ストリーム利用不可',
        hide: '非表示',
        show: '表示',
        settings: '設定',
        remove: '削除',
        clearDrawings: '描画をすべてクリア',
        candle: 'ローソク',
        ohlc: 'OHLC',
        area: 'エリア',
        live: 'LIVE',
        error: 'ERROR',
        static: 'STATIC'
      },
      backtestRunIdLabel: 'バックテスト実行を選択...',
      investorReadOnly: '投資家（閲覧のみ）',
      masterTrading: 'マスター（取引）',
      riskControls: 'コード内のリスク管理ルール',
      jumpToCode: 'コードへ移動',
      runningStatus: '実行中...',
      completedStatus: '完了',
      backtestResultsLabel: 'バックテスト結果',
      gateTab: 'Gate'
    },
    codeQuality: {
      category: {
        FUTURE_DATA_LEAK: '未来データ漏洩',
        MISSING_PARAM: 'パラメーター不足',
        UNREAD_PARAM: '未読パラメーター',
        NDARRAY_PANDAS_MISUSE: 'ndarray/pandas 誤用',
        NO_STOP_AND_TAKE_PROFIT: 'ストップロス/テイクプロフィットなし',
        NO_ENTRY_PCT: 'エントリー%なし'
      }
    },
    backtestParams: {
      title: 'バックテスト',
      currentDraft: '📝 現在の下書き',
      dateRange: '期間',
      execution: '執行',
      capital: '元本',
      leverage: 'レバレッジ',
      commission: '手数料',
      slippage: 'スリッページ',
      trade: '取引',
      direction: '方向',
      long: '↑ 買い',
      short: '↓ 売り',
      both: '両方向',
      strictMode: '厳密モード',
      strictModeOn: 'ON',
      strictModeOff: 'OFF',
      strictModeOnDesc: '次のバーのオープンで執行。標準的で保守的。',
      strictModeOffDesc: '同一バーのクローズ + MTF 1分足。より高精度。',
      strictModeOnTooltip: 'ON：シグナルはバー終了時に確定し、次のバーオープンで執行',
      strictModeOffTooltip: 'OFF：同一バークローズ執行、1分サブ解像度を使用',
      vectorizedMode: 'ベクトル化',
      eventDrivenMode: 'Run(context)',
      runtimeMode: 'ランタイム',
      history: 'バックテスト履歴',
      run: '▶ 実行',
      settingsSave: 'マイデフォルトとして保存',
      settingsLoad: 'マイデフォルトを読み込み',
      settingsReset: '工場出荷時にリセット',
      defaultsSaved: 'デフォルトを保存しました',
      defaultsLoaded: 'デフォルトを読み込みました',
      defaultsReset: '工場出荷時デフォルトにリセットしました',
      presets: {
        liveAligned: '本番準拠',
        exploration: '探索'
      },
      enterCodeAndSymbol: '戦略コードを入力し銘柄を選択してください',
      backtestFailed: 'バックテスト失敗'
    },
    tuning: {
      optimizerMethod: '最適化手法',
      parameterDimensions: 'パラメーター次元数',
      enabledCombinations: '有効{{enabled}} · {{combos}}通りの組み合わせ',
      hide: '非表示',
      preview: 'プレビュー',
      previewTitle: 'プレビュー（表示{{shown}} / 全{{total}}）',
      truncated: '省略',
      results: '結果（{{count}}）',
      rank: '#',
      grade: 'グレード',
      score: 'スコア',
      parameters: 'パラメーター',
      summary: '概要',
      oosScore: 'OOSスコア',
      degradation: '劣化度',
      overfit: 'オーバーフィット',
      overfitWarning: '⚠ オーバーフィット',
      apply: '適用',
      run: '実行（{{count}}）',
      tuning: 'チューニング中...',
      requiresAI: 'AIプロバイダーの設定が必要です',
      switchToDE: 'DEに切り替え',
      waiting: '実験の結果を待機中...（SSE自動更新）',
      gridWarning: 'グリッドサーチでは<b>{{count}}</b>通りの組み合わせをテストします（予算：48）。大規模パラメーター空間には<b>Differential Evolution</b>への切り替えをご検討ください。',
      oosFootnote: 'OOS検証は上位5候補（ISスコア順）で実行。緑=劣化20%未満、橙=20-40%、赤=40%超。',
      optimizer: {
        grid: 'グリッドサーチ',
        random: 'ランダムサーチ',
        de: '差分進化',
        tpe: 'TPE（KDE）',
        ags: '焼きなましガウス',
        ai: 'AIオプティマイザー',
        gridDesc: '全組み合わせを網羅。3パラメーター以下に最適。',
        randomDesc: '一様ランダムサンプリング。探索に適する。',
        deDesc: 'rand/1/bin突然変異。滑らかなランドスケープで高速収束。',
        tpeDesc: 'Tree-structured Parzen Estimator。KDEで良/悪分布をモデル化。',
        agsDesc: 'ガウシアンジッターとシグマ焼きなまし。TPEの軽量代替。',
        aiDesc: 'LLMによる複数ラウンド提案。過去の結果から学習（3ラウンド）。'
      },
      started: 'スマートチューニング開始'
    },
    paper: {
      title: '📊 ペーパー取引',
      createAccount: 'ペーパー口座を作成',
      accountName: '口座名',
      create: '作成',
      noAccounts: 'ペーパー口座がありません。作成して模擬取引を開始してください。',
      running: '実行中 {{symbol}} {{timeframe}}',
      start: '開始',
      stop: '停止',
      watch: '監視',
      paper: 'ペーパー',
      startStrategy: 'ペーパー戦略を開始',
      symbol: '銘柄',
      timeframe: '時間枠',
      strategyCode: '戦略コード (Python)',
      messages: {
        enterName: '名前を入力してください',
        created: 'ペーパー口座が作成されました',
        createFailed: '作成に失敗しました',
        pasteCode: '戦略コードを貼り付けてください',
        strategyStarted: 'ペーパー戦略が開始されました',
        startFailed: '開始に失敗しました',
        strategyStopped: 'ペーパー戦略が停止されました',
        stopFailed: '停止に失敗しました'
      }
    },
    aiChat: {
      title: 'AI チャット',
      you: 'あなた',
      ai: 'AI',
      revise: '修正',
      feedback: '🔄 フィードバック',
      streaming: '生成中',
      analyzing: '分析中',
      reset: 'リセット',
      applyCode: 'コードを適用',
      dismiss: '閉じる',
      reviewCode: 'AIがコードを生成しました — 適用前に上の会話を確認してください。'
    },
    assetAnalysis: {
      title: 'AI 資産分析',
      subtitle: 'マルチタイムフレームのトレンド展望、サポート/レジスタンス検出、ボラティリティ分類、AI戦略推奨',
      symbolPlaceholder: '銘柄を入力 (例: EURUSD, XAUUSD, BTCUSD)',
      analyze: '分析',
      fetchingData: '市場データを取得中...',
      phase: 'フェーズ: {{phase}}',
      mtfOutlook: 'マルチタイムフレーム展望',
      srLevels: 'サポート / レジスタンス',
      volatility: 'ボラティリティ',
      state: '状態',
      atrPct: 'ATR %',
      aiRecommendation: 'AI 戦略推奨',
      aiUnavailable: 'AI推奨は利用できません。設定でAIプロバイダーを構成してください。',
      configureAI: 'AIプロバイダーを設定',
      noLevels: '有意なレベルは検出されませんでした',
      noResults: '分析結果が返されませんでした。別の銘柄をお試しください。',
      volLow: '低ボラティリティ — ブレイクアウトまたは平均回帰戦略をタイトなストップで検討。',
      volNormal: '通常ボラティリティ — ほとんどの戦略タイプに適しています。',
      volHigh: '高ボラティリティ — 広めのストップ推奨；トレンドフォローとブレイクアウト戦略が有利。',
      volExtreme: '極端なボラティリティ — ポジションサイズを大幅に縮小；広いストップが必要。'
    },
    ai: {
      checkSettings: '检查AI设置',
      refreshFailed: '刷新失败',
      settings: 'AI设置'
    },
    backtest: {
      annualReturn: '年化收益',
      equityCurve: '权益曲线',
      maxDrawdown: '最大回撤',
      sharpe: '夏普比率',
      totalReturn: '总收益',
      totalTrades: '总交易',
      winRate: '胜率',
      tradeLog: '交易日志',
      tradeTime: '时间',
      tradeSide: '方向',
      tradePrice: '价格',
      tradeVolume: '数量'
    },
    chartTools: {
      clearDrawings: '清除所有绘图',
      hide: '隐藏',
      show: '显示',
      settings: '设置',
      remove: '移除'
    },
    quickTradeSection: {
      amountLots: '数量(手)',
      marginMode: '保证金模式',
      cross: '跨式',
      isolated: '逐仓',
      mt4CrossOnly: 'MT4 仅支持跨式保证金',
      selectSymbol: '请选择交易品种',
      validVolume: '交易量需 ≥ 0.01 手',
      priceRequired: '请输入价格',
      orderPlaced: '下单成功',
      orderFailed: '下单失败'
    },
    library: {
      title: '戦略ライブラリ',
      myStrategies: 'マイ戦略',
      create: '作成',
      filterAll: 'すべて',
      filterMine: 'マイ',
      filterSystem: 'プリセット',
      searchPlaceholder: '戦略を検索...',
      empty: '戦略がありません',
      published: '公開済み',
      draft: '下書き',
      unpublish: '公開取消',
      unpublishShort: '取消',
      publish: 'マーケットに公開',
      publishSuccess: '公開しました',
      unpublishSuccess: '公開を取り消しました',
      publishStatus: '公開ステータス',
      selectHint: '左側から戦略を選択してください',
      overview: '概要',
      schedules: '実行',
      backtestHistory: 'バックテスト履歴',
      scheduleCount: '{{count}} 件実行中',
      scheduleRunningCount: '{{count}} 件実行中',
      noSchedules: '実行なし',
      openInWorkspace: 'Workspaceで開く',
      createSchedule: '実行を作成',
      saveAsMine: '自分の戦略として保存',
      saveAsMineSuccess: '自分の戦略に保存しました',
      myCopy: 'マイコピー',
      codePreview: 'コードプレビュー',
      viewCode: '戦略コードを表示',
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
    paramDescription: '説明'
  }
} as const;

export default strategy;
