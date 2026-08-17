import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest'
import { render, screen, act } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

// STREAM-FREEZE-1 R2: LiveStrategyPage watchActive watchdog integration test.
//
// Simulates a zombie watchActive stream: the async iterable hangs (never yields,
// never rejects) until the abort signal fires. The watchdog in the useEffect
// should detect staleness after 60s, abort the connection, and set streamError
// to show the "Connection interrupted" alert.
//
// Adversarial proof: remove the `createStreamWatchdog` from the useEffect →
// the hanging stream never aborts → streamError never becomes true →
// no "Connection interrupted" alert → RED.

const { createHangingStream } = vi.hoisted(() => {
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
  return { createHangingStream }
})

vi.mock('@/client/strategy', () => ({
  strategyActiveApi: {
    watchActive: vi.fn((_id: string, signal: AbortSignal) => createHangingStream(signal)),
    stop: vi.fn(async () => ({ success: true })),
    start: vi.fn(async () => ({ success: true })),
  },
}))

vi.mock('@/client/strategy-schedules', () => ({
  strategyScheduleV2Api: {
    list: vi.fn(async () => []),
    watch: vi.fn((signal: AbortSignal) => createHangingStream(signal)),
    toggle: vi.fn(async () => {}),
    delete: vi.fn(async () => {}),
  },
  strategyTemplateApi: {
    list: vi.fn(async () => []),
    get: vi.fn(async () => ({ id: '', name: '', code: '' })),
  },
}))

vi.mock('@/client/scheduleHealth', () => ({
  scheduleHealthApi: {
    getScheduleHealth: vi.fn(async () => null),
  },
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { defaultValue?: string }) => opts?.defaultValue ?? key,
  }),
}))

vi.mock('@/pages/strategy/LiveStrategyPageSignalDrawer', () => ({
  formatTime: (v: unknown) => String(v ?? '—'),
}))

vi.mock('@/pages/strategy/hooks/useAccountsAndSymbols', () => ({
  useAccountsAndSymbols: () => ({ accounts: [], fetchAccounts: vi.fn() }),
}))

vi.mock('@/pages/strategy/components/ScheduleLogsModal', () => ({ default: () => null }))
vi.mock('@/pages/strategy/components/ScheduleHealthModal', () => ({ default: () => null }))
vi.mock('@/pages/strategy/components/live/MyStrategiesTable', () => ({ default: () => null }))
vi.mock('@/pages/strategy/components/live/RunHistoryTab', () => ({ default: () => null }))
vi.mock('@/pages/strategy/components/live/EditParamsModal', () => ({ default: () => null }))

import LiveStrategyPage from '@/pages/strategy/LiveStrategyPage'

describe('LiveStrategyPage watchActive watchdog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })
  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllTimers()
  })

  it('watchdog detects stale watchActive stream and shows error alert', async () => {
    render(
      <MemoryRouter initialEntries={['/strategy/live?tab=strategies']}>
        <LiveStrategyPage />
      </MemoryRouter>,
    )

    // Let initial effects run and streams start
    await act(async () => {
      await vi.advanceTimersByTimeAsync(100)
    })

    // No error alert initially
    expect(screen.queryByText(/Connection interrupted/i)).toBeFalsy()

    // Advance past watchdog stale threshold (60s)
    await act(async () => {
      await vi.advanceTimersByTimeAsync(60_001)
    })

    // Stream error alert should appear (watchdog fired onStale → setStreamError(true))
    expect(screen.getByText(/Connection interrupted/i)).toBeTruthy()
  })
})
