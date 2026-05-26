import { timestampFromDate } from '@bufbuild/protobuf/wkt';
import { marketRegimeClient } from './connect';
import type { MarketRegime } from '../gen/ant/v1/market_regime_pb';

export type { MarketRegime };

type DetectMarketRegimeParams = {
  accountId: string;
  symbol: string;
  timeframe: string;
  count?: number;
  from?: Date;
  to?: Date;
};

export const marketRegimeApi = {
  detect: (params: DetectMarketRegimeParams) =>
    marketRegimeClient.detectMarketRegime({
      accountId: params.accountId,
      symbol: params.symbol,
      timeframe: params.timeframe,
      count: params.count ?? 120,
      from: params.from ? timestampFromDate(params.from) : undefined,
      to: params.to ? timestampFromDate(params.to) : undefined,
    }),

  get: (regimeId: string) => marketRegimeClient.getMarketRegime({ regimeId }),
};
