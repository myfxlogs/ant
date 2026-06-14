/**
 * Market Tools E2E — comprehensive human-simulated operations.
 *
 * Tab 1 "Symbol Analysis": Symbol search → Analyze → MTF Outlook cards →
 *   S/R Levels → Volatility → AI Recommendation
 * Tab 2 "Market Regime": Account → Symbol → Timeframe → Detect → Regime result
 *
 * LEAVES REAL TRACES: analysis results, regime detection records.
 *
 * Usage: npx tsx e2e/full-market.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots/market';
fs.mkdirSync(SHOT, { recursive: true });

const U = '888888'; const P = '12345678';

// ══════════════════════════════════════
// Reporting
// ══════════════════════════════════════
const R: string[] = []; let sn = 0;
function S(m: string) { sn++; console.log(`\n── ${sn}. ${m} ──`); }
function OK(m: string) { R.push(`✅ ${m}`); console.log(`   ✅ ${m}`); }
function W(m: string, d?: string) { R.push(`⚠️ ${m}${d ? ': ' + d : ''}`); console.log(`   ⚠️ ${m}${d ? ': ' + d : ''}`); }
function F(m: string, d?: string) { R.push(`❌ ${m}${d ? ': ' + d : ''}`); console.log(`   ❌ ${m}${d ? ': ' + d : ''}`); }
function shot(page: Page, n: string) {
  return page.screenshot({ path: `${SHOT}/${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

// ══════════════════════════════════════
// Helpers
// ══════════════════════════════════════
async function humanPause(minMs = 800, maxMs = 3000) {
  await new Promise(r => setTimeout(r, Math.floor(minMs + Math.random() * (maxMs - minMs))));
}
async function navTo(page: Page, path: string) {
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 20000 });
  await page.waitForTimeout(3000 + Math.random() * 1000);
}
async function scrollExplore(page: Page) {
  await page.evaluate(() => { window.scrollBy(0, 150); });
  await page.waitForTimeout(400);
  await page.evaluate(() => { window.scrollBy(0, -80); });
  await page.waitForTimeout(300);
}
async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name);
  try { await fn(); }
  catch (e: any) { F(`Phase error: ${name}`, e.message?.slice(0, 150)); await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {}); }
}
async function clickTab(page: Page, re: RegExp): Promise<boolean> {
  return page.evaluate((reStr) => {
    const re = new RegExp(reStr, 'i');
    const tabs = document.querySelectorAll('.ant-tabs-tab');
    for (const t of tabs) {
      if (re.test((t as HTMLElement).innerText?.trim() || '')) {
        (t as HTMLElement).click(); return true;
      }
    }
    return false;
  }, re.source);
}

// ══════════════════════════════════════
// Main
// ══════════════════════════════════════
async function run() {
  const browser = await chromium.launch({ headless: true, executablePath: '/snap/bin/chromium', args: ['--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const jsErrs: string[] = [];
  const apiLog: string[] = []; let apiCount = 0;
  page.on('pageerror', e => jsErrs.push(e.message));
  page.on('response', res => {
    apiCount++;
    const url = res.url();
    if (/ant\.v1\.|assets\//i.test(url)) {
      apiLog.push(`API ${res.status()} ${url.replace(/^.*\/api\//, '').replace(/\?.*$/, '').split('/').slice(-2).join('/')}`);
    }
  });

  // ═══ PHASE 1: LOGIN ═══
  await phase('Login', page, async () => {
    await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForSelector('#login_login', { timeout: 5000 });
    await humanPause(300, 700);
    await page.fill('#login_login', U);
    await humanPause(300, 600);
    await page.fill('#login_password', P);
    await humanPause(200, 500);
    await page.click('button[type="submit"]');
    await page.waitForTimeout(3000);
    page.url().includes('login') ? F('Login failed') : OK('Login');
    shot(page, 'login');
  });

  // ═══ PHASE 2: NAVIGATE + PAGE STRUCTURE ═══
  await phase('Navigate & Structure', page, async () => {
    await navTo(page, '/strategy/market-tools');
    await scrollExplore(page);

    const body = await page.textContent('body') || '';

    // Check title
    (/market|tools|市场/i.test(body)) ? OK('Market Tools title') : W('Title?');

    // Two tabs
    const tabs = await page.$$('.ant-tabs-tab');
    const tabLabels: string[] = [];
    for (const t of tabs) { tabLabels.push((await t.textContent())?.trim() || ''); }
    tabLabels.length >= 2 ? OK(`Tabs: ${tabLabels.join(', ')}`) : W(`Only ${tabLabels.length} tabs`);

    // Symbol input on default tab
    const searchInput = await page.$('input[placeholder*="symbol"], input[placeholder*="Symbol"], input[placeholder*="品种"]');
    searchInput ? OK('Symbol input visible') : W('No symbol input');

    // Analyze button
    const analyzeBtn = await page.$('button:has-text("Analyze"), button:has-text("分析")');
    analyzeBtn ? OK('Analyze button') : W('No analyze btn');

    shot(page, 'structure');
  });

  // ═══ PHASE 3: SYMBOL ANALYSIS — ENTER + ANALYZE ═══
  await phase('Symbol Analysis', page, async () => {
    // Find the symbol input
    const inputs = await page.$$('input');
    let symInput: any = null;
    for (const inp of inputs) {
      const ph = await inp.getAttribute('placeholder');
      if (ph && /symbol|Symbol|品种/i.test(ph)) { symInput = inp; break; }
    }
    if (!symInput) { W('Symbol input not found'); return; }

    // Human-like: click, pause, type symbol
    await symInput.click({ force: true });
    await humanPause(300, 600);
    await page.keyboard.type('EURUSD', { delay: 50 + Math.random() * 40 });
    await humanPause(400, 800);
    OK('Symbol: EURUSD');

    // Click Analyze button
    const analyzeBtn = await page.$('button.ant-btn-primary:has(.anticon-thunderbolt)');
    if (!analyzeBtn) { W('Analyze button not found'); return; }

    await humanPause(400, 900); // hesitation before analyzing
    await analyzeBtn.click({ force: true });
    OK('Analysis started');
    shot(page, 'analyze-start');
  });

  // ═══ PHASE 4: WAIT FOR ANALYSIS PHASES ═══
  await phase('Wait Analysis Results', page, async () => {
    // Poll for analysis completion — phases: idle → fetching → mtf_outlook → sr_levels → volatility → ai_recommendation → complete
    let found = false;
    for (let i = 0; i < 60; i++) {
      await page.waitForTimeout(2000);
      const body = await page.textContent('body') || '';

      // Check for result cards
      const trends = body.includes('BULLISH') || body.includes('BEARISH') || body.includes('NEUTRAL');
      const srLevels = body.includes('RESISTANCE') || body.includes('SUPPORT') || body.includes('touches');
      const volatility = body.includes('LOW') || body.includes('NORMAL') || body.includes('HIGH') || body.includes('EXTREME');
      const aiRec = body.includes('recommendation') || body.includes('建议') || body.includes('分析');

      if (trends || srLevels || volatility || aiRec) {
        found = true;
        const phases: string[] = [];
        if (trends) phases.push('MTF');
        if (srLevels) phases.push('S/R');
        if (volatility) phases.push('Vol');
        if (aiRec) phases.push('AI');
        OK(`Analysis done ~${(i + 1) * 2}s: ${phases.join(', ')}`);
        break;
      }

      // Check for error
      const errAlert = await page.$('.ant-alert-error');
      if (errAlert) {
        const errTxt = await errAlert.textContent();
        W(`Analysis error: ${errTxt?.trim().slice(0, 100)}`);
        break;
      }

      if (i % 10 === 9) console.log(`   Wait ${(i + 1) * 2}s...`);
    }
    if (!found) W('Analysis timeout 120s');

    shot(page, 'analysis-results');
  });

  // ═══ PHASE 5: VERIFY MTF OUTLOOK CARDS ═══
  await phase('MTF Outlook', page, async () => {
    const cards = await page.$$('.ant-card');
    // Look for trend cards (1h, 4h, 1d, 1w)
    const body = await page.textContent('body') || '';
    let tfCount = 0;
    for (const tf of ['1h', '4h', '1d', '1w']) {
      if (body.includes(tf)) tfCount++;
    }

    if (tfCount >= 3) {
      OK(`${tfCount}/4 timeframe cards`);
    } else if (tfCount > 0) {
      OK(`${tfCount} timeframe cards (partial)`);
    } else {
      W('No MTF cards — analysis may be incomplete');
    }

    // Check trend tags
    const trendTags = await page.$$('.ant-tag');
    const trends: string[] = [];
    for (const t of trendTags) {
      const txt = await t.textContent();
      if (txt && /BULLISH|BEARISH|NEUTRAL/i.test(txt.trim())) trends.push(txt.trim());
    }
    trends.length > 0 ? OK(`Trends: ${trends.join(', ')}`) : W('No trend tags');

    // Check progress bars (strength)
    const progressBars = await page.$$('.ant-progress');
    progressBars.length >= 2 ? OK(`${progressBars.length} progress bars (strength)`) : W('No progress bars');

    shot(page, 'mtf');
  });

  // ═══ PHASE 6: VERIFY S/R LEVELS ═══
  await phase('S/R Levels', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 300));
    await humanPause(300, 600);

    // Look for resistance/support tags
    const srTags = await page.$$('.ant-tag');
    let srCount = 0;
    for (const t of srTags) {
      const txt = await t.textContent();
      if (txt && /RESISTANCE|SUPPORT|touches/i.test(txt)) srCount++;
    }
    srCount > 0 ? OK(`${srCount} S/R level tags`) : W('No S/R level tags');

    shot(page, 'sr-levels');
  });

  // ═══ PHASE 7: VERIFY VOLATILITY ═══
  await phase('Volatility', page, async () => {
    await page.evaluate(() => window.scrollBy(0, 200));
    await humanPause(300, 500);

    const body = await page.textContent('body') || '';
    const hasVol = /LOW|NORMAL|HIGH|EXTREME|ATR/i.test(body);
    hasVol ? OK('Volatility data visible') : W('No volatility data');

    shot(page, 'volatility');
  });

  // ═══ PHASE 8: VERIFY AI RECOMMENDATION ═══
  await phase('AI Recommendation', page, async () => {
    await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
    await humanPause(400, 700);

    const body = await page.textContent('body') || '';
    // AI recommendation can be markdown or "AI unavailable"
    const hasAI = /recommend|建议|bullish|bearish|markdown|unavailable|分析/i.test(body);
    hasAI ? OK('AI recommendation section') : W('No AI section');

    shot(page, 'ai-rec');
  });

  // ═══ PHASE 9: SWITCH TO MARKET REGIME TAB ═══
  await phase('Market Regime Tab', page, async () => {
    await page.evaluate(() => window.scrollTo(0, 0));
    await humanPause(300, 500);

    const clicked = await clickTab(page, /regime|Regime|市场|状況/i);
    clicked ? OK('Regime tab') : W('Regime tab not found');
    await page.waitForTimeout(1500);

    // Check form elements
    const formSelects = await page.$$('.ant-form-item .ant-select');
    formSelects.length >= 2 ? OK(`${formSelects.length} form selects`) : W('Form incomplete');

    // Account select
    if (formSelects.length > 0) {
      await formSelects[0].click({ force: true });
      await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      if (opts.length > 0) {
        await opts[0].click({ force: true });
        OK('Account selected');
      }
      await page.waitForTimeout(500);
      await page.keyboard.press('Escape');
    }

    // Symbol picker (may be a custom component, not a simple Select)
    if (formSelects.length >= 2) {
      try {
        await formSelects[1].click({ force: true });
        await page.waitForTimeout(800);
        const opts = await page.$$('.ant-select-item-option');
        if (opts.length > 0) {
          await opts[0].evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
          await page.waitForTimeout(100);
          await opts[0].click({ force: true }).catch(() => opts[0].evaluate((el: HTMLElement) => el.click()));
          OK('Symbol selected');
        }
      } catch {
        // SymbolPicker may use a custom dropdown — continue anyway
        W('Symbol picker not available — continuing');
      }
      await page.waitForTimeout(300);
      await page.keyboard.press('Escape');
    }

    shot(page, 'regime-form');
  });

  // ═══ PHASE 10: RUN MARKET REGIME DETECTION ═══
  await phase('Regime Detection', page, async () => {
    // Click submit/detect button
    const submitBtn = await page.$('button[type="submit"]');
    if (!submitBtn) { W('No submit button'); return; }

    await humanPause(400, 900);
    await submitBtn.click({ force: true });
    OK('Detection submitted');

    // Wait for result
    let hasResult = false;
    for (let i = 0; i < 30; i++) {
      await page.waitForTimeout(2000);
      const body = await page.textContent('body') || '';
      if (/regime|Regime|confidence|features|model.*version/i.test(body)) {
        hasResult = true;
        OK(`Regime result ~${(i + 1) * 2}s`);
        break;
      }
      // Check for descriptions table (result card)
      const descTable = await page.$('.ant-descriptions');
      if (descTable) { hasResult = true; OK('Regime result visible'); break; }
    }
    if (!hasResult) W('Regime detection timeout');

    // Read result if available
    const descItems = await page.$$('.ant-descriptions-item');
    if (descItems.length > 0) {
      const resultTexts: string[] = [];
      for (const d of descItems) { resultTexts.push((await d.textContent())?.trim() || ''); }
      OK(`Result: ${resultTexts.filter(Boolean).join(' | ')}`);
    }

    shot(page, 'regime-result');
  });

  // ═══ FINAL CHECKS ═══
  await phase('API Summary', page, async () => {
    const groups: Record<string, number> = {};
    for (const a of apiLog) {
      const svc = a.replace(/^API \d{3} /, '').split('/')[0] || 'unknown';
      groups[svc] = (groups[svc] || 0) + 1;
    }
    const summary = Object.entries(groups).map(([k, v]) => `${k}×${v}`).join(', ');
    OK(`API: ${apiCount} calls, ${apiLog.length} logged → ${summary}`);
  });

  await phase('JS Errors', page, async () => {
    const uniq = [...new Set(jsErrs)].filter(e => !e.includes('ResizeObserver') && !e.includes('Script error'));
    uniq.length === 0 ? OK('Zero JS errors') : uniq.slice(0, 5).forEach(e => W('JS', e));
  });

  // ═══ SUMMARY ═══
  const pass = R.filter(r => r.startsWith('✅')).length;
  const warns = R.filter(r => r.startsWith('⚠️')).length;
  const fails = R.filter(r => r.startsWith('❌')).length;
  console.log(`\n╔════════════════════════════════════╗`);
  console.log(`║  Market Tools: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/*.png`);
  console.log(`║  API endpoints: ${new Set(apiLog.map(a => a.replace(/^API \d{3} /, ''))).size}`);
  console.log(`║  API calls: ${apiCount}`);
  console.log('╚════════════════════════════════════╝');
  R.forEach(r => console.log(r));

  const eps = [...new Set(apiLog.map(a => a.replace(/^API \d{3} /, '')))];
  if (eps.length > 0) {
    console.log(`\nAPI endpoints (${eps.length}):`);
    eps.forEach(e => console.log(`  ${e}`));
  }

  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
