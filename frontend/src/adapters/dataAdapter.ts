/**
 * 前端数据适配器
 * 用于在后端 JSON 数据（下划线命名）和前端 TypeScript 类型（驼峰命名）之间进行转换
 */

export interface AccountInfo {
  id: string;
  userId: string;
  mtType: string;
  brokerCompany: string;
  brokerServer: string;
  brokerHost: string;
  login: string;
  alias: string;
  isDisabled: boolean;
  balance: number;
  credit: number;
  profit: number;
  equity: number;
  margin: number;
  freeMargin: number;
  marginLevel: number;
  leverage: number;
  currency: string;
  type: string;
  isInvestor: boolean;
  accountStatus?: string;
  streamStatus?: string;
  lastError?: string;
  lastConnectedAt?: string;
  lastCheckedAt?: string;
  createdAt: string;
  updatedAt: string;
}

export interface Position {
  ticket: number;
  symbol: string;
  orderType: number;
  volume: number;
  openPrice: number;
  currentPrice: number;
  profit: number;
  action?: string;
  stopLoss?: number;
  takeProfit?: number;
  closePrice?: number;
  openTime: number;
  closeTime?: number;
  swap?: number;
  commission?: number;
  comment?: string;
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

export interface Quote {
  symbol: string;
  bid: number;
  ask: number;
  high?: number;
  low?: number;
  last?: number;
  volume?: number;
  timestamp: string;
}

export interface ProfitUpdate {
  accountId: string;
  balance: number;
  credit: number;
  profit: number;
  equity: number;
  margin: number;
  freeMargin: number;
  marginLevel: number;
  orders: OrderProfitItem[];
  platform: string;
  updatedAt: string;
}

export interface OrderProfitItem {
  ticket: number;
  symbol: string;
  profit: number;
  volume: number;
  currentPrice: number;
}

export interface OrderUpdate {
  accountId: string;
  ticket: number;
  symbol: string;
  type: string;
  volume: number;
  openPrice: number;
  profit: number;
  action: string;
  stopLoss?: number;
  takeProfit?: number;
  closePrice?: number;
  openTime: number;
  closeTime?: number;
  swap?: number;
  commission?: number;
  comment?: string;
}

export function toCamelCase<T>(obj: any): T {
  if (obj === null || obj === undefined) {
    return obj as T;
  }

  if (Array.isArray(obj)) {
    return obj.map(item => toCamelCase(item)) as T;
  }

  if (typeof obj !== 'object') {
    return obj as T;
  }

  const result: any = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const camelKey = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
      result[camelKey] = toCamelCase(obj[key]);
    }
  }
  return result as T;
}

export function toSnakeCase<T>(obj: any): T {
  if (obj === null || obj === undefined) {
    return obj as T;
  }

  if (Array.isArray(obj)) {
    return obj.map(item => toSnakeCase(item)) as T;
  }

  if (typeof obj !== 'object') {
    return obj as T;
  }

  const result: any = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const snakeKey = key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
      result[snakeKey] = toSnakeCase(obj[key]);
    }
  }
  return result as T;
}
