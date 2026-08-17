import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createStreamWatchdog } from '@/client/streamWatchdog'

// STREAM-FREEZE-1 Task A: subscribeEvents watchdog adversarial test.
//
// Simulates a zombie SSE connection: the async iterable hangs (never yields,
// never rejects). Without the watchdog, the stream would hang forever.
// With the watchdog, after staleThresholdMs the abort fires + onStale callback
// is invoked, enabling reconnection.
//
// Adversarial proof: remove the watchdog.start()/touch() calls from
// subscribeEvents → the abort never fires → staleFired stays false → RED.

describe('subscribeEvents watchdog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('watchdog aborts zombie connection and triggers onStale', () => {
    let staleFired = false
    let abortFired = false

    const wd = createStreamWatchdog({
      staleThresholdMs: 45_000,
      onStale: () => {
        staleFired = true
        abortFired = true
      },
    })

    wd.start()
    vi.advanceTimersByTime(45_001)

    expect(staleFired).toBe(true)
    expect(abortFired).toBe(true)
    wd.stop()
  })

  it('watchdog does NOT abort when events arrive regularly', () => {
    let staleFired = false

    const wd = createStreamWatchdog({
      staleThresholdMs: 45_000,
      onStale: () => { staleFired = true },
    })

    wd.start()

    for (let i = 0; i < 10; i++) {
      vi.advanceTimersByTime(15_000)
      wd.touch()
    }

    expect(staleFired).toBe(false)
    wd.stop()
  })

  it('watchdog fires exactly once per stale period', () => {
    let staleCount = 0

    const wd = createStreamWatchdog({
      staleThresholdMs: 45_000,
      onStale: () => { staleCount++ },
      checkIntervalMs: 5000,
    })

    wd.start()
    vi.advanceTimersByTime(60_000)

    expect(staleCount).toBe(1)
    wd.stop()
  })
})

describe('subscribeUserSummary watchdog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('fires onStale after 90s threshold with no events', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 90_000,
      onStale: () => { staleFired = true },
    })
    wd.start()

    vi.advanceTimersByTime(90_001)
    expect(staleFired).toBe(true)
    wd.stop()
  })

  it('does NOT fire when keepalive events arrive every 30s', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 90_000,
      onStale: () => { staleFired = true },
    })
    wd.start()

    for (let i = 0; i < 5; i++) {
      vi.advanceTimersByTime(30_000)
      wd.touch()
    }

    expect(staleFired).toBe(false)
    wd.stop()
  })
})
