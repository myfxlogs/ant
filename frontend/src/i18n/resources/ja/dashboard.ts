const dashboard = {
  dashboard: {
    welcome: 'おかえりなさい、{{name}}',
    subtitle: '口座の概要を確認できます',
    bindAccount: '口座を連携',
    accountOverview: '口座概要',
    accountList: '口座一覧',
    viewAll: 'すべて表示',
    noAccounts: '口座がありません。右上から連携してください。',
    stats: {
      totalEquity: '有効証拠金合計',
      connected: '接続中',
      accountCount: '口座数',
      totalProfit: '含み損益合計',
    },
    fields: {
      balance: '残高',
      equity: '有効証拠金',
      floating: '含み',
    },
    accountStatus: {
      disabled: '無効',
      connected: '接続済み',
      connecting: '接続中',
      disconnected: '未接続',
    },
    quickActions: {
      title: 'クイック操作',
      trading: '取引',
      market: '相場',
      accounts: '口座',
      analytics: '分析',
      bindAccount: '連携',
      closePosition: '決済',
    },
  },
} as const;

export default dashboard;
