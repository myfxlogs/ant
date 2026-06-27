import type { OrderUpdate, ProfitUpdate } from '../adapters/dataAdapter';
import { toCamelCase } from '../adapters/dataAdapter';

type Listener<T> = {
  onData: (v: T) => void;
  onError?: (error: unknown) => void;
};

export type SharedStreamState<T> = {
  abortController: AbortController;
  listeners: Map<string, Listener<T>>;
  started: boolean;
};

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

  const runStream = async (retryCount = 0) => {
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

      // Stream ended cleanly — reconnect with backoff if listeners remain.
      if (state.listeners.size > 0) {
        const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
        setTimeout(() => runStream(retryCount + 1), delay);
        return;
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
      // Reconnect with backoff if listeners remain.
      if (state.listeners.size > 0) {
        const delay = Math.min(1000 * Math.pow(2, retryCount), 30000);
        setTimeout(() => runStream(retryCount + 1), delay);
        return;
      }
    } finally {
      state.started = false;
      const current = store.get(key);
      if (current && current.listeners.size === 0) {
        store.delete(key);
      }
    }
  };

  runStream();
}

export function subscribeShared<T>(
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
