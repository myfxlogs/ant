export interface Quote {
  symbol: string;
  bid: number;
  ask: number;
  spread: number;
  time: string;
  change?: number;
  changePct?: number;
}

export interface SymbolInfo {
  name: string;
  description: string;
  digits: number;
  point: number;
  spread: number;
  minLot: number;
  maxLot: number;
  lotStep: number;
  contractSize: number;
  marginRequired: number;
  currency?: string;
}

export interface KlineRequest {
  accountId: string;
  symbol: string;
  timeframe: string;
  from?: string;
  to?: string;
  count?: number;
}

export interface KlineBar {
  openTime: string;
  closeTime: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
}

export interface KlineData {
  symbol: string;
  timeframe: string;
  bars: KlineBar[];
  count: number;
}

export interface WSMessage {
  type: string;
  payload: unknown;
}

export interface QuoteMessage {
  accountId: string;
  symbol: string;
  bid: number;
  ask: number;
  time: string;
}
