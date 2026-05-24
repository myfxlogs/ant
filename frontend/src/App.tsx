import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Result, Button, Spin } from 'antd';
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
import { useEffect, useState, lazy, Suspense } from 'react';

import MainLayout from '@/components/layout/MainLayout';
import LoginPage from '@/pages/auth/LoginPage';

const MarketplacePage = lazy(() => import('@/pages/marketplace/Marketplace'));
const AdminDashboard = lazy(() => import('@/pages/admin/AdminDashboard'));

const localeMap: Record<SupportedLanguage, typeof zhCN> = {
  'zh-CN': zhCN, en: enUS, ja: jaJP, vi: viVN,
};

const Loading = () => (
  <div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}>
    <Spin size="large" />
  </div>
);

function HomePage() {
  return (
    <Result
      status="info"
      title="Ant v2"
      subTitle="Market data pipeline operational — MT4/MT5 quotes verified."
      extra={
        <Button type="primary" href="/marketplace">
          Browse Marketplace
        </Button>
      }
    />
  );
}

function App() {
  const [locale, setLocale] = useState<SupportedLanguage>('zh-CN');
  const [auth, setAuth] = useState<{ token: string; userId: string } | null>(() => {
    const token = localStorage.getItem('auth_token');
    const userId = localStorage.getItem('userId');
    return token && userId ? { token, userId } : null;
  });

  useEffect(() => {
    const lang = normalizeLanguage(i18n.language);
    setLocale(lang);
    const dl = lang === 'zh-CN' ? 'zh-cn' : lang === 'zh-TW' ? 'zh-tw' : lang;
    dayjs.locale(dl);
  }, []);

  function handleLogin(token: string, userId: string) {
    setAuth({ token, userId });
  }

  if (!auth) {
    return (
      <ConfigProvider locale={localeMap[locale] || enUS}>
        <BrowserRouter>
          <Routes>
            <Route path="*" element={<LoginPage onLogin={handleLogin} />} />
          </Routes>
        </BrowserRouter>
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider locale={localeMap[locale] || enUS}>
      <BrowserRouter>
        <Suspense fallback={<Loading />}>
          <Routes>
            <Route path="/" element={<MainLayout />}>
              <Route index element={<HomePage />} />
              <Route path="marketplace" element={<MarketplacePage />} />
              <Route path="admin" element={<AdminDashboard />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;
