export interface ConnectionLog {
  id: string;
  accountId: string;
  eventType: string;
  status: string;
  message: string;
  errorDetail?: string;
  serverHost: string;
  serverPort: number;
  loginId: number;
  connectionDurationSeconds: number;
  createdAt: string;
}

export interface ExecutionLog {
  id: string;
  accountId?: string;
  scheduleId?: string;
  symbol: string;
  timeframe: string;
  status: string;
  signalType?: string;
  signalPrice?: number;
  signalVolume?: number;
  signalStopLoss?: number;
  signalTakeProfit?: number;
  executedOrderId?: string;
  executedPrice?: number;
  executedVolume?: number;
  profit?: number;
  errorMessage?: string;
  executionTimeMs?: number;
  createdAt: string;
}

export interface OrderHistoryRecord {
  id: string;
  accountId: string;
  scheduleId?: string;
  ticket: number;
  symbol: string;
  orderType: string;
  lots: number;
  openPrice: number;
  closePrice?: number;
  profit: number;
  openTime: string;
  closeTime?: string;
}

export interface OperationLog {
  id: string;
  userId: string;
  module: string;
  action: string;
  details?: string;
  ip?: string;
  userAgent?: string;
  status?: string;
  errorMessage?: string;
  resourceType?: string;
  resourceId?: string;
  durationMs?: number;
  createdAt: string;
}
