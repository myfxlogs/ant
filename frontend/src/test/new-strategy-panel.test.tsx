import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// NewStrategyPanel：新建策略来源选择（AI 生成 / 手动编写 / 导入 MQL / 使用模板），
// 每张卡点击路由到对应动作；模板为空时"使用模板"置灰不可点。

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))

import NewStrategyPanel from '@/pages/strategy/components/workspace/NewStrategyPanel'

function renderPanel(templateCount = 2) {
  const props = {
    onAI: vi.fn(), onManual: vi.fn(), onImport: vi.fn(), onTemplate: vi.fn(),
    templateCount,
  }
  render(<NewStrategyPanel {...props} />)
  return props
}

describe('NewStrategyPanel sources', () => {
  it('renders four sources and routes each action', () => {
    const props = renderPanel()
    expect(screen.getByTestId('new-source-ai')).toBeTruthy()
    expect(screen.getByTestId('new-source-manual')).toBeTruthy()
    expect(screen.getByTestId('new-source-import')).toBeTruthy()
    expect(screen.getByTestId('new-source-template')).toBeTruthy()

    fireEvent.click(screen.getByTestId('new-source-ai'))
    expect(props.onAI).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTestId('new-source-manual'))
    expect(props.onManual).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTestId('new-source-import'))
    expect(props.onImport).toHaveBeenCalledTimes(1)
    fireEvent.click(screen.getByTestId('new-source-template'))
    expect(props.onTemplate).toHaveBeenCalledTimes(1)
  })

  it('disables the template card when no templates exist', () => {
    const props = renderPanel(0)
    fireEvent.click(screen.getByTestId('new-source-template'))
    expect(props.onTemplate).not.toHaveBeenCalled()
  })
})
