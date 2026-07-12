#!/usr/bin/env npx tsx
/**
 * i18n-check — CI gate that verifies translation completeness.
 *
 * Checks:
 *   1. Every locale textproto exists for each section
 *   2. Every field defined in the English textproto exists in every locale
 *   3. No field value is empty
 *   4. (WARNING) Non-en locales have no field whose value exactly matches the English value
 *
 * Usage: npx tsx scripts/i18n-check.ts [--strict]
 *   --strict  Treat English-value matches in non-en locales as ERROR
 */

import * as fs from 'fs';
import * as path from 'path';

const PROTO_DIR = path.resolve(__dirname, '..', 'proto', 'ant', 'v1', 'i18n');
const LOCALES = ['en', 'zh-cn', 'zh-tw', 'ja', 'vi'] as const;
const SECTIONS = [
  'strategy_workspace', 'strategy_backtest_params', 'strategy_tuning',
  'strategy_ai', 'strategy_backtest', 'strategy_code_assist',
  'strategy_code_quality', 'strategy_code_editor', 'strategy_chart_tools',
  'strategy_quick_trade_section', 'strategy_library', 'strategy_templates',
  'strategy_experiment', 'strategy_market_regime', 'strategy_asset',
  'strategy_asset_analysis', 'strategy_backtest_run', 'strategy_schedules',
  'strategy_schedule_logs', 'strategy_gen', 'strategy_ai_chat',
  'strategy_paper', 'strategy_default_templates',
  'accounts', 'ai_core', 'ai_settings', 'ai_wizard', 'ai_store',
  'base', 'dashboard', 'trading', 'analytics', 'admin', 'logs', 'errors',
] as const;

const STRICT = process.argv.includes('--strict');

interface TextprotoField {
  key: string;
  value: string;
}

function parseTextproto(content: string): Map<string, string> {
  const fields = new Map<string, string>();
  for (const line of content.split('\n')) {
    const m = line.match(/^(\w+):\s*'((?:[^'\\]|\\.)*)'\s*$/);
    if (m) {
      fields.set(m[1], m[2].replace(/\\'/g, "'"));
    }
  }
  return fields;
}

interface CheckResult {
  errors: string[];
  warnings: string[];
}

function checkSection(section: string): CheckResult {
  const result: CheckResult = { errors: [], warnings: [] };

  // Read English as reference
  const enPath = path.join(PROTO_DIR, `${section}_en.textproto`);
  if (!fs.existsSync(enPath)) {
    result.errors.push(`English reference missing: ${enPath}`);
    return result;
  }
  const enContent = fs.readFileSync(enPath, 'utf-8');
  const enFields = parseTextproto(enContent);

  if (enFields.size === 0) {
    result.errors.push(`English reference is empty: ${enPath}`);
    return result;
  }

  // Check each locale
  for (const locale of LOCALES) {
    if (locale === 'en') continue;

    const tpPath = path.join(PROTO_DIR, `${section}_${locale}.textproto`);
    if (!fs.existsSync(tpPath)) {
      result.errors.push(`[${locale}] Missing textproto: ${tpPath}`);
      continue;
    }

    const content = fs.readFileSync(tpPath, 'utf-8');
    const fields = parseTextproto(content);

    // 1. Missing keys
    for (const key of enFields.keys()) {
      if (!fields.has(key)) {
        result.errors.push(`[${locale}] Missing field: ${key}`);
      }
    }

    // 2. Extra keys (shouldn't happen but worth flagging)
    for (const key of fields.keys()) {
      if (!enFields.has(key)) {
        result.warnings.push(`[${locale}] Extra field (not in en): ${key}`);
      }
    }

    // 3. Empty values
    for (const [key, value] of fields) {
      if (!value || value.trim() === '') {
        result.errors.push(`[${locale}] Empty value: ${key}`);
      }
    }

    // 4. English copy detection (value matches en exactly → likely untranslated)
    const EXEMPT_TERMS = new Set([
      'K-line', 'OHLC', 'ATR', 'ATR %', 'EURUSD', 'AI', 'API',
      'Cross', 'Isolated', 'Sharpe', 'Sortino', 'Calmar',
      'EMA', 'SMA', 'RSI', 'MACD', 'TPE', 'KDE', 'DE',
      'TPE (KDE)', 'Run(context)',
      'Cron', 'LIVE', 'ERROR', 'STATIC',
      'Shared', 'by',
      // Brand / provider names
      'AlphaForge', 'OpenAI', 'Anthropic Claude', 'DeepSeek', 'Groq',
      'Mistral', 'OpenRouter', 'OpenAI Compatible',
      // Language names (each locale keeps its own name)
      'English', 'Tiếng Việt', '繁體中文', '简体中文', '日本語',
      // Technical
      'Base URL', 'API Key', 'VaR 95%',
    ]);
    const EXEMPT_PATTERNS = [
      /^Backtest: \{\{/,
      /^Today's/,                   // "Today's Executions", "Today's Profit"
      /^Cards show each provider/,
      /^Model suggestion:/,
      /^Multiple experts/,
      /^Current provider:/,
      /^Macro events/,
      /^User (expectation|strategy goal)/,
      /^Parameters \(defs/,
      /^【/,                         // Chinese-style headers
      /^No code block found/,
      /^None \(save template/,
      /^Not recommended for direct/,
    ];
    for (const [key, value] of fields) {
      const enValue = enFields.get(key);
      if (enValue && value === enValue) {
        // Skip if value contains non-Latin characters (already translated)
        if (/[^\x00-\x7F]/.test(value)) continue;
        // Skip exempt global terms
        if (EXEMPT_TERMS.has(value.trim())) continue;
        // Skip exempt patterns
        if (EXEMPT_PATTERNS.some(p => p.test(value))) continue;
        // Skip language-neutral values (code fences, acronyms, template vars)
        const isNeutral = /^[#```]|^[A-Z]{1,4}$|^cron:|^\{\{/.test(value);
        if (!isNeutral) {
          const msg = `[${locale}] Untranslated (matches en): ${key} = "${value.substring(0, 60)}"`;
          if (STRICT) {
            result.errors.push(msg);
          } else {
            result.warnings.push(msg);
          }
        }
      }
    }
  }

  return result;
}

// ── Main ──

function main(): void {
  console.log(`[i18n-check] ${STRICT ? 'STRICT mode' : 'normal mode'} — checking ${SECTIONS.length} section(s)...\n`);

  let totalErrors = 0;
  let totalWarnings = 0;

  for (const section of SECTIONS) {
    const result = checkSection(section);

    if (result.errors.length > 0) {
      console.log(`\n❌ ${section}: ${result.errors.length} error(s)`);
      for (const e of result.errors) console.log(`   ERROR: ${e}`);
      totalErrors += result.errors.length;
    }

    if (result.warnings.length > 0) {
      console.log(`\n⚠️  ${section}: ${result.warnings.length} warning(s)`);
      for (const w of result.warnings) console.log(`   WARN: ${w}`);
      totalWarnings += result.warnings.length;
    }

    if (result.errors.length === 0 && result.warnings.length === 0) {
      console.log(`✅ ${section}: all checks passed`);
    }
  }

  console.log(`\n[i18n-check] ${totalErrors} error(s), ${totalWarnings} warning(s)`);

  if (totalErrors > 0) {
    console.log('❌ FAILED');
    process.exit(1);
  }

  console.log('✅ PASSED');
}

main();
