import { streamClient } from './connect';
import type { OrderUpdate, ProfitUpdate } from '../adapters/dataAdapter';
import type { AccountStatusEvent } from '../gen/stream_event_account_pb';
import type { StreamEvent } from '../gen/stream_event_core_pb';
import { toCamelCase } from '../adapters/dataAdapter';
import { isLikelyStreamTransportFailure } from '../utils/streamErrors';
import type { UserSummary } from '../stores/tradingStore';

/** Stop reconnecting after repeated proxy/HTTP2-style failures (reduces browser network error spam). */
const STREAM_TRANSPORT_FAILURE_CAP = 12;

export type { StreamEvent } from '../gen/stream_event_core_pb';
export type { OrderUpdateEvent } from '../gen/stream_event_trade_pb';
export type { ProfitUpdateEvent, UserSummaryEvent } from '../gen/stream_event_account_pb';

export interface StreamCallbacks {
  onOrder?: (order: OrderUpdate) => void;
  onProfit?: (profit: ProfitUpdate) => void;
  onStatus?: (status: AccountStatusEvent) => void;
  onError?: (error: Error) => void;
}

type Listener<T> = {
  onData: (v: T) => void;
  onError?: (error: unknown) => void;
};

type SharedStreamState<T> = {
  abortController: AbortController;
  listeners: Map<string, Listener<T>>;
  started: boolean;
};

const sharedProfitStreams = new Map<string, SharedStreamState<ProfitUpdate>>();
const sharedOrderStreams = new Map<string, SharedStreamState<OrderUpdate>>();

function startSharedStream<T>(
  state: SharedStreamState<T>,
  start: (signal: AbortSignal) => AsyncIterable<T>,
  key: string,
  store: Map<string, SharedStreamState<T>>,
) {
  if (state.started) return;
  state.started = true;
  (async () => {
    try {
      const stream = start(state.abortController.signal);
      for await (const item of stream) {
        const val = toCamelCase(item) as T;
        for (const l of state.listeners.values()) {
          try {
            l.onData(val);
          } catch {
            // ignore listener errors
          }
        }
      }
    } catch (error) {
      const errorStr = String(error);
      if ((error as Error).name === 'AbortError' || errorStr.includes('canceled')) {
        return;
      }
      for (const l of state.listeners.values()) {
        try {
          l.onError?.(error);
        } catch {
          // ignore
        }
      }
    } finally {
      const current = store.get(key);
      if (current && current.listeners.size === 0) {
        store.delete(key);
      }
    }
  })();
}

function subscribeShared<T>(
  store: Map<string, SharedStreamState<T>>,
  key: string,
  start: (signal: AbortSignal) => AsyncIterable<T>,
  listener: Listener<T>,
) {
  let state = store.get(key);
  if (!state) {
    state = {
      abortController: new AbortController(),
      listeners: new Map(),
      started: false,
    };
    store.set(key, state);
  }
  const id = Math.random().toString(36).slice(2);
  state.listeners.set(id, listener);
  startSharedStream(state, start, key, store);
  return () => {
    const cur = store.get(key);
    if (!cur) return;
    cur.listeners.delete(id);
    if (cur.listeners.size === 0) {
      cur.abortController.abort();
      store.delete(key);
    }
  };
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
        const stream = streamClient.subscribeEvents({ accountIds }, { signal: abortController.signal });

        for await (const event of stream) {
          if (isAborted) break;
          transportFailStreak = 0;
          retryCount = 0;

          const e = event as StreamEvent;

          switch (e.payload.case) {
            case 'orderUpdate':
              callbacks.onOrder?.(toCamelCase(e.payload.value));
              break;
            case 'profitUpdate': {
              const profit = toCamelCase<ProfitUpdate>(e.payload.value);
              callbacks.onProfit?.(profit);
              break;
            }
            case 'accountStatus':
              callbacks.onStatus?.(toCamelCase<AccountStatusEvent>(e.payload.value));
              break;
            default:
              break;
          }
        }

        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        if (isAborted) return;
        const errorStr = String(error);
        if (
          (error as Error).name === 'AbortError' ||
          errorStr.includes('canceled') ||
          errorStr.includes('aborted')
        ) {
          return;
        }
        if (isLikelyStreamTransportFailure(error)) {
          transportFailStreak++;
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP) {
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
    callback: (summary: Partial<UserSummary>) => void,
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
          callback(toCamelCase<Partial<UserSummary>>(summary));
        }

        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        if (isAborted) return;
        const errorStr = String(error);
        if (
          (error as Error).name === 'AbortError' ||
          errorStr.includes('canceled') ||
          errorStr.includes('aborted')
        ) {
          return;
        }
        if (isLikelyStreamTransportFailure(error)) {
          transportFailStreak++;
          if (transportFailStreak >= STREAM_TRANSPORT_FAILURE_CAP) {
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
  callback: (summary: Partial<UserSummary>) => void,
  onError?: (error: unknown) => void,
) {
  return streamApi.subscribeUserSummary(callback, onError);
}
