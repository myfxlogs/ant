const strategy = {
  strategy: {
    validation: {
      passed: '代码验证通过',
      notPassed: '代码验证未通过',
      riskEval: {
        title: '风险评估',
        riskHigh: '风险等级：high',
        riskUnreliable: '风险评估：不可靠（isReliable=false）',
        riskLoading: '风险评估仍在计算中'
      }
    },
    codeEditor: {
      title: '策略编辑器',
      labels: {
        account: '账户',
        symbol: '品种',
        timeframe: '时间周期',
        code: '策略代码',
        disabledSuffix: '（已禁用）'
      },
      actions: {
        copy: '复制',
        validate: '验证代码',
        preview: '预览信号',
        saveAsTemplate: '保存为模板',
        sendToAI: '发给AI修改',
        sendToAIFixTitleValidate: '验证未通过/有警告',
        sendToAIFixTitlePreview: '预览信号执行失败/需要优化'
      },
      placeholders: {
        selectAccount: '选择账号',
        selectAccountFirst: '请先选择账号',
        loadingSymbols: '可用品种加载中…',
        selectSymbol: '选择品种',
        noSymbols: '未获取到品种列表',
        code: '输入Python策略代码...'
      },
      cards: {
        validationResult: '验证结果',
        previewResult: '预览结果'
      },
      hints: {
        previewInfo: 'Preview 取最近 N 根K线（默认 500，配置：strategy.preview_bars）；回测取最近 N 个月（默认 3，配置：strategy.backtest_window_months）。'
      },
      messages: {
        enterCode: '请输入策略代码',
        selectAccount: '请选择账号',
        validateOk: '代码验证通过',
        validateFailed: '代码验证失败',
        validateError: '验证失败',
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
        outputTitle: '【输出信息】',
        outro: '请直接给出修改后的完整代码（用 ```python 包裹），并说明修改点。',
        pythonFenceStart: '```python',
        fenceEnd: '```'
      }
    },
    schedules: {
      title: '策略调度',
      createSchedule: '创建调度',
      format: {
        interval: '每 {{s}}秒',
        cron: '定时: {{expr}}',
      },
      actions: {
        create: '新建调度',
        logs: '日志',
        healthCheck: '健康检查',
        runNow: '立即执行'
      },
      health: {
        title: '策略健康检查 {{name}}',
        summaryBanner: '健康分级：{{grade}}；最近样本 {{totalRuns}} 次，成功率 {{successRate}}%',
        grade: {
          pending: '待检测',
          noSample: '无样本',
          healthy: '健康',
          watch: '关注',
          alert: '告警'
        },
        notes: {
          pending: '请先执行健康检查。',
          noSample: '样本不足，至少需要 {{minSampleSize}} 条运行记录。',
          healthy: '成功率高且失败次数可控。',
          watch: '成功率达到关注阈值（>= {{yellowSuccessRate}}%），建议持续观察。',
          alert: '成功率偏低，建议立即排查策略与账户状态。'
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
          latestError: '最近错误信息'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}；绿色：成功率>={{greenSuccessRate}}% 且失败次数<={{greenMaxFailedRuns}}；黄色：成功率>={{yellowSuccessRate}}%',
        sections: {
          runLogs: '最近执行日志',
          orders: '最近成交记录'
        },
        runLogs: {
          signalType: '信号'
        },
        messages: {
          loadFailed: '加载健康检查数据失败',
          clickRefresh: '点击刷新加载健康数据'
        }
      },
      editModal: {
        title: {
          create: '新建调度任务',
          edit: '编辑调度任务'
        },
        autoName: {
          strategy: '策略'
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
          enableExtra: 'EA 体验：启用后会持续运行，直到你手动停用'
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
          triggerModeRequired: '请选择触发模式'
        },
        runFrequencyExtra: {
          cron: '高级：使用 Cron 精确控制执行时间',
          byTimeframe: '默认：跟随周期(timeframe)触发（最像EA的OnTick/OnTimer体验）'
        },
        runFrequencyOptions: {
          byTimeframe: '按周期触发（推荐）',
          cron: 'Cron（高级）'
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
            hf: '高频（报价/tick）'
          },
          stableOverrideIntervalSeconds: '稳定模式高级：间隔(秒)',
          stableOverrideIntervalSecondsExtra: '可选。默认绑定周期(timeframe)。填写后将覆盖稳定模式的触发间隔',
          hfCooldownMs: '高频模式：最小触发间隔(ms)',
          hfCooldownMsExtra: '用于去抖：两次评估/下单之间的最小间隔',
          parametersJson: '参数(JSON对象)',
          parametersJsonExtra: '用于给策略代码传参，形如 key->value，运行时自动注入策略上下文。示例：{ "fast": 10, "slow": 20, "risk": "low" }'
        }
      },
      triggerModal: {
        title: '立即执行(直接下单)',
        actions: {
          rerun: '重新执行',
          confirmOrder: '确认下单'
        },
        confirmOrder: {
          title: '确认要下单吗？',
          ok: '确认下单'
        },
        summary: {
          scheduleName: '调度名称',
          account: '账号',
          symbol: '品种',
          timeframe: '周期'
        },
        messages: {
          signalNotOrderable: '当前信号不可下单：需要 buy/sell 且 volume > 0'
        },
        cards: {
          logs: '执行日志',
          signal: '信号(用于下单)'
        },
        emptyLogs: '(无日志)',
        emptySignal: '(无信号)'
      },
      table: {
        name: '名称',
        template: '模板',
        account: '账号',
        tradeParams: '交易参数',
        schedule: '计划',
        status: '状态',
        lastRun: '最近运行',
        actions: '操作'
      },
      templateVisibility: {
        public: '公开',
        private: '私有'
      },
      status: {
        running: '运行中',
        disabled: '已停用'
      },
      nextRunAt: '下次运行',
      enableCount: '启用次数',
      deleteConfirm: {
        title: '确认删除该调度任务？'
      },
      validation: {
        parametersMustBeJsonObject: '参数必须是 JSON 对象'
      },
      messages: {
        parametersParseFailed: '参数解析失败',
        defaultTemplateNotFound: '默认模板不存在，请刷新页面重试',
        importDefaultTemplateFailedNoId: '导入默认模板失败：未返回模板ID',
        templateCodeEmptyCannotExecute: '模板 code 为空，无法执行',
        executeFailed: '执行失败',
        strategyExecuteFailed: '策略执行失败',
        noOrderableSignal: '没有可下单的信号',
        signalHoldCannotOrder: '当前信号为 hold/无交易动作，不能下单',
        volumeInvalid: '下单手数无效（volume 必须 > 0）',
        orderSubmitted: '已提交下单',
        orderFailed: '下单失败'
      }
    },
    scheduleLogs: {
      title: '记录',
      titleWithName: '记录 - {{name}}',
      tabs: {
        exec: '运行记录',
        orders: '交易记录',
        execLogs: '执行日志',
        orderLogs: '订单日志'
      },
      messages: {
        missingScheduleId: '缺少 scheduleId'
      },
      summary: {
        name: '名称',
        status: '状态',
        trade: '交易',
        enableCount: '启用次数',
        lastRun: '最近运行',
        lastError: '最近错误'
      },
      execStatus: {
        pending: '待执行',
        running: '运行中',
        completed: '已完成',
        failed: '失败',
        skipped: '已跳过'
      },
      operationStatus: {
        success: '成功',
        failed: '失败',
        running: '执行中'
      },
      execTable: {
        time: '时间',
        action: '操作',
        status: '状态',
        durationMs: '耗时(ms)',
        error: '错误',
        execute: '执行'
      },
      ordersTable: {
        time: '时间',
        side: '方向',
        symbol: '品种',
        lots: '手数',
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
        sellStopLimit: '限价突破卖'
      },
      scheduleIdLabel: '调度ID:',
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
      title: '策略模板',
      tabs: {
        system: '系统模板',
        user: '自建模板'
      },
      copySuffix: ' (副本)',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
      badges: {
        preset: '预设'
      },
      visibility: {
        public: '公开',
        private: '私有'
      },
      codeModal: {
        title: '策略代码',
        actions: {
          copy: '复制'
        }
      },
      status: {
        draft: '草稿',
        published: '已发布'
      },
      actions: {
        create: '新建模板',
        createTemplate: '新建模板',
        edit: '编辑',
        delete: '删除',
        copy: '复制',
        viewCode: '查看代码',
        backtest: '回测',
        launchSchedule: '上线到调度'
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
        defaultHint: '默认',
        emptyUser: '暂无自建模板，点击右上角「新建模板」开始'
      },
      scheduleLaunch: {
        title: '调度上线',
        noRun: '暂无回测运行',
        backtestRunningHint: '回测正在运行，请稍候。',
        score: '评分',
        keyMetrics: '关键指标',
        launchSection: '上线调度',
        actions: {
          publishTemplate: '发布模板',
          createScheduleNoEnable: '创建调度',
          createAndEnable: '创建并启用',
          create: '创建计划',
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
      editTemplateModal: {
        title: {
          create: '新建模板',
          edit: '编辑模板'
        },
        fields: {
          name: '名称',
          description: '描述',
          code: '策略代码',
          publicShare: '公开分享'
        },
        placeholders: {
          name: '例如：均线交叉策略',
          description: '可选：策略说明',
          codeSample: `# 策略代码示例
# 可用的变量: close, open, high, low, volume, symbol
# 返回: signal字典

import numpy as np

# 计算指标
maFast = np.mean(close[-10:])
maSlow = np.mean(close[-20:])

# 生成信号
if maFast > maSlow:
    signal = 'buy'
elif maFast < maSlow:
    signal = 'sell'
else:
    signal = 'hold'

# 返回结果
signal = {
    'signal': signal,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.7,
    'reason': f'MA快线={maFast:.5f}, MA慢线={maSlow:.5f}'
}`
        },
        validation: {
          nameRequired: '请输入名称',
          codeRequired: '请输入策略代码'
        },
        actions: {
          validateCode: '验证代码'
        }
      },
      backtest: {
        title: '回测',
        parameters: {
          title: '策略参数'
        },
        fields: {
          title: '标题',
          account: '账户',
          symbol: '品种',
          timeframe: '周期',
          initialCapital: '初始资金',
          range: '回测区间',
          extraSymbols: '辅助标的（可多选）'
        },
        placeholders: {
          account: '选择账户',
          symbol: '选择品种',
          range: '请选择回测区间',
          extraSymbols: '可选，配对/轮动策略常用'
        },
        tooltips: {
          extraSymbols: '除主标的外，额外拉取的 K 线（同账户、同周期）。策略通过 context["closes_by_symbol"] 访问。'
        },
        validation: {
          accountRequired: '请选择账户',
          symbolRequired: '请选择品种',
          timeframeRequired: '请选择周期',
          initialCapitalRequired: '请输入初始资金',
          rangeRequired: '请选择回测区间'
        },
        quickRange: {
          '1d': '1天',
          '3d': '3天',
          '1w': '1周',
          '1y': '1年',
          custom: '自定义'
        },
        accountDisabledSuffix: ' (已禁用)',
        modalTitleWithName: '回测：{{name}}'
      },
      messages: {
        fetchTemplateListFailed: '获取模板列表失败',
        deepLinkNavigate: '已为你定位模板与回测详情',
        enterStrategyCode: '请输入策略代码',
        codeValidationPassed: '代码验证通过',
        codeValidationNotPassed: '代码未通过验证',
        codeValidationFailed: '代码验证失败',
        templateUpdated: '模板已更新',
        templateCreated: '模板已创建',
        templateDeleted: '模板已删除',
        readStrategyCodeFailed: '读取策略代码失败',
        strategyCodeEmptyCannotBacktest: '策略代码为空，无法回测',
        selectBacktestRange: '请选择回测区间',
        backtestRangeInvalid: '回测区间无效',
        backtestSubmitted: '已提交回测',
        backtestSubmitFailed: '提交回测失败',
        backtestCancelRequested: '已请求取消回测',
        backtestCancelFailed: '取消回测失败',
        backtestReportDeleted: '回测报告已删除',
        backtestReportNotFound: '回测报告不存在',
        codeCopied: '代码已复制',
        copyFailed: '复制失败',
        missingScheduleInfo: '缺少调度必要信息',
        templateNotPublishedCannotCreateSchedule: '模板未发布，无法创建调度',
        readTemplateStatusFailed: '读取模板状态失败',
        scheduleCreated: '调度已创建',
        scheduleCreatedAndEnabled: '调度已创建并启用',
        createScheduleFailed: '创建调度失败',
        templatePublished: '模板已发布',
        cannotPublishAndCreateDraftFailed: '无法发布，草稿创建失败。',
        republishedButNoTemplateId: '已重新发布，但缺少模板 ID。',
        backtestRunningCannotPublish: '回测正在运行，暂无法发布。',
        missingDraftIdCannotPublish: '缺少草稿 ID，无法发布。',
        publishedButNoTemplateId: '已发布，但缺少模板 ID。',
        templateRepublished: '模板已重新发布',
        templateAlreadyPublished: '模板已发布',
        templateNotDraftUnknownPublishStatus: '模板非草稿状态，发布状态未知。',
        publishFailed: '发布失败',
        backtestRunNoPublishedTemplate: '回测运行缺少已发布模板',
        strategyCodeEmptyCannotPublish: '策略代码为空，请先保存代码再发布。',
        systemTemplateReadOnly: '系统模板为只读，请克隆后再编辑。'
      },
      backtestRuns: {
        title: '回测报告',
        empty: '暂无回测记录',
        deleteConfirm: '确定要删除这条回测报告吗？',
        batchDelete: '删除 {{count}} 条',
        batchDeleteConfirm: '确定要删除 {{count}} 条回测报告吗？',
        batchDeleteSuccess: '已删除 {{count}} 条回测报告',
        status: {
          queued: '排队中',
          running: '运行中',
          completed: '已完成',
          failed: '失败',
          canceling: '取消中',
          canceled: '已取消'
        },
        table: {
          title: '标题',
          status: '状态',
          symbol: '品种',
          timeframe: '周期',
          createdAt: '创建时间',
          actions: '操作'
        },
        actions: {
          view: '查看',
          launchSchedule: '调度上线',
          createSchedule: '创建调度'
        }
      },
      deleteConfirm: '确定删除此模板？',
      defaultDraftName: '草稿模板'
    },
    backtestRun: {
      title: '回测运行',
      status: {
        queued: '排队中',
        running: '运行中',
        completed: '已完成',
        failed: '失败',
        canceling: '取消中',
        canceled: '已取消',
        ended: '已结束'
      },
      actions: {
        cancel: '取消'
      },
      hints: {
        queued: '回测排队中',
        running: '回测运行中',
        canceling: '正在取消回测'
      },
      fields: {
        status: '状态',
        error: '错误',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率'
      },
      metrics: {
        totalReturn: '总收益',
        annualReturn: '年化收益',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率',
        winRate: '胜率',
        totalTrades: '交易次数',
        equityCurvePoints: '资金曲线点数'
      },
      trades: {
        title: '订单明细',
        empty: '暂无成交记录',
        loadFailed: '加载订单明细失败',
        ticket: '编号',
        side: '方向',
        sideBuy: '买入',
        sideSell: '卖出',
        volume: '手数',
        openTime: '开仓时间',
        openPrice: '开仓价',
        closeTime: '平仓时间',
        closePrice: '平仓价',
        pnl: '盈亏',
        commission: '手续费',
        reason: '平仓原因',
        reasons: {
          signal: '信号',
          sl: '止损',
          tp: '止盈',
          margin_call: '强平',
          expired: '到期',
          end_of_test: '回测结束'
        },
        summary: '共 {{count}} 笔，胜 {{wins}} / 负 {{losses}}，净盈亏 {{pnl}}'
      }
    },
    defaultTemplates: {
      maCross: {
        name: '双均线交叉策略',
        description: '当快均线上穿慢均线时买入，下穿时卖出'
      },
      forceBuy: {
        name: '测试下单（强制BUY）',
        description: '用于验证下单链路：每次执行都返回 buy，并从 context/params 读取 lot 作为 volume'
      },
      rsi: {
        name: 'RSI超买超卖策略',
        description: 'RSI低于30超买区买入，高于70超卖区卖出'
      },
      macd: {
        name: 'MACD策略',
        description: 'MACD金叉买入，死叉卖出'
      }
    },
    asset: {
      title: '策略资产库',
      subtitle: '资产发布、审核状态和克隆由系统维护，克隆结果是独立用户模板。',
      submitAsset: '提交资产',
      assetList: '资产列表',
      name: '名称',
      visibility: '可见性',
      reviewStatus: '审核状态',
      cloneCount: '克隆数',
      version: '版本',
      description: '说明',
      actions: '操作',
      cloneAsDraft: '克隆为草稿',
      sourceTemplate: '来源模板',
      assetName: '资产名称',
      submit: '提交',
      messages: {
        loadFailed: '加载策略资产失败',
        submitSuccess: '已提交策略资产',
        submitFailed: '提交策略资产失败',
        cloneSuccess: '已克隆为模板：{{templateId}}',
        cloneFailed: '克隆策略资产失败'
      },
      validation: {
        selectTemplate: '请选择来源模板',
        enterName: '请输入资产名称'
      },
      empty: '暂无策略资产'
    },
    gen: {
      title: '策略生成',
      send: '生成策略',
      regenerate: '重新生成',
      reset: '重新开始',
      template: '模板',
      generating: '生成中',
      validating: '合规检查',
      backtestStarted: '回测已启动',
      done: '完成',
      backtestMsg: '回测任务已创建',
      clarifyTitle: '需要确认几个细节：',
      useDefaults: '使用默认设置继续',
      placeholder: '描述你想创建的交易策略，例如："做一个 EURUSD 的布林带均值回归策略，1小时周期"',
      chat: {
        generate: '⚡ 生成',
        revise: '✏️ 修改',
        repair: '🔧 修复',
        discuss: '💬 分析'
      },
      feedback: {
        heading: '📊 回测结果',
        placeholder: '输入反馈继续迭代（如"太激进了"、"加入止损"）'
      },
      metrics: {
        sharpe: '夏普',
        maxDrawdown: '最大回撤',
        winRate: '胜率',
        trades: '交易',
        return: '收益'
      }
    },
    codeAssist: {
      tabAI: 'AI 修改',
      tabExplain: '代码解释',
      explain: '解释代码',
      requiredParamsTitle: '必填参数',
      requiredParamsDesc: '策略代码读取了下列参数但没有给出默认值，请在保存前填写。',
      optionalParamsTitle: '可选参数',
      optionalParamsDesc: '这些参数在代码里已有默认值。留空表示使用默认值；填入新值仅作用于本次运行，不会修改已保存的策略。',
      defaultLabel: '默认',
      paramDescriptions: {
        riskLevel: '风险等级，常用 low / medium / high，影响仓位与止损止盈幅度。',
        takeProfit: '止盈幅度（%），价格相对开仓价上涨/下跌该百分比时平仓获利。',
        stopLoss: '止损幅度（%），价格反向波动该百分比时强制平仓止损。',
        maxLoss: '单笔最大可承受亏损（占账户的比例，0.01 = 1%）。',
        confidence: '信号置信度阈值（0~1），低于此值的信号会被忽略。',
        threshold: '触发信号的阈值，具体含义看代码里的判断条件。',
        lotSize: '下单手数 / 交易量，越大风险越高。',
        fastPeriod: '快线周期（K 线根数），常用于 MACD/双均线，越小越敏感。',
        slowPeriod: '慢线周期（K 线根数），常用于 MACD/双均线，越大越平滑。',
        signalPeriod: '信号线周期（K 线根数），MACD DIF 与 DEA 的平滑周期。',
        rsiPeriod: 'RSI 计算周期（K 线根数），常用 14。',
        emaPeriod: 'EMA 指数移动平均的周期（K 线根数）。',
        smaPeriod: 'SMA 简单移动平均的周期（K 线根数）。',
        genericPeriod: '回看周期（K 线根数），用于指标计算的窗口长度。',
        genericPercent: '百分比/比率类参数，单位通常为 %（如 1 表示 1%）。'
      },
      required: '必填',
      suggested: '建议',
      applyAllSuggestions: '一键填入建议值',
      fillRequiredParams: '请先填写必填参数：{{keys}}',
      aiReviseTitle: 'AI 助手 — 修改代码',
      reviseInputPlaceholder: '例如：把 SMA(20) 换成 EMA(50)，并加 1% 止损。',
      reviseSend: '发送给 AI',
      enterInstruction: '请描述你想做的修改。',
      codeEmpty: '当前没有可修改的代码。',
      codeUpdated: '代码已更新，请重新进行代码验证后再保存。',
      noPython: 'AI 没有返回 Python 代码块，请换种说法再试。',
      saveBlockedNotValidated: '请先点击"验证代码"，验证通过后才能保存。',
      generatePlaceholder: '描述你的策略需求...'
    },
    marketRegime: {
      title: '市场状态识别',
      subtitle: '基于历史 K 线数据分析趋势方向、波动率状态和价格效率，识别当前市场环境。',
      ruleVersionAlert: '当前为规则版检测模型 rule-v1，由实时 K 线市场数据驱动。',
      detectSuccess: '市场状态检测完成',
      detectFailed: '市场状态检测失败',
      form: {
        title: '检测参数',
        accountId: '账户 ID',
        accountIdRequired: '请输入账户 ID',
        accountIdPlaceholder: 'MT 账户 UUID',
        symbol: '交易品种',
        symbolRequired: '请输入交易品种',
        symbolPlaceholder: 'EURUSD',
        timeframe: '周期',
        klineCount: 'K 线数量',
        submit: '开始检测'
      },
      result: {
        title: '检测结果',
        status: '状态',
        confidence: '置信度',
        modelVersion: '模型版本',
        strategyFamilies: '策略族',
        features: '特征',
        recordId: '记录 ID'
      }
    },
    experiment: {
      title: '策略实验',
      subtitle: '提交参数组合后，系统自动运行实验、评分候选策略并生成草稿。',
      ruleVersionAlert: '当前为确定性参数实验最小闭环，候选仅生成草稿，不会自动发布、调度或交易。',
      jobEventStream: 'Job 事件流',
      noEvents: '暂无事件',
      selectJobToView: '选择带 Job 的实验后显示事件。',
      submitForm: {
        title: '提交实验',
        baseTemplate: '基础策略模板',
        baseTemplateRequired: '请选择基础策略模板',
        baseTemplatePlaceholder: '选择模板',
        parameterSpace: '参数空间 JSON',
        parameterSpaceRequired: '请输入参数空间 JSON',
        searchMethod: '搜索方式',
        maxCandidates: '候选上限',
        objective: '目标',
        submit: '提交实验'
      },
      list: {
        title: '实验列表',
        column: {
          status: '状态',
          searchMethod: '搜索方式',
          maxCandidates: '候选上限',
          objective: '目标',
          actions: '操作',
          viewCandidates: '查看候选'
        }
      },
      candidates: {
        title: '候选列表',
        titleWithId: '候选列表：{{id}}',
        column: {
          rank: '排名',
          grade: '等级',
          score: '评分',
          parameters: '参数',
          summary: '摘要',
          recommendation: '建议',
          actions: '操作',
          viewCandidates: '查看候选',
          generateDraft: '生成草稿'
        }
      },
      messages: {
        loadTemplatesFailed: '加载策略模板失败',
        loadExperimentsFailed: '加载实验列表失败',
        loadCandidatesFailed: '加载候选失败',
        subscribeJobFailed: '订阅实验 Job 事件失败',
        candidatesGenerated: '策略实验已生成候选',
        submitFailed: '提交策略实验失败，请确认参数空间是合法 JSON',
        draftGenerated: '已生成草稿模板：{{templateId}}',
        promoteFailed: '提升候选为草稿失败'
      }
    },
    tuning: {
      optimizerMethod: '优化方法',
      parameterDimensions: '参数维度',
      enabledCombinations: '{{enabled}} 个已启用 · {{combos}} 个组合',
      hide: '隐藏',
      preview: '预览',
      previewTitle: '预览 ({{shown}} / {{total}})',
      truncated: '已截断',
      results: '结果 ({{count}})',
      rank: '#',
      grade: '评级',
      score: '评分',
      parameters: '参数',
      summary: '摘要',
      oosScore: 'OOS 评分',
      degradation: '衰减',
      overfit: '过拟合',
      overfitWarning: '⚠ 过拟合',
      apply: '应用',
      run: '运行 ({{count}})',
      tuning: '调参中…',
      requiresAI: '需要配置 AI 提供商',
      switchToDE: '切换到 DE',
      waiting: '等待实验完成... (SSE 自动刷新)',
      gridWarning: '网格搜索将测试 <b>{{count}}</b> 个组合 (预算: 48)。建议切换到<b>差分进化</b>，高效处理大参数空间。',
      oosFootnote: '对前 5 候选进行样本外验证 (按 IS 评分)。绿色衰减 <20%，橙色 20-40%，红色 >40%。',
      optimizer: {
        grid: '网格搜索',
        random: '随机搜索',
        de: '差分进化',
        tpe: 'TPE (核密度估计)',
        ags: '退火高斯',
        ai: 'AI 优化器',
        gridDesc: '穷举笛卡尔积。最适合 ≤3 个参数。',
        randomDesc: '均匀随机采样。适合探索阶段。',
        deDesc: 'rand/1/bin 变异。在平滑曲面上快速收敛。',
        tpeDesc: '树状 Parzen 估计器。用 KDE 建模好/坏分布。',
        agsDesc: '高斯抖动 + sigma 退火。TPE 的轻量替代方案。',
        aiDesc: 'LLM 多轮提议。基于前序结果学习，共 3 轮。'
      },
      started: '智能调参已启动'
    },
    templateModal: {
      title: '保存为模板',
      fields: {
        name: '名称',
        description: '描述'
      },
      placeholders: {
        name: '输入模板名称',
        description: '输入描述'
      }
    },
    workspace: {
      title: '策略工作台',
      account: '账户',
      accountPlaceholder: '账户 ID',
      chartWindow: '图表',
      hideCode: '隐藏代码',
      showCode: '显示代码',
      quickTrade: '快捷交易',
      quickTradeHint: '请先选择品种',
      tradePanelPlaceholder: '交易面板 — 即将推出',
      selectSymbolHint: '选择交易账户和品种以查看图表',
      noAccounts: '暂无可用账户',
      selectSymbol: '品种',
      code: '策略代码',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
      validate: '验证',
      validatePass: '验证通过',
      validateFailed: '验证失败',
      validateBeforeSave: '请先验证代码再保存',
      runBacktest: '运行回测',
      save: '保存',
      copy: '复制',
      copySuccess: '已复制',
      copyFailed: '复制失败',
      saveSuccess: '已保存',
      chart: 'K线',
      backtest: '回测',
      backtestRunning: '回测运行中...',
      backtestCompleted: '已完成',
      backtestError: '回测失败',
      backtestEmpty: '运行回测查看结果',
      backtestTab: '回测结果',
      tuningTab: '智能调参',
      execAssumptions: 'ℹ 执行假设',
      execAssumptionsFields: {
        mode: '模式',
        timing: '时机',
        fillRule: '成交规则',
        direction: '方向',
        commission: '手续费',
        slippage: '滑点',
        leverage: '杠杆',
        mtfFallback: 'MTF 回退'
      },
      aiAssist: 'AI 助手',
      ai: 'AI',
      runtimeMode: '运行时',
      saveFailed: '保存失败',
      autoFix: {
        fixing: '修复中...',
        button: '自动修复',
        askAI: '询问 AI',
        dismiss: '关闭',
        passed: '自动修复通过（{{iterations}} 次迭代）{{plural}}',
        failed: '自动修复：{{remaining}} 个问题未解决（{{iterations}} 次迭代后）',
        fixed: '✅ 已修复 ({{count}})',
        remaining: '⚠️ 剩余 ({{count}})',
        newRegression: '❌ 新增回归 ({{count}})',
        lineInfo: '第 {{line}} 行'
      },
      template: {
        title: '模板',
        selectPlaceholder: '选择一个模板...',
        load: '加载',
        saveAs: '另存为',
        loaded: '已加载'
      },
      watchlist: '自选',
      selectAccount: '选择账户',
      openPositions: '持仓 ({{count}})',
      noOpenPositions: '此账户暂无持仓',
      chartError: '图表错误 — 请尝试刷新',
      smartTuning: '智能调参',
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
      backtestRunIdLabel: '选择回测运行...',
      investorReadOnly: '投资者（只读）',
      masterTrading: '主账户（交易）',
      riskControls: '代码中的风控规则',
      jumpToCode: '跳转到代码',
      runningStatus: '运行中...',
      completedStatus: '已完成',
      backtestResultsLabel: '回测结果',
      gateTab: 'Gate'
    },
    codeQuality: {
      category: {
        FUTURE_DATA_LEAK: '未来数据泄露',
        MISSING_PARAM: '缺少参数声明',
        UNREAD_PARAM: '未使用参数',
        NDARRAY_PANDAS_MISUSE: 'ndarray/pandas 误用',
        NO_STOP_AND_TAKE_PROFIT: '缺少止损止盈',
        NO_ENTRY_PCT: '缺少入场比例'
      }
    },
    backtestParams: {
      title: '回测参数',
      currentDraft: '📝 当前草稿',
      dateRange: '日期范围',
      execution: '执行参数',
      capital: '本金',
      leverage: '杠杆',
      commission: '手续费',
      slippage: '滑点',
      trade: '交易设置',
      direction: '方向',
      long: '↑ 做多',
      short: '↓ 做空',
      both: '双向',
      strictMode: '严格模式',
      strictModeOn: '开',
      strictModeOff: '关',
      strictModeOnDesc: '下一根 K 线开仓。标准保守模式。',
      strictModeOffDesc: '同 K 线收盘价 + MTF 1m。更高精度。',
      strictModeOnTooltip: '开：信号在 K 线收盘确认，下一根 K 线开盘执行',
      strictModeOffTooltip: '关：同 K 线收盘执行，1m 子分辨率',
      vectorizedMode: 'Vectorized',
      eventDrivenMode: 'Run(context)',
      runtimeMode: '运行时',
      history: '回测历史',
      run: '▶ 运行',
      settingsSave: '保存为我的默认',
      settingsLoad: '加载我的默认',
      settingsReset: '恢复出厂默认',
      defaultsSaved: '默认已保存',
      defaultsLoaded: '默认已加载',
      defaultsReset: '已恢复出厂默认',
      presets: {
        liveAligned: '实盘对齐',
        exploration: '探索模式'
      },
      enterCodeAndSymbol: '请输入策略代码并选择品种',
      backtestFailed: '回测失败'
    },
    quickTradeSection: {
      selectSymbol: '请选择交易品种',
      validVolume: '交易量需 ≥ 0.01 手',
      priceRequired: '请输入价格',
      orderPlaced: '下单成功',
      orderFailed: '下单失败',
      amountLots: '数量(手)',
      marginMode: '保证金模式',
      cross: '跨式',
      isolated: '逐仓',
      mt4CrossOnly: 'MT4 仅支持跨式保证金'
    },
    chartTools: {
      streamActive: '实时K线流已连接',
      streamUnavailable: '数据流不可用',
      hide: '隐藏',
      show: '显示',
      settings: '设置',
      remove: '移除',
      clearDrawings: '清除所有绘图',
      candle: '蜡烛图',
      ohlc: 'OHLC',
      area: '面积图',
      live: '实时',
      error: '错误',
      static: '静态'
    },
    paper: {
      title: '📊 模拟交易',
      createAccount: '创建模拟账户',
      accountName: '账户名称',
      create: '创建',
      noAccounts: '暂无模拟账户。创建一个开始模拟交易。',
      running: '运行中 {{symbol}} {{timeframe}}',
      start: '启动',
      stop: '停止',
      watch: '监控',
      paper: '模拟',
      startStrategy: '启动模拟策略',
      symbol: '品种',
      timeframe: '周期',
      strategyCode: '策略代码 (Python)',
      messages: {
        enterName: '请输入名称',
        created: '模拟账户已创建',
        createFailed: '创建失败',
        pasteCode: '粘贴您的策略代码',
        strategyStarted: '模拟策略已启动',
        startFailed: '启动失败',
        strategyStopped: '模拟策略已停止',
        stopFailed: '停止失败'
      }
    },
    aiChat: {
      title: 'AI 对话',
      you: '你',
      ai: 'AI',
      revise: '修改',
      feedback: '🔄 反馈',
      streaming: '生成中',
      analyzing: '分析中',
      reset: '重置',
      applyCode: '应用代码',
      dismiss: '关闭',
      reviewCode: 'AI 已生成代码 — 请在应用前查看上方的对话。'
    },
    assetAnalysis: {
      title: 'AI 资产分析',
      subtitle: '多周期趋势展望、支撑阻力位检测、波动率分类及 AI 策略推荐',
      symbolPlaceholder: '输入品种 (例如 EURUSD, XAUUSD, BTCUSD)',
      analyze: '分析',
      fetchingData: '正在获取市场数据...',
      phase: '阶段: {{phase}}',
      mtfOutlook: '多周期展望',
      srLevels: '支撑 / 阻力位',
      volatility: '波动率',
      state: '状态',
      atrPct: 'ATR %',
      aiRecommendation: 'AI 策略推荐',
      aiUnavailable: 'AI 推荐不可用。请在设置中配置 AI 提供商。',
      configureAI: '配置 AI 提供商',
      noLevels: '未检测到显著价位',
      noResults: '未返回分析结果。请尝试其他品种。',
      volLow: '低波动率 — 可考虑突破或均值回归策略，配合紧凑止损。',
      volNormal: '正常波动率 — 适合大多数策略类型。',
      volHigh: '高波动率 — 建议扩大止损；趋势跟踪和突破策略更有利。',
      volExtreme: '极端波动率 — 请大幅降低仓位；需要宽止损。'
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
    library: {
      title: '策略库',
      myStrategies: '我的策略',
      create: '新建',
      filterAll: '全部',
      filterMine: '我的',
      filterSystem: '预置',
      searchPlaceholder: '搜索策略...',
      empty: '暂无策略',
      system: '系统',
      shared: '已分享',
      private: '私有',
      share: '分享',
      published: '已发布',
      draft: '草稿',
      unpublish: '下架',
      unpublishShort: '下架',
      publish: '发布到市场',
      publishSuccess: '已发布',
      unpublishSuccess: '已下架',
      publishStatus: '市场状态',
      selectHint: '选择左侧策略查看详情',
      overview: '概览',
      schedules: '运行',
      backtestHistory: '回测历史',
      scheduleCount: '{{count}} 个运行中',
      scheduleRunningCount: '{{count}} 个运行中',
      noSchedules: '未运行',
      openInWorkspace: '在 Workspace 中打开',
      createSchedule: '创建运行',
      saveAsMine: '保存为我的策略',
      saveAsMineSuccess: '已保存到我的策略',
      myCopy: '我的副本',
      codePreview: '代码预览',
      viewCode: '查看策略代码',
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
