import { tradingClient } from './connect';
import type { OHLCV } from '../gen/ant/v1/mthub_service_pb';
import type { Timestamp } from '@bufbuild/protobuf/wkt';

export interface SymbolInfo {
  symbol: string;
  description?: string;
  currency?: string;
  digits?: number;
  tickSize?: number;
  tickValue?: number;
  contractSize?: number;
  minLot?: number;
  maxLot?: number;
  lotStep?: number;
}

export interface KlineData {
  time: number; // unix seconds
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

/** Detect MT session errors from ConnectRPC errors. */
export function isMTSessionError(e: unknown): boolean {
  const msg = String((e as Error)?.message || '');
  return msg.includes('session not found') || msg.includes('mthub');
}

function toUnixSeconds(ts: Timestamp | undefined): number {
  if (!ts) return 0;
  return Number(ts.seconds ?? BigInt(0));
}

export const marketApi = {
  // getSymbols returns all available symbol names from the connected broker.
  // Uses SymbolList RPC (fast, returns just names).
  // Throws on MT session errors so the UI can show a helpful message.
  getSymbols: async (accountId: string): Promise<SymbolInfo[]> => {
    const resp = await tradingClient.symbolList({ accountId });
    return (resp.symbols || []).map((s) => ({ symbol: s }));
  },

  // resolveSymbol passthrough (broker symbols used directly for K-line queries).
  resolveSymbol: (name: string): string => name,

  // clearSymbolCache no-op.
  clearSymbolCache: () => {},

  // getSymbolParams fetches detailed trading params for a batch of symbols.
  getSymbolParams: async (accountId: string, canonicals: string[]): Promise<SymbolInfo[]> => {
    try {
      const resp = await tradingClient.symbolParams({ accountId, canonicals });
      return (resp.params || []).map((p) => ({
        symbol: p.canonical,
        description: p.symbolRaw !== p.canonical ? p.symbolRaw : undefined,
        digits: p.digits,
        tickValue: Number(p.pointValue ?? '0'),
        contractSize: Number(p.lotSize ?? '0'),
        lotStep: Number(p.lotStep ?? '0'),
        minLot: Number(p.lotMin ?? '0'),
        maxLot: Number(p.lotMax ?? '0'),
      }));
    } catch (e) { console.warn('market API call failed', e);
      return [];
    }
  },

  // subscribeBars tells the backend to subscribe the gateway to this symbol's ticks,
  // enabling real-time bar aggregation and SSE push.
  // Errors are thrown so callers can surface subscription failures to the user.
  subscribeBars: async (params: { accountId: string; symbol: string }): Promise<void> => {
    await tradingClient.subscribeBars({ accountId: params.accountId, symbol: params.symbol });
  },

  // getKlines fetches OHLCV kline bars — broker-first via PriceHistory with ClickHouse fallback.
  // Requires accountId to locate the broker session; returns [] without it.
  getKlines: async (params: { symbol: string; timeframe: string; count?: number; before?: number; accountId?: string }): Promise<KlineData[]> => {
    if (!params.accountId) return [];

    const req: Record<string, unknown> = {
      accountId: params.accountId,
      canonical: params.symbol,
      period: params.timeframe,
      limit: params.count ?? 300,
    };
    if (params.before) {
      req.to = { seconds: BigInt(params.before), nanos: 0 };
    }

    try {
      const resp: unknown = await tradingClient.priceHistory(req);
      const bars = (resp.bars || []) as OHLCV[];
      return bars.map((bar) => ({
        time: toUnixSeconds(bar.openTime),
        open: Number(bar.open ?? '0'),
        high: Number(bar.high ?? '0'),
        low: Number(bar.low ?? '0'),
        close: Number(bar.close ?? '0'),
        volume: Number(bar.volume ?? 0),
      }));
    } catch (e) { console.warn('market API call failed', e);
      return [];
    }
  },
};
