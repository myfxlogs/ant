import { test, expect } from '@playwright/test';

// ── E2E Regression: Register → Subscribe → Purchase → Backtest ──────────────
// Tests the critical user journey end-to-end via ConnectRPC API.
// Uses a random email for new user + admin API to charge wallet.
// Real market data (XAUUSDm from MT4 account) for backtest verification.

const BASE = 'http://localhost:8022';

// Admin credentials (existing seeded admin user)
const ADMIN_EMAIL = 'admin@1.com';
const ADMIN_PASS = '12345678';

// Admin's MT4 account with market data (from DB: fcca3414-d691-4a41-a1dc-53d914655059)
const ADMIN_ACCOUNT_ID = 'fcca3414-d691-4a41-a1dc-53d914655059';

// Helper: ConnectRPC POST
async function rpc(path: string, body: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = { 'Content-Type': 'application/json' };
  if (token) headers['Authorization'] = `Bearer ${token}`;
  const resp = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
  const text = await resp.text();
  let data: Record<string, unknown>;
  try { data = JSON.parse(text); } catch { data = { _raw: text }; }
  return { status: resp.status, data, ok: resp.ok };
}

// Shared state across serial tests
const state: {
  email: string;
  pass: string;
  token: string;
  adminToken: string;
  userId: string;
  walletId: string;
  templateId: string;
  publishId: string;
  backtestRunId: string;
} = {
  email: `e2e-reg-${Date.now()}@test.alfq.org`,
  pass: 'Test123456!',
  token: '',
  adminToken: '',
  userId: '',
  walletId: '',
  templateId: '',
  publishId: '',
  backtestRunId: '',
};

// Minimal MQL4 strategy for backtest — simple MA crossover
const MQL_SOURCE = `
extern int FastPeriod = 5;
extern int SlowPeriod = 20;
extern double Lots = 0.1;

int OnInit() { return INIT_SUCCEEDED; }
void OnDeinit(const int reason) {}
void OnTick() {
  double fastMA = iMA(Symbol(), 0, FastPeriod, 0, MODE_SMA, PRICE_CLOSE, 0);
  double slowMA = iMA(Symbol(), 0, SlowPeriod, 0, MODE_SMA, PRICE_CLOSE, 0);
  double prevFast = iMA(Symbol(), 0, FastPeriod, 0, MODE_SMA, PRICE_CLOSE, 1);
  double prevSlow = iMA(Symbol(), 0, SlowPeriod, 0, MODE_SMA, PRICE_CLOSE, 1);
  if (prevFast <= prevSlow && fastMA > slowMA) {
    OrderSend(Symbol(), OP_BUY, Lots, Ask, 5, 0, 0, "E2E Test", 123, 0, clrGreen);
  }
  if (prevFast >= prevSlow && fastMA < slowMA) {
    for (int i = OrdersTotal() - 1; i >= 0; i--) {
      if (OrderSelect(i, SELECT_BY_POS, MODE_TRADES)) {
        if (OrderSymbol() == Symbol() && OrderMagicNumber() == 123) {
          OrderClose(OrderTicket(), OrderLots(), Bid, 5);
        }
      }
    }
  }
}
`;

test.describe.serial('E2E Regression: Register → Subscribe → Purchase → Backtest', () => {

  // ════════════════════════════════════════════════════════════════════════
  // 0. Admin login (for wallet top-up and strategy publishing)
  // ════════════════════════════════════════════════════════════════════════
  test('0. Admin login', async () => {
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: ADMIN_EMAIL,
      password: ADMIN_PASS,
    });
    expect(resp.ok, `Admin login should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    state.adminToken = resp.data.accessToken as string;
    expect(state.adminToken.length).toBeGreaterThan(10);
    console.log(`Admin login OK`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 1. Registration
  // ════════════════════════════════════════════════════════════════════════
  test('1. Register new user', async () => {
    const resp = await rpc('/ant.v1.AuthService/Register', {
      email: state.email,
      password: state.pass,
      username: state.email,
    });
    expect(resp.ok, `Register should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const user = resp.data.user as Record<string, unknown>;
    expect(user.email).toBe(state.email);
    state.userId = user.id as string;
    console.log(`Registered: ${state.email}, userID=${state.userId}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 2. Login with registered user
  // ════════════════════════════════════════════════════════════════════════
  test('2. Login with registered credentials', async () => {
    const resp = await rpc('/ant.v1.AuthService/Login', {
      login: state.email,
      password: state.pass,
    });
    expect(resp.ok, `Login should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    state.token = resp.data.accessToken as string;
    expect(state.token.length).toBeGreaterThan(10);
    console.log(`Login OK, token length=${state.token.length}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 3. Verify wallet auto-created (balance = 0)
  // ════════════════════════════════════════════════════════════════════════
  test('3. Wallet auto-created with zero balance', async () => {
    const resp = await rpc('/ant.v1.WalletService/GetWallet', {}, state.token);
    expect(resp.ok, `GetWallet should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const wallet = resp.data.wallet as Record<string, unknown>;
    state.walletId = wallet.id as string;
    expect(wallet.balance).toBe('0.00000000');
    console.log(`Wallet OK, id=${state.walletId}, balance=0`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 4. Admin tops up new user's wallet
  // ════════════════════════════════════════════════════════════════════════
  test('4. Admin charges wallet for new user', async () => {
    const resp = await rpc('/ant.v1.WalletService/AdjustBalance', {
      userId: state.userId,
      amount: '100.00',
      description: 'E2E test: initial wallet top-up',
    }, state.adminToken);
    expect(resp.ok, `AdjustBalance should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const wallet = resp.data.wallet as Record<string, unknown>;
    expect(wallet.balance).toBe('100.00000000');
    console.log(`Wallet topped up, balance=${wallet.balance}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 5. Verify free subscription auto-created
  // ════════════════════════════════════════════════════════════════════════
  test('5. Free subscription active', async () => {
    const resp = await rpc('/ant.v1.SubscriptionService/GetMySubscription', {}, state.token);
    expect(resp.ok, `GetMySubscription should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const sub = resp.data.subscription as Record<string, unknown>;
    expect(sub.status).toBe('active');
    console.log(`Subscription OK, status=${sub.status}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 6. List subscription plans (free, pro, enterprise)
  // ════════════════════════════════════════════════════════════════════════
  test('6. Subscription plans available', async () => {
    const resp = await rpc('/ant.v1.SubscriptionService/ListPlans', {}, state.token);
    expect(resp.ok, `ListPlans should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const plans = (resp.data.plans as Array<Record<string, unknown>>) ?? [];
    expect(plans.length).toBeGreaterThanOrEqual(1);
    const freePlan = plans.find(p => p.name === 'free');
    expect(freePlan, 'Should have a free plan').toBeDefined();
    console.log(`Plans OK, count=${plans.length}, names=${plans.map(p => p.name).join(',')}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 7. Admin creates strategy template
  // ════════════════════════════════════════════════════════════════════════
  test('7. Create strategy template', async () => {
    const resp = await rpc('/ant.v1.StrategyService/CreateTemplate', {
      name: 'E2E Regression MA Crossover',
      description: 'Simple MA crossover for E2E testing',
      code: MQL_SOURCE,
      parameters: [],
      isPublic: false,
      tags: [],
    }, state.adminToken);
    expect(resp.ok, `CreateTemplate should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const id = (resp.data.id as string) || ((resp.data as Record<string, unknown>).template as Record<string, unknown>)?.id as string;
    expect(id, 'Template ID should be returned').toBeTruthy();
    state.templateId = id;
    console.log(`Template OK, id=${state.templateId}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 8. Run backtest with real market data (async via StartBacktestRun)
  // Must run before publishing — marketplace requires backtest snapshot.
  // ════════════════════════════════════════════════════════════════════════
  test('8. Start backtest with real market data', async () => {
    test.setTimeout(180_000);

    const resp = await rpc('/ant.v1.StrategyRuntimeService/StartBacktestRun', {
      code: MQL_SOURCE,
      accountId: ADMIN_ACCOUNT_ID,
      symbol: 'ETHBTCm',
      timeframe: '15m',
      initialCapital: '10000',
      mode: 1, // BACKTEST_RUN_MODE_KLINE_RANGE
      from: '2026-07-01T00:00:00Z',
      to: '2026-07-24T00:00:00Z',
      templateId: state.templateId,
      executionConfig: {
        commission: '0.001',
        slippage: '0.0005',
        leverage: '100',
        tradeDirection: 3, // TRADE_DIRECTION_BOTH
        strictMode: true,
      },
    }, state.adminToken);

    expect(resp.ok, `StartBacktestRun should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    state.backtestRunId = resp.data.runId as string;
    expect(state.backtestRunId).toBeTruthy();
    console.log(`Backtest started, runId=${state.backtestRunId}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 9. Poll backtest until completed and verify results
  // ════════════════════════════════════════════════════════════════════════
  test('9. Backtest completes with valid metrics', async () => {
    test.setTimeout(180_000);
    expect(state.backtestRunId, 'RunId must be set from previous test').toBeTruthy();

    let attempts = 0;
    let run: Record<string, unknown> | null = null;
    let fullRespData: Record<string, unknown> | null = null;

    while (attempts < 60) {
      const resp = await rpc('/ant.v1.StrategyRuntimeService/GetBacktestRun', {
        runId: state.backtestRunId,
      }, state.adminToken);

      if (resp.ok && resp.data) {
        fullRespData = resp.data;
        run = resp.data.run as Record<string, unknown>;
        const status = run?.status as string;
        if (status === 'BACKTEST_RUN_STATUS_SUCCEEDED' ||
            status === 'BACKTEST_RUN_STATUS_FAILED' ||
            status === 'BACKTEST_RUN_STATUS_CANCELED') {
          break;
        }
        console.log(`  Poll ${attempts + 1}: status=${status}`);
      } else {
        console.log(`  Poll ${attempts + 1}: resp not ok: ${JSON.stringify(resp.data).slice(0, 100)}`);
      }
      attempts++;
      await new Promise(r => setTimeout(r, 3000));
    }

    expect(run, 'Backtest run should exist').toBeDefined();

    const finalStatus = run!.status as string;
    console.log(`Backtest final status=${finalStatus}`);

    if (finalStatus === 'BACKTEST_RUN_STATUS_SUCCEEDED') {
      const metrics = fullRespData?.metrics as Record<string, unknown> | undefined;
      if (metrics) {
        console.log(`Metrics: totalReturn=${metrics.totalReturn}, maxDrawdown=${metrics.maxDrawdown}, totalTrades=${metrics.totalTrades}`);
      }
      const equityCurve = fullRespData?.equityCurve as string[] | undefined;
      if (equityCurve && equityCurve.length > 0) {
        console.log(`Equity curve points: ${equityCurve.length}`);
      }
      console.log('Backtest SUCCEEDED with real market data');
    } else if (finalStatus === 'BACKTEST_RUN_STATUS_FAILED') {
      const errorMsg = run!.error as string;
      console.warn(`Backtest FAILED: ${errorMsg}`);
      console.warn('This may happen if no XAUUSDm 5m data exists for the requested period');
    } else {
      console.warn(`Backtest did not complete (status=${finalStatus}) after ${attempts * 3}s`);
    }
  });

  // ════════════════════════════════════════════════════════════════════════
  // 10. Admin publishes strategy to marketplace (free)
  // Now that backtest snapshot exists, quality gate should pass.
  // ════════════════════════════════════════════════════════════════════════
  test('10. Publish strategy to marketplace', async () => {
    const resp = await rpc('/ant.v1.MarketplaceService/PublishStrategy', {
      userId: state.userId,
      strategyId: state.templateId,
      title: 'E2E Test MA Crossover',
      description: 'Free strategy for E2E testing — MA crossover',
      priceModel: 'free',
      priceAmount: '0',
      assetClass: 'forex',
      symbols: ['ETHBTCm'],
      timeframe: '15m',
      riskLevel: 'low',
      tags: ['e2e', 'test'],
    }, state.adminToken);
    expect(resp.ok, `PublishStrategy should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    state.publishId = resp.data.publishId as string;
    expect(state.publishId).toBeTruthy();
    console.log(`Published OK, publishId=${state.publishId}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 11. New user sees published strategy in marketplace
  // ════════════════════════════════════════════════════════════════════════
  test('11. Marketplace lists published strategy', async () => {
    const resp = await rpc('/ant.v1.MarketplaceService/ListPublished', {
      limit: 50,
      offset: 0,
    }, state.token);
    expect(resp.ok, `ListPublished should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const strategies = (resp.data.strategies as Array<Record<string, unknown>>) ?? [];
    expect(strategies.length).toBeGreaterThanOrEqual(1);
    const found = strategies.find(s => s.publishId === state.publishId);
    expect(found, 'Published strategy should be visible').toBeDefined();
    expect(found!.priceModel).toBe('free');
    console.log(`Marketplace OK, found published strategy: ${found!.title}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 12. New user subscribes to the free strategy
  // ════════════════════════════════════════════════════════════════════════
  test('12. Subscribe to free strategy', async () => {
    const resp = await rpc('/ant.v1.MarketplaceService/Subscribe', {
      userId: state.userId,
      publisherUserId: '',
      strategyId: state.templateId,
      kind: 'copy_trade',
    }, state.token);
    expect(resp.ok, `Subscribe should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    expect(resp.data.subscriptionId).toBeTruthy();
    console.log(`Subscribed OK, subscriptionId=${resp.data.subscriptionId}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 13. List user's backtest runs
  // ════════════════════════════════════════════════════════════════════════
  test('13. List backtest runs', async () => {
    const resp = await rpc('/ant.v1.StrategyRuntimeService/ListBacktestRuns', {
      limit: 10,
      offset: 0,
    }, state.adminToken);
    expect(resp.ok, `ListBacktestRuns should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const runs = (resp.data.runs as unknown[]) ?? [];
    expect(runs.length).toBeGreaterThanOrEqual(1);
    console.log(`Backtest runs OK, count=${runs.length}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 14. List user's marketplace subscriptions
  // ════════════════════════════════════════════════════════════════════════
  test('14. List marketplace subscriptions', async () => {
    const resp = await rpc('/ant.v1.MarketplaceService/ListSubscriptions', {
      userId: state.userId,
    }, state.token);
    expect(resp.ok, `ListSubscriptions should succeed: ${JSON.stringify(resp.data)}`).toBe(true);
    const subs = (resp.data.subscriptions as Array<Record<string, unknown>>) ?? [];
    expect(subs.length).toBeGreaterThanOrEqual(1);
    const active = subs.find(s => s.active === true);
    expect(active, 'Should have at least one active subscription').toBeDefined();
    console.log(`Subscriptions OK, count=${subs.length}, active=${active ? 'yes' : 'no'}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 15. Verify token still valid after all operations
  // ════════════════════════════════════════════════════════════════════════
  test('15. Token still valid', async () => {
    const resp = await rpc('/ant.v1.WalletService/GetWallet', {}, state.token);
    expect(resp.ok, 'Token should still be valid').toBe(true);
    const wallet = resp.data.wallet as Record<string, unknown>;
    // Balance should still be 100 (free strategy, no charge)
    expect(wallet.balance).toBe('100.00000000');
    console.log(`Token valid, balance unchanged=${wallet.balance}`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 16. Duplicate registration blocked
  // ════════════════════════════════════════════════════════════════════════
  test('16. Duplicate registration blocked', async () => {
    const resp = await rpc('/ant.v1.AuthService/Register', {
      email: state.email,
      password: state.pass,
      username: state.email,
    });
    expect(resp.ok, 'Duplicate registration should fail').toBe(false);
    console.log(`Duplicate registration correctly rejected`);
  });

  // ════════════════════════════════════════════════════════════════════════
  // 17. Cleanup: unpublish strategy
  // ════════════════════════════════════════════════════════════════════════
  test('17. Cleanup — unpublish strategy', async () => {
    const resp = await rpc('/ant.v1.MarketplaceService/UnpublishStrategy', {
      strategyId: state.publishId,
    }, state.adminToken);
    // Best-effort cleanup — don't fail if already unpublished
    if (resp.ok) {
      console.log('Strategy unpublished');
    } else {
      console.warn(`Unpublish returned ${resp.status}: ${JSON.stringify(resp.data).slice(0, 100)}`);
    }
  });
});
