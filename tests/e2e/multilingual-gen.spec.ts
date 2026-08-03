import { test, expect, type Page } from '@playwright/test';

// ── Phase 1.2 E2E: multilingual strategy generation → backtest ──
// For each enabled locale (en, zh-cn), 3 test cases:
//   1. Describe → Generate → Backtest succeeds
//   2. Different strategy family → Generate → Backtest succeeds
//   3. Edge case description → Generate → Backtest succeeds

const BASE_URL = 'http://localhost:8022';
const TEST_USER = 'admin@1.com';
const TEST_PASSWORD = '12345678';
const SYMBOL = 'BTCUSDm';
const TIMEFRAME = '15m';
const ACCOUNT_ID = '95172262';

interface GenerateResult {
  success: boolean;
  phase: string;
  code: string;
  backtestResult?: {
    totalReturn: number;
    maxDrawdown: number;
    sharpeRatio: number;
    winRate: number;
    totalTrades: number;
  };
  error?: string;
}

async function loginViaAPI(): Promise<string> {
  const resp = await fetch(`${BASE_URL}/ant.v1.AuthService/Login`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ login: TEST_USER, password: TEST_PASSWORD }),
  });
  if (!resp.ok) throw new Error(`Login API failed: ${resp.status}`);
  const data = await resp.json();
  return data.accessToken;
}

// ConnectRPC server-streaming via JSON: POST with application/connect+json,
// response is newline-delimited JSON chunks.
async function generateStrategy(
  token: string,
  message: string,
  locale: string,
): Promise<GenerateResult> {
  const resp = await fetch(`${BASE_URL}/ant.v1.AgentGatewayService/GenerateStrategy`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'Authorization': `Bearer ${token}`,
      'Connect-Protocol-Version': '1',
    },
    body: JSON.stringify({
      message,
      symbol: SYMBOL,
      timeframe: TIMEFRAME,
      planMode: 'generate',
      locale,
      accountId: ACCOUNT_ID,
      backtestConfig: {
        symbol: SYMBOL,
        timeframe: TIMEFRAME,
        startDateMs: String(BigInt(Date.now() - 90 * 24 * 60 * 60 * 1000) * 1_000_000n),
        endDateMs: String(BigInt(Date.now()) * 1_000_000n),
        initialCapital: '10000',
        commission: '0.001',
        slippage: '0.0005',
        leverage: '1',
        strictMode: false,
      },
    }),
  });

  if (!resp.ok) {
    const errText = await resp.text().catch(() => 'unknown');
    return { success: false, phase: 'init', code: '', error: `HTTP ${resp.status}: ${errText}` };
  }

  const result: GenerateResult = { success: false, phase: '', code: '' };
  const reader = resp.body?.getReader();
  if (!reader) return { success: false, phase: 'init', code: '', error: 'No response body' };

  const decoder = new TextDecoder();
  let buffer = '';

  while (true) {
    const { done, value } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });

    // ConnectRPC streaming: newline-delimited JSON objects
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) continue;
      try {
        const chunk = JSON.parse(trimmed);
        if (chunk.phase) result.phase = chunk.phase;
        if (chunk.delta) result.code += chunk.delta;
        if (chunk.pythonSource) result.code = chunk.pythonSource;
        if (chunk.phase === 'done') {
          result.success = true;
          if (chunk.result) {
            result.backtestResult = {
              totalReturn: chunk.result.totalReturn ?? 0,
              maxDrawdown: chunk.result.maxDrawdown ?? 0,
              sharpeRatio: chunk.result.sharpeRatio ?? 0,
              winRate: chunk.result.winRate ?? 0,
              totalTrades: chunk.result.totalTrades ?? 0,
            };
          }
        }
        if (chunk.phase === 'error' || chunk.compileError || chunk.backtestError) {
          result.error = chunk.compileError || chunk.backtestError || chunk.error || 'Unknown error';
        }
      } catch {
        // skip non-JSON lines
      }
    }
  }

  return result;
}

// ── English test cases ──

test.describe('Phase 1.2 E2E — English locale', () => {
  test('EN-1: RSI oversold bounce strategy', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      'Create a strategy that buys when RSI(14) crosses below 30 and sells when RSI crosses above 70. Use a stop loss of 50 pips and take profit of 100 pips.',
      'en',
    );
    console.log(`EN-1: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`EN-1 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });

  test('EN-2: Moving average crossover strategy', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      'Build a moving average crossover strategy: fast EMA(9), slow EMA(21). Go long when fast crosses above slow, go short when fast crosses below. Add ATR-based stop loss.',
      'en',
    );
    console.log(`EN-2: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`EN-2 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });

  test('EN-3: Bollinger Bands breakout strategy', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      'Design a Bollinger Bands(20, 2) breakout strategy: buy when price closes above upper band, sell when price closes below lower band. Include trailing stop based on ATR(14).',
      'en',
    );
    console.log(`EN-3: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`EN-3 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });
});

// ── Chinese test cases ──

test.describe('Phase 1.2 E2E — Chinese locale (zh-cn)', () => {
  test('ZH-1: RSI 超卖反弹策略', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      '创建一个RSI(14)超卖反弹策略：当RSI跌破30时做多，当RSI升破70时平仓。设置50点止损和100点止盈。',
      'zh-cn',
    );
    console.log(`ZH-1: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`ZH-1 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });

  test('ZH-2: 均线交叉策略', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      '构建一个均线交叉策略：快线EMA(9)，慢线EMA(21)。快线上穿慢线时做多，快线下穿慢线时做空。加入基于ATR的动态止损。',
      'zh-cn',
    );
    console.log(`ZH-2: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`ZH-2 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });

  test('ZH-3: 布林带突破策略', async ({ page }: { page: Page }) => {
    const token = await loginViaAPI();
    const result = await generateStrategy(
      token,
      '设计一个布林带(20,2)突破策略：价格收盘突破上轨时做多，收盘跌破下轨时做空。加入基于ATR(14)的追踪止损。',
      'zh-cn',
    );
    console.log(`ZH-3: phase=${result.phase}, codeLen=${result.code.length}, success=${result.success}`);
    if (result.error) console.log(`ZH-3 error: ${result.error}`);
    expect(result.success).toBe(true);
    expect(result.code.length).toBeGreaterThan(0);
  });
});
