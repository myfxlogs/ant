import i18n from 'i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import { initReactI18next } from 'react-i18next';

import zhCN from './resources/zh-cn/index';
import zhTW from './resources/zh-tw/index';
import en from './resources/en/index';
import ja from './resources/ja/index';
import vi from './resources/vi/index';

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

export const resources = {
  'zh-cn': { translation: zhCN },
  'zh-tw': { translation: zhTW },
  'zh-hans': { translation: zhCN },
  'zh-hant': { translation: zhTW },
  en: { translation: en },
  ja: { translation: ja },
  vi: { translation: vi },
} as const;

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

export function setLanguage(lng: SupportedLanguage) {
  const normalized = normalizeLanguage(lng);
  i18n.changeLanguage(normalized);
  try {
    localStorage.setItem(LANGUAGE_STORAGE_KEY, normalized);
  } catch (_e) {
    // ignore
  }
}

if (!i18n.isInitialized) {
  const initial = getInitialLanguage();
  const normalized = normalizeLanguage(initial);

  i18n
    .use(LanguageDetector)
    .use(initReactI18next)
    .init({
      resources: resources as any,
      lng: normalized,
      fallbackLng: 'en',
      cleanCode: false,
      lowerCaseLng: true,
      load: 'currentOnly',
      initImmediate: false, // 同步初始化，避免首屏英文
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

  // 确保中/繁资源可用（若未注册）
  if (!i18n.hasResourceBundle('zh-cn', 'translation')) {
    i18n.addResourceBundle('zh-cn', 'translation', zhCN, true, true);
  }
  if (!i18n.hasResourceBundle('zh-tw', 'translation')) {
    i18n.addResourceBundle('zh-tw', 'translation', zhTW, true, true);
  }
}

if (typeof window !== 'undefined') {
  (window as any).__ANTTRADER_I18N__ = i18n;
}

export default i18n;
