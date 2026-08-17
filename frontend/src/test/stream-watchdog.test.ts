import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { createStreamWatchdog } from '@/client/streamWatchdog'

// STREAM-FREEZE-1 Task A0: streamWatchdog helper unit test.
//
// Adversarial proof: delete the staleness check inside `check()` →
// onStale never fires after timeout → test RED.

describe('streamWatchdog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  it('fires onStale when no touch exceeds threshold', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleFired = true },
      checkIntervalMs: 1000,
    })
    wd.start()

    // Advance past threshold without touching.
    vi.advanceTimersByTime(10_001)
    expect(staleFired).toBe(true)
    wd.stop()
  })

  it('does NOT fire onStale when touch keeps stream alive', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleFired = true },
      checkIntervalMs: 1000,
    })
    wd.start()

    // Touch every 5s — never exceeds 10s threshold.
    vi.advanceTimersByTime(5_000)
    wd.touch()
    vi.advanceTimersByTime(5_000)
    wd.touch()
    vi.advanceTimersByTime(5_000)
    wd.touch()
    vi.advanceTimersByTime(5_000)

    expect(staleFired).toBe(false)
    wd.stop()
  })

  it('does NOT fire after stop (zero leak)', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleFired = true },
      checkIntervalMs: 1000,
    })
    wd.start()
    wd.stop()

    vi.advanceTimersByTime(30_000)
    expect(staleFired).toBe(false)
  })

  it('checkNow triggers immediate staleness check', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleFired = true },
      checkIntervalMs: 60_000, // long interval so only checkNow can fire
    })
    wd.start()

    // Advance past threshold but interval hasn't fired yet.
    vi.advanceTimersByTime(10_001)
    expect(staleFired).toBe(false)

    // checkNow should detect stale immediately.
    wd.checkNow()
    expect(staleFired).toBe(true)
    wd.stop()
  })

  it('start resets lastEventAt (grace period = threshold)', () => {
    let staleFired = false
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleFired = true },
      checkIntervalMs: 1000,
    })
    // Start, then immediately advance just under threshold.
    wd.start()
    vi.advanceTimersByTime(9_999)
    expect(staleFired).toBe(false)
    wd.stop()
  })

  it('touch resets staleTriggered so subsequent stale can fire again', () => {
    let staleCount = 0
    const wd = createStreamWatchdog({
      staleThresholdMs: 10_000,
      onStale: () => { staleCount++ },
      checkIntervalMs: 1000,
    })
    wd.start()
    vi.advanceTimersByTime(10_001)
    expect(staleCount).toBe(1)

    // Touch resets, then stale can fire again.
    wd.touch()
    vi.advanceTimersByTime(11_000)
    expect(staleCount).toBe(2)
    wd.stop()
  })
})
