/**
 * 前端数据适配器
 * 用于在后端 JSON 数据（下划线命名）和前端 TypeScript 类型（驼峰命名）之间进行转换
 */

export interface ProfitUpdate {
  accountId: string;
  balance: number | string;
  credit: number | string;
  profit: number | string;
  equity: number | string;
  margin: number | string;
  freeMargin: number | string;
  marginLevel: number | string;
  orders: OrderProfitItem[];
  platform: string;
  updatedAt: string;
}

export interface OrderProfitItem {
  ticket: number;
  symbol: string;
  profit: number | string;
  volume: number | string;
  currentPrice: number | string;
}

export interface OrderUpdate {
  accountId: string;
  ticket: number;
  symbol: string;
  type: string;
  volume: number | string;
  openPrice: number | string;
  profit: number | string;
  action: string;
  stopLoss?: number | string;
  takeProfit?: number | string;
  closePrice?: number | string;
  openTime: number;
  closeTime?: number;
  swap?: number | string;
  commission?: number | string;
  comment?: string;
}

/**
 * Recursively convert all BigInt values in an object to Number.
 * Protobuf int64 fields are deserialized as BigInt in JS; antd components
 * (Pagination, Table, Statistic) crash when mixing BigInt with Number.
 *
 * Call this on every API response before it reaches React components.
 */
export function deepConvertBigIntToNumber<T>(obj: T): T {
  if (obj === null || obj === undefined) return obj;
  if (typeof obj === 'bigint') return Number(obj) as T;
  if (Array.isArray(obj)) return obj.map(deepConvertBigIntToNumber) as T;
  if (typeof obj === 'object') {
    const result: Record<string, unknown> = {};
    for (const key of Object.keys(obj as Record<string, unknown>)) {
      result[key] = deepConvertBigIntToNumber((obj as Record<string, unknown>)[key]);
    }
    return result as T;
  }
  return obj;
}

export function toCamelCase<T>(obj: unknown): T {
  if (obj === null || obj === undefined) {
    return obj as T;
  }

  // Convert BigInt (protobuf int64) to Number before any rendering/mixing.
  if (typeof obj === 'bigint') {
    return Number(obj) as T;
  }

  if (Array.isArray(obj)) {
    return obj.map(item => toCamelCase(item)) as T;
  }

  if (typeof obj !== 'object') {
    return obj as T;
  }

  const result: unknown = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const camelKey = key.replace(/_([a-z])/g, (_, letter) => letter.toUpperCase());
      result[camelKey] = toCamelCase(obj[key]);
    }
  }
  return result as T;
}

export function toSnakeCase<T>(obj: unknown): T {
  if (obj === null || obj === undefined) {
    return obj as T;
  }

  // Convert BigInt (protobuf int64) to Number.
  if (typeof obj === 'bigint') {
    return Number(obj) as T;
  }

  if (Array.isArray(obj)) {
    return obj.map(item => toSnakeCase(item)) as T;
  }

  if (typeof obj !== 'object') {
    return obj as T;
  }

  const result: unknown = {};
  for (const key in obj) {
    if (Object.prototype.hasOwnProperty.call(obj, key)) {
      const snakeKey = key.replace(/[A-Z]/g, letter => `_${letter.toLowerCase()}`);
      result[snakeKey] = toSnakeCase(obj[key]);
    }
  }
  return result as T;
}
