import { tradingClient } from './connect';

export type { Order } from '../gen/api_pb';

export interface OrderSendResult {
  order?: any;
  error: string;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: {
    code?: string;
    reason?: string;
    userMessage?: string;
    retryable?: boolean;
    contextJson?: string;
  };
}

export interface OrderModifyResult {
  order?: any;
  error: string;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: {
    code?: string;
    reason?: string;
    userMessage?: string;
    retryable?: boolean;
    contextJson?: string;
  };
}

export interface OrderCloseResult {
  order?: any;
  error: string;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: {
    code?: string;
    reason?: string;
    userMessage?: string;
    retryable?: boolean;
    contextJson?: string;
  };
}

export interface OrderHistoryResult {
  orders: any[];
  total: number;
  page: number;
  pageSize: number;
}

export interface SyncOrderHistoryResult {
  syncedRecords: number;
}

export const tradingApi = {
  orderSend: async (params: {
    accountId: string;
    symbol: string;
    type: string;
    volume: number;
    price?: number;
    stopLoss?: number;
    takeProfit?: number;
    comment?: string;
    magicNumber?: bigint;
  }): Promise<OrderSendResult> => {
    const response: any = await tradingClient.orderSend({
      accountId: params.accountId,
      symbol: params.symbol,
      type: params.type,
      volume: params.volume,
      price: params.price || 0,
      stopLoss: params.stopLoss || 0,
      takeProfit: params.takeProfit || 0,
      comment: params.comment || '',
      magicNumber: params.magicNumber || BigInt(0),
    });
    return {
      order: response.order,
      error: response.error,
      retcode: response.retcode,
      message: response.message,
      requestId: response.requestId,
      riskError: response.riskError,
    };
  },

  orderModify: async (params: {
    accountId: string;
    ticket: bigint;
    stopLoss?: number;
    takeProfit?: number;
    price?: number;
  }): Promise<OrderModifyResult> => {
    const response: any = await tradingClient.orderModify({
      accountId: params.accountId,
      ticket: params.ticket,
      stopLoss: params.stopLoss || 0,
      takeProfit: params.takeProfit || 0,
      price: params.price || 0,
    });
    return {
      order: response.order,
      error: response.error,
      retcode: response.retcode,
      message: response.message,
      requestId: response.requestId,
      riskError: response.riskError,
    };
  },

  orderClose: async (params: {
    accountId: string;
    ticket: bigint;
    volume?: number;
    price?: number;
  }): Promise<OrderCloseResult> => {
    const response: any = await tradingClient.orderClose({
      accountId: params.accountId,
      ticket: params.ticket,
      volume: params.volume || 0,
      price: params.price || 0,
    });
    return {
      order: response.order,
      error: response.error,
      retcode: response.retcode,
      message: response.message,
      requestId: response.requestId,
      riskError: response.riskError,
    };
  },

  getPositions: async (accountId: string) => {
    const response: any = await tradingClient.getPositions({ accountId });
    return response.positions;
  },

  getPendingOrders: async (accountId: string) => {
    const response: any = await tradingClient.getPendingOrders({ accountId });
    return response.orders;
  },

  getOrderHistory: async (params: {
    accountId: string;
    from?: string;
    to?: string;
    page?: number;
    pageSize?: number;
  }): Promise<OrderHistoryResult> => {
    const response: any = await tradingClient.getOrderHistory({
      accountId: params.accountId,
      from: params.from || '',
      to: params.to || '',
      page: params.page || 1,
      pageSize: params.pageSize || 50,
    });
    return {
      orders: response.orders,
      total: response.total,
      page: response.page,
      pageSize: response.pageSize,
    };
  },

  syncOrderHistory: async (accountId: string): Promise<SyncOrderHistoryResult> => {
    const response: any = await tradingClient.syncOrderHistory({ accountId });
    return {
      syncedRecords: response.syncedRecords,
    };
  },
};
