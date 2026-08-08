import { describe, it, expect, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { Button, Tag, Statistic, Empty, Spin, Typography } from 'antd'
import { useAuthStore } from '@/stores/authStore'
import { useUIStore } from '@/stores/uiStore'
import { useTradingStore } from '@/stores/tradingStore'
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

beforeEach(() => {
  useAuthStore.setState({
    user: null,
    accessToken: null,
    isAuthenticated: false,
    _hasHydrated: false,
    _rememberMe: false,
  })
  useUIStore.setState({ theme: 'light', sidebarCollapsed: false })
  useTradingStore.setState({
    positions: [],
    positionsMap: new Map(),
    tradeLogs: [],
    accountInfo: { balance: 0, credit: 0, profit: 0, profitPercent: 0, equity: 0, margin: 0, freeMargin: 0, marginLevel: 0 },
    accountInfoMap: new Map(),
    accountReceivedData: new Set(),
    lastStreamProfitAtByAccount: new Map(),
    currentAccountId: null,
    loading: false,
  })
})

describe('Component smoke: Antd Button', () => {
  it('renders button with text', () => {
    render(<Button type="primary">Submit</Button>)
    expect(screen.getByText('Submit')).toBeTruthy()
  })

  it('renders disabled button', () => {
    render(<Button disabled>Disabled</Button>)
    const btn = screen.getByText('Disabled').closest('button')
    expect(btn?.disabled).toBe(true)
  })
})

describe('Component smoke: Antd Tag', () => {
  it('renders tag with content', () => {
    render(<Tag color="green">Connected</Tag>)
    expect(screen.getByText('Connected')).toBeTruthy()
  })
})

describe('Component smoke: Antd Statistic', () => {
  it('renders statistic with value', () => {
    const { container } = render(<Statistic title="Balance" value={1234.56} />)
    expect(screen.getByText('Balance')).toBeTruthy()
    expect(container.textContent).toContain('1,234.56')
  })
})

describe('Component smoke: Antd Empty', () => {
  it('renders empty state', () => {
    const { container } = render(<Empty description="No data" />)
    expect(container.querySelector('.ant-empty')).toBeTruthy()
    expect(container.textContent).toContain('No data')
  })
})

describe('Component smoke: Antd Spin', () => {
  it('renders spin with tip', () => {
    render(<Spin tip="Loading..." />)
    expect(screen.getByText('Loading...')).toBeTruthy()
  })
})

describe('Component smoke: Antd Typography', () => {
  it('renders paragraph', () => {
    render(<Typography.Paragraph>Hello World</Typography.Paragraph>)
    expect(screen.getByText('Hello World')).toBeTruthy()
  })
})

describe('Component smoke: authStore + UI integration', () => {
  it('renders user nickname when authenticated', () => {
    const user = makeUser({ nickname: 'Alice' })
    useAuthStore.getState().setUser(user)
    render(
      <div>
        <span data-testid="user-name">{useAuthStore.getState().user?.nickname}</span>
        <Tag color={useAuthStore.getState().isAuthenticated ? 'green' : 'red'}>
          {useAuthStore.getState().isAuthenticated ? 'Online' : 'Offline'}
        </Tag>
      </div>,
    )
    expect(screen.getByTestId('user-name').textContent).toBe('Alice')
    expect(screen.getByText('Online')).toBeTruthy()
  })

  it('renders offline when not authenticated', () => {
    render(
      <Tag color={useAuthStore.getState().isAuthenticated ? 'green' : 'red'}>
        {useAuthStore.getState().isAuthenticated ? 'Online' : 'Offline'}
      </Tag>,
    )
    expect(screen.getByText('Offline')).toBeTruthy()
  })
})

describe('Component smoke: tradingStore + Statistic integration', () => {
  it('renders zero balance by default', () => {
    const balance = useTradingStore.getState().accountInfo.balance
    const { container } = render(<Statistic title="Balance" value={balance} precision={2} />)
    expect(container.textContent).toContain('Balance')
    expect(container.textContent).toContain('0.00')
  })

  it('renders updated balance after setAccountInfo', () => {
    useTradingStore.getState().setCurrentAccountId('acc-1')
    useTradingStore.getState().setAccountInfo({ balance: 5000 })
    const balance = useTradingStore.getState().accountInfo.balance
    const { container } = render(<Statistic title="Balance" value={balance} precision={2} />)
    expect(container.textContent).toContain('5,000.00')
  })
})
