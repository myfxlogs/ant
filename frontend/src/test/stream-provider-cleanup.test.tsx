import { describe, it, expect, vi } from 'vitest'
import { render, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { StreamProvider } from '@/providers/StreamProvider'
import { useAuthStore } from '@/stores/authStore'
import type { User } from '@/types/auth'

// STREAM-FREEZE-1 Task C: StreamProvider cleanup order + honest status.
//
// Adversarial proof #5: Trigger onError → assert old unsubscribe is called
// BEFORE being set to null. If the cleanup order is wrong (null before call),
// the old stream's backoff setTimeout can revive a dead stream → double events.

let unsubEventsCalled = false

vi.mock('@/client/stream', () => ({
  subscribeEvents: vi.fn(() => {
    return () => { unsubEventsCalled = true }
  }),
  subscribeUserSummary: vi.fn(() => {
    return () => {}
  }),
}))

vi.mock('@/bridge/bridgeStreamEvents', () => ({
  handleOrderUpdate: vi.fn(),
  handleAccountStatus: vi.fn(),
  handlePositionSnapshot: vi.fn(),
}))

vi.mock('@/bridge/bridgeProfitEvents', () => ({
  handleProfitUpdate: vi.fn(),
  cleanupProfitBridge: vi.fn(),
}))

vi.mock('@/bridge/bridgeUserSummary', () => ({
  handleUserSummary: vi.fn(),
}))

function makeUser(id: string): User {
  return {
    id, email: `${id}@test.com`, nickname: id, avatar: '', role: 'user',
    permissions: [], capabilityTier: 0, status: 'active', accountNumber: '',
    last_login_at: null, created_at: '', updated_at: '',
  }
}

describe('StreamProvider cleanup order (Task C)', () => {
  it('onError calls unsubEventsRef before nulling it', async () => {
    const queryClient = new QueryClient()
    queryClient.setDefaultOptions({ queries: { retry: false } })

    render(
      <QueryClientProvider client={queryClient}>
        <StreamProvider>{null as ReactNode}</StreamProvider>
      </QueryClientProvider>,
    )

    const { subscribeEvents } = await import('@/client/stream')
    const mockSub = vi.mocked(subscribeEvents)

    // Login to trigger stream subscription.
    act(() => {
      useAuthStore.setState({ user: makeUser('u1'), isAuthenticated: true })
    })

    await waitFor(() => {
      expect(mockSub).toHaveBeenCalled()
    })

    // The onError callback is in the callbacks passed to subscribeEvents.
    const callbacks = mockSub.mock.calls[0][1] as {
      onError: () => void
      onStale: () => void
    }

    // Reset tracking before triggering error.
    unsubEventsCalled = false

    // Simulate error → should call unsub (cleanup) before nulling.
    expect(() => callbacks.onError()).not.toThrow()

    // After onError, the old unsub should have been called (cleanup order fix).
    expect(unsubEventsCalled).toBe(true)

    // onStale should not throw (honest status callback).
    expect(() => callbacks.onStale()).not.toThrow()

    // Cleanup
    act(() => {
      useAuthStore.setState({ user: null, isAuthenticated: false })
    })
  })
})
