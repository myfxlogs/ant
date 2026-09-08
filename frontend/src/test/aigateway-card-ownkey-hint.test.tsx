import { describe, it, expect, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'

// 遗留项④审计修复：AIGatewayCard 网关模式需提示"自有 Key 优先于网关模型"
// 的运行时语义（resolveAllChatProviders 仅把网关当兜底）。
//
// mutation: 还原 AIGatewayCard.tsx 的提示 → 本用例 RED。

const { listSystemAIConfigsMock, listSystemModelsMock } = vi.hoisted(() => ({
  listSystemAIConfigsMock: vi.fn(),
  listSystemModelsMock: vi.fn(),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))
vi.mock('@/client/aiGateway', () => ({
  aiGatewayApi: { listSystemModels: listSystemModelsMock, getTokenUsage: vi.fn(async () => ({ featureTokens: {}, monthlyCost: '0' })) },
}))
vi.mock('@/client/ai', () => ({ aiApi: {} }))
vi.mock('@/pages/ai/systemai/api', () => ({ listSystemAIConfigs: listSystemAIConfigsMock }))

import AIGatewayCard from '@/pages/ai/systemai/components/AIGatewayCard'

function renderCard() {
  return render(
    <AIGatewayCard useGateway onToggle={vi.fn()} selectedModel={undefined} onModelChange={vi.fn()} />,
  )
}

describe('AIGatewayCard own-key precedence hint', () => {
  it('warns that own keys take precedence over gateway models', async () => {
    listSystemAIConfigsMock.mockResolvedValue({
      items: [{ enabled: true, has_secret: true, provider_id: 'openai_compatible_x' }],
    })
    listSystemModelsMock.mockResolvedValue([])
    renderCard()
    expect(await screen.findByText(/优先使用自有 Key/)).toBeTruthy()
  })

  it('shows no hint when the user has no own keyed provider', async () => {
    listSystemAIConfigsMock.mockResolvedValue({
      items: [{ enabled: true, has_secret: false, provider_id: 'zhipu' }],
    })
    listSystemModelsMock.mockResolvedValue([])
    renderCard()
    await waitFor(() => expect(listSystemAIConfigsMock).toHaveBeenCalled())
    expect(screen.queryByText(/优先使用自有 Key/)).toBeNull()
  })
})
