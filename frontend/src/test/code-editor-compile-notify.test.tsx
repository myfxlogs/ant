import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

// 业主反馈：工作台"编译失败"状态条只显示文案、不带原因。
// 修复：进入失败态时右下角 notification 弹出完整原因 + 状态条显示原因首行；
// AI chat 上下文由服务端注入（编译 currentCode），前端提示用户可直接让 AI 修。
//
// mutation: 还原 CodeEditorArea.tsx → 本用例 RED（无弹窗、状态条无原因）。

const { checkCodeMock, notificationErrorMock } = vi.hoisted(() => ({
  checkCodeMock: vi.fn(),
  notificationErrorMock: vi.fn(),
}))

vi.mock('antd', async (importOriginal) => {
  const actual = await importOriginal<typeof import('antd')>()
  return {
    ...actual,
    notification: { error: notificationErrorMock, open: vi.fn(), success: vi.fn(), warning: vi.fn() },
  }
})
vi.mock('@/client/strategy', () => ({
  strategyVersionApi: { checkCode: checkCodeMock },
}))
vi.mock('@/components/strategy/StrategyCodeEditor', () => ({
  default: (props: { diagnostics: Array<{ message: string }> }) => (
    <div data-testid="editor">{props.diagnostics.map((d) => d.message).join('||')}</div>
  ),
}))
vi.mock('../editor/ImportEAPanel', () => ({ default: () => null }))
vi.mock('react-i18next', () => ({
  initReactI18next: { type: '3rdParty', init: () => {} },
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))

import CodeEditorArea from '@/pages/strategy/components/workspace/CodeEditorArea'

const BROKEN_CODE = 'def run_dataframe(df, params):\n    price = undefined_fn(df)\n    return price\n'.repeat(2)

function renderEditor(code = BROKEN_CODE) {
  return render(
    <CodeEditorArea
      code={code}
      importMode={false}
      onSetImportMode={vi.fn()}
      onSetCode={vi.fn()}
    />,
  )
}

describe('CodeEditorArea compile-failure visibility', () => {
  beforeEach(() => {
    checkCodeMock.mockReset()
    notificationErrorMock.mockReset()
  })

  it('surfaces the compile error reason via notification and status bar', async () => {
    const reason = 'compile Python to IR: line 3: local arrays not supported: price'
    checkCodeMock.mockResolvedValue({ compileSuccess: false, compileError: reason, blindSpots: [], coverageScore: 0 })
    renderEditor()

    // status bar carries the reason's first line, not just the bare label
    expect((await screen.findAllByText(/compile Python to IR: line 3/)).length).toBeGreaterThanOrEqual(1)
    // editor diagnostics include the reason
    expect(screen.getByTestId('editor').textContent).toContain(reason)
    // prominent notification fired once with the full reason
    await waitFor(() => expect(notificationErrorMock).toHaveBeenCalledTimes(1))
    const arg = notificationErrorMock.mock.calls[0][0] as { message: string; description: unknown }
    expect(String(arg.message)).toContain('CompileFailed')
    expect(arg.description).toBeTruthy()
  })

  it('does not re-notify while the failure persists across re-checks', async () => {
    const reason = 'still broken'
    checkCodeMock.mockResolvedValue({ compileSuccess: false, compileError: reason, blindSpots: [], coverageScore: 0 })
    const { rerender } = renderEditor(BROKEN_CODE)
    await waitFor(() => expect(notificationErrorMock).toHaveBeenCalledTimes(1))

    checkCodeMock.mockClear()
    rerender(<CodeEditorArea code={BROKEN_CODE + '\n# edited'} importMode={false} onSetImportMode={vi.fn()} onSetCode={vi.fn()} />)
    await waitFor(() => expect(checkCodeMock).toHaveBeenCalledTimes(1), { timeout: 3000 })
    // still in error state: no duplicate notification
    expect(notificationErrorMock).toHaveBeenCalledTimes(1)
  })
})
