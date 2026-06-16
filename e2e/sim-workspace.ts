/**
 * E2E: Human-like workspace operation simulation — v4 (final).
 *
 * Flow:
 *   1. Login → 2. Workspace → 3. Close code panel →
 *   4. Select account (Exness-Trial) → 5. Select symbol (ADAUSDm) →
 *   6. Open code panel → 7. Insert code via CM6 dispatch →
 *   8. Ctrl+Enter validate → 9. Close code panel →
 *   10. Configure backtest params → 11. Run backtest →
 *   12. Wait for results → 13. Review metrics →
 *   14. Tuning tab → 15. Gate tab → 16. JS errors
 *
 * Usage: npx tsx e2e/sim-workspace.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT, { recursive: true });

const USER = '888888';
const PASS = '12345678';
const ACCOUNT_TEXT = '277259925';  // Exness-MT5Trial5
const SYMBOL = 'BTCUSDm';
const TIMEFRAME = 'M15';

const STRATEGY_CODE = `"""
@strategy SMA crossover scalping
@param fast_period: 5
@param slow_period: 20
@param stop_loss_pct: 0.5
@param take_profit_pct: 1.0
@param position_size: 0.01
"""

def init():
    pass

def run(ctx):
    close = ctx.close()
    if len(close) < 22:
        return

    # SMA calculation without numpy
    fast_sma = sum(close[-5:]) / 5.0
    slow_sma = sum(close[-20:]) / 20.0
    prev_fast = sum(close[-6:-1]) / 5.0
    prev_slow = sum(close[-21:-1]) / 20.0

    pos = ctx.position()

    if prev_fast <= prev_slow and fast_sma > slow_sma and pos.side != 'buy':
        ctx.open_long(lot=0.01, sl_pct=0.5, tp_pct=1.0)
    elif prev_fast >= prev_slow and fast_sma < slow_sma and pos.side != 'sell':
        ctx.open_short(lot=0.01, sl_pct=0.5, tp_pct=1.0)
`;

// ── Reporting ──
const results: string[] = [];
let stepNum = 0;
function step(msg: string) { stepNum++; console.log(`\n── ${stepNum}. ${msg} ──`); }
function ok(msg: string) { results.push(`✅ ${msg}`); console.log(`   ✅ ${msg}`); }
function warn(msg: string, detail?: string) { results.push(`⚠️ ${msg}${detail ? ': ' + detail : ''}`); console.log(`   ⚠️ ${msg}${detail ? ': ' + detail : ''}`); }
function fail(msg: string, detail?: string) { results.push(`❌ ${msg}${detail ? ': ' + detail : ''}`); console.log(`   ❌ ${msg}${detail ? ': ' + detail : ''}`); }
function shot(page: Page, name: string) {
  return page.screenshot({ path: `${SHOT}/sim-${String(stepNum).padStart(2, '0')}-${name}.png`, fullPage: false }).catch(() => {});
}

// ── Helpers ──
async function waitSettle(page: Page, ms = 3000) {
  await page.waitForLoadState('domcontentloaded').catch(() => {});
  await page.waitForTimeout(ms);
}

async function selectOption(page: Page, selectorText: string, optionText: string): Promise<boolean> {
  const selects = await page.$$('.ant-select');
  let target: any = null;
  for (const sel of selects) {
    const txt = await sel.textContent();
    if (txt && txt.includes(selectorText)) { target = sel; break; }
  }
  if (!target) {
    if (selectorText.includes('Select account')) target = selects[0] || null;
    else if (selectorText.includes('Select symbol')) target = selects[1] || null;
    if (!target) { console.log(`   Cannot find select: "${selectorText}"`); return false; }
  }

  await target.click({ force: true });
  await page.waitForTimeout(800);

  // For symbol selector: search using the combobox input
  if (!selectorText.includes('account')) {
    const comboboxInputs = await page.$$('input[role="combobox"]');
    // Symbol select is the second combobox (after account)
    if (comboboxInputs.length >= 2) {
      await comboboxInputs[1].fill(optionText);
      await page.waitForTimeout(1200);
    }
  }

  const options = await page.$$('.ant-select-item-option');
  const isAccountSelect = selectorText.includes('account');

  // Prefer exact match
  for (const opt of options) {
    const txt = await opt.textContent();
    if (txt && txt.trim() === optionText) { await opt.click({ force: true }); await page.waitForTimeout(500); return true; }
  }
  // Partial match
  for (const opt of options) {
    const txt = await opt.textContent();
    if (!txt) continue;
    // For account: allow "·" (e.g. "Exness-MT5Trial5 · 277259925")
    // For symbol: exclude "·" (account names leak into symbol dropdown)
    const allowed = isAccountSelect || !txt.includes('·');
    if (txt.includes(optionText) && allowed) {
      await opt.click({ force: true });
      await page.waitForTimeout(500);
      return true;
    }
  }
  // Fallback: take first non-account option (or any for account select)
  for (const opt of options) {
    const txt = await opt.textContent();
    if (txt && (isAccountSelect || !txt.includes('·'))) {
      await opt.click({ force: true });
      await page.waitForTimeout(500);
      return true;
    }
  }
  await page.keyboard.press('Escape');
  return false;
}

/** Click the vertical gradient code toggle strip at x≈252 */
async function toggleCodePanel(page: Page) {
  const divs = await page.$$('div');
  for (const div of divs) {
    const box = await div.boundingBox();
    if (box && box.x >= 250 && box.x <= 260 && box.width <= 35 && box.height >= 100) {
      await div.click({ force: true });
      await page.waitForTimeout(1000);
      return;
    }
  }
  console.log('   Code toggle strip not found');
}

async function codePanelVisible(page: Page): Promise<boolean> {
  return page.$$eval('[style*="position"]', els => {
    for (const e of els) {
      const r = (e as HTMLElement).getBoundingClientRect();
      if (r.width >= 600 && r.width <= 800 && r.height > 100) return true;
    }
    return false;
  });
}

async function editorReady(page: Page): Promise<boolean> {
  return (await page.$$('.cm-editor')).length > 0;
}

/** Insert code into CodeMirror — triggers React onChange so canRun becomes true */
async function insertCode(page: Page, code: string) {
  const result = await page.evaluate((codeContent) => {
    const editor = document.querySelector('.cm-editor');
    if (!editor) {
      // Fallback: try to find any visible input/textarea in the code panel area
      const inputs = document.querySelectorAll('textarea, [contenteditable="true"], .cm-content');
      for (const inp of inputs) {
        const r = (inp as HTMLElement).getBoundingClientRect();
        if (r.width > 200 && r.height > 100) {
          (inp as HTMLElement).focus();
          document.execCommand('selectAll', false);
          document.execCommand('insertText', false, codeContent);
          return 'ok-fallback-input';
        }
      }
      return 'no-editor';
    }

    // Find CM6 view
    let view: any = null;
    for (const key of Object.keys(editor)) {
      const val = (editor as any)[key];
      if (val && typeof val === 'object' && val.state && val.dispatch && val.viewport) {
        view = val;
        break;
      }
    }

    if (view) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: codeContent }
      });
      return 'ok-cm6';
    }

    // execCommand fallback
    const cmContent = editor.querySelector('.cm-content') as HTMLElement;
    if (cmContent) {
      cmContent.focus();
      document.execCommand('selectAll', false);
      document.execCommand('insertText', false, codeContent);
      return 'ok-execCommand';
    }

    return 'no-method';
  }, code);

  console.log(`   Code insert: ${result}`);
  await page.waitForTimeout(1000);
}

// ═══════════════════════════════════════════════════════════
async function run() {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/snap/bin/chromium',
    args: ['--no-sandbox', '--disable-setuid-sandbox'],
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });

  const jsErrors: string[] = [];
  page.on('pageerror', err => jsErrors.push(err.message));

  try {
    // ═══ 1. Login ═══
    step('Login');
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await page.fill('#login_login', USER);
    await page.fill('#login_password', PASS);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    if (!page.url().includes('login')) ok('Login successful');
    else { fail('Login failed'); await browser.close(); return; }
    shot(page, 'login');

    // ═══ 2. Navigate to Workspace ═══
    step('Navigate to Strategy Workspace');
    await page.goto(`${BASE}/strategy/workspace`, { waitUntil: 'domcontentloaded', timeout: 20000 });
    await waitSettle(page, 6000);
    ok('Workspace loaded');
    shot(page, 'workspace-initial');

    // ═══ 3. Select Account ═══
    step('Select Account');
    await selectOption(page, 'Select account', ACCOUNT_TEXT);
    await page.waitForTimeout(1500);
    const sel0 = await page.$$eval('.ant-select', els => els[0]?.textContent?.trim());
    if (sel0 && sel0 !== 'Select account') ok(`Account: "${sel0}"`);
    else warn('Account selection unclear');
    shot(page, 'account');

    // ═══ 4. Select Symbol ═══
    step(`Select Symbol (${SYMBOL})`);
    await selectOption(page, 'Select symbol', SYMBOL);
    await page.waitForTimeout(2000);
    const sel1txt = await page.$$eval('.ant-select', els => els[1]?.textContent?.trim());
    if (sel1txt && sel1txt !== 'Select symbol') ok(`Symbol: "${sel1txt}"`);
    else warn('Symbol selection unclear');
    shot(page, 'symbol');

    // ═══ 5. Ensure Code Panel Open + CodeMirror Ready ═══
    step('Ensure Code Panel Open with CodeMirror');
    // Panel may already be open from persisted state — don't blindly toggle!
    let cmReady = await editorReady(page);

    if (!cmReady) {
      // Panel is closed — toggle it open
      console.log('   Panel appears closed, toggling open...');
      await toggleCodePanel(page);
      await page.waitForTimeout(2000);

      for (let i = 0; i < 15; i++) {
        if (await editorReady(page)) { cmReady = true; break; }
        await page.waitForTimeout(1000);
      }
    }

    if (cmReady) ok('CodeMirror editor ready');
    else { warn('CodeMirror not found — trying fallback'); }
    shot(page, 'code-panel');

    // ═══ 6. Insert Strategy Code ═══
    step('Insert Strategy Code');
    await insertCode(page, STRATEGY_CODE);

    // Verify content
    const content = await page.$eval('.cm-content', el => el.textContent?.slice(0, 80)).catch(() => '');
    if (content && content.includes('@strategy')) ok('Strategy code inserted');
    else if (content) ok(`Code inserted (content: ${content.slice(0, 40)}...)`);
    else warn('Code content verification failed');
    shot(page, 'code-written');

    // ═══ 7. Validate Code ═══
    step('Validate Code (Ctrl+Enter)');
    const cmEd = await page.$('.cm-editor');
    if (cmEd) { await cmEd.click({ force: true }); await page.waitForTimeout(200); }
    else { console.log('   No CM editor for validation, trying keyboard anyway'); }
    await page.keyboard.press('Control+Enter');
    await page.waitForTimeout(4000);

    const errorAlerts = await page.$$('.ant-alert-error, [class*="alert-error"]');
    if (errorAlerts.length > 0) {
      const errTxt = await errorAlerts[0].textContent();
      warn('Validation issues', errTxt?.slice(0, 150));
    } else {
      ok('Validation completed (no error alert)');
    }
    shot(page, 'validated');

    // ═══ 8. Close Code Panel ═══
    step('Close Code Panel to access backtest area');
    await toggleCodePanel(page);
    await page.waitForTimeout(1500);
    // Verify panel is closed — CM editor should be gone or hidden
    const cmAfterClose = await page.$$('.cm-editor');
    const panelAfterClose = await codePanelVisible(page);
    if (!panelAfterClose || cmAfterClose.length === 0) ok('Code panel closed');
    else {
      // Try once more
      console.log('   Panel still visible, toggle again...');
      await toggleCodePanel(page);
      await page.waitForTimeout(1500);
      const stillVisible = await codePanelVisible(page);
      if (!stillVisible) ok('Code panel closed (2nd attempt)');
      else warn('Panel still visible — using force clicks through overlay');
    }
    shot(page, 'panel-closed');

    // ═══ 9. Configure Backtest ═══
    step('Configure Backtest Parameters');

    // Date: 3M
    const segs = await page.$$('.ant-segmented-item');
    for (const s of segs) {
      if ((await s.textContent())?.trim() === '3M') {
        await s.click({ force: true }); await page.waitForTimeout(300); ok('Date: 3M'); break;
      }
    }

    // Capital: 10000
    const spinInputs = await page.$$('input[role="spinbutton"]');
    if (spinInputs.length > 0) {
      await spinInputs[0].click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('10000', { delay: 30 });
      await page.waitForTimeout(200);
      ok('Capital: 10,000');
    }

    // Direction: Both
    const radios = await page.$$('.ant-radio-button-wrapper, .ant-radio-wrapper');
    for (const r of radios) {
      const t = await r.textContent();
      if (t?.trim() === 'Both' || t?.trim() === '双向') {
        await r.click({ force: true }); await page.waitForTimeout(200); ok('Direction: Both'); break;
      }
    }

    // Preset: Live Aligned
    const liveBtn = await page.$('button:has-text("Live Aligned")');
    if (liveBtn) { await liveBtn.click({ force: true }); await page.waitForTimeout(200); ok('Preset: Live Aligned'); }

    shot(page, 'backtest-config');

    // ═══ 10. Run Backtest ═══
    step('Run Backtest');
    const runBtn = await page.$('button:has-text("Run")');
    if (!runBtn) {
      fail('Run button not found');
    } else {
      const disabled = await runBtn.evaluate(el => (el as HTMLButtonElement).disabled);
      if (disabled) {
        fail('Run button disabled');
        const ss = await page.$$eval('.ant-select', els => els.map(e => e.textContent?.trim()));
        console.log('   Selects:', JSON.stringify(ss));
      } else {
        await runBtn.click({ force: true });
        ok('Backtest submitted ✓');
      }
    }
    shot(page, 'running');

    // ═══ 11. Wait for Results ═══
    step('Wait for Results (up to 180s)');
    let btDone = false, btFailed = false;
    for (let i = 0; i < 60; i++) {
      await page.waitForTimeout(3000);
      const elapsed = (i + 1) * 3;

      // Periodic status dump
      if (i % 10 === 0 && i > 0) {
        const snap = await page.textContent('body').catch(() => '');
        const statusWords = (snap || '').match(/(processing|running|completed|failed|error|queued|回测中|已[完成败])/gi) || [];
        console.log(`   ${elapsed}s: ${statusWords.join(', ') || 'no status'}`);
      }

      // Error patterns
      const bodyText = await page.textContent('body') || '';
      const errTags = await page.$$('.ant-tag:has-text("error"), .ant-tag:has-text("Error"), [class*="error-tag"], .ant-alert-error');
      if (errTags.length > 0) {
        fail('Backtest error', await errTags[0].textContent()?.slice(0, 200));
        btFailed = true; break;
      }

      // Success: metrics patterns
      const hasMetrics = /total.*return|net.*profit|return.*%/i.test(bodyText) ||
                        /win\s*rate|winrate/i.test(bodyText) ||
                        /sharpe/i.test(bodyText) ||
                        /max\s*drawdown|最大回撤/i.test(bodyText);
      const equityCharts = (await page.$$('.recharts-surface')).length;
      const tables = (await page.$$('table')).length;

      if (hasMetrics || equityCharts >= 3 || tables >= 2) {
        ok(`Backtest completed in ~${elapsed}s`);
        if (equityCharts >= 2) ok(`${equityCharts} chart surfaces`);
        if (tables > 0) ok(`Trade log found`);
        btDone = true; break;
      }

      // Also check for results panel with content
      const resultItems = await page.$$eval('[class*="result"], [class*="Result"], [class*="metric"], [class*="Metric"]',
        els => els.map(e => (e as HTMLElement).innerText?.trim()).filter(Boolean)
      ).catch(() => []);
      if (resultItems.length >= 5) {
        ok(`Results panel populated (${resultItems.length} items)`);
        btDone = true; break;
      }
    }
    if (!btDone && !btFailed) warn('Backtest timed out (180s)');
    shot(page, 'results');

    // ═══ 12. Review Results ═══
    if (btDone) {
      step('Review Results');
      const pageText = await page.textContent('body') || '';
      const checks: [string, RegExp][] = [
        ['Return', /return|回报/i],
        ['Win Rate', /win\s*rate|winrate|胜率/i],
        ['Drawdown', /drawdown|回撤/i],
        ['Sharpe', /sharpe|夏普/i],
        ['Trade Count', /trades|笔数|total/i],
      ];
      let found = 0;
      for (const [n, re] of checks) {
        if (re.test(pageText)) { ok(`Metric: ${n}`); found++; }
        else warn(`"${n}" not found`);
      }
      if (found >= 3) ok(`${found}/5 metrics visible`);
      shot(page, 'reviewed');

      // ═══ 13. Smart Tuning Tab ═══
      step('Smart Tuning Tab');
      // Find "Smart Tuning" div and click
      let tuningClicked = false;
      const divs = await page.$$('div');
      for (const div of divs) {
        const txt = await div.textContent();
        if (txt && txt.trim() === 'Smart Tuning') {
          const box = await div.boundingBox();
          if (box && box.height < 40) {
            await div.click({ force: true });
            await page.waitForTimeout(1500);
            tuningClicked = true;
            break;
          }
        }
      }
      if (tuningClicked) {
        ok('Smart Tuning tab clicked');
        const opts = await page.$$eval('.ant-radio-button-wrapper, .ant-radio-wrapper',
          els => els.map(e => (e as HTMLElement).innerText?.trim()).filter(Boolean));
        const optimizers = opts.filter(t => /grid|random|differential|tpe|annealed|ai/i.test(t));
        if (optimizers.length >= 3) ok(`${optimizers.length} optimizers available`);
        else if (optimizers.length > 0) ok(`${optimizers.length} optimizers visible`);
        else warn('Optimizer list empty');
      } else { warn('Smart Tuning tab not clickable'); }
      shot(page, 'tuning');

      // ═══ 14. Gate Tab ═══
      step('Gate Evaluation Tab');
      let gateClicked = false;
      const divs2 = await page.$$('div');
      for (const div of divs2) {
        const txt = await div.textContent();
        if (txt && txt.trim() === 'Gate') {
          const box = await div.boundingBox();
          if (box && box.height < 40) {
            await div.click({ force: true });
            await page.waitForTimeout(1500);
            gateClicked = true;
            break;
          }
        }
      }
      if (gateClicked) {
        ok('Gate tab clicked');
        const gateBtn = await page.$('button:has-text("Run Gate"), button:has-text("Evaluate")');
        if (gateBtn) ok('Gate evaluation button visible');
        else warn('Gate run button not visible');
      } else { warn('Gate tab not clickable'); }
      shot(page, 'gate');
    }

    // ═══ 15. JS Errors ═══
    step('Console JS Errors');
    const unique = [...new Set(jsErrors)];
    if (unique.length === 0) ok('Zero JS console errors');
    else {
      unique.slice(0, 5).forEach(e => warn('JS Error', e));
      if (unique.length > 5) warn(`+${unique.length - 5} more`);
    }

  } catch (e: any) {
    fail('FATAL', e.message);
    await shot(page, 'fatal');
    console.error(e);
  }

  // ═══════════ SUMMARY ═══════════
  const passed = results.filter(r => r.startsWith('✅')).length;
  const total = results.length;
  console.log('\n╔══════════════════════════════════╗');
  console.log(`║  Workspace Simulation: ${passed}/${total} passed`);
  console.log(`║  Screenshots: ${SHOT}/sim-*.png`);
  console.log('╚══════════════════════════════════╝');
  results.forEach(r => console.log(r));

  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
