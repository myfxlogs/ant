// Shared schedule parameter helpers to keep launch and edit forms consistent

export type RiskFields = {
  defaultVolume?: number;
  maxPositions?: number;
  stopLossPriceOffset?: number;
  takeProfitPriceOffset?: number;
  maxDrawdownPct?: number; // 0~1
};

export type CommonFields = RiskFields & {
  scheduleName?: string;
  // commonly used strategy params
  lot?: number;
  grid_count?: number;
  lower_price?: number;
  upper_price?: number;
  interval_hours?: number;
};

// Build parameters map<string,string> for create/update schedule
export function buildParametersFromForm(v: CommonFields): Record<string, string> {
  const out: Record<string, string> = {};
  setIfPositive(out, '__schedule.name', v.scheduleName, (val) => String(val).trim());
  setIfPositive(out, '__risk.default_volume', v.defaultVolume, String);
  setIfPositive(out, '__risk.max_positions', v.maxPositions, (val) => String(Math.floor(val)));
  setIfPositive(out, '__risk.stop_loss_price_offset', v.stopLossPriceOffset, String);
  setIfPositive(out, '__risk.take_profit_price_offset', v.takeProfitPriceOffset, String);
  if (v.maxDrawdownPct && v.maxDrawdownPct > 0 && v.maxDrawdownPct <= 1) out['__risk.max_drawdown_pct'] = String(v.maxDrawdownPct);
  setIfPositive(out, 'lot', v.lot, String);
  setIfPositive(out, 'grid_count', v.grid_count, (val) => String(Math.floor(val)));
  if (typeof v.lower_price === 'number') out['lower_price'] = String(v.lower_price);
  if (typeof v.upper_price === 'number') out['upper_price'] = String(v.upper_price);
  setIfPositive(out, 'interval_hours', v.interval_hours, (val) => String(Math.floor(val)));
  return out;
}

function setIfPositive(out: Record<string, string>, key: string, val: unknown, fmt: (v: unknown) => string): void {
  if (val && (typeof val === 'number' ? val > 0 : String(val).trim())) out[key] = fmt(val);
}

// Parse parameters map back to form-friendly fields
export function parseParametersToForm(p: Record<string, unknown> | undefined): CommonFields {
  const out: CommonFields = {};
  const getNum = (k: string) => {
    const raw = (p || {})[k];
    if (raw === undefined || raw === null || raw === '') return undefined;
    const n = typeof raw === 'number' ? raw : Number(raw);
    return Number.isFinite(n) ? n : undefined;
  };
  if (!p) return out;
  out.scheduleName = String(p['__schedule.name'] || '').trim() || undefined;
  out.defaultVolume = getNum('__risk.default_volume');
  const mp = getNum('__risk.max_positions');
  out.maxPositions = typeof mp === 'number' ? Math.floor(mp) : undefined;
  out.stopLossPriceOffset = getNum('__risk.stop_loss_price_offset');
  out.takeProfitPriceOffset = getNum('__risk.take_profit_price_offset');
  const md = getNum('__risk.max_drawdown_pct');
  out.maxDrawdownPct = typeof md === 'number' ? md : undefined;
  // common params
  out.lot = getNum('lot');
  const gc = getNum('grid_count');
  out.grid_count = typeof gc === 'number' ? Math.floor(gc) : undefined;
  out.lower_price = getNum('lower_price');
  out.upper_price = getNum('upper_price');
  const ih = getNum('interval_hours');
  out.interval_hours = typeof ih === 'number' ? Math.floor(ih) : undefined;
  return out;
}
