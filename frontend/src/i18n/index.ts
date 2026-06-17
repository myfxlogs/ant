import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

export const SUPPORTED_LANGUAGES = ['zh-cn'] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

export const LANGUAGE_STORAGE_KEY = 'anttrader_lang';

export function normalizeLanguage(_input?: string | null): SupportedLanguage {
  return 'zh-cn';
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

  const nav = navigator as { languages?: readonly string[]; language?: string };
  const navLang =
    (typeof navigator !== 'undefined' &&
      ((Array.isArray(nav.languages) && nav.languages[0]) || nav.language)) ||
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

// Polyfill: Intl.PluralRules.select() throws on BigInt values (protobuf int64).
// i18next may receive BigInt counts through interpolation, so we coerce to Number.
// https://tc39.es/ecma402/#sec-intl.pluralrules.prototype.select
if (typeof Intl !== 'undefined' && Intl.PluralRules) {
  const _select = Intl.PluralRules.prototype.select;
  Intl.PluralRules.prototype.select = function (count: number | bigint) {
    if (typeof count === 'bigint') count = Number(count);
    return _select.call(this, count as number);
  };
}

if (!i18n.isInitialized) {
  const initial = getInitialLanguage();

  i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      lng: initial,
      fallbackLng: 'zh-cn',
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
  (window as Window & { __ANTTRADER_I18N__?: typeof i18n }).__ANTTRADER_I18N__ = i18n;
}

export default i18n;
