// serverIndicators.ts — Shared state for server-computed indicator values + custom klinecharts indicators.
//
// Architecture:
//   1. useServerIndicators hook subscribes to streamClient.subscribeIndicators()
//   2. Server sends IndicatorUpdateEvent → values stored in serverIndicatorStore Map
//   3. Custom klinecharts indicators (ANT_SMA, ANT_EMA, ...) read from this store in their calc()
//   4. chart.applyNewData() triggers indicator recalculation with fresh server values
//
// This follows push-first architecture: backend computes all indicators using decimal.Decimal,
// frontend only renders the pre-computed values.

import { registerIndicator, type KLineData } from 'klinecharts';
import type { IndicatorCreate } from 'klinecharts';

// ── Shared state: indicator values populated by useServerIndicators hook ──

export interface ServerIndicatorData {
  values: number[];          // single-line indicators
  series: Record<string, number[]>; // multi-line (MACD, BOLL, KDJ)
  pane: string;              // "overlay" | "sub"
}

// Module-level store: indicatorId → ServerIndicatorData
const serverStore = new Map<string, ServerIndicatorData>();

export function setServerIndicatorData(id: string, data: ServerIndicatorData) {
  serverStore.set(id, data);
}

export function getServerIndicatorData(id: string): ServerIndicatorData | undefined {
  return serverStore.get(id);
}

export function clearServerIndicatorData(id?: string) {
  if (id) {
    serverStore.delete(id);
  } else {
    serverStore.clear();
  }
}

// ── Helper: read server-computed values from the shared store ──

function readServerValues(antId: string, list: KLineData[], field?: string): number[] {
  const data = serverStore.get(antId);
  if (!data) return [];

  if (field && data.series?.[field]) {
    return data.series[field].slice(0, list.length);
  }
  return data.values.slice(0, list.length);
}

// ── Indicator definitions (factory) ──

interface AntIndicatorDef {
  antId: string;
  name: string;
  shortName: string;
  isOverlay: boolean;
  precision: number;
  shouldOhlc: boolean;
  figures: Array<{ key: string; title: string; type: string }>;
  colors: string[];
  lineSize: number;
  dashed?: number[];
}

const DEFS: AntIndicatorDef[] = [
  { antId: 'SMA', name: 'ANT_SMA', shortName: 'SMA', isOverlay: true, precision: 5, shouldOhlc: false, figures: [{ key: 'sma', title: 'SMA', type: 'line' }], colors: ['#2962FF'], lineSize: 1.5 },
  { antId: 'EMA', name: 'ANT_EMA', shortName: 'EMA', isOverlay: true, precision: 5, shouldOhlc: false, figures: [{ key: 'ema', title: 'EMA', type: 'line' }], colors: ['#FF6D00'], lineSize: 1.5 },
  { antId: 'BOLL', name: 'ANT_BOLL', shortName: 'BOLL', isOverlay: true, precision: 5, shouldOhlc: false, figures: [
    { key: 'mid', title: 'MID', type: 'line' },
    { key: 'upper', title: 'UPPER', type: 'line' },
    { key: 'lower', title: 'LOWER', type: 'line' },
  ], colors: ['#FFD600', '#2979FF', '#2979FF'], lineSize: 1.5, dashed: [1.5, 1, 1.5, 1] },
  { antId: 'RSI', name: 'ANT_RSI', shortName: 'RSI', isOverlay: false, precision: 2, shouldOhlc: false, figures: [{ key: 'rsi', title: 'RSI', type: 'line' }], colors: ['#7C4DFF'], lineSize: 1.5 },
  { antId: 'MACD', name: 'ANT_MACD', shortName: 'MACD', isOverlay: false, precision: 5, shouldOhlc: false, figures: [
    { key: 'macd', title: 'MACD', type: 'line' },
    { key: 'signal', title: 'SIGNAL', type: 'line' },
    { key: 'histogram', title: 'HIST', type: 'bar' },
  ], colors: ['#F44336', '#2196F3', '#4CAF50'], lineSize: 1.5 },
  { antId: 'ATR', name: 'ANT_ATR', shortName: 'ATR', isOverlay: false, precision: 5, shouldOhlc: false, figures: [{ key: 'atr', title: 'ATR', type: 'line' }], colors: ['#00BCD4'], lineSize: 1.5 },
  { antId: 'CCI', name: 'ANT_CCI', shortName: 'CCI', isOverlay: false, precision: 2, shouldOhlc: false, figures: [{ key: 'cci', title: 'CCI', type: 'line' }], colors: ['#FF9800'], lineSize: 1.5 },
  { antId: 'WILLR', name: 'ANT_WILLR', shortName: 'W%R', isOverlay: false, precision: 2, shouldOhlc: false, figures: [{ key: 'willr', title: 'W%R', type: 'line' }], colors: ['#E91E63'], lineSize: 1.5 },
  { antId: 'MFI', name: 'ANT_MFI', shortName: 'MFI', isOverlay: false, precision: 2, shouldOhlc: false, figures: [{ key: 'mfi', title: 'MFI', type: 'line' }], colors: ['#9C27B0'], lineSize: 1.5 },
  { antId: 'ADX', name: 'ANT_ADX', shortName: 'ADX', isOverlay: false, precision: 2, shouldOhlc: false, figures: [
    { key: 'adx', title: 'ADX', type: 'line' },
    { key: 'pdi', title: '+DI', type: 'line' },
    { key: 'mdi', title: '-DI', type: 'line' },
  ], colors: ['#FFD600', '#4CAF50', '#F44336'], lineSize: 1.5 },
  { antId: 'OBV', name: 'ANT_OBV', shortName: 'OBV', isOverlay: false, precision: 0, shouldOhlc: false, figures: [{ key: 'obv', title: 'OBV', type: 'line' }], colors: ['#00E676'], lineSize: 1.5 },
  { antId: 'ADOSC', name: 'ANT_ADOSC', shortName: 'ADOSC', isOverlay: false, precision: 0, shouldOhlc: false, figures: [{ key: 'adosc', title: 'ADOSC', type: 'line' }], colors: ['#FFAB40'], lineSize: 1.5 },
  { antId: 'AD', name: 'ANT_AD', shortName: 'A/D', isOverlay: false, precision: 0, shouldOhlc: false, figures: [{ key: 'ad', title: 'A/D', type: 'line' }], colors: ['#18FFFF'], lineSize: 1.5 },
  { antId: 'KDJ', name: 'ANT_KDJ', shortName: 'KDJ', isOverlay: false, precision: 2, shouldOhlc: false, figures: [
    { key: 'k', title: 'K', type: 'line' },
    { key: 'd', title: 'D', type: 'line' },
    { key: 'j', title: 'J', type: 'line' },
  ], colors: ['#FF6D00', '#2196F3', '#E91E63'], lineSize: 1.5 },
  { antId: 'VOL', name: 'ANT_VOL', shortName: 'VOL', isOverlay: false, precision: 0, shouldOhlc: true, figures: [
    { key: 'up', title: 'UP', type: 'bar' },
    { key: 'down', title: 'DOWN', type: 'bar' },
  ], colors: ['#EF5350', '#26A69A'], lineSize: 1 },
];

// ── Register all custom indicators ──

DEFS.forEach((def) => {
  const indicator: IndicatorCreate = {
    name: def.name,
    shortName: def.shortName,
    precision: def.precision,
    shouldOhlc: def.shouldOhlc,
    figures: def.figures,
    styles: {
      lines: def.figures.map((_, i) => ({
        color: def.colors[i] || '#888',
        size: def.lineSize,
        style: 'solid' as unknown,
        smooth: false,
        ...(def.dashed ? { dashedValue: def.dashed } : {}),
      })),
    },

    calc: (list: KLineData[]) => {
      const data = serverStore.get(def.antId);
      if (!data) return list.map(() => {
        const empty: Record<string, number> = {};
        def.figures.forEach((f) => { empty[f.key] = 0; });
        return empty;
      });

      const result: Record<string, number>[] = [];
      const n = Math.min(list.length, data.values.length || 0);

      if (def.figures.length === 1) {
        // Single-line indicator
        const key = def.figures[0].key;
        for (let i = 0; i < n; i++) {
          result.push({ [key]: data.values[i] || 0 });
        }
        for (let i = n; i < list.length; i++) {
          result.push({ [key]: 0 });
        }
      } else {
        // Multi-line indicator: read from data.series
        for (let i = 0; i < n; i++) {
          const obj: Record<string, number> = {};
          def.figures.forEach((f) => {
            const series = data.series?.[f.key];
            obj[f.key] = series?.[i] ?? 0;
          });
          result.push(obj);
        }
        for (let i = n; i < list.length; i++) {
          const obj: Record<string, number> = {};
          def.figures.forEach((f) => { obj[f.key] = 0; });
          result.push(obj);
        }
      }

      return result;
    },
  };

  try { registerIndicator(indicator); } catch { /* already registered */ }
});

// Export the ANT_ prefix for KLINECHARTS_MAP
export const ANT_PREFIX = 'ANT_';
