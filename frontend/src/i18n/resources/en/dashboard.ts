const dashboard = {
  dashboard: {
    welcome: 'Welcome back, {{name}}',
    subtitle: 'View your account overview',
    bindAccount: 'Bind Account',
    accountOverview: 'Account Overview',
    accountList: 'Account List',
    viewAll: 'View all',
    noAccounts: 'No accounts yet. Click "Bind Account" to get started.',
    stats: {
    totalEquity: 'Total Equity',
    connected: 'Connected',
    accountCount: 'Accounts',
    totalProfit: 'Total Floating P/L',
  },
  fields: {
    balance: 'Balance',
    equity: 'Equity',
    floating: 'Floating P/L',
  },
  accountStatus: {
    disabled: 'Disabled',
    connected: 'Connected',
    connecting: 'Connecting',
    disconnected: 'Disconnected',
  },
  quickActions: {
    title: 'Quick Actions',
    trading: 'Trading',
    market: 'Market',
    accounts: 'Accounts',
    analytics: 'Analytics',
    bindAccount: 'Bind',
    closePosition: 'Close',
  },
  },
} as const;

export default dashboard;
