import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// WORKSPACE-IA：侧栏分区 = 导航。展开哪个分区，主内容区就切换到对应视图。
// 新建策略分区：点头部 → onSectionChange('new')；主区由 CenterColumn 渲染
// 来源选择面板（含手动编写）。
//
// mutation: 还原 WorkspaceSidebar.tsx（内部 accordion 状态）→ 本用例 RED。

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))
vi.mock('@/pages/strategy/components/workspace/SidebarStrategyList', () => ({ default: () => <div data-testid="strategy-list" /> }))
vi.mock('@/pages/strategy/components/workspace/SidebarRunList', () => ({ default: () => <div data-testid="run-list" /> }))

import WorkspaceSidebar from '@/pages/strategy/components/workspace/WorkspaceSidebar'

function renderSidebar(activeSection: 'new' | 'strategies' | 'history' = 'strategies') {
  const props = {
    templates: [{ id: 't1', name: '均线' }],
    loading: false,
    selectedId: '',
    onSelect: vi.fn(),
    backtestRuns: [{ id: 'r1', totalReturn: 5.2, totalTrades: 12 }],
    runsLoading: false,
    onOpenHistory: vi.fn(),
    onImport: vi.fn(),
    onNew: vi.fn(),
    onNewSource: vi.fn(),
    activeSection,
    onSectionChange: vi.fn(),
    collapsed: false,
    onToggle: vi.fn(),
  }
  render(<WorkspaceSidebar {...props} />)
  return props
}

describe('WorkspaceSidebar navigation sections', () => {
  it('only the active section renders its list', () => {
    renderSidebar('strategies')
    expect(screen.getByTestId('strategy-list')).toBeTruthy()
    expect(screen.queryByTestId('run-list')).toBeNull()
  })

  it('clicking a section header switches the active section', () => {
    const props = renderSidebar('strategies')
    fireEvent.click(screen.getByText('Backtest History'))
    expect(props.onSectionChange).toHaveBeenCalledWith('history')

    fireEvent.click(screen.getByText('New Strategy'))
    expect(props.onSectionChange).toHaveBeenCalledWith('new')
  })

  it('expanding the new-strategy section shows the four uniform source items', () => {
    const props = renderSidebar('new')
    expect(screen.getByText('AI Generate')).toBeTruthy()
    expect(screen.getByText('Manual Coding')).toBeTruthy()
    expect(screen.getByText('Import MQL')).toBeTruthy()
    expect(screen.getByText('Use Template')).toBeTruthy()

    fireEvent.click(screen.getByText('AI Generate'))
    expect(props.onNewSource).toHaveBeenCalledWith('ai')
    // main area switch also closes the right panel (CenterColumn behavior)
  })

  it('highlights the active section and shows counts on inactive ones', () => {
    renderSidebar('history')
    expect(screen.getByTestId('run-list')).toBeTruthy()
    expect(screen.queryByTestId('strategy-list')).toBeNull()
    expect(screen.getByText('1')).toBeTruthy() // strategies count badge
  })
})
