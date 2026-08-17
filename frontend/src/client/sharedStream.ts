import type { OrderUpdate, ProfitUpdate } from '../adapters/dataAdapter';
import { toCamelCase } from '../adapters/dataAdapter';
import { createStreamWatchdog, type StreamWatchdog } from './streamWatchdog';

type Listener<T> = {
  onData: (v: T) => void;
  onError?: (error: unknown) => void;
};

export type SharedStreamState<T> = {
  abortController: AbortController;
  listeners: Map<string, Listener<T>>;
  started: boolean;
  staleThresholdMs?: number;
  watchdog?: StreamWatchdog;
  staleDetected?: boolean;
};

function backoffDelay(retryCount: number): number {
  return Math.min(1000 * Math.pow(2, retryCount), 30000);
}

function isAbortError(error: unknown): boolean {
  const errorStr = String(error);
  return (error as Error).name === 'AbortError' || errorStr.includes('canceled');
}

function notifyListenersError<T>(state: SharedStreamState<T>, error: unknown): void {
  for (const l of state.listeners.values()) {
    try {
      l.onError?.(error);
    } catch {
      // ignore
    }
  }
}

function dispatchToListeners<T>(state: SharedStreamState<T>, val: T): void {
  // Skip heartbeat events (empty accountId from backend keepalive).
  if (!(val as { accountId?: string }).accountId) return;
  for (const l of state.listeners.values()) {
    try {
      l.onData(val);
    } catch {
      // ignore listener errors
    }
  }
}

function cleanupIfEmpty<T>(key: string, store: Map<string, SharedStreamState<T>>): void {
  const current = store.get(key);
  if (current && current.listeners.size === 0) {
    current.watchdog?.stop();
    store.delete(key);
  }
}

function handleStreamCatch<T>(
  state: SharedStreamState<T>,
  error: unknown,
  runStream: (n: number) => void,
  retryCount: number,
): void {
  state.watchdog?.stop();
  if (state.staleDetected && state.listeners.size > 0) {
    runStream(0);
    return;
  }
  if (isAbortError(error)) return;
  notifyListenersError(state, error);
  scheduleReconnect(state, runStream, retryCount);
}

function scheduleReconnect<T>(
  state: SharedStreamState<T>,
  runStream: (retryCount: number) => void,
  retryCount: number,
): boolean {
  if (state.listeners.size === 0) return false;
  setTimeout(() => runStream(retryCount + 1), backoffDelay(retryCount));
  return true;
}

const sharedProfitStreams = new Map<string, SharedStreamState<ProfitUpdate>>();
const sharedOrderStreams = new Map<string, SharedStreamState<OrderUpdate>>();

export { sharedProfitStreams, sharedOrderStreams };

export function startSharedStream<T>(
  state: SharedStreamState<T>,
  start: (signal: AbortSignal) => AsyncIterable<T>,
  key: string,
  store: Map<string, SharedStreamState<T>>,
) {
  if (state.started) return;
  state.started = true;
  state.staleDetected = false;

  // Create watchdog if stale threshold is configured.
  if (state.staleThresholdMs && !state.watchdog) {
    state.watchdog = createStreamWatchdog({
      staleThresholdMs: state.staleThresholdMs,
      onStale: () => {
        console.warn('[sharedStream] stale detected, forcing reconnect');
        state.staleDetected = true;
        state.abortController.abort();
      },
    });
  }

  const runStream = async (retryCount = 0) => {
    state.abortController = new AbortController();
    state.staleDetected = false;
    state.watchdog?.start();

    try {
      const stream = start(state.abortController.signal);
      for await (const item of stream) {
        state.watchdog?.touch();
        dispatchToListeners(state, toCamelCase(item) as T);
      }

      state.watchdog?.stop();
      scheduleReconnect(state, runStream, retryCount);
    } catch (error) {
      handleStreamCatch(state, error, runStream, retryCount);
    } finally {
      state.started = false;
      cleanupIfEmpty(key, store);
    }
  };

  runStream();
}

export function subscribeShared<T>(
  store: Map<string, SharedStreamState<T>>,
  key: string,
  start: (signal: AbortSignal) => AsyncIterable<T>,
  listener: Listener<T>,
  streamOpts?: { staleThresholdMs?: number },
) {
  let state = store.get(key);
  if (!state) {
    state = {
      abortController: new AbortController(),
      listeners: new Map(),
      started: false,
      staleThresholdMs: streamOpts?.staleThresholdMs,
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
      cur.watchdog?.stop();
      cur.abortController.abort();
      store.delete(key);
    }
  };
}
