const strategy = {
  strategy: {
    templates: {
      title: 'Templates',
      tabs: {
        system: 'System templates',
        user: 'User templates'
      },
      table: {
        name: 'Name',
        description: 'Description',
        tags: 'Tags',
        visibility: 'Visibility',
        status: 'Status',
        useCount: 'Use count',
        createdAt: 'Created at',
        updatedAt: 'Updated at',
        actions: 'Actions',
        loadingDefault: 'Loading default templates...',
        defaultHint: 'Default',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      badges: {
        preset: 'Preset'
      },
      visibility: {
        public: 'Public',
        private: 'Private'
      },
      status: {
        draft: 'Draft',
        published: 'Published'
      },
      actions: {
        create: 'New Template',
        edit: 'Edit',
        delete: 'Delete',
        backtest: 'Backtest',
        viewCode: 'View code',
        copy: 'Copy',
        launchSchedule: 'Launch schedule',
        createTemplate: 'Create template'
      },
      copySuffix: ' (copy)',
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      scheduleName: 'Schedule name: {{name}}',
      scheduleLaunch: {
        title: 'Launch schedule',
        noRun: 'No backtest run',
        backtestRunningHint: 'Backtest is running. Please wait.',
        score: 'Score',
        keyMetrics: 'Key metrics',
        launchSection: 'Launch schedule',
        actions: {
          publishTemplate: 'Publish template',
          createScheduleNoEnable: 'Create schedule',
          createAndEnable: 'Create & enable',
          create: 'Create Schedule',
          addAccount: 'Add Account',
          updateTradingPassword: 'Update Trading Password'
        },
        metrics: {
          totalReturn: 'Total return',
          annualReturn: 'Annual return',
          maxDrawdown: 'Max drawdown',
          sharpe: 'Sharpe ratio',
          winRate: 'Win rate',
          totalTrades: 'Total trades'
        },
        form: {
          account: 'Account',
          accountPlaceholder: 'Select account',
          scheduleName: 'Schedule Name',
          scheduleNamePlaceholder: 'Enter schedule name',
          scheduleNameMax: 'Max 64 characters',
          scheduleType: 'Schedule Type',
          scheduleTypes: {
            interval: 'Interval',
            hfQuote: 'High-Freq Quote',
            klineClose: 'K-line Close'
          },
          intervalMs: 'Interval (ms)',
          intervalMsTip: 'Minimum 1000ms for non-HF modes',
          hfCooldownMs: 'HF Cooldown (ms)',
          hfCooldownMsTip: 'Cooldown between quote-driven executions',
          symbol: 'Symbol',
          symbolPlaceholder: 'Select symbol',
          symbolPlaceholderEmpty: 'No symbols configured',
          timeframe: 'Timeframe',
          defaultVolume: 'Default Volume (lots)',
          defaultVolumeTip: 'Default order volume per signal',
          enableAfterCreate: 'Enable after creation',
          riskSection: 'Risk Controls',
          maxDrawdownPct: 'Max Drawdown %',
          maxDrawdownPctTip: 'Auto-stop if drawdown exceeds this threshold',
          maxPositions: 'Max Positions',
          maxPositionsTip: 'Maximum concurrent open positions',
          stopLossOffset: 'Stop Loss Offset',
          stopLossOffsetTip: 'SL offset from entry price (pips)',
          takeProfitOffset: 'Take Profit Offset',
          takeProfitOffsetTip: 'TP offset from entry price (pips)',
          strategyParamsSection: 'Strategy Parameters',
          investorTag: 'Investor (Read-only)'
        },
        noAccountTitle: 'No Account',
        noAccountBody: 'You need to bind an MT account before launching a schedule.',
        investorWarningTitle: 'Investor Account',
        investorWarningBody: 'This account is in investor (read-only) mode. You need trading permission to launch schedules.',
        errorInvestorAccount: 'Cannot launch schedule with investor-only account. Update trading password to enable trading.',
        verifyingPermission: 'Verifying trading permission...',
        tradePermissionOk: 'Trading permission verified',
        updatePasswordTitle: 'Update Trading Password',
        updatePasswordHint: 'Enter the trading password for this account to enable trading.',
        updatePasswordOk: 'Trading password updated',
        updatePasswordFailed: 'Failed to update trading password',
        updatePasswordStillInvestor: 'Password update succeeded but account still in investor mode. Contact support.',
        newPasswordPlaceholder: 'Enter new trading password'
      },
      editTemplateModal: {
        title: {
          edit: 'Edit template',
          create: 'Create template'
        },
        actions: {
          validateCode: 'Validate code'
        },
        fields: {
          name: 'Name',
          description: 'Description',
          code: 'Code',
          publicShare: 'Public'
        },
        validation: {
          nameRequired: 'Name is required',
          codeRequired: 'Code is required'
        },
        placeholders: {
          name: 'Enter name',
          description: 'Enter description',
          codeSample: 'Paste strategy code here'
        }
      },
      codeModal: {
        title: 'Strategy code',
        actions: {
          copy: 'Copy'
        }
      },
      backtest: {
        title: 'Backtest',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: 'Strategy Parameters'
        },
        fields: {
          title: 'Title',
          account: 'Account',
          symbol: 'Symbol',
          timeframe: 'Timeframe',
          initialCapital: 'Initial capital',
          range: 'Range',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        validation: {
          accountRequired: 'Account is required',
          symbolRequired: 'Symbol is required',
          timeframeRequired: 'Timeframe is required',
          initialCapitalRequired: 'Initial capital is required',
          rangeRequired: 'Range is required'
        },
        placeholders: {
          account: 'Select an account',
          symbol: 'Select a symbol',
          range: 'Select range',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
        },
        accountDisabledSuffix: ' (disabled)',
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
        enterStrategyCode: 'Please enter strategy code',
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
        copyFailed: 'Copy failed',
        strategyCodeEmptyCannotPublish: 'Strategy code is empty. Please save your code before publishing.',
        systemTemplateReadOnly: 'System templates are read-only. Clone to edit.'
      },
      backtestRuns: {
        title: 'Backtest runs',
        empty: 'No backtest runs',
        table: {
          title: 'Title',
          status: 'Status',
          symbol: 'Symbol',
          timeframe: 'Timeframe',
          createdAt: 'Created at',
          actions: 'Actions'
        },
        actions: {
          view: 'View',
          launchSchedule: 'View score',
          createSchedule: 'Create schedule'
        },
        deleteConfirm: 'Delete this run?',
        batchDeleteConfirm: 'Delete {{count}} backtest report(s)?',
        batchDeleteSuccess: '{{count}} backtest report(s) deleted',
        status: {
          queued: 'Queued',
          running: 'Running',
          completed: 'Completed',
          failed: 'Failed',
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
      passed: 'Validation passed',
      notPassed: 'Validation did not pass',
      riskEval: {
        title: 'Risk Assessment',
        riskHigh: 'Risk level: high',
        riskUnreliable: 'Risk assessment: unreliable (isReliable=false)',
        riskLoading: 'Backend risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: 'Strategy editor',
      labels: {
        code: 'Strategy code',
        account: 'Account',
        symbol: 'Symbol',
        timeframe: 'Timeframe',
        disabledSuffix: ' (disabled)'
      },
      actions: {
        copy: 'Copy',
        validate: 'Validate',
        preview: 'Preview',
        saveAsTemplate: 'Save as template',
        sendToAI: 'Send to AI',
        sendToAIFixTitleValidate: 'Fix validation issues',
        sendToAIFixTitlePreview: 'Fix preview issues'
      },
      placeholders: {
        code: 'Paste strategy code here',
        selectAccount: 'Select an account',
        selectAccountFirst: 'Select an account first',
        loadingSymbols: 'Loading symbols...',
        selectSymbol: 'Select a symbol',
        noSymbols: 'No symbols available'
      },
      hints: {
        previewInfo: 'Preview will execute with sample market data.'
      },
      cards: {
        validationResult: 'Validation result',
        previewResult: 'Preview result'
      },
      messages: {
        enterCode: 'Please enter strategy code',
        validateFailed: 'Validation failed',
        validateError: 'Validation error',
        validateOk: 'Validation passed',
        selectAccount: 'Please select an account',
        previewOk: 'Preview completed',
        previewSuccess: 'Preview succeeded',
        previewFailed: 'Preview failed',
        execFailed: 'Execution failed',
        savedAsTemplate: 'Saved as template',
        copied: 'Copied',
        copyFailed: 'Copy failed'
      },
      aiPrompt: {
        intro: 'Please help fix the strategy based on the following issues:',
        problem: 'Problem',
        currentCodeTitle: 'Current code',
        pythonFenceStart: '```python',
        fenceEnd: '```',
        outputTitle: 'Output fixed code',
        outro: 'Return only the fixed code wrapped in ```python```.'
      }
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
        error: 'Error',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe'
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
    scheduleLogs: {
      title: 'Schedule logs',
      titleWithName: 'Schedule logs: {{name}}',
      messages: {
        missingScheduleId: 'Missing schedule ID'
      },
      execStatus: {
        pending: 'Pending',
        running: 'Running',
        completed: 'Completed',
        failed: 'Failed',
        skipped: 'Skipped'
      },
      operationStatus: {
        success: 'Success',
        failed: 'Failed',
        running: 'Running'
      },
      execTable: {
        time: 'Time',
        action: 'Action',
        execute: 'Execute',
        status: 'Status',
        durationMs: 'Duration (ms)',
        error: 'Error'
      },
      ordersTable: {
        time: 'Time',
        side: 'Side',
        symbol: 'Symbol',
        lots: 'Lots',
        openPrice: 'Open price',
        closePrice: 'Close price',
        profit: 'P/L',
        ticket: 'Ticket'
      },
      orderSide: {
        buy: 'Market buy',
        sell: 'Market sell',
        close: 'Close',
        buyLimit: 'Buy limit',
        sellLimit: 'Sell limit',
        buyStop: 'Buy stop',
        sellStop: 'Sell stop',
        buyStopLimit: 'Buy stop limit',
        sellStopLimit: 'Sell stop limit'
      },
      scheduleIdLabel: 'Schedule ID:',
      summary: {
        name: 'Name',
        status: 'Status',
        trade: 'Trade',
        enableCount: 'Enabled count',
        lastRun: 'Last run',
        lastError: 'Last error'
      },
      tabs: {
        exec: 'Executions',
        orders: 'Orders',
        execLogs: 'Execution Logs',
        orderLogs: 'Order Logs'
      },
      status: {
        success: 'Success',
        failed: 'Failed'
      },
      action: {
        start: 'Start',
        stop: 'Stop',
        restart: 'Restart'
      }
    },
    schedules: {
      title: 'Schedules',
      createSchedule: 'Create Schedule',
      format: {
        interval: 'every {{s}}s',
        cron: 'cron: {{expr}}',
      },
      status: {
        running: 'Running',
        disabled: 'Disabled'
      },
      templateVisibility: {
        public: 'Public',
        private: 'Private'
      },
      table: {
        name: 'Name',
        template: 'Template',
        account: 'Account',
        tradeParams: 'Trade params',
        schedule: 'Schedule',
        status: 'Status',
        lastRun: 'Last run',
        actions: 'Actions'
      },
      nextRunAt: 'Next run at',
      enableCount: 'Enable count',
      actions: {
        create: 'Create',
        logs: 'Logs',
        healthCheck: 'Health check',
        runNow: 'Run now'
      },
      health: {
        title: 'Strategy health check {{name}}',
        summaryBanner: 'Grade: {{grade}}; samples: {{totalRuns}}, success rate: {{successRate}}%',
        grade: {
          pending: 'Pending',
          noSample: 'No sample',
          healthy: 'Healthy',
          watch: 'Watch',
          alert: 'Alert'
        },
        notes: {
          pending: 'Run health check first.',
          noSample: 'Not enough samples to evaluate (minimum {{minSampleSize}}).',
          healthy: 'High success rate and controlled failures.',
          watch: 'Success rate is acceptable but should be monitored (>= {{yellowSuccessRate}}%).',
          alert: 'Low success rate. Investigate strategy/account conditions now.'
        },
        fields: {
          grade: 'Health grade',
          rule: 'Rule',
          thresholds: 'Current thresholds',
          configKey: 'Config key',
          lastRunAt: 'Last run',
          latestTicket: 'Latest filled ticket',
          successOverTotal: 'Success / Total',
          failedRuns: 'Failed runs',
          latestProfit: 'Latest profit',
          latestError: 'Latest error'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}, green: success>={{greenSuccessRate}}% & failed<={{greenMaxFailedRuns}}, yellow: success>={{yellowSuccessRate}}%',
        sections: {
          runLogs: 'Recent execution logs',
          orders: 'Recent order records'
        },
        runLogs: {
          signalType: 'Signal'
        },
        messages: {
          loadFailed: 'Failed to load health data',
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
        parametersParseFailed: 'Failed to parse parameters',
        defaultTemplateNotFound: 'Default template not found',
        importDefaultTemplateFailedNoId: 'Failed to import default template (missing id)',
        templateCodeEmptyCannotExecute: 'Template code is empty. Cannot execute.',
        strategyExecuteFailed: 'Strategy execution failed',
        executeFailed: 'Execution failed',
        noOrderableSignal: 'No orderable signal',
        signalHoldCannotOrder: 'Signal is HOLD. Cannot place order.',
        volumeInvalid: 'Invalid volume',
        orderSubmitted: 'Order submitted',
        orderFailed: 'Order failed'
      },
      editModal: {
        title: {
          edit: 'Edit schedule',
          create: 'Create schedule'
        },
        fields: {
          template: 'Template',
          templateExtra: 'Select a template to run',
          account: 'Account',
          name: 'Name',
          symbol: 'Symbol',
          lot: 'Lots',
          lotExtra: 'Lots per trade',
          runFrequency: 'Run frequency',
          cronExpression: 'Cron expression',
          cronExtra: 'Use cron format to schedule runs',
          intervalSeconds: 'Interval (seconds)',
          intervalSecondsExtra: 'Run every N seconds',
          enableExtra: 'Enable schedule after creating'
        },
        placeholders: {
          name: 'Enter schedule name',
          selectAccountFirst: 'Select an account first',
          symbol: 'Select a symbol'
        },
        validation: {
          templateRequired: 'Template is required',
          accountRequired: 'Account is required',
          nameRequired: 'Name is required',
          symbolRequired: 'Symbol is required',
          lotRequired: 'Lots is required',
          runFrequencyRequired: 'Run frequency is required',
          cronRequired: 'Cron expression is required',
          timeframeRequired: 'Timeframe is required',
          triggerModeRequired: 'Trigger mode is required'
        },
        runFrequencyExtra: {
          cron: 'Run by cron expression',
          byTimeframe: 'Run by timeframe'
        },
        runFrequencyOptions: {
          byTimeframe: 'By timeframe',
          cron: 'Cron'
        },
        autoName: {
          strategy: 'Strategy'
        },
        advanced: {
          title: 'Advanced',
          fixedIntervalSeconds: 'Fixed interval (seconds)',
          fixedIntervalSecondsExtra: 'Override default interval',
          timeframe: 'Timeframe',
          timeframeExtra: 'Select timeframe for execution',
          triggerMode: 'Trigger mode',
          triggerModeExtra: 'Choose when to trigger signals',
          triggerModeOptions: {
            stable: 'Stable K-line',
            hf: 'High-frequency signal stream'
          },
          stableOverrideIntervalSeconds: 'Stable override interval (seconds)',
          stableOverrideIntervalSecondsExtra: 'Override stable timeframe interval',
          hfCooldownMs: 'HF cooldown (ms)',
          hfCooldownMsExtra: 'Minimum interval between HF signals',
          parametersJson: 'Parameters (JSON)',
          parametersJsonExtra: 'JSON parameters for the strategy'
        }
      },
      triggerModal: {
        title: 'Trigger schedule',
        confirmOrder: {
          title: 'Confirm order',
          ok: 'Confirm'
        },
        actions: {
          confirmOrder: 'Confirm order',
          rerun: 'Re-run'
        },
        summary: {
          scheduleName: 'Schedule name',
          account: 'Account',
          symbol: 'Symbol',
          timeframe: 'Timeframe'
        },
        messages: {
          signalNotOrderable: 'Signal is not orderable'
        },
        cards: {
          logs: 'Logs',
          signal: 'Signal'
        },
        emptyLogs: 'No logs',
        emptySignal: 'No signal'
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
      },
      empty: 'No strategy assets yet'
    },
    paper: {
      title: '📊 Paper Trading',
      createAccount: 'Create Paper Account',
      accountName: 'Account name',
      create: 'Create',
      noAccounts: 'No paper accounts. Create one to start simulated trading.',
      running: 'Running {{symbol}} {{timeframe}}',
      start: 'Start',
      stop: 'Stop',
      watch: 'Watch',
      paper: 'Paper',
      startStrategy: 'Start Paper Strategy',
      symbol: 'Symbol',
      timeframe: 'Timeframe',
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
      reviseSend: 'Send to AI',
      enterInstruction: 'Please describe what you want to change.',
      codeEmpty: 'There is no code to revise yet.',
      codeUpdated: 'Code updated. Please re-run validation before saving.',
      noPython: 'AI did not return a Python block. Try rephrasing.',
      saveBlockedNotValidated: 'Please click "Validate code" first. Save is disabled until validation passes.',
      generatePlaceholder: 'Describe your strategy requirements...'
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
      backtestRunIdLabel: 'Select backtest run...',
      hideCode: 'Hide Code',
      showCode: 'Show Code',
      investorReadOnly: 'Investor (Read-only)',
      masterTrading: 'Master (Trading)',
      riskControls: 'Risk Controls from Code',
      jumpToCode: 'Jump to code',
      runningStatus: 'Running...',
      completedStatus: 'Completed',
      backtestResultsLabel: 'Backtest Results',
      watchlist: 'Watchlist',
      selectAccount: 'Select account',
      openPositions: 'Open Positions ({{count}})',
      noOpenPositions: 'No open positions for this account',
      chartError: 'Chart error — try refreshing',
      smartTuning: 'Smart Tuning',
      quickTrade: 'Quick Trade',
      quickTradeHint: 'Select a symbol first',
      selectSymbolHint: 'Select a trading account and symbol to view chart',
      noAccounts: 'No available accounts',
      selectSymbol: 'Symbol',
      code: 'Strategy Code',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
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
      },
      quickTradeSection: {
        selectSymbol: 'Select a symbol first',
        validVolume: 'Enter a valid volume',
        priceRequired: 'Price is required for Limit/Stop orders',
        orderPlaced: '{{side}} order placed',
        orderFailed: 'Order failed',
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
      tradeTime: 'Time',
      tradeSide: 'Side',
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
      orderFailed: 'Order failed'
    },
    library: {
      title: 'Strategy Library',
      myStrategies: 'My Strategies',
      create: 'Create',
      filterAll: 'All',
      filterMine: 'My',
      filterSystem: 'Preset',
      searchPlaceholder: 'Search strategies...',
      empty: 'No strategies found',
      system: 'System',
      shared: 'Shared',
      private: 'Private',
      share: 'Share',
      published: 'Published',
      draft: 'Draft',
      unpublish: 'Unpublish',
      unpublishShort: 'Off',
      publish: 'Publish to Market',
      publishSuccess: 'Published',
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
    title: 'Indicator Catalog',
    description: 'Technical indicators and risk parameters available in the strategy sandbox. Use only these helpers and parameter keys in your strategy code.',
    indicatorsTitle: 'Technical Indicators',
    riskSectionTitle: 'Risk Management Parameters',
    riskParamsTitle: 'Universal Risk Parameters',
    riskParamsDesc: 'Every strategy should respect these risk-management parameters regardless of which indicators are selected.',
    paramKey: 'Key',
    paramLabel: 'Label',
    paramType: 'Type',
    paramDefault: 'Default',
    paramRange: 'Range',
    paramDescription: 'Description'
  }
} as const;

export default strategy;
