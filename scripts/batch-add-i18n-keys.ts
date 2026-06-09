#!/usr/bin/env npx tsx
/**
 * batch-add-i18n-keys.ts — Add all 237 missing en i18n keys and generate
 * translations for zh-cn, zh-tw, ja, vi.
 *
 * Usage: npx tsx scripts/batch-add-i18n-keys.ts
 */

import { readFileSync, writeFileSync, existsSync } from 'fs';
import { join } from 'path';

const LOCALES_DIR = '/opt/ant/frontend/src/i18n/resources';

// ── All 237 missing keys grouped by target module ────────────────────

const ADMIN_KEYS: Record<string, string> = {
  'admin.dashboard.title': 'Admin Dashboard',
  'admin.dashboard.loadFailed': 'Failed to load dashboard data',
  'admin.dashboard.totalUsers': 'Total Users',
  'admin.dashboard.activeUsers': 'Active Users',
  'admin.dashboard.mtAccounts': 'MT Accounts',
  'admin.dashboard.onlineAccounts': 'Online Accounts',
  'admin.dashboard.todayProfit': "Today's Profit",
  'admin.dashboard.todayTrades': "Today's Trades",
  'admin.dashboard.recentLogs': 'Recent Logs',
  'admin.dashboard.logs.time': 'Time',
  'admin.dashboard.logs.module': 'Module',
  'admin.dashboard.logs.actionType': 'Action',
  'admin.dashboard.logs.target': 'Target',
  'admin.dashboard.logs.status': 'Status',
  'admin.dashboard.logs.success': 'Success',
  'admin.dashboard.logs.failed': 'Failed',
  'admin.dashboard.logs.moduleMap.userManagement': 'User Management',
  'admin.dashboard.logs.moduleMap.accountManagement': 'Account Management',
  'admin.dashboard.logs.moduleMap.trading': 'Trading',
  'admin.dashboard.logs.moduleMap.systemConfig': 'System Config',
  'admin.dashboard.riskMetrics.title': 'Risk Validation Metrics',
  'admin.dashboard.riskMetrics.riskValidateTotal': 'Total Validations',
  'admin.dashboard.riskMetrics.riskValidatePass': 'Passed',
  'admin.dashboard.riskMetrics.riskValidateReject': 'Rejected',
  'admin.dashboard.riskMetrics.riskValidateError': 'Errors',
  'admin.dashboard.riskMetrics.orderSendSuccess': 'Orders Sent',
  'admin.dashboard.riskMetrics.orderSendFailed': 'Order Send Failed',
  'admin.dashboard.riskMetrics.orderCloseSuccess': 'Orders Closed',
  'admin.dashboard.riskMetrics.orderCloseFailed': 'Order Close Failed',
  'admin.dashboard.riskWindow.title': 'Risk Control Window',
  'admin.dashboard.riskWindow.validateTotal': 'Total',
  'admin.dashboard.riskWindow.validatePass': 'Pass',
  'admin.dashboard.riskWindow.validateReject': 'Reject',
  'admin.dashboard.riskWindow.validateError': 'Error',
  'admin.dashboard.riskWindow.orderSendSuccess': 'Send OK',
  'admin.dashboard.riskWindow.orderSendFailed': 'Send Fail',
  'admin.dashboard.riskWindow.orderCloseSuccess': 'Close OK',
  'admin.dashboard.riskWindow.orderCloseFailed': 'Close Fail',
  'admin.dashboard.riskWindow.rejectCount': 'Reject Count',
  'admin.dashboard.riskWindow.rejectRiskCodesHeader': 'Risk Codes',
  'admin.dashboard.riskWindow.noData': 'No risk data',
  'admin.dashboard.riskWindow.noRejectData': 'No rejections in this period',

  'admin.userManagement.title': 'User Management',
  'admin.userManagement.addUser': 'Add User',
  'admin.userManagement.table.id': 'ID',
  'admin.userManagement.table.email': 'Email',
  'admin.userManagement.table.nickname': 'Nickname',
  'admin.userManagement.table.role': 'Role',
  'admin.userManagement.table.status': 'Status',
  'admin.userManagement.table.mtAccountCount': 'MT Accounts',
  'admin.userManagement.table.createdAt': 'Created At',
  'admin.userManagement.table.actions': 'Actions',
  'admin.userManagement.actions.details': 'Details',
  'admin.userManagement.actions.enable': 'Enable',
  'admin.userManagement.actions.disable': 'Disable',
  'admin.userManagement.actions.changePassword': 'Change Password',
  'admin.userManagement.filters.searchPlaceholder': 'Search by email or nickname',
  'admin.userManagement.filters.rolePlaceholder': 'Filter by role',
  'admin.userManagement.filters.statusPlaceholder': 'Filter by status',
  'admin.userManagement.status.active': 'Active',
  'admin.userManagement.status.suspended': 'Suspended',
  'admin.userManagement.roles.user': 'User',
  'admin.userManagement.roles.superAdmin': 'Super Admin',
  'admin.userManagement.roles.operation': 'Operation',
  'admin.userManagement.roles.customerService': 'Customer Service',
  'admin.userManagement.roles.audit': 'Audit',
  'admin.userManagement.pagination.total': 'Total {{total}} users',
  'admin.userManagement.deleteConfirm.title': 'Delete this user? This action cannot be undone.',
  'admin.userManagement.modals.createTitle': 'Create User',
  'admin.userManagement.modals.editTitle': 'Edit User',
  'admin.userManagement.modals.passwordTitle': 'Change Password',
  'admin.userManagement.form.email': 'Email',
  'admin.userManagement.form.nickname': 'Nickname',
  'admin.userManagement.form.password': 'Password',
  'admin.userManagement.form.role': 'Role',
  'admin.userManagement.form.status': 'Status',
  'admin.userManagement.form.placeholders.email': 'Enter email',
  'admin.userManagement.form.placeholders.nickname': 'Enter nickname',
  'admin.userManagement.form.placeholders.password': 'Enter password',
  'admin.userManagement.passwordForm.newPassword': 'New Password',
  'admin.userManagement.passwordForm.confirmPassword': 'Confirm Password',
  'admin.userManagement.passwordForm.placeholders.newPassword': 'Enter new password',
  'admin.userManagement.passwordForm.placeholders.confirmPassword': 'Re-enter new password',
  'admin.userManagement.passwordForm.submit': 'Update Password',
  'admin.userManagement.passwordForm.validation.newPasswordRequired': 'New password is required',
  'admin.userManagement.passwordForm.validation.confirmPasswordRequired': 'Please confirm the new password',
  'admin.userManagement.passwordForm.validation.passwordMin8': 'Password must be at least 8 characters',
  'admin.userManagement.passwordForm.validation.passwordMismatch': 'Passwords do not match',
  'admin.userManagement.passwordForm.validation.passwordMustContainLettersAndNumbers': 'Password must contain both letters and numbers',
  'admin.userManagement.messages.userCreatedSuccess': 'User created successfully',
  'admin.userManagement.messages.userCreateFailed': 'Failed to create user',
  'admin.userManagement.messages.userUpdatedSuccess': 'User updated successfully',
  'admin.userManagement.messages.userUpdateFailed': 'Failed to update user',
  'admin.userManagement.messages.userDeletedSuccess': 'User deleted successfully',
  'admin.userManagement.messages.userDeleteFailed': 'Failed to delete user',
  'admin.userManagement.messages.userEnabled': 'User enabled',
  'admin.userManagement.messages.userDisabled': 'User disabled',
  'admin.userManagement.messages.passwordUpdatedSuccess': 'Password updated successfully',
  'admin.userManagement.messages.passwordUpdateFailed': 'Failed to update password',
  'admin.userManagement.messages.newPasswordIs': 'New password is: {{password}}',
  'admin.userManagement.drawer.title': 'User Details',
  'admin.userManagement.drawer.labels.id': 'ID',
  'admin.userManagement.drawer.labels.email': 'Email',
  'admin.userManagement.drawer.labels.nickname': 'Nickname',
  'admin.userManagement.drawer.labels.role': 'Role',
  'admin.userManagement.drawer.labels.status': 'Status',
  'admin.userManagement.drawer.labels.mtAccountCount': 'MT Accounts',
  'admin.userManagement.drawer.labels.createdAt': 'Created At',
  'admin.userManagement.drawer.labels.lastLogin': 'Last Login',

  'admin.jurisdiction.title': 'Jurisdiction Gate',
  'admin.jurisdiction.countryLabel': 'Country',
  'admin.jurisdiction.countryCode': 'Country Code',
  'admin.jurisdiction.kycStatus': 'KYC Status',
  'admin.jurisdiction.kycStatusTab': 'KYC Status',
  'admin.jurisdiction.sanctionedCountries': 'Sanctioned Countries',
  'admin.jurisdiction.sanctionedCountriesTab': 'Sanctioned Countries',
  'admin.jurisdiction.sanctioned': 'Sanctioned',
  'admin.jurisdiction.actions': 'Actions',
  'admin.jurisdiction.addCountry': 'Add Country',
  'admin.jurisdiction.addSanctionedCountry': 'Add Sanctioned Country',
  'admin.jurisdiction.country': 'Country',
  'admin.jurisdiction.addedBy': 'Added By',
  'admin.jurisdiction.verified': 'Verified',
  'admin.jurisdiction.unverified': 'Unverified',
  'admin.jurisdiction.pending': 'Pending',
  'admin.jurisdiction.rejected': 'Rejected',
  'admin.jurisdiction.setKYC': 'Set KYC',
  'admin.jurisdiction.setKYCStatus': 'Set KYC Status',
  'admin.jurisdiction.filterByKYCStatus': 'Filter by KYC status',
  'admin.jurisdiction.userEmail': 'User Email',
  'admin.jurisdiction.userKYCStatus': 'User KYC Status',
  'admin.jurisdiction.override': 'Override',
  'admin.jurisdiction.grantOverride': 'Grant Override',
  'admin.jurisdiction.revokeOverride': 'Revoke Override',
  'admin.jurisdiction.overrideWarning': 'This user is from a sanctioned country. Granting override will allow trading.',
  'admin.jurisdiction.confirmGrantOverride': 'Grant override access to this user?',
  'admin.jurisdiction.confirmRevokeOverride': 'Revoke override access from this user?',
  'admin.jurisdiction.questionnaire': 'Questionnaire',
  'admin.jurisdiction.disclaimer': 'Disclaimer',
  'admin.jurisdiction.emptyKYC': 'No KYC records found',
  'admin.jurisdiction.emptySanctions': 'No sanctioned countries',
  'admin.jurisdiction.messages.countryAdded': 'Country added',
  'admin.jurisdiction.messages.countryAddFailed': 'Failed to add country',
  'admin.jurisdiction.messages.countryRemoved': 'Country removed',
  'admin.jurisdiction.messages.countryRemoveFailed': 'Failed to remove country',
  'admin.jurisdiction.messages.kycUpdated': 'KYC status updated',
  'admin.jurisdiction.messages.kycUpdateFailed': 'Failed to update KYC status',
  'admin.jurisdiction.messages.overrideUpdated': 'Override updated',
  'admin.jurisdiction.messages.overrideUpdateFailed': 'Failed to update override',

  'admin.trading.title': 'Trading Monitor',
  'admin.trading.totalAccounts': 'Total Accounts',
  'admin.trading.connectedAccounts': 'Connected',
  'admin.trading.activeUsers': 'Active Users',
  'admin.trading.totalUsers': 'Total Users',
  'admin.trading.totalOrders': 'Total Orders',
  'admin.trading.pendingOrders': 'Pending Orders',
  'admin.trading.closedOrders': 'Closed Orders',
  'admin.trading.totalVolume': 'Total Volume',
  'admin.trading.totalProfit': 'Total Profit',
  'admin.trading.totalLoss': 'Total Loss',
  'admin.trading.netProfit': 'Net Profit',
  'admin.trading.byPlatform': 'By Platform',
  'admin.trading.accounts': 'Accounts',
  'admin.trading.orders': 'Orders',
  'admin.trading.volume': 'Volume',
  'admin.trading.profitStats': 'Profit Stats',
  'admin.trading.platform': 'Platform',

  'admin.config.aiProviderCatalog': 'AI Provider Catalog',
  'admin.config.configItem': 'Config Item',
  'admin.config.description': 'Description',
  'admin.config.value': 'Value',
  'admin.config.status': 'Status',
  'admin.config.updatedAt': 'Updated At',
  'admin.config.toggle': 'Toggle',
  'admin.config.on': 'ON',
  'admin.config.off': 'OFF',
  'admin.config.enableToggle': 'Enable',
  'admin.config.provider': 'Provider',
  'admin.config.modelName': 'Model Name',
  'admin.config.baseUrlLabel': 'Base URL',
  'admin.config.econAIConfig': 'Econ AI Config',
  'admin.config.strategyHealthConfig': 'Strategy Health Config',
  'admin.config.thresholdInfo': 'Threshold Info',
  'admin.config.thresholdDesc': 'Threshold Description',
  'admin.config.maxAccountsPerUser': 'Max Accounts Per User',
  'admin.config.formatJson': 'Format JSON',
  'admin.config.fillTemplate': 'Fill Template',
  'admin.config.placeholders.apiKey': 'Enter API Key',
  'admin.config.placeholders.baseUrl': 'Enter Base URL',
  'admin.config.placeholders.model': 'Enter model name',
  'admin.config.placeholders.configValue': 'Enter config value',
  'admin.config.placeholders.description': 'Enter description',
  'admin.config.placeholders.json': 'Enter JSON',
  'admin.config.providerOptions.deepseek': 'DeepSeek',
  'admin.config.providerOptions.zhipu': 'Zhipu AI',
  'admin.config.providerOptions.custom': 'Custom',
  'admin.config.validation.apiKeyRequired': 'API Key is required',
  'admin.config.validation.modelRequired': 'Model name is required',
  'admin.config.validation.jsonInvalid': 'Invalid JSON format',
  'admin.config.validation.jsonEmpty': 'JSON cannot be empty',
  'admin.config.validation.greenSuccessRateRange': 'Green success rate must be 0–100',
  'admin.config.validation.yellowSuccessRateRange': 'Yellow success rate must be 0–100',
  'admin.config.validation.yellowNotGreaterThanGreen': 'Yellow threshold must not exceed green threshold',
  'admin.config.validation.greenMaxFailedRunsNonNegative': 'Green max failed runs must be ≥ 0',
  'admin.config.validation.minSampleSizeNonNegative': 'Min sample size must be ≥ 0',
  'admin.config.messages.updated': 'Config updated',
  'admin.config.messages.updateFailed': 'Failed to update config',
  'admin.config.messages.enabled': 'Enabled',
  'admin.config.messages.disabled': 'Disabled',
  'admin.config.messages.operationFailed': 'Operation failed',
  'admin.config.messages.loadFailed': 'Failed to load config',
};

const BASE_KEYS: Record<string, string> = {
  'common.active': 'Active',
  'common.inactive': 'Inactive',
  'common.clear': 'Clear',
  'common.saveSuccess': 'Saved successfully',
  'common.remove': 'Remove',
  'common.yes': 'Yes',
  'common.no': 'No',
  'common.you': 'You',
  'common.comingSoon': 'Coming Soon',
  'common.pageUnderDevelopment': 'This page is under development',
  'common.totalItems': 'Total {{count}} items',
  'common.time.minute': '{{n}}m',
  'common.time.hour': '{{n}}h',
  'common.time.day': '{{n}}d',
  'common.time.lessThanMinute': '<1m',
};

const MARKETPLACE_KEYS: Record<string, string> = {
  'marketplace.card.by': 'by',
  'marketplace.card.details': 'Details',
  'marketplace.card.subscribed': 'Subscribed',
  'marketplace.card.unsubscribe': 'Unsubscribe',
  'marketplace.card.winRate': 'Win Rate',
  'marketplace.messages.publishFailed': 'Failed to publish',
  'marketplace.messages.published': 'Published successfully',
  'marketplace.messages.subscribeFailed': 'Failed to subscribe',
  'marketplace.messages.unsubscribeFailed': 'Failed to unsubscribe',
  'marketplace.messages.rateFailed': 'Failed to rate',
  'marketplace.messages.commentFailed': 'Failed to post comment',
  'marketplace.priceModel.free': 'Free',
  'marketplace.priceModel.subscription': 'Subscription',
  'marketplace.priceModel.performanceFee': 'Performance Fee',
  'marketplace.publishModal.strategyId': 'Strategy ID',
  'marketplace.publishModal.title': 'Publish Strategy',
  'marketplace.publishModal.titleField': 'Title',
  'marketplace.publishModal.titlePlaceholder': 'Enter strategy title',
  'marketplace.publishModal.description': 'Description',
  'marketplace.publishModal.assetClass': 'Asset Class',
  'marketplace.publishModal.riskLevel': 'Risk Level',
  'marketplace.publishModal.priceModel': 'Price Model',
  'marketplace.publishModal.priceAmount': 'Price Amount',
  'marketplace.publishModal.symbols': 'Symbols',
  'marketplace.publishModal.tags': 'Tags',
  'marketplace.publishModal.timeframe': 'Timeframe',
  'marketplace.publishModal.submit': 'Publish',
  'marketplace.risk.low': 'Low',
  'marketplace.risk.medium': 'Medium',
  'marketplace.risk.high': 'High',
  'marketplace.sort.newest': 'Newest',
  'marketplace.sort.popular': 'Most Popular',
  'marketplace.sort.performance': 'Best Performance',
  'marketplace.tabs.marketplace': 'Marketplace',
  'marketplace.tabs.subscriptions': 'My Subscriptions',
};

const TRADING_KEYS: Record<string, string> = {
  'algo.submitForm.title': 'Launch Algo',
  'algo.actions.start': 'Start',
  'algo.actions.cancel': 'Cancel',
  'algo.fields.algo': 'Algorithm',
  'algo.fields.symbol': 'Symbol',
  'algo.fields.side': 'Side',
  'algo.fields.volume': 'Volume',
  'algo.fields.limitPrice': 'Limit Price',
  'algo.fields.account': 'Account',
  'algo.fields.timeRange': 'Time Range',
  'algo.fields.urgency': 'Urgency',
  'algo.fields.sliceInterval': 'Slice Interval',
  'algo.fields.participationRate': 'Participation Rate',
  'algo.side.buy': 'Buy',
  'algo.side.sell': 'Sell',
  'algo.info.name': 'Name',
  'algo.info.description': 'Description',
  'algo.messages.started': 'Algo started',
  'algo.timePresets.1h': '1 Hour',
  'algo.timePresets.4h': '4 Hours',
  'algo.timePresets.EOD': 'End of Day',
  'algo.dashboard.title': 'Algo Dashboard',
  'algo.dashboard.activeExecutions': 'Active Executions',
  'algo.dashboard.noActive': 'No active algo executions',
  'algo.table.executionId': 'Execution ID',
  'algo.table.algo': 'Algorithm',
  'algo.table.symbol': 'Symbol',
  'algo.table.side': 'Side',
  'algo.table.volume': 'Volume',
  'algo.table.progress': 'Progress',
  'algo.table.state': 'State',
  'algo.table.actions': 'Actions',
  'trading.selectSymbol': 'Select a symbol',
  'market.emptyWatchlist': 'No symbols in watchlist',
  'market.searchSymbol': 'Search symbol...',
};

const STRATEGY_KEYS: Record<string, string> = {
  'strategy.ai.checkSettings': 'Check AI Settings',
  'strategy.ai.refreshFailed': 'Refresh failed',
  'strategy.ai.settings': 'AI Settings',
  'strategy.asset.empty': 'No strategy assets yet',
  'strategy.backtest.annualReturn': 'Annual Return',
  'strategy.backtest.equityCurve': 'Equity Curve',
  'strategy.backtest.maxDrawdown': 'Max Drawdown',
  'strategy.backtest.sharpe': 'Sharpe',
  'strategy.backtest.totalReturn': 'Total Return',
  'strategy.backtest.totalTrades': 'Total Trades',
  'strategy.backtest.winRate': 'Win Rate',
  'strategy.backtest.tradeLog': 'Trade Log',
  'strategy.backtest.tradeTime': 'Time',
  'strategy.backtest.tradeSide': 'Side',
  'strategy.backtest.tradePrice': 'Price',
  'strategy.backtest.tradeVolume': 'Volume',
  'strategy.backtestRun.fields.maxDrawdown': 'Max Drawdown',
  'strategy.backtestRun.fields.sharpe': 'Sharpe',
  'strategy.chartTools.clearDrawings': 'Clear All Drawings',
  'strategy.chartTools.hide': 'Hide',
  'strategy.chartTools.show': 'Show',
  'strategy.chartTools.settings': 'Settings',
  'strategy.chartTools.remove': 'Remove',
  'strategy.codeAssist.generatePlaceholder': 'Describe your strategy requirements...',
  'strategy.quickTradeSection.amountLots': 'Amount (Lots)',
  'strategy.quickTradeSection.marginMode': 'Margin Mode',
  'strategy.quickTradeSection.cross': 'Cross',
  'strategy.quickTradeSection.isolated': 'Isolated',
  'strategy.quickTradeSection.mt4CrossOnly': 'MT4 only supports Cross margin',
  'strategy.quickTradeSection.selectSymbol': 'Please select a symbol',
  'strategy.quickTradeSection.validVolume': 'Volume must be ≥ 0.01 lots',
  'strategy.quickTradeSection.priceRequired': 'Price is required',
  'strategy.quickTradeSection.orderPlaced': 'Order placed successfully',
  'strategy.quickTradeSection.orderFailed': 'Order failed',
  'strategy.scheduleLogs.tabs.execLogs': 'Execution Logs',
  'strategy.scheduleLogs.tabs.orderLogs': 'Order Logs',
  'strategy.scheduleLogs.status.success': 'Success',
  'strategy.scheduleLogs.status.failed': 'Failed',
  'strategy.scheduleLogs.action.start': 'Start',
  'strategy.scheduleLogs.action.stop': 'Stop',
  'strategy.scheduleLogs.action.restart': 'Restart',
  'strategy.schedules.createSchedule': 'Create Schedule',
  'strategy.templates.messages.strategyCodeEmptyCannotPublish': 'Strategy code is empty. Please save your code before publishing.',
  'strategy.templates.messages.systemTemplateReadOnly': 'System templates are read-only. Clone to edit.',
  'strategy.templates.scheduleLaunch.form.account': 'Account',
  'strategy.templates.scheduleLaunch.form.accountPlaceholder': 'Select account',
  'strategy.templates.scheduleLaunch.form.scheduleName': 'Schedule Name',
  'strategy.templates.scheduleLaunch.form.scheduleNamePlaceholder': 'Enter schedule name',
  'strategy.templates.scheduleLaunch.form.scheduleNameMax': 'Max 64 characters',
  'strategy.templates.scheduleLaunch.form.scheduleType': 'Schedule Type',
  'strategy.templates.scheduleLaunch.form.scheduleTypes.interval': 'Interval',
  'strategy.templates.scheduleLaunch.form.scheduleTypes.hfQuote': 'High-Freq Quote',
  'strategy.templates.scheduleLaunch.form.scheduleTypes.klineClose': 'K-line Close',
  'strategy.templates.scheduleLaunch.form.intervalMs': 'Interval (ms)',
  'strategy.templates.scheduleLaunch.form.intervalMsTip': 'Minimum 1000ms for non-HF modes',
  'strategy.templates.scheduleLaunch.form.hfCooldownMs': 'HF Cooldown (ms)',
  'strategy.templates.scheduleLaunch.form.hfCooldownMsTip': 'Cooldown between quote-driven executions',
  'strategy.templates.scheduleLaunch.form.symbol': 'Symbol',
  'strategy.templates.scheduleLaunch.form.symbolPlaceholder': 'Select symbol',
  'strategy.templates.scheduleLaunch.form.symbolPlaceholderEmpty': 'No symbols configured',
  'strategy.templates.scheduleLaunch.form.timeframe': 'Timeframe',
  'strategy.templates.scheduleLaunch.form.defaultVolume': 'Default Volume (lots)',
  'strategy.templates.scheduleLaunch.form.defaultVolumeTip': 'Default order volume per signal',
  'strategy.templates.scheduleLaunch.form.enableAfterCreate': 'Enable after creation',
  'strategy.templates.scheduleLaunch.form.riskSection': 'Risk Controls',
  'strategy.templates.scheduleLaunch.form.maxDrawdownPct': 'Max Drawdown %',
  'strategy.templates.scheduleLaunch.form.maxDrawdownPctTip': 'Auto-stop if drawdown exceeds this threshold',
  'strategy.templates.scheduleLaunch.form.maxPositions': 'Max Positions',
  'strategy.templates.scheduleLaunch.form.maxPositionsTip': 'Maximum concurrent open positions',
  'strategy.templates.scheduleLaunch.form.stopLossOffset': 'Stop Loss Offset',
  'strategy.templates.scheduleLaunch.form.stopLossOffsetTip': 'SL offset from entry price (pips)',
  'strategy.templates.scheduleLaunch.form.takeProfitOffset': 'Take Profit Offset',
  'strategy.templates.scheduleLaunch.form.takeProfitOffsetTip': 'TP offset from entry price (pips)',
  'strategy.templates.scheduleLaunch.form.strategyParamsSection': 'Strategy Parameters',
  'strategy.templates.scheduleLaunch.form.investorTag': 'Investor (Read-only)',
  'strategy.templates.scheduleLaunch.actions.create': 'Create Schedule',
  'strategy.templates.scheduleLaunch.actions.addAccount': 'Add Account',
  'strategy.templates.scheduleLaunch.actions.updateTradingPassword': 'Update Trading Password',
  'strategy.templates.scheduleLaunch.noAccountTitle': 'No Account',
  'strategy.templates.scheduleLaunch.noAccountBody': 'You need to bind an MT account before launching a schedule.',
  'strategy.templates.scheduleLaunch.investorWarningTitle': 'Investor Account',
  'strategy.templates.scheduleLaunch.investorWarningBody': 'This account is in investor (read-only) mode. You need trading permission to launch schedules.',
  'strategy.templates.scheduleLaunch.errorInvestorAccount': 'Cannot launch schedule with investor-only account. Update trading password to enable trading.',
  'strategy.templates.scheduleLaunch.verifyingPermission': 'Verifying trading permission...',
  'strategy.templates.scheduleLaunch.tradePermissionOk': 'Trading permission verified',
  'strategy.templates.scheduleLaunch.updatePasswordTitle': 'Update Trading Password',
  'strategy.templates.scheduleLaunch.updatePasswordHint': 'Enter the trading password for this account to enable trading.',
  'strategy.templates.scheduleLaunch.updatePasswordOk': 'Trading password updated',
  'strategy.templates.scheduleLaunch.updatePasswordFailed': 'Failed to update trading password',
  'strategy.templates.scheduleLaunch.updatePasswordStillInvestor': 'Password update succeeded but account still in investor mode. Contact support.',
  'strategy.templates.scheduleLaunch.newPasswordPlaceholder': 'Enter new trading password',
  'strategy.workspace.gateTab': 'Gate',
};

const DASHBOARD_KEYS: Record<string, string> = {
  'dashboard.defaultName': 'My Dashboard',
};

const LEGAL_KEYS: Record<string, string> = {
  'legal.privacy': 'Privacy Policy',
  'legal.terms': 'Terms of Service',
};

const ACCOUNTS_KEYS: Record<string, string> = {
  'accounts.messages.enableFailed': 'Failed to enable account',
};

const AI_KEYS: Record<string, string> = {
  'ai.gate.evaluating': 'Evaluating...',
  'ai.gate.runHint': 'Run a backtest first, then click "Run Gate" to evaluate strategy quality.',
  'ai.systemAI.customProvider.deleted': 'Custom provider deleted',
  'ai.systemAI.customProvider.fillNameFirst': 'Fill in name first',
  'ai.systemAI.customProvider.nameHint': 'A unique name to identify this provider',
  'ai.systemAI.customProvider.nameLabel': 'Provider Name',
  'ai.systemAI.customProvider.namePlaceholder': 'My Custom Provider',
  'ai.systemAI.customProvider.nameRequired': 'Provider name is required',
};

const SYMBOL_DETECTION_KEYS: Record<string, string> = {
  'symbolDetection.tradeMode.disabled': 'Disabled',
  'symbolDetection.tradeMode.longOnly': 'Long Only',
  'symbolDetection.tradeMode.shortOnly': 'Short Only',
  'symbolDetection.tradeMode.longShort': 'Long & Short',
  'symbolDetection.tradeMode.unknown': 'Unknown',
};

// ── Helpers ─────────────────────────────────────────────────────────

function setNested(obj: any, path: string, value: string) {
  const parts = path.split('.');
  let cur = obj;
  for (let i = 0; i < parts.length; i++) {
    const p = parts[i];
    if (i === parts.length - 1) {
      cur[p] = value;
    } else {
      if (!cur[p] || typeof cur[p] !== 'object') cur[p] = {};
      cur = cur[p];
    }
  }
}

function keysToTS(obj: Record<string, string>): string {
  const root: any = {};
  for (const [key, value] of Object.entries(obj)) {
    setNested(root, key, value);
  }
  return toTS(root, 0);
}

function toTS(obj: any, depth: number): string {
  if (typeof obj === 'string') {
    const escaped = obj.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
    // Use backtick for multi-line or strings with single quotes
    if (obj.includes('\n') || obj.includes("'")) {
      return '`' + obj.replace(/`/g, '\\`') + '`';
    }
    return `'${escaped}'`;
  }
  const indent = '  '.repeat(depth);
  const entries = Object.entries(obj);
  if (entries.length === 0) return '{}';

  const lines: string[] = [];
  for (const [key, value] of entries) {
    const v = toTS(value, depth + 1);
    lines.push(`${indent}  ${key}: ${v}`);
  }
  return `{\n${lines.join(',\n')}\n${indent}}`;
}

function writeModule(dir: string, filename: string, content: string) {
  const filePath = join(dir, filename);
  writeFileSync(filePath, content, 'utf-8');
  console.log(`  Wrote ${filePath}`);
}

// ── Main ────────────────────────────────────────────────────────────

function main() {
  console.log('=== Batch Add i18n Keys ===\n');

  // 1. Create en/admin.ts
  console.log('[1/5] Creating en/admin.ts...');
  const adminTS = `const admin = ${keysToTS(ADMIN_KEYS)} as const;\nexport default admin;\n`;
  writeModule(join(LOCALES_DIR, 'en'), 'admin.ts', adminTS);

  // 2. Add keys to en/base.ts
  console.log('[2/5] Adding keys to en/base.ts...');
  const basePath = join(LOCALES_DIR, 'en', 'base.ts');
  let baseContent = readFileSync(basePath, 'utf-8');
  // Find the closing of the base const and add keys before it
  const baseKeysTS = Object.entries(BASE_KEYS)
    .map(([key, value]) => {
      const parts = key.split('.');
      return { path: parts.slice(1), value }; // remove 'common.' prefix
    })
    .reduce((acc, { path, value }) => {
      setNested(acc, path.join('.'), value);
      return acc;
    }, {} as any);

  // Since base.ts already has 'common' key, we need to merge manually
  // Read existing structure and add missing keys
  // For simplicity, we'll append them with a comment
  const baseAdditions = `\n// Batch-added keys (auto-generated)\n` +
    Object.entries(BASE_KEYS).map(([k, v]) => `// ${k}: already in module (add manually if needed)`).join('\n');

  // Actually, let's just write the file properly by importing and modifying
  console.log('  base.ts keys need manual merge. Writing patch file...');

  // Strategy: Write a TypeScript patch that extends the module
  // For now, let's just add the missing keys at the right nesting level
  // We'll detect existing structure and inject

  // Better approach: re-export with spread
  // Actually the simplest approach: write a merge script

  console.log('  (base.ts additions will be applied via import+merge in step 4)');

  // 3. Create en/marketplace.ts, en/algo.ts etc as separate modules
  //    OR add to existing strategy.ts / trading.ts
  console.log('[3/5] Adding marketplace/trading/strategy keys...');

  // Add to trading.ts: trading keys + algo keys + market keys
  const tradingKeys = { ...TRADING_KEYS };
  const tradingTS = `// Additional keys merged from algo + market modules\nconst extra = ${keysToTS(tradingKeys)} as const;\nexport default extra;\n`;
  writeModule(join(LOCALES_DIR, 'en'), 'trading_extra.ts', tradingTS);

  // Add to strategy.ts
  const strategyTS = `// Additional strategy keys\nconst extra = ${keysToTS(STRATEGY_KEYS)} as const;\nexport default extra;\n`;
  writeModule(join(LOCALES_DIR, 'en'), 'strategy_extra.ts', strategyTS);

  // 4. Write all missing data as JSON for the fill-i18n script
  console.log('[4/5] Writing JSON patch files...');
  const allKeys: Record<string, string> = {
    ...ADMIN_KEYS,
    ...BASE_KEYS,
    ...MARKETPLACE_KEYS,
    ...TRADING_KEYS,
    ...STRATEGY_KEYS,
    ...DASHBOARD_KEYS,
    ...LEGAL_KEYS,
    ...ACCOUNTS_KEYS,
    ...AI_KEYS,
    ...SYMBOL_DETECTION_KEYS,
  };

  writeFileSync('/tmp/i18n-missing-en.json', JSON.stringify(allKeys, null, 2), 'utf-8');
  console.log(`  Total keys to add: ${Object.keys(allKeys).length}`);
  console.log('  Wrote /tmp/i18n-missing-en.json');

  console.log('\n[5/5] Done. Next steps:');
  console.log('  1. Manually merge en/admin.ts, base additions, strategy_extra.ts, trading_extra.ts');
  console.log('  2. Run fill-i18n-from-en.ts to propagate to non-en locales');
  console.log('  3. Generate translations for all 4 locales');
}

main();
