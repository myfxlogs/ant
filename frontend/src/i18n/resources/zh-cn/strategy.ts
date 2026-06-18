const strategy = {
  strategy: {
    templates: {
      title: '策略模板',
      tabs: {
        system: '系统模板',
        user: 'User templates'
      },
      table: {
        name: '名称',
        description: '描述',
        tags: '标签',
        visibility: '可见性',
        status: '状态',
        useCount: '使用次数',
        createdAt: '创建时间',
        updatedAt: '更新时间',
        actions: '操作',
        loadingDefault: '正在加载默认模板...',
        defaultHint: '默认值',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      badges: {
        preset: '预设'
      },
      visibility: {
        public: '公开',
        private: '私有'
      },
      status: {
        draft: '草稿',
        published: '已发布'
      },
      actions: {
        create: '新建模板',
        edit: '编辑',
        delete: '删除',
        backtest: '回测',
        viewCode: '查看代码',
        copy: '复制',
        launchSchedule: '上线调度',
        createTemplate: 'Create template'
      },
      copySuffix: ' (副本)',
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
      scheduleLaunch: {
        title: '上线调度',
        noRun: '暂无回测运行',
        backtestRunningHint: '回测正在运行，请稍候。',
        score: '评分',
        keyMetrics: '关键指标',
        launchSection: '上线调度',
        actions: {
          publishTemplate: '发布模板',
          createScheduleNoEnable: '新建调度任务',
          createAndEnable: '创建并启用',
          create: '创建调度',
          addAccount: '添加账户',
          updateTradingPassword: '更新交易密码'
        },
        metrics: {
          totalReturn: '总收益',
          annualReturn: '年化收益',
          maxDrawdown: '最大回撤',
          sharpe: '夏普比率',
          winRate: '胜率',
          totalTrades: '交易次数'
        },
        form: {
          account: '账号',
          accountPlaceholder: '选择账户',
          scheduleName: '计划名称',
          scheduleNamePlaceholder: '例如：EURUSD M5 早盘策略',
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
          symbol: '品种',
          symbolPlaceholder: '选择品种',
          symbolPlaceholderEmpty: '未配置品种',
          timeframe: '周期',
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
          edit: '编辑模板',
          create: 'Create template'
        },
        actions: {
          validateCode: 'Validate code'
        },
        fields: {
          name: '名称',
          description: '描述',
          code: '策略代码',
          publicShare: '公开'
        },
        validation: {
          nameRequired: '请输入名称',
          codeRequired: 'Code is required'
        },
        placeholders: {
          name: '例如：均线交叉策略',
          description: '可选：策略说明',
          codeSample: '输入Python策略代码...'
        }
      },
      codeModal: {
        title: '策略代码',
        actions: {
          copy: '复制'
        }
      },
      backtest: {
        title: '回测',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: '策略参数'
        },
        fields: {
          title: 'Title',
          account: '账号',
          symbol: '品种',
          timeframe: '周期',
          initialCapital: 'Initial capital',
          range: '范围',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        validation: {
          accountRequired: '请选择账号',
          symbolRequired: '请选择品种',
          timeframeRequired: '请选择周期',
          initialCapitalRequired: 'Initial capital is required',
          rangeRequired: 'Range is required'
        },
        placeholders: {
          account: '选择账号',
          symbol: '选择品种',
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
        enterStrategyCode: '请输入策略代码',
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
        copyFailed: '复制失败，请手动复制',
        strategyCodeEmptyCannotPublish: 'Strategy code is empty. Please save your code before publishing.',
        systemTemplateReadOnly: 'System templates are read-only. Clone to edit.'
      },
      backtestRuns: {
        title: 'Backtest runs',
        empty: 'No backtest runs',
        table: {
          title: 'Title',
          status: '状态',
          symbol: '品种',
          timeframe: '周期',
          createdAt: '创建时间',
          actions: '操作'
        },
        actions: {
          view: 'View',
          launchSchedule: 'View score',
          createSchedule: '新建调度任务'
        },
        deleteConfirm: 'Delete this run?',
        batchDelete: 'Delete {{count}}',
        batchDeleteConfirm: 'Delete {{count}} backtest report(s)?',
        batchDeleteSuccess: '{{count}} backtest report(s) deleted',
        status: {
          queued: 'Queued',
          running: '运行中',
          completed: '已完成',
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
      passed: '代码验证通过',
      notPassed: '代码验证未通过',
      riskEval: {
        title: '风险评估',
        riskHigh: '风险等级：high',
        riskUnreliable: '风险评估：不可靠（isReliable=false）',
        riskLoading: 'Risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: '策略编辑器',
      labels: {
        code: '策略代码',
        account: '账号',
        symbol: '品种',
        timeframe: '周期',
        disabledSuffix: '（已禁用）'
      },
      actions: {
        copy: '复制',
        validate: '验证代码',
        preview: '预览信号',
        saveAsTemplate: '保存为模板',
        sendToAI: '发给AI修改',
        sendToAIFixTitleValidate: '验证未通过/有警告',
        sendToAIFixTitlePreview: 'Fix preview issues'
      },
      placeholders: {
        code: '输入Python策略代码...',
        selectAccount: '选择账号',
        selectAccountFirst: '先选账号',
        loadingSymbols: '可用品种加载中…',
        selectSymbol: '选择品种',
        noSymbols: 'No symbols available'
      },
      hints: {
        previewInfo: 'Preview will execute with sample market data.'
      },
      cards: {
        validationResult: '验证结果',
        previewResult: 'Preview result'
      },
      messages: {
        enterCode: '请输入策略代码',
        validateFailed: '代码验证失败',
        validateError: '验证失败',
        validateOk: '代码验证通过',
        selectAccount: '请选择账号',
        previewOk: '预览完成',
        previewSuccess: '预览成功',
        previewFailed: '预览失败',
        execFailed: '执行失败',
        savedAsTemplate: '已保存为模板',
        copied: '代码已复制',
        copyFailed: '复制失败，请手动复制'
      },
      aiPrompt: {
        intro: '请根据以下信息修改策略代码，使其通过验证并且预览信号执行成功。',
        problem: '【问题】{{title}}',
        currentCodeTitle: '【当前代码】',
        pythonFenceStart: '```python',
        fenceEnd: '```',
        outputTitle: '【输出信息】',
        outro: 'Return only the fixed code wrapped in ```python```.'
      }
    },
    templateModal: {
      title: '保存为模板',
      fields: {
        name: '名称',
        description: '描述'
      },
      placeholders: {
        name: 'Enter template name',
        description: '可选：策略说明'
      }
    },
    backtestRun: {
      title: 'Backtest run',
      status: {
        queued: 'Queued',
        running: '运行中',
        completed: '已完成',
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
        status: '状态',
        error: '错误',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe'
      },
      metrics: {
        totalReturn: '总收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '胜率',
        totalTrades: '交易次数',
        equityCurvePoints: 'Equity curve points'
      },
      trades: {
        title: 'Order details',
        empty: 'No trades recorded',
        loadFailed: 'Failed to load order details',
        ticket: '订单号',
        side: '方向',
        sideBuy: 'Buy',
        sideSell: 'Sell',
        volume: 'Volume',
        openTime: 'Open time',
        openPrice: '开仓价',
        closeTime: 'Close time',
        closePrice: '平仓价',
        pnl: 'P&L',
        commission: 'Commission',
        reason: 'Close reason',
        reasons: {
          signal: '信号(用于下单)',
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
      title: '记录',
      titleWithName: '记录 - {{name}}',
      messages: {
        missingScheduleId: 'Missing schedule ID'
      },
      execStatus: {
        pending: '待检测',
        running: '运行中',
        completed: '已完成',
        failed: '失败',
        skipped: 'Skipped'
      },
      operationStatus: {
        success: '成功',
        failed: '失败',
        running: '运行中'
      },
      execTable: {
        time: '时间',
        action: '操作',
        execute: '执行',
        status: '状态',
        durationMs: '耗时(ms)',
        error: '错误'
      },
      ordersTable: {
        time: '时间',
        side: '方向',
        symbol: '品种',
        lots: '手数(Lot)',
        openPrice: '开仓价',
        closePrice: '平仓价',
        profit: '盈亏',
        ticket: '订单号'
      },
      orderSide: {
        buy: '市价买入',
        sell: '市价卖出',
        close: '平仓',
        buyLimit: '限价买入',
        sellLimit: '限价卖出',
        buyStop: '突破买入',
        sellStop: '突破卖出',
        buyStopLimit: '限价突破买',
        sellStopLimit: 'Sell stop limit'
      },
      scheduleIdLabel: '调度ID:',
      summary: {
        name: '名称',
        status: '状态',
        trade: '交易',
        enableCount: '启用次数',
        lastRun: '最后运行时间',
        lastError: 'Last error'
      },
      tabs: {
        exec: '运行记录',
        orders: '交易记录',
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
      title: '策略调度',
      createSchedule: '创建调度',
      format: {
        interval: '每 {{s}}秒',
        cron: '定时: {{expr}}',
      },
      status: {
        running: '运行中',
        disabled: 'Disabled'
      },
      templateVisibility: {
        public: '公开',
        private: '私有'
      },
      table: {
        name: '名称',
        template: '模板',
        account: '账号',
        tradeParams: '交易参数',
        schedule: '计划',
        status: '状态',
        lastRun: '最后运行时间',
        actions: '操作'
      },
      nextRunAt: '下次运行',
      enableCount: '启用次数',
      actions: {
        create: '新建调度',
        logs: '执行日志',
        healthCheck: '健康检查',
        runNow: 'Run now'
      },
      health: {
        title: '策略健康检查 {{name}}',
        summaryBanner: '健康分级：{{grade}}；最近样本 {{totalRuns}} 次，成功率 {{successRate}}%',
        grade: {
          pending: '待检测',
          noSample: '无样本',
          healthy: '健康',
          watch: '关注',
          alert: 'Alert'
        },
        notes: {
          pending: '请先执行健康检查。',
          noSample: '样本不足，至少需要 {{minSampleSize}} 条运行记录。',
          healthy: '成功率高且失败次数可控。',
          watch: '成功率达到关注阈值（>= {{yellowSuccessRate}}%），建议持续观察。',
          alert: 'Low success rate. Investigate strategy/account conditions now.'
        },
        fields: {
          grade: '健康级别',
          rule: '判定依据',
          thresholds: '当前阈值',
          configKey: '配置键',
          lastRunAt: '最后运行时间',
          latestTicket: '最近成交 Ticket',
          successOverTotal: '执行成功/总次数',
          failedRuns: '执行失败次数',
          latestProfit: '最近成交盈亏',
          latestError: 'Latest error'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}；绿色：成功率>={{greenSuccessRate}}% 且失败次数<={{greenMaxFailedRuns}}；黄色：成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近执行日志',
          orders: 'Recent order records'
        },
        runLogs: {
          signalType: '信号(用于下单)'
        },
        messages: {
          loadFailed: '加载健康检查数据失败',
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
        parametersParseFailed: '参数解析失败',
        defaultTemplateNotFound: '默认模板不存在，请刷新页面重试',
        importDefaultTemplateFailedNoId: '导入默认模板失败：未返回模板ID',
        templateCodeEmptyCannotExecute: '模板 code 为空，无法执行',
        strategyExecuteFailed: '策略执行失败',
        executeFailed: '执行失败',
        noOrderableSignal: '没有可下单的信号',
        signalHoldCannotOrder: '当前信号为 hold/无交易动作，不能下单',
        volumeInvalid: '下单手数无效（volume 必须 > 0）',
        orderSubmitted: '已提交下单',
        orderFailed: '下单失败'
      },
      editModal: {
        title: {
          edit: '编辑调度任务',
          create: '新建调度任务'
        },
        fields: {
          template: '模板',
          templateExtra: '来自「策略管理」中保存的模板',
          account: '账号',
          name: '名称',
          symbol: '品种',
          lot: '手数(Lot)',
          lotExtra: '下单手数，建议从 0.01 开始',
          runFrequency: '运行频率',
          cronExpression: 'Cron 表达式',
          cronExtra: '标准 5 段：分钟 小时 日 月 周。例如：*/5 * * * * 每5分钟；0 9 * * 1-5 工作日9点',
          intervalSeconds: '间隔(秒)',
          intervalSecondsExtra: '自动跟随周期(timeframe)，无需修改',
          enableExtra: 'Enable schedule after creating'
        },
        placeholders: {
          name: '例如：EURUSD M5 早盘策略',
          selectAccountFirst: '先选账号',
          symbol: '选择品种'
        },
        validation: {
          templateRequired: '请选择模板',
          accountRequired: '请选择账号',
          nameRequired: '请输入名称',
          symbolRequired: '请选择品种',
          lotRequired: '请输入手数',
          runFrequencyRequired: '请选择运行频率',
          cronRequired: '请输入 cron',
          timeframeRequired: '请选择周期',
          triggerModeRequired: 'Trigger mode is required'
        },
        runFrequencyExtra: {
          cron: '高级：使用 Cron 精确控制执行时间',
          byTimeframe: 'Run by timeframe'
        },
        runFrequencyOptions: {
          byTimeframe: '按周期触发（推荐）',
          cron: 'Cron'
        },
        autoName: {
          strategy: 'Strategy'
        },
        advanced: {
          title: '高级设置',
          fixedIntervalSeconds: '固定间隔(秒)',
          fixedIntervalSecondsExtra: '可选。填写后将按固定间隔执行（不再自动跟随周期）。例如：60 表示每 60 秒执行一次',
          timeframe: '周期',
          timeframeExtra: '默认即可。仅用于K线与指标计算',
          triggerMode: '触发模式',
          triggerModeExtra: '稳定：按K线/周期触发（更稳但有延迟）；高频：报价流触发（更快但噪声大，需要去抖）',
          triggerModeOptions: {
            stable: '稳定（K线/周期）',
            hf: 'High-frequency signal stream'
          },
          stableOverrideIntervalSeconds: '稳定模式高级：间隔(秒)',
          stableOverrideIntervalSecondsExtra: '可选。默认绑定周期(timeframe)。填写后将覆盖稳定模式的触发间隔',
          hfCooldownMs: '高频模式：最小触发间隔(ms)',
          hfCooldownMsExtra: '用于去抖：两次评估/下单之间的最小间隔',
          parametersJson: '参数(JSON对象)',
          parametersJsonExtra: 'JSON parameters for the strategy'
        }
      },
      triggerModal: {
        title: '立即执行(直接下单)',
        confirmOrder: {
          title: '确认下单',
          ok: 'Confirm'
        },
        actions: {
          confirmOrder: '确认下单',
          rerun: 'Re-run'
        },
        summary: {
          scheduleName: '调度名称',
          account: '账号',
          symbol: '品种',
          timeframe: '周期'
        },
        messages: {
          signalNotOrderable: 'Signal is not orderable'
        },
        cards: {
          logs: '执行日志',
          signal: '信号(用于下单)'
        },
        emptyLogs: '(无日志)',
        emptySignal: 'No signal'
      }
    },
    asset: {
      title: 'Strategy Assets',
      subtitle: 'Asset publishing, review status, and cloning are maintained by the system. Cloned results are independent user templates.',
      submitAsset: 'Submit Asset',
      assetList: 'Asset List',
      name: '名称',
      visibility: '可见性',
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
      create: '新建调度',
      noAccounts: 'No paper accounts. Create one to start simulated trading.',
      running: 'Running {{symbol}} {{timeframe}}',
      start: '启动',
      stop: '停止',
      watch: '关注',
      paper: 'Paper',
      startStrategy: 'Start Paper Strategy',
      symbol: '品种',
      timeframe: '周期',
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
      reviseSend: '发给AI修改',
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
        symbol: '品种',
        symbolRequired: '请选择品种',
        symbolPlaceholder: 'EURUSD',
        timeframe: '周期',
        klineCount: 'K-line Count',
        submit: 'Start Detection'
      },
      result: {
        title: 'Detection Result',
        status: '状态',
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
          status: '状态',
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
          score: '评分',
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
      account: '账号',
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
      completedStatus: '已完成',
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
      selectSymbol: '品种',
      code: 'Strategy Code',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
      validate: '验证代码',
      validatePass: '代码验证通过',
      validateFailed: '代码验证失败',
      validateBeforeSave: 'Please validate code before saving',
      runBacktest: 'Run Backtest',
      save: 'Save',
      copy: '复制',
      copySuccess: '代码已复制',
      copyFailed: '复制失败，请手动复制',
      saveSuccess: 'Saved',
      chart: 'K-line',
      backtest: '回测',
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
        orderFailed: '下单失败',
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
      title: '回测',
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
      preview: '预览信号',
      previewTitle: 'Preview ({{shown}} of {{total}})',
      truncated: 'TRUNCATED',
      results: 'Results ({{count}})',
      rank: '#',
      grade: 'Grade',
      score: '评分',
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
      tradeTime: '时间',
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
      orderFailed: '下单失败'
    },
    library: {
      title: 'Strategy Library',
      myStrategies: 'My Strategies',
      create: '新建调度',
      filterAll: 'All',
      filterMine: 'My',
      filterSystem: '预设',
      searchPlaceholder: 'Search strategies...',
      empty: 'No strategies found',
      system: 'System',
      shared: 'Shared',
      private: '私有',
      share: 'Share',
      published: '已发布',
      draft: '草稿',
      unpublish: 'Unpublish',
      unpublishShort: 'Off',
      publish: 'Publish to Market',
      publishSuccess: '已发布',
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
    title: '指标目录',
    description: '策略沙箱中可用的技术指标和风险参数。请在策略代码中仅使用这些辅助函数和参数键。',
    indicatorsTitle: '技术指标',
    riskSectionTitle: '风险管理参数',
    riskParamsTitle: '通用风险参数',
    riskParamsDesc: '无论选择哪些指标，每个策略都应遵循这些风险管理参数。',
    paramKey: '键',
    paramLabel: '标签',
    paramType: '类型',
    paramDefault: '默认值',
    paramRange: '范围',
    paramDescription: '描述'
  }
} as const;

export default strategy;
