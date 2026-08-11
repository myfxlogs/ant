import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { marketplaceClient } from '@/client/connect'
import BottomPanelSection from '@/pages/strategy/components/workspace/BottomPanelSection'
import LivePerformanceTab from '@/pages/marketplace/components/LivePerformanceTab'

// Mock marketplaceClient for LivePerformanceTab (T8)
vi.mock('@/client/connect', () => ({
  marketplaceClient: {
    getLivePerformance: vi.fn(),
    linkLiveAccount: vi.fn(),
  },
}))
vi.mock('@/client/account', () => ({
  accountApi: {
    list: vi.fn().mockResolvedValue([]),
  },
}))

// T3: Render real BottomPanelSection with isMobile=true, open Drawer,
// assert backtest content is visible.
// Adversarial: delete `{mobileTab === 'backtest' && backtestContent}` line →
// backtest content not in DOM → getByTestId throws → red.
describe('UX-4 T3: BottomPanelSection mobile Drawer renders backtest content', () => {
  it('renders backtest content when Drawer is opened on mobile', async () => {
    render(
      <BottomPanelSection
        isMobile={true}
        collapsed={false}
        onToggleCollapsed={() => {}}
        positions={[]}
        recentTrades={[]}
        onClosePosition={() => {}}
        accountId=""
        symbol=""
        qtPositions={[]}
        quickTradeCollapsed={false}
        onToggleQuickTrade={() => {}}
        backtestContent={<div data-testid="test-backtest-content">Backtest Results</div>}
      />
    )

    // Click the floating button to open the Drawer
    const openBtn = screen.getByRole('button')
    fireEvent.click(openBtn)

    // Drawer content should now be rendered — backtest tab is default
    await waitFor(() => {
      expect(screen.getByTestId('test-backtest-content')).toBeTruthy()
    })
    expect(screen.getByText('Backtest Results')).toBeTruthy()
  })
})

// T8: Render real LivePerformanceTab with mocked fetch failure,
// assert Alert + Retry button appear.
// Adversarial: delete error rendering block (lines 110-121) →
// Alert not rendered → querySelector finds nothing → red.
describe('UX-2 T8: LivePerformanceTab error state', () => {
  beforeEach(() => {
    vi.mocked(marketplaceClient.getLivePerformance).mockReset()
  })

  it('renders Alert with retry button when fetch fails', async () => {
    vi.mocked(marketplaceClient.getLivePerformance).mockRejectedValue(new Error('network error'))

    const { container } = render(
      <LivePerformanceTab strategyId="test-strategy-1" isOwner={false} />
    )

    // Wait for error state to render (component starts with loading=true)
    await waitFor(() => {
      const alert = container.querySelector('.ant-alert-error')
      expect(alert).toBeTruthy()
    })

    // Verify retry button exists
    const retryBtn = container.querySelector('.ant-alert .ant-btn')
    expect(retryBtn).toBeTruthy()

    // Click retry → getLivePerformance called again
    fireEvent.click(retryBtn!)
    await waitFor(() => {
      expect(marketplaceClient.getLivePerformance).toHaveBeenCalledTimes(2)
    })
  })
})

// T4: tsc --noEmit -p tsconfig.app.json catches type errors.
// This is a meta-test: verify the build script contains tsc.
// Adversarial: remove tsc from build script → assertion fails → red.
describe('UX-8 T4/T5: build script contains tsc gate', () => {
  it('package.json build script includes tsc --noEmit', async () => {
    const pkg = await import('../../package.json') as { scripts?: { build?: string } }
    const buildScript = pkg.scripts?.build || ''
    expect(buildScript).toContain('tsc')
    expect(buildScript).toContain('--noEmit')
    expect(buildScript).toContain('tsconfig.app.json')
    expect(buildScript).toContain('vite build')
  })

  it('build script is not just vite build (tsc must come first)', async () => {
    const pkg = await import('../../package.json') as { scripts?: { build?: string } }
    const buildScript = pkg.scripts?.build || ''
    const tscIndex = buildScript.indexOf('tsc')
    const viteIndex = buildScript.indexOf('vite build')
    expect(tscIndex).toBeGreaterThanOrEqual(0)
    expect(viteIndex).toBeGreaterThan(tscIndex)
  })
})

