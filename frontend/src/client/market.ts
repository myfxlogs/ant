import { marketClient } from './connect';
import type { Kline } from '../gen/market_pb';

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

  getKlines: async (params: { accountId: string; symbol: string; timeframe: string; count?: number }): Promise<Kline[]> => {
    const response: any = await marketClient.getKlines({
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      from: '',
      to: '',
      count: params.count ?? 200,
    } as any);
    return (response.klines || []) as Kline[];
  },
};
