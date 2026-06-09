#!/usr/bin/env npx tsx
/**
 * fix-i18n-hardcoded.ts — Batch-fix hardcoded English strings in components.
 *
 * For each hardcoded string found by lint-i18n-hardcoded.ts, this script:
 * 1. Maps it to an i18n key in the appropriate module
 * 2. Adds the key+English value to the en module file (precise insertion)
 * 3. Replaces the hardcoded string in the source .tsx/.ts file with t('key')
 * 4. Fills all other locales from English
 *
 * Usage:
 *   npx tsx scripts/fix-i18n-hardcoded.ts --dry-run   # report only
 *   npx tsx scripts/fix-i18n-hardcoded.ts --write     # apply fixes
 */

import * as fs from 'fs';
import * as path from 'path';
import { execSync } from 'child_process';
import { parseSource, ensureKey } from './lib/i18n-source-editor.js';

const dryRun = process.argv.includes('--dry-run') || !process.argv.includes('--write');

const SRC = path.join(import.meta.dirname || path.dirname(new URL(import.meta.url).pathname), '..', 'frontend', 'src');
const I18N_EN = path.join(SRC, 'i18n', 'resources', 'en');

// Mapping: hardcoded string → { key, module, context }
// context: 'jsx' = use {t('key')}, 'prop' = use {t('key')}, 'msg' = use t('key')
interface Fix {
  match: string;          // exact string to find in source
  key: string;            // i18n key (e.g., 'strategy.gate.compliance')
  module: string;         // which en/*.ts to add the key to
  context: 'jsx' | 'prop' | 'msg';
  file: string;           // relative path to fix
}

// All fixes needed — derived from the agent's audit
const FIXES: Fix[] = [
  // === GatePanel.tsx ===
  { match: "'Compliance'", key: 'strategy.gate.compliance', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'Look-Ahead'", key: 'strategy.gate.lookahead', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'Walk-Forward'", key: 'strategy.gate.walkforward', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'Deflated Sharpe'", key: 'strategy.gate.deflatedSharpe', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'Paper (14d)'", key: 'strategy.gate.paper', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'Correlation'", key: 'strategy.gate.correlation', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "Evaluating...", key: 'strategy.gate.evaluating', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'SKIPPED", key: 'strategy.gate.skipped', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'no data'", key: 'strategy.gate.noData', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'PASS'", key: 'strategy.gate.pass', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'FAIL", key: 'strategy.gate.fail', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: "'unknown'", key: 'strategy.gate.unknown', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/GatePanel.tsx' },
  { match: '"Select backtest run..."', key: 'strategy.gate.selectRun', module: 'strategy', context: 'prop', file: 'pages/strategy/components/workspace/GatePanel.tsx' },

  // === WorkspaceToolbar.tsx ===
  { match: "Watchlist", key: 'strategy.workspace.watchlist', module: 'strategy', context: 'jsx', file: 'pages/strategy/components/workspace/WorkspaceToolbar.tsx' },
  { match: '"Select account"', key: 'strategy.workspace.selectAccount', module: 'strategy', context: 'prop', file: 'pages/strategy/components/workspace/WorkspaceToolbar.tsx' },
  { match: '"No accounts"', key: 'strategy.workspace.noAccounts', module: 'strategy', context: 'prop', file: 'pages/strategy/components/workspace/WorkspaceToolbar.tsx' },
  { match: "'Hide Code'", key: 'strategy.workspace.hideCode', module: 'strategy', context: 'prop', file: 'pages/strategy/components/workspace/WorkspaceToolbar.tsx' },
  { match: "'Show Code'", key: 'strategy.workspace.showCode', module: 'strategy', context: 'prop', file: 'pages/strategy/components/workspace/WorkspaceToolbar.tsx' },

  // === QuickTradePanel.tsx ===
  { match: "'Select a symbol first'", key: 'strategy.quickTrade.selectSymbol', module: 'strategy', context: 'msg', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "'Enter a valid volume'", key: 'strategy.quickTrade.validVolume', module: 'strategy', context: 'msg', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "'Price is required for Limit/Stop orders'", key: 'strategy.quickTrade.priceRequired', module: 'strategy', context: 'msg', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "'Order failed'", key: 'strategy.quickTrade.orderFailed', module: 'strategy', context: 'msg', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "Amount (lots)", key: 'strategy.quickTrade.amountLots', module: 'strategy', context: 'jsx', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "Margin Mode", key: 'strategy.quickTrade.marginMode', module: 'strategy', context: 'jsx', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "'Cross'", key: 'strategy.quickTrade.cross', module: 'strategy', context: 'jsx', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "'Isolated'", key: 'strategy.quickTrade.isolated', module: 'strategy', context: 'jsx', file: 'components/chart/QuickTradePanel.tsx' },
  { match: "MT4 supports Cross margin only", key: 'strategy.quickTrade.mt4CrossOnly', module: 'strategy', context: 'jsx', file: 'components/chart/QuickTradePanel.tsx' },

  // === Chart components ===
  { match: "Live bar stream active", key: 'strategy.chartTools.streamActive', module: 'strategy', context: 'prop', file: 'components/chart/ChartToolbar.tsx' },
  { match: "Stream unavailable", key: 'strategy.chartTools.streamUnavailable', module: 'strategy', context: 'prop', file: 'components/chart/ChartToolbar.tsx' },
  { match: "'Hide'", key: 'strategy.chartTools.hideIndicator', module: 'strategy', context: 'prop', file: 'components/chart/ActiveIndicatorsBar.tsx' },
  { match: "'Show'", key: 'strategy.chartTools.showIndicator', module: 'strategy', context: 'prop', file: 'components/chart/ActiveIndicatorsBar.tsx' },
  { match: "'Settings'", key: 'strategy.chartTools.indicatorSettings', module: 'strategy', context: 'prop', file: 'components/chart/ActiveIndicatorsBar.tsx' },
  { match: '"Remove"', key: 'strategy.chartTools.removeIndicator', module: 'strategy', context: 'prop', file: 'components/chart/ActiveIndicatorsBar.tsx' },
  { match: "'Clear All Drawings'", key: 'strategy.chartTools.clearDrawings', module: 'strategy', context: 'prop', file: 'components/chart/DrawingToolbar.tsx' },

  // === Common/ErrorBoundary ===
  { match: "Page Error", key: 'common.pageError', module: 'base', context: 'prop', file: 'components/common/ErrorBoundary.tsx' },
  { match: "'An unexpected error occurred'", key: 'common.unexpectedError', module: 'base', context: 'prop', file: 'components/common/ErrorBoundary.tsx' },
  { match: "'Retry'", key: 'common.retry', module: 'base', context: 'jsx', file: 'components/common/ErrorBoundary.tsx' },
  { match: '"No data"', key: 'common.noData', module: 'base', context: 'prop', file: 'components/common/StatusResult.tsx' },

  // === Market ===
  { match: "'Watchlist'", key: 'market.watchlist', module: 'base', context: 'jsx', file: 'components/chart/SymbolPicker.tsx' },
  { match: "'All Symbols'", key: 'market.allSymbols', module: 'base', context: 'jsx', file: 'components/chart/SymbolPicker.tsx' },
  { match: "'Select symbol'", key: 'market.selectSymbol', module: 'base', context: 'prop', file: 'components/chart/SymbolPicker.tsx' },
  { match: "'Loading", key: 'common.loading', module: 'base', context: 'jsx', file: 'components/chart/SymbolPicker.tsx' },
  { match: "'No symbols found'", key: 'market.noSymbolsFound', module: 'base', context: 'jsx', file: 'components/chart/SymbolPicker.tsx' },
  { match: '"Bid"', key: 'market.bid', module: 'base', context: 'prop', file: 'pages/market/Market.tsx' },
  { match: '"Ask"', key: 'market.ask', module: 'base', context: 'prop', file: 'pages/market/Market.tsx' },
  { match: '"Spread"', key: 'market.spread', module: 'base', context: 'prop', file: 'pages/market/Market.tsx' },
  { match: "'Common'", key: 'market.common', module: 'base', context: 'jsx', file: 'pages/market/Market.tsx' },
];

// Group by module for i18n key insertion
const keysByModule: Record<string, Record<string, string>> = {};
for (const f of FIXES) {
  if (!keysByModule[f.module]) keysByModule[f.module] = {};
  keysByModule[f.module][f.key] = f.match.replace(/^['"]/, '').replace(/['"]$/, '');
}

if (dryRun) {
  console.log('DRY RUN — would fix ' + FIXES.length + ' hardcoded strings:');
  for (const f of FIXES) {
    console.log(`  ${f.file}: "${f.match}" → {t('${f.key}')}`);
  }
  console.log('\nNew i18n keys to add:');
  for (const [mod, keys] of Object.entries(keysByModule)) {
    console.log(`  ${mod}.ts: ${Object.keys(keys).length} keys`);
  }
} else {
  // Step 1: Add keys to en modules
  console.log('Step 1: Adding i18n keys to en modules...');
  for (const [mod, keys] of Object.entries(keysByModule)) {
    const modPath = path.join(I18N_EN, mod + '.ts');
    if (!fs.existsSync(modPath)) { console.log(`  SKIP (no file): ${mod}.ts`); continue; }

    let source = fs.readFileSync(modPath, 'utf8');
    let locMap = parseSource(source);

    for (const [key, value] of Object.entries(keys)) {
      const result = ensureKey(source, key, value, locMap);
      source = result.source;
      locMap = result.locMap;
    }

    fs.writeFileSync(modPath, source, 'utf8');
    console.log(`  Added ${Object.keys(keys).length} keys to en/${mod}.ts`);
  }

  // Step 2: Fix components
  console.log('\nStep 2: Replacing hardcoded strings in components...');
  let fixedCount = 0;
  const filesFixed = new Set<string>();

  for (const f of FIXES) {
    const filePath = path.join(SRC, f.file);
    if (!fs.existsSync(filePath)) { console.log(`  SKIP (no file): ${f.file}`); continue; }

    let content = fs.readFileSync(filePath, 'utf8');
    let replacement: string;

    switch (f.context) {
      case 'jsx':
        // Replace the literal string in JSX context
        if (content.includes(f.match)) {
          // For simple string matches, replace with {t('key')}
          // Need to handle different patterns:
          // >String< → >{t('key')}<
          // {string} → {t('key')}
          // For GATE_LABELS object values, replace 'String' with t('key') in the object
          if (content.includes(`'${f.key}'`)) break; // already fixed

          // For GATE_LABELS pattern: compliance: 'Compliance' → compliance: t('...')
          content = content.replace(new RegExp(`: ${f.match}([,\\n\\}])`, 'g'), `: t('${f.key}')$1`);
          // Also handle >String< pattern
          content = content.replace(new RegExp(`>${f.match.replace(/['"]/g, '')}<`, 'g'), `>{t('${f.key}')}<`);
          // Handle prop= pattern
          content = content.replace(new RegExp(`=${f.match}([\\s>\\/])`, 'g'), `={t('${f.key}')}$1`);
        }
        break;
      case 'prop':
        // Replace prop="String" with prop={t('key')}
        content = content.replace(f.match, `{t('${f.key}')}`);
        break;
      case 'msg':
        // Replace message.warning('String') with message.warning(t('key'))
        content = content.replace(new RegExp(`(message\\.\\w+\\()${f.match}`, 'g'), `$1t('${f.key}')`);
        break;
    }

    fs.writeFileSync(filePath, content, 'utf8');
    fixedCount++;
    filesFixed.add(f.file);
  }
  console.log(`  Fixed ${fixedCount} strings across ${filesFixed.size} files`);

  // Step 3: Fill other locales
  console.log('\nStep 3: Filling other locales...');
  execSync('npx tsx ' + path.join(import.meta.dirname || '.', 'fill-i18n-from-en.ts') + ' --write', {
    cwd: path.join(import.meta.dirname || '.', '..'),
    stdio: 'inherit',
  });

  console.log('\nDone. Run: npx tsx scripts/lint-i18n-hardcoded.ts --strict  to verify.');
}
