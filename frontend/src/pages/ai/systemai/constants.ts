import type { TFunction } from 'i18next'

export const PROVIDER_LINKS: Record<string, string> = {
  openai: 'https://platform.openai.com/api-keys',
  openai_compatible: 'https://platform.openai.com/docs/api-reference/introduction',
  anthropic: 'https://console.anthropic.com/settings/keys',
  deepseek: 'https://platform.deepseek.com/api_keys',
  moonshot: 'https://platform.moonshot.cn/console/api-keys',
  qwen: 'https://bailian.console.aliyun.com/?apiKey=1',
  zhipu: 'https://open.bigmodel.cn/usercenter/apikeys',
}

export const ALL_PURPOSES = ['chat', 'embedding', 'summarizer', 'reasoning']

export const OFFICIAL_PROVIDER_BASE_URLS: Record<string, string> = {
  openai: 'https://api.openai.com/v1',
  anthropic: 'https://api.anthropic.com/v1',
  deepseek: 'https://api.deepseek.com/v1',
  moonshot: 'https://api.moonshot.cn/v1',
  qwen: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
  zhipu: 'https://open.bigmodel.cn/api/paas/v4',
}

const DK = 'ai.settings.discoverErrors'

/** Map upstream / backend error text to a localized message (locale follows UI i18n). */
export function toFriendlyDiscoverMessage(msg: string, t: TFunction): string {
  const lower = msg.toLowerCase()
  if (msg === '__DISCOVER_BASE_URL_EMPTY__' || msg.includes('base_url')) return t(`${DK}.baseUrlRequired`)
  if (msg.includes('base url format invalid')) return t(`${DK}.baseUrlInvalid`)
  if (lower.includes('free-tier exhausted') || lower.includes('freetieronly') || lower.includes('free tier') || lower.includes('free-tier only')) {
    return t(`${DK}.freeTierExhausted`)
  }
  if (
    lower.includes('quota exhausted') ||
    lower.includes('[resource_exhausted]') ||
    lower.includes('status 429') ||
    lower.includes('too many requests') ||
    lower.includes('rate limit')
  ) {
    return t(`${DK}.quotaOrRateLimit`)
  }
  if (lower.includes('status 403') && (lower.includes('quota') || lower.includes('exhaust') || lower.includes('allocation'))) {
    return t(`${DK}.quotaForbidden403`)
  }
  if (msg.includes('unauthorized')) return t(`${DK}.unauthorized`)
  if (msg.includes('endpoint')) return t(`${DK}.endpoint404`)
  if (msg.includes('timeout')) return t(`${DK}.timeout`)
  if (msg.includes('unreachable')) return t(`${DK}.unreachable`)
  if (msg.includes('invalid /models')) return t(`${DK}.invalidModelsResponse`)
  if (msg.includes('no models returned')) return t(`${DK}.noModelsReturned`)
  if (lower.includes('user location is not supported')) return t(`${DK}.providerRegionBlocked`)
  const detail = (msg || '').trim()
  return detail ? t(`${DK}.genericDetail`, { detail }) : t(`${DK}.generic`)
}
