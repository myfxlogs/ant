const dashboard = {
  dashboard: {
    welcome: 'Chào mừng trở lại, {{name}}',
    subtitle: 'Xem tổng quan tài khoản của bạn',
    templates: 'Mẫu',
    logs: 'Nhật ký',
    bindAccount: 'Liên kết tài khoản',
    accountOverview: 'Tổng quan tài khoản',
    accountList: 'Danh sách tài khoản',
    viewAll: 'Xem tất cả',
    streamLive: 'Kết nối trực tiếp',
    streamOffline: 'Ngoại tuyến',
    noAccounts: 'Chưa có tài khoản. Hãy nhấn “Liên kết tài khoản”.',
    stats: {
      totalBalance: 'Tổng số dư',
      totalEquity: 'Tổng vốn',
      connected: 'Đã kết nối',
      accountCount: 'Tài khoản',
      totalProfit: 'Total Floating P/L'
    },
    fields: {
      balance: 'Số dư',
      equity: 'Vốn',
      floating: 'Floating P/L'
    },
    accountStatus: {
      disabled: 'Đã tắt',
      connected: 'Đã kết nối',
      connecting: 'Đang kết nối',
      disconnected: 'Disconnected'
    },
    quickActions: {
      title: 'Thao tác nhanh',
      trading: 'Giao dịch',
      market: 'Thị trường',
      accounts: 'Tài khoản',
      analytics: 'Phân tích',
      library: 'Thư viện',
      templates: 'Mẫu',
      logs: 'Nhật ký',
      bindAccount: 'Liên kết',
      closePosition: 'Close'
    },
    defaultName: 'My Dashboard'
  }
} as const;

export default dashboard;
