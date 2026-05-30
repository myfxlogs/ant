import { useEffect, useState } from 'react';
import { ConfigProvider } from 'antd';
import dayjs from 'dayjs';
import i18n, { normalizeLanguage, type SupportedLanguage } from '@/i18n';

const antdLocaleCache: Record<string, unknown> = {};

const dayjsLocaleLoaders: Record<string, () => Promise<void>> = {
  'zh-cn': () => import('dayjs/locale/zh-cn'),
  'zh-tw': () => import('dayjs/locale/zh-tw'),
  ja: () => import('dayjs/locale/ja'),
  vi: () => import('dayjs/locale/vi'),
};

const antdLocaleLoaders: Record<string, () => Promise<{ default: unknown }>> = {
  zh_CN: () => import('antd/locale/zh_CN'),
  zh_TW: () => import('antd/locale/zh_TW'),
  ja_JP: () => import('antd/locale/ja_JP'),
  vi_VN: () => import('antd/locale/vi_VN'),
  en_US: () => import('antd/locale/en_US'),
};

const dayjsLocaleMap: Record<string, string> = { 'zh-cn': 'zh-cn', 'zh-tw': 'zh-tw', ja: 'ja', vi: 'vi' };
const antdLocaleKeyMap: Record<string, string> = { 'zh-cn': 'zh_CN', 'zh-tw': 'zh_TW', ja: 'ja_JP', vi: 'vi_VN' };

export function LocaleProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(i18n.language));
  const [antdLocale, setAntdLocale] = useState<unknown>(null);

  useEffect(() => {
    const handler = (lng: string) => setLang(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => { i18n.off('languageChanged', handler); };
  }, []);

  useEffect(() => {
    const dl = dayjsLocaleMap[lang] || 'en';
    const ak = antdLocaleKeyMap[lang] || 'en_US';
    if (dl !== 'en') {
      const loader = dayjsLocaleLoaders[dl];
      if (loader) loader().then(() => { dayjs.locale(dl); });
    } else {
      dayjs.locale('en');
    }
    if (antdLocaleCache[ak]) {
      setAntdLocale(antdLocaleCache[ak]);
    } else {
      const loader = antdLocaleLoaders[ak];
      if (loader) loader().then((m) => { antdLocaleCache[ak] = m.default; setAntdLocale(m.default); });
    }
  }, [lang]);

  return <ConfigProvider locale={antdLocale || undefined}>{children}</ConfigProvider>;
}
