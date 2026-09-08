import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen } from '@testing-library/react'

// FIX-2026-09-08-AI-SETTINGS-ENTRY 对抗证明。
//
// 业主要求：AI 网关设置入口必须常驻工作区 tab 栏（右侧齿轮），
// 不打开 AI 聊天/代码 tab 也能进入模型配置。
//
// mutation: 还原 WorkspaceCenterTabBar.tsx（无齿轮入口）→ 本测试 RED。

const ModalStub = () => <div data-testid="ai-settings-modal-stub">AI_SETTINGS_OPEN</div>
vi.mock('@/pages/strategy/components/workspace/AISettingsModal', () => ({ default: ModalStub }))
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, fb?: unknown) => (typeof fb === 'string' ? fb : k) }),
}))

import WorkspaceCenterTabBar from '@/pages/strategy/components/workspace/WorkspaceCenterTabBar'

function renderBar(centerTab: 'code' | 'chat' = 'code') {
  return render(
    <WorkspaceCenterTabBar
      centerTab={centerTab}
      setCenterTab={vi.fn()}
      setSidebarDrawerOpen={vi.fn()}
      setBtModalOpen={vi.fn()}
      setIndicatorDrawerOpen={vi.fn()}
      rightPanelTab={null}
      setRightPanelTab={vi.fn()}
      code={{ code: '', lastValidatedCode: '', lastSavedId: '', setSaveModalOpen: vi.fn() } as never}
      account={{} as never}
      templates={{ list: [], selectedId: '' } as never}
    />,
  )
}

describe('WorkspaceCenterTabBar AI settings entry', () => {
  it('shows the gear on the code tab and opens AI settings on click', async () => {
    renderBar('code')
    const gear = document.querySelector('.anticon-setting')?.closest('button') as HTMLButtonElement
    expect(gear).toBeTruthy()
    fireEvent.click(gear)
    expect(await screen.findByTestId('ai-settings-modal-stub')).toBeTruthy()
  })

  it('gear stays available on the chat tab (entry is unconditional)', () => {
    renderBar('chat')
    expect(document.querySelector('.anticon-setting')).toBeTruthy()
  })
})
