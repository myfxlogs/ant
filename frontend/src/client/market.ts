import { marketClient } from './connect';
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
  getSymbols: async (accountId: string): Promise<SymbolInfo[]> => {
    const response: any = await marketClient.getSymbols({ accountId });
    return (response.symbols || []).map((s: any) => ({
      symbol: s.symbol,
      description: s.description,
      currency: s.currency,
      digits: s.digits,
      tickSize: s.tickSize,
      tickValue: s.tickValue,
      contractSize: s.contractSize,
      minLot: s.minLot,
      maxLot: s.maxLot,
      lotStep: s.lotStep,
    }));
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
