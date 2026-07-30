// Backtest parameter helpers: directives parsing, presets, constants.
// Extracted from useBacktestParams.ts to stay under the 250-line limit.

export interface StrategyDirective {
  key: string; label: string; value: string | number | boolean;
  isSet: boolean; display: string;
}

// Presets for commission/slippage (QuantDinger-compatible).
export const PRESETS = {
  live_aligned: { label: 'Live Aligned', commission: 0.0005, slippage: 0.0005 },
  exploration:  { label: 'Exploration',  commission: 0.001,  slippage: 0.0 },
} as const;

export type PresetKey = keyof typeof PRESETS;

/** Convert backend-provided @strategy directives to frontend StrategyDirective[].
 *  Backend zero-trust: the backend extracts raw key-value pairs from code;
 *  frontend applies display formatting (labels, percentage conversion). */
export function backendDirectivesToStrategyDirectives(
  dirs: { key: string; value: string }[],
): StrategyDirective[] {
  const labels: Record<string, string> = {
    stopLossPct: 'Stop Loss %', takeProfitPct: 'Take Profit %',
    entryPct: 'Entry Size %', trailingEnabled: 'Trailing Stop',
    trailingStopPct: 'Trailing Dist %', trailingActivationPct: 'Trailing Act %',
    tradeDirection: 'Direction',
  };
  const seen = new Map<string, string>();
  for (const d of dirs) seen.set(d.key, d.value);

  const out: StrategyDirective[] = [];
  for (const [key, raw] of seen) {
    const label = labels[key] || key;
    const isSet = raw !== undefined && raw !== '';
    const display = formatDirectiveDisplay(key, raw);
    out.push({ key, label, value: raw, isSet, display });
  }
  return out;
}

const PCT_KEYS = new Set(['entryPct', 'stopLossPct', 'takeProfitPct', 'trailingStopPct', 'trailingActivationPct']);

function formatDirectiveDisplay(key: string, raw: string): string {
  if (key === 'trailingEnabled') {
    return raw.toLowerCase() === 'true' || raw === '1' ? 'On' : 'Off';
  }
  const n = parseFloat(raw);
  if (isNaN(n)) return raw;
  if (PCT_KEYS.has(key)) {
    const pct = n <= 1 && n > 0 ? n * 100 : n;
    return `${pct.toFixed(pct < 1 ? 2 : 1)}%`;
  }
  return String(n);
}

export interface SweepDimension {
  key: string; label: string; source: 'code' | 'risk';
  enabled: boolean; values: number[];
}

/** Convert backend-provided sweep dimensions to frontend SweepDimension[].
 *  Backend zero-trust: the backend parses @param annotations; frontend
 *  only converts and renders. Values array is generated from min/max/step. */
export function backendSweepToSweepDimensions(dims: { key: string; type: string; default: number; min: number; max: number; step: number; hasRange: boolean }[]): SweepDimension[] {
  return dims.filter(d => d.hasRange).map(d => {
    const values: number[] = [];
    if (d.step > 0) {
      for (let v = d.min; v <= d.max + d.step * 0.5; v += d.step) {
        values.push(Math.round(v * 1e10) / 1e10);
      }
    }
    if (values.length === 0) values.push(d.default);
    return {
      key: d.key, label: d.key, source: 'code' as const,
      enabled: values.length <= 20,
      values: values.length > 100 ? values.slice(0, 100) : values,
    };
  });
}

export const DEFAULT_SWEEP_DIMS: SweepDimension[] = [
  { key: 'length', label: 'Period / Length', source: 'code', enabled: true, values: [10, 14, 20, 30, 50, 100] },
  { key: 'mult', label: 'Multiplier', source: 'code', enabled: true, values: [1.5, 2.0, 2.5, 3.0] },
  { key: 'stopLoss', label: 'Stop Loss %', source: 'risk', enabled: false, values: [2, 5, 8, 10, 15] },
  { key: 'takeProfit', label: 'Take Profit %', source: 'risk', enabled: false, values: [3, 5, 10, 15, 20] },
  { key: 'maxPositions', label: 'Max Positions', source: 'risk', enabled: false, values: [1, 3, 5, 10] },
];

export function dateFromPreset(months: number): { start: string; end: string } {
  const end = new Date(); const start = new Date();
  start.setMonth(start.getMonth() - months);
  return { start: start.toISOString().slice(0, 10), end: end.toISOString().slice(0, 10) };
}

export const DATE_PRESETS = [
  { key: '1M', label: '1M', months: 1, maxTimeframe: '1m' },
  { key: '3M', label: '3M', months: 3, maxTimeframe: '5m' },
  { key: '6M', label: '6M', months: 6, maxTimeframe: '5m' },
  { key: '1Y', label: '1Y', months: 12, maxTimeframe: '1h' },
];

// Maximum recommended months for a given timeframe.
export const TIMEFRAME_MAX_MONTHS: Record<string, number> = {
  '1m': 1, '5m': 6, '15m': 12, '30m': 12, '1h': 36, '4h': 36, '1d': 36, '1w': 36,
};

export type TuneMethod = 'grid' | 'random' | 'de' | 'tpe' | 'ags' | 'ai';

export const OPTIMIZER_INFO: Record<TuneMethod, { label: string; desc: string }> = {
  grid:    { label: 'Grid Search',     desc: 'Exhaustive Cartesian product. Best for ≤3 params.' },
  random:  { label: 'Random Search',   desc: 'Uniform random sampling. Good for exploration.' },
  de:      { label: 'Differential Evolution', desc: 'rand/1/bin mutation. Converges fast on smooth landscapes.' },
  tpe:     { label: 'TPE (KDE)',       desc: 'Tree-structured Parzen Estimator. KDE models good/bad distributions.' },
  ags:     { label: 'Annealed Gaussian', desc: 'Gaussian jitter with sigma annealing. Lightweight alternative to TPE.' },
  ai:      { label: 'AI Optimizer',    desc: 'LLM multi-round proposal. Learns from previous results over 3 rounds.' },
};
