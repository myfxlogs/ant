import { tradingClient } from './connect';

export interface RiskError {
  code?: string;
  reason?: string;
  userMessage?: string;
  retryable?: boolean;
  contextJson?: string;
}

export interface OrderSendResult {
  order?: unknown;
  error: string;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: RiskError;
}

export interface OrderCloseResult {
  order?: unknown;
  error: string;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: RiskError;
}

export interface OrderHistoryResult {
  orders: unknown[];
  total: number;
  page: number;
  pageSize: number;
}

// Proto OrderRecord uses string for decimal fields and Timestamp for time fields;
// frontend expects number. Convert orders from proto wire format to JS-friendly format.
function fromProtoOrders(orders: unknown[]): unknown[] {
  return (orders || []).map((o: Record<string, unknown>) => {
    // @bufbuild/protobuf Timestamp is {seconds: bigint, nanos: number}.
    // Convert to unix seconds (number) for downstream consumers.
    const toUnixSeconds = (ts: unknown): number => {
      if (ts == null) return 0;
      if (typeof ts === 'number') return ts;
      if (typeof ts === 'string') return Number(ts) || 0;
      const t = ts as Record<string, unknown>;
      if (t.seconds != null) {
        const secs = typeof t.seconds === 'bigint' ? Number(t.seconds) : Number(t.seconds);
        const nanos = Number(t.nanos || 0);
        return secs + nanos / 1_000_000_000;
      }
      return 0;
    };
    return {
      ...o,
      ticket: Number(o.ticket),
      volume: Number(o.volume),
      openPrice: Number(o.openPrice),
      closePrice: Number(o.closePrice || 0),
      profit: Number(o.profit),
      commission: Number(o.commission || 0),
      swap: Number(o.swap || 0),
      magic: Number(o.magic || 0),
      openTime: toUnixSeconds(o.openTime),
      closeTime: toUnixSeconds(o.closeTime),
    };
  });
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
    // Params use pre-transform field names (symbol, type, magicNumber) that
    // differ from the proto schema (canonical, side+orderType, magic).
    const response = await tradingClient.placeOrder(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      {
        accountId: params.accountId,
        symbol: params.symbol,
        type: params.type,
        volume: params.volume,
        price: params.price || 0,
        stopLoss: params.stopLoss || 0,
        takeProfit: params.takeProfit || 0,
        comment: params.comment || '',
        magicNumber: params.magicNumber || BigInt(0),
      } as any,
    ) as unknown as Record<string, unknown>;
    return {
      order: response.order,
      error: String(response.error ?? ''),
      retcode: response.retcode as number | undefined,
      message: response.message as string | undefined,
      requestId: response.requestId as string | undefined,
      riskError: response.riskError as RiskError | undefined,
    };
  },

  orderClose: async (params: {
    accountId: string;
    ticket: bigint;
    volume?: number;
    price?: number;
  }): Promise<OrderCloseResult> => {
    // Params use pre-transform field names; proto CloseOrderRequest uses lots (string).
    const response = await tradingClient.closeOrder(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      {
        accountId: params.accountId,
        ticket: params.ticket,
        volume: params.volume || 0,
        price: params.price || 0,
      } as any,
    ) as unknown as Record<string, unknown>;
    return {
      order: response.order,
      error: String(response.error ?? ''),
      retcode: response.retcode as number | undefined,
      message: response.message as string | undefined,
      requestId: response.requestId as string | undefined,
      riskError: response.riskError as RiskError | undefined,
    };
  },

  getPositions: async (accountId: string) => {
    const response = await tradingClient.openedOrders({ accountId });
    return fromProtoOrders(response.orders as unknown[]);
  },

	syncOrderHistory: async (accountId: string) => {
		const response = await tradingClient.syncOrderHistory({ accountId });
		return { syncedRecords: Number(response.syncedRecords ?? 0) };
	},

  getOrderHistory: async (params: {
    accountId: string;
    from?: string;
    to?: string;
    page?: number;
    pageSize?: number;
  }): Promise<OrderHistoryResult> => {
    // Proto OrderHistoryRequest uses Timestamp for from/to and does not
    // include page/pageSize. The backend handles string-to-Timestamp
    // conversion and pagination.
    const response = await tradingClient.orderHistory(
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      {
        accountId: params.accountId,
        from: params.from || '',
        to: params.to || '',
        page: params.page || 1,
        pageSize: params.pageSize || 50,
      } as any,
    ) as unknown as Record<string, unknown>;
    return {
      orders: fromProtoOrders(response.orders as unknown[]),
      total: Number(response.total ?? 0),
      page: Number(response.page ?? 1),
      pageSize: Number(response.pageSize ?? 50),
    };
  },

};
