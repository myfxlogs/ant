import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import type { SupportedLanguage } from '@/i18n';

export const LANGUAGE_LABELS: Record<SupportedLanguage, string> = {
  'zh-cn': '简体中文',
  'zh-tw': '繁體中文',
  en: 'English',
  ja: '日本語',
  vi: 'Tiếng Việt',
};

export function BrandLogo({ name }: { name: string }) {
  return (
    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
      <span style={{ width: 40, height: 40, borderRadius: 12, background: PRIMARY_GRADIENT, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
          <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
        </svg>
      </span>
      <span style={{ fontWeight: 700, fontSize: 'clamp(16px, 4vw, 20px)', fontFamily: 'Poppins, sans-serif' }}>{name}</span>
    </div>
  );
}

export function toNum(v: unknown): number {
  if (typeof v === 'bigint') return Number(v);
  if (typeof v === 'number') return v;
  if (typeof v === 'string') { const n = Number(v); return Number.isFinite(n) ? n : 0; }
  return 0;
}

export function fmt(n: number, d = 2) { return Number.isFinite(n) ? n.toFixed(d) : '-'; }

export function avgHoldingText(ms: number) {
  if (ms <= 0) return '-';
  const h = ms / 3600000;
  if (h < 1) return `${Math.round(ms / 60000)}m`;
  if (h < 24) return `${h.toFixed(1)}h`;
  return `${(h / 24).toFixed(1)}d`;
}

export interface ShareData {
  userName: string;
  totalReturn: number;
  winRate: number;
  maxDrawdown: number;
  totalTrades: number;
  totalVolume: number;
  profitFactor: number;
  avgHoldingMs: number;
  sharpeRatio: number;
  equityCurve: number[];
  equityTimesMs?: number[];
  trades: Array<{ symbol: string; side: string; volume: number; profit: number; closeTimeMs: number }>;
  positions?: Array<{ symbol: string; type: string; volume: number; openPrice: number; currentPrice: number; profit: number; openTimeMs: number }> | null;
  showPositions?: boolean;
  expired?: boolean;
}

export function computeMaxDrawdownPct(equity: number[], fallback: number): number {
  if (equity.length > 1) {
    let peak = -Infinity, maxDD = 0;
    for (const raw of equity) {
      const e = toNum(raw);
      if (e > peak) peak = e;
      if (peak > 0) { const dd = (peak - e) / peak * 100; if (dd > maxDD) maxDD = dd; }
    }
    return maxDD;
  }
  return fallback;
}

export function aggregateBySymbol(trades: ShareData['trades']) {
  const symbolMap = new Map<string, { symbol: string; count: number; net: number }>();
  for (const tr of trades) {
    const k = tr.symbol || '-';
    const cur = symbolMap.get(k) || { symbol: k, count: 0, net: 0 };
    cur.count += 1;
    cur.net += toNum(tr.profit);
    symbolMap.set(k, cur);
  }
  return Array.from(symbolMap.values()).sort((a, b) => b.net - a.net);
}
