import { describe, it, expect, beforeEach } from 'vitest'
import { render, waitFor, act } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { QueryCacheGuard } from '@/providers/QueryProvider'
import { useAuthStore } from '@/stores/authStore'
import { queryKeys } from '@/queries/queryKeys'
import type { User } from '@/types/auth'

// QC-CACHE-LEAK: 用户切换时 TanStack Query 缓存未清除（A 登出 → B 登录看到 A 的账户列表）。
// 根因：账户列表等查询缓存 key 不含 userId（queryKeys.accounts.list()），staleTime 30s 内
// B 登录后命中 A 的缓存。修复 = QueryCacheGuard 监听 user.id 变化 → queryClient.clear()。
//
// 对抗证明：删掉 QueryProvider.tsx 中 QueryCacheGuard 的 queryClient.clear() 行，
// 下面 "clears query cache" 两个用例必红（缓存残留 → B 看到 A 的数据）。

function makeUser(id: string): User {
  return {
    id,
    email: `${id}@test.com`,
    nickname: id,
    avatar: '',
    role: 'user',
    permissions: [],
    capabilityTier: 0,
    status: 'active',
    accountNumber: '',
    last_login_at: null,
    created_at: '',
    updated_at: '',
  }
}

function renderGuard(): QueryClient {
  const queryClient = new QueryClient()
  queryClient.setDefaultOptions({ queries: { retry: false } })
  render(
    <QueryClientProvider client={queryClient}>
      <QueryCacheGuard>{null as ReactNode}</QueryCacheGuard>
    </QueryClientProvider>,
  )
  return queryClient
}

function seedAccountCache(queryClient: QueryClient, owner: string) {
  queryClient.setQueryData(queryKeys.accounts.list(), [
    { id: `acc-${owner}`, login: '12345', userId: owner },
  ])
}

describe('QueryCacheGuard — cross-user query cache leak (QC-CACHE-LEAK)', () => {
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
  })

  it('clears query cache when user id changes (A logout → B login)', async () => {
    act(() => useAuthStore.setState({ user: makeUser('user-a'), isAuthenticated: true }))
    const queryClient = renderGuard()
    seedAccountCache(queryClient, 'a')
    expect(queryClient.getQueryData(queryKeys.accounts.list())).toBeDefined()

    // A 登出（store logout 置 user null — 被动登出 tokenLifecycle 走同一路径）
    act(() => useAuthStore.setState({ user: null, isAuthenticated: false }))
    await waitFor(() =>
      expect(queryClient.getQueryData(queryKeys.accounts.list())).toBeUndefined(),
    )

    // B 登录后即使缓存被重新写入，也是 B 自己的数据
    act(() => useAuthStore.setState({ user: makeUser('user-b'), isAuthenticated: true }))
    seedAccountCache(queryClient, 'b')
    expect(queryClient.getQueryData(queryKeys.accounts.list())).toBeDefined()
  })

  it('clears query cache on direct A → B switch without intermediate logout', async () => {
    act(() => useAuthStore.setState({ user: makeUser('user-a'), isAuthenticated: true }))
    const queryClient = renderGuard()
    seedAccountCache(queryClient, 'a')

    act(() => useAuthStore.setState({ user: makeUser('user-b') }))
    await waitFor(() =>
      expect(queryClient.getQueryData(queryKeys.accounts.list())).toBeUndefined(),
    )
  })

  it('does NOT clear cache on initial mount with an already-hydrated user (remember-me boot)', async () => {
    // 模拟 remember-me：authStore 同步 hydrate 后 Guard 才首次挂载。
    // 首次挂载不得 clear（否则每次刷新都白清缓存）。
    act(() => useAuthStore.setState({ user: makeUser('user-a'), isAuthenticated: true }))
    const queryClient = renderGuard()
    seedAccountCache(queryClient, 'a')

    // 等一个事件循环，确认没有延迟的 clear
    await new Promise((r) => setTimeout(r, 20))
    expect(queryClient.getQueryData(queryKeys.accounts.list())).toBeDefined()
  })

  it('does NOT clear on undefined → B login (boot logged-out, nothing stale to clear)', async () => {
    // 启动时无用户（prevUserId === undefined 分支不 clear）。
    const queryClient = renderGuard()
    act(() => useAuthStore.setState({ user: makeUser('user-b'), isAuthenticated: true }))
    await new Promise((r) => setTimeout(r, 20))
    // 无 clear 发生 — QueryClient 上无任何查询被移除
    expect(queryClient.getQueryCache().getAll().length).toBe(0)
  })
})
