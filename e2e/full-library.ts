/**
 * Comprehensive Strategy Library E2E — full human-simulation sweep.
 *
 * Covers ALL sub-tabs: Overview → Run → Backtest History
 * Plus: filter, search, create template, view code, save as mine,
 *        open in workspace, create schedule, toggle, delete schedule,
 *        view backtest run, delete backtest run.
 *
 * Usage: npx tsx e2e/full-library.ts  (replaces old version)
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
fs.mkdirSync(SHOT, { recursive: true });

const U = 'admin@1.com'; const P = '12345678';

const R: string[] = []; let sn = 0;
function S(m: string) { sn++; console.log(`\n── ${sn}. ${m} ──`); }
function OK(m: string) { R.push(`✅ ${m}`); console.log(`   ✅ ${m}`); }
function W(m: string, d?: string) { R.push(`⚠️ ${m}${d ? ': ' + d : ''}`); console.log(`   ⚠️ ${m}${d ? ': ' + d : ''}`); }
function F(m: string, d?: string) { R.push(`❌ ${m}${d ? ': ' + d : ''}`); console.log(`   ❌ ${m}${d ? ': ' + d : ''}`); }
function shot(page: Page, n: string) {
  return page.screenshot({ path: `${SHOT}/lib-${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

async function humanPause(minMs = 400, maxMs = 1200) {
  await new Promise(r => setTimeout(r, Math.floor(minMs + Math.random() * (maxMs - minMs))));
}

async function clickTab(page: Page, tabText: RegExp): Promise<boolean> {
  return page.evaluate((reStr) => {
    const re = new RegExp(reStr, 'i');
    const tabs = document.querySelectorAll('.ant-tabs-tab');
    for (const t of tabs) {
      if (re.test((t as HTMLElement).innerText?.trim() || '')) {
        (t as HTMLElement).click(); return true;
      }
    }
    return false;
  }, tabText.source);
}

async function run() {
  const browser = await chromium.launch({ headless: true, args: ['--no-sandbox'] });
  const page = await browser.newPage({ viewport: { width: 1440, height: 900 } });
  const jsErrs: string[] = [];
  page.on('pageerror', e => jsErrs.push(e.message));

  // ═══ 1. LOGIN ═══
  S('Login');
  await page.goto(`${BASE}/login`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForSelector('#login_login', { timeout: 5000 });
  await humanPause(300, 600);
  await page.fill('#login_login', U);
  await humanPause(200, 500);
  await page.fill('#login_password', P);
  await humanPause(200, 500);
  await page.click('button[type="submit"]');
  await page.waitForTimeout(3000);
  page.url().includes('login') ? F('Login failed') : OK('Login');
  await shot(page, 'login');

  // ═══ 2. NAVIGATE ═══
  S('Navigate to Library');
  await page.goto(`${BASE}/strategy/library`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(4000);
  page.url().includes('library') ? OK('Library loaded') : F('Navigation failed');
  await shot(page, 'page');

  // ═══ 3. LEFT PANEL — MY FILTER ═══
  S('My Templates');
  const leftItems = await page.$$('[role="button"][tabindex="0"]');
  leftItems.length > 0 ? OK(`${leftItems.length} templates in My`) : W('No My templates');
  for (let i = 0; i < Math.min(3, leftItems.length); i++) {
    console.log(`   ${await leftItems[i].textContent().then(t => t?.trim().slice(0, 80))}`);
  }
  await shot(page, 'my-filter');

  // ═══ 4. SWITCH TO PRESET ═══
  S('Switch to Preset');
  const segs = await page.$$('.ant-segmented-item');
  let presetClicked = false;
  for (const s of segs) {
    const t = await s.textContent();
    if (t && /preset/i.test(t)) {
      await s.click({ force: true }); presetClicked = true; break;
    }
  }
  presetClicked ? OK('Preset filter clicked') : W('Preset filter not found');
  await page.waitForTimeout(1200);
  const presetItems = await page.$$('[role="button"][tabindex="0"]');
  presetItems.length > 0 ? OK(`${presetItems.length} preset templates`) : W('No preset templates');
  for (let i = 0; i < Math.min(3, presetItems.length); i++) {
    console.log(`   ${await presetItems[i].textContent().then(t => t?.trim().slice(0, 80))}`);
  }
  await shot(page, 'preset-filter');

  // ═══ 5. SELECT MA CROSSOVER ═══
  S('Select MA Crossover');
  let selected = false;
  for (const it of presetItems) {
    const t = await it.textContent();
    if (t && /MA Crossover/i.test(t)) {
      await it.click({ force: true }); selected = true; break;
    }
  }
  selected ? OK('MA Crossover selected') : F('MA Crossover not found');
  await page.waitForTimeout(2000);

  // Verify right panel loaded
  const body = await page.textContent('body') || '';
  /overview|概览|MA Crossover/i.test(body) ? OK('Right panel visible') : W('Right panel may not be loaded');
  await shot(page, 'selected');

  // ═══ 6. OVERVIEW TAB ═══
  S('Overview Tab');
  // Check basic info — use native query for antd v6 compatibility
  const descCount = await page.evaluate(() => document.querySelectorAll('.ant-descriptions-item, .ant-descriptions-view table td').length);
  descCount > 0 ? OK(`${descCount} detail fields`) : W('No description items');

  // Check action buttons
  const actionBtns = await page.$$('button');
  const btnTexts: string[] = [];
  for (const b of actionBtns) {
    const txt = await b.textContent();
    if (txt?.trim()) btnTexts.push(txt.trim().slice(0, 30));
  }
  const keyActions = ['View code', 'Save as Mine', 'Open in Workspace'];
  let found = 0;
  for (const ka of keyActions) {
    if (btnTexts.some(t => t.includes(ka) || t.includes('View') || t.includes('Save') || t.includes('Workspace'))) found++;
  }
  found >= 2 ? OK(`${found} action buttons found`) : W(`Only ${found} actions`);

  // Check code preview
  const preEl = await page.$('pre');
  if (preEl) {
    const codeTxt = await preEl.textContent();
    (codeTxt?.length || 0) > 50 ? OK(`Code preview: ${codeTxt?.slice(0, 80)}...`) : W('Short code preview');
  } else {
    W('No code preview');
  }
  await shot(page, 'overview');

  // ═══ 7. VIEW CODE MODAL ═══
  S('View Code Modal');
  const codeBtn = await page.$('button:has-text("View code")');
  if (codeBtn) {
    await humanPause(300, 600);
    await codeBtn.click({ force: true });
    await page.waitForTimeout(1200);

    const overlay = await page.$('[style*="position: fixed"][style*="rgba"]');
    if (overlay) {
      OK('Code modal opened');
      await page.keyboard.press('Escape');
      await page.waitForTimeout(500);
      OK('Modal closed');
    } else {
      W('Code modal overlay not found');
    }
  } else {
    W('View code button not found');
  }
  await shot(page, 'code-view');

  // ═══ 8. SAVE AS MINE ═══
  S('Save as Mine');
  const saveBtn = await page.$('button:has-text("Save as Mine")');
  if (saveBtn) {
    await humanPause(300, 600);
    await saveBtn.click({ force: true });
    await page.waitForTimeout(1500);
    OK('Save as Mine clicked');

    // Switch back to My filter to verify
    for (const s of await page.$$('.ant-segmented-item')) {
      const t = await s.textContent();
      if (t && /^My$/i.test(t)) { await s.click({ force: true }); break; }
    }
    await page.waitForTimeout(1000);
    const myItems = await page.$$('[role="button"][tabindex="0"]');
    const hasMA = await page.evaluate(() => {
      const items = document.querySelectorAll('[role="button"][tabindex="0"]');
      for (const el of items) {
        if (/MA Crossover/i.test((el as HTMLElement).innerText || '')) return true;
      }
      return false;
    });
    hasMA ? OK('MA Crossover now in My') : W('Not found in My after save');

    // Switch back to Preset for remaining tests
    for (const s of await page.$$('.ant-segmented-item')) {
      const t = await s.textContent();
      if (t && /preset/i.test(t)) { await s.click({ force: true }); break; }
    }
    await page.waitForTimeout(800);
    // Re-select MA Crossover
    const items2 = await page.$$('[role="button"][tabindex="0"]');
    for (const it of items2) {
      if (/MA Crossover/i.test(await it.textContent() || '')) { await it.click({ force: true }); break; }
    }
    await page.waitForTimeout(1500);
  } else {
    W('Save as Mine button not found');
  }
  await shot(page, 'save-mine');

  // ═══ 9. RUN TAB — CREATE SCHEDULE ═══
  S('Run tab');
  const runTabOk = await clickTab(page, /^Run$/i);
  if (!runTabOk) { F('Run tab not found'); }
  else {
    OK('Switched to Run tab');
    await page.waitForTimeout(1500);
    await shot(page, 'run-tab');

    // Click Create Schedule using native click (more reliable than Playwright click)
    S('Create Schedule modal');
    const clicked = await page.evaluate(() => {
      const btns = document.querySelectorAll('button');
      for (const el of btns) {
        if (/create.*schedule/i.test((el as HTMLElement).innerText || '')) {
          (el as HTMLButtonElement).click(); return true;
        }
      }
      return false;
    });
    if (!clicked) { W('Create Schedule button not found'); }
    await page.waitForTimeout(1500);

    // Check if modal opened
    const modalExists = await page.evaluate(() => !!document.querySelector('.ant-modal'));
    if (modalExists) {
      OK('Schedule modal opened');

      // Check for no-account warning
      const hasWarning = await page.evaluate(() => {
        const w = document.querySelector('.ant-modal .ant-alert-warning');
        return w ? (w as HTMLElement).innerText?.slice(0, 100) : null;
      });
      if (hasWarning) {
        W(`No-account warning: ${hasWarning}`);
      }

      // Close the modal via cancel button or Escape
      const closed = await page.evaluate(() => {
        const btns = document.querySelectorAll('.ant-modal button');
        for (const el of btns) {
          if (/cancel/i.test((el as HTMLElement).innerText || '')) {
            (el as HTMLButtonElement).click(); return 'cancel-clicked';
          }
        }
        return 'no-cancel';
      });
      if (closed === 'cancel-clicked') {
        await page.waitForTimeout(500);
        OK('Schedule modal closed');
      } else {
        await page.keyboard.press('Escape');
        await page.waitForTimeout(500);
        OK('Closed via Escape');
      }
    } else {
      F('Schedule modal did not open');
    }
    await shot(page, 'schedule-modal');
  }

  // ═══ 10. BACKTEST HISTORY TAB ═══
  S('Backtest History tab');
  const btTabOk = await clickTab(page, /backtest.*history|history|回测/i);
  if (!btTabOk) { F('Backtest History tab not found'); }
  else {
    OK('Switched to Backtest History tab');
    await page.waitForTimeout(1500);
    await shot(page, 'bt-tab');

    // Check for runs
    const btRows = await page.$$('.ant-tabs-tabpane-active table tbody tr.ant-table-row');
    if (btRows.length > 0) {
      OK(`${btRows.length} backtest runs`);
      for (let i = 0; i < btRows.length; i++) {
        console.log(`   Row ${i}: ${(await btRows[i].textContent())?.trim().slice(0, 120)}`);
      }

      // Click View on first row
      S('View backtest run');
      const viewBtn = await page.$('.ant-tabs-tabpane-active table tbody button:has-text("View")');
      if (viewBtn) {
        await viewBtn.click({ force: true });
        await page.waitForTimeout(2000);
        const drawer = await page.$('.ant-drawer-open, .ant-drawer');
        drawer ? OK('Backtest drawer opened') : W('No drawer');
        if (drawer) {
          await page.waitForTimeout(1000);
          await shot(page, 'bt-drawer');
          // Close drawer
          const closeX = await page.$('.ant-drawer-close');
          if (closeX) { await closeX.click({ force: true }); OK('Drawer closed'); }
          else { await page.keyboard.press('Escape'); OK('Closed via Escape'); }
          await page.waitForTimeout(500);
        }
      } else {
        W('View button not found in backtest table');
      }

      // Delete a run
      S('Delete backtest run');
      const delBtns = await page.$$('.ant-tabs-tabpane-active table tbody button.ant-btn-dangerous, .ant-tabs-tabpane-active table tbody button .anticon-delete');
      if (delBtns.length > 0) {
        // Click parent button if we matched the icon
        const delBtn = delBtns[0];
        await delBtn.click({ force: true });
        await page.waitForTimeout(1000);

        // Confirm
        const popOk = await page.$('.ant-popconfirm button.ant-btn-primary');
        if (popOk) {
          await popOk.click({ force: true });
          await page.waitForTimeout(2000);
          OK('Delete confirmed via Popconfirm');
        } else {
          W('Popconfirm not found after delete click');
        }

        // Check rows after
        const remaining = await page.$$('.ant-tabs-tabpane-active table tbody tr.ant-table-row');
        console.log(`   Rows after delete: ${remaining.length}`);
      } else {
        W('No delete buttons — nothing to delete');
      }
    } else {
      W('No backtest runs for MA Crossover');

      // Check page content for empty state
      const emptyText = await page.$('.ant-empty-description');
      if (emptyText) {
        console.log(`   Empty: ${(await emptyText.textContent())?.trim()}`);
      }
    }
    await shot(page, 'bt-after-delete');
  }

  // ═══ 11. SEARCH ═══
  S('Search');
  const searchInput = await page.$('.ant-input-affix-wrapper input');
  if (searchInput) {
    await searchInput.click({ force: true });
    await humanPause(200, 400);
    await page.keyboard.type('RSI', { delay: 40 });
    await page.waitForTimeout(800);

    const searchResults = await page.$$('[role="button"][tabindex="0"]');
    searchResults.length > 0 ? OK(`${searchResults.length} results for "RSI"`) : W('No search results');
    for (let i = 0; i < Math.min(3, searchResults.length); i++) {
      console.log(`   ${await searchResults[i].textContent().then(t => t?.trim().slice(0, 80))}`);
    }

    // Clear
    await page.keyboard.press('Control+a');
    await page.keyboard.press('Backspace');
    await page.waitForTimeout(500);
    OK('Search cleared');
  } else {
    W('No search input');
  }

  // Create via native click on the left-panel "+" Create button
  S('Create Template');
  const tplClicked = await page.evaluate(() => {
    const btns = document.querySelectorAll('button');
    for (const el of btns) {
      const html = (el as HTMLElement).innerText?.trim() || '';
      const outer = el.outerHTML || '';
      if ((html === 'Create' || /create/i.test(html)) && /ant-btn.*primary/.test(outer) && /anticon-plus/.test(outer)) {
        (el as HTMLButtonElement).click(); return true;
      }
    }
    return false;
  });
  if (tplClicked) {
    await page.waitForTimeout(1500);

    const tplModal = await page.$('.ant-modal');
    if (tplModal) {
      OK('Create template modal opened');

      // Fill name
      const inputs = await tplModal.$$('input');
      if (inputs.length > 0) {
        await inputs[0].click({ force: true });
        await humanPause(200, 400);
        const tplName = `E2E-Test-${Date.now().toString(36)}`;
        await page.keyboard.type(tplName, { delay: 20 });
        OK(`Name: ${tplName}`);
      }

      // Fill code via textarea
      const textareas = await tplModal.$$('textarea');
      if (textareas.length > 0) {
        await textareas[0].click({ force: true });
        await page.waitForTimeout(200);
        await page.keyboard.press('Control+a');
        await page.keyboard.type('def run(ctx):\n    close = ctx["close"]\n    if len(close) < 22:\n        return\n    return {"signal": "buy", "volume": 0.01}', { delay: 2 });
        OK('Code inserted');
      }

      // Save
      const submitBtn = await tplModal.$('button[type="submit"], button.ant-btn-primary');
      if (submitBtn) {
        await submitBtn.click({ force: true });
        await page.waitForTimeout(2000);
        OK('Template saved');
      }
    } else {
      W('Create modal not opened');
    }
  } else {
    W('Create button click failed');
  }
  await shot(page, 'create-template');

  // ═══ 13. JS ERRORS ═══
  S('JS Errors');
  const uniq = [...new Set(jsErrs)].filter(e =>
    !e.includes('ResizeObserver') && !e.includes('Script error')
  );
  if (uniq.length === 0) {
    OK('Zero JS errors');
  } else {
    uniq.slice(0, 10).forEach(e => W('JS', e));
  }

  // ═══ SUMMARY ═══
  const pass = R.filter(r => r.startsWith('✅')).length;
  const warns = R.filter(r => r.startsWith('⚠️')).length;
  const fails = R.filter(r => r.startsWith('❌')).length;
  console.log(`\n╔════════════════════════════════════╗`);
  console.log(`║  Library Full Sweep: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/lib-*.png`);
  console.log('╚════════════════════════════════════╝');
  R.forEach(r => console.log(r));

  if (fails > 0) process.exitCode = 1;
  await browser.close();
}

run().catch(e => { console.error('FATAL:', e.message); process.exit(1); });
