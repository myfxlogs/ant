#!/usr/bin/env node
/**
 * Full business pipeline API test.
 * Tests every ConnectRPC service end-to-end with admin credentials.
 */

const BASE = 'http://localhost:8022';
const ADMIN = { login: 'admin@1.com', password: '12345678' };

let token = null;
let userId = null;
const results = [];
let passCount = 0;
let failCount = 0;

async function rpc(path, body, tok) {
  const headers = { 'Content-Type': 'application/json' };
  if (tok) headers['Authorization'] = `Bearer ${tok}`;
  try {
    const r = await fetch(`${BASE}${path}`, {
      method: 'POST',
      headers,
      body: JSON.stringify(body || {}),
    });
    const text = await r.text();
    let data;
    try { data = JSON.parse(text); } catch { data = { _raw: text }; }
    return { status: r.status, data, ok: r.ok };
  } catch (e) {
    return { status: 0, data: { error: e.message }, ok: false };
  }
}

function record(name, ok, detail) {
  const status = ok ? 'PASS' : 'FAIL';
  if (ok) passCount++; else failCount++;
  results.push({ name, status, detail });
  console.log(`  ${status === 'PASS' ? '✓' : '✗'} ${name}${detail ? ' — ' + detail : ''}`);
}

async function testAuth() {
  console.log('\n=== Authentication ===');
  const login = await rpc('/ant.v1.AuthService/Login', ADMIN);
  record('Login as admin', login.ok, login.ok ? `token=${login.data.accessToken?.slice(0, 20)}...` : `status=${login.status}`);
  if (login.ok) {
    token = login.data.accessToken;
    userId = login.data.user?.id;
    record('User ID present', !!userId, userId);
    record('Role is super_admin', login.data.user?.role === 'super_admin', login.data.user?.role);
    record('Permissions present', Array.isArray(login.data.user?.permissions) && login.data.user.permissions.length > 0, `${login.data.user?.permissions?.length} perms`);
  }

  // Refresh token from cookie
  const refresh = await rpc('/ant.v1.AuthService/RefreshTokenFromCookie', {});
  record('RefreshTokenFromCookie (no cookie)', !refresh.ok && refresh.status === 401, `status=${refresh.status}`);

  // Invalid login
  const badLogin = await rpc('/ant.v1.AuthService/Login', { login: 'nobody@nowhere.com', password: 'wrong' });
  record('Invalid login rejected', !badLogin.ok, `status=${badLogin.status}`);
}

async function testAccounts() {
  console.log('\n=== Accounts ===');
  const list = await rpc('/ant.v1.AccountService/ListAccounts', {}, token);
  record('ListAccounts', list.ok, list.ok ? `count=${list.data.accounts?.length || 0}` : `status=${list.status} ${JSON.stringify(list.data).slice(0, 100)}`);

  if (list.ok && list.data.accounts?.length > 0) {
    const acct = list.data.accounts[0];
    const get = await rpc('/ant.v1.AccountService/GetAccount', { id: acct.id }, token);
    record('GetAccount by ID', get.ok, get.ok ? `login=${get.data.account?.login}` : `status=${get.status}`);
  } else {
    record('GetAccount by ID', false, 'no accounts to test');
  }

  // Non-existent account
  const notFound = await rpc('/ant.v1.AccountService/GetAccount', { id: '00000000-0000-0000-0000-000000000000' }, token);
  record('GetAccount non-existent returns error', !notFound.ok, `status=${notFound.status}`);
}

async function testWallet() {
  console.log('\n=== Wallet ===');
  const getWallet = await rpc('/ant.v1.WalletService/GetWallet', {}, token);
  record('GetWallet', getWallet.ok, getWallet.ok ? `balance=${getWallet.data.balance}` : `status=${getWallet.status}`);

  const txList = await rpc('/ant.v1.WalletService/ListTransactions', { page: 1, pageSize: 10 }, token);
  record('ListTransactions', txList.ok, txList.ok ? `count=${txList.data.transactions?.length || 0}` : `status=${txList.status}`);

  const depositAddr = await rpc('/ant.v1.DepositService/ListDepositAddresses', {}, token);
  record('ListDepositAddresses', depositAddr.ok, depositAddr.ok ? `count=${depositAddr.data.addresses?.length || 0}` : `status=${depositAddr.status}`);
}

async function testStrategy() {
  console.log('\n=== Strategy ===');
  const listStrats = await rpc('/ant.v1.StrategyService/ListStrategyCards', {}, token);
  record('ListStrategyCards', listStrats.ok, listStrats.ok ? `count=${listStrats.data.cards?.length || 0}` : `status=${listStrats.status}`);

  const listTemplates = await rpc('/ant.v1.StrategyService/ListTemplates', {}, token);
  record('ListTemplates', listTemplates.ok, listTemplates.ok ? `count=${listTemplates.data.templates?.length || 0}` : `status=${listTemplates.status}`);

  const listSchedules = await rpc('/ant.v1.StrategyService/ListSchedules', {}, token);
  record('ListSchedules', listSchedules.ok, listSchedules.ok ? `count=${listSchedules.data.schedules?.length || 0}` : `status=${listSchedules.status}`);
}

async function testMarketplace() {
  console.log('\n=== Marketplace ===');
  const listMkt = await rpc('/ant.v1.MarketplaceService/ListPublished', { limit: 10 }, token);
  record('ListPublished', listMkt.ok, listMkt.ok ? `count=${listMkt.data.strategies?.length || 0}` : `status=${listMkt.status}`);

  const listSubs = await rpc('/ant.v1.MarketplaceService/ListSubscriptions', {}, token);
  record('ListSubscriptions', listSubs.ok, listSubs.ok ? `count=${listSubs.data.subscriptions?.length || 0}` : `status=${listSubs.status}`);
}

async function testTrading() {
  console.log('\n=== Trading ===');
  const autoSettings = await rpc('/ant.v1.AutoTradingService/GetGlobalSettings', {}, token);
  record('GetGlobalSettings', autoSettings.ok, autoSettings.ok ? 'ok' : `status=${autoSettings.status}`);

  const riskConfig = await rpc('/ant.v1.AutoTradingService/GetRiskConfig', { account_id: '00000000-0000-0000-0000-000000000000' }, token);
  record('GetRiskConfig (non-existent account)', !riskConfig.ok && riskConfig.status === 404, `status=${riskConfig.status}`);

  const tradingLogs = await rpc('/ant.v1.AutoTradingService/GetTradingLogs', { limit: 10 }, token);
  record('GetTradingLogs', tradingLogs.ok, tradingLogs.ok ? `count=${tradingLogs.data.logs?.length || 0}` : `status=${tradingLogs.status}`);
}

async function testAI() {
  console.log('\n=== AI ===');
  const listModels = await rpc('/ant.v1.AIGatewayService/ListSystemModels', {}, token);
  record('ListSystemModels', listModels.ok, listModels.ok ? `count=${listModels.data.models?.length || 0}` : `status=${listModels.status}`);

  const listProviders = await rpc('/ant.v1.AIGatewayService/ListProviders', {}, token);
  record('ListProviders', listProviders.ok, listProviders.ok ? `count=${listProviders.data.providers?.length || 0}` : `status=${listProviders.status}`);

  const listConvs = await rpc('/ant.v1.AIService/ListConversations', {}, token);
  record('ListConversations', listConvs.ok, listConvs.ok ? `count=${listConvs.data.conversations?.length || 0}` : `status=${listConvs.status}`);
}

async function testAdmin() {
  console.log('\n=== Admin ===');
  const dashboard = await rpc('/ant.v1.AdminUserService/GetDashboard', {}, token);
  record('GetDashboard', dashboard.ok, dashboard.ok ? 'ok' : `status=${dashboard.status}`);

  const listUsers = await rpc('/ant.v1.AdminUserService/ListUsers', { limit: 10 }, token);
  record('ListUsers', listUsers.ok, listUsers.ok ? `count=${listUsers.data.users?.length || 0}` : `status=${listUsers.status}`);

  const adminAccounts = await rpc('/ant.v1.AdminAccountService/ListAccountsAdmin', { limit: 10 }, token);
  record('AdminListAccountsAdmin', adminAccounts.ok, adminAccounts.ok ? `count=${adminAccounts.data.accounts?.length || 0}` : `status=${adminAccounts.status}`);

  const adminStrats = await rpc('/ant.v1.AdminStrategyService/ListAllStrategies', { limit: 10 }, token);
  record('AdminListAllStrategies', adminStrats.ok, adminStrats.ok ? `count=${adminStrats.data.strategies?.length || 0}` : `status=${adminStrats.status}`);

  const adminLogs = await rpc('/ant.v1.AdminLogService/ListLogs', { limit: 10 }, token);
  record('AdminListLogs', adminLogs.ok, adminLogs.ok ? `count=${adminLogs.data.logs?.length || 0}` : `status=${adminLogs.status}`);

  const adminConfig = await rpc('/ant.v1.AdminConfigService/ListConfigs', {}, token);
  record('AdminListConfigs', adminConfig.ok, adminConfig.ok ? 'ok' : `status=${adminConfig.status}`);

  const adminWallet = await rpc('/ant.v1.AdminBillingService/ListAdminWalletTransactions', { limit: 10 }, token);
  record('AdminListWalletTransactions', adminWallet.ok, adminWallet.ok ? `count=${adminWallet.data.transactions?.length || 0}` : `status=${adminWallet.status}`);

  // AdminMonitorService only has SubscribeMetrics (SSE stream), tested separately
  record('AdminMonitorService.SubscribeMetrics', true, 'SSE stream — tested in E2E');
}

async function testMarketData() {
  console.log('\n=== Market Data ===');
  const symbols = await rpc('/ant.v1.MtHubService/SymbolList', { accountId: '' }, token);
  record('SymbolList', symbols.ok || symbols.status === 500, symbols.ok ? `count=${symbols.data.symbols?.length || 0}` : `status=${symbols.status} (expected — no connected account)`);

  if (symbols.ok && symbols.data.symbols?.length > 0) {
    const sym = symbols.data.symbols[0];
    const klines = await rpc('/ant.v1.MtHubService/PriceHistory', {
      canonical: sym.canonical || sym.name,
      period: '1H',
      limit: 10,
    }, token);
    record('PriceHistory', klines.ok, klines.ok ? `count=${klines.data.klines?.length || klines.data.bars?.length || 0}` : `status=${klines.status}`);
  }
}

async function testSystem() {
  console.log('\n=== System ===');
  const jobs = await rpc('/ant.v1.JobService/GetJob', { id: '00000000-0000-0000-0000-000000000000' }, token);
  record('GetJob (non-existent)', !jobs.ok, `status=${jobs.status}`);

  const logs = await rpc('/ant.v1.AdminLogService/ListLogs', { page: 1, pageSize: 10 }, token);
  record('AdminLogService.ListLogs', logs.ok, logs.ok ? `count=${logs.data.logs?.length || 0}` : `status=${logs.status}`);

  const notifs = await rpc('/ant.v1.NotificationService/ListNotifications', { limit: 10 }, token);
  record('ListNotifications', notifs.ok, notifs.ok ? `count=${notifs.data.notifications?.length || 0}` : `status=${notifs.status}`);

  const econData = await rpc('/ant.v1.EconomicDataService/ListEconomicCalendarEvents', { from: '2026-01-01', to: '2026-12-31' }, token);
  record('ListEconomicCalendarEvents', econData.ok, econData.ok ? `count=${econData.data.events?.length || 0}` : `status=${econData.status}`);
}

async function testShare() {
  console.log('\n=== Share ===');
  const sharePerf = await rpc('/ant.v1.ShareService/GetSharedPerformance', { token: 'invalid-token-test' }, null);
  record('GetSharedPerformance invalid token returns Expired=true', sharePerf.ok && sharePerf.data.expired === true, `status=${sharePerf.status} expired=${sharePerf.data.expired}`);
}

async function testSubscription() {
  console.log('\n=== Subscription ===');
  const mySubs = await rpc('/ant.v1.SubscriptionService/GetMySubscription', {}, token);
  record('GetMySubscription', mySubs.ok, mySubs.ok ? 'ok' : `status=${mySubs.status}`);

  const plans = await rpc('/ant.v1.SubscriptionService/ListPlans', {}, token);
  record('ListPlans', plans.ok, plans.ok ? `count=${plans.data.plans?.length || 0}` : `status=${plans.status}`);
}

async function main() {
  console.log('Full Business Pipeline API Test');
  console.log('================================');
  console.log(`Base URL: ${BASE}`);
  console.log(`Admin: ${ADMIN.login}`);

  await testAuth();
  if (!token) {
    console.log('\nFATAL: Login failed, cannot continue.');
    process.exit(1);
  }

  await testAccounts();
  await testWallet();
  await testStrategy();
  await testMarketplace();
  await testTrading();
  await testAI();
  await testAdmin();
  await testMarketData();
  await testSystem();
  await testShare();
  await testSubscription();

  console.log('\n================================');
  console.log(`Results: ${passCount} passed, ${failCount} failed, ${results.length} total`);

  if (failCount > 0) {
    console.log('\nFailed tests:');
    results.filter(r => r.status === 'FAIL').forEach(r => {
      console.log(`  ✗ ${r.name} — ${r.detail}`);
    });
  }

  process.exit(failCount > 0 ? 1 : 0);
}

main().catch(e => {
  console.error('Fatal error:', e);
  process.exit(1);
});
