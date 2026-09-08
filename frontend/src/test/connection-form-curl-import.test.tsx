import { describe, it, expect, vi } from 'vitest'
import { render, fireEvent, screen, waitFor } from '@testing-library/react'

// antd TextArea's rc-resize-observer needs a constructor-compatible mock
// (setup.ts vi.fn arrow mocks are not constructible).
class ResizeObserverMock {
  observe() {}
  unobserve() {}
  disconnect() {}
  takeRecords() { return [] }
}
globalThis.ResizeObserver = ResizeObserverMock as unknown as typeof ResizeObserver

// FIX-2026-09-08-CURL-IMPORT 对抗证明。
//
// ConnectionForm 的「从厂商 curl 示例导入」：粘贴 curl → 后端解析 →
// 表单回填 base_url / models / default_model / API Key。
//
// mutation: 还原 ConnectionForm.tsx（无导入框）→ 本测试 RED。

const { parseProviderCurlMock } = vi.hoisted(() => ({ parseProviderCurlMock: vi.fn() }))
vi.mock('@/pages/ai/systemai/api', () => ({ parseProviderCurl: parseProviderCurlMock }))
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (k: string, o?: unknown) => {
      if (typeof o === 'string') return o
      if (o && typeof o === 'object' && 'defaultValue' in (o as object)) return String((o as { defaultValue: unknown }).defaultValue)
      return k
    },
  }),
}))

import { ConnectionFormSection } from '@/pages/ai/systemai/components/ConnectionForm'
import type { AIConfig } from '@/pages/ai/systemai/model'

function renderForm(over: Partial<Parameters<typeof ConnectionFormSection>[0]> = {}) {
  const draft: AIConfig = {
    provider_id: 'openai_compatible_x',
    name: '',
    base_url: '',
    organization: '',
    models: [],
    default_model: '',
    temperature: 0,
    timeout_seconds: 0,
    max_tokens: 0,
    purposes: [],
    primary_for: [],
    enabled: true,
    has_secret: false,
    updated_at: '',
  }
  const props = {
    draft,
    providerLabel: vi.fn((id: string) => id),
    isCustomProvider: vi.fn(() => true),
    urlHttps: true,
    urlOk: true,
    secretInput: '',
    onSecretInputChange: vi.fn(),
    onDraftChange: vi.fn(),
    onClearSecret: vi.fn(),
    savingSecret: false,
    discovering: false,
    discoveredModels: [],
    ...over,
  }
  render(<ConnectionFormSection {...props} />)
  return props
}

describe('ConnectionForm curl import', () => {
  it('prefills base_url/models/api key from a parsed curl example', async () => {
    parseProviderCurlMock.mockResolvedValue({
      base_url: 'https://token.sensenova.cn/v1',
      api_key: '',
      default_model: 'kimi-k3',
      models: ['kimi-k3'],
      name_hint: 'sensenova',
      warnings: ['示例中的 API Key 是占位符——请在下方 API Key 输入框粘贴你的真实 Key'],
    })
    const props = renderForm()

    const textarea = await screen.findByPlaceholderText(/curl https:\/\/api\.example\.com/)
    fireEvent.change(textarea, { target: { value: 'curl https://token.sensenova.cn/v1/chat/completions ...' } })
    fireEvent.click(screen.getByRole('button', { name: /导入并回填/ }))

    await waitFor(() => expect(parseProviderCurlMock).toHaveBeenCalledWith(expect.stringContaining('sensenova'), false))
    await waitFor(() => expect(props.onDraftChange).toHaveBeenCalled())
    expect(props.onDraftChange).toHaveBeenCalledWith(expect.objectContaining({
      base_url: 'https://token.sensenova.cn/v1',
      models: ['kimi-k3'],
      default_model: 'kimi-k3',
      name: 'sensenova',
    }))
    // placeholder key is NOT auto-filled into the secret input
    expect(props.onSecretInputChange).not.toHaveBeenCalledWith(expect.stringContaining('{'))
    // parse-time warning is surfaced inline for review
    expect(await screen.findByText(/占位符/)).toBeTruthy()
  })

  it('surfaces parse errors inline instead of crashing', async () => {
    parseProviderCurlMock.mockRejectedValue(new Error('未能识别 URL：请粘贴完整的 curl 命令（含厂商文档中的请求地址）'))
    renderForm()

    const textarea = await screen.findByPlaceholderText(/curl https:\/\/api\.example\.com/)
    fireEvent.change(textarea, { target: { value: 'echo hello' } })
    fireEvent.click(screen.getByRole('button', { name: /导入并回填/ }))

    expect(await screen.findByText(/未能识别 URL/)).toBeTruthy()
  })

  it('passes has_secret so backend mutes key warnings for already-configured providers', async () => {
    parseProviderCurlMock.mockResolvedValue({ base_url: 'https://h.cn/v1', api_key: '', default_model: 'm1', models: ['m1'], name_hint: '', warnings: [] })
    const draftWithKey = { has_secret: true }
    const props = renderForm({ draft: { ...draftWithKey } as AIConfig })

    const textarea = await screen.findByPlaceholderText(/curl https:\/\/api\.example\.com/)
    fireEvent.change(textarea, { target: { value: 'curl https://h.cn/v1/chat/completions ...' } })
    fireEvent.click(screen.getByRole('button', { name: /导入并回填/ }))

    await waitFor(() => expect(parseProviderCurlMock).toHaveBeenCalledWith(expect.any(String), true))
    expect(props.onDraftChange).toHaveBeenCalled()
  })
})
