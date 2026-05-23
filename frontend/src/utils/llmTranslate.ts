import i18n, { normalizeLanguage } from '@/i18n';
import { aiApi } from '@/client/ai';
import { listSystemAIConfigs } from '@/pages/ai/systemai/api';

const CACHE_PREFIX = 'anttrader_llm_translate_cache_v1:';

function redactSensitive(text: string): string {
  let t = String(text || '');

  // Authorization headers / bearer tokens
  t = t.replace(/(authorization\s*:\s*)(bearer\s+)[^\s]+/gi, '$1$2***');
  t = t.replace(/(bearer\s+)[A-Za-z0-9._-]+/gi, '$1***');

  // API keys / tokens (heuristics)
  t = t.replace(/(api[_-]?key\s*[:=]\s*)[^\s"']+/gi, '$1***');
  t = t.replace(/(token\s*[:=]\s*)[^\s"']+/gi, '$1***');

  // Passwords
  t = t.replace(/(password\s*[:=]\s*)[^\s"']+/gi, '$1***');

  // Emails
  t = t.replace(/[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}/gi, '***@***');

  // IP addresses
  t = t.replace(/\b\d{1,3}(?:\.\d{1,3}){3}\b/g, '***.***.***.***');

  // URLs / domains
  t = t.replace(/https?:\/\/[^\s"']+/gi, '***');

  // UUIDs
  t = t.replace(/\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b/gi, '***');

  return t;
}

async function sha256Hex(input: string): Promise<string> {
  try {
    const enc = new TextEncoder();
    const buf = enc.encode(input);
    const digest = await crypto.subtle.digest('SHA-256', buf);
    const bytes = Array.from(new Uint8Array(digest));
    return bytes.map((b) => b.toString(16).padStart(2, '0')).join('');
  } catch {
    // Fallback: not cryptographically strong, but ok for cache key.
    return String(input.length) + ':' + btoa(unescape(encodeURIComponent(input))).slice(0, 64);
  }
}

async function getAIConfigOk(): Promise<boolean> {
  try {
    const { items } = await listSystemAIConfigs();
    return items.some(
      (it) => it.enabled && it.has_secret && (it.default_model || '').trim() !== '',
    );
  } catch {
    return false;
  }
}

export interface TranslateWithLLMParams {
  text: string;
  targetLang?: string;
  purpose?: 'error_detail';
}

export async function translateTextWithLLM(params: TranslateWithLLMParams): Promise<string> {
  const raw = String(params.text || '').trim();
  if (!raw) return '';

  const ok = await getAIConfigOk();
  if (!ok) {
    throw new Error('errors.ai.not_configured');
  }

  const target = normalizeLanguage(params.targetLang || i18n.language);
  const redacted = redactSensitive(raw);

  const cacheKeyHash = await sha256Hex([target, params.purpose || 'error_detail', redacted].join('\n'));
  const cacheKey = `${CACHE_PREFIX}${cacheKeyHash}`;

  try {
    const cached = localStorage.getItem(cacheKey);
    if (cached) return cached;
  } catch {
    // ignore
  }

  const langName =
    target === 'zh-cn'
      ? i18n.t('language.simplifiedChinese')
      : target === 'zh-tw'
        ? i18n.t('language.traditionalChinese')
        : target === 'ja'
          ? i18n.t('language.japanese')
          : target === 'vi'
            ? i18n.t('language.vietnamese')
            : i18n.t('language.english');

  const prompt = [
    `You are a professional translator. Translate the following text into ${langName}.`,
    'Rules:',
    '- Translate only. Do not add explanations.',
    '- Keep code blocks, JSON keys, stack traces, and identifiers as-is when appropriate.',
    '- Preserve numbers and timestamps exactly.',
    '',
    'Text:',
    '```',
    redacted,
    '```',
  ].join('\n');

  const res = await aiApi.chat({ message: prompt, context: '' });
  const out = String(res?.message || '').trim();
  if (!out) {
    throw new Error('errors.translate_failed');
  }

  try {
    localStorage.setItem(cacheKey, out);
  } catch {
    // ignore
  }

  return out;
}

export function extractErrorDetail(err: unknown): string {
  const e: any = err as any;
  const candidates = [e?.rawMessage, e?.details, e?.cause?.message, e?.message, e?.toString?.()];
  for (const c of candidates) {
    const s = String(c || '').trim();
    if (!s) continue;
    // Avoid showing messageKey as "detail".
    if (s.startsWith('errors.')) continue;
    return s;
  }
  try {
    const json = JSON.stringify(err);
    if (json && json !== '{}' && !json.includes('errors.')) return json;
  } catch {
    // ignore
  }
  return '';
}
