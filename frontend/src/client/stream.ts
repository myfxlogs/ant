import { streamClient } from './connect';
import type { OrderUpdate, ProfitUpdate } from '../adapters/dataAdapter';
import type { AccountStatusEvent } from '../gen/ant/v1/stream_event_account_pb';
import type { StreamEvent, BarUpdateEvent } from '../gen/ant/v1/stream_pb';
import { toCamelCase } from '../adapters/dataAdapter';
import { isLikelyStreamTransportFailure } from '../utils/streamErrors';
import type { UserSummaryData } from '../bridge/bridgeUserSummary';
import {
  subscribeShared,
  sharedProfitStreams,
  sharedOrderStreams,
} from './sharedStream';

export type { StreamEvent } from '../gen/ant/v1/stream_pb';
export type { OrderUpdateEvent } from '../gen/ant/v1/stream_event_trade_pb';
export type { ProfitUpdateEvent, UserSummaryEvent } from '../gen/ant/v1/stream_event_account_pb';

/** Stop reconnecting after repeated proxy/HTTP2-style failures (reduces browser network error spam). */
const STREAM_TRANSPORT_FAILURE_CAP = 12;

export interface StreamCallbacks {
  onOrder?: (order: OrderUpdate) => void;
  onProfit?: (profit: ProfitUpdate) => void;
  onStatus?: (status: AccountStatusEvent) => void;
  onPositionSnapshot?: (accountId: string, positions: OrderUpdate[]) => void;
  onBar?: (bar: BarUpdateEvent) => void;
  onError?: (error: Error) => void;
}

type _Listener<T> = {
  onData: (v: T) => void;
  onError?: (error: unknown) => void;
};

function dispatchStreamEvent(e: StreamEvent, callbacks: StreamCallbacks): void {
  switch (e.payload.case) {
    case 'orderUpdate':
      callbacks.onOrder?.(toCamelCase(e.payload.value));
      break;
    case 'profitUpdate':
      callbacks.onProfit?.(toCamelCase<ProfitUpdate>(e.payload.value));
      break;
    case 'accountStatus':
      callbacks.onStatus?.(toCamelCase<AccountStatusEvent>(e.payload.value));
      break;
    case 'positionSnapshot': {
      const snap = toCamelCase<{ accountId: string; positions: Record<string, unknown>[] }>(e.payload.value);
      const orders = (snap.positions || []).map((o) => toCamelCase<OrderUpdate>(o));
      callbacks.onPositionSnapshot?.(snap.accountId, orders);
      break;
    }
    case 'barUpdate':
      callbacks.onBar?.(toCamelCase<BarUpdateEvent>(e.payload.value));
      break;
    default:
      break;
  }
}

function isAbortError(error: unknown): boolean {
  const errorStr = String(error);
  return (error as Error).name === 'AbortError' || errorStr.includes('canceled') || errorStr.includes('aborted');
}

export const streamApi = {
  subscribeEvents: (accountIds: string[], callbacks: StreamCallbacks) => {
    let isAborted = false;
    let currentAbort: AbortController | null = null;
    let transportFailStreak = 0;

    const runStream = async (retryCount = 0) => {
      if (isAborted) return;
      const abortController = new AbortController();
      currentAbort = abortController;

      try {
        const stream = streamClient.subscribeEvents(
          { accountIds },
          { signal: abortController.signal },
        );

        for await (const event of stream) {
          if (isAborted) break;
          transportFailStreak = 0;
          retryCount = 0;
          dispatchStreamEvent(event as StreamEvent, callbacks);
        }

        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        if (isAborted || isAbortError(error)) return;
        if (isLikelyStreamTransportFailure(error)) {
          transportFailStreak++;
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP - 2) {
            console.warn(`[stream] transport failures approaching cap: ${transportFailStreak}/${STREAM_TRANSPORT_FAILURE_CAP}`);
          }
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP) {
            callbacks.onError?.(new Error('stream transport failure cap reached'));
            return;
          }
        } else {
          transportFailStreak = 0;
          callbacks.onError?.(error as Error);
        }
        const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
        setTimeout(() => runStream(retryCount + 1), delay);
      }
    };

    runStream();

    return () => {
      isAborted = true;
      currentAbort?.abort();
    };
  },

  subscribeProfitUpdates: (
    accountId: string,
    callback: (profit: ProfitUpdate) => void,
    onError?: (error: unknown) => void,
  ) => {
    return subscribeShared(
      sharedProfitStreams,
      accountId,
      (signal) => streamClient.subscribeProfitUpdates({ accountId }, { signal }),
      { onData: callback, onError },
    );
  },

  subscribeOrderUpdates: (
    accountId: string,
    callback: (order: OrderUpdate) => void,
    onError?: (error: unknown) => void,
  ) => {
    return subscribeShared(
      sharedOrderStreams,
      accountId,
      (signal) => streamClient.subscribeOrderUpdates({ accountId }, { signal }),
      { onData: callback, onError },
    );
  },

  subscribeUserSummary: (
    callback: (summary: Partial<UserSummaryData>) => void,
    onError?: (error: unknown) => void,
  ) => {
    let isAborted = false;
    let currentAbort: AbortController | null = null;
    let transportFailStreak = 0;

    const runStream = async (retryCount = 0) => {
      if (isAborted) return;
      const abortController = new AbortController();
      currentAbort = abortController;

      try {
        const stream = streamClient.subscribeUserSummary({}, { signal: abortController.signal });

        for await (const summary of stream) {
          if (isAborted) break;
          transportFailStreak = 0;
          retryCount = 0;
          callback(toCamelCase<Partial<UserSummaryData>>(summary));
        }

        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        if (isAborted || isAbortError(error)) return;
        if (isLikelyStreamTransportFailure(error)) {
          transportFailStreak++;
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP - 2) {
            console.warn(`[stream] transport failures approaching cap: ${transportFailStreak}/${STREAM_TRANSPORT_FAILURE_CAP}`);
          }
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP) {
            onError?.(new Error('stream transport failure cap reached'));
            return;
          }
        } else {
          transportFailStreak = 0;
          onError?.(error);
        }
        const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
        setTimeout(() => runStream(retryCount + 1), delay);
      }
    };

    runStream();

    return () => {
      isAborted = true;
      currentAbort?.abort();
    };
  },
};

export function subscribeEvents(accountIds: string[], callbacks: StreamCallbacks) {
  return streamApi.subscribeEvents(accountIds, callbacks);
}

export function subscribeProfitUpdates(
  accountId: string,
  callback: (profit: ProfitUpdate) => void,
  onError?: (error: unknown) => void,
) {
  return streamApi.subscribeProfitUpdates(accountId, callback, onError);
}

export function subscribeOrderUpdates(
  accountId: string,
  callback: (order: OrderUpdate) => void,
  onError?: (error: unknown) => void,
) {
  return streamApi.subscribeOrderUpdates(accountId, callback, onError);
}

export function subscribeUserSummary(
  callback: (summary: Partial<UserSummaryData>) => void,
  onError?: (error: unknown) => void,
) {
  return streamApi.subscribeUserSummary(callback, onError);
}
