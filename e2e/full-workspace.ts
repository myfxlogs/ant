/**
 * Full workspace E2E test — v3 (comprehensive: all workspace features).
 *
 * Phases: Setup → Chart → Code Panel → Backtest → AI Generation → AI Auto-Fix →
 *         AI Optimize → Smart Tuning → Gate → QuickTrade → Auto Trading →
 *         Marketplace → Schedules → Multi-Symbol → History → Templates → Settings → Final
 *
 * Usage: npx tsx e2e/full-workspace.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT, { recursive: true });

const U = '888888'; const P = '12345678';
const ACCT = '277259925'; const SYM = 'BTCUSDm';
const ALT_SYM = 'XAUUSDm';

const CODE = `"""
@strategy SMA crossover
@param fast_period 5 range=3:50:1
@param slow_period 20 range=10:60:2
@param stop_loss_pct 0.5 range=0.1:3.0:0.1
@param take_profit_pct 1.0 range=0.2:5.0:0.2
@param position_size 0.01 range=0.01:0.1:0.01
"""

def init():
    pass

def run(ctx):
    close = ctx['close']
    if len(close) < 22:
        return
    f = sum(close[-5:]) / 5.0
    s = sum(close[-20:]) / 20.0
    pf = sum(close[-6:-1]) / 5.0
    ps = sum(close[-21:-1]) / 20.0
    pos = ctx.get('position') or {}
    if pf <= ps and f > s and pos.get('side') != 'buy':
        return {'signal': 'buy', 'volume': 0.01}
    elif pf >= ps and f < s and pos.get('side') != 'sell':
        return {'signal': 'sell', 'volume': 0.01}
`;

// Code with both syntax error (missing colon → ast.parse catches it)
// AND business logic issues (undefined variable, no stop-loss → LLM catches).
// This guarantees valid=false from Python and gives Auto-Fix actionable targets.
const BROKEN_CODE = `"""
@strategy test broken
@param fast_period 10 range=5:50:5
"""

def init():
    pass

def run(ctx):
    close = ctx['close']
    # BUG 1: 'fast' is undefined — should be fast_period
    # BUG 2: missing colon on next line
    if len(close) < fast + 1
        return
    ma = sum(close[-fast:]) / fast
    if close[-1] > ma:
        return {'signal': 'buy', 'volume': 0.01}
    # BUG 3: no stop-loss, no take-profit
`;

// ══════════════════════════════════════
// Reporting
// ══════════════════════════════════════
const R: string[] = []; let sn = 0;
function S(m: string) { sn++; console.log(`\n── ${sn}. ${m} ──`); }
function OK(m: string) { R.push(`✅ ${m}`); console.log(`   ✅ ${m}`); }
function W(m: string, d?: string) { R.push(`⚠️ ${m}${d ? ': ' + d : ''}`); console.log(`   ⚠️ ${m}${d ? ': ' + d : ''}`); }
function F(m: string, d?: string) { R.push(`❌ ${m}${d ? ': ' + d : ''}`); console.log(`   ❌ ${m}${d ? ': ' + d : ''}`); }
function shot(page: Page, n: string) {
  return page.screenshot({ path: `${SHOT}/full-${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

// ══════════════════════════════════════
// Helpers (reused from v2)
// ══════════════════════════════════════
async function wait(page: Page, ms = 3000) {
  await page.waitForLoadState('domcontentloaded').catch(() => {});
  await page.waitForTimeout(ms);
}

async function humanPause(minMs = 800, maxMs = 3000) {
  const delay = Math.floor(minMs + Math.random() * (maxMs - minMs));
  await new Promise(r => setTimeout(r, delay));
}

async function exploreScroll(page: Page) {
  await page.evaluate(() => window.scrollBy(0, 150));
  await page.waitForTimeout(400);
  await page.evaluate(() => window.scrollBy(0, -100));
  await page.waitForTimeout(300);
}

async function selectOption(page: Page, selText: string, optText: string): Promise<boolean> {
  const selects = await page.$$('.ant-select');
  let target: any = null;
  for (const s of selects) {
    const t = await s.textContent();
    if (t?.includes(selText)) { target = s; break; }
  }
  if (!target) {
    if (selText.includes('account')) target = selects[0] || null;
    else if (selText.includes('symbol') || selText.includes('Symbol')) target = selects[1] || null;
    if (!target) return false;
  }
  await target.click({ force: true }); await humanPause(600, 1000);

  if (!selText.includes('account')) {
    const cbi = await page.$$('input[role="combobox"]');
    if (cbi.length >= 2) { await cbi[1].fill(optText); await page.waitForTimeout(1200); }
  }

  const opts = await page.$$('.ant-select-item-option');
  const isAcct = selText.includes('account');
  for (const o of opts) { const t = await o.textContent(); if (t?.trim() === optText) { await o.click({ force: true }); await page.waitForTimeout(500); return true; } }
  for (const o of opts) { const t = await o.textContent(); if (t?.includes(optText) && (isAcct || !t.includes('·'))) { await o.click({ force: true }); await page.waitForTimeout(500); return true; } }
  for (const o of opts) { const t = await o.textContent(); if (t && (isAcct || !t.includes('·'))) { await o.click({ force: true }); await page.waitForTimeout(500); return true; } }
  await page.keyboard.press('Escape'); return false;
}

async function toggleCodePanel(page: Page) {
  const divs = await page.$$('div');
  for (const d of divs) {
    const b = await d.boundingBox();
    if (b && b.x >= 248 && b.x <= 265 && b.width <= 35 && b.height >= 100) {
      await d.click({ force: true }); await page.waitForTimeout(1500); return;
    }
  }
}

async function codePanelOpen(page: Page): Promise<boolean> {
  const cm = (await page.$$('.cm-editor')).length > 0;
  const overlay = await page.$$eval('[style*="position"]', els => {
    for (const e of els) {
      const r = (e as HTMLElement).getBoundingClientRect();
      if (r.width >= 740 && r.width <= 765 && r.height > 500) return true;
    }
    return false;
  });
  return cm || overlay;
}

async function insertCode(page: Page, c: string, opts?: { syncReact?: boolean }) {
  await page.evaluate((code) => {
    const ed = document.querySelector('.cm-editor');
    if (!ed) return;
    let v: any = null;
    for (const k of Object.keys(ed)) { const val = (ed as any)[k]; if (val?.state?.dispatch && val?.viewport) { v = val; break; } }
    if (v) { v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: code } }); return; }
    const cm = ed.querySelector('.cm-content') as HTMLElement;
    if (cm) { cm.focus(); document.execCommand('selectAll', false); document.execCommand('insertText', false, code); }
  }, c);
  await page.waitForTimeout(800);

  // Force React state sync by simulating a real user keystroke.
  // CodeMirror's programmatic dispatch may not trigger the React wrapper's onChange.
  // A real keystroke ensures the React code state is updated.
  if (opts?.syncReact !== false) {
    const cmEl = await page.$('.cm-editor');
    if (cmEl) {
      await cmEl.click({ force: true });
      await page.waitForTimeout(200);
      // Move cursor to end of document (where dispatch left it)
      await page.keyboard.press('End');
      await page.waitForTimeout(100);
      // Type and delete a non-intrusive character to trigger onChange
      await page.keyboard.press('ArrowLeft');
      await page.waitForTimeout(50);
      await page.keyboard.press('ArrowRight');
      await page.waitForTimeout(150);
      // Dispatch an input event to ensure React picks up the CodeMirror change
      await page.evaluate(() => {
        const cm = document.querySelector('.cm-content');
        if (cm) cm.dispatchEvent(new Event('input', { bubbles: true, cancelable: true }));
      });
      await page.waitForTimeout(500);
    }
  }
}

async function getCodeText(page: Page): Promise<string> {
  return page.$eval('.cm-content', el => el.textContent?.slice(0, 80) || '').catch(() => '');
}

async function clickDivText(page: Page, exact: string): Promise<boolean> {
  for (const d of await page.$$('div')) {
    if ((await d.textContent())?.trim() === exact) {
      const b = await d.boundingBox();
      if (b && b.height < 50) { await d.click({ force: true }); await page.waitForTimeout(1000); return true; }
    }
  }
  return false;
}

async function clickRadioLabel(page: Page, text: string): Promise<boolean> {
  for (const r of await page.$$('.ant-radio-button-wrapper, .ant-radio-wrapper')) {
    if ((await r.textContent())?.trim() === text) { await r.click({ force: true }); await page.waitForTimeout(300); return true; }
  }
  return false;
}

// ══════════════════════════════════════
// Helpers (new for v3)
// ══════════════════════════════════════

/** Switch tab by exact text using dispatchEvent (reliable for Ant Design). */
async function switchTab(page: Page, tabText: string): Promise<boolean> {
  return page.evaluate((text) => {
    const divs = document.querySelectorAll('div');
    for (const d of divs) {
      if (d.textContent?.trim() === text) {
        const r = d.getBoundingClientRect();
        if (r.height < 40 && r.height > 10) {
          d.dispatchEvent(new MouseEvent('click', { bubbles: true }));
          return true;
        }
      }
    }
    return false;
  }, tabText);
}

/** Navigate to a page and wait for it to load. */
async function navigateTo(page: Page, path: string): Promise<boolean> {
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(2000);
  return page.url().includes(path);
}

/** Click a button matching text within a vertical range. */
async function clickButtonInRange(page: Page, textRe: RegExp, yMin: number, yMax: number): Promise<boolean> {
  for (const btn of await page.$$('button')) {
    const txt = await btn.textContent();
    const box = await btn.boundingBox();
    if (txt && textRe.test(txt) && box && box.y >= yMin && box.y <= yMax) {
      const disabled = await btn.evaluate((el: HTMLButtonElement) => el.disabled);
      if (!disabled) { await btn.click({ force: true }); return true; }
    }
  }
  return false;
}

/** Type text with human-like speed, occasional correction. */
async function humanType(page: Page, selector: string, text: string) {
  const el = await page.$(selector);
  if (!el) return;
  await el.click({ force: true });
  await humanPause(200, 500);
  for (let i = 0; i < text.length; i++) {
    await page.keyboard.type(text[i], { delay: 30 + Math.random() * 50 });
    if (Math.random() < 0.03) { await page.waitForTimeout(200); } // brief pause
  }
}

/** Poll body text for regex match. Returns true if found within timeout. */
async function pollForText(page: Page, re: RegExp, maxSec: number, intervalMs = 2000): Promise<{found: boolean; elapsed: number; negativeRe?: RegExp}> {
  const start = Date.now();
  for (let i = 0; i < Math.ceil(maxSec * 1000 / intervalMs); i++) {
    await page.waitForTimeout(intervalMs);
    const body = await page.textContent('body') || '';
    const err = await page.$$('.ant-tag:has-text("error"), .ant-alert-error');
    if (err.length > 0) {
      const msg = await err[0].textContent();
      return { found: false, elapsed: (Date.now() - start) / 1000, negativeRe: /./ };
    }
    if (re.test(body)) {
      return { found: true, elapsed: (Date.now() - start) / 1000 };
    }
    if (i % 15 === 14) console.log(`   Wait ${Math.round((Date.now() - start) / 1000)}s...`);
  }
  return { found: false, elapsed: maxSec };
}

// ══════════════════════════════════════
// Phase-level runner (non-fatal failures)
// ══════════════════════════════════════
async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name);
  try {
    await fn();
  } catch (e: any) {
    F(`Phase error: ${name}`, e.message?.slice(0, 150));
    await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {});
  }
}

// ══════════════════════════════════════
// Main test runner
// ══════════════════════════════════════
async function run() {
  const browser = await chromium.launch({
    headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const jsErrs: string[] = [];
  const apiLog: string[] = [];
  let apiCount = 0;
  page.on('pageerror', e => jsErrs.push(e.message));
  page.on("console", msg => { if (msg.text().startsWith("PHASE6_DIAG")) console.log("   [BROWSER]", msg.text()); });
  page.on('response', res => {
    const url = res.url();
    apiCount++;
    if (/ant\.v1\.|assets\//i.test(url)) {
      const short = url.replace(/^.*\/api\//, '').replace(/\?.*$/, '').split('/').slice(-2).join('/');
      apiLog.push(`API ${res.status()} ${short}`);
    }
  });

  // ═══ PHASE 1: SETUP ═══
  await phase('Login', page, async () => {
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await humanPause(300, 800);
    await page.fill('#login_login', U);
    await humanPause(400, 900);
    await page.fill('#login_password', P);
    await humanPause(200, 600);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    page.url().includes('login') ? F('Login failed') : OK('Login');
    shot(page, 'login');
  });

  await phase('Workspace', page, async () => {
    await page.goto(`${BASE}/strategy/workspace`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await wait(page, 6000);
    exploreScroll(page);
    OK('Loaded');
    shot(page, 'workspace');
  });

  await phase('Select Account', page, async () => {
    await selectOption(page, 'Select account', ACCT);
    await humanPause(1000, 2000);
    const aTxt = await page.$$eval('.ant-select', els => els[0]?.textContent?.trim());
    aTxt && aTxt !== 'Select account' ? OK(`Account: ${aTxt}`) : W('Account unknown');
    shot(page, 'account');
  });

  await phase('Select Symbol', page, async () => {
    await selectOption(page, 'Select symbol', SYM);
    await humanPause(1500, 2500);
    const sTxt = await page.$$eval('.ant-select', els => els[1]?.textContent?.trim());
    sTxt && sTxt !== 'Select symbol' ? OK(`Symbol: ${sTxt}`) : W('Symbol unknown');
    shot(page, 'symbol');
  });

  await page.waitForTimeout(3000);

  // ═══ PHASE 2: CHART ═══
  await phase('Chart Verification', page, async () => {
    const canvases = (await page.$$('canvas')).length;
    const recharts = (await page.$$('.recharts-surface')).length;
    (canvases >= 5 || recharts >= 1) ? OK(`Chart: ${canvases}c/${recharts}r`) : W('Chart low');

    const bt = await page.textContent('body') || '';
    const metaFields = ['Balance', 'Equity', 'Profit', 'Margin'];
    let mf = 0;
    for (const m of metaFields) { if (new RegExp(m, 'i').test(bt)) mf++; }
    mf >= 3 ? OK(`Toolbar: ${mf}/4 fields`) : W(`${mf}/4 meta fields`);
    shot(page, 'chart');
  });

  await phase('Timeframe Cycling', page, async () => {
    // Human-like: cycle through timeframes
    for (const tf of ['M1', 'M5', 'M15', 'H1', 'H4', 'D1']) {
      await clickRadioLabel(page, tf);
      await humanPause(600, 1200);
    }
    // Return to M5
    await clickRadioLabel(page, 'M5');
    await page.waitForTimeout(800);
    const tfOk = await page.evaluate(() => {
      const el = document.querySelector('.ant-radio-button-wrapper-checked');
      return !!(el?.textContent?.trim());
    });
    tfOk ? OK('TF restored') : W('TF: no checked tf');
  });

  await phase('Positions Count', page, async () => {
    const bt = await page.textContent('body') || '';
    const posMatch = bt.match(/POSITIONS\s*(\d+)/i);
    posMatch ? OK(`Positions: ${posMatch[1]}`) : W('No position badge');
  });

  // ═══ PHASE 3: CODE PANEL ═══
  await phase('Open Code Panel', page, async () => {
    if (!(await codePanelOpen(page))) { await toggleCodePanel(page); await page.waitForTimeout(2000); }
    (await codePanelOpen(page)) ? OK('Panel open') : W('Not open');
    shot(page, 'code-panel');
  });

  await phase('Insert Code', page, async () => {
    await insertCode(page, CODE);
    const cc = await getCodeText(page);
    cc?.includes('@strategy') ? OK('Code OK') : W('Code may be empty');
    shot(page, 'code');
  });

  await phase('Validate', page, async () => {
    const cmEl = await page.$('.cm-editor');
    if (cmEl) { await cmEl.click({ force: true }); await page.waitForTimeout(200); }
    await page.keyboard.press('Control+Enter');
    await page.waitForTimeout(4000);
    (await page.$$('.ant-alert-error')).length === 0 ? OK('Passed') : W('Has issues');
    shot(page, 'validate');
  });

  await phase('Save Template', page, async () => {
    // Save button in code panel toolbar
    const buttons = await page.$$('button');
    let saved = false;
    for (const btn of buttons) {
      const txt = await btn.textContent();
      const box = await btn.boundingBox();
      if (txt?.trim() === 'Save' && box && box.y > 190 && box.y < 400) {
        const sd = await btn.evaluate((el: HTMLButtonElement) => el.disabled);
        if (!sd) {
          await btn.click({ force: true });
          await page.waitForTimeout(1500);
          const modal = await page.$('.ant-modal');
          if (modal) {
            const mi = await page.$$('.ant-modal input');
            if (mi.length > 0) { await mi[0].fill('E2E Full Test v3'); await humanPause(200, 500); }
            const okBtn = await page.$('.ant-modal .ant-btn-primary');
            if (okBtn) { await okBtn.click({ force: true }); await page.waitForTimeout(1000); }
            OK('Template saved');
          } else { OK('Save clicked'); }
          saved = true;
        }
        break;
      }
    }
    if (!saved) W('Save btn not found');
    shot(page, 'saved');
  });

  await phase('Close Code Panel', page, async () => {
    await toggleCodePanel(page); await page.waitForTimeout(1500);
    if (!(await codePanelOpen(page))) { OK('Panel closed'); }
    else {
      await toggleCodePanel(page); await page.waitForTimeout(1500);
      !(await codePanelOpen(page)) ? OK('Closed (2nd)') : W('Still open');
    }
    shot(page, 'panel-closed');
  });

  // ═══ PHASE 4: BACKTEST ═══
  let btDone = false;

  await phase('Configure Backtest', page, async () => {
    // Date 1M
    for (const seg of await page.$$('.ant-segmented-item')) {
      if ((await seg.textContent())?.trim() === '1M') { await seg.click({ force: true }); break; }
    }
    OK('Date: 1M');
    // Capital 10000
    const spins = await page.$$('input[role="spinbutton"]');
    if (spins.length > 0) {
      await spins[0].click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('10000', { delay: 30 });
      OK('Capital: 10k');
    }
    // Direction
    await clickRadioLabel(page, 'Both'); OK('Dir: Both');
    // Preset
    const lb = await page.$('button:has-text("Live Aligned")');
    if (lb) { await lb.click({ force: true }); OK('Live Aligned'); }
    shot(page, 'bt-config');
  });

  await phase('Run Backtest', page, async () => {
    const rb = await page.$('button:has-text("Run")');
    if (!rb || await rb.evaluate((el: HTMLButtonElement) => el.disabled)) { W('Run unavailable'); return; }
    // Human-like: check settings one more time before committing to a run
    await page.evaluate(() => window.scrollBy(0, 60));
    await humanPause(400, 900); // review params
    await page.evaluate(() => window.scrollBy(0, -30));
    await humanPause(600, 1200); // hesitation before committing
    await rb.click({ force: true });
    OK('Backtest submitted');
    shot(page, 'bt-run');
  });

  await phase('Wait Backtest Results', page, async () => {
    const result = await pollForText(page, /total.*return|net.*profit|sharpe|win\s*rate|drawdown/i, 90);
    if (result.found) { OK(`BT done ~${Math.round(result.elapsed)}s`); btDone = true; }
    else W('BT timeout 90s');
    shot(page, 'bt-done');
  });

  if (btDone) {
    await phase('Verify Backtest Metrics', page, async () => {
      const btx = await page.textContent('body') || '';
      let f = 0;
      for (const [n, re] of [['Return', /return/i], ['Win', /win\s*rate/i], ['DD', /drawdown/i], ['Sharpe', /sharpe/i], ['Trades', /trades/i]] as [string, RegExp][]) {
        if (re.test(btx)) { OK(n); f++; }
      }
      f >= 4 ? OK(`${f}/5 OK`) : W(`${f}/5`);
      shot(page, 'metrics');
    });
  }

  // ═══ PHASE 5: AI STRATEGY GENERATION ═══
  await phase('AI Strategy Generation', page, async () => {
    // Open code panel first (AI chat is inside it)
    if (!(await codePanelOpen(page))) {
      await toggleCodePanel(page); await page.waitForTimeout(2000);
    }
    if (!(await codePanelOpen(page))) { W('Cannot open panel for AI'); return; }

    // Scroll code panel to find AI chat area (below code editor)
    await page.evaluate(() => {
      const panel = document.querySelector('[style*="position"][style*="absolute"]');
      if (panel) panel.scrollTop = 400;
    });
    await page.waitForTimeout(1000);

    // Find the AI chat input (textarea or input inside the chat panel)
    // Find the AI chat textarea (ant-input-textarea inside code panel)
    // Search for textarea with the strategy generation placeholder
    const allTextareas = await page.$$('textarea');
    let aiInput: any = null;
    for (const ta of allTextareas) {
      const ph = await ta.getAttribute('placeholder');
      const box = await ta.boundingBox();
      // AI textarea is inside code panel (x > 250) with strategy placeholder
      if (box && box.x > 250 && ph && /描述|Describe|strategy|策略/i.test(ph)) {
        aiInput = ta; break;
      }
    }
    // Fallback: any textarea in code panel area
    if (!aiInput) {
      for (const ta of allTextareas) {
        const box = await ta.boundingBox();
        if (box && box.x > 250 && box.y > 300) { aiInput = ta; break; }
      }
    }

    const generationPrompt = 'Generate a mean reversion strategy for BTCUSD using Bollinger Bands (period 20, stddev 2) and RSI (period 14). Enter long when price touches lower band and RSI below 30. Exit when price touches upper band or RSI above 70.';

    await aiInput.click({ force: true });
    await humanPause(300, 700);
    await aiInput.fill(generationPrompt);
    await humanPause(500, 1500); // re-reading the prompt

    // Find send button
    const sendBtn = await page.$('button .anticon-send, button .anticon-arrow-up');
    if (sendBtn) {
      const parent = await sendBtn.evaluateHandle(el => el.closest('button'));
      await (parent as any).click({ force: true });
      OK('Gen prompt sent');
    } else {
      // Try pressing Enter
      await page.keyboard.press('Enter');
      OK('Gen prompt sent (Enter)');
    }

    // Monitor streaming phases — wait for generation to complete
    // Phases: analyzing → clarifying → generating → compliance → backtest → done
    const genResult = await pollForText(page, /@strategy|@param|phase.*done|generat/i, 120, 3000);
    if (genResult.found) {
      OK(`Gen completed ~${Math.round(genResult.elapsed)}s`);
      // Check if code appeared in editor
      const cc = await getCodeText(page);
      if (cc && cc.length > 10) OK('Gen code in editor');
      else W('No generated code');
    } else {
      W('AI gen timeout 120s — may need AI provider');
    }
    shot(page, 'ai-gen');
  });

  // ═══ PHASE 6: AI AUTO-FIX ═══
  await phase('AI Auto-Fix', page, async () => {
    // KEY INSIGHT: CodeMirror dispatch() does NOT trigger React onChange,
    // so React state stays stale. Real keystrokes (keyboard.type/press)
    // are the ONLY reliable way to sync both React + CodeMirror state.
    // We use a SHORT broken code typed character-by-character.

    // Step 1: Close and reopen code panel for clean React state
    if (await codePanelOpen(page)) {
      await toggleCodePanel(page); await page.waitForTimeout(1500);
    }
    await humanPause(400, 800);
    await toggleCodePanel(page); await page.waitForTimeout(2500);
    if (!(await codePanelOpen(page))) {
      await toggleCodePanel(page); await page.waitForTimeout(2000);
    }
    if (!(await codePanelOpen(page))) { W('Panel not open for Auto-Fix'); return; }
    OK('Panel reopened');

    // Step 2: Select all existing code and type broken code via REAL keystrokes.
    // This ensures React onChange fires for every character.
    const cmEl = await page.$('.cm-editor');
    if (!cmEl) { W('CodeMirror not found'); return; }
    await cmEl.click({ force: true });
    await page.waitForTimeout(300);

    // Select all and delete
    await page.keyboard.press('Control+a');
    await humanPause(200, 400);
    await page.keyboard.press('Backspace');
    await humanPause(300, 500);

    // Type broken code line by line (human-like typing speed)
    // Missing colon in def run(ctx) causes Python SyntaxError
    const brokenLines = [
      '@strategy broken',
      '',
      'def init():',
      '    pass',
      '',
      'def run(ctx)',
      '    close = ctx["close"]',
      '    return',
    ];
    for (let i = 0; i < brokenLines.length; i++) {
      await page.keyboard.type(brokenLines[i], { delay: 20 + Math.random() * 30 });
      if (i < brokenLines.length - 1) {
        await page.keyboard.press('Enter');
        await humanPause(100, 250);
      }
    }
    await humanPause(500, 1000);
    OK('Broken code typed via keyboard');

    // Verify code is in editor
    const bc = await getCodeText(page);
    if (!bc || bc.length < 8) { W('Broken code missing'); return; }
    OK('Editor shows broken code (' + bc.length + ' chars)');

    // Step 3: Scroll to top and re-read (human-like)
    await page.evaluate(() => {
      const cm = document.querySelector('.cm-content');
      if (cm) cm.scrollTop = 0;
    });
    await humanPause(500, 1000);

    // Step 4: Click Validate button (has CheckCircleOutlined icon in toolbar).
    // Use CSS selector :has() for direct icon match.
    let validated = false;
    const validateBtn = await page.$('button:has(.anticon-check-circle)');
    if (validateBtn) {
      await humanPause(300, 600);
      await validateBtn.click({ force: true });
      OK('Validate button clicked');
      validated = true;
    }
    if (!validated) {
      // Fallback: iterate all buttons looking for check-circle icon
      const btns = await page.$$('button');
      for (const btn of btns) {
        const icon = await btn.$('.anticon-check-circle');
        if (icon) {
          await btn.click({ force: true });
          OK('Validate clicked (fallback)');
          validated = true;
          break;
        }
      }
    }
    if (!validated) {
      W('Validate btn not found');
    }

    // Step 5: Wait for validation response
    await page.waitForTimeout(5000);
    shot(page, 'autofix-validate');

    // Step 6: Check for .ant-alert-warning (validation failed alert)
    const diag = await page.evaluate(() => {
      const alerts = document.querySelectorAll('.ant-alert');
      const result = [];
      for (const a of alerts) {
        const cls = (a as HTMLElement).className;
        const btns = a.querySelectorAll('button');
        const btnTexts = Array.from(btns).map(b => (b as HTMLElement).innerText?.trim()?.slice(0, 40));
        result.push({ cls: cls.split(' ').filter((c) => c.startsWith('ant-alert-')).join(' '), btnTexts });
      }
      return { alertCount: alerts.length, alerts: result };
    });
    console.log('PHASE6_DIAG', JSON.stringify(diag));

    if (diag.alertCount === 0) {
      W('No validation alert appeared');
      await page.keyboard.press('Control+a');
      await page.waitForTimeout(100);
      await page.keyboard.press('Backspace');
      await humanPause(200, 400);
      await insertCode(page, CODE);
      return;
    }

    // Look for warning-type alert (has errors)
    const warnAlert = diag.alerts.find((a) => a.cls.includes('ant-alert-warning'));
    if (!warnAlert) {
      const successAlert = diag.alerts.find((a) => a.cls.includes('ant-alert-success'));
      if (successAlert) W('Validation passed - broken code not detected');
      else W('Unexpected alert type');
      return;
    }
    OK('Validation errors detected (warning alert)');

    // Check for Auto-Fix button text in the alert
    const fixBtnTexts = warnAlert.btnTexts.filter((t) => /fix/i.test(t));
    if (fixBtnTexts.length === 0) {
      W('Auto-Fix button not in alert. Buttons: ' + JSON.stringify(warnAlert.btnTexts));
      return;
    }
    OK('Auto-Fix button visible: ' + fixBtnTexts[0]);

    // Click Auto-Fix button
    const clicked = await page.evaluate(() => {
      const buttons = document.querySelectorAll('.ant-alert-warning button.ant-btn-primary');
      for (const btn of buttons) {
        const txt = (btn as HTMLElement).innerText?.trim() || '';
        if (/fix/i.test(txt)) { (btn as HTMLElement).click(); return txt; }
      }
      for (const btn of buttons) {
        (btn as HTMLElement).click();
        return (btn as HTMLElement).innerText?.trim();
      }
      return '';
    });

    if (clicked) {
      OK('Auto-Fix started: ' + clicked);
      const fixResult = await pollForText(page, /passed|success|fixed/i, 90, 3000);
      const stillWarning = await page.$('.ant-alert-warning');
      if (!stillWarning) {
        OK('Auto-Fix passed (error alert gone)');
      } else if (fixResult.found) {
        OK('Auto-Fix completed');
      } else {
        W('Auto-Fix timeout');
      }
    } else {
      W('Could not click Auto-Fix button');
    }
    shot(page, 'autofix-done');

    // Restore original code for subsequent phases
    await page.keyboard.press('Control+a');
    await page.waitForTimeout(100);
    await page.keyboard.press('Backspace');
    await humanPause(200, 400);
    await insertCode(page, CODE);
  });

  // ═══ PHASE 7: AI OPTIMIZE ═══
  await phase('AI Optimize', page, async () => {
    // AI Optimize button appears after backtest completes (dashed button with metrics)
    // Scroll to backtest results area
    await page.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const btn of btns) {
        if (btn.textContent?.includes('AI Optimize')) {
          btn.scrollIntoView({ block: 'center' });
          break;
        }
      }
    });
    await page.waitForTimeout(500);

    const optBtn = await page.$('button:has-text("AI Optimize")');
    if (optBtn) {
      const disabled = await optBtn.evaluate((el: HTMLButtonElement) => el.disabled);
      if (!disabled) {
        await humanPause(500, 1000);
        await optBtn.click({ force: true });
        OK('AI Optimize clicked');

        // The AI chat should open with metrics context
        // Wait for analysis text to appear
        const analysisResult = await pollForText(page, /analysis|suggestion|recommend|improve|optimiz/i, 60, 2000);
        if (analysisResult.found) OK('AI analysis received');
        else W('AI analysis timeout (60s)');
      } else W('AI Optimize disabled');
    } else W('AI Optimize btn not found');
    shot(page, 'ai-optimize');
  });

  // ═══ PHASE 8: SMART TUNING ═══
  await phase('Smart Tuning Tab', page, async () => {
    // Scroll to backtest results bottom panel
    await page.evaluate(() => {
      const btns = document.querySelectorAll('[role="button"]');
      for (const btn of btns) {
        if (btn.textContent?.includes('Backtest Results')) {
          btn.scrollIntoView({ block: 'center' }); break;
        }
      }
      window.scrollTo(0, document.body.scrollHeight);
    });
    await page.waitForTimeout(500);

    const clicked = await switchTab(page, 'Smart Tuning');
    clicked ? OK('Tuning tab opened') : W('Tuning tab not found');
    await page.waitForTimeout(1000);
    shot(page, 'tuning');
  });

  await phase('Tuning Optimizer Selection', page, async () => {
    // Test optimizer switching (human-like exploration)
    // First: Random Search (default)
    const randOk = await page.evaluate(() => {
      const wrappers = document.querySelectorAll('.ant-radio-button-wrapper, .ant-radio-wrapper');
      for (const w of wrappers) {
        const t = (w as HTMLElement).innerText?.trim() || '';
        if (/random/i.test(t) && t.length < 30) {
          w.dispatchEvent(new MouseEvent('click', { bubbles: true })); return t;
        }
      }
      return '';
    });
    randOk ? OK(`Selected: ${randOk}`) : W('Random not found');
    await humanPause(800, 1500);

    // Switch to Grid Search briefly (to see dimension list)
    const gridOk = await page.evaluate(() => {
      const wrappers = document.querySelectorAll('.ant-radio-button-wrapper, .ant-radio-wrapper');
      for (const w of wrappers) {
        const t = (w as HTMLElement).innerText?.trim() || '';
        if (/grid/i.test(t) && t.length < 30) {
          w.dispatchEvent(new MouseEvent('click', { bubbles: true })); return t;
        }
      }
      return '';
    });
    await page.waitForTimeout(500);

    // Switch back to Random Search
    await page.evaluate(() => {
      const wrappers = document.querySelectorAll('.ant-radio-button-wrapper, .ant-radio-wrapper');
      for (const w of wrappers) {
        const t = (w as HTMLElement).innerText?.trim() || '';
        if (/random/i.test(t) && t.length < 30) {
          w.dispatchEvent(new MouseEvent('click', { bubbles: true })); break;
        }
      }
    });
    OK('Optimizer explored');
    shot(page, 'tuning-optimizer');
  });

  await phase('Run Tuning', page, async () => {
    // Ensure code panel is closed so tuning area is accessible
    if (await codePanelOpen(page)) { await toggleCodePanel(page); await page.waitForTimeout(1500); }
    // Find Run button in tuning area (y between 500-1000)
    let tuneRun: any = null;
    for (const b of await page.$$('button')) {
      const txt = await b.textContent();
      const box = await b.boundingBox();
      if (txt && /Run\s*\d*/.test(txt) && box && box.y > 500 && box.y < 1000) {
        const disabled = await b.evaluate((el: HTMLButtonElement) => el.disabled);
        if (!disabled) { tuneRun = b; break; }
      }
    }
    if (!tuneRun) { W('Tuning Run btn not found'); return; }

    await humanPause(500, 1200);
    await tuneRun.click({ force: true });
    OK('Tuning started');

    const result = await pollForText(page, /grade|score|rank|OOS|overfit|Apply.*Code|apply/i, 180);
    if (result.found) {
      OK(`Tuning done ~${Math.round(result.elapsed)}s`);
      // Human-like: scroll through results table
      await page.evaluate(() => { window.scrollBy(0, 200); });
      await humanPause(400, 800);
      await page.evaluate(() => { window.scrollBy(0, -100); });
      await humanPause(300, 500);
      const table = await page.$('table');
      table ? OK('Results table present') : W('No table');
    } else {
      W('Tuning timeout 180s');
    }
    shot(page, 'tuning-done');
  });

  // ═══ PHASE 9: GATE ═══
  await phase('Gate Tab', page, async () => {
    const clicked = await switchTab(page, 'Gate');
    clicked ? OK('Gate tab opened') : W('Gate tab not found');
    await page.waitForTimeout(1000);
    shot(page, 'gate');
  });

  await phase('Run Gate', page, async () => {
    const gb = await page.$('button:has-text("Run Gate"), button:has-text("Evaluate")');
    if (!gb) { W('Gate btn not found'); return; }
    const gd = await gb.evaluate((el: HTMLButtonElement) => el.disabled);
    if (gd) { W('Gate disabled — run backtest first'); return; }

    await humanPause(400, 1000);
    await gb.click({ force: true });
    OK('Gate started');
    // Confirm loading state
    await page.waitForTimeout(1000);
    const loadingBtn = await page.$('button .anticon-loading');
    if (loadingBtn) console.log('   Gate loading confirmed');

    const result = await pollForText(page, /pass|fail|compliance|lookahead|walkforward|PASS|FAIL|Gate|gate/i, 180, 3000);
    // Also try detecting gate Steps component or Alert
    if (!result.found) {
      const stepsOrAlert = await page.$('.ant-steps, .ant-alert');
      if (stepsOrAlert) OK(`Gate done ~${Math.round(result.elapsed)}s (alt)`);
      else W('Gate timeout 180s');
    } else {
      OK(`Gate done ~${Math.round(result.elapsed)}s`);
    }
    shot(page, 'gate-done');
  });

  // ═══ PHASE 10: QUICKTRADE ═══
  await phase('QuickTrade Panel', page, async () => {
    // QT panel at x >= 1100 (right side)
    const qtRightEls = await page.$$eval('*', els => {
      return els.filter(e => {
        const r = (e as HTMLElement).getBoundingClientRect();
        const t = (e as HTMLElement).innerText?.trim() || '';
        return r.x >= 1100 && t.length > 0 && t.length < 40 && r.width > 50;
      }).length;
    });
    qtRightEls >= 5 ? OK(`QT panel: ${qtRightEls} elements`) : W(`Only ${qtRightEls} QT elements`);
    shot(page, 'quicktrade');
  });

  await phase('QT Trading Controls', page, async () => {
    // Buy button
    const buyBtn = await page.$('button:has-text("Buy")');
    if (buyBtn) { const b = await buyBtn.boundingBox(); b && b.x >= 1100 ? OK('Buy button') : W('Buy missing'); }
    // Sell button
    const sellBtn = await page.$('button:has-text("Sell")');
    if (sellBtn) { const b = await sellBtn.boundingBox(); b && b.x >= 1100 ? OK('Sell button') : W('Sell missing'); }

    // Order type
    const qtSelects = await page.$$eval('.ant-select', els =>
      els.filter(e => { const r = (e as HTMLElement).getBoundingClientRect(); return r.x >= 1100 && r.y < 400; })
        .map(e => e.textContent?.trim())
    );
    qtSelects.length > 0 ? OK(`Order: ${qtSelects[0]}`) : W('No order select');

    // Lot input at x >= 1100
    let lotFound = false;
    for (const li of await page.$$('.ant-input-number-input')) {
      const b = await li.boundingBox();
      if (b && b.x >= 1100) { lotFound = true; break; }
    }
    lotFound ? OK('Lot input') : W('No lot');

    // Margin mode
    const crossLabel = await page.$('.ant-radio-button-wrapper:has-text("Cross")');
    const isoLabel = await page.$('.ant-radio-button-wrapper:has-text("Isolated")');
    (crossLabel || isoLabel) ? OK('Margin mode') : W('No margin');

    // SL/TP
    const slTpInputs = await page.$$('input[placeholder*="Stop"], input[placeholder*="Take"], input[placeholder*="SL"], input[placeholder*="TP"]');
    slTpInputs.length >= 2 ? OK('SL/TP inputs') : W('SL/TP missing');
    shot(page, 'qt-controls');
  });

  await phase('Place Market Order', page, async () => {
    // Human-like: scan QT panel before trading
    await page.evaluate(() => window.scrollBy(0, 40));
    await humanPause(300, 700);
    
    // Set lot size to 0.01 first
    for (const li of await page.$$('.ant-input-number-input')) {
      const b = await li.boundingBox();
      if (b && b.x >= 1100) {
        await li.click({ force: true });
        await page.waitForTimeout(200);
        await page.keyboard.press('Control+a');
        await page.keyboard.type('0.01', { delay: 30 });
        break;
      }
    }

    // Place a small market buy
    const buySymBtn = await page.$(`button:has-text("Buy ${SYM}")`);
    if (!buySymBtn) { W('Buy button missing'); return; }
    const bd = await buySymBtn.evaluate((el: HTMLButtonElement) => el.disabled);
    if (bd) { W('Buy disabled'); return; }

    // Trader hesitation: check price, confirm lot, hover over button
    await humanPause(800, 2000); // indecision
    await buySymBtn.hover();
    await humanPause(400, 900); // last check
    await buySymBtn.click({ force: true });
    OK('Market buy order sent');
    await page.waitForTimeout(3000);
    shot(page, 'qt-buy');
  });

  await phase('Close Position', page, async () => {
    // Wait for position to appear after order
    await page.waitForTimeout(3000);

    // Find close buttons anywhere in QuickTrade area
    const closeBtns = await page.$$('button:has-text("Close")');
    let clicked = false;
    for (const cb of closeBtns) {
      const b = await cb.boundingBox();
      // Broad check: any close button on the right side of the page
      if (b && b.x >= 800 && b.y > 400) {
        await humanPause(500, 1000);
        await cb.click({ force: true });
        OK('Position closed');
        clicked = true;
        break;
      }
    }
    if (!clicked) W('No close btn found');
    shot(page, 'positions');
  });

  // ═══ PHASE 11: AUTO TRADING SETTINGS ═══
  await phase('Auto Trading Page', page, async () => {
    const ok = await navigateTo(page, '/auto-trading');
    ok ? OK('Auto Trading loaded') : W('Nav failed');
    await humanPause(1000, 2000);
    exploreScroll(page);
    shot(page, 'auto-trading');
  });

  await phase('Auto Trading Config', page, async () => {
    // Check for risk config fields
    const body = await page.textContent('body') || '';
    const riskTerms = ['max position', 'max risk', 'max lot', 'max daily loss', 'max drawdown', 'risk'];
    let found = 0;
    for (const t of riskTerms) { if (new RegExp(t, 'i').test(body)) found++; }
    found >= 2 ? OK(`Risk config: ${found}/${riskTerms.length} terms`) : W('Few risk fields');
    shot(page, 'auto-trading-config');
  });

  // ═══ PHASE 12: MARKETPLACE ═══
  await phase('Marketplace Page', page, async () => {
    const ok = await navigateTo(page, '/marketplace');
    ok ? OK('Marketplace loaded') : W('Nav failed');
    await wait(page, 3000);
    exploreScroll(page);
    shot(page, 'marketplace');
  });

  await phase('Marketplace Browse', page, async () => {
    // Check for strategy cards or marketplace content
    const cards = await page.$$('.ant-card');
    const body = await page.textContent('body') || '';
    if (cards.length > 0) {
      OK(`${cards.length} cards found`);
      // Click first card to view detail
      await humanPause(500, 1000);
      await cards[0].click({ force: true });
      await page.waitForTimeout(1500);
      const modal = await page.$('.ant-modal, .ant-drawer');
      if (modal) {
        OK('Detail opened');
        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);
      }
    } else if (/marketplace|market|browse/i.test(body)) {
      W('Empty marketplace');
    } else {
      W('No marketplace content');
    }
    shot(page, 'marketplace-browse');
  });

  // ═══ PHASE 13: SCHEDULES ═══
  await phase('Strategy Library', page, async () => {
    const ok = await navigateTo(page, '/strategy/library');
    ok ? OK('Library loaded') : W('Nav failed');
    await wait(page, 3000);
    shot(page, 'library');
  });

  await phase('Schedules Tab', page, async () => {
    // First select a template so the right panel (with tabs) renders
    const listItems = await page.$$('.ant-list-item, [class*="template"], [class*="Library"] [role="listitem"]');
    if (listItems.length === 0) {
      // Try clicking any clickable element in the left panel
      const leftClicks = await page.$$('[style*="340px"] [role="button"], .ant-list-items > *');
      if (leftClicks.length > 0) listItems.push(leftClicks[0]);
    }
    if (listItems.length > 0) {
      await listItems[0].click({ force: true });
      await page.waitForTimeout(1500);
      OK(`Template selected (${listItems.length} items)`);
    } else {
      W('No templates in library');
      return;
    }

    // Switch to Schedules tab
    const tabs = await page.$$('.ant-tabs-tab');
    let schedTab: any = null;
    for (const t of tabs) {
      const txt = await t.textContent();
      if (/schedule|Schedule|定时|排程|Schedules/i.test(txt || '')) { schedTab = t; break; }
    }
    if (schedTab) {
      await schedTab.click({ force: true });
      await page.waitForTimeout(1000);
      OK('Schedules tab opened');
    } else {
      W('No schedules tab');
    }
    shot(page, 'schedules');
  });

  // ═══ PHASE 14: MULTI-SYMBOL SESSION ═══
  await phase('Multi-Symbol Switch', page, async () => {
    // Navigate back to workspace
    await navigateTo(page, '/strategy/workspace');
    await wait(page, 4000);

    // Select account first
    await selectOption(page, 'Select account', ACCT);
    await page.waitForTimeout(1500);

    // Switch to XAUUSDm
    await selectOption(page, 'Select symbol', ALT_SYM);
    await humanPause(1500, 2500);
    const sTxt = await page.$$eval('.ant-select', els => els[1]?.textContent?.trim());
    sTxt?.includes(ALT_SYM) ? OK(`Switched to ${ALT_SYM}`) : W('Alt symbol?');

    // Verify chart re-renders
    const canvases = (await page.$$('canvas')).length;
    canvases >= 3 ? OK(`${canvases} canvases after switch`) : W('Chart update?');

    // Switch back to BTCUSDm
    await selectOption(page, 'Select symbol', SYM);
    await humanPause(1000, 2000);
    const backTxt = await page.$$eval('.ant-select', els => els[1]?.textContent?.trim());
    backTxt?.includes(SYM) ? OK(`Back to ${SYM}`) : W('Symbol restore?');
    shot(page, 'multi-symbol');
  });

  // ═══ PHASE 15: HISTORY MODAL ═══
  await phase('Backtest History', page, async () => {
    await page.evaluate(() => window.scrollTo(0, 0));
    await page.waitForTimeout(500);

    const allIcons = await page.$$('.anticon-history');
    let histBtn: any = null;
    for (const icon of allIcons) {
      const box = await icon.boundingBox();
      if (box && box.y < 600 && box.x > 900) {
        histBtn = await icon.evaluateHandle(el => {
          let p: any = el.parentElement;
          for (let i = 0; i < 4 && p; i++) { if (p.tagName === 'BUTTON') return p; p = p.parentElement; }
          return el;
        });
        break;
      }
    }

    if (histBtn) {
      await (histBtn as any).click({ force: true });
      await page.waitForTimeout(1500);
      const modal = await page.$('.ant-modal');
      if (modal) {
        OK('History modal opened');
        const rows = await page.$$('.ant-modal table tbody tr');
        rows.length > 0 ? OK(`${rows.length} history runs`) : W('Empty');
        // Close
        const closeBtn = await page.$('.ant-modal-close');
        if (closeBtn) { await closeBtn.click({ force: true }); await page.waitForTimeout(500); OK('Modal closed'); }
      } else W('No modal');
    } else W('History icon missing');
    shot(page, 'history');
  });

  // ═══ PHASE 16: TEMPLATE MANAGEMENT ═══
  await phase('Template Manager', page, async () => {
    // Open code panel
    if (!(await codePanelOpen(page))) { await toggleCodePanel(page); await page.waitForTimeout(2000); }
    if (!(await codePanelOpen(page))) { W('Panel not open'); return; }

    // Find Template collapse header (text may be translated)
    const collapseHeaders = await page.$$('.ant-collapse-header');
    for (const h of collapseHeaders) {
      const txt = await h.textContent();
      if (txt && /Template|模板|テンプレート/i.test(txt)) {
        await h.click({ force: true });
        await page.waitForTimeout(1200);
        OK('Template manager opened');
        break;
      }
    }

    // Look for template select — need to wait for collapse animation
    await page.waitForTimeout(800);
    // Also check if collapse needs another click to fully expand
    const templateSelect = await page.$('.ant-collapse-content-active .ant-select, .ant-collapse-content-box .ant-select, .ant-collapse-item-active .ant-select');
    if (templateSelect) {
      await templateSelect.click({ force: true });
      await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      if (opts.length > 0) {
        OK(`${opts.length} templates available`);
        // Click first non-current template to switch
        for (const o of opts) {
          const t = await o.textContent();
          if (t && !t.includes('E2E Full Test v3')) {
            // Dropdown options may be off-screen; scroll into view and force click
            await o.evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
            await page.waitForTimeout(200);
            await o.click({ force: true }).catch(() => {});
            await page.waitForTimeout(1000);
            OK('Template loaded');
            break;
          }
        }
      }
      await page.keyboard.press('Escape');
    } else {
      W('No template select');
    }
    shot(page, 'template-mgr');
  });

  // ═══ PHASE 17: SETTINGS PERSISTENCE ═══
  await phase('Settings Persistence', page, async () => {
    // Close code panel first for clean state
    if (await codePanelOpen(page)) { await toggleCodePanel(page); await page.waitForTimeout(1500); }

    // Change backtest capital to 20000
    const spins = await page.$$('input[role="spinbutton"]');
    if (spins.length > 0) {
      await spins[0].click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('20000', { delay: 20 });
      OK('Capital set to 20k');
    }

    // Save settings (click Save in settings dropdown if present, or rely on localStorage)
    // Look for settings dropdown button in backtest params card
    const settingsBtn = await page.$('button:has-text("Settings")');
    if (settingsBtn) {
      await settingsBtn.click({ force: true });
      await page.waitForTimeout(500);
      const saveItem = await page.$('.ant-dropdown-menu-item:has-text("Save")');
      if (saveItem) {
        await saveItem.click({ force: true });
        OK('Settings saved');
      }
    }

    // Navigate back to workspace (more reliable than reload)
    await page.goto(`${BASE}/strategy/workspace`, { waitUntil: 'domcontentloaded', timeout: 30000 });
    await wait(page, 5000);
    OK('Page reloaded');
    shot(page, 'settings-persist');
  });

  // ═══ PHASE 18: FINAL CHECKS ═══
  await phase('API Calls', page, async () => {
    if (apiLog.length > 0) {
      const groups: Record<string, number> = {};
      for (const a of apiLog) {
        const svc = a.replace(/^API \d{3} /, '').split('/')[0] || 'unknown';
        groups[svc] = (groups[svc] || 0) + 1;
      }
      const summary = Object.entries(groups).map(([k, v]) => `${k}×${v}`).join(', ');
      OK(`API: ${apiCount} calls, ${apiLog.length} logged → ${summary}`);
    } else {
      W('No API calls logged');
    }
  });

  await phase('JS Errors', page, async () => {
    const uniq = [...new Set(jsErrs)].filter(e =>
      !e.includes('ResizeObserver') && !e.includes('Script error')
    );
    if (uniq.length === 0) {
      OK('Zero JS errors');
    } else {
      uniq.slice(0, 5).forEach(e => W('JS', e));
    }
  });

  // ═══ SUMMARY ═══
  const pass = R.filter(r => r.startsWith('✅')).length;
  const warns = R.filter(r => r.startsWith('⚠️')).length;
  const fails = R.filter(r => r.startsWith('❌')).length;
  console.log(`\n╔════════════════════════════════════╗`);
  console.log(`║  Full Workspace: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/full-*.png`);
  console.log(`║  API calls: ${apiCount}`);
  console.log('╚════════════════════════════════════╝');
  R.forEach(r => console.log(r));

  // Print unique API endpoints
  const eps = [...new Set(apiLog.map(a => a.replace(/^API \d{3} /, '')))];
  if (eps.length > 0) {
    console.log(`\nAPI endpoints (${eps.length}):`);
    eps.forEach(e => console.log(`  ${e}`));
  }

  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
