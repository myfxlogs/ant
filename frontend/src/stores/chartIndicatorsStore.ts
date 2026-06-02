import { create } from 'zustand';

// Matches backend indicator.Def (see backend/internal/mdgateway/indicator/types.go)
export interface IndicatorParam {
  key: string;
  label: string;
  type: 'int' | 'float';
  default: number;
  min: number;
  max: number;
  step: number;
}

export interface IndicatorDef {
  id: string;
  name: string;
  kind: 'overlay' | 'sub';
  params: IndicatorParam[];
  defaults: Record<string, number>;
}

// Active instance of an indicator (with user-configured params + visibility)
export interface ActiveIndicator {
  instanceId: string; // unique per instance (allows multiple MAs with different periods)
  defId: string;
  params: Record<string, number>; // merged user+default params
  visible: boolean;
}

// ── Registry: matches backend indicator/registry.go ──

const INDICATOR_REGISTRY: IndicatorDef[] = [
  { id: 'SMA', name: 'SMA', kind: 'overlay', params: [{ key: 'length', label: 'Period', type: 'int', default: 20, min: 1, max: 500, step: 1 }], defaults: { length: 20 } },
  { id: 'EMA', name: 'EMA', kind: 'overlay', params: [{ key: 'length', label: 'Period', type: 'int', default: 20, min: 1, max: 500, step: 1 }], defaults: { length: 20 } },
  { id: 'BOLL', name: 'Bollinger Bands', kind: 'overlay', params: [
    { key: 'length', label: 'Period', type: 'int', default: 20, min: 1, max: 500, step: 1 },
    { key: 'mult', label: 'Multiplier', type: 'float', default: 2, min: 0.1, max: 10, step: 0.1 },
  ], defaults: { length: 20, mult: 2 } },
  { id: 'RSI', name: 'RSI', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 14, min: 1, max: 500, step: 1 }], defaults: { length: 14 } },
  { id: 'MACD', name: 'MACD', kind: 'sub', params: [
    { key: 'fast', label: 'Fast', type: 'int', default: 12, min: 1, max: 500, step: 1 },
    { key: 'slow', label: 'Slow', type: 'int', default: 26, min: 1, max: 500, step: 1 },
    { key: 'signal', label: 'Signal', type: 'int', default: 9, min: 1, max: 500, step: 1 },
  ], defaults: { fast: 12, slow: 26, signal: 9 } },
  { id: 'ATR', name: 'ATR', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 14, min: 1, max: 500, step: 1 }], defaults: { length: 14 } },
  { id: 'CCI', name: 'CCI', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 20, min: 1, max: 500, step: 1 }], defaults: { length: 20 } },
  { id: 'WILLR', name: 'Williams %R', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 14, min: 1, max: 500, step: 1 }], defaults: { length: 14 } },
  { id: 'MFI', name: 'MFI', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 14, min: 1, max: 500, step: 1 }], defaults: { length: 14 } },
  { id: 'ADX', name: 'ADX', kind: 'sub', params: [{ key: 'length', label: 'Period', type: 'int', default: 14, min: 1, max: 500, step: 1 }], defaults: { length: 14 } },
  { id: 'OBV', name: 'OBV', kind: 'sub', params: [], defaults: {} },
  { id: 'ADOSC', name: 'AD Oscillator', kind: 'sub', params: [
    { key: 'fast', label: 'Fast', type: 'int', default: 3, min: 1, max: 100, step: 1 },
    { key: 'slow', label: 'Slow', type: 'int', default: 10, min: 1, max: 100, step: 1 },
  ], defaults: { fast: 3, slow: 10 } },
  { id: 'AD', name: 'A/D Line', kind: 'sub', params: [], defaults: {} },
  { id: 'KDJ', name: 'KDJ', kind: 'sub', params: [
    { key: 'period', label: 'Period', type: 'int', default: 9, min: 1, max: 500, step: 1 },
    { key: 'k', label: 'K', type: 'int', default: 3, min: 1, max: 100, step: 1 },
    { key: 'd', label: 'D', type: 'int', default: 3, min: 1, max: 100, step: 1 },
  ], defaults: { period: 9, k: 3, d: 3 } },
  { id: 'VOL', name: 'Volume', kind: 'sub', params: [], defaults: {} },
];

// Maps store defId → klinecharts built-in indicator name + calcParams builder.
// Only indicators supported by klinecharts v9.8 are listed here.
export const KLINECHARTS_MAP: Record<string, { name: string; buildParams: (p: Record<string, number>) => number[] }> = {
  SMA:   { name: 'MA',   buildParams: (p) => [p.length] },
  EMA:   { name: 'EMA',  buildParams: (p) => [p.length] },
  BOLL:  { name: 'BOLL', buildParams: (p) => [p.length, p.mult] },
  RSI:   { name: 'RSI',  buildParams: (p) => [p.length] },
  MACD:  { name: 'MACD', buildParams: (p) => [p.fast, p.slow, p.signal] },
  KDJ:   { name: 'KDJ',  buildParams: (p) => [p.period, p.k, p.d] },
  CCI:   { name: 'CCI',  buildParams: (p) => [p.length] },
  WILLR: { name: 'WR',   buildParams: (p) => [p.length] },
  OBV:   { name: 'OBV',  buildParams: () => [] },
  ADX:   { name: 'DMI',  buildParams: (p) => [p.length] },
  VOL:   { name: 'VOL',  buildParams: () => [] },
};

// ── Store ──

interface ChartIndicatorsState {
  active: ActiveIndicator[];
  addIndicator: (defId: string) => void;
  removeIndicator: (instanceId: string) => void;
  toggleVisibility: (instanceId: string) => void;
  updateParams: (instanceId: string, params: Record<string, number>) => void;
  getDef: (defId: string) => IndicatorDef | undefined;
  registry: IndicatorDef[];
}

let nextId = 1;
function genId() { return `ind_${nextId++}`; }

export const useChartIndicatorsStore = create<ChartIndicatorsState>((set, get) => ({
  active: [],
  registry: INDICATOR_REGISTRY,

  addIndicator: (defId: string) => {
    const def = INDICATOR_REGISTRY.find((d) => d.id === defId);
    if (!def) return;
    set((s) => ({
      active: [...s.active, {
        instanceId: genId(),
        defId,
        params: { ...def.defaults },
        visible: true,
      }],
    }));
  },

  removeIndicator: (instanceId: string) => {
    set((s) => ({ active: s.active.filter((a) => a.instanceId !== instanceId) }));
  },

  toggleVisibility: (instanceId: string) => {
    set((s) => ({
      active: s.active.map((a) =>
        a.instanceId === instanceId ? { ...a, visible: !a.visible } : a,
      ),
    }));
  },

  updateParams: (instanceId: string, params: Record<string, number>) => {
    set((s) => ({
      active: s.active.map((a) =>
        a.instanceId === instanceId ? { ...a, params } : a,
      ),
    }));
  },

  getDef: (defId: string) => INDICATOR_REGISTRY.find((d) => d.id === defId),
}));
