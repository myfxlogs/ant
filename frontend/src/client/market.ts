import { marketClient, tradingClient } from './connect';
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

function toUnixSeconds(ts: Timestamp | undefined): number {
  if (!ts) return 0;
  return Number(ts.seconds ?? BigInt(0));
}

export const marketApi = {
  // getSymbols returns the full real broker symbol list with params.
  // 1. SymbolList → all available symbol names from the connected broker
  // 2. SymbolParams → batch-fetch params for all symbols
  getSymbols: async (accountId: string): Promise<SymbolInfo[]> => {
    try {
      // Step 1: get all symbol names from the broker.
      const listResp = await tradingClient.symbolList({ accountId });
      const symbols = listResp.symbols || [];
      if (symbols.length === 0) return [];

      // Step 2: batch-fetch params for all symbols.
      const paramsResp = await tradingClient.symbolParams({
        accountId,
        canonicals: symbols,
      });
      const params = paramsResp.params || [];
      return params.map((p) => ({
        symbol: p.canonical,
        description: p.symbolRaw !== p.canonical ? p.symbolRaw : undefined,
        digits: p.digits,
        tickValue: Number(p.pointValue ?? '0'),
        contractSize: Number(p.lotSize ?? '0'),
        lotStep: Number(p.lotStep ?? '0'),
        minLot: Number(p.lotMin ?? '0'),
        maxLot: Number(p.lotMax ?? '0'),
      }));
    } catch {
      return [];
    }
  },

  getKlines: async (params: { symbol: string; timeframe: string; count?: number; before?: number }): Promise<KlineData[]> => {
    const req: Record<string, unknown> = {
      canonical: params.symbol,
      period: params.timeframe,
      limit: params.count ?? 300,
    };
    if (params.before) {
      req.to = { seconds: BigInt(params.before), nanos: 0 };
    }
    const response: any = await marketClient.getKlines(req);
    return ((response.bars || []) as OHLCV[]).map((bar) => ({
      time: toUnixSeconds(bar.openTime),
      open: Number(bar.open ?? '0'),
      high: Number(bar.high ?? '0'),
      low: Number(bar.low ?? '0'),
      close: Number(bar.close ?? '0'),
      volume: Number(bar.volume ?? 0),
    }));
  },
};
