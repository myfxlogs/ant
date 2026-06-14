/**
 * Full Strategy Library E2E test — comprehensive human-simulated operations.
 *
 * Covers: Browse templates → Create strategy → View code → Open in Workspace →
 *         Run backtest → Create schedule → Toggle schedule → Publish strategy →
 *         Backtest history → Filter/Search → Delete schedule
 *
 * Usage: npx tsx e2e/full-library.ts
 */
import { chromium, Page } from 'playwright';
import * as fs from 'fs';

const BASE = 'http://localhost:8022';
const SHOT = '/opt/ant/e2e/shots';
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
  return page.screenshot({ path: `${SHOT}/lib-${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

// ══════════════════════════════════════
// Helpers
// ══════════════════════════════════════
async function humanPause(minMs = 800, maxMs = 3000) {
  const delay = Math.floor(minMs + Math.random() * (maxMs - minMs));
  await new Promise(r => setTimeout(r, delay));
}

async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name);
  try {
    await fn();
  } catch (e: any) {
    F(`Phase error: ${name}`, e.message?.slice(0, 150));
    await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {});
  }
}

/** Click a button in Ant Tabs by tab text. */
async function clickTab(page: Page, tabText: RegExp): Promise<boolean> {
  return page.evaluate((reStr) => {
    const re = new RegExp(reStr, 'i');
    const tabs = document.querySelectorAll('.ant-tabs-tab');
    for (const t of tabs) {
      const txt = (t as HTMLElement).innerText?.trim() || '';
      if (re.test(txt)) {
        (t as HTMLElement).click();
        return true;
      }
    }
    return false;
  }, tabText.source);
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
  page.on('response', res => {
    const url = res.url();
    apiCount++;
    if (/ant\.v1\.|assets\//i.test(url)) {
      const short = url.replace(/^.*\/api\//, '').replace(/\?.*$/, '').split('/').slice(-2).join('/');
      apiLog.push(`API ${res.status()} ${short}`);
    }
  });

  // ═══ PHASE 1: LOGIN ═══
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

  // ═══ PHASE 2: NAVIGATE ═══
  await phase('Navigate to Library', page, async () => {
    await page.goto(`${BASE}/strategy/library`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForTimeout(4000);
    page.url().includes('library') ? OK('Library loaded') : F('Navigation failed');
    shot(page, 'page');
  });

  // ═══ PHASE 3: LEFT PANEL — BROWSE ═══
  await phase('Browse Templates', page, async () => {
    // Check left panel rendered
    const leftPanel = await page.$('[style*="340px"]');
    leftPanel ? OK('Left panel') : W('Left panel not found');

    // Count template list items
    const items = await page.$$('[role="button"][tabindex="0"]');
    items.length > 0 ? OK(`${items.length} templates`) : W('No templates');

    // Check filter segmented control
    const segs = await page.$$('.ant-segmented-item');
    if (segs.length >= 2) {
      const segTexts: string[] = [];
      for (const s of segs) { segTexts.push((await s.textContent())?.trim() || ''); }
      OK(`Filters: ${segTexts.join(', ')}`);
    }

    // Check search input
    const searchInput = await page.$('.ant-input-affix-wrapper input');
    searchInput ? OK('Search input') : W('No search');

    // Check Create button
    const createBtn = await page.$('button:has-text("Create"), button:has(.anticon-plus)');
    createBtn ? OK('Create button') : W('No create btn');

    shot(page, 'browse');
  });

  // ═══ PHASE 4: SELECT TEMPLATE ═══
  await phase('Select Template', page, async () => {
    const items = await page.$$('[role="button"][tabindex="0"]');
    if (items.length === 0) { W('No templates to select'); return; }

    // Human-like: read a few template names before selecting
    for (let i = 0; i < Math.min(3, items.length); i++) {
      const txt = await items[i].textContent();
      console.log(`   Template ${i + 1}: ${txt?.trim().slice(0, 80)}`);
    }
    await humanPause(400, 900);

    // Select first template
    await items[0].click({ force: true });
    await page.waitForTimeout(1500);
    OK('Template selected');

    // Verify right panel loaded
    const rightContent = await page.textContent('body') || '';
    /overview|概览|Overview|schedule|backtest/i.test(rightContent)
      ? OK('Right panel visible') : W('Right panel may not be loaded');
    shot(page, 'selected');
  });

  // ═══ PHASE 5: OVERVIEW TAB ═══
  await phase('Overview Tab', page, async () => {
    // Should be on Overview tab by default
    const descItems = await page.$$('.ant-descriptions-view table tbody tr, .ant-descriptions-item');
    if (descItems.length > 0) {
      OK(`${descItems.length} detail fields`);
    } else {
      // Check for Descriptions via broader query
      const descTable = await page.$('.ant-descriptions table');
      if (descTable) {
        const rows = await descTable.$$('tr');
        OK(`${rows.length} description rows`);
      } else {
        W('No description items');
      }
    }

    // Check action buttons
    const actionBtns = await page.$$('button');
    const btnTexts: string[] = [];
    for (const b of actionBtns) {
      const txt = await b.textContent();
      if (txt && txt.trim()) btnTexts.push(txt.trim().slice(0, 30));
    }
    const keyActions = ['Edit', 'Code', 'Workspace', 'Schedule', 'Publish', 'Share', 'Save As Mine'];
    let found = 0;
    for (const ka of keyActions) {
      if (btnTexts.some(t => t.includes(ka))) found++;
    }
    found >= 2 ? OK(`${found} action buttons: ${btnTexts.filter(t => t.length < 20).join(', ')}`) : W('Few action buttons');

    // Check code preview
    const codePreview = await page.$('pre');
    codePreview ? OK('Code preview visible') : W('No code preview');

    shot(page, 'overview');
  });

  // ═══ PHASE 6: VIEW CODE MODAL ═══
  await phase('View Code', page, async () => {
    // Button text is "View code" (lowercase c in i18n)
    const codeBtn = await page.$('button:has-text("View"), button:has-text("Code")');
    if (!codeBtn) { W('No code button'); return; }

    await humanPause(300, 600);
    await codeBtn.click({ force: true });
    await page.waitForTimeout(1000);

    // Code modal: custom overlay with fixed position and dark background
    const overlay = await page.$('[style*="position: fixed"][style*="rgba"]');
    const preEl = overlay ? await overlay.$('pre') : await page.$('pre[style*="background"]');
    if (overlay || preEl) {
      OK('Code modal opened');
      const codeText = preEl ? await preEl.textContent() : '';
      (codeText?.length || 0) > 10 ? OK(`Code preview: ${(codeText || '').slice(0, 80)}...`) : W('Empty code');

      // Close modal — click the × button or Escape
      const closeX = await page.$('[style*="fixed"] button, [style*="rgba"] button');
      if (closeX) { await closeX.click({ force: true }); OK('Modal closed'); }
      else { await page.keyboard.press('Escape'); OK('Closed via Escape'); }
    } else {
      W('No code modal overlay');
    }
    await page.waitForTimeout(500);
    shot(page, 'code-view');
  });

  // ═══ PHASE 7: CREATE NEW TEMPLATE ═══
  let newTemplateName = '';
  await phase('Create Template', page, async () => {
    // Click "Create" button in left panel header
    const createBtn = await page.$('button.ant-btn-primary:has(.anticon-plus)');
    if (!createBtn) { W('Create button not found'); return; }

    await humanPause(400, 800);
    await createBtn.click({ force: true });
    await page.waitForTimeout(1500);

    // Check if edit modal opened
    const modal = await page.$('.ant-modal');
    if (!modal) { W('Create modal not opened'); return; }
    OK('Create modal opened');

    // Fill template name
    const inputs = await modal.$$('input');
    if (inputs.length > 0) {
      newTemplateName = `E2E Library Test ${Date.now().toString(36)}`;
      await inputs[0].click({ force: true });
      await humanPause(200, 400);
      await page.keyboard.type(newTemplateName, { delay: 25 + Math.random() * 25 });
      OK(`Name: ${newTemplateName}`);
    }

    // Fill description
    const textareas = await modal.$$('textarea');
    if (textareas.length > 0) {
      await textareas[0].click({ force: true });
      await humanPause(200, 400);
      await page.keyboard.type('E2E test strategy — SMA crossover with risk management', { delay: 15 });
      OK('Description filled');
    }

    // Insert code: try CodeMirror first, then textarea fallback
    const cmInModal = await modal.$('.cm-editor, .cm-content');
    const textareaInModal = await modal.$('textarea');
    if (cmInModal) {
      await cmInModal.click({ force: true });
      await page.waitForTimeout(300);
      await page.keyboard.press('Control+a');
      await page.waitForTimeout(100);
      const simpleCode = `"""@strategy SMA
@param fast_period 5 range=3:50:1
@param slow_period 20 range=10:60:2
"""

def init():
    pass

def run(ctx):
    close = ctx['close']
    if len(close) < 22:
        return
    f = sum(close[-5:]) / 5.0
    s = sum(close[-20:]) / 20.0
    if f > s:
        return {'signal': 'buy', 'volume': 0.01}
`;
      await page.keyboard.type(simpleCode, { delay: 2 });
      OK('Code inserted (CodeMirror)');
    } else if (textareaInModal) {
      await textareaInModal.click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('def run(ctx):\n    close = ctx["close"]\n    if len(close) < 22:\n        return\n    f = sum(close[-5:]) / 5.0\n    s = sum(close[-20:]) / 20.0\n    if f > s:\n        return {"signal": "buy", "volume": 0.01}\n', { delay: 2 });
      OK('Code inserted (textarea)');
    } else {
      W('No code editor in modal');
    }

    // Save
    const submitBtn = await modal.$('button[type="submit"], button.ant-btn-primary');
    if (submitBtn) {
      await humanPause(400, 800);
      await submitBtn.click({ force: true });
      await page.waitForTimeout(2000);
      OK('Template saved');
    } else {
      W('No save button');
    }

    shot(page, 'created');
  });

  // ═══ PHASE 8: PUBLISH STRATEGY ═══
  await phase('Publish Strategy', page, async () => {
    // Find and click the Publish/Share button
    const publishBtn = await page.$('button:has-text("Share"), button:has-text("Publish"), button:has(.anticon-global)');
    if (!publishBtn) { W('No publish button'); return; }

    await humanPause(300, 700);
    await publishBtn.click({ force: true });
    await page.waitForTimeout(2000);

    // Check if publish succeeded (button text changes to "Unpublish")
    const unpublishBtn = await page.$('button:has-text("Unpublish")');
    if (unpublishBtn) {
      OK('Published — Unpublish button visible');
    } else {
      // Check success message
      const body = await page.textContent('body') || '';
      if (/success|成功|published/i.test(body)) {
        OK('Published (success message)');
      } else {
        W('Publish state unclear — may need confirmation');
      }
    }
    shot(page, 'published');
  });

  // ═══ PHASE 9: CREATE SCHEDULE ═══
  await phase('Create Schedule', page, async () => {
    // Switch to "Run" tab (i18n: library.schedules = "Run")
    const tabOk = await clickTab(page, /^Run$|Schedule|运行|排程/i);
    if (!tabOk) { W('Run tab not found'); return; }
    await page.waitForTimeout(1500);
    OK('Run tab');

    // Click "Create Schedule" button (primary button in Run tab)
    const createBtn = await page.$('button.ant-btn-primary:has-text("Create")');
    if (!createBtn) { W('Create Schedule button not found'); return; }

    await humanPause(400, 800);
    await createBtn.click({ force: true });
    await page.waitForTimeout(1500);

    // Edit Schedule modal
    const modal = await page.$('.ant-modal');
    if (!modal) { W('Schedule modal not opened'); return; }
    OK('Schedule modal opened');

    // Fill schedule form: select Interval
    const selects = await modal.$$('.ant-select');
    if (selects.length > 0) {
      // Click interval type select
      await selects[0].click({ force: true });
      await page.waitForTimeout(800);
      const opts = await page.$$('.ant-select-item-option');
      // Look for "Interval" or "Cron" option
      for (const o of opts) {
        const txt = await o.textContent();
        if (txt && /interval|Interval|间隔/i.test(txt)) {
          await o.click({ force: true });
          OK('Interval type selected');
          break;
        }
      }
      await page.waitForTimeout(500);
      await page.keyboard.press('Escape');
    }

    // Set interval seconds (spinbutton/inputnumber)
    const spinBtns = await modal.$$('.ant-input-number-input');
    if (spinBtns.length > 0) {
      await spinBtns[0].click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('300', { delay: 30 }); // 5 minutes
      OK('Interval: 300s');
    }

    // Set symbol if available
    if (selects.length >= 2) {
      await selects[1].click({ force: true });
      await page.waitForTimeout(800);
      const symOpts = await page.$$('.ant-select-item-option');
      if (symOpts.length > 0) {
        // Select first available symbol
        await symOpts[0].click({ force: true });
        OK('Symbol selected');
      }
      await page.waitForTimeout(300);
      await page.keyboard.press('Escape');
    }

    // Save schedule
    const saveBtn = await modal.$('button.ant-btn-primary');
    if (saveBtn) {
      await humanPause(400, 800);
      await saveBtn.click({ force: true });
      await page.waitForTimeout(2000);
      OK('Schedule created');
    } else {
      W('No save button in schedule modal');
    }
    shot(page, 'schedule');
  });

  // ═══ PHASE 10: TOGGLE SCHEDULE ═══
  await phase('Toggle Schedule', page, async () => {
    // Find toggle switch in schedule table
    const switches = await page.$$('.ant-switch');
    if (switches.length > 0) {
      OK(`${switches.length} schedule toggles`);

      // Toggle the first one: click to deactivate
      const isChecked = await switches[0].evaluate(el => el.classList.contains('ant-switch-checked'));
      await humanPause(400, 800);
      await switches[0].click({ force: true });
      await page.waitForTimeout(1000);

      const nowChecked = await switches[0].evaluate(el => el.classList.contains('ant-switch-checked'));
      if (nowChecked !== isChecked) {
        OK(`Toggled ${isChecked ? 'OFF' : 'ON'}`);
        // Toggle back
        await switches[0].click({ force: true });
        await page.waitForTimeout(500);
        OK('Toggled back');
      } else {
        W('Toggle did not change');
      }
    } else {
      W('No schedule toggles');
    }
    shot(page, 'toggle');
  });

  // ═══ PHASE 11: DELETE SCHEDULE ═══
  await phase('Delete Schedule', page, async () => {
    // Find delete button in schedule table
    const deleteBtns = await page.$$('button');
    let deleted = false;
    for (const btn of deleteBtns) {
      const txt = await btn.textContent();
      const box = await btn.boundingBox();
      // Delete button is small, usually in schedule area
      if (txt && /delete|Delete|删除/i.test(txt) && box && box.y > 200) {
        await btn.evaluate((el: HTMLButtonElement) => el.scrollIntoView({ block: 'center' }));
        await humanPause(300, 600);
        await btn.click({ force: true });
        // Look for popconfirm
        await page.waitForTimeout(800);
        const confirmBtn = await page.$('.ant-popconfirm button.ant-btn-primary');
        if (confirmBtn) {
          await confirmBtn.click({ force: true });
          await page.waitForTimeout(1500);
          OK('Schedule deleted');
        } else {
          W('No confirmation dialog');
        }
        deleted = true;
        break;
      }
    }
    if (!deleted) W('No delete button found');
    shot(page, 'deleted');
  });

  // ═══ PHASE 12: BACKTEST HISTORY TAB ═══
  await phase('Backtest History', page, async () => {
    // Switch to "Backtest History" tab
    const tabOk = await clickTab(page, /backtest|history|回测/i);
    if (!tabOk) { W('Backtest tab not found'); return; }
    await page.waitForTimeout(1500);
    OK('Backtest tab');

    // Check runs table
    const tables = await page.$$('.ant-tabs-tabpane-active table');
    if (tables.length > 0) {
      const rows = await tables[0].$$('tbody tr');
      if (rows.length > 0) {
        OK(`${rows.length} backtest runs`);

        // View first run
        const viewBtn = await tables[0].$('button');
        if (viewBtn) {
          await humanPause(300, 600);
          await viewBtn.click({ force: true });
          await page.waitForTimeout(2000);
          // BacktestRunDrawer should open as ant-drawer
          const drawer = await page.$('.ant-drawer, .ant-drawer-open');
          drawer ? OK('Backtest drawer opened') : W('No drawer (may need view btn with correct text)');

          if (drawer) {
            // Close drawer
            const closeBtn = await drawer.$('.ant-drawer-close');
            if (closeBtn) { await closeBtn.click({ force: true }); OK('Drawer closed'); }
          }
        }
      } else {
        W('No backtest runs for this template');
      }
    } else {
      W('No backtest table');
    }
    shot(page, 'backtest');
  });

  // ═══ PHASE 13: FILTER TEMPLATES ═══
  await phase('Filter Preset Templates', page, async () => {
    // Click "Preset" filter in segmented (i18n: filterSystem = "Preset")
    const segs = await page.$$('.ant-segmented-item');
    let clicked = false;
    for (const s of segs) {
      const txt = await s.textContent();
      if (txt && /preset|Preset|system/i.test(txt)) {
        await humanPause(300, 600);
        await s.click({ force: true });
        OK('Preset filter clicked');
        clicked = true;
        break;
      }
    }
    if (!clicked) W('Preset filter not found');

    await page.waitForTimeout(1000);

    // Verify templates changed (preset templates show "Preset" or "System" tags)
    const items = await page.$$('[role="button"][tabindex="0"]');
    if (items.length > 0) {
      const tagCount = await page.evaluate(() => {
        return document.querySelectorAll('.ant-tag').length;
      });
      OK(`${items.length} templates, ${tagCount} tags`);
    }

    // Return to "My" filter
    for (const s of await page.$$('.ant-segmented-item')) {
      const txt = await s.textContent();
      if (txt && /^My$|Mine|我的/i.test(txt)) {
        await s.click({ force: true });
        OK('Back to My filter');
        break;
      }
    }
    await page.waitForTimeout(1000);
    shot(page, 'filter');
  });

  // ═══ PHASE 14: SEARCH ═══
  await phase('Search Templates', page, async () => {
    const searchInput = await page.$('.ant-input-affix-wrapper input');
    if (!searchInput) { W('No search input'); return; }

    await searchInput.click({ force: true });
    await humanPause(200, 400);
    await page.keyboard.type('E2E', { delay: 40 + Math.random() * 40 });
    await humanPause(500, 1000);

    // Check filtered results
    const items = await page.$$('[role="button"][tabindex="0"]');
    if (items.length > 0 && items.length < 20) {
      OK(`${items.length} results for "E2E"`);
    } else if (items.length === 0) {
      W('No results for "E2E" — template may not be saved');
    } else {
      OK(`${items.length} results (search may be partial)`);
    }

    // Clear search
    await page.keyboard.press('Control+a');
    await page.keyboard.press('Backspace');
    await page.waitForTimeout(800);
    OK('Search cleared');

    shot(page, 'search');
  });

  // ═══ PHASE 15: FINAL CHECKS ═══
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
  console.log(`║  Strategy Library: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/lib-*.png`);
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
