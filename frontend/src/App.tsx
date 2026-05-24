import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { ConfigProvider, Result, Button } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import enUS from 'antd/locale/en_US';
import jaJP from 'antd/locale/ja_JP';
import viVN from 'antd/locale/vi_VN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/zh-tw';
import 'dayjs/locale/ja';
import 'dayjs/locale/vi';
import i18n, { normalizeLanguage, type SupportedLanguage } from '@/i18n';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import MainLayout from '@/components/layout/MainLayout';

const localeMap: Record<SupportedLanguage, typeof zhCN> = {
  'zh-CN': zhCN, en: enUS, ja: jaJP, vi: viVN,
};

function PlaceholderPage({ title, description }: { title: string; description: string }) {
  return (
    <Result
      status="info"
      title={title}
      subTitle={description}
      extra={<Button type="primary" onClick={() => window.location.reload()}>Refresh</Button>}
    />
  );
}

function App() {
  const { t } = useTranslation();
  const [locale, setLocale] = useState<SupportedLanguage>('zh-CN');

  useEffect(() => {
    const lang = normalizeLanguage(i18n.language);
    setLocale(lang);
    const dl = lang === 'zh-CN' ? 'zh-cn' : lang === 'zh-TW' ? 'zh-tw' : lang;
    dayjs.locale(dl);
  }, []);

  return (
    <ConfigProvider locale={localeMap[locale] || enUS}>
      <BrowserRouter>
        <Routes>
          <Route path="/" element={<MainLayout />}>
            <Route index element={
              <PlaceholderPage
                title={t('common.loading', 'Ant v2')}
                description="Market data pipeline operational. Frontend pages being rebuilt on ConnectRPC ant/v1/."
              />
            } />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;
