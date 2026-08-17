import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { streamApi } from '@/client/stream'
import type { StreamCallbacks } from '@/client/stream'

// STREAM-FREEZE-1 R1: subscribeEvents onStale→abort→reconnect integration test.
//
// Simulates a zombie SSE connection: the async iterable hangs (never yields,
// never rejects) until the abort signal fires. The watchdog should detect
// staleness after 45s, abort the connection, and trigger an immediate reconnect.
//
// Adversarial proof: remove `currentAbort?.abort()` from the onStale callback
// in subscribeEvents → the hanging stream never aborts → the for-await loop
// never exits → no reconnect → callCount stays at 1 → RED.

vi.mock('@/client/connect', () => {
  function createHangingStream(signal: AbortSignal): AsyncIterable<unknown> {
    return {
      [Symbol.asyncIterator]() {
        return {
          next(): Promise<IteratorResult<unknown>> {
            return new Promise((_resolve, reject) => {
              const err = new Error('The operation was aborted')
              err.name = 'AbortError'
              if (signal.aborted) { reject(err); return }
              signal.addEventListener('abort', () => reject(err), { once: true })
            })
          },
          return(): Promise<IteratorResult<unknown>> {
            return Promise.resolve({ done: true, value: undefined })
          },
        }
      },
    }
  }
  return {
    streamClient: {
      subscribeEvents: vi.fn((_req: unknown, opts: { signal: AbortSignal }) =>
        createHangingStream(opts.signal),
      ),
    },
  }
})

vi.mock('@/client/sharedStream', () => ({
  subscribeShared: vi.fn(() => () => {}),
  sharedProfitStreams: new Map(),
  sharedOrderStreams: new Map(),
}))

vi.mock('@/adapters/dataAdapter', () => ({
  toCamelCase: vi.fn((x: unknown) => x),
}))

vi.mock('@/utils/streamErrors', () => ({
  isLikelyStreamTransportFailure: vi.fn(() => false),
}))

import { streamClient } from '@/client/connect'

describe('subscribeEvents onStale→abort→reconnect', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.mocked(streamClient.subscribeEvents).mockClear()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('watchdog detects stale, aborts zombie stream, and triggers reconnect', async () => {
    let staleFired = false
    let errorFired = false

    const callbacks: StreamCallbacks = {
      onStale: () => { staleFired = true },
      onError: () => { errorFired = true },
    }

    const unsub = streamApi.subscribeEvents(['acc1'], callbacks)

    // Initial stream started synchronously
    expect(streamClient.subscribeEvents).toHaveBeenCalledTimes(1)

    // Advance past stale threshold (45s) — watchdog fires, aborts, reconnects
    await vi.advanceTimersByTimeAsync(45_001)

    // Watchdog should have fired onStale
    expect(staleFired).toBe(true)
    // Error should NOT fire (stale reconnect is silent)
    expect(errorFired).toBe(false)
    // subscribeEvents should have been called again (immediate reconnect)
    expect(streamClient.subscribeEvents).toHaveBeenCalledTimes(2)

    // Cleanup
    unsub()
    await vi.advanceTimersByTimeAsync(100)
  })
})
