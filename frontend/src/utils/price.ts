const SYMBOL_DECIMALS: Record<string, number> = {
  XAUUSD: 2,
  XAUUSDm: 2,
  XAGUSD: 3,
  BTCUSD: 2,
  BTCUSDm: 2,
  ETHUSD: 2,
  ETHUSDm: 2,
  EURUSD: 5,
  GBPUSD: 5,
  USDJPY: 3,
  AUDUSD: 5,
  USDCAD: 5,
  USDCHF: 5,
  NZDUSD: 5,
  EURJPY: 3,
  GBPJPY: 3,
  EURGBP: 5,
  EURCHF: 5,
  AUDJPY: 3,
  CADJPY: 3,
  NZDJPY: 3,
  GBPAUD: 5,
  GBPCAD: 5,
  EURNZD: 5,
  GBPNZD: 5,
  AUDCAD: 5,
  AUDNZD: 5,
  NZDCAD: 5,
  NZDCHF: 5,
  AUDCHF: 5,
  CADCHF: 5,
  EURAUD: 5,
  EURCAD: 5,
  EURNOK: 5,
  EURSEK: 5,
  GBPCHF: 5,
  USDDKK: 5,
  USDHKD: 5,
  USDNOK: 5,
  USDSEK: 5,
  USDSGD: 5,
  USDZAR: 5,
};

const DEFAULT_DECIMALS = 5;

export function getSymbolDecimals(symbol: string): number {
  const normalizedSymbol = symbol.toUpperCase().replace('m', 'm');
  if (SYMBOL_DECIMALS[normalizedSymbol]) {
    return SYMBOL_DECIMALS[normalizedSymbol];
  }
  const baseSymbol = normalizedSymbol.replace('m', '');
  if (SYMBOL_DECIMALS[baseSymbol]) {
    return SYMBOL_DECIMALS[baseSymbol];
  }
  return DEFAULT_DECIMALS;
}

export function formatPrice(price: number | undefined | null, symbol?: string): string {
  if (price === undefined || price === null || isNaN(price)) {
    return '-';
  }
  
  const decimals = symbol ? getSymbolDecimals(symbol) : DEFAULT_DECIMALS;
  const fixed = price.toFixed(decimals);
  
  return parseFloat(fixed).toString();
}
