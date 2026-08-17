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
import { createStreamWatchdog } from './streamWatchdog';

/** Stale threshold for subscribeEvents: 3 × backend 15s ping = 45s. */
const EVENTS_STALE_THRESHOLD_MS = 45_000;
/** Stale threshold for subscribeUserSummary: 3 × backend 30s keepalive = 90s. */
const SUMMARY_STALE_THRESHOLD_MS = 90_000;
/** Stale threshold for shared streams (order/profit): 3 × backend 15s heartbeat = 45s. */
const SHARED_STALE_THRESHOLD_MS = 45_000;

export type { StreamEvent } from '../gen/ant/v1/stream_pb';

/** Stop reconnecting after repeated proxy/HTTP2-style failures (reduces browser network error spam). */
const STREAM_TRANSPORT_FAILURE_CAP = 12;

export interface StreamCallbacks {
  onOrder?: (order: OrderUpdate) => void;
  onProfit?: (profit: ProfitUpdate) => void;
  onStatus?: (status: AccountStatusEvent) => void;
  onPositionSnapshot?: (accountId: string, positions: OrderUpdate[]) => void;
  onBar?: (bar: BarUpdateEvent) => void;
  onError?: (error: Error) => void;
  onStale?: () => void;
}

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
    let staleDetected = false;

    const watchdog = createStreamWatchdog({
      staleThresholdMs: EVENTS_STALE_THRESHOLD_MS,
      onStale: () => {
        staleDetected = true;
        callbacks.onStale?.();
        currentAbort?.abort();
      },
    });

    const runStream = async (retryCount = 0) => {
      if (isAborted) return;
      staleDetected = false;
      const abortController = new AbortController();
      currentAbort = abortController;
      watchdog.start();

      try {
        const stream = streamClient.subscribeEvents(
          { accountIds },
          { signal: abortController.signal },
        );

        for await (const event of stream) {
          if (isAborted) break;
          watchdog.touch();
          transportFailStreak = 0;
          retryCount = 0;
          dispatchStreamEvent(event as StreamEvent, callbacks);
        }

        watchdog.stop();
        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        watchdog.stop();
        if (staleDetected && !isAborted) {
          runStream(0);
          return;
        }
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
      watchdog.stop();
      currentAbort?.abort();
    };
  },

  subscribeProfitUpdates: (
    accountId: string,
    callback: (profit: ProfitUpdate) => void,
    onError?: (error: unknown) => void,
  ) => {
    return subscribeShared<ProfitUpdate>(
      sharedProfitStreams,
      accountId,
      (signal) => streamClient.subscribeProfitUpdates({ accountId }, { signal }) as unknown as AsyncIterable<ProfitUpdate>,
      { onData: callback, onError },
      { staleThresholdMs: SHARED_STALE_THRESHOLD_MS },
    );
  },

  subscribeOrderUpdates: (
    accountId: string,
    callback: (order: OrderUpdate) => void,
    onError?: (error: unknown) => void,
  ) => {
    return subscribeShared<OrderUpdate>(
      sharedOrderStreams,
      accountId,
      (signal) => streamClient.subscribeOrderUpdates({ accountId }, { signal }) as unknown as AsyncIterable<OrderUpdate>,
      { onData: callback, onError },
      { staleThresholdMs: SHARED_STALE_THRESHOLD_MS },
    );
  },

  subscribeUserSummary: (
    callback: (summary: Partial<UserSummaryData>) => void,
    onError?: (error: unknown) => void,
    onStale?: () => void,
  ) => {
    let isAborted = false;
    let currentAbort: AbortController | null = null;
    let transportFailStreak = 0;
    let staleDetected = false;

    const watchdog = createStreamWatchdog({
      staleThresholdMs: SUMMARY_STALE_THRESHOLD_MS,
      onStale: () => {
        staleDetected = true;
        onStale?.();
        currentAbort?.abort();
      },
    });

    const runStream = async (retryCount = 0) => {
      if (isAborted) return;
      staleDetected = false;
      const abortController = new AbortController();
      currentAbort = abortController;
      watchdog.start();

      try {
        const stream = streamClient.subscribeUserSummary({}, { signal: abortController.signal });

        for await (const summary of stream) {
          if (isAborted) break;
          watchdog.touch();
          transportFailStreak = 0;
          retryCount = 0;
          callback(toCamelCase<Partial<UserSummaryData>>(summary));
        }

        watchdog.stop();
        if (!isAborted) {
          const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
          setTimeout(() => runStream(retryCount + 1), delay);
        }
      } catch (error) {
        watchdog.stop();
        if (staleDetected && !isAborted) {
          runStream(0);
          return;
        }
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
      watchdog.stop();
      currentAbort?.abort();
    };
  },
};

export function subscribeEvents(accountIds: string[], callbacks: StreamCallbacks) {
  return streamApi.subscribeEvents(accountIds, callbacks);
}

export function subscribeUserSummary(
  callback: (summary: Partial<UserSummaryData>) => void,
  onError?: (error: unknown) => void,
  onStale?: () => void,
) {
  return streamApi.subscribeUserSummary(callback, onError, onStale);
}
