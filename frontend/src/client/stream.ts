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

/** Classify a stream error and update transport failure streak. Returns the action to take. */
type StreamErrorAction = 'stale-reconnect' | 'abort' | 'stop' | 'retry';

function classifyStreamError(
  error: unknown,
  opts: {
    isAborted: boolean;
    staleDetected: boolean;
    transportFailStreak: number;
    onError?: (error: Error) => void;
  },
): { action: StreamErrorAction; streak: number } {
  if (opts.staleDetected && !opts.isAborted) return { action: 'stale-reconnect', streak: opts.transportFailStreak };
  if (opts.isAborted || isAbortError(error)) return { action: 'abort', streak: opts.transportFailStreak };

  let streak = opts.transportFailStreak;
  if (isLikelyStreamTransportFailure(error)) {
    streak++;
    if (streak >= STREAM_TRANSPORT_FAILURE_CAP - 2) {
      console.warn(`[stream] transport failures approaching cap: ${streak}/${STREAM_TRANSPORT_FAILURE_CAP}`);
    }
    if (streak >= STREAM_TRANSPORT_FAILURE_CAP) {
      opts.onError?.(new Error('stream transport failure cap reached'));
      return { action: 'stop', streak };
    }
  } else {
    streak = 0;
    opts.onError?.(error as Error);
  }
  return { action: 'retry', streak };
}

function backoffDelay(retryCount: number): number {
  return Math.min(1000 * Math.pow(2, retryCount), 30000);
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
          setTimeout(() => runStream(retryCount + 1), backoffDelay(retryCount));
        }
      } catch (error) {
        watchdog.stop();
        const result = classifyStreamError(error, {
          isAborted, staleDetected, transportFailStreak, onError: callbacks.onError,
        });
        transportFailStreak = result.streak;
        if (result.action === 'stale-reconnect') { runStream(0); return; }
        if (result.action === 'abort' || result.action === 'stop') return;
        setTimeout(() => runStream(retryCount + 1), backoffDelay(retryCount));
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
          setTimeout(() => runStream(retryCount + 1), backoffDelay(retryCount));
        }
      } catch (error) {
        watchdog.stop();
        const result = classifyStreamError(error, {
          isAborted, staleDetected, transportFailStreak, onError: onError as ((e: Error) => void) | undefined,
        });
        transportFailStreak = result.streak;
        if (result.action === 'stale-reconnect') { runStream(0); return; }
        if (result.action === 'abort' || result.action === 'stop') return;
        setTimeout(() => runStream(retryCount + 1), backoffDelay(retryCount));
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
