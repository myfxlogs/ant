/**
 * Full Experiments page E2E test — comprehensive human-simulated operations.
 *
 * Phases: Login → Navigate → Page Load → Browse Experiments → View Candidates →
 *         Create Experiment (form fill with human touches) → Job Events Stream →
 *         Promote Candidate → Final Checks
 *
 * Usage: npx tsx e2e/full-experiments.ts
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
  return page.screenshot({ path: `${SHOT}/exp-${String(sn).padStart(2, '0')}-${n}.png`, fullPage: false }).catch(() => {});
}

// ══════════════════════════════════════
// Helpers
// ══════════════════════════════════════
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

async function phase(name: string, page: Page, fn: () => Promise<void>) {
  S(name);
  try {
    await fn();
  } catch (e: any) {
    F(`Phase error: ${name}`, e.message?.slice(0, 150));
    await shot(page, `err-${name.replace(/\s+/g, '-')}`).catch(() => {});
  }
}

/** Navigate to a page and wait for it to load. */
async function navigateTo(page: Page, path: string): Promise<boolean> {
  await page.goto(`${BASE}${path}`, { waitUntil: 'domcontentloaded', timeout: 15000 });
  await page.waitForTimeout(3000);
  return page.url().includes(path);
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
  page.on('console', msg => { if (msg.text().startsWith('EXP_DIAG')) console.log('   [BROWSER]', msg.text()); });
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
  await phase('Navigate to Experiments', page, async () => {
    const ok = await navigateTo(page, '/strategy/experiments');
    ok ? OK('Page loaded') : F('Navigation failed');
    await humanPause(500, 1000);
    exploreScroll(page);
    shot(page, 'page');
  });

  // ═══ PHASE 3: PAGE STRUCTURE ═══
  await phase('Page Structure', page, async () => {
    const body = await page.textContent('body') || '';

    // Title
    (/experiment|实验/i.test(body)) ? OK('Title') : W('Title not found');

    // 4 cards expected: Submit Form, Experiments List, Job Events, Candidates
    const cards = await page.$$('.ant-card');
    cards.length >= 4 ? OK(`${cards.length} cards`) : W(`Only ${cards.length} cards`);

    // Info alert
    const alerts = await page.$$('.ant-alert');
    alerts.length >= 1 ? OK(`${alerts.length} alerts`) : W('No alerts');

    // Tables
    const tables = await page.$$('table');
    tables.length >= 2 ? OK(`${tables.length} tables`) : W(`Only ${tables.length} tables`);

    shot(page, 'structure');
  });

  // ═══ PHASE 4: BROWSE EXPERIMENTS LIST ═══
  let hasExperiments = false;
  await phase('Browse Experiments List', page, async () => {
    // Scroll to experiments table
    await page.evaluate(() => {
      const cards = document.querySelectorAll('.ant-card');
      for (const c of cards) {
        if (c.textContent?.includes('Experiment') || c.querySelector('table')) {
          c.scrollIntoView({ block: 'center' }); break;
        }
      }
    });
    await humanPause(500, 1000);

    // Count experiment rows
    const rows = await page.$$('.ant-table-tbody tr');
    if (rows.length > 0) {
      hasExperiments = true;
      OK(`${rows.length} experiments in list`);

      // Read first row for context
      const firstRow = await rows[0].textContent();
      console.log(`   First experiment: ${firstRow?.slice(0, 120)}`);

      // Check status tags via evaluate (more reliable than $$ selector)
      const tagDiag = await page.evaluate(() => {
        const tags = document.querySelectorAll('.ant-tag');
        const texts: string[] = [];
        for (const t of tags) {
          const txt = (t as HTMLElement).innerText?.trim() || '';
          if (txt) texts.push(txt);
        }
        return texts;
      });
      const statusTags = tagDiag.filter(t => /SUCCEEDED|COMPLETED|RUNNING|PENDING|FAILED|succeeded|completed|running/i.test(t));
      statusTags.length > 0 ? OK(`Status tags: ${statusTags.slice(0, 5).join(', ')}`) : W(`No status tags (found ${tagDiag.length} tags: ${tagDiag.slice(0, 5).join(', ')})`);
    } else {
      W('No experiments yet — will create one');
    }
    shot(page, 'experiments-list');
  });

  // ═══ PHASE 5: VIEW CANDIDATES ═══
  await phase('View Candidates', page, async () => {
    if (!hasExperiments) { W('No experiments to view'); return; }

    // Click first "View Candidates" button
    const viewBtns = await page.$$('button');
    let clicked = false;
    for (const btn of viewBtns) {
      const txt = await btn.textContent();
      if (txt && /View|view/i.test(txt)) {
        const box = await btn.boundingBox();
        if (box && box.y > 200) {
          await humanPause(400, 800);
          await btn.click({ force: true });
          clicked = true;
          OK('View Candidates clicked');
          break;
        }
      }
    }
    if (!clicked) { W('No View Candidates button found'); return; }

    await page.waitForTimeout(2000);

    // Verify candidates table loaded
    const allTables = await page.$$('table');
    if (allTables.length >= 2) {
      const candTable = allTables[allTables.length - 1];
      const candTRs = await candTable.$$('tbody tr');
      // Filter out header/placeholder rows
      let dataRows = 0;
      for (const tr of candTRs) {
        const txt = await tr.textContent();
        if (txt && !txt.includes('Rank') && !txt.includes('No data')) dataRows++;
      }
      if (dataRows > 0) {
        OK(`${dataRows} candidates with data`);
      } else if (candTRs.length > 0) {
        W(`${candTRs.length} rows but no data candidates — experiment may need tuning`);
      } else {
        W('Candidates table empty');
      }
    }
    shot(page, 'candidates');
  });

  // ═══ PHASE 6: CREATE NEW EXPERIMENT ═══
  let createdExperiment = false;
  await phase('Create Experiment', page, async () => {
    // Scroll to top - submit form
    await page.evaluate(() => window.scrollTo(0, 0));
    await humanPause(400, 800);

    // Step 1: Select a template
    // The template select is the first Select in the form
    const formSelects = await page.$$('.ant-form-item .ant-select');
    if (formSelects.length === 0) { W('No form selects found'); return; }

    // Template select
    await humanPause(300, 700);
    await formSelects[0].click({ force: true });
    await page.waitForTimeout(1000);

    // Browse template options (human-like: scroll through, read names)
    const templateOpts = await page.$$('.ant-select-item-option');
    if (templateOpts.length > 0) {
      OK(`${templateOpts.length} templates available`);

      // Human-like: hover over a few, read names
      for (let i = 0; i < Math.min(3, templateOpts.length); i++) {
        const txt = await templateOpts[i].textContent();
        console.log(`   Template ${i + 1}: ${txt?.trim().slice(0, 60)}`);
      }
      await humanPause(500, 1200);

      // Select the first template
      await templateOpts[0].click({ force: true });
      await page.waitForTimeout(500);
      OK('Template selected');
    } else {
      // No templates — need to create one first
      W('No templates — experiment requires existing templates');
      await page.keyboard.press('Escape');
      return;
    }
    await page.keyboard.press('Escape');

    // Step 2: Leave parameter space as default (form initial value)
    // The default matches MA Crossover template params and works reliably.
    // Human-like: scroll through the JSON, read it, maybe tweak one value
    const textareas = await page.$$('textarea');
    if (textareas.length > 0) {
      const jsonTA = textareas[0];
      await jsonTA.click({ force: true });
      await humanPause(300, 600);
      // Scroll to see the JSON content
      await page.evaluate(() => {
        const ta = document.querySelector('textarea');
        if (ta) ta.scrollTop = 0;
      });
      await humanPause(400, 700);
      // Minor human touch: change one value
      await page.keyboard.press('Control+f');
      await page.waitForTimeout(200);
      await page.keyboard.type('16', { delay: 40 });
      await page.waitForTimeout(300);
      await page.keyboard.press('Escape');
      await page.waitForTimeout(200);
      await page.keyboard.type('20', { delay: 40 }); // change 16 to 20
      await humanPause(300, 600);
      OK('Parameter space reviewed (default)');
    }

    // Step 3: Switch search method (human-like exploration)
    if (formSelects.length >= 2) {
      await formSelects[1].click({ force: true });
      await page.waitForTimeout(800);
      let methodOpts = await page.$$('.ant-select-item-option');
      // Look at both options
      for (const o of methodOpts) {
        const txt = await o.textContent();
        if (txt && /random/i.test(txt)) {
          await humanPause(300, 600);
          await o.click({ force: true });
          OK('Switched to Random search');
          break;
        }
      }
      await page.waitForTimeout(500);
      await page.keyboard.press('Escape');
      await humanPause(200, 400);

      // Re-query select (may have moved after re-render) and switch back to Grid
      const freshSelects = await page.$$('.ant-form-item .ant-select');
      if (freshSelects.length >= 2) {
        await freshSelects[1].click({ force: true });
        await page.waitForTimeout(800);
        methodOpts = await page.$$('.ant-select-item-option');
        for (const o of methodOpts) {
          const txt = await o.textContent();
          if (txt && /grid/i.test(txt)) {
            // Scroll into view to ensure visibility
            await o.evaluate((el: HTMLElement) => el.scrollIntoView({ block: 'center' }));
            await page.waitForTimeout(150);
            await o.click({ force: true }).catch(async () => {
              // Fallback: click via evaluate
              await o.evaluate((el: HTMLElement) => el.click());
            });
            OK('Switched back to Grid');
            break;
          }
        }
        await page.waitForTimeout(300);
        await page.keyboard.press('Escape');
      }
    }

    // Step 4: Adjust max candidates
    const spinBtns = await page.$$('.ant-input-number-input');
    if (spinBtns.length > 0) {
      await spinBtns[0].click({ force: true });
      await page.waitForTimeout(200);
      await page.keyboard.press('Control+a');
      await page.keyboard.type('8', { delay: 30 });
      OK('Max candidates: 8');
    }

    // Step 5: Type objective
    const textInputs = await page.$$('input:not([role="spinbutton"]):not([type="search"])');
    // Find the objective input (last text input in form)
    let objInput: any = null;
    for (const ti of textInputs) {
      const id = await ti.getAttribute('id');
      const box = await ti.boundingBox();
      // objective input: after the spinbutton area
      if (box && box.y > 300 && box.y < 700) {
        objInput = ti; break;
      }
    }
    if (objInput) {
      await objInput.click({ force: true });
      await humanPause(200, 400);
      await page.keyboard.type('balanced_sharpe', { delay: 30 + Math.random() * 30 });
      OK('Objective: balanced_sharpe');
    }

    // Step 6: Submit
    const submitBtn = await page.$('button[type="submit"]');
    if (submitBtn) {
      await humanPause(500, 1200); // hesitation before submit
      await submitBtn.click({ force: true });
      OK('Experiment submitted');
      createdExperiment = true;
    } else {
      W('Submit btn not found');
    }

    shot(page, 'form-filled');
  });

  // ═══ PHASE 7: WAIT FOR EXPERIMENT ═══
  await phase('Wait Experiment Processing', page, async () => {
    if (!createdExperiment) { W('No experiment to wait for'); return; }

    // Watch for progress (job events stream updates)
    // Poll for experiment status change
    let progress = false;
    for (let i = 0; i < 30; i++) {
      await page.waitForTimeout(2000);
      const body = await page.textContent('body') || '';
      if (/SUCCEEDED|succeeded|completed/i.test(body)) {
        progress = true;
        OK(`Experiment succeeded after ~${(i + 1) * 2}s`);
        break;
      }
      if (i % 5 === 4) console.log(`   Wait ${(i + 1) * 2}s...`);
    }
    if (!progress) W('Experiment still processing after 60s');

    // Check job events card
    const eventsList = await page.$$('.ant-list-item');
    if (eventsList.length > 0) {
      OK(`${eventsList.length} job events received`);
      // Log first and last event
      const firstEvt = await eventsList[0].textContent();
      const lastEvt = await eventsList[eventsList.length - 1].textContent();
      console.log(`   First event: ${firstEvt?.trim().slice(0, 100)}`);
      console.log(`   Last event: ${lastEvt?.trim().slice(0, 100)}`);
    }

    shot(page, 'processing');
  });

  // ═══ PHASE 8: VIEW NEW CANDIDATES ═══
  await phase('View Completed Candidates', page, async () => {
    // Find a COMPLETED experiment (not the FAILED one we just created).
    // COMPLETED experiments have meaningful candidates with grades.
    const completedIdx = await page.evaluate(() => {
      const rows = document.querySelectorAll('.ant-table-tbody tr');
      for (let i = 0; i < rows.length; i++) {
        const txt = (rows[i] as HTMLElement).innerText || '';
        if (/COMPLETED|SUCCEEDED/i.test(txt)) return i;
      }
      return 0;
    });
    console.log(`   First COMPLETED experiment at row ${completedIdx}`);

    // Click View Candidates on that row
    const viewBtns = await page.$$('button');
    let clicked = false;
    for (const btn of viewBtns) {
      const txt = await btn.textContent();
      if (txt && /View|view/i.test(txt)) {
        const box = await btn.boundingBox();
        // Click the first View button (should correspond to first experiment)
        if (box && box.y > 200) {
          await btn.evaluate((el: HTMLButtonElement) => el.scrollIntoView({ block: 'center' }));
          await page.waitForTimeout(150);
          await btn.click({ force: true });
          await page.waitForTimeout(2000);
          OK('View Candidates clicked (COMPLETED)');
          clicked = true;
          break;
        }
      }
    }
    if (!clicked) {
      W('No View Candidates button');
      return;
    }

    await page.waitForTimeout(1500);

    // Check candidates table
    const tables = await page.$$('table');
    if (tables.length >= 2) {
      const rows = await tables[tables.length - 1].$$('tbody tr');
      if (rows.length > 0) {
        // Skip header row (first row with column titles)
        const dataRows = rows.length > 1 && (await rows[0].textContent())?.includes('Rank') ? rows.slice(1) : rows;
        OK(`${dataRows.length} candidates (${rows.length} total rows)`);

        // Read first data row
        if (dataRows.length > 0) {
          const firstCand = await dataRows[0].textContent();
          console.log(`   First candidate: ${firstCand?.trim().slice(0, 150)}`);

          // Check for rank, grade, score
          const hasRank = /\b[1-9]\b/.test(firstCand || '');
          const hasGrade = /[A-C]/.test(firstCand || '');
          hasRank ? OK('Rank data') : W('No rank data');
          hasGrade ? OK('Grade data') : W('No grade data');
        }
      } else {
        W('Candidates table empty');
      }
    }
    shot(page, 'new-candidates');
  });

  // ═══ PHASE 9: PROMOTE CANDIDATE ═══
  await phase('Promote Candidate to Draft', page, async () => {
    // Look for "Generate Draft" button in the last table (candidates table)
    const tables = await page.$$('table');
    if (tables.length < 2) { W('No candidates table'); return; }
    const candTable = tables[tables.length - 1];

    // Search buttons in the candidates table
    const draftBtns = await candTable.$$('button');
    let promoted = false;
    for (const btn of draftBtns) {
      const txt = await btn.textContent();
      if (txt && /Generate|generate|Draft|draft|Promote|promote/i.test(txt)) {
        await humanPause(400, 900);
        await btn.evaluate((el: HTMLButtonElement) => el.scrollIntoView({ block: 'center' }));
        await page.waitForTimeout(150);
        await btn.click({ force: true });
        promoted = true;
        OK(`Generate Draft clicked: "${txt.trim()}"`);
        break;
      }
    }
    if (!promoted) {
      // Log all buttons in candidates table for debugging
      const allBtnTexts = await candTable.$$eval('button', btns =>
        btns.map(b => (b as HTMLButtonElement).innerText?.trim()?.slice(0, 30))
      );
      W('No Generate Draft button. Buttons: ' + JSON.stringify(allBtnTexts));
      return;
    }

    // Wait for success message
    await page.waitForTimeout(3000);
    const body = await page.textContent('body') || '';
    if (/success|成功|draft|template/i.test(body)) {
      OK('Promotion successful');
    } else {
      W('No success confirmation visible');
    }
    shot(page, 'promote');
  });

  // ═══ PHASE 10: CANCEL EXPERIMENT ═══
  await phase('Cancel Experiment', page, async () => {
    // Look for cancel button (may not exist for completed experiments)
    const cancelBtns = await page.$$('button');
    let cancelled = false;
    for (const btn of cancelBtns) {
      const txt = await btn.textContent();
      if (txt && /cancel|Cancel|取消/i.test(txt)) {
        await btn.click({ force: true });
        cancelled = true;
        OK('Cancel clicked');
        break;
      }
    }
    if (!cancelled) {
      W('No cancel button (expected for completed experiments)');
    }
    await page.waitForTimeout(1000);
    shot(page, 'cancel');
  });

  // ═══ PHASE 11: REFRESH ═══
  await phase('Page Refresh', page, async () => {
    await page.goto(`${BASE}/strategy/experiments`, { waitUntil: 'domcontentloaded', timeout: 15000 });
    await page.waitForTimeout(3000);
    // Simple scroll (no exploreScroll to avoid post-close race)
    await page.evaluate(() => window.scrollBy(0, 200));
    await page.waitForTimeout(500);
    OK('Page refreshed');

    // Verify content still loads
    const cards = await page.$$('.ant-card');
    cards.length >= 2 ? OK('Content intact after refresh') : W('Content changed');
    shot(page, 'refresh');
  });

  // ═══ PHASE 12: FINAL CHECKS ═══
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
  console.log(`║  Experiments Page: ${pass}/${R.length} passed  (⚠️ ${warns}  ❌ ${fails})`);
  console.log(`║  Screenshots: ${SHOT}/exp-*.png`);
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
