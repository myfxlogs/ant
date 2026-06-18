const accounts = {
  accounts: {
    title: 'Tài khoản',
    subtitle: 'Quản lý tài khoản MT4/MT5',
    bindNew: 'Liên kết tài khoản mới',
    bind: {
      title: 'Liên kết tài khoản MT',
      errorModal: {
        title: 'Binding failed'
      },
      step1: {
        title: 'Chọn nền tảng và nhà môi giới',
        subtitle: 'Select your trading platform and search for your broker'
      },
      step2: {
        title: 'Nhập thông tin tài khoản',
        subtitle: 'Enter your trading account and password'
      },
      step3: {
        title: 'Xác nhận liên kết',
        subtitle: 'Verify credentials and confirm to complete'
      },
      fields: {
        platform: 'Nền tảng',
        brokerName: 'Tên môi giới',
        company: 'Công ty',
        server: 'Máy chủ',
        tradingAccount: 'Tài khoản giao dịch',
        password: 'Mật khẩu'
      },
      placeholders: {
        brokerName: 'Nhập tên môi giới, ví dụ: XM, IC Markets',
        company: 'Chọn công ty môi giới',
        server: 'Chọn máy chủ',
        tradingAccount: 'Nhập tài khoản giao dịch',
        password: 'Enter password'
      },
      labels: {
        serverCount: '{{count}} servers'
      },
      actions: {
        search: 'Tìm kiếm',
        verifyAccount: 'Xác minh tài khoản',
        confirmBind: 'Xác nhận',
        retryVerify: 'Retry'
      },
      passwordHint: 'Mật khẩu được truyền qua HTTPS. Backend lưu dưới dạng băm Argon2id không thể đảo ngược.',
      summary: {
        broker: 'Nhà môi giới',
        server: 'Máy chủ',
        platform: 'Nền tảng',
        tradingAccount: 'Tài khoản giao dịch',
        password: 'Mật khẩu',
        verified: 'Tài khoản đã xác minh',
        balance: 'Số dư',
        equity: 'Vốn',
        margin: 'Ký quỹ',
        freeMargin: 'Ký quỹ khả dụng',
        leverage: 'Đòn bẩy',
        currency: 'Currency'
      },
      messages: {
        enterBrokerName: 'Vui lòng nhập tên môi giới',
        foundBrokers: 'Tìm thấy {{count}} nhà môi giới',
        noBrokersFound: 'Không tìm thấy nhà môi giới phù hợp. Vui lòng kiểm tra tên.',
        searchFailed: 'Tìm kiếm thất bại. Vui lòng thử lại sau.',
        selectServer: 'Vui lòng chọn máy chủ',
        enterTradingAccount: 'Vui lòng nhập tài khoản giao dịch',
        enterPassword: 'Vui lòng nhập mật khẩu',
        noAccessHosts: 'Không có máy chủ khả dụng',
        verifyFailed: 'Xác minh tài khoản thất bại',
        bindSuccess: 'Liên kết tài khoản thành công',
        bindFailed: 'Liên kết tài khoản thất bại',
        loginDigitsOnly: 'Trading account must contain only digits'
      },
      errors: {
        brokerUnavailable: 'Lỗi máy chủ hoặc mật khẩu không đúng',
        invalidCredentials: 'Không tìm thấy tài khoản hoặc mật khẩu không đúng',
        connectionFailed: 'Không thể kết nối đến máy chủ môi giới, vui lòng kiểm tra mạng',
        timeout: 'Connection timed out, please try again later'
      }
    },
    empty: {
      title: 'Chưa có tài khoản',
      subtitle: 'Click the button below to bind your MT4/MT5 trading account'
    },
    legend: {
      title: 'Chú giải:',
      connected: 'Đã kết nối',
      connecting: 'Đang kết nối',
      disconnectedOrError: 'Mất kết nối/Lỗi',
      disabled: 'Đã tắt'
    },
    messages: {
      disabledSuccess: 'Vô hiệu hóa tài khoản thành công',
      connectingMtServer: 'Đang kết nối đến máy chủ MT',
      enabledSuccess: 'Kích hoạt tài khoản thành công',
      fetchListFailed: 'Không thể tải danh sách tài khoản',
      fetchAccountFailed: 'Không thể tải thông tin tài khoản',
      createdSuccess: 'Tạo tài khoản thành công',
      createFailed: 'Tạo tài khoản thất bại',
      connectSuccess: 'Kết nối thành công',
      connectFailed: 'Kết nối thất bại',
      disconnectFailed: 'Ngắt kết nối thất bại',
      disableFailed: 'Vô hiệu hóa tài khoản thất bại',
      deleted: 'Đã xóa',
      deleteFailed: 'Xóa thất bại',
      enableFailed: 'Failed to enable account'
    },
    analytics: {
      monthlyAnalysis: {
        title: 'Phan tich theo thang',
        chartMainTitle: 'Loi nhuan theo thang ({{metric}})',
        metrics: {
          change: 'Thay doi',
          profit: 'Lợi nhuận',
          lots: 'Lot',
          pips: 'Pips'
        },
        focusedValue: '{{period}} · {{metric}}: {{value}}',
        bonus: {
          chartRiskTitle: 'Bonus: Ty so loi nhuan (profit factor) theo symbol — {{month}}.',
          chartPopularTitle: `{{month}}'s currency popularity.`,
          chartHoldingTitle: `{{month}}'s average holding time.`,
          legendBulls: 'Mua',
          legendShortTerm: 'Ban',
          sliceOther: 'Khac',
          emptyCharts: 'Khong co lenh trong thang',
          popularityShare: 'Lot volume share'
        }
      },
      monthlyDetail: {
        metricsTitle: 'Chỉ số tháng',
        symbolPnLTitle: 'Lợi nhuận theo cặp',
        holdingTitle: 'Thời gian giữ lệnh',
        riskRewardTitle: 'Tỷ lệ R:R',
        popularityTitle: 'Mức độ phổ biến',
        long: 'Mua',
        short: 'Bán',
        fields: {
          netReturn: 'Lợi nhuận ròng',
          totalTrades: 'Tổng lệnh',
          winRate: 'Tỷ lệ thắng',
          profitFactor: 'Hệ số lợi nhuận',
          bestTrade: 'Lệnh tốt nhất',
          worstTrade: 'Lệnh tệ nhất',
          averageHours: 'TB',
          medianHours: 'Trung vị',
          maxHours: 'Tối đa',
          minHours: 'Tối thiểu',
        },
      },
      chartType: {
        equity: 'Vốn',
        balance: 'Số dư',
        profit: 'Lợi nhuận'
      },
      chartPeriod: {
        day: 'Hôm nay',
        week: 'Tuần này',
        month: 'Tháng này',
        year: 'Năm nay',
        all: 'All'
      },
      chartSeries: {
        equity: 'Vốn',
        balance: 'Số dư',
        profit: 'Lợi nhuận',
        tradeCount: 'Giao dịch'
      },
      empty: {
        equityCurve: 'Không có dữ liệu đường vốn',
        monthlyProfit: 'Không có dữ liệu lãi/lỗ theo tháng',
        symbolDistribution: 'Không có dữ liệu phân bổ mã',
        dailyPnL: 'Không có dữ liệu lãi/lỗ theo ngày',
        hourly: 'No time-of-day analysis data'
      },
      monthlyProfitTitle: 'Lãi/lỗ theo tháng',
      advancedStatsTitle: 'Thống kê nâng cao',
      symbolDistributionTitle: 'Phân bổ mã',
      dailyPnLTitle: '📅 Lãi/lỗ theo ngày',
      hourlyTitle: '⏰ Phân tích theo giờ',
      advancedTabs: {
        hourly: 'Theo giờ',
        daily: 'Daily'
      },
      timeDetail: {
        lots: 'Lot',
        trades: 'Giao dịch',
        profitAmount: 'Lợi nhuận',
        balance: 'Số dư',
        profitFactor: 'Hệ số lợi nhuận',
        maxFloatingLossAmount: 'Lỗ thả nổi tối đa',
        maxFloatingLossRatio: 'Tỷ lệ lỗ thả nổi tối đa',
        maxFloatingProfitAmount: 'Lợi nhuận thả nổi tối đa',
        maxFloatingProfitRatio: 'Max floating profit ratio'
      },
      stats: {
        winRate: 'Tỷ lệ thắng',
        profitFactor: 'Hệ số lợi nhuận',
        maxDrawdown: 'Drawdown tối đa',
        totalTrades: 'Tổng số lệnh',
        avgProfit: 'Lãi trung bình',
        avgLoss: 'Lỗ trung bình',
        avgHolding: 'Thời gian giữ TB',
        consecutiveWinsLosses: 'Chuỗi thắng/thua',
        sharpe: 'Sharpe',
        sortino: 'Sortino',
        calmar: 'Calmar',
        largestWin: 'Lãi lớn nhất',
        largestLoss: 'Lỗ lớn nhất',
        avgDailyReturn: 'Lợi nhuận ngày TB',
        volatility: 'Biến động',
        netProfit: 'Lợi nhuận ròng',
        totalDeposit: 'Nạp',
        totalWithdrawal: 'Rút',
        netDeposit: 'Net deposit'
      }
    },
    card: {
      status: {
        disabled: 'Đã tắt',
        connected: 'Đã kết nối',
        connecting: 'Đang kết nối',
        disconnected: 'Mất kết nối',
        error: 'Error'
      },
      fields: {
        balance: 'Số dư',
        equity: 'Vốn',
        broker: 'Nhà môi giới',
        server: 'Máy chủ'
      },
      actions: {
        positions: 'Vị thế',
        orders: 'Lệnh',
        details: 'Details'
      },
      deleteConfirm: {
        title: 'Xóa tài khoản này?',
        content: 'This action cannot be undone'
      }
    },
    disabled: {
      title: 'Tài khoản đã tắt',
      table: {
        account: 'Tài khoản',
        type: 'Loại',
        broker: 'Nhà môi giới',
        balance: 'Số dư',
        equity: 'Vốn',
        actions: 'Actions'
      },
      confirmDelete: {
        title: 'Xóa tài khoản này?',
        content: 'This action cannot be undone'
      },
      mobile: {
        balanceLabel: 'Số dư: ',
        equityLabel: 'Equity: '
      }
    },
    tradeTabs: {
      positionsWithCount: 'Vị thế ({{count}})',
      pendingWithCount: 'Lệnh chờ ({{count}})',
      historyWithCount: 'Lịch sử ({{count}})',
      emptyPositions: 'Chưa có vị thế',
      emptyHistory: 'Chưa có lịch sử lệnh',
      syncHistory: 'Đồng bộ lịch sử',
      table: {
        orderId: 'Mã lệnh',
        symbol: 'Mã',
        side: 'Hướng',
        type: 'Loại',
        volume: 'Khối lượng',
        openPrice: 'Giá mở',
        currentPrice: 'Giá hiện tại',
        pendingPrice: 'Giá đặt',
        closePrice: 'Giá đóng',
        profit: 'Lợi nhuận',
        openTime: 'Thời gian mở',
        pendingTime: 'Thời gian đặt',
        closeTime: 'Close time'
      },
      pagination: {
        total: '{{total}} total'
      }
    },
    edit: {
      title: 'Chỉnh sửa tài khoản',
      fields: {
        tradingAccount: 'Tài khoản giao dịch',
        server: 'Máy chủ',
        password: 'Mật khẩu mới',
        oldPassword: 'Current password'
      },
      placeholders: {
        newPassword: 'Nhập mật khẩu mới',
        oldPassword: 'Enter current password'
      },
      messages: {
        enterPassword: 'Vui lòng nhập mật khẩu mới',
        enterOldPassword: 'Vui lòng nhập mật khẩu hiện tại',
        passwordVerifyFailed: 'Thay đổi mật khẩu thất bại',
        passwordSaved: 'Password saved'
      }
    },
    detail: {
      messages: {
        fetchAccountFailed: 'Không thể tải thông tin tài khoản. Vui lòng thử lại sau.',
        syncHistorySuccess: 'Đồng bộ lịch sử lệnh thành công',
        syncHistoryFailed: 'Failed to sync order history. Please ensure the account is connected to the MT server.'
      },
      orderTypes: {
        buyLimit: 'Mua giới hạn',
        sellLimit: 'Bán giới hạn',
        buyStop: 'Mua dừng',
        sellStop: 'Sell stop'
      },
      balanceRecord: {
        deposit: '💰 Nạp tiền',
        withdraw: '💸 Rút tiền',
        depositIconText: '💰 Nạp tiền',
        withdrawIconText: '💸 Rút tiền'
      },
      syncHistory: {
        title: 'Đồng bộ lịch sử lệnh',
        content: 'Đồng bộ lịch sử lệnh 1 năm gần nhất từ máy chủ MT? Việc này có thể mất một chút thời gian.',
        ok: 'Sync'
      },
      actions: {
        enableAccount: 'Bật tài khoản',
        disableAccount: 'Tắt tài khoản',
        deleteAccount: 'Xóa tài khoản',
        deleteConfirm: 'Xác minh & Xóa',
        deleteWarning: 'Hành động này không thể hoàn tác. Toàn bộ dữ liệu tài khoản (lịch sử giao dịch, phân tích, v.v.) sẽ bị xóa vĩnh viễn.',
        deletePasswordHint: 'Nhập mật khẩu giao dịch MT hoặc mật khẩu chỉ-đọc để xác minh:',
        deletePasswordPlaceholder: 'Mật khẩu giao dịch / chỉ-đọc MT',
        syncHistory: 'Sync history'
      },
      status: {
        disabled: 'Đã tắt',
        connected: 'Đã kết nối',
        connecting: 'Đang kết nối',
        disconnected: 'Mất kết nối',
        error: 'Error'
      },
      accountType: {
        real: 'Thực',
        demo: 'Demo'
      },
      mode: {
        investor: 'Chế độ nhà đầu tư',
        trader: 'Trader mode'
      },
      connected: 'Đã kết nối',
        lastConnected: '{{time}}',
        leverage: 'Đòn bẩy {{leverage}}x',
      cards: {
        balance: 'Số dư',
        equity: 'Vốn',
        floatingProfit: 'Lãi/lỗ thả nổi',
        marginUsed: 'Ký quỹ đã dùng',
        marginFree: 'Ký quỹ khả dụng',
        marginLevel: 'Tỷ lệ ký quỹ',
        credit: 'Credit'
      }
    },
    report: {
      title: 'Báo cáo giao dịch',
      titleShort: 'Báo cáo',
      generate: 'Tạo báo cáo',
      goToAISettings: 'Đi tới Cài đặt AI →',
      aiAnalysis: 'Phân tích AI',
      symbolPnL: 'Lãi/lỗ theo mã',
      direction: 'Phân tích hướng',
      directionLong: 'Mua',
      directionShort: 'Bán',
      tradeDistribution: 'Phân phối lãi/lỗ',
      drawdownOverlay: 'Đường vốn + Drawdown',
      drawdownEvents: 'Sự kiện drawdown',
      recovered: 'Đã phục hồi',
      winRateTrend: 'Xu hướng tỷ lệ thắng',
      periods: {
        week: 'Tuần này',
        month: 'Tháng này',
        quarter: 'Quý này',
        year: 'Năm nay',
      },
      sections: {
        summary: 'Tổng quan',
        findings: 'Phát hiện chính',
        recommendations: 'Khuyến nghị',
      },
    },
  }
} as const;

export default accounts;
