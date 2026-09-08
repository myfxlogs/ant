import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// 业主指令：新建策略升级为侧栏分区（与我的策略/回测历史同级，含 AI 生成/
// 导入 MQL/从模板三个来源项），取消底部"新建策略/导入 MQL"按钮区。
//
// mutation: 还原 WorkspaceSidebar.tsx → 本用例 RED（底部按钮区回归、
// 新建分区缺失）。

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))
vi.mock('./SidebarStrategyList', () => ({ default: () => <div data-testid="strategy-list" /> }))
vi.mock('./SidebarRunList', () => ({ default: () => <div data-testid="run-list" /> }))

import WorkspaceSidebar from '@/pages/strategy/components/workspace/WorkspaceSidebar'

function renderSidebar() {
  const props = {
    templates: [{ id: 't1', name: '均线' }, { id: 't2', name: 'MACD' }],
    loading: false,
    selectedId: '',
    onSelect: vi.fn(),
    backtestRuns: [],
    runsLoading: false,
    onOpenHistory: vi.fn(),
    onImport: vi.fn(),
    onNew: vi.fn(),
    onNewAI: vi.fn(),
    onFirstTemplate: vi.fn(),
    collapsed: false,
    onToggle: vi.fn(),
  }
  render(<WorkspaceSidebar {...props} />)
  return props
}

describe('WorkspaceSidebar new-strategy section', () => {
  it('renders the section at sidebar top; bottom button block is gone', () => {
    renderSidebar()
    expect(screen.getByText('New Strategy')).toBeTruthy()
    // bottom action block removed: no block-style standalone import button
    expect(screen.queryByText('Import MQL')).toBeNull()
  })

  it('expands three sources and routes each action, collapsing afterwards', async () => {
    const props = renderSidebar()
    fireEvent.click(screen.getByText('New Strategy'))

    expect(await screen.findByText('AI Generate')).toBeTruthy()
    expect(screen.getByText('Import MQL')).toBeTruthy()
    expect(screen.getByText('Use Template')).toBeTruthy()

    fireEvent.click(screen.getByText('Import MQL'))
    expect(props.onImport).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText('New Strategy'))
    fireEvent.click(screen.getByText('AI Generate'))
    expect(props.onNewAI).toHaveBeenCalledTimes(1)

    fireEvent.click(screen.getByText('New Strategy'))
    fireEvent.click(screen.getByText('Use Template'))
    expect(props.onFirstTemplate).toHaveBeenCalledTimes(1)

    // selection section collapsed back after choosing a source
    expect(screen.queryByText('AI Generate')).toBeNull()
  })
})
