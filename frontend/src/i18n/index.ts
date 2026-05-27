import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

export const SUPPORTED_LANGUAGES = ['zh-cn', 'zh-tw', 'en', 'ja', 'vi'] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

export const LANGUAGE_STORAGE_KEY = 'anttrader_lang';

export function normalizeLanguage(input?: string | null): SupportedLanguage {
  const raw = String(input || '').trim();
  if (!raw) return 'zh-cn';

  const lower = raw.toLowerCase();

  if (lower === 'zh-cn' || lower === 'zh_cn' || lower.startsWith('zh-hans')) return 'zh-cn';
  if (lower === 'zh-tw' || lower === 'zh_tw' || lower.startsWith('zh-hant') || lower === 'zh-hk' || lower === 'zh-mo') return 'zh-tw';

  if (lower.startsWith('zh')) return 'zh-cn';
  if (lower.startsWith('ja')) return 'ja';
  if (lower.startsWith('vi')) return 'vi';
  if (lower.startsWith('en')) return 'en';

  return 'en';
}

const resourceCache = new Map<string, Record<string, unknown>>();

async function loadBundle(lng: string): Promise<Record<string, unknown>> {
  if (resourceCache.has(lng)) return resourceCache.get(lng)!;
  const mod = await import(`./resources/${lng}/index.ts`);
  resourceCache.set(lng, mod.default);
  return mod.default;
}

export function getInitialLanguage(): SupportedLanguage {
  try {
    const stored = localStorage.getItem(LANGUAGE_STORAGE_KEY);
    if (stored) return normalizeLanguage(stored);
  } catch (_e) {
    // ignore
  }

  const navLang =
    (typeof navigator !== 'undefined' &&
      ((Array.isArray((navigator as any).languages) && (navigator as any).languages[0]) || (navigator as any).language)) ||
    '';

  return normalizeLanguage(navLang);
}

export async function setLanguage(lng: SupportedLanguage) {
  const normalized = normalizeLanguage(lng);
  const bundle = await loadBundle(normalized);
  if (!i18n.hasResourceBundle(normalized, 'translation')) {
    i18n.addResourceBundle(normalized, 'translation', bundle, true, true);
  }
  i18n.changeLanguage(normalized);
  try {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, normalized);
  } catch (_e) {
    // ignore
  }
}

if (!i18n.isInitialized) {
  const initial = getInitialLanguage();

  i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      lng: initial,
      fallbackLng: 'en',
      cleanCode: false,
      lowerCaseLng: true,
      load: 'currentOnly',
      initImmediate: false,
      interpolation: {
        escapeValue: false,
      },
      detection: {
        order: ['localStorage', 'navigator'],
        lookupLocalStorage: LANGUAGE_STORAGE_KEY,
        caches: [],
      },
      react: {
        useSuspense: false,
      },
    });

  loadBundle(initial).then((bundle) => {
    i18n.addResourceBundle(initial, 'translation', bundle, true, true);
    i18n.changeLanguage(initial);
  });
}

if (typeof window !== 'undefined') {
  (window as any).__ANTTRADER_I18N__ = i18n;
}

export default i18n;
