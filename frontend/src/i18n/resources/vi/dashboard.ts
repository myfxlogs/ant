const dashboard = {
  dashboard: {
    welcome: 'Chào mừng trở lại, {{name}}',
    subtitle: 'Xem tổng quan tài khoản của bạn',
    bindAccount: 'Liên kết tài khoản',
    accountOverview: 'Tổng quan tài khoản',
    accountList: 'Danh sách tài khoản',
    viewAll: 'Xem tất cả',
    noAccounts: 'Chưa có tài khoản. Hãy nhấn “Liên kết tài khoản”.',
    stats: {
      totalEquity: 'Tổng vốn',
      connected: 'Đang kết nối',
      accountCount: 'Số tài khoản',
      totalProfit: 'Tổng lãi/lỗ thả nổi',
    },
    fields: {
      balance: 'Số dư',
      equity: 'Vốn',
      floating: 'Thả nổi',
    },
    accountStatus: {
      disabled: 'Đã tắt',
      connected: 'Đã kết nối',
      connecting: 'Đang kết nối',
      disconnected: 'Chưa kết nối',
    },
    quickActions: {
      title: 'Thao tác nhanh',
      trading: 'Giao dịch',
      market: 'Thị trường',
      accounts: 'Tài khoản',
      analytics: 'Phân tích',
      bindAccount: 'Liên kết',
      closePosition: 'Đóng lệnh',
    },
  },
} as const;

export default dashboard;
