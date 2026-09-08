import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// NewStrategyPanel：新建策略来源选择（AI 生成 / 手动编写 / 导入 MQL），
// 每张卡点击路由到对应动作。使用模板已按业主指令移除。

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

function renderPanel() {
  const props = {
    onNewSource: vi.fn(),
  }
  render(<NewStrategyPanel {...props} />)
  return props
}

describe('NewStrategyPanel sources', () => {
  it('renders three sources and routes each action', () => {
    const props = renderPanel()
    expect(screen.getByTestId('new-source-ai')).toBeTruthy()
    expect(screen.getByTestId('new-source-manual')).toBeTruthy()
    expect(screen.getByTestId('new-source-import')).toBeTruthy()

    fireEvent.click(screen.getByTestId('new-source-ai'))
    expect(props.onNewSource).toHaveBeenCalledWith('ai')
    fireEvent.click(screen.getByTestId('new-source-manual'))
    expect(props.onNewSource).toHaveBeenCalledWith('manual')
    fireEvent.click(screen.getByTestId('new-source-import'))
    expect(props.onNewSource).toHaveBeenCalledWith('import')
  })

})
