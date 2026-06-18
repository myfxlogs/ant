const strategy = {
  strategy: {
    templates: {
      title: 'Mẫu chiến lược',
      tabs: {
        system: 'Mẫu hệ thống',
        user: 'User templates'
      },
      table: {
        name: 'Tên',
        description: 'Mô tả',
        tags: 'Nhãn',
        visibility: 'Hiển thị',
        status: 'Trạng thái',
        useCount: 'Số lần dùng',
        createdAt: 'Thời gian tạo',
        updatedAt: 'Cập nhật lúc',
        actions: 'Thao tác',
        loadingDefault: 'Đang tải mẫu mặc định...',
        defaultHint: 'Mặc định',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
      },
      badges: {
        preset: 'Mặc định'
      },
      visibility: {
        public: 'Công khai',
        private: 'Riêng tư'
      },
      status: {
        draft: 'Nháp',
        published: 'Đã xuất bản'
      },
      actions: {
        create: 'Tạo mẫu',
        edit: 'Chỉnh sửa',
        delete: 'Xóa',
        backtest: 'Kiểm thử lùi',
        viewCode: 'Xem code',
        copy: 'Sao chép',
        launchSchedule: 'Khởi chạy lịch',
        createTemplate: 'Create template'
      },
      copySuffix: ' (Bản sao)',
      deleteConfirm: 'Delete this template?',
      defaultDraftName: 'Draft template',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
      scheduleLaunch: {
        title: 'Khởi chạy lịch',
        noRun: 'Chưa có lần chạy backtest',
        backtestRunningHint: 'Backtest đang chạy. Vui lòng đợi.',
        score: 'Điểm',
        keyMetrics: 'Chỉ số chính',
        launchSection: 'Khởi chạy lịch',
        actions: {
          publishTemplate: 'Xuất bản template',
          createScheduleNoEnable: 'Tạo lịch chạy',
          createAndEnable: 'Tạo & bật',
          create: 'Tạo lịch',
          addAccount: '添加账户',
          updateTradingPassword: '更新交易密码'
        },
        metrics: {
          totalReturn: 'Tổng lợi nhuận',
          annualReturn: 'Lợi nhuận năm',
          maxDrawdown: 'Sụt giảm tối đa',
          sharpe: 'Sharpe',
          winRate: 'Tỷ lệ thắng',
          totalTrades: 'Số lệnh'
        },
        form: {
          account: 'Tài khoản',
          accountPlaceholder: '选择账户',
          scheduleName: '计划名称',
          scheduleNamePlaceholder: 'VD: EURUSD M5 chiến lược buổi sáng',
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
          symbol: 'Mã',
          symbolPlaceholder: '选择品种',
          symbolPlaceholderEmpty: '未配置品种',
          timeframe: 'Khung thời gian',
          defaultVolume: '默认手数',
          defaultVolumeTip: '每个信号的默认下单量',
          enableAfterCreate: '创建后立即启用',
          riskSection: 'Kiểm Soát Rủi Ro',
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
          edit: 'Chỉnh sửa mẫu',
          create: 'Create template'
        },
        actions: {
          validateCode: 'Validate code'
        },
        fields: {
          name: 'Tên',
          description: 'Mô tả',
          code: 'Code chiến lược',
          publicShare: 'Công khai'
        },
        validation: {
          nameRequired: 'Vui lòng nhập tên',
          codeRequired: 'Code is required'
        },
        placeholders: {
          name: 'VD: Chiến lược cắt MA',
          description: 'Tùy chọn: mô tả',
          codeSample: 'Dán mã chiến lược Python...'
        }
      },
      codeModal: {
        title: 'Mã chiến lược',
        actions: {
          copy: 'Sao chép'
        }
      },
      backtest: {
        title: 'Kiểm thử lùi',
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: '策略参数'
        },
        fields: {
          title: 'Title',
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian',
          initialCapital: 'Initial capital',
          range: 'Phạm vi',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        validation: {
          accountRequired: 'Vui lòng chọn tài khoản',
          symbolRequired: 'Vui lòng chọn mã',
          timeframeRequired: 'Vui lòng chọn timeframe',
          initialCapitalRequired: 'Initial capital is required',
          rangeRequired: 'Range is required'
        },
        placeholders: {
          account: 'Chọn tài khoản',
          symbol: 'Chọn mã',
          range: 'Select range',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
        },
        accountDisabledSuffix: ' (đã tắt)',
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
        enterStrategyCode: 'Vui lòng nhập mã chiến lược',
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
        copyFailed: 'Sao chép thất bại, vui lòng sao chép thủ công',
        strategyCodeEmptyCannotPublish: 'Strategy code is empty. Please save your code before publishing.',
        systemTemplateReadOnly: 'System templates are read-only. Clone to edit.'
      },
      backtestRuns: {
        title: 'Backtest runs',
        empty: 'No backtest runs',
        table: {
          title: 'Title',
          status: 'Trạng thái',
          symbol: 'Mã',
          timeframe: 'Khung thời gian',
          createdAt: 'Thời gian tạo',
          actions: 'Thao tác'
        },
        actions: {
          view: 'View',
          launchSchedule: 'View score',
          createSchedule: 'Tạo lịch chạy'
        },
        deleteConfirm: 'Delete this run?',
        batchDelete: 'Delete {{count}}',
        batchDeleteConfirm: 'Delete {{count}} backtest report(s)?',
        batchDeleteSuccess: '{{count}} backtest report(s) deleted',
        status: {
          queued: 'Queued',
          running: 'Đang chạy',
          completed: 'Hoàn tất',
          failed: 'Thất Bại',
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
      passed: 'Xác thực thành công',
      notPassed: 'Xác minh code không đạt',
      riskEval: {
        title: 'Đánh giá rủi ro',
        riskHigh: 'Mức rủi ro: cao',
        riskUnreliable: 'Đánh giá rủi ro: không đáng tin cậy (isReliable=false)',
        riskLoading: 'Risk assessment is still calculating'
      }
    },
    codeEditor: {
      title: 'Trình soạn thảo chiến lược',
      labels: {
        code: 'Mã chiến lược',
        account: 'Tài khoản',
        symbol: 'Mã',
        timeframe: 'Khung thời gian',
        disabledSuffix: ' (đã tắt)'
      },
      actions: {
        copy: 'Sao chép',
        validate: 'Xác thực mã',
        preview: 'Xem trước tín hiệu',
        saveAsTemplate: 'Lưu thành template',
        sendToAI: 'Gửi AI để sửa',
        sendToAIFixTitleValidate: 'Xác thực thất bại / có cảnh báo',
        sendToAIFixTitlePreview: 'Fix preview issues'
      },
      placeholders: {
        code: 'Dán mã chiến lược Python...',
        selectAccount: 'Chọn tài khoản',
        selectAccountFirst: 'Chọn tài khoản trước',
        loadingSymbols: 'Đang tải danh sách mã...',
        selectSymbol: 'Chọn mã',
        noSymbols: 'No symbols available'
      },
      hints: {
        previewInfo: 'Preview will execute with sample market data.'
      },
      cards: {
        validationResult: 'Kết quả xác thực',
        previewResult: 'Preview result'
      },
      messages: {
        enterCode: 'Vui lòng nhập mã chiến lược',
        validateFailed: 'Xác thực thất bại',
        validateError: 'Lỗi xác thực',
        validateOk: 'Xác thực thành công',
        selectAccount: 'Vui lòng chọn tài khoản',
        previewOk: 'Xem trước hoàn tất',
        previewSuccess: 'Xem trước thành công',
        previewFailed: 'Xem trước thất bại',
        execFailed: 'Thực thi thất bại',
        savedAsTemplate: 'Đã lưu thành template',
        copied: 'Đã sao chép',
        copyFailed: 'Sao chép thất bại, vui lòng sao chép thủ công'
      },
      aiPrompt: {
        intro: 'Vui lòng chỉnh sửa mã chiến lược theo thông tin bên dưới để vượt qua xác thực và chạy xem trước thành công.',
        problem: '[Vấn đề] {{title}}',
        currentCodeTitle: '[Mã hiện tại]',
        pythonFenceStart: '```python',
        fenceEnd: '```',
        outputTitle: '[Đầu ra]',
        outro: 'Return only the fixed code wrapped in ```python```.'
      }
    },
    templateModal: {
      title: 'Lưu thành template',
      fields: {
        name: 'Tên',
        description: 'Mô tả'
      },
      placeholders: {
        name: 'Enter template name',
        description: 'Tùy chọn: mô tả'
      }
    },
    backtestRun: {
      title: 'Backtest run',
      status: {
        queued: 'Queued',
        running: 'Đang chạy',
        completed: 'Hoàn tất',
        failed: 'Thất Bại',
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
        status: 'Trạng thái',
        error: 'Lỗi',
        maxDrawdown: 'Max Drawdown',
        sharpe: 'Sharpe'
      },
      metrics: {
        totalReturn: 'Tổng lợi nhuận',
        annualReturn: 'Lợi nhuận năm',
        maxDrawdown: 'Sụt giảm tối đa',
        sharpe: 'Sharpe',
        winRate: 'Tỷ lệ thắng',
        totalTrades: 'Số lệnh',
        equityCurvePoints: 'Equity curve points'
      },
      trades: {
        title: 'Order details',
        empty: 'No trades recorded',
        loadFailed: 'Failed to load order details',
        ticket: 'Mã lệnh',
        side: 'Hướng',
        sideBuy: 'Buy',
        sideSell: 'Sell',
        volume: 'Volume',
        openTime: 'Open time',
        openPrice: 'Giá mở',
        closeTime: 'Close time',
        closePrice: 'Giá đóng',
        pnl: 'P&L',
        commission: 'Commission',
        reason: 'Close reason',
        reasons: {
          signal: 'Tín hiệu (đặt lệnh)',
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
      title: 'Nhật ký',
      titleWithName: 'Nhật ký - {{name}}',
      messages: {
        missingScheduleId: 'Missing schedule ID'
      },
      execStatus: {
        pending: 'Chưa kiểm tra',
        running: 'Đang chạy',
        completed: 'Hoàn tất',
        failed: 'Thất Bại',
        skipped: 'Skipped'
      },
      operationStatus: {
        success: 'Thành Công',
        failed: 'Thất Bại',
        running: 'Đang chạy'
      },
      execTable: {
        time: 'Thời gian',
        action: 'Hành động',
        execute: 'Thực thi',
        status: 'Trạng thái',
        durationMs: 'Thời lượng (ms)',
        error: 'Lỗi'
      },
      ordersTable: {
        time: 'Thời gian',
        side: 'Hướng',
        symbol: 'Mã',
        lots: 'Khối lượng (Lot)',
        openPrice: 'Giá mở',
        closePrice: 'Giá đóng',
        profit: 'Lãi/Lỗ',
        ticket: 'Mã lệnh'
      },
      orderSide: {
        buy: 'Mua thị trường',
        sell: 'Bán thị trường',
        close: 'Đóng lệnh',
        buyLimit: 'Mua giới hạn',
        sellLimit: 'Bán giới hạn',
        buyStop: 'Mua stop',
        sellStop: 'Bán stop',
        buyStopLimit: 'Mua stop limit',
        sellStopLimit: 'Sell stop limit'
      },
      scheduleIdLabel: 'ID lịch chạy:',
      summary: {
        name: 'Tên',
        status: 'Trạng thái',
        trade: 'Giao dịch',
        enableCount: 'Số lần bật',
        lastRun: 'Lần chạy gần nhất',
        lastError: 'Last error'
      },
      tabs: {
        exec: 'Lần chạy',
        orders: 'Lệnh',
        execLogs: '执行日志',
        orderLogs: 'Order Logs'
      },
      status: {
        success: 'Thành Công',
        failed: 'Thất Bại'
      },
      action: {
        start: 'Bắt Đầu',
        stop: 'Dừng',
        restart: 'Restart'
      }
    },
    schedules: {
      title: 'Lịch chạy chiến lược',
      createSchedule: 'Tạo lịch',
      format: {
        interval: 'mỗi {{s}}s',
        cron: 'cron: {{expr}}',
      },
      status: {
        running: 'Đang chạy',
        disabled: 'Disabled'
      },
      templateVisibility: {
        public: 'Công khai',
        private: 'Riêng tư'
      },
      table: {
        name: 'Tên',
        template: 'Mẫu',
        account: 'Tài khoản',
        tradeParams: 'Tham số giao dịch',
        schedule: 'Lịch',
        status: 'Trạng thái',
        lastRun: 'Lần chạy gần nhất',
        actions: 'Thao tác'
      },
      nextRunAt: 'Lần chạy kế tiếp',
      enableCount: 'Số lần bật',
      actions: {
        create: 'Tạo lịch',
        logs: 'Nhật ký chạy',
        healthCheck: 'Kiểm tra sức khỏe',
        runNow: 'Run now'
      },
      health: {
        title: 'Kiểm tra sức khỏe chiến lược {{name}}',
        summaryBanner: 'Mức sức khỏe: {{grade}}; mẫu gần nhất {{totalRuns}} lần, tỷ lệ thành công {{successRate}}%',
        grade: {
          pending: 'Chưa kiểm tra',
          noSample: 'Thiếu mẫu',
          healthy: 'Tốt',
          watch: 'Cần theo dõi',
          alert: 'Alert'
        },
        notes: {
          pending: 'Vui lòng chạy kiểm tra sức khỏe trước.',
          noSample: 'Không đủ mẫu để đánh giá (tối thiểu {{minSampleSize}}).',
          healthy: 'Tỷ lệ thành công cao và số lần thất bại trong ngưỡng.',
          watch: 'Tỷ lệ thành công đạt ngưỡng theo dõi (>= {{yellowSuccessRate}}%).',
          alert: 'Low success rate. Investigate strategy/account conditions now.'
        },
        fields: {
          grade: 'Mức sức khỏe',
          rule: 'Tiêu chí đánh giá',
          thresholds: 'Ngưỡng hiện tại',
          configKey: 'Khóa cấu hình',
          lastRunAt: 'Lần chạy gần nhất',
          latestTicket: 'Ticket khớp lệnh gần nhất',
          successOverTotal: 'Thành công / Tổng',
          failedRuns: 'Số lần thất bại',
          latestProfit: 'Lãi/lỗ gần nhất',
          latestError: 'Latest error'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}; xanh: success>={{greenSuccessRate}}% & failed<={{greenMaxFailedRuns}}; vàng: success>={{yellowSuccessRate}}%',
        sections: {
          runLogs: 'Nhật ký chạy gần đây',
          orders: 'Recent order records'
        },
        runLogs: {
          signalType: 'Tín hiệu (đặt lệnh)'
        },
        messages: {
          loadFailed: 'Tải dữ liệu sức khỏe thất bại',
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
        parametersParseFailed: 'Phân tích tham số thất bại',
        defaultTemplateNotFound: 'Không tìm thấy mẫu mặc định. Vui lòng làm mới và thử lại.',
        importDefaultTemplateFailedNoId: 'Nhập mẫu mặc định thất bại: thiếu template id',
        templateCodeEmptyCannotExecute: 'Code mẫu trống. Không thể thực thi.',
        strategyExecuteFailed: 'Thực thi chiến lược thất bại',
        executeFailed: 'Thực thi thất bại',
        noOrderableSignal: 'Không có tín hiệu có thể đặt lệnh',
        signalHoldCannotOrder: 'Tín hiệu là hold/không hành động. Không thể đặt lệnh.',
        volumeInvalid: 'Khối lượng không hợp lệ (phải > 0)',
        orderSubmitted: 'Đã gửi lệnh',
        orderFailed: 'Đặt lệnh thất bại'
      },
      editModal: {
        title: {
          edit: 'Chỉnh sửa lịch chạy',
          create: 'Tạo lịch chạy'
        },
        fields: {
          template: 'Mẫu',
          templateExtra: 'Mẫu đã lưu trong “Quản lý chiến lược”',
          account: 'Tài khoản',
          name: 'Tên',
          symbol: 'Mã',
          lot: 'Khối lượng (Lot)',
          lotExtra: 'Khối lượng đặt lệnh. Khuyến nghị bắt đầu từ 0.01',
          runFrequency: 'Tần suất chạy',
          cronExpression: 'Cron (nâng cao)',
          cronExtra: 'Cron 5 phần: phút giờ ngày tháng thứ. VD: */5 * * * *; 0 9 * * 1-5',
          intervalSeconds: 'Khoảng (giây)',
          intervalSecondsExtra: 'Tự theo timeframe; không cần chỉnh',
          enableExtra: 'Enable schedule after creating'
        },
        placeholders: {
          name: 'VD: EURUSD M5 chiến lược buổi sáng',
          selectAccountFirst: 'Chọn tài khoản trước',
          symbol: 'Chọn mã'
        },
        validation: {
          templateRequired: 'Vui lòng chọn mẫu',
          accountRequired: 'Vui lòng chọn tài khoản',
          nameRequired: 'Vui lòng nhập tên',
          symbolRequired: 'Vui lòng chọn mã',
          lotRequired: 'Vui lòng nhập khối lượng',
          runFrequencyRequired: 'Vui lòng chọn tần suất chạy',
          cronRequired: 'Vui lòng nhập cron',
          timeframeRequired: 'Vui lòng chọn timeframe',
          triggerModeRequired: 'Trigger mode is required'
        },
        runFrequencyExtra: {
          cron: 'Nâng cao: dùng Cron để điều khiển thời điểm chạy',
          byTimeframe: 'Run by timeframe'
        },
        runFrequencyOptions: {
          byTimeframe: 'Theo timeframe (khuyến nghị)',
          cron: 'Cron'
        },
        autoName: {
          strategy: 'Strategy'
        },
        advanced: {
          title: 'Nâng cao',
          fixedIntervalSeconds: 'Khoảng cố định (giây)',
          fixedIntervalSecondsExtra: 'Tùy chọn. Chạy theo khoảng cố định thay vì theo timeframe. VD: 60 = mỗi 60 giây',
          timeframe: 'Khung thời gian',
          timeframeExtra: 'Dùng cho tính nến/chỉ báo',
          triggerMode: 'Chế độ kích hoạt',
          triggerModeExtra: 'Ổn định: theo nến/timeframe; Cao tần: theo báo giá (nhanh hơn, cần debounce)',
          triggerModeOptions: {
            stable: 'Ổn định (nến/timeframe)',
            hf: 'High-frequency signal stream'
          },
          stableOverrideIntervalSeconds: 'Ghi đè khoảng ổn định (giây)',
          stableOverrideIntervalSecondsExtra: 'Tùy chọn. Ghi đè khoảng kích hoạt ở chế độ ổn định',
          hfCooldownMs: 'Cooldown cao tần (ms)',
          hfCooldownMsExtra: 'Debounce: khoảng tối thiểu giữa các lần đánh giá/đặt lệnh',
          parametersJson: 'Tham số (JSON object)',
          parametersJsonExtra: 'JSON parameters for the strategy'
        }
      },
      triggerModal: {
        title: 'Chạy ngay (đặt lệnh)',
        confirmOrder: {
          title: 'Xác nhận đặt lệnh',
          ok: 'Confirm'
        },
        actions: {
          confirmOrder: 'Xác nhận đặt lệnh',
          rerun: 'Re-run'
        },
        summary: {
          scheduleName: 'Tên lịch',
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian'
        },
        messages: {
          signalNotOrderable: 'Signal is not orderable'
        },
        cards: {
          logs: 'Nhật ký chạy',
          signal: 'Tín hiệu (đặt lệnh)'
        },
        emptyLogs: '(không có nhật ký)',
        emptySignal: 'No signal'
      }
    },
    asset: {
      title: 'Strategy Assets',
      subtitle: 'Asset publishing, review status, and cloning are maintained by the system. Cloned results are independent user templates.',
      submitAsset: 'Submit Asset',
      assetList: 'Asset List',
      name: 'Tên',
      visibility: 'Hiển thị',
      reviewStatus: 'Review Status',
      cloneCount: 'Clones',
      version: 'Version',
      description: 'Mô tả',
      actions: 'Thao tác',
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
      create: 'Tạo lịch',
      noAccounts: 'No paper accounts. Create one to start simulated trading.',
      running: 'Running {{symbol}} {{timeframe}}',
      start: 'Bắt Đầu',
      stop: 'Dừng',
      watch: 'Cần theo dõi',
      paper: 'Paper',
      startStrategy: 'Start Paper Strategy',
      symbol: 'Mã',
      timeframe: 'Khung thời gian',
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
      template: 'Mẫu',
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
      reviseSend: 'Gửi AI để sửa',
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
        symbol: 'Mã',
        symbolRequired: 'Vui lòng chọn mã',
        symbolPlaceholder: 'EURUSD',
        timeframe: 'Khung thời gian',
        klineCount: 'K-line Count',
        submit: 'Start Detection'
      },
      result: {
        title: 'Detection Result',
        status: 'Trạng thái',
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
          status: 'Trạng thái',
          searchMethod: 'Search Method',
          maxCandidates: 'Max Candidates',
          objective: 'Objective',
          actions: 'Thao tác',
          viewCandidates: 'View Candidates'
        }
      },
      candidates: {
        title: 'Candidates',
        titleWithId: 'Candidates: {{id}}',
        column: {
          rank: 'Rank',
          grade: 'Grade',
          score: 'Điểm',
          parameters: 'Parameters',
          summary: 'Summary',
          recommendation: 'Recommendation',
          actions: 'Thao tác',
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
      account: 'Tài khoản',
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
      completedStatus: 'Hoàn tất',
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
      selectSymbol: 'Mã',
      code: 'Strategy Code',
      codePlaceholder: `# Python strategy code...
def run(context):
    return {"signal": "hold"}`,
      validate: 'Xác thực mã',
      validatePass: 'Xác thực thành công',
      validateFailed: 'Xác thực thất bại',
      validateBeforeSave: 'Please validate code before saving',
      runBacktest: 'Run Backtest',
      save: 'Save',
      copy: 'Sao chép',
      copySuccess: 'Đã sao chép',
      copyFailed: 'Sao chép thất bại, vui lòng sao chép thủ công',
      saveSuccess: 'Saved',
      chart: 'K-line',
      backtest: 'Kiểm thử lùi',
      backtestRunning: 'Backtest running...',
      backtestCompleted: 'Hoàn tất',
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
        title: 'Mẫu',
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
        orderFailed: 'Đặt lệnh thất bại',
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
      title: 'Kiểm thử lùi',
      currentDraft: '📝 Current Draft',
      dateRange: 'Date Range',
      execution: 'Execution',
      capital: 'Capital',
      leverage: 'Leverage',
      commission: 'Commission',
      slippage: 'Slippage',
      trade: 'Giao dịch',
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
      preview: 'Xem trước tín hiệu',
      previewTitle: 'Preview ({{shown}} of {{total}})',
      truncated: 'TRUNCATED',
      results: 'Results ({{count}})',
      rank: '#',
      grade: 'Grade',
      score: 'Điểm',
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
      tradeTime: 'Thời gian',
      tradeSide: 'Hướng',
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
      orderFailed: 'Đặt lệnh thất bại'
    },
    library: {
      title: 'Strategy Library',
      myStrategies: 'My Strategies',
      create: 'Tạo lịch',
      filterAll: 'All',
      filterMine: 'My',
      filterSystem: 'Mặc định',
      searchPlaceholder: 'Search strategies...',
      empty: 'No strategies found',
      system: 'System',
      shared: 'Shared',
      private: 'Riêng tư',
      share: 'Share',
      published: 'Đã xuất bản',
      draft: 'Nháp',
      unpublish: 'Unpublish',
      unpublishShort: 'Off',
      publish: 'Publish to Market',
      publishSuccess: 'Đã xuất bản',
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
    title: 'Danh mục chỉ báo',
    description: 'Các chỉ báo kỹ thuật và tham số rủi ro có sẵn trong sandbox chiến lược. Chỉ sử dụng các helper và khóa tham số này trong mã chiến lược.',
    indicatorsTitle: 'Chỉ báo kỹ thuật',
    riskSectionTitle: 'Tham số quản lý rủi ro',
    riskParamsTitle: 'Tham số rủi ro chung',
    riskParamsDesc: 'Mọi chiến lược nên tuân thủ các tham số quản lý rủi ro này bất kể chỉ báo nào được chọn.',
    paramKey: 'Khóa',
    paramLabel: 'Nhãn',
    paramType: 'Loại',
    paramDefault: 'Mặc định',
    paramRange: 'Phạm vi',
    paramDescription: 'Mô tả'
  }
} as const;

export default strategy;
