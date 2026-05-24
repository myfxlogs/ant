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
  if (v.scheduleName && String(v.scheduleName).trim()) out['__schedule.name'] = String(v.scheduleName).trim();
  if (v.defaultVolume && v.defaultVolume > 0) out['__risk.default_volume'] = String(v.defaultVolume);
  if (v.maxPositions && v.maxPositions >= 1) out['__risk.max_positions'] = String(Math.floor(v.maxPositions));
  if (v.stopLossPriceOffset && v.stopLossPriceOffset > 0) out['__risk.stop_loss_price_offset'] = String(v.stopLossPriceOffset);
  if (v.takeProfitPriceOffset && v.takeProfitPriceOffset > 0) out['__risk.take_profit_price_offset'] = String(v.takeProfitPriceOffset);
  if (v.maxDrawdownPct && v.maxDrawdownPct > 0 && v.maxDrawdownPct <= 1) out['__risk.max_drawdown_pct'] = String(v.maxDrawdownPct);
  // common strategy params (flat)
  if (v.lot && v.lot > 0) out['lot'] = String(v.lot);
  if (v.grid_count && v.grid_count > 0) out['grid_count'] = String(Math.floor(v.grid_count));
  if (typeof v.lower_price === 'number') out['lower_price'] = String(v.lower_price);
  if (typeof v.upper_price === 'number') out['upper_price'] = String(v.upper_price);
  if (v.interval_hours && v.interval_hours > 0) out['interval_hours'] = String(Math.floor(v.interval_hours));
  return out;
}

// Parse parameters map back to form-friendly fields
export function parseParametersToForm(p: Record<string, any> | undefined): CommonFields {
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
