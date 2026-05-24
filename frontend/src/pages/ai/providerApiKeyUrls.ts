// Maps provider type -> public URL where the user can create / manage API keys.
// Used by AISettings to render the "Go to provider to manage API key" link
// without hard-coding URLs inside the page component.
export const providerApiKeyUrls: Record<string, string> = {
  openai: 'https://platform.openai.com/api-keys',
  anthropic: 'https://console.anthropic.com/settings/keys',
  deepseek: 'https://platform.deepseek.com/api_keys',
  zhipu: 'https://open.bigmodel.cn/usercenter/apikeys',
  qwen: 'https://dashscope.console.aliyun.com/apiKey',
  moonshot: 'https://platform.moonshot.cn/console/api-keys',
  doubao: 'https://console.volcengine.com/ark/region:ark+cn-beijing/apiKey',
  siliconflow: 'https://cloud.siliconflow.cn/account/ak',
  openrouter: 'https://openrouter.ai/keys',
  mistral: 'https://console.mistral.ai/api-keys/',
  groq: 'https://console.groq.com/keys',
};

export function getProviderApiKeyUrl(provider: string): string {
  return providerApiKeyUrls[provider?.toLowerCase?.()] || '';
}
