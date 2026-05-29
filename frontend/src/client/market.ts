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

const COMMON_SYMBOLS = [
  'EURUSD', 'GBPUSD', 'USDJPY', 'AUDUSD', 'NZDUSD', 'USDCAD', 'USDCHF',
  'EURGBP', 'EURJPY', 'GBPJPY', 'AUDJPY', 'NZDJPY', 'CADJPY', 'CHFJPY',
  'XAUUSD', 'XAGUSD', 'BTCUSD', 'ETHUSD', 'US30', 'US100', 'GER40',
  'EURCHF', 'EURAUD', 'EURNZD', 'GBPCHF', 'GBPAUD', 'GBPNZD',
  'GBPCAD', 'AUDCAD', 'AUDCHF', 'AUDNZD', 'NZDCAD', 'NZDCHF',
  'CADCHF', 'XAUJPY',
];

export const marketApi = {
  getSymbols: async (accountId: string): Promise<SymbolInfo[]> => {
    // Use SymbolParams RPC on MtHubService to get real broker symbols.
    // Pass common canonicals; the broker returns params only for known ones.
    try {
      const resp = await tradingClient.symbolParams({
        accountId,
        canonicals: COMMON_SYMBOLS,
      });
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
    } catch {
      // Fallback: return common symbols with no broker params.
      return COMMON_SYMBOLS.map((s) => ({ symbol: s }));
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
