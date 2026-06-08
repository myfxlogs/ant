const strategy = {
  strategy: {
    validation: {
      passed: 'Xác minh code thành công',
      notPassed: 'Xác minh code không đạt',
      riskEval: {
        title: 'Risk Assessment',
        riskHigh: 'Risk level: high',
        riskUnreliable: 'Risk assessment: unreliable (isReliable=false)',
        riskLoading: 'Backend risk assessment is still calculating'
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
        code: 'Dán mã chiến lược Python...'
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
        outro: 'Vui lòng xuất toàn bộ mã đã chỉnh sửa (bọc bằng \`\`\`python) và giải thích thay đổi.',
        pythonFenceStart: '\`\`\`python',
        fenceEnd: '\`\`\`'
      }
    },
    schedules: {
      title: 'Lịch chạy chiến lược',
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
          timeframe: 'Timeframe',
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
          timeframe: 'Timeframe'
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
        orders: 'Lệnh'
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
      scheduleIdLabel: 'ID lịch chạy:'
    },
    templates: {
      title: 'Mẫu chiến lược',
      tabs: {
        system: 'Mẫu hệ thống',
        user: 'Mẫu tự tạo'
      },
      copySuffix: ' (Bản sao)',
      scheduleName: '{{symbol}} {{timeframe}} {{nowText}}',
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
        backtest: 'Backtest',
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
        loadingDefault: 'Loading default templates...',
        defaultHint: 'Default',
        emptyUser: 'No user templates yet. Click "Create Template" above to get started.'
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
          createAndEnable: 'Tạo & bật'
        },
        metrics: {
          totalReturn: 'Tổng lợi nhuận',
          annualReturn: 'Lợi nhuận năm',
          maxDrawdown: 'Sụt giảm tối đa',
          sharpe: 'Sharpe',
          winRate: 'Tỷ lệ thắng',
          totalTrades: 'Số lệnh'
        }
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
        title: 'Backtest',
        fields: {
          title: 'Tiêu đề',
          account: 'Tài khoản',
          symbol: 'Mã',
          timeframe: 'Timeframe',
          initialCapital: 'Vốn ban đầu',
          range: 'Khoảng thời gian',
          extraSymbols: 'Extra symbols (multi-select)'
        },
        placeholders: {
          account: 'Chọn tài khoản',
          symbol: 'Chọn mã',
          range: 'Chọn khoảng thời gian',
          extraSymbols: 'Optional, useful for pairs/rotation strategies'
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
        modalTitleWithName: 'Backtest: {{name}}',
        parameters: {
          title: 'Strategy Parameters'
        },
        tooltips: {
          extraSymbols: 'Additional symbols to fetch K-lines for (same account, same timeframe). Strategy can access them via context["closes_by_symbol"].'
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
        title: 'Báo cáo backtest',
        empty: 'Chưa có backtest',
        deleteConfirm: 'Xóa báo cáo backtest này?',
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
          timeframe: 'Timeframe',
          createdAt: 'Thời gian tạo',
          actions: 'Thao tác'
        },
        actions: {
          view: 'Xem',
          launchSchedule: 'Khởi chạy lịch',
          createSchedule: 'Tạo lịch'
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
      saveBlockedNotValidated: 'Vui lòng nhấn "Xác thực mã" trước. Lưu sẽ bị vô hiệu hóa cho đến khi xác thực thành công.'
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
    paramDescription: 'Mô tả',
    workspace: {
      title: 'Không gian chiến lược',
      account: 'Tài khoản',
      accountPlaceholder: 'ID tài khoản',
      chartWindow: 'Biểu đồ',
      hideCode: 'Ẩn mã',
      showCode: 'Hiện mã',
      quickTrade: 'Giao dịch',
      quickTradeHint: 'Chọn mã trước',
      tradePanelPlaceholder: 'Bảng giao dịch — sắp ra mắt',
      selectSymbolHint: 'Chọn tài khoản và mã để xem biểu đồ',
      noAccounts: 'Không có tài khoản khả dụng',
      selectSymbol: 'Mã',
      code: 'Mã chiến lược',
      codePlaceholder: `# Mã Python...
def run(context):
    return {"signal": "hold"}`,
      validate: 'Xác minh',
      validatePass: 'Xác minh thành công',
      validateFailed: 'Xác minh thất bại',
      validateBeforeSave: 'Vui lòng xác minh mã trước khi lưu',
      runBacktest: 'Chạy backtest',
      save: 'Lưu',
      copy: 'Sao chép',
      copySuccess: 'Đã sao chép',
      copyFailed: 'Sao chép thất bại',
      saveSuccess: 'Đã lưu',
      chart: 'K-line',
      backtest: 'Backtest',
      backtestRunning: 'Đang chạy...',
      backtestCompleted: 'Hoàn thành',
      backtestError: 'Thất bại',
      backtestEmpty: 'Chạy backtest để xem kết quả',
      aiAssist: 'Trợ lý AI',
      ai: 'AI',
      template: {
        title: 'Mẫu',
        selectPlaceholder: 'Chọn mẫu...',
        load: 'Tải',
        saveAs: 'Lưu thành mới',
        loaded: 'Đã tải'
      }
    }
  }
} as const;

export default strategy;
