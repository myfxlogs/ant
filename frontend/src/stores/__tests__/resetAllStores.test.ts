import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/stores/authStore'
import { useTradingStore } from '@/stores/tradingStore'
import { useNotificationStore } from '@/stores/notificationStore'
import { useWorkspaceStore } from '@/stores/workspaceStore'
import { useChartIndicatorsStore } from '@/stores/chartIndicatorsStore'
import { resetAllStores } from '@/stores/resetAllStores'
import type { User } from '@/types/auth'

function makeUser(overrides: Partial<User> = {}): User {
  return {
    id: 'u1',
    email: 'test@test.com',
    nickname: 'tester',
    avatar: '',
    role: 'user',
    permissions: [],
    capabilityTier: 0,
    status: 'active',
    accountNumber: '',
    last_login_at: null,
    created_at: '',
    updated_at: '',
    ...overrides,
  }
}

describe('resetAllStores', () => {
  beforeEach(() => {
    localStorage.clear()
    sessionStorage.clear()
    useAuthStore.setState({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      _hasHydrated: false,
      _rememberMe: false,
    })
    useTradingStore.getState().reset()
    useNotificationStore.getState().reset()
    useWorkspaceStore.getState().reset()
    useChartIndicatorsStore.getState().reset()
  })

  it('clears tradingStore positions and account data', () => {
    const user = makeUser()
    useAuthStore.getState().setTokens('tok', 'refresh', user, true)
    useTradingStore.getState().setPositions('acc-1', [{ ticket: 1, symbol: 'EURUSD', type: 'buy', volume: 0.1, openPrice: 1.1, openTime: '', currentPrice: 1.1, sl: 0, tp: 0, profit: 0, swap: 0, commission: 0 }])
    useTradingStore.getState().setAccountInfo({ balance: 10000, equity: 10000 })
    useTradingStore.getState().setCurrentAccountId('acc-1')

    expect(useTradingStore.getState().positions.length).toBe(1)
    expect(useTradingStore.getState().currentAccountId).toBe('acc-1')

    resetAllStores()

    expect(useTradingStore.getState().positions).toEqual([])
    expect(useTradingStore.getState().positionsMap.size).toBe(0)
    expect(useTradingStore.getState().currentAccountId).toBeNull()
    expect(useTradingStore.getState().accountInfo.balance).toBe(0)
  })

  it('clears notificationStore', () => {
    useNotificationStore.getState().addNotification({ type: 'system', title: 'Test', message: 'Hello' })
    expect(useNotificationStore.getState().notifications.length).toBe(1)

    resetAllStores()

    expect(useNotificationStore.getState().notifications).toEqual([])
    expect(useNotificationStore.getState().unreadCount).toBe(0)
  })

  it('clears workspaceStore user-scoped state', () => {
    useWorkspaceStore.getState().setAccountId('acc-1')
    useWorkspaceStore.getState().setSymbol('EURUSD')
    useWorkspaceStore.getState().setCurrentCode('some code')

    resetAllStores()

    expect(useWorkspaceStore.getState().accountId).toBe('')
    expect(useWorkspaceStore.getState().symbol).toBe('')
    expect(useWorkspaceStore.getState().currentCode).toBe('')
  })

  it('clears chartIndicatorsStore', () => {
    useChartIndicatorsStore.getState().addIndicator('SMA')
    expect(useChartIndicatorsStore.getState().active.length).toBe(1)

    resetAllStores()

    expect(useChartIndicatorsStore.getState().active).toEqual([])
  })

  // ── Adversarial proof: without resetAllStores, data leaks ──

  it('adversarial: simulating A→B switch without reset leaves A data in tradingStore', () => {
    // User A logs in, loads positions
    const userA = makeUser({ id: 'user-a' })
    useAuthStore.getState().setTokens('tok-a', 'refresh', userA, true)
    useTradingStore.getState().setPositions('acc-a', [{ ticket: 1, symbol: 'EURUSD', type: 'buy', volume: 0.1, openPrice: 1.1, openTime: '', currentPrice: 1.1, sl: 0, tp: 0, profit: 0, swap: 0, commission: 0 }])
    useTradingStore.getState().setCurrentAccountId('acc-a')
    expect(useTradingStore.getState().positions.length).toBe(1)

    // User A logs out (only authStore cleared, no resetAllStores — the bug)
    useAuthStore.getState().logout()

    // User B logs in
    const userB = makeUser({ id: 'user-b' })
    useAuthStore.getState().setTokens('tok-b', 'refresh', userB, true)

    // WITHOUT resetAllStores, A's data is still in tradingStore
    expect(useTradingStore.getState().positions.length).toBe(1) // ← bug: A's data leaked
  })

  it('adversarial: with resetAllStores, A→B switch has clean tradingStore', () => {
    // User A logs in, loads positions
    const userA = makeUser({ id: 'user-a' })
    useAuthStore.getState().setTokens('tok-a', 'refresh', userA, true)
    useTradingStore.getState().setPositions('acc-a', [{ ticket: 1, symbol: 'EURUSD', type: 'buy', volume: 0.1, openPrice: 1.1, openTime: '', currentPrice: 1.1, sl: 0, tp: 0, profit: 0, swap: 0, commission: 0 }])
    useTradingStore.getState().setCurrentAccountId('acc-a')
    expect(useTradingStore.getState().positions.length).toBe(1)

    // User A logs out WITH resetAllStores (the fix)
    useAuthStore.getState().logout()
    resetAllStores()

    // User B logs in
    const userB = makeUser({ id: 'user-b' })
    useAuthStore.getState().setTokens('tok-b', 'refresh', userB, true)

    // WITH resetAllStores, tradingStore is clean
    expect(useTradingStore.getState().positions).toEqual([])
    expect(useTradingStore.getState().currentAccountId).toBeNull()
  })
})
