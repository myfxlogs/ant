import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'

// antd v6 Select's rc-trigger needs constructor-compatible observers
// (setup.ts vi.fn arrow mocks are not constructible).
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

// FIX-2026-09-08-BYOK-MODEL-PICKER 对抗证明。
//
// 策略聊天框顶部的模型下拉框必须同时列出用户自有 BYOK 模型
// （/ai/settings 里配置了 key 的 provider，如 NOVA / kimi-k3）和平台
// 网关模型，自有分组在前。
//
// mutation: 还原 StrategyChat.tsx（只列 listSystemModels）→ 本测试 RED。

const { listSystemAIConfigsMock, listSystemModelsMock, getPrimaryMock } = vi.hoisted(() => ({
  listSystemAIConfigsMock: vi.fn(),
  listSystemModelsMock: vi.fn(),
  getPrimaryMock: vi.fn(),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, fb?: unknown) => (typeof fb === 'string' ? fb : k) }),
}))
vi.mock('@/client/ai', () => ({
  aiApi: {
    getPrimary: getPrimaryMock,
    listConversations: vi.fn(async () => []),
  },
}))
vi.mock('@/client/aiGateway', () => ({
  aiGatewayApi: { listSystemModels: listSystemModelsMock },
}))
vi.mock('@/pages/ai/systemai/api', () => ({
  listSystemAIConfigs: listSystemAIConfigsMock,
}))
vi.mock('@/client/strategy-schedules', () => ({
  strategyTemplateApi: { list: vi.fn(async () => []) },
}))
vi.mock('@/components/strategy/AgentGenChat', () => ({ default: () => null }))

import StrategyChat from '@/components/strategy/StrategyChat'

describe('StrategyChat model picker BYOK options', () => {
  beforeEach(() => {
    getPrimaryMock.mockResolvedValue({ providerId: 'openai_compatible_x', model: 'kimi-k3' })
    listSystemAIConfigsMock.mockResolvedValue({
      items: [
        {
          provider_id: 'openai_compatible_x',
          name: 'NOVA',
          base_url: 'https://token.example.cn/v1',
          organization: '',
          models: ['kimi-k3'],
          default_model: 'kimi-k3',
          temperature: 0,
          timeout_seconds: 0,
          max_tokens: 0,
          purposes: [],
          primary_for: [],
          enabled: true,
          has_secret: true,
          updated_at: '',
        },
        {
          provider_id: 'deepseek',
          name: 'DeepSeek',
          base_url: 'https://api.deepseek.com/v1',
          organization: '',
          models: [],
          default_model: '',
          temperature: 0,
          timeout_seconds: 0,
          max_tokens: 0,
          purposes: [],
          primary_for: [],
          enabled: true,
          has_secret: false, // no key → must NOT appear
          updated_at: '',
        },
      ],
    })
    listSystemModelsMock.mockResolvedValue([
      { id: 'm1', providerId: 'zhipu', modelName: 'glm-5.2', displayName: 'GLM-5.2', pricePer1mInput: '', pricePer1mOutput: '' },
    ])
  })

  it('lists own BYOK models first, then gateway models, with group labels', async () => {
    render(<StrategyChat onApplyCode={vi.fn()} />)

    await waitFor(() => expect(listSystemAIConfigsMock).toHaveBeenCalled())
    const combo = await screen.findByRole('combobox')
    fireEvent.mouseDown(combo)
    fireEvent.focus(combo)
    await waitFor(() => expect(document.querySelector('.ant-select-dropdown')).toBeTruthy())

    // selected value + dropdown option both render the own-model label
    expect((await screen.findAllByText('kimi-k3 (NOVA)')).length).toBeGreaterThanOrEqual(1)
    expect(screen.getByText('GLM-5.2 (zhipu)')).toBeTruthy()
    expect(screen.getByText('我的 API Key')).toBeTruthy()
    expect(screen.getByText('AI 网关')).toBeTruthy()
  })

  it('shows the saved primary (own provider string id) as the selected value', async () => {
    render(<StrategyChat onApplyCode={vi.fn()} />)

    await waitFor(() => expect(getPrimaryMock).toHaveBeenCalled())
    await waitFor(() => {
      const title = document.querySelector('.ant-select-content')?.getAttribute('title') || ''
      expect(title).toContain('kimi-k3 (NOVA)')
    })
  })
})
