const strategy = {
  strategy: {
    templates: {
      title: '策略模板',
      tabs: {
        system: '系統模板',
        user: 'User templates'
      },
      table: {
        name: '名稱',
        description: '描述',
        tags: '標籤',
        visibility: '可見性',
        status: '狀態',
        useCount: '使用次數',
        createdAt: '建立時間',
        updatedAt: '更新時間',
        actions: '操作',
        loadingDefault: '正在載入預設模板...',
        defaultHint: '預設值',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      badges: {
        preset: '預設'
      },
      visibility: {
        public: '公開',
        private: '私有'
      },
      status: {
        draft: '草稿',
        published: '已發布'
      },
      actions: {
        create: '新建模板',
        edit: '編輯',
        delete: '刪除',
        backtest: '回測',
        viewCode: '查看代碼',
        copy: '複製',
        launchSchedule: '上線調度',
        createTemplate: 'Create template'
      },
      copySuffix: ' (副本)',
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
      scheduleLaunch: {
        title: '上線調度',
        noRun: '暫無回測運行',
        backtestRunningHint: '回測正在運行，請稍候。',
        score: '評分',
        keyMetrics: '關鍵指標',
        launchSection: '上線調度',
        actions: {
          publishTemplate: '發布模板',
          createScheduleNoEnable: '新建調度任務',
          createAndEnable: '建立並啟用',
          create: '建立調度',
          addAccount: '新增账戶',
          updateTradingPassword: '更新交易密碼'
        },
        metrics: {
          totalReturn: '總收益',
          annualReturn: '年化收益',
          maxDrawdown: '最大回撤',
          sharpe: '夏普比率',
          winRate: '勝率',
          totalTrades: '交易次數'
        },
        form: {
          account: '帳號',
          accountPlaceholder: '選擇账戶',
          scheduleName: '计划名稱',
          scheduleNamePlaceholder: '例如：EURUSD M5 早盤策略',
          scheduleNameMax: '最多64字元',
          scheduleType: '排程類型',
          scheduleTypes: {
            interval: '定時執行',
            hfQuote: '高頻報價',
            klineClose: 'K-line Close'
          },
          intervalMs: '間隔(毫秒)',
          intervalMsTip: '非高頻模式最小1000ms',
          hfCooldownMs: '高頻冷卻(毫秒)',
          hfCooldownMsTip: '报价驱动执行间的冷却時間',
          symbol: '品種',
          symbolPlaceholder: '選擇商品',
          symbolPlaceholderEmpty: '未配置商品',
          timeframe: '週期',
          defaultVolume: '默认手數',
          defaultVolumeTip: '每個信号的默认下單量',
          enableAfterCreate: '建立后立即啟用',
          riskSection: '風控設定',
          maxDrawdownPct: '最大回撤%',
          maxDrawdownPctTip: '回撤超過此閾值自动停止',
          maxPositions: '最大持仓數',
          maxPositionsTip: '同時持有的最大仓位元數量',
          stopLossOffset: '止損偏移',
          stopLossOffsetTip: '距入場價的止損距離(點)',
          takeProfitOffset: '止盈偏移',
          takeProfitOffsetTip: '距入場價的止盈距離(點)',
          strategyParamsSection: '策略参數',
          investorTag: '投資者(唯讀)'
        },
        noAccountTitle: '无账戶',
        noAccountBody: '啟動计划前需要先绑定MT账戶。',
        investorWarningTitle: '投資者账戶',
        investorWarningBody: '此账戶为投資者(唯讀)模式，需要交易權限才能啟動计划。',
        errorInvestorAccount: '无法使用投資者账戶啟動计划。请更新交易密碼以啟用交易。',
        verifyingPermission: '驗證交易權限中...',
        tradePermissionOk: '交易權限驗證通過',
        updatePasswordTitle: '更新交易密碼',
        updatePasswordHint: '輸入此账戶的交易密碼以啟用交易。',
        updatePasswordOk: '交易密碼已更新',
        updatePasswordFailed: '更新交易密碼失敗',
        updatePasswordStillInvestor: '密碼更新成功但账戶仍为投資者模式，請聯絡客服。',
        newPasswordPlaceholder: 'Enter new trading password'
      },
      editTemplateModal: {
        title: {
          edit: '編輯模板',
          create: 'Create template'
        },
        actions: {
          validateCode: 'Validate code'
        },
        fields: {
          name: '名稱',
          description: '描述',
          code: '策略代碼',
          publicShare: '公開'
        },
        validation: {
          nameRequired: '請輸入名稱',
          codeRequired: 'Code is required'
        },
        placeholders: {
          name: '例如：均線交叉策略',
          description: '可選：策略說明',
          codeSample: '輸入Python策略代碼...'
        }
      },
      codeModal: {
        title: '策略代碼',
        actions: {
          copy: '複製'
        }
      },
      backtest: {
        title: '回測',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: '策略参數'
        },
        fields: {
          title: 'Title',
          account: '帳號',
          symbol: '品種',
          timeframe: '週期',
          initialCapital: 'Initial capital',
          range: '範圍',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        validation: {
          accountRequired: '請選擇帳號',
          symbolRequired: '請輸入品種',
          timeframeRequired: '請選擇週期',
          initialCapitalRequired: 'Initial capital is required',
          rangeRequired: 'Range is required'
        },
        placeholders: {
          account: '選擇帳號',
          symbol: '選擇品種',
          range: 'Select range',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
        },
        accountDisabledSuffix: '（已禁用）',
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
        enterStrategyCode: '請輸入策略代碼',
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
        copyFailed: '複製失敗，請手動複製',
        strategyCodeEmptyCannotPublish: 'Strategy code is empty. Please save your code before publishing.',
        systemTemplateReadOnly: 'System templates are read-only. Clone to edit.'
      },
      backtestRuns: {
        title: 'Backtest runs',
        empty: 'No backtest runs',
        table: {
          title: 'Title',
          status: '狀態',
          symbol: '品種',
          timeframe: '週期',
          createdAt: '建立時間',
          actions: '操作'
        },
        actions: {
          view: 'View',
          launchSchedule: 'View score',
          createSchedule: '新建調度任務'
        },
        deleteConfirm: 'Delete this run?',
        batchDelete: 'Delete {{count}}',
        batchDeleteConfirm: 'Delete {{count}} backtest report(s)?',
        batchDeleteSuccess: '{{count}} backtest report(s) deleted',
        status: {
          queued: 'Queued',
          running: '運行中',
          completed: '已完成',
          failed: '失敗',
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
      passed: '代碼驗證通過',
      notPassed: '代碼驗證未通過',
      riskEval: {
        title: '風險評估',
        riskHigh: '風險等級：高',
        riskUnreliable: '風險評估：不可靠（isReliable=false）',
        riskLoading: 'Risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: '策略編輯器',
      labels: {
        code: '策略代碼',
        account: '帳號',
        symbol: '品種',
        timeframe: '週期',
        disabledSuffix: '（已禁用）'
      },
      actions: {
        copy: '複製',
        validate: '驗證代碼',
        preview: '預覽訊號',
        saveAsTemplate: '保存為模板',
        sendToAI: '發給AI修改',
        sendToAIFixTitleValidate: '驗證未通過/有警告',
        sendToAIFixTitlePreview: 'Fix preview issues'
      },
      placeholders: {
        code: '輸入Python策略代碼...',
        selectAccount: '選擇帳號',
        selectAccountFirst: '先選帳號',
        loadingSymbols: '可用品種載入中…',
        selectSymbol: '選擇品種',
        noSymbols: 'No symbols available'
      },
      hints: {
        previewInfo: 'Preview will execute with sample market data.'
      },
      cards: {
        validationResult: '驗證結果',
        previewResult: 'Preview result'
      },
      messages: {
        enterCode: '請輸入策略代碼',
        validateFailed: '代碼驗證失敗',
        validateError: '驗證失敗',
        validateOk: '代碼驗證通過',
        selectAccount: '請選擇帳號',
        previewOk: '預覽完成',
        previewSuccess: '預覽成功',
        previewFailed: '預覽失敗',
        execFailed: '執行失敗',
        savedAsTemplate: '已保存為模板',
        copied: '代碼已複製',
        copyFailed: '複製失敗，請手動複製'
      },
      aiPrompt: {
        intro: '請根據以下資訊修改策略代碼，使其通過驗證並且預覽訊號執行成功。',
        problem: '【問題】{{title}}',
        currentCodeTitle: '【當前代碼】',
        pythonFenceStart: '```python',
        fenceEnd: '```',
        outputTitle: '【輸出資訊】',
        outro: 'Return only the fixed code wrapped in ```python```.'
      }
    },
    templateModal: {
      title: '保存為模板',
      fields: {
        name: '名稱',
        description: '描述'
      },
      placeholders: {
        name: 'Enter template name',
        description: '可選：策略說明'
      }
    },
    backtestRun: {
      title: 'Backtest run',
      status: {
        queued: 'Queued',
        running: '運行中',
        completed: '已完成',
        failed: '失敗',
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
        status: '狀態',
        error: '錯誤',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe'
      },
      metrics: {
        totalReturn: '總收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '勝率',
        totalTrades: '交易次數',
        equityCurvePoints: 'Equity curve points'
      },
      trades: {
        title: 'Order details',
        empty: 'No trades recorded',
        loadFailed: 'Failed to load order details',
        ticket: '訂單號',
        side: '方向',
        sideBuy: 'Buy',
        sideSell: 'Sell',
        volume: 'Volume',
        openTime: 'Open time',
        openPrice: '開倉價',
        closeTime: 'Close time',
        closePrice: '平倉價',
        pnl: 'P&L',
        commission: 'Commission',
        reason: 'Close reason',
        reasons: {
          signal: '信號(用於下單)',
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
      title: '記錄',
      titleWithName: '記錄 - {{name}}',
      messages: {
        missingScheduleId: 'Missing schedule ID'
      },
      execStatus: {
        pending: '待檢測',
        running: '運行中',
        completed: '已完成',
        failed: '失敗',
        skipped: 'Skipped'
      },
      operationStatus: {
        success: '成功',
        failed: '失敗',
        running: '運行中'
      },
      execTable: {
        time: '時間',
        action: '操作',
        execute: '執行',
        status: '狀態',
        durationMs: '耗時(ms)',
        error: '錯誤'
      },
      ordersTable: {
        time: '時間',
        side: '方向',
        symbol: '品種',
        lots: '手數(Lot)',
        openPrice: '開倉價',
        closePrice: '平倉價',
        profit: '盈虧',
        ticket: '訂單號'
      },
      orderSide: {
        buy: '市價買入',
        sell: '市價賣出',
        close: '平倉',
        buyLimit: '限價買入',
        sellLimit: '限價賣出',
        buyStop: '突破買入',
        sellStop: '突破賣出',
        buyStopLimit: '限價突破買',
        sellStopLimit: 'Sell stop limit'
      },
      scheduleIdLabel: '排程ID:',
      summary: {
        name: '名稱',
        status: '狀態',
        trade: '交易',
        enableCount: '啟用次數',
        lastRun: '最後運行時間',
        lastError: 'Last error'
      },
      tabs: {
        exec: '運行記錄',
        orders: '交易記錄',
        execLogs: '执行日誌',
        orderLogs: 'Order Logs'
      },
      status: {
        success: '成功',
        failed: '失敗'
      },
      action: {
        start: '啟動',
        stop: '停止',
        restart: 'Restart'
      }
    },
    schedules: {
      title: '策略調度',
      createSchedule: '建立調度',
      format: {
        interval: '每 {{s}}秒',
        cron: '定時: {{expr}}',
      },
      status: {
        running: '運行中',
        disabled: 'Disabled'
      },
      templateVisibility: {
        public: '公開',
        private: '私有'
      },
      table: {
        name: '名稱',
        template: '模板',
        account: '帳號',
        tradeParams: '交易參數',
        schedule: '計劃',
        status: '狀態',
        lastRun: '最後運行時間',
        actions: '操作'
      },
      nextRunAt: '下次運行',
      enableCount: '啟用次數',
      actions: {
        create: '新建調度',
        logs: '執行日誌',
        healthCheck: '健康檢查',
        runNow: 'Run now'
      },
      health: {
        title: '策略健康檢查 {{name}}',
        summaryBanner: '健康分級：{{grade}}；最近樣本 {{totalRuns}} 次，成功率 {{successRate}}%',
        grade: {
          pending: '待檢測',
          noSample: '無樣本',
          healthy: '健康',
          watch: '關注',
          alert: 'Alert'
        },
        notes: {
          pending: '請先執行健康檢查。',
          noSample: '樣本不足，至少需要 {{minSampleSize}} 筆運行記錄。',
          healthy: '成功率高且失敗次數可控。',
          watch: '成功率達到關注門檻（>= {{yellowSuccessRate}}%），建議持續觀察。',
          alert: 'Low success rate. Investigate strategy/account conditions now.'
        },
        fields: {
          grade: '健康級別',
          rule: '判定依據',
          thresholds: '當前門檻',
          configKey: '設定鍵',
          lastRunAt: '最後運行時間',
          latestTicket: '最近成交 Ticket',
          successOverTotal: '執行成功/總次數',
          failedRuns: '執行失敗次數',
          latestProfit: '最近成交盈虧',
          latestError: 'Latest error'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}；綠色：成功率>={{greenSuccessRate}}% 且失敗次數<={{greenMaxFailedRuns}}；黃色：成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近執行日誌',
          orders: 'Recent order records'
        },
        runLogs: {
          signalType: '信號(用於下單)'
        },
        messages: {
          loadFailed: '載入健康檢查資料失敗',
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
        parametersParseFailed: '參數解析失敗',
        defaultTemplateNotFound: '預設模板不存在，請刷新頁面重試',
        importDefaultTemplateFailedNoId: '匯入預設模板失敗：未返回模板ID',
        templateCodeEmptyCannotExecute: '模板 code 為空，無法執行',
        strategyExecuteFailed: '策略執行失敗',
        executeFailed: '執行失敗',
        noOrderableSignal: '沒有可下單的信號',
        signalHoldCannotOrder: '當前信號為 hold/無交易動作，不能下單',
        volumeInvalid: '下單手數無效（volume 必須 > 0）',
        orderSubmitted: '已提交下單',
        orderFailed: '下單失敗'
      },
      editModal: {
        title: {
          edit: '編輯調度任務',
          create: '新建調度任務'
        },
        fields: {
          template: '模板',
          templateExtra: '來自「策略管理」中保存的模板',
          account: '帳號',
          name: '名稱',
          symbol: '品種',
          lot: '手數(Lot)',
          lotExtra: '下單手數，建議從 0.01 開始',
          runFrequency: '運行頻率',
          cronExpression: 'Cron 表達式',
          cronExtra: '標準 5 段：分鐘 小時 日 月 週。例如：*/5 * * * * 每5分鐘；0 9 * * 1-5 工作日9點',
          intervalSeconds: '間隔(秒)',
          intervalSecondsExtra: '自動跟隨週期(timeframe)，無需修改',
          enableExtra: 'Enable schedule after creating'
        },
        placeholders: {
          name: '例如：EURUSD M5 早盤策略',
          selectAccountFirst: '先選帳號',
          symbol: '選擇品種'
        },
        validation: {
          templateRequired: '請選擇模板',
          accountRequired: '請選擇帳號',
          nameRequired: '請輸入名稱',
          symbolRequired: '請輸入品種',
          lotRequired: '請輸入手數',
          runFrequencyRequired: '請選擇運行頻率',
          cronRequired: '請輸入 cron',
          timeframeRequired: '請選擇週期',
          triggerModeRequired: 'Trigger mode is required'
        },
        runFrequencyExtra: {
          cron: '高級：使用 Cron 精確控制執行時間',
          byTimeframe: 'Run by timeframe'
        },
        runFrequencyOptions: {
          byTimeframe: '按週期觸發（推薦）',
          cron: 'Cron'
        },
        autoName: {
          strategy: 'Strategy'
        },
        advanced: {
          title: '高級設定',
          fixedIntervalSeconds: '固定間隔(秒)',
          fixedIntervalSecondsExtra: '可選。填寫後將按固定間隔執行（不會再自動跟隨週期）。例如：60 表示每 60 秒執行一次',
          timeframe: '週期',
          timeframeExtra: '預設即可。僅用於K線與指標計算，不影響EA本質（策略驅動交易）',
          triggerMode: '觸發模式',
          triggerModeExtra: '穩定：按K線/週期觸發（信號更穩但有延遲）；高頻：報價流觸發（更快但噪聲大，需要去抖）',
          triggerModeOptions: {
            stable: '穩定（K線/週期）',
            hf: 'High-frequency signal stream'
          },
          stableOverrideIntervalSeconds: '穩定模式高級：間隔(秒)',
          stableOverrideIntervalSecondsExtra: '可選。預設綁定週期(timeframe)。填寫後將覆蓋穩定模式的觸發間隔',
          hfCooldownMs: '高頻模式：最小觸發間隔(ms)',
          hfCooldownMsExtra: '用於去抖：兩次評估/下單之間的最小間隔',
          parametersJson: '參數(JSON對象)',
          parametersJsonExtra: 'JSON parameters for the strategy'
        }
      },
      triggerModal: {
        title: '立即執行(直接下單)',
        confirmOrder: {
          title: '確認下單',
          ok: 'Confirm'
        },
        actions: {
          confirmOrder: '確認下單',
          rerun: 'Re-run'
        },
        summary: {
          scheduleName: '調度名稱',
          account: '帳號',
          symbol: '品種',
          timeframe: '週期'
        },
        messages: {
          signalNotOrderable: 'Signal is not orderable'
        },
        cards: {
          logs: '執行日誌',
          signal: '信號(用於下單)'
        },
        emptyLogs: '(無日誌)',
        emptySignal: 'No signal'
      }
    },
    asset: {
      title: 'Strategy Assets',
      subtitle: 'Asset publishing, review status, and cloning are maintained by the system. Cloned results are independent user templates.',
      submitAsset: 'Submit Asset',
      assetList: 'Asset List',
      name: '名稱',
      visibility: '可見性',
      reviewStatus: 'Review Status',
      cloneCount: 'Clones',
      version: 'Version',
      description: '描述',
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
      create: '新建調度',
      noAccounts: 'No paper accounts. Create one to start simulated trading.',
      running: 'Running {{symbol}} {{timeframe}}',
      start: '啟動',
      stop: '停止',
      watch: '關注',
      paper: 'Paper',
      startStrategy: 'Start Paper Strategy',
      symbol: '品種',
      timeframe: '週期',
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
      template: '模板',
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
      reviseSend: '發給AI修改',
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
        symbol: '品種',
        symbolRequired: '請輸入品種',
        symbolPlaceholder: 'EURUSD',
        timeframe: '週期',
        klineCount: 'K-line Count',
        submit: 'Start Detection'
      },
      result: {
        title: 'Detection Result',
        status: '狀態',
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
          status: '狀態',
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
          score: '評分',
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
      account: '帳號',
      accountPlaceholder: 'Account ID',
      chartWindow: 'Chart',
      backtestRunIdLabel: 'Select backtest run...',
      hideCode: 'Hide Code',
      showCode: 'Show Code',
      investorReadOnly: '投資者(唯讀)',
      masterTrading: 'Master (Trading)',
      riskControls: 'Risk Controls from Code',
      jumpToCode: 'Jump to code',
      runningStatus: 'Running...',
      completedStatus: '已完成',
      backtestResultsLabel: 'Backtest Results',
      watchlist: 'Watchlist',
      selectAccount: '選擇账戶',
      openPositions: 'Open Positions ({{count}})',
      noOpenPositions: 'No open positions for this account',
      chartError: 'Chart error — try refreshing',
      smartTuning: 'Smart Tuning',
      quickTrade: 'Quick Trade',
      quickTradeHint: 'Select a symbol first',
      selectSymbolHint: 'Select a trading account and symbol to view chart',
      noAccounts: 'No available accounts',
      selectSymbol: '品種',
      code: 'Strategy Code',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
      validate: '驗證代碼',
      validatePass: '代碼驗證通過',
      validateFailed: '代碼驗證失敗',
      validateBeforeSave: 'Please validate code before saving',
      runBacktest: 'Run Backtest',
      save: 'Save',
      copy: '複製',
      copySuccess: '代碼已複製',
      copyFailed: '複製失敗，請手動複製',
      saveSuccess: 'Saved',
      chart: 'K-line',
      backtest: '回測',
      backtestRunning: 'Backtest running...',
      backtestCompleted: '已完成',
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
        title: '模板',
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
        orderFailed: '下單失敗',
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
      title: '回測',
      currentDraft: '📝 Current Draft',
      dateRange: 'Date Range',
      execution: 'Execution',
      capital: 'Capital',
      leverage: 'Leverage',
      commission: 'Commission',
      slippage: 'Slippage',
      trade: '交易',
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
      preview: '預覽訊號',
      previewTitle: 'Preview ({{shown}} of {{total}})',
      truncated: 'TRUNCATED',
      results: 'Results ({{count}})',
      rank: '#',
      grade: 'Grade',
      score: '評分',
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
      orderFailed: '下單失敗'
    },
    library: {
      title: 'Strategy Library',
      myStrategies: 'My Strategies',
      create: '新建調度',
      filterAll: 'All',
      filterMine: 'My',
      filterSystem: '預設',
      searchPlaceholder: 'Search strategies...',
      empty: 'No strategies found',
      system: 'System',
      shared: 'Shared',
      private: '私有',
      share: 'Share',
      published: '已發布',
      draft: '草稿',
      unpublish: 'Unpublish',
      unpublishShort: 'Off',
      publish: 'Publish to Market',
      publishSuccess: '已發布',
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
    title: '指標目錄',
    description: '策略沙箱中可用的技術指標和風險參數。請在策略代碼中僅使用這些輔助函數和參數鍵。',
    indicatorsTitle: '技術指標',
    riskSectionTitle: '風險管理參數',
    riskParamsTitle: '通用風險參數',
    riskParamsDesc: '無論選擇哪些指標，每個策略都應遵循這些風險管理參數。',
    paramKey: '鍵',
    paramLabel: '標籤',
    paramType: '類型',
    paramDefault: '預設值',
    paramRange: '範圍',
    paramDescription: '描述'
  }
} as const;

export default strategy;
