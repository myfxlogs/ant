#!/usr/bin/env tsx
/**
 * Translates English placeholder entries in zh-cn textproto.
 *
 * I18N-MIXED-2 S1: the engine + DICT moved to scripts/i18n-translate.ts and
 * scripts/i18n-dict/zh-cn.ts. This file is a thin shim so existing
 * Makefile/doc references (`npx tsx scripts/i18n-translate-zh-cn.ts`) keep
 * working.
 *
 * Usage: npx tsx scripts/i18n-translate-zh-cn.ts
 */
import './i18n-translate';
