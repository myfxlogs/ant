const dashboard = {
  dashboard: {
    welcome: '歡迎回來, {{name}}',
    subtitle: '查看您的帳戶總覽',
    bindAccount: '綁定帳戶',
    accountOverview: '帳戶總覽',
    accountList: '帳戶列表',
    viewAll: '查看全部',
    noAccounts: '暫無帳戶，點擊右上角綁定',
    stats: {
    totalEquity: '總淨值',
    connected: '已連線',
    accountCount: '帳戶數',
    totalProfit: '總浮動盈虧',
  },
  fields: {
    balance: '餘額',
    equity: '淨值',
    floating: '浮動',
  },
  accountStatus: {
    disabled: '已停用',
    connected: '已連線',
    connecting: '連線中',
    disconnected: '未連線',
  },
  quickActions: {
    title: '快速操作',
    trading: '交易',
    market: '行情',
    accounts: '帳戶',
    analytics: '分析',
    bindAccount: '綁帳戶',
    closePosition: '平倉',
  },
  },
} as const;

export default dashboard;
