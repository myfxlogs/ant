export interface Position {
  ticket: number;
  symbol: string;
  type: 'buy' | 'sell' | 'buy_limit' | 'sell_limit' | 'buy_stop' | 'sell_stop';
  volume: number;
  openPrice: number;
  currentPrice: number;
  sl: number;
  tp: number;
  profit: number;
  swap: number;
  commission: number;
  openTime: string;
  comment: string;
  action?: string;
  closePrice?: number;
  closeTime?: string;
  expertId?: number;
  magicNumber?: number;
  rateOpen?: number;
  rateClose?: number;
  rateMargin?: number;
  placedType?: number;
  dealType?: number;
  state?: number;
  contractSize?: number;
  closeVolume?: number;
  closeLots?: number;
  closeComment?: string;
  stopLimitPrice?: number;
  profitRate?: number;
  expirationTime?: number;
}

export interface TradeLog {
  id: string;
  userId: string;
  accountId: string;
  action: string;
  symbol: string;
  orderType: string;
  volume: number;
  price: number;
  ticket: number;
  profit: number;
  message: string;
  createdAt: string;
}
