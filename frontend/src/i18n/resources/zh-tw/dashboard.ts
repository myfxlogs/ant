const dashboard = {
  dashboard: {
    welcome: '歡迎回來, {{name}}',
    subtitle: '查看您的帳戶總覽',
    templates: '策略模板',
    logs: '日誌',
    bindAccount: '綁定帳戶',
    accountOverview: '帳戶總覽',
    accountList: '帳戶列表',
    viewAll: '查看全部',
    streamLive: '即時連接',
    streamOffline: '即時離線',
    noAccounts: '暫無帳戶，點擊右上角綁定',
    stats: {
      totalBalance: '總餘額',
      totalEquity: '總淨值',
      connected: '已連線',
      accountCount: '帳戶',
      totalProfit: 'Total Floating P/L'
    },
    fields: {
      balance: '餘額',
      equity: '淨值',
      floating: 'Floating P/L'
    },
    accountStatus: {
      disabled: '已停用',
      connected: '已連線',
      connecting: '連線中',
      disconnected: 'Disconnected'
    },
    quickActions: {
      title: '快速操作',
      trading: '交易',
      market: '行情',
      accounts: '帳戶',
      analytics: '分析',
      library: '策略庫',
      templates: '策略模板',
      logs: '日誌',
      bindAccount: '綁帳戶',
      closePosition: 'Close'
    },
    defaultName: 'My Dashboard'
  }
} as const;

export default dashboard;
