const strategy = {
  strategy: {
    templates: {
      title: '戦略テンプレート',
      tabs: {
        system: 'システムテンプレート',
        user: 'User templates'
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
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      badges: {
        preset: 'プリセット'
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
        edit: '編集',
        delete: '削除',
        backtest: 'バックテスト',
        viewCode: 'コード表示',
        copy: 'コピー',
        launchSchedule: 'スケジュール起動',
        createTemplate: 'Create template'
      },
      copySuffix: '（コピー）',
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
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
          create: 'スケジュール作成',
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
          account: '口座',
          accountPlaceholder: '选择账户',
          scheduleName: '计划名称',
          scheduleNamePlaceholder: '例：EURUSD M5 朝の戦略',
          scheduleNameMax: '最多64字符',
          scheduleType: '计划类型',
          scheduleTypes: {
            interval: '定时执行',
            hfQuote: '高频报价',
            klineClose: 'K-line Close'
          },
          intervalMs: '间隔(毫秒)',
          intervalMsTip: '非高频模式最小1000ms',
          hfCooldownMs: '高频冷却(毫秒)',
          hfCooldownMsTip: '报价驱动执行间的冷却时间',
          symbol: '銘柄',
          symbolPlaceholder: '选择品种',
          symbolPlaceholderEmpty: '未配置品种',
          timeframe: '時間足',
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
        newPasswordPlaceholder: 'Enter new trading password'
      },
      editTemplateModal: {
        title: {
          edit: 'テンプレート編集',
          create: 'Create template'
        },
        actions: {
          validateCode: 'Validate code'
        },
        fields: {
          name: '名称',
          description: '説明',
          code: '戦略コード',
          publicShare: '公開'
        },
        validation: {
          nameRequired: '名称を入力してください',
          codeRequired: 'Code is required'
        },
        placeholders: {
          name: '例：移動平均クロス戦略',
          description: '任意：説明',
          codeSample: 'Python 戦略コードを入力...'
        }
      },
      codeModal: {
        title: '戦略コード',
        actions: {
          copy: 'コピー'
        }
      },
      backtest: {
        title: 'バックテスト',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: '策略参数'
        },
        fields: {
          title: 'Title',
          account: '口座',
          symbol: '銘柄',
          timeframe: '時間足',
          initialCapital: 'Initial capital',
          range: '範囲',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        validation: {
          accountRequired: '口座を選択してください',
          symbolRequired: '銘柄を選択してください',
          timeframeRequired: '時間足を選択してください',
          initialCapitalRequired: 'Initial capital is required',
          rangeRequired: 'Range is required'
        },
        placeholders: {
          account: '口座を選択',
          symbol: '銘柄を選択',
          range: 'Select range',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
        },
        accountDisabledSuffix: '（無効）',
        quickRange: {
          '1d': '1D',
          '3d': '3D',
          '1w': '1W',
          '1y': '1Y',
          custom: 'Custom'
        }
      },
      messages: {
        deepLinkNavigate: 'Opened template and latest run details from external link',
        missingScheduleInfo: 'Missing schedule info',
        templateNotPublishedCannotCreateSchedule: 'Template is not published. Cannot create schedule.',
        readTemplateStatusFailed: 'Failed to read template status',
        scheduleCreatedAndEnabled: 'Schedule created and enabled',
        scheduleCreated: 'Schedule created',
        createScheduleFailed: 'Failed to create schedule',
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
        fetchTemplateListFailed: 'Failed to load template list',
        enterStrategyCode: '戦略コードを入力してください',
        codeValidationPassed: 'Code validation passed',
        codeValidationNotPassed: 'Code validation did not pass',
        codeValidationFailed: 'Code validation failed',
        templateUpdated: 'Template updated',
        templateCreated: 'Template created',
        templateDeleted: 'Template deleted',
        readStrategyCodeFailed: 'Failed to read strategy code',
        strategyCodeEmptyCannotBacktest: 'Strategy code is empty. Cannot backtest.',
        selectBacktestRange: 'Please select backtest range',
        backtestRangeInvalid: 'Invalid backtest range',
        backtestSubmitted: 'Backtest submitted',
        backtestSubmitFailed: 'Failed to submit backtest',
        backtestCancelRequested: 'Backtest cancel requested',
        backtestCancelFailed: 'Failed to cancel backtest',
        backtestReportDeleted: 'Backtest report deleted',
        backtestReportNotFound: 'Backtest report not found',
        backtestRunNoPublishedTemplate: 'Backtest run has no published template',
        codeCopied: 'Code copied',
        copyFailed: 'コピーに失敗しました。手動でコピーしてください',
        strategyCodeEmptyCannotPublish: 'Strategy code is empty. Please save your code before publishing.',
        systemTemplateReadOnly: 'System templates are read-only. Clone to edit.'
      },
      backtestRuns: {
        title: 'Backtest runs',
        empty: 'No backtest runs',
        table: {
          title: 'Title',
          status: '状態',
          symbol: '銘柄',
          timeframe: '時間足',
          createdAt: '作成日時',
          actions: '操作'
        },
        actions: {
          view: 'View',
          launchSchedule: 'View score',
          createSchedule: 'スケジュール作成'
        },
        deleteConfirm: 'Delete this run?',
        batchDelete: 'Delete {{count}}',
        batchDeleteConfirm: 'Delete {{count}} backtest report(s)?',
        batchDeleteSuccess: '{{count}} backtest report(s) deleted',
        status: {
          queued: 'Queued',
          running: '実行中',
          completed: '完了',
          failed: '失败',
          canceling: 'Canceling',
          canceled: 'Canceled'
        }
      }
    },
    defaultTemplates: {
      maCross: {
        name: 'Dual MA Crossover Strategy',
        description: 'Buy when fast MA crosses above slow MA, sell when it crosses below'
      },
      forceBuy: {
        name: 'Force BUY Test',
        description: 'For verifying the order pipeline: always returns buy on each execution, reads lot from context/params as volume'
      },
      rsi: {
        name: 'RSI Overbought/Oversold Strategy',
        description: 'Buy when RSI < 30 (oversold), sell when RSI > 70 (overbought)'
      },
      macd: {
        name: 'MACD Strategy',
        description: 'Buy on MACD golden cross, sell on death cross'
      }
    },
    validation: {
      passed: '検証に成功しました',
      notPassed: 'コード検証に失敗しました',
      riskEval: {
        title: 'リスク評価',
        riskHigh: 'リスクレベル：高',
        riskUnreliable: 'リスク評価：信頼できない（isReliable=false）',
        riskLoading: 'Risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: '戦略エディタ',
      labels: {
        code: '戦略コード',
        account: '口座',
        symbol: '銘柄',
        timeframe: '時間足',
        disabledSuffix: '（無効）'
      },
      actions: {
        copy: 'コピー',
        validate: 'コード検証',
        preview: 'シグナルプレビュー',
        saveAsTemplate: 'テンプレートとして保存',
        sendToAI: 'AI に修正を依頼',
        sendToAIFixTitleValidate: '検証失敗 / 警告あり',
        sendToAIFixTitlePreview: 'Fix preview issues'
      },
      placeholders: {
        code: 'Python 戦略コードを入力...',
        selectAccount: '口座を選択',
        selectAccountFirst: '先に口座を選択',
        loadingSymbols: '銘柄を読み込み中...',
        selectSymbol: '銘柄を選択',
        noSymbols: 'No symbols available'
      },
      hints: {
        previewInfo: 'Preview will execute with sample market data.'
      },
      cards: {
        validationResult: '検証結果',
        previewResult: 'Preview result'
      },
      messages: {
        enterCode: '戦略コードを入力してください',
        validateFailed: '検証に失敗しました',
        validateError: '検証エラー',
        validateOk: '検証に成功しました',
        selectAccount: '口座を選択してください',
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
        pythonFenceStart: '```python',
        fenceEnd: '```',
        outputTitle: '【出力】',
        outro: 'Return only the fixed code wrapped in ```python```.'
      }
    },
    templateModal: {
      title: 'テンプレートとして保存',
      fields: {
        name: '名称',
        description: '説明'
      },
      placeholders: {
        name: 'Enter template name',
        description: '任意：説明'
      }
    },
    backtestRun: {
      title: 'Backtest run',
      status: {
        queued: 'Queued',
        running: '実行中',
        completed: '完了',
        failed: '失败',
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
        status: '状態',
        error: 'エラー',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe'
      },
      metrics: {
        totalReturn: '総リターン',
        annualReturn: '年率リターン',
        maxDrawdown: '最大ドローダウン',
        sharpe: 'シャープレシオ',
        winRate: '勝率',
        totalTrades: '取引回数',
        equityCurvePoints: 'Equity curve points'
      },
      trades: {
        title: 'Order details',
        empty: 'No trades recorded',
        loadFailed: 'Failed to load order details',
        ticket: 'チケット',
        side: '方向',
        sideBuy: 'Buy',
        sideSell: 'Sell',
        volume: 'Volume',
        openTime: 'Open time',
        openPrice: '建値',
        closeTime: 'Close time',
        closePrice: '決済値',
        pnl: 'P&L',
        commission: 'Commission',
        reason: 'Close reason',
        reasons: {
          signal: 'シグナル（発注用）',
          sl: 'Stop loss',
          tp: 'Take profit',
          margin_call: 'Margin call',
          expired: 'Expired',
          end_of_test: 'End of test'
        },
        summary: '{{count}} trades · {{wins}} wins / {{losses}} losses · net P&L {{pnl}}'
      }
    },
    scheduleLogs: {
      title: '記録',
      titleWithName: '記録 - {{name}}',
      messages: {
        missingScheduleId: 'Missing schedule ID'
      },
      execStatus: {
        pending: '未評価',
        running: '実行中',
        completed: '完了',
        failed: '失败',
        skipped: 'Skipped'
      },
      operationStatus: {
        success: '成功',
        failed: '失败',
        running: '実行中'
      },
      execTable: {
        time: '時間',
        action: '操作',
        execute: '実行',
        status: '状態',
        durationMs: '所要(ms)',
        error: 'エラー'
      },
      ordersTable: {
        time: '時間',
        side: '方向',
        symbol: '銘柄',
        lots: 'ロット',
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
        sellStopLimit: 'Sell stop limit'
      },
      scheduleIdLabel: 'スケジュールID:',
      summary: {
        name: '名称',
        status: '状態',
        trade: '取引',
        enableCount: '有効化回数',
        lastRun: '最終実行',
        lastError: 'Last error'
      },
      tabs: {
        exec: '実行履歴',
        orders: '取引履歴',
        execLogs: '执行日志',
        orderLogs: 'Order Logs'
      },
      status: {
        success: '成功',
        failed: '失败'
      },
      action: {
        start: '启动',
        stop: '停止',
        restart: 'Restart'
      }
    },
    schedules: {
      title: '戦略スケジュール',
      createSchedule: 'スケジュール作成',
      format: {
        interval: '{{s}}秒毎',
        cron: 'cron: {{expr}}',
      },
      status: {
        running: '実行中',
        disabled: 'Disabled'
      },
      templateVisibility: {
        public: '公開',
        private: '非公開'
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
      nextRunAt: '次回実行',
      enableCount: '有効化回数',
      actions: {
        create: 'スケジュール作成',
        logs: '実行ログ',
        healthCheck: 'ヘルスチェック',
        runNow: 'Run now'
      },
      health: {
        title: '戦略ヘルスチェック {{name}}',
        summaryBanner: 'ヘルス評価: {{grade}}、サンプル {{totalRuns}} 件、成功率 {{successRate}}%',
        grade: {
          pending: '未評価',
          noSample: 'サンプル不足',
          healthy: '健全',
          watch: '要注意',
          alert: 'Alert'
        },
        notes: {
          pending: 'まずヘルスチェックを実行してください。',
          noSample: '評価に必要なサンプル不足（最低 {{minSampleSize}} 件）。',
          healthy: '成功率が高く、失敗回数も許容範囲です。',
          watch: '成功率は監視対象です（>= {{yellowSuccessRate}}%）。',
          alert: 'Low success rate. Investigate strategy/account conditions now.'
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
          latestError: 'Latest error'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}、緑: 成功率>={{greenSuccessRate}}% かつ 失敗<={{greenMaxFailedRuns}}、黄: 成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近の実行ログ',
          orders: 'Recent order records'
        },
        runLogs: {
          signalType: 'シグナル（発注用）'
        },
        messages: {
          loadFailed: 'ヘルスデータの読み込みに失敗しました',
          clickRefresh: 'Click refresh to load health data'
        }
      },
      deleteConfirm: {
        title: 'Delete this schedule?'
      },
      validation: {
        parametersMustBeJsonObject: 'Parameters must be a JSON object'
      },
      messages: {
        parametersParseFailed: 'パラメータ解析に失敗しました',
        defaultTemplateNotFound: 'デフォルトテンプレートが見つかりません。更新して再試行してください。',
        importDefaultTemplateFailedNoId: 'デフォルトテンプレートの取り込みに失敗しました（IDがありません）',
        templateCodeEmptyCannotExecute: 'テンプレートコードが空です。実行できません。',
        strategyExecuteFailed: '戦略実行に失敗しました',
        executeFailed: '実行に失敗しました',
        noOrderableSignal: '発注可能なシグナルがありません',
        signalHoldCannotOrder: 'シグナルが保留/無操作のため発注できません',
        volumeInvalid: '数量が不正です（> 0）',
        orderSubmitted: '注文を送信しました',
        orderFailed: '注文に失敗しました'
      },
      editModal: {
        title: {
          edit: 'スケジュール編集',
          create: 'スケジュール作成'
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
          enableExtra: 'Enable schedule after creating'
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
          triggerModeRequired: 'Trigger mode is required'
        },
        runFrequencyExtra: {
          cron: '高度：Cron で実行時間を精密に制御',
          byTimeframe: 'Run by timeframe'
        },
        runFrequencyOptions: {
          byTimeframe: '時間足でトリガー（推奨）',
          cron: 'Cron'
        },
        autoName: {
          strategy: 'Strategy'
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
            hf: 'High-frequency signal stream'
          },
          stableOverrideIntervalSeconds: '安定モード上書き間隔(秒)',
          stableOverrideIntervalSecondsExtra: '任意。安定モードの間隔を上書き',
          hfCooldownMs: '高頻度クールダウン(ms)',
          hfCooldownMsExtra: 'デバウンス：評価/発注の最小間隔',
          parametersJson: 'パラメータ(JSONオブジェクト)',
          parametersJsonExtra: 'JSON parameters for the strategy'
        }
      },
      triggerModal: {
        title: '今すぐ実行（即時発注）',
        confirmOrder: {
          title: '発注する',
          ok: 'Confirm'
        },
        actions: {
          confirmOrder: '発注する',
          rerun: 'Re-run'
        },
        summary: {
          scheduleName: 'スケジュール名',
          account: '口座',
          symbol: '銘柄',
          timeframe: '時間足'
        },
        messages: {
          signalNotOrderable: 'Signal is not orderable'
        },
        cards: {
          logs: '実行ログ',
          signal: 'シグナル（発注用）'
        },
        emptyLogs: '(ログなし)',
        emptySignal: 'No signal'
      }
    },
    asset: {
      title: 'Strategy Assets',
      subtitle: 'Asset publishing, review status, and cloning are maintained by the system. Cloned results are independent user templates.',
      submitAsset: 'Submit Asset',
      assetList: 'Asset List',
      name: '名称',
      visibility: '公開範囲',
      reviewStatus: 'Review Status',
      cloneCount: 'Clones',
      version: 'Version',
      description: '説明',
      actions: '操作',
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
      },
      empty: 'No strategy assets yet'
    },
    paper: {
      title: '📊 Paper Trading',
      createAccount: 'Create Paper Account',
      accountName: 'Account name',
      create: 'スケジュール作成',
      noAccounts: 'No paper accounts. Create one to start simulated trading.',
      running: 'Running {{symbol}} {{timeframe}}',
      start: '启动',
      stop: '停止',
      watch: '要注意',
      paper: 'Paper',
      startStrategy: 'Start Paper Strategy',
      symbol: '銘柄',
      timeframe: '時間足',
      strategyCode: 'Strategy Code (Python)',
      messages: {
        enterName: 'Enter a name',
        created: 'Paper account created',
        createFailed: 'Create failed',
        pasteCode: 'Paste your strategy code',
        strategyStarted: 'Paper strategy started',
        startFailed: 'Start failed',
        strategyStopped: 'Paper strategy stopped',
        stopFailed: 'Stop failed'
      }
    },
    aiChat: {
      title: 'AI Chat',
      you: 'You',
      ai: 'AI',
      revise: 'revise',
      feedback: '🔄 feedback',
      streaming: 'streaming',
      analyzing: 'analyzing',
      reset: 'reset',
      applyCode: 'Apply Code',
      dismiss: 'Dismiss',
      reviewCode: 'AI generated code — review the chat above before applying.'
    },
    assetAnalysis: {
      title: 'AI Asset Analysis',
      subtitle: 'Multi-timeframe trend outlook, S/R level detection, volatility classification, and AI strategy recommendation',
      symbolPlaceholder: 'Enter symbol (e.g. EURUSD, XAUUSD, BTCUSD)',
      analyze: 'Analyze',
      fetchingData: 'Fetching market data...',
      phase: 'Phase: {{phase}}',
      mtfOutlook: 'Multi-Timeframe Outlook',
      srLevels: 'Support / Resistance Levels',
      volatility: 'Volatility',
      state: 'State',
      atrPct: 'ATR %',
      aiRecommendation: 'AI Strategy Recommendation',
      aiUnavailable: 'AI recommendation unavailable. Please configure an AI provider in Settings.',
      configureAI: 'Configure AI Provider',
      noLevels: 'No significant levels detected',
      noResults: 'No analysis results returned. Try a different symbol.',
      volLow: 'Low volatility — consider breakout or mean-reversion strategies with tight stops.',
      volNormal: 'Normal volatility — suitable for most strategy types.',
      volHigh: 'High volatility — wider stops recommended; trend-following and breakout strategies favored.',
      volExtreme: 'Extreme volatility — reduce position sizes significantly; wide stops required.'
    },
    gen: {
      title: 'Strategy Generation',
      send: 'Generate Strategy',
      regenerate: 'Regenerate',
      reset: 'Start Over',
      template: 'テンプレート',
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
      },
      metrics: {
        sharpe: 'Sharpe',
        maxDrawdown: 'Max DD',
        winRate: 'Win',
        trades: 'Trades',
        return: 'Return'
      }
    },
    codeAssist: {
      tabAI: 'AI revise',
      tabExplain: 'Explain code',
      explain: 'Explain code',
      requiredParamsTitle: 'Required parameters',
      requiredParamsDesc: 'The strategy reads these parameters but no default was provided. Fill them in before saving.',
      optionalParamsTitle: 'Optional parameters',
      optionalParamsDesc: 'These parameters already have defaults from the code. Leave a field blank to use the default; entering a value only applies to this run and does not modify the saved strategy.',
      defaultLabel: 'default',
      paramDescriptions: {
        riskLevel: 'Risk level (low / medium / high). Controls position size and stop/take-profit width.',
        takeProfit: 'Take-profit distance (%). Close the position once price moves this far in your favour.',
        stopLoss: 'Stop-loss distance (%). Close the position once price moves this far against you.',
        maxLoss: 'Max loss per trade as a fraction of equity (0.01 = 1%).',
        confidence: 'Signal confidence threshold (0-1). Signals below this value are ignored.',
        threshold: 'Threshold that triggers a signal. Exact meaning depends on the strategy logic.',
        lotSize: 'Order size (lots / volume). Larger size means more risk.',
        fastPeriod: 'Fast period (number of bars). Used by MACD / dual-MA; smaller is more reactive.',
        slowPeriod: 'Slow period (number of bars). Used by MACD / dual-MA; larger is smoother.',
        signalPeriod: 'Signal period (number of bars). Smoothing length for MACD DIF/DEA.',
        rsiPeriod: 'RSI lookback (number of bars). Typical value: 14.',
        emaPeriod: 'EMA (exponential moving average) lookback in bars.',
        smaPeriod: 'SMA (simple moving average) lookback in bars.',
        genericPeriod: 'Lookback window in bars used for indicator calculation.',
        genericPercent: 'Percentage / ratio parameter (e.g. 1 means 1%).'
      },
      required: 'required',
      suggested: 'suggested',
      applyAllSuggestions: 'Apply suggested defaults',
      fillRequiredParams: 'Please fill the required parameters: {{keys}}',
      aiReviseTitle: 'AI assistant — revise code',
      reviseInputPlaceholder: 'e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.',
      reviseSend: 'AI に修正を依頼',
      enterInstruction: 'Please describe what you want to change.',
      codeEmpty: 'There is no code to revise yet.',
      codeUpdated: 'Code updated. Please re-run validation before saving.',
      noPython: 'AI did not return a Python block. Try rephrasing.',
      saveBlockedNotValidated: 'Please click "Validate code" first. Save is disabled until validation passes.',
      generatePlaceholder: 'Describe your strategy requirements...'
    },
    marketRegime: {
      title: 'Market Regime Detection',
      subtitle: 'Analyzes trend direction, volatility regime, and price efficiency from historical K-line data to classify current market conditions.',
      ruleVersionAlert: 'Currently using rule-based detection model rule-v1, driven by real-time K-line market data.',
      detectSuccess: 'Market regime detection completed',
      detectFailed: 'Market regime detection failed',
      form: {
        title: 'Detection Parameters',
        accountId: 'Account ID',
        accountIdRequired: 'Account ID is required',
        accountIdPlaceholder: 'MT account UUID',
        symbol: '銘柄',
        symbolRequired: '銘柄を選択してください',
        symbolPlaceholder: 'EURUSD',
        timeframe: '時間足',
        klineCount: 'K-line Count',
        submit: 'Start Detection'
      },
      result: {
        title: 'Detection Result',
        status: '状態',
        confidence: 'Confidence',
        modelVersion: 'Model Version',
        strategyFamilies: 'Strategy Families',
        features: 'Features',
        recordId: 'Record ID'
      }
    },
    experiment: {
      title: 'Strategy Experiment',
      subtitle: 'Submit parameter combinations to automatically run experiments, score candidate strategies, and generate drafts.',
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
          status: '状態',
          searchMethod: 'Search Method',
          maxCandidates: 'Max Candidates',
          objective: 'Objective',
          actions: '操作',
          viewCandidates: 'View Candidates'
        }
      },
      candidates: {
        title: 'Candidates',
        titleWithId: 'Candidates: {{id}}',
        column: {
          rank: 'Rank',
          grade: 'Grade',
          score: 'スコア',
          parameters: 'Parameters',
          summary: 'Summary',
          recommendation: 'Recommendation',
          actions: '操作',
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
      account: '口座',
      accountPlaceholder: 'Account ID',
      chartWindow: 'Chart',
      backtestRunIdLabel: 'Select backtest run...',
      hideCode: 'Hide Code',
      showCode: 'Show Code',
      investorReadOnly: '投资者(只读)',
      masterTrading: 'Master (Trading)',
      riskControls: 'Risk Controls from Code',
      jumpToCode: 'Jump to code',
      runningStatus: 'Running...',
      completedStatus: '完了',
      backtestResultsLabel: 'Backtest Results',
      watchlist: 'Watchlist',
      selectAccount: '选择账户',
      openPositions: 'Open Positions ({{count}})',
      noOpenPositions: 'No open positions for this account',
      chartError: 'Chart error — try refreshing',
      smartTuning: 'Smart Tuning',
      quickTrade: 'Quick Trade',
      quickTradeHint: 'Select a symbol first',
      selectSymbolHint: 'Select a trading account and symbol to view chart',
      noAccounts: 'No available accounts',
      selectSymbol: '銘柄',
      code: 'Strategy Code',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
      validate: 'コード検証',
      validatePass: '検証に成功しました',
      validateFailed: '検証に失敗しました',
      validateBeforeSave: 'Please validate code before saving',
      runBacktest: 'Run Backtest',
      save: 'Save',
      copy: 'コピー',
      copySuccess: 'コピーしました',
      copyFailed: 'コピーに失敗しました。手動でコピーしてください',
      saveSuccess: 'Saved',
      chart: 'K-line',
      backtest: 'バックテスト',
      backtestRunning: 'Backtest running...',
      backtestCompleted: '完了',
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
        title: 'テンプレート',
        selectPlaceholder: 'Select a template...',
        load: 'Load',
        saveAs: 'Save As New',
        loaded: 'Loaded'
      },
      quickTradeSection: {
        selectSymbol: 'Select a symbol first',
        validVolume: 'Enter a valid volume',
        priceRequired: 'Price is required for Limit/Stop orders',
        orderPlaced: '{{side}} order placed',
        orderFailed: '注文に失敗しました',
        amountLots: 'Amount (lots)',
        marginMode: 'Margin Mode',
        cross: 'Cross',
        isolated: 'Isolated',
        mt4CrossOnly: 'MT4 supports Cross margin only'
      },
      chartTools: {
        streamActive: 'Live bar stream active',
        streamUnavailable: 'Stream unavailable',
        hide: 'Hide',
        show: 'Show',
        settings: 'Settings',
        remove: 'Remove',
        clearDrawings: 'Clear All Drawings',
        candle: 'Candle',
        ohlc: 'OHLC',
        area: 'Area',
        live: 'LIVE',
        error: 'ERROR',
        static: 'STATIC'
      },
      gateTab: 'Gate'
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
      title: 'バックテスト',
      currentDraft: '📝 Current Draft',
      dateRange: 'Date Range',
      execution: 'Execution',
      capital: 'Capital',
      leverage: 'Leverage',
      commission: 'Commission',
      slippage: 'Slippage',
      trade: '取引',
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
      enterCodeAndSymbol: 'Please enter strategy code and select a symbol',
      backtestFailed: 'Backtest failed',
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
      preview: 'シグナルプレビュー',
      previewTitle: 'Preview ({{shown}} of {{total}})',
      truncated: 'TRUNCATED',
      results: 'Results ({{count}})',
      rank: '#',
      grade: 'Grade',
      score: 'スコア',
      parameters: 'Parameters',
      summary: 'Summary',
      oosScore: 'OOS Score',
      degradation: 'Degradation',
      overfit: 'Overfit',
      overfitWarning: '⚠ OVERFIT',
      apply: 'Apply',
      run: 'Run ({{count}})',
      started: 'Smart Tuning started',
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
    },
    ai: {
      checkSettings: 'Check AI Settings',
      refreshFailed: 'Refresh failed',
      settings: 'AI Settings'
    },
    backtest: {
      annualReturn: 'Annual Return',
      equityCurve: 'Equity Curve',
      maxDrawdown: 'Max Drawdown',
      sharpe: 'Sharpe',
      totalReturn: 'Total Return',
      totalTrades: 'Total Trades',
      winRate: 'Win Rate',
      tradeLog: 'Trade Log',
      tradeTime: '時間',
      tradeSide: '方向',
      tradePrice: 'Price',
      tradeVolume: 'Volume'
    },
    chartTools: {
      clearDrawings: 'Clear All Drawings',
      hide: 'Hide',
      show: 'Show',
      settings: 'Settings',
      remove: 'Remove'
    },
    quickTradeSection: {
      amountLots: 'Amount (Lots)',
      marginMode: 'Margin Mode',
      cross: 'Cross',
      isolated: 'Isolated',
      mt4CrossOnly: 'MT4 only supports Cross margin',
      selectSymbol: 'Please select a symbol',
      validVolume: 'Volume must be ≥ 0.01 lots',
      priceRequired: 'Price is required',
      orderPlaced: 'Order placed successfully',
      orderFailed: '注文に失敗しました'
    },
    library: {
      title: 'Strategy Library',
      myStrategies: 'My Strategies',
      create: 'スケジュール作成',
      filterAll: 'All',
      filterMine: 'My',
      filterSystem: 'プリセット',
      searchPlaceholder: 'Search strategies...',
      empty: 'No strategies found',
      system: 'System',
      shared: 'Shared',
      private: '非公開',
      share: '共有',
      published: '公開済み',
      draft: '下書き',
      unpublish: 'Unpublish',
      unpublishShort: 'Off',
      publish: 'Publish to Market',
      publishSuccess: '公開済み',
      unpublishSuccess: 'Unpublished',
      publishStatus: 'Marketplace Status',
      selectHint: 'Select a strategy from the list to view details',
      overview: 'Overview',
      schedules: 'Run',
      backtestHistory: 'Backtest History',
      scheduleCount: '{{count}} running',
      scheduleRunningCount: '{{count}} running',
      noSchedules: 'Not running',
      openInWorkspace: 'Open in Workspace',
      createSchedule: 'Create Run',
      saveAsMine: 'Save as Mine',
      saveAsMineSuccess: 'Saved to My Strategies',
      myCopy: 'My Copy',
      codePreview: 'Code Preview',
      viewCode: 'View Strategy Code',
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
