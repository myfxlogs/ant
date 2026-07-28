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

interface DiscoverPattern {
  key: string;
  lower?: string[];
  raw?: string[];
  exact?: string;
  compound?: (lower: string, msg: string) => boolean;
}

const DISCOVER_PATTERNS: DiscoverPattern[] = [
  { key: 'baseUrlRequired', raw: ['base_url'], exact: '__DISCOVER_BASE_URL_EMPTY__' },
  { key: 'baseUrlInvalid', raw: ['base url format invalid'] },
  { key: 'freeTierExhausted', lower: ['free-tier exhausted', 'freetieronly', 'free tier', 'free-tier only'] },
  {
    key: 'quotaForbidden403',
    compound: (lower) => lower.includes('status 403') && (lower.includes('quota') || lower.includes('exhaust') || lower.includes('allocation')),
  },
  { key: 'quotaOrRateLimit', lower: ['quota exhausted', '[resource_exhausted]', 'status 429', 'too many requests', 'rate limit'] },
  { key: 'providerRegionBlocked', lower: ['user location is not supported'] },
  { key: 'unauthorized', raw: ['unauthorized'] },
  { key: 'endpoint404', lower: ['endpoint not found', 'status 404'] },
  { key: 'timeout', raw: ['timeout'] },
  { key: 'unreachable', raw: ['unreachable'] },
  { key: 'invalidModelsResponse', raw: ['invalid /models'] },
  { key: 'noModelsReturned', raw: ['no models returned'] },
];

/** Map upstream / backend error text to a localized message (locale follows UI i18n). */
export function toFriendlyDiscoverMessage(msg: string, t: TFunction): string {
  const lower = msg.toLowerCase()
  for (const p of DISCOVER_PATTERNS) {
    if (p.exact && msg === p.exact) return t(`${DK}.${p.key}`)
    if (p.raw?.some(s => msg.includes(s))) return t(`${DK}.${p.key}`)
    if (p.lower?.some(s => lower.includes(s))) return t(`${DK}.${p.key}`)
    if (p.compound?.(lower, msg)) return t(`${DK}.${p.key}`)
  }
  const detail = (msg || '').trim()
  return detail ? t(`${DK}.genericDetail`, { detail }) : t(`${DK}.generic`)
}
