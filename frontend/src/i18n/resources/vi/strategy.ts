const strategy = {
  strategy: {
    validation: {
      passed: 'Xác minh code thành công',
      notPassed: 'Xác minh code không đạt',
      riskEval: {
        title: 'Đánh giá rủi ro',
        riskHigh: 'Mức rủi ro: cao',
        riskUnreliable: 'Đánh giá rủi ro: không đáng tin cậy (isReliable=false)',
        riskLoading: 'Đánh giá rủi ro phía máy chủ vẫn đang tính toán'
      }
    },
    codeEditor: {
      title: 'Trình soạn thảo chiến lược',
      labels: {
        account: 'Tài khoản',
        symbol: 'Mã',
        timeframe: 'Khung thời gian',
        code: 'Mã chiến lược',
        disabledSuffix: ' (đã tắt)'
      },
      actions: {
        copy: 'Sao chép',
        validate: 'Xác thực mã',
        preview: 'Xem trước tín hiệu',
        saveAsTemplate: 'Lưu thành template',
        sendToAI: 'Gửi AI để sửa',
        sendToAIFixTitleValidate: 'Xác thực thất bại / có cảnh báo',
        sendToAIFixTitlePreview: 'Xem trước thất bại / cần tối ưu'
      },
      placeholders: {
        selectAccount: 'Chọn tài khoản',
        selectAccountFirst: 'Vui lòng chọn tài khoản trước',
        loadingSymbols: 'Đang tải danh sách mã...',
        selectSymbol: 'Chọn mã',
        noSymbols: 'Không lấy được danh sách mã',
        code: 'Dán mã chiến lược Python...',
        codeSample: 'Dán mã chiến lược vào đây',
      },
      cards: {
        validationResult: 'Kết quả xác thực',
        previewResult: 'Kết quả xem trước'
      },
      hints: {
        previewInfo: 'Xem trước dùng N nến gần nhất (mặc định 500, cấu hình: strategy.preview_bars); backtest dùng N tháng gần nhất (mặc định 3, cấu hình: strategy.backtest_window_months).'
      },
      messages: {
        enterCode: 'Vui lòng nhập mã chiến lược',
        selectAccount: 'Vui lòng chọn tài khoản',
        validateOk: 'Xác thực thành công',
        validateFailed: 'Xác thực thất bại',
        validateError: 'Lỗi xác thực',
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
        outputTitle: '[Đầu ra]',
        outro: 'Vui lòng xuất toàn bộ mã đã chỉnh sửa (bọc bằng ```python) và giải thích thay đổi.',
        pythonFenceStart: '```python',
        fenceEnd: '```'
      }
    },
    schedules: {
      title: 'Lịch chạy chiến lược',
      createSchedule: 'Tạo lịch',
      format: {
        interval: 'mỗi {{s}}s',
        cron: 'cron: {{expr}}',
      },
      actions: {
        create: 'Tạo lịch',
        logs: 'Nhật ký',
        healthCheck: 'Kiểm tra sức khỏe',
        runNow: 'Chạy ngay'
      },
      health: {
        title: 'Kiểm tra sức khỏe chiến lược {{name}}',
        summaryBanner: 'Mức sức khỏe: {{grade}}; mẫu gần nhất {{totalRuns}} lần, tỷ lệ thành công {{successRate}}%',
        grade: {
          pending: 'Chưa kiểm tra',
          noSample: 'Thiếu mẫu',
          healthy: 'Tốt',
          watch: 'Cần theo dõi',
          alert: 'Cảnh báo'
        },
        notes: {
          pending: 'Vui lòng chạy kiểm tra sức khỏe trước.',
          noSample: 'Không đủ mẫu để đánh giá (tối thiểu {{minSampleSize}}).',
          healthy: 'Tỷ lệ thành công cao và số lần thất bại trong ngưỡng.',
          watch: 'Tỷ lệ thành công đạt ngưỡng theo dõi (>= {{yellowSuccessRate}}%).',
          alert: 'Tỷ lệ thành công thấp, cần kiểm tra ngay chiến lược và trạng thái tài khoản.'
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
          latestError: 'Lỗi gần nhất'
        },
        thresholdsSummary: 'min_sample_size={{minSampleSize}}; xanh: success>={{greenSuccessRate}}% & failed<={{greenMaxFailedRuns}}; vàng: success>={{yellowSuccessRate}}%',
        sections: {
          runLogs: 'Nhật ký chạy gần đây',
          orders: 'Lịch sử khớp lệnh gần đây'
        },
        runLogs: {
          signalType: 'Tín hiệu'
        },
        messages: {
          loadFailed: 'Tải dữ liệu sức khỏe thất bại',
          clickRefresh: 'Nhấn làm mới để tải dữ liệu sức khỏe'
        }
      },
      editModal: {
        title: {
          create: 'Tạo lịch chạy',
          edit: 'Chỉnh sửa lịch chạy'
        },
        autoName: {
          strategy: 'Chiến lược'
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
          cronExpression: 'Cron',
          cronExtra: 'Cron 5 phần: phút giờ ngày tháng thứ. VD: */5 * * * *; 0 9 * * 1-5',
          intervalSeconds: 'Khoảng (giây)',
          intervalSecondsExtra: 'Tự theo timeframe; không cần chỉnh',
          enableExtra: 'Giống EA: bật sẽ chạy liên tục đến khi bạn tắt'
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
          triggerModeRequired: 'Vui lòng chọn chế độ kích hoạt'
        },
        runFrequencyExtra: {
          cron: 'Nâng cao: dùng Cron để điều khiển thời điểm chạy',
          byTimeframe: 'Mặc định: kích hoạt theo timeframe (giống EA)'
        },
        runFrequencyOptions: {
          byTimeframe: 'Theo timeframe (khuyến nghị)',
          cron: 'Cron (nâng cao)'
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
            hf: 'Cao tần (báo giá/tick)'
          },
          stableOverrideIntervalSeconds: 'Ghi đè khoảng ổn định (giây)',
          stableOverrideIntervalSecondsExtra: 'Tùy chọn. Ghi đè khoảng kích hoạt ở chế độ ổn định',
          hfCooldownMs: 'Cooldown cao tần (ms)',
          hfCooldownMsExtra: 'Debounce: khoảng tối thiểu giữa các lần đánh giá/đặt lệnh',
          parametersJson: 'Tham số (JSON object)',
          parametersJsonExtra: 'Truyền tham số vào code chiến lược (dạng chuỗi). VD: { "fast": 10, "slow": 20, "risk": "low" }'
        }
      },
      triggerModal: {
        title: 'Chạy ngay (đặt lệnh)',
        actions: {
          rerun: 'Chạy lại',
          confirmOrder: 'Xác nhận đặt lệnh'
        },
        confirmOrder: {
          title: 'Xác nhận đặt lệnh?',
          ok: 'Xác nhận'
        },
        summary: {
          scheduleName: 'Tên lịch',
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian'
        },
        messages: {
          signalNotOrderable: 'Không thể đặt lệnh: cần buy/sell và volume > 0'
        },
        cards: {
          logs: 'Nhật ký chạy',
          signal: 'Tín hiệu (đặt lệnh)'
        },
        emptyLogs: '(không có nhật ký)',
        emptySignal: '(không có tín hiệu)'
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
      templateVisibility: {
        public: 'Công khai',
        private: 'Riêng tư'
      },
      status: {
        running: 'Đang chạy',
        disabled: 'Đã tắt'
      },
      nextRunAt: 'Lần chạy kế tiếp',
      enableCount: 'Số lần bật',
      deleteConfirm: {
        title: 'Xóa lịch này?'
      },
      validation: {
        parametersMustBeJsonObject: 'Tham số phải là một đối tượng JSON'
      },
      messages: {
        parametersParseFailed: 'Phân tích tham số thất bại',
        defaultTemplateNotFound: 'Không tìm thấy mẫu mặc định. Vui lòng làm mới và thử lại.',
        importDefaultTemplateFailedNoId: 'Nhập mẫu mặc định thất bại: thiếu template id',
        templateCodeEmptyCannotExecute: 'Code mẫu trống. Không thể thực thi.',
        executeFailed: 'Thực thi thất bại',
        strategyExecuteFailed: 'Thực thi chiến lược thất bại',
        noOrderableSignal: 'Không có tín hiệu có thể đặt lệnh',
        signalHoldCannotOrder: 'Tín hiệu là hold/không hành động. Không thể đặt lệnh.',
        volumeInvalid: 'Khối lượng không hợp lệ (phải > 0)',
        orderSubmitted: 'Đã gửi lệnh',
        orderFailed: 'Đặt lệnh thất bại'
      }
    },
    scheduleLogs: {
      title: 'Nhật ký',
      titleWithName: 'Nhật ký - {{name}}',
      tabs: {
        exec: 'Lần chạy',
        orders: 'Lệnh',
        execLogs: '执行日志',
        orderLogs: '订单日志'
      },
      messages: {
        missingScheduleId: 'Thiếu scheduleId'
      },
      summary: {
        name: 'Tên',
        status: 'Trạng thái',
        trade: 'Giao dịch',
        enableCount: 'Số lần bật',
        lastRun: 'Lần chạy gần nhất',
        lastError: 'Lỗi gần nhất'
      },
      execStatus: {
        pending: 'Chờ',
        running: 'Đang chạy',
        completed: 'Hoàn tất',
        failed: 'Thất bại',
        skipped: 'Bỏ qua'
      },
      operationStatus: {
        success: 'Thành công',
        failed: 'Thất bại',
        running: 'Đang chạy'
      },
      execTable: {
        time: 'Thời gian',
        action: 'Hành động',
        status: 'Trạng thái',
        durationMs: 'Thời lượng (ms)',
        error: 'Lỗi',
        execute: 'Thực thi'
      },
      ordersTable: {
        time: 'Thời gian',
        side: 'Hướng',
        symbol: 'Ký hiệu',
        lots: 'Khối lượng',
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
        sellStopLimit: 'Bán stop limit'
      },
      scheduleIdLabel: 'ID lịch chạy:',
      status: {
        success: 'Thành Công',
        failed: 'Thất Bại'
      },
      action: {
        start: 'Bắt Đầu',
        stop: 'Dừng',
        restart: 'Khởi Động Lại'
      }
    },
    templates: {
      title: 'Mẫu chiến lược',
      tabs: {
        system: 'Mẫu hệ thống',
        user: 'Mẫu tự tạo'
      },
      copySuffix: ' (Bản sao)',
      scheduleName: '{{symbol}} {{timeframe}} {{name}}',
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
        createTemplate: 'Tạo mẫu',
        edit: 'Chỉnh sửa',
        delete: 'Xóa',
        copy: 'Sao chép',
        viewCode: 'Xem code',
        backtest: 'Kiểm thử lùi',
        launchSchedule: 'Khởi chạy lịch'
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
        emptyUser: 'Chưa có mẫu nào. Nhấp "Tạo mẫu" ở trên để bắt đầu.'
      },
      scheduleLaunch: {
        title: 'Khởi chạy lịch',
        noRun: 'Chưa có lần chạy backtest',
        backtestRunningHint: 'Backtest đang chạy. Vui lòng đợi.',
        score: 'Điểm',
        keyMetrics: 'Chỉ số chính',
        launchSection: 'Khởi chạy lịch',
        actions: {
          publishTemplate: 'Xuất bản template',
          createScheduleNoEnable: 'Tạo lịch',
          createAndEnable: 'Tạo & bật',
          create: '创建计划',
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
          account: 'Tài Khoản',
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
          symbol: 'Mã',
          symbolPlaceholder: '选择品种',
          symbolPlaceholderEmpty: '未配置品种',
          timeframe: 'Khung Thời Gian',
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
        newPasswordPlaceholder: '输入新交易密码'
      },
      editTemplateModal: {
        title: {
          create: 'Tạo mẫu',
          edit: 'Chỉnh sửa mẫu'
        },
        fields: {
          name: 'Tên',
          description: 'Mô tả',
          code: 'Code chiến lược',
          publicShare: 'Chia sẻ công khai'
        },
        placeholders: {
          name: 'VD: Chiến lược cắt MA',
          description: 'Tùy chọn: mô tả',
          codeSample: `# Ví dụ code chiến lược
# Biến có sẵn: close, open, high, low, volume, symbol
# Trả về: signal (dict)

import numpy as np

# Chỉ báo
maFast = np.mean(close[-10:])
maSlow = np.mean(close[-20:])

# Tín hiệu
if maFast > maSlow:
    signal = 'buy'
elif maFast < maSlow:
    signal = 'sell'
else:
    signal = 'hold'

# Kết quả
signal = {
    'signal': signal,
    'symbol': symbol,
    'price': close[-1],
    'confidence': 0.7,
    'reason': f'maFast={maFast:.5f}, maSlow={maSlow:.5f}'
}`
        },
        validation: {
          nameRequired: 'Vui lòng nhập tên',
          codeRequired: 'Vui lòng nhập code chiến lược'
        },
        actions: {
          validateCode: 'Xác thực mã'
        }
      },
      backtest: {
        title: 'Kiểm thử lùi',
        fields: {
          title: 'Tiêu đề',
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Khung thời gian',
          initialCapital: 'Vốn ban đầu',
          range: 'Khoảng thời gian',
          extraSymbols: 'Mã bổ sung (chọn nhiều)'
        },
        placeholders: {
          account: 'Chọn tài khoản',
          symbol: 'Chọn mã',
          range: 'Chọn khoảng thời gian',
          extraSymbols: 'Tùy chọn, hữu ích cho chiến lược cặp/xoay vòng'
        },
        validation: {
          accountRequired: 'Vui lòng chọn tài khoản',
          symbolRequired: 'Vui lòng chọn mã',
          timeframeRequired: 'Vui lòng chọn timeframe',
          initialCapitalRequired: 'Vui lòng nhập vốn ban đầu',
          rangeRequired: 'Vui lòng chọn khoảng thời gian'
        },
        quickRange: {
          '1d': '1 ngày',
          '3d': '3 ngày',
          '1w': '1 tuần',
          '1y': '1 năm',
          custom: 'Tùy chỉnh'
        },
        accountDisabledSuffix: ' (Đã tắt)',
        modalTitleWithName: 'Kiểm thử lùi: {{name}}',
        parameters: {
          title: 'Tham số chiến lược'
        },
        tooltips: {
          extraSymbols: 'Các mã bổ sung để lấy nến K (cùng tài khoản, cùng khung thời gian). Chiến lược có thể truy cập qua context["closes_by_symbol"].'
        }
      },
      messages: {
        fetchTemplateListFailed: 'Tải danh sách mẫu thất bại',
        enterStrategyCode: 'Vui lòng nhập code chiến lược',
        codeValidationPassed: 'Xác minh code thành công',
        codeValidationNotPassed: 'Xác minh code không đạt',
        codeValidationFailed: 'Xác minh code thất bại',
        templateUpdated: 'Đã cập nhật mẫu',
        templateCreated: 'Đã tạo mẫu',
        templateDeleted: 'Đã xóa mẫu',
        readStrategyCodeFailed: 'Tải code chiến lược thất bại',
        strategyCodeEmptyCannotBacktest: 'Code chiến lược trống. Không thể backtest.',
        selectBacktestRange: 'Vui lòng chọn khoảng backtest',
        backtestRangeInvalid: 'Khoảng backtest không hợp lệ',
        backtestSubmitted: 'Đã gửi backtest',
        backtestSubmitFailed: 'Gửi backtest thất bại',
        backtestCancelRequested: 'Đã yêu cầu hủy backtest',
        backtestCancelFailed: 'Hủy backtest thất bại',
        backtestReportDeleted: 'Đã xóa báo cáo backtest',
        backtestReportNotFound: 'Không tìm thấy báo cáo backtest',
        codeCopied: 'Đã sao chép code',
        copyFailed: 'Sao chép thất bại',
        missingScheduleInfo: 'Thiếu thông tin cần thiết để tạo lịch',
        templateNotPublishedCannotCreateSchedule: 'Mẫu chưa xuất bản. Không thể tạo lịch.',
        readTemplateStatusFailed: 'Không thể đọc trạng thái mẫu',
        scheduleCreated: 'Đã tạo lịch',
        scheduleCreatedAndEnabled: 'Đã tạo lịch và bật',
        createScheduleFailed: 'Tạo lịch thất bại',
        deepLinkNavigate: 'Đã mở mẫu và chi tiết lần chạy gần nhất từ liên kết ngoài',
        templatePublished: 'Đã xuất bản mẫu',
        cannotPublishAndCreateDraftFailed: 'Không thể xuất bản. Tạo bản nháp thất bại.',
        republishedButNoTemplateId: 'Đã xuất bản lại, nhưng thiếu ID mẫu.',
        backtestRunningCannotPublish: 'Backtest đang chạy. Không thể xuất bản ngay bây giờ.',
        missingDraftIdCannotPublish: 'Thiếu ID bản nháp. Không thể xuất bản.',
        publishedButNoTemplateId: 'Đã xuất bản, nhưng thiếu ID mẫu.',
        templateRepublished: 'Đã xuất bản lại mẫu',
        templateAlreadyPublished: 'Mẫu đã được xuất bản',
        templateNotDraftUnknownPublishStatus: 'Mẫu không phải bản nháp. Không xác định trạng thái xuất bản.',
        publishFailed: 'Xuất bản thất bại',
        backtestRunNoPublishedTemplate: 'Lần chạy backtest không có mẫu đã xuất bản',
        strategyCodeEmptyCannotPublish: '策略代码为空，请先保存代码再发布。',
        systemTemplateReadOnly: '系统模板为只读，请克隆后再编辑。'
      },
      backtestRuns: {
        title: 'Báo cáo backtest',
        empty: 'Chưa có backtest',
        deleteConfirm: 'Xóa báo cáo backtest này?',
        batchDelete: 'Xóa {{count}}',
        batchDeleteConfirm: 'Xóa {{count}} báo cáo backtest?',
        batchDeleteSuccess: 'Đã xóa {{count}} báo cáo backtest',
        status: {
          queued: 'Đang chờ',
          running: 'Đang chạy',
          completed: 'Hoàn tất',
          failed: 'Thất bại',
          canceling: 'Đang hủy',
          canceled: 'Đã hủy'
        },
        table: {
          title: 'Tiêu đề',
          status: 'Trạng thái',
          symbol: 'Mã',
          timeframe: 'Khung thời gian',
          createdAt: 'Thời gian tạo',
          actions: 'Thao tác'
        },
        actions: {
          view: 'Xem',
          launchSchedule: 'Khởi chạy lịch',
          createSchedule: 'Tạo lịch'
        }
      },
      deleteConfirm: 'Xóa mẫu này?',
      defaultDraftName: 'Mẫu nháp',
      codeModal: {
        title: 'Mã chiến lược',
        actions: {
          copy: 'Sao chép'
        }
      }
    },
    defaultTemplates: {
      maCross: {
        name: 'MA Crossover',
        description: 'Mua khi MA nhanh cắt lên MA chậm; bán khi cắt xuống'
      },
      forceBuy: {
        name: 'Force BUY (Test)',
        description: 'Dùng để kiểm tra luồng đặt lệnh: luôn trả về buy; đọc lot từ context/params làm volume'
      },
      rsi: {
        name: 'RSI Quá mua/Quá bán',
        description: 'Mua khi RSI < 30; bán khi RSI > 70'
      },
      macd: {
        name: 'MACD Crossover',
        description: 'Mua khi MACD cắt lên; bán khi cắt xuống'
      }
    },
    codeAssist: {
      tabAI: 'AI sửa code',
      tabExplain: 'Giải thích code',
      explain: 'Giải thích code',
      requiredParamsTitle: 'Tham số bắt buộc',
      requiredParamsDesc: 'Chiến lược đọc các tham số này nhưng không có giá trị mặc định. Vui lòng điền trước khi lưu.',
      optionalParamsTitle: 'Tham số tùy chọn',
      optionalParamsDesc: 'Các tham số này đã có giá trị mặc định trong code. Để trống để dùng mặc định; nhập giá trị mới chỉ áp dụng cho lần chạy này và không sửa chiến lược đã lưu.',
      defaultLabel: 'mặc định',
      paramDescriptions: {
        riskLevel: 'Mức rủi ro (low / medium / high). Ảnh hưởng kích thước lệnh và biên cắt lỗ / chốt lời.',
        takeProfit: 'Biên chốt lời (%). Đóng lệnh khi giá đi đúng hướng đạt phần trăm này.',
        stopLoss: 'Biên cắt lỗ (%). Đóng lệnh khi giá đi ngược đạt phần trăm này.',
        maxLoss: 'Lỗ tối đa cho mỗi giao dịch (theo tỉ lệ vốn, 0.01 = 1%).',
        confidence: 'Ngưỡng độ tin cậy của tín hiệu (0–1). Tín hiệu thấp hơn sẽ bị bỏ qua.',
        threshold: 'Ngưỡng kích hoạt tín hiệu. Ý nghĩa cụ thể tuỳ theo logic trong code.',
        lotSize: 'Khối lượng lệnh (lot). Càng lớn rủi ro càng cao.',
        fastPeriod: 'Chu kỳ nhanh (số nến). Dùng cho MACD / 2 đường MA; càng nhỏ càng nhạy.',
        slowPeriod: 'Chu kỳ chậm (số nến). Dùng cho MACD / 2 đường MA; càng lớn càng mượt.',
        signalPeriod: 'Chu kỳ đường tín hiệu (số nến). Làm mượt DIF/DEA của MACD.',
        rsiPeriod: 'Chu kỳ RSI (số nến). Giá trị thường dùng: 14.',
        emaPeriod: 'Chu kỳ EMA (đường trung bình hàm mũ), tính bằng số nến.',
        smaPeriod: 'Chu kỳ SMA (đường trung bình đơn giản), tính bằng số nến.',
        genericPeriod: 'Chu kỳ nhìn lại (số nến) dùng để tính chỉ báo.',
        genericPercent: 'Tham số dạng phần trăm / tỉ lệ (ví dụ 1 nghĩa là 1%).'
      },
      required: 'bắt buộc',
      suggested: 'gợi ý',
      applyAllSuggestions: 'Áp dụng giá trị gợi ý',
      fillRequiredParams: 'Vui lòng điền các tham số bắt buộc: {{keys}}',
      aiReviseTitle: 'Trợ lý AI — sửa code',
      reviseInputPlaceholder: 'Ví dụ: thay SMA(20) bằng EMA(50) và thêm cắt lỗ 1%.',
      reviseSend: 'Gửi cho AI',
      enterInstruction: 'Mô tả thay đổi bạn muốn thực hiện.',
      codeEmpty: 'Chưa có code để sửa.',
      codeUpdated: 'Code đã được cập nhật. Vui lòng chạy lại xác thực trước khi lưu.',
      noPython: 'AI không trả về khối Python. Hãy diễn đạt lại và thử lại.',
      saveBlockedNotValidated: 'Vui lòng nhấn "Xác thực mã" trước. Lưu sẽ bị vô hiệu hóa cho đến khi xác thực thành công.',
      generatePlaceholder: '描述你的策略需求...'
    },
    templateModal: {
      title: 'Lưu làm mẫu',
      fields: {
        name: 'Tên',
        description: 'Mô tả'
      },
      placeholders: {
        name: 'Nhập tên mẫu',
        description: 'Nhập mô tả'
      }
    },
    backtestRun: {
      title: 'Lần chạy backtest',
      status: {
        queued: 'Đang chờ',
        running: 'Đang chạy',
        completed: 'Hoàn thành',
        failed: 'Thất bại',
        canceling: 'Đang hủy',
        canceled: 'Đã hủy',
        ended: 'Đã kết thúc'
      },
      actions: {
        cancel: 'Hủy'
      },
      hints: {
        queued: 'Backtest đang trong hàng đợi',
        running: 'Backtest đang chạy',
        canceling: 'Đang hủy backtest'
      },
      fields: {
        status: 'Trạng thái',
        error: 'Lỗi',
        maxDrawdown: '最大回撤',
        sharpe: '夏普比率'
      },
      metrics: {
        totalReturn: 'Tổng lợi nhuận',
        annualReturn: 'Lợi nhuận hàng năm',
        maxDrawdown: 'Sụt giảm tối đa',
        sharpe: 'Tỷ lệ Sharpe',
        winRate: 'Tỷ lệ thắng',
        totalTrades: 'Tổng giao dịch',
        equityCurvePoints: 'Điểm đường cong vốn'
      },
      trades: {
        title: 'Chi tiết lệnh',
        empty: 'Không có giao dịch nào được ghi lại',
        loadFailed: 'Tải chi tiết lệnh thất bại',
        ticket: 'Vé',
        side: 'Chiều',
        sideBuy: 'Mua',
        sideSell: 'Bán',
        volume: 'Khối lượng',
        openTime: 'Thời gian mở',
        openPrice: 'Giá mở',
        closeTime: 'Thời gian đóng',
        closePrice: 'Giá đóng',
        pnl: 'Lãi/Lỗ',
        commission: 'Hoa hồng',
        reason: 'Lý do đóng',
        reasons: {
          signal: 'Tín hiệu',
          sl: 'Cắt lỗ',
          tp: 'Chốt lời',
          margin_call: 'Gọi ký quỹ',
          expired: 'Hết hạn',
          end_of_test: 'Kết thúc kiểm tra'
        },
        summary: '{{count}} giao dịch · {{wins}} thắng / {{losses}} thua · Lãi/Lỗ ròng {{pnl}}'
      }
    },
    asset: {
      title: 'Tài sản chiến lược',
      subtitle: 'Xuất bản tài sản, trạng thái xét duyệt và nhân bản được duy trì bởi máy chủ. Kết quả nhân bản là các mẫu người dùng độc lập.',
      submitAsset: 'Gửi tài sản',
      assetList: 'Danh sách tài sản',
      name: 'Tên',
      visibility: 'Hiển thị',
      reviewStatus: 'Trạng thái xét duyệt',
      cloneCount: 'Bản sao',
      version: 'Phiên bản',
      description: 'Mô tả',
      actions: 'Hành động',
      cloneAsDraft: 'Nhân bản làm bản nháp',
      sourceTemplate: 'Mẫu nguồn',
      assetName: 'Tên tài sản',
      submit: 'Gửi',
      messages: {
        loadFailed: 'Tải tài sản chiến lược thất bại',
        submitSuccess: 'Đã gửi tài sản chiến lược',
        submitFailed: 'Gửi tài sản chiến lược thất bại',
        cloneSuccess: 'Đã nhân bản làm mẫu: {{templateId}}',
        cloneFailed: 'Nhân bản tài sản chiến lược thất bại'
      },
      validation: {
        selectTemplate: 'Vui lòng chọn mẫu nguồn',
        enterName: 'Vui lòng nhập tên tài sản'
      },
      empty: '暂无策略资产'
    },
    gen: {
      title: 'Tạo chiến lược',
      send: 'Tạo chiến lược',
      regenerate: 'Tạo lại',
      reset: 'Bắt đầu lại',
      template: 'Mẫu',
      generating: 'Đang tạo...',
      validating: 'Kiểm tra tuân thủ',
      backtestStarted: 'Backtest đã bắt đầu',
      done: 'Hoàn thành',
      backtestMsg: 'Đã tạo tác vụ backtest',
      clarifyTitle: 'Một số chi tiết cần xác nhận:',
      useDefaults: 'Tiếp tục với mặc định',
      placeholder: 'Mô tả chiến lược giao dịch bạn muốn tạo, ví dụ: "Tạo chiến lược mean-reversion Bollinger Band cho EURUSD trên 1H"',
      chat: {
        generate: 'Tạo',
        revise: 'Chỉnh sửa',
        repair: 'Sửa chữa',
        discuss: 'Thảo luận'
      },
      feedback: {
        heading: 'Kết quả backtest',
        placeholder: 'Cung cấp phản hồi để lặp lại (ví dụ: "Quá hung hăng", "Thêm cắt lỗ")'
      },
      metrics: {
        sharpe: 'Sharpe',
        maxDrawdown: 'Max DD',
        winRate: 'Thắng',
        trades: 'Giao dịch',
        return: 'Lợi nhuận'
      }
    },
    marketRegime: {
      title: 'Phát hiện chế độ thị trường',
      subtitle: 'Máy chủ tính toán các đặc trưng xu hướng, biến động và hiệu quả từ nến K. Giao diện chỉ hiển thị kết quả.',
      ruleVersionAlert: 'Hiện đang sử dụng mô hình phát hiện dựa trên quy tắc rule-v1. Nguồn nến K có thẩm quyền vẫn là dịch vụ Thị trường/Nến K của máy chủ.',
      detectSuccess: 'Phát hiện chế độ thị trường đã hoàn thành',
      detectFailed: 'Phát hiện chế độ thị trường thất bại',
      form: {
        title: 'Tham số phát hiện',
        accountId: 'ID tài khoản',
        accountIdRequired: 'Yêu cầu ID tài khoản',
        accountIdPlaceholder: 'UUID tài khoản MT',
        symbol: 'Mã chứng khoán',
        symbolRequired: 'Yêu cầu mã chứng khoán',
        symbolPlaceholder: 'EURUSD',
        timeframe: 'Khung thời gian',
        klineCount: 'Số lượng nến K',
        submit: 'Bắt đầu phát hiện'
      },
      result: {
        title: 'Kết quả phát hiện',
        status: 'Trạng thái',
        confidence: 'Độ tin cậy',
        modelVersion: 'Phiên bản mô hình',
        strategyFamilies: 'Họ chiến lược',
        features: 'Đặc trưng',
        recordId: 'ID bản ghi'
      }
    },
    experiment: {
      title: 'Thử nghiệm chiến lược',
      subtitle: 'Thử nghiệm tham số, chấm điểm ứng viên và tạo bản nháp do máy chủ xử lý. Giao diện chỉ gửi và hiển thị.',
      ruleVersionAlert: 'Vòng lặp tối thiểu hiện tại: thử nghiệm tham số xác định. Ứng viên chỉ tạo bản nháp và sẽ không tự động xuất bản, lập lịch hoặc giao dịch.',
      jobEventStream: 'Luồng sự kiện công việc',
      noEvents: 'Không có sự kiện',
      selectJobToView: 'Chọn một thử nghiệm có Công việc để xem sự kiện.',
      submitForm: {
        title: 'Gửi thử nghiệm',
        baseTemplate: 'Mẫu chiến lược cơ sở',
        baseTemplateRequired: 'Vui lòng chọn mẫu chiến lược cơ sở',
        baseTemplatePlaceholder: 'Chọn mẫu',
        parameterSpace: 'Không gian tham số JSON',
        parameterSpaceRequired: 'Vui lòng nhập JSON không gian tham số',
        searchMethod: 'Phương pháp tìm kiếm',
        maxCandidates: 'Số ứng viên tối đa',
        objective: 'Mục tiêu',
        submit: 'Gửi thử nghiệm'
      },
      list: {
        title: 'Danh sách thử nghiệm',
        column: {
          status: 'Trạng thái',
          searchMethod: 'Phương pháp tìm kiếm',
          maxCandidates: 'Số ứng viên tối đa',
          objective: 'Mục tiêu',
          actions: 'Hành động',
          viewCandidates: 'Xem ứng viên'
        }
      },
      candidates: {
        title: 'Ứng viên',
        titleWithId: 'Ứng viên: {{id}}',
        column: {
          rank: 'Hạng',
          grade: 'Điểm',
          score: 'Điểm số',
          parameters: 'Tham số',
          summary: 'Tóm tắt',
          recommendation: 'Đề xuất',
          actions: 'Hành động',
          viewCandidates: 'Xem ứng viên',
          generateDraft: 'Tạo bản nháp'
        }
      },
      messages: {
        loadTemplatesFailed: 'Tải mẫu chiến lược thất bại',
        loadExperimentsFailed: 'Tải danh sách thử nghiệm thất bại',
        loadCandidatesFailed: 'Tải ứng viên thất bại',
        subscribeJobFailed: 'Đăng ký sự kiện Công việc thử nghiệm thất bại',
        candidatesGenerated: 'Đã tạo ứng viên thử nghiệm chiến lược',
        submitFailed: 'Gửi thử nghiệm thất bại. Vui lòng kiểm tra không gian tham số có phải JSON hợp lệ không.',
        draftGenerated: 'Đã tạo bản nháp mẫu: {{templateId}}',
        promoteFailed: 'Thăng cấp ứng viên lên bản nháp thất bại'
      }
    },
    workspace: {
      title: 'Không gian làm việc chiến lược',
      account: 'Tài khoản',
      accountPlaceholder: 'ID tài khoản',
      chartWindow: 'Biểu đồ',
      hideCode: 'Ẩn mã',
      showCode: 'Hiện mã',
      quickTrade: 'Giao dịch nhanh',
      quickTradeHint: 'Chọn mã chứng khoán trước',
      tradePanelPlaceholder: 'Bảng giao dịch — sắp ra mắt',
      selectSymbolHint: 'Chọn tài khoản giao dịch và mã chứng khoán để xem biểu đồ',
      noAccounts: 'Không có tài khoản khả dụng',
      selectSymbol: 'Mã chứng khoán',
      code: 'Mã chiến lược',
      codePlaceholder: `# Mã chiến lược Python...
def run(context):
    return {"signal": "hold"}`,
      validate: 'Xác thực',
      validatePass: 'Xác thực thành công',
      validateFailed: 'Xác thực thất bại',
      validateBeforeSave: 'Vui lòng xác thực mã trước khi lưu',
      runBacktest: 'Chạy Backtest',
      save: 'Lưu',
      copy: 'Sao chép',
      copySuccess: 'Đã sao chép',
      copyFailed: 'Sao chép thất bại',
      saveSuccess: 'Đã lưu',
      chart: 'Nến K',
      backtest: 'Kiểm thử lùi',
      backtestRunning: 'Backtest đang chạy...',
      backtestCompleted: 'Hoàn thành',
      backtestError: 'Backtest thất bại',
      backtestEmpty: 'Chạy backtest để xem kết quả',
      backtestTab: 'Kết quả backtest',
      tuningTab: 'Tinh chỉnh thông minh',
      execAssumptions: 'Giả định thực thi',
      execAssumptionsFields: {
        mode: 'Chế độ',
        timing: 'Thời điểm',
        fillRule: 'Quy tắc khớp lệnh',
        direction: 'Hướng',
        commission: 'Hoa hồng',
        slippage: 'Trượt giá',
        leverage: 'Đòn bẩy',
        mtfFallback: 'Dự phòng MTF'
      },
      aiAssist: 'Trợ lý AI',
      ai: 'AI',
      runtimeMode: 'Môi trường chạy',
      saveFailed: 'Lưu thất bại',
      autoFix: {
        fixing: 'Đang sửa...',
        button: 'Tự động sửa',
        askAI: 'Hỏi AI',
        dismiss: 'Bỏ qua',
        passed: 'Tự động sửa đã vượt qua trong {{iterations}} lần lặp{{plural}}',
        failed: 'Tự động sửa: còn {{remaining}} vấn đề sau {{iterations}} lần lặp',
        fixed: 'Đã sửa ({{count}})',
        remaining: 'Còn lại ({{count}})',
        newRegression: 'Hồi quy mới ({{count}})',
        lineInfo: 'dòng {{line}}'
      },
      template: {
        title: 'Mẫu',
        selectPlaceholder: 'Chọn mẫu...',
        load: 'Tải',
        saveAs: 'Lưu thành mới',
        loaded: 'Đã tải'
      },
      watchlist: 'Danh sách theo dõi',
      selectAccount: 'Chọn tài khoản',
      openPositions: 'Vị thế mở ({{count}})',
      noOpenPositions: 'Không có vị thế mở cho tài khoản này',
      chartError: 'Lỗi biểu đồ — hãy thử làm mới',
      smartTuning: 'Tinh chỉnh thông minh',
      quickTradeSection: {
        selectSymbol: 'Vui lòng chọn mã giao dịch trước',
        validVolume: 'Nhập khối lượng hợp lệ',
        priceRequired: 'Cần nhập giá cho lệnh Limit/Stop',
        orderPlaced: 'Lệnh {{side}} đã được đặt',
        orderFailed: 'Đặt lệnh thất bại',
        amountLots: 'Khối lượng (lô)',
        marginMode: 'Chế độ ký quỹ',
        cross: 'Cross',
        isolated: 'Cô lập',
        mt4CrossOnly: 'MT4 chỉ hỗ trợ ký quỹ Cross'
      },
      chartTools: {
        streamActive: 'Luồng dữ liệu nến trực tiếp đang hoạt động',
        streamUnavailable: 'Luồng dữ liệu không khả dụng',
        hide: 'Ẩn',
        show: 'Hiện',
        settings: 'Cài đặt',
        remove: 'Xóa',
        clearDrawings: 'Xóa tất cả hình vẽ',
        candle: 'Nến',
        ohlc: 'OHLC',
        area: 'Vùng',
        live: 'LIVE',
        error: 'ERROR',
        static: 'STATIC'
      },
      backtestRunIdLabel: 'Chọn lần chạy backtest...',
      investorReadOnly: 'Nhà đầu tư (Chỉ xem)',
      masterTrading: 'Chủ (Giao dịch)',
      riskControls: 'Quy tắc rủi ro từ mã',
      jumpToCode: 'Đi tới mã',
      runningStatus: 'Đang chạy...',
      completedStatus: 'Hoàn thành',
      backtestResultsLabel: 'Kết quả backtest',
      gateTab: 'Gate'
    },
    codeQuality: {
      category: {
        FUTURE_DATA_LEAK: 'Rò rỉ dữ liệu tương lai',
        MISSING_PARAM: 'Thiếu tham số',
        UNREAD_PARAM: 'Tham số chưa đọc',
        NDARRAY_PANDAS_MISUSE: 'Sử dụng sai ndarray/pandas',
        NO_STOP_AND_TAKE_PROFIT: 'Thiếu cắt lỗ/chốt lời',
        NO_ENTRY_PCT: 'Thiếu % vào lệnh'
      }
    },
    backtestParams: {
      title: 'Kiểm thử lùi',
      currentDraft: 'Bản nháp hiện tại',
      dateRange: 'Khoảng thời gian',
      execution: 'Thực thi',
      capital: 'Vốn',
      leverage: 'Đòn bẩy',
      commission: 'Hoa hồng',
      slippage: 'Trượt giá',
      trade: 'Giao dịch',
      direction: 'Hướng',
      long: 'Mua',
      short: 'Bán',
      both: 'Cả hai',
      strictMode: 'Chế độ nghiêm ngặt',
      strictModeOn: 'BẬT',
      strictModeOff: 'TẮT',
      strictModeOnDesc: 'Mở nến tiếp theo. Tiêu chuẩn, bảo thủ.',
      strictModeOffDesc: 'Đóng cùng nến + MTF 1m. Độ chính xác cao hơn.',
      strictModeOnTooltip: 'BẬT: tín hiệu xác nhận khi đóng nến, thực thi khi mở nến tiếp theo',
      strictModeOffTooltip: 'TẮT: thực thi đóng cùng nến với độ phân giải phụ 1m',
      vectorizedMode: 'Vector hóa',
      eventDrivenMode: 'Run(context)',
      runtimeMode: 'Môi trường chạy',
      history: 'Lịch sử backtest',
      run: 'Chạy',
      settingsSave: 'Lưu làm mặc định của tôi',
      settingsLoad: 'Tải mặc định của tôi',
      settingsReset: 'Đặt lại về mặc định',
      defaultsSaved: 'Đã lưu mặc định',
      defaultsLoaded: 'Đã tải mặc định',
      defaultsReset: 'Đặt lại về mặc định gốc',
      presets: {
        liveAligned: 'Căn chỉnh thực tế',
        exploration: 'Khám phá'
      },
      enterCodeAndSymbol: 'Vui lòng nhập mã chiến lược và chọn mã',
      backtestFailed: 'Backtest thất bại'
    },
    tuning: {
      optimizerMethod: 'Phương pháp tối ưu hóa',
      parameterDimensions: 'Số chiều tham số',
      enabledCombinations: '{{enabled}} đã bật · {{combos}} tổ hợp',
      hide: 'Ẩn',
      preview: 'Xem trước',
      previewTitle: 'Xem trước ({{shown}}/{{total}})',
      truncated: 'ĐÃ CẮT',
      results: 'Kết quả ({{count}})',
      rank: '#',
      grade: 'Điểm',
      score: 'Điểm số',
      parameters: 'Tham số',
      summary: 'Tóm tắt',
      oosScore: 'Điểm OOS',
      degradation: 'Suy giảm',
      overfit: 'Quá khớp',
      overfitWarning: 'CẢNH BÁO QUÁ KHỚP',
      apply: 'Áp dụng',
      run: 'Chạy ({{count}})',
      tuning: 'Đang tinh chỉnh...',
      requiresAI: 'Yêu cầu đã cấu hình nhà cung cấp AI',
      switchToDE: 'Chuyển sang DE',
      waiting: 'Đang chờ thử nghiệm... (SSE tự động làm mới)',
      gridWarning: 'Tìm kiếm lưới sẽ kiểm tra <b>{{count}}</b> tổ hợp (ngân sách: 48). Cân nhắc chuyển sang <b>Tiến hóa vi phân</b> để xử lý không gian tham số lớn hiệu quả.',
      oosFootnote: 'Xác thực OOS chạy trên top-5 ứng viên (theo điểm IS). Xanh <20%, cam 20-40%, đỏ >40%.',
      optimizer: {
        grid: 'Tìm kiếm lưới',
        random: 'Tìm kiếm ngẫu nhiên',
        de: 'Tiến hóa vi phân',
        tpe: 'TPE (KDE)',
        ags: 'Gaussian ủ nhiệt',
        ai: 'Trình tối ưu AI',
        gridDesc: 'Tích Descartes đầy đủ. Tốt nhất cho <=3 tham số.',
        randomDesc: 'Lấy mẫu ngẫu nhiên đồng nhất. Tốt cho khám phá.',
        deDesc: 'Đột biến rand/1/bin. Hội tụ nhanh trên bề mặt trơn.',
        tpeDesc: 'Ước lượng Parzen có cấu trúc cây. Mô hình KDE phân phối tốt/xấu.',
        agsDesc: 'Nhiễu Gaussian với ủ nhiệt sigma. Giải pháp nhẹ hơn TPE.',
        aiDesc: 'Đề xuất đa vòng LLM. Học từ kết quả trước qua 3 vòng.'
      },
      started: 'Đã bắt đầu tinh chỉnh thông minh'
    },
    paper: {
      title: '📊 Giao dịch giấy',
      createAccount: 'Tạo tài khoản giấy',
      accountName: 'Tên tài khoản',
      create: 'Tạo',
      noAccounts: 'Chưa có tài khoản giấy. Tạo một tài khoản để bắt đầu giao dịch mô phỏng.',
      running: 'Đang chạy {{symbol}} {{timeframe}}',
      start: 'Bắt đầu',
      stop: 'Dừng',
      watch: 'Theo dõi',
      paper: 'Giấy',
      startStrategy: 'Bắt đầu chiến lược giấy',
      symbol: 'Mã',
      timeframe: 'Khung thời gian',
      strategyCode: 'Mã chiến lược (Python)',
      messages: {
        enterName: 'Nhập tên',
        created: 'Đã tạo tài khoản giấy',
        createFailed: 'Tạo thất bại',
        pasteCode: 'Dán mã chiến lược của bạn',
        strategyStarted: 'Chiến lược giấy đã bắt đầu',
        startFailed: 'Bắt đầu thất bại',
        strategyStopped: 'Chiến lược giấy đã dừng',
        stopFailed: 'Dừng thất bại'
      }
    },
    aiChat: {
      title: 'Trò chuyện AI',
      you: 'Bạn',
      ai: 'AI',
      revise: 'sửa đổi',
      feedback: '🔄 phản hồi',
      streaming: 'đang tạo',
      analyzing: 'đang phân tích',
      reset: 'đặt lại',
      applyCode: 'Áp dụng mã',
      dismiss: 'Bỏ qua',
      reviewCode: 'AI đã tạo mã — hãy xem lại cuộc trò chuyện ở trên trước khi áp dụng.'
    },
    assetAnalysis: {
      title: 'Phân tích tài sản AI',
      subtitle: 'Triển vọng xu hướng đa khung thời gian, phát hiện mức hỗ trợ/kháng cự, phân loại biến động và đề xuất chiến lược AI',
      symbolPlaceholder: 'Nhập mã (ví dụ: EURUSD, XAUUSD, BTCUSD)',
      analyze: 'Phân tích',
      fetchingData: 'Đang lấy dữ liệu thị trường...',
      phase: 'Giai đoạn: {{phase}}',
      mtfOutlook: 'Triển vọng đa khung',
      srLevels: 'Mức Hỗ trợ / Kháng cự',
      volatility: 'Biến động',
      state: 'Trạng thái',
      atrPct: 'ATR %',
      aiRecommendation: 'Đề xuất chiến lược AI',
      aiUnavailable: 'Đề xuất AI không khả dụng. Vui lòng cấu hình nhà cung cấp AI trong Cài đặt.',
      configureAI: 'Cấu hình nhà cung cấp AI',
      noLevels: 'Không phát hiện mức đáng kể',
      noResults: 'Không có kết quả phân tích. Thử mã khác.',
      volLow: 'Biến động thấp — cân nhắc chiến lược breakout hoặc mean-reversion với stop chặt.',
      volNormal: 'Biến động bình thường — phù hợp với hầu hết các loại chiến lược.',
      volHigh: 'Biến động cao — khuyến nghị stop rộng hơn; chiến lược theo xu hướng và breakout có lợi thế.',
      volExtreme: 'Biến động cực đoan — giảm đáng kể kích thước vị thế; cần stop rộng.'
    },
    ai: {
      checkSettings: '检查AI设置',
      refreshFailed: '刷新失败',
      settings: 'AI设置'
    },
    backtest: {
      annualReturn: 'Lợi Nhuận Hàng Năm',
      equityCurve: '权益曲线',
      maxDrawdown: 'Sụt Giảm Tối Đa',
      sharpe: 'Sharpe',
      totalReturn: 'Tổng Lợi Nhuận',
      totalTrades: 'Tổng Giao Dịch',
      winRate: 'Tỷ Lệ Thắng',
      tradeLog: '交易日志',
      tradeTime: '时间',
      tradeSide: '方向',
      tradePrice: '价格',
      tradeVolume: '数量'
    },
    chartTools: {
      clearDrawings: '清除所有绘图',
      hide: 'Ẩn',
      show: 'Hiện',
      settings: 'Cài Đặt',
      remove: 'Xóa'
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
      orderPlaced: 'Đã đặt lệnh',
      orderFailed: 'Đặt lệnh thất bại'
    },
    library: {
      title: 'Thư viện chiến lược',
      myStrategies: 'Chiến lược của tôi',
      create: 'Tạo mới',
      filterAll: 'Tất cả',
      filterMine: 'Của tôi',
      filterSystem: 'Có sẵn',
      searchPlaceholder: 'Tìm kiếm chiến lược...',
      empty: 'Chưa có chiến lược',
      published: 'Đã xuất bản',
      draft: 'Bản nháp',
      unpublish: 'Gỡ xuống',
      unpublishShort: 'Gỡ',
      publish: 'Xuất bản lên Market',
      publishSuccess: 'Đã xuất bản',
      unpublishSuccess: 'Đã gỡ xuống',
      publishStatus: 'Trạng thái',
      selectHint: 'Chọn chiến lược từ danh sách để xem chi tiết',
      overview: 'Tổng quan',
      schedules: 'Chạy',
      backtestHistory: 'Lịch sử backtest',
      scheduleCount: '{{count}} đang chạy',
      scheduleRunningCount: '{{count}} đang chạy',
      noSchedules: 'Chưa chạy',
      openInWorkspace: 'Mở trong Workspace',
      createSchedule: 'Tạo lịch chạy',
      saveAsMine: 'Lưu thành của tôi',
      saveAsMineSuccess: 'Đã lưu vào Chiến lược của tôi',
      myCopy: 'Bản sao của tôi',
      codePreview: 'Xem trước code',
      viewCode: 'Xem code chiến lược',
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
