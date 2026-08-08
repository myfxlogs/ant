import { describe, it, expect, beforeEach } from 'vitest'
import { useAuthStore } from '@/stores/authStore'
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

describe('authStore', () => {
  beforeEach(() => {
    useAuthStore.setState({
      user: null,
      accessToken: null,
      isAuthenticated: false,
      _hasHydrated: false,
      _rememberMe: false,
    })
  })

  it('starts unauthenticated', () => {
    const s = useAuthStore.getState()
    expect(s.isAuthenticated).toBe(false)
    expect(s.user).toBeNull()
    expect(s.accessToken).toBeNull()
  })

  it('setUser sets user and isAuthenticated', () => {
    const user = makeUser()
    useAuthStore.getState().setUser(user)
    expect(useAuthStore.getState().user).toEqual(user)
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
  })

  it('setUser(null) clears auth', () => {
    const user = makeUser()
    useAuthStore.getState().setUser(user)
    useAuthStore.getState().setUser(null)
    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().isAuthenticated).toBe(false)
  })

  it('setAccessToken sets token and isAuthenticated', () => {
    useAuthStore.getState().setAccessToken('tok123')
    expect(useAuthStore.getState().accessToken).toBe('tok123')
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
  })

  it('setTokens sets all fields', () => {
    const user = makeUser()
    useAuthStore.getState().setTokens('access1', 'refresh1', user, true)
    const s = useAuthStore.getState()
    expect(s.accessToken).toBe('access1')
    expect(s.isAuthenticated).toBe(true)
    expect(s._hasHydrated).toBe(true)
    expect(s.user).toEqual(user)
    expect(s._rememberMe).toBe(true)
  })

  it('setTokens without user sets user to null', () => {
    useAuthStore.getState().setTokens('access1', 'refresh1')
    expect(useAuthStore.getState().user).toBeNull()
    expect(useAuthStore.getState().isAuthenticated).toBe(true)
  })

  it('logout clears all auth state', () => {
    const user = makeUser()
    useAuthStore.getState().setTokens('access1', 'refresh1', user, true)
    useAuthStore.getState().logout()
    const s = useAuthStore.getState()
    expect(s.user).toBeNull()
    expect(s.accessToken).toBeNull()
    expect(s.isAuthenticated).toBe(false)
    expect(s._rememberMe).toBe(false)
  })

  it('setHydrated toggles _hasHydrated', () => {
    useAuthStore.getState().setHydrated(true)
    expect(useAuthStore.getState()._hasHydrated).toBe(true)
    useAuthStore.getState().setHydrated(false)
    expect(useAuthStore.getState()._hasHydrated).toBe(false)
  })
})
