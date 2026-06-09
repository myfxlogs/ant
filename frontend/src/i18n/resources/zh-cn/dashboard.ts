const dashboard = {
  dashboard: {
    welcome: '欢迎回来, {{name}}',
    subtitle: '查看您的账户总览',
    templates: '策略模板',
    logs: '日志',
    bindAccount: '绑定账户',
    accountOverview: '账户总览',
    accountList: '账户列表',
    viewAll: '查看全部',
    streamLive: '实时连接',
    streamOffline: '实时离线',
    noAccounts: '暂无账户，点击右上角绑定',
    stats: {
      totalEquity: '总净值',
      connected: '已连接',
      accountCount: '账户数',
      totalProfit: '总浮动盈亏'
    },
    fields: {
      balance: '余额',
      equity: '净值',
      floating: '浮动'
    },
    accountStatus: {
      disabled: '已禁用',
      connected: '已连接',
      connecting: '连接中',
      disconnected: '未连接'
    },
    quickActions: {
      title: '快速操作',
      trading: '交易',
      market: '行情',
      accounts: '账户',
      analytics: '分析',
      templates: '策略模板',
      logs: '日志',
      bindAccount: '绑账户',
      closePosition: '平仓'
    },
    defaultName: '我的仪表盘'
  }
} as const;

export default dashboard;
