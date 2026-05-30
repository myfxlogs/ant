import { tradingClient } from './connect';
import { Side, OrderType, PlaceOrderRequestSchema, CloseOrderRequestSchema } from '@/gen/ant/v1/mthub_service_pb';
import { create } from '@bufbuild/protobuf';
import type { PlaceOrderResponse } from '@/gen/ant/v1/mthub_service_pb';
import type { OrderRecord } from '@/gen/ant/v1/mthub_service_pb';
import type { OpenedOrdersResponse } from '@/gen/ant/v1/mthub_service_pb';
import type { OrderHistoryResponse } from '@/gen/ant/v1/mthub_service_pb';

/**
 * Intermediate position shape returned by fromProtoOrders.
 * The Position fields in @/types/trading are a superset; these objects
 * have the subset that comes directly from the OrderRecord proto.
 */
export interface ProtoPosition {
  ticket: number;
  symbol: string;
  type: string;
  volume: number;
  openPrice: number;
  closePrice: number;
  profit: number;
  commission: number;
  swap: number;
  magic: number;
  openTime: number;
  closeTime: number;
}

export interface RiskError {
  code?: string;
  reason?: string;
  userMessage?: string;
  retryable?: boolean;
  contextJson?: string;
}

/**
 * Backend enriches the proto PlaceOrderResponse / CloseOrderResponse with
 * additional fields (error, retcode, message, requestId, riskError, order).
 * These fields are NOT defined in the proto schema. We type them here
 * so consumers can access them without casts.
 */
interface EnrichedOrderResponse {
  error?: unknown;
  retcode?: number;
  message?: string;
  requestId?: string;
  riskError?: RiskError;
  order?: OrderRecord | undefined;
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
function fromProtoOrders(orders: OrderRecord[]): ProtoPosition[] {
  return (orders || []).map((o): ProtoPosition => {
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
      ticket: Number(o.ticket),
      symbol: o.symbolRaw || o.canonical || '',
      type: o.side === Side.BUY ? 'buy' : o.side === Side.SELL ? 'sell' : '',
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

/**
 * Parse a combined side+orderType string (e.g. "buy_limit", "sell")
 * into the proto Side and OrderType enum values.
 */
function parseSideOrderType(type: string): { side: Side; orderType: OrderType } {
  const upper = type.toUpperCase();
  let side: Side;
  if (upper.startsWith('BUY')) {
    side = Side.BUY;
  } else if (upper.startsWith('SELL')) {
    side = Side.SELL;
  } else {
    side = Side.UNSPECIFIED;
  }

  let orderType: OrderType;
  if (upper.includes('STOP_LIMIT')) {
    orderType = OrderType.STOP_LIMIT;
  } else if (upper.includes('STOP')) {
    orderType = OrderType.STOP;
  } else if (upper.includes('LIMIT')) {
    orderType = OrderType.LIMIT;
  } else {
    orderType = OrderType.MARKET;
  }

  return { side, orderType };
}

/** Convert a numeric price/volume to a decimal string for the proto wire format. */
function toDecimalString(n: number | undefined, fallback = '0'): string {
  if (n === undefined || n === null) return fallback;
  return n.toString();
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
    const { side, orderType } = parseSideOrderType(params.type);
    const response = await tradingClient.placeOrder(
      create(PlaceOrderRequestSchema, {
        accountId: params.accountId,
        canonical: params.symbol,
        side,
        orderType,
        volume: toDecimalString(params.volume, '0'),
        price: toDecimalString(params.price, '0'),
        stopLoss: toDecimalString(params.stopLoss, '0'),
        takeProfit: toDecimalString(params.takeProfit, '0'),
        comment: params.comment || '',
        magic: Number(params.magicNumber || 0),
      }),
    );
    // Backend enriches the response with fields beyond the proto schema.
    const enriched = response as PlaceOrderResponse & EnrichedOrderResponse;
    return {
      order: enriched.order,
      error: String(enriched.error ?? ''),
      retcode: enriched.retcode,
      message: enriched.message,
      requestId: enriched.requestId,
      riskError: enriched.riskError,
    };
  },

  orderClose: async (params: {
    accountId: string;
    ticket: bigint;
    volume?: number;
    price?: number;
  }): Promise<OrderCloseResult> => {
    const response = await tradingClient.closeOrder(
      create(CloseOrderRequestSchema, {
        accountId: params.accountId,
        ticket: params.ticket,
        lots: String(params.volume || 0),
      }),
    );
    // Backend enriches the response with fields beyond the proto schema.
    const enriched = response as PlaceOrderResponse & EnrichedOrderResponse;
    return {
      order: enriched.order,
      error: String(enriched.error ?? ''),
      retcode: enriched.retcode,
      message: enriched.message,
      requestId: enriched.requestId,
      riskError: enriched.riskError,
    };
  },

  getPositions: async (accountId: string) => {
    const response: OpenedOrdersResponse = await tradingClient.openedOrders({ accountId });
    return fromProtoOrders(response.orders);
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
    // Proto OrderHistoryRequest only has accountId, from, to.
    // Backend handles pagination via an envelope that adds page/pageSize/total
    // to the response. Pass the known proto fields as-is.
    const response = await tradingClient.orderHistory({
      accountId: params.accountId,
      from: undefined,
      to: undefined,
    });
    // Backend enriches response with pagination fields beyond proto schema.
    const enriched = response as OrderHistoryResponse & {
      page?: number;
      pageSize?: number;
      total?: bigint;
    };
    return {
      orders: fromProtoOrders(enriched.orders),
      total: Number(enriched.total ?? 0),
      page: Number(enriched.page ?? 1),
      pageSize: Number(enriched.pageSize ?? 50),
    };
  },

};
