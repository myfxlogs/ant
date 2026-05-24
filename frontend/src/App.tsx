import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import zhCN from 'antd/locale/zh_CN';
import zhTW from 'antd/locale/zh_TW';
import enUS from 'antd/locale/en_US';
import jaJP from 'antd/locale/ja_JP';
import viVN from 'antd/locale/vi_VN';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import 'dayjs/locale/zh-tw';
import 'dayjs/locale/ja';
import 'dayjs/locale/vi';
import { useAuthStore } from '@/stores/authStore';
import i18n, { normalizeLanguage, type SupportedLanguage } from '@/i18n';
import { useEffect, useState, Suspense, lazy } from 'react';

import MainLayout from '@/components/layout/MainLayout';
import AdminLayout from '@/components/layout/AdminLayout';
import AIAssistantLayout from '@/pages/ai/AIAssistantLayout';

const Login = lazy(() => import('@/pages/auth/Login'));
const Register = lazy(() => import('@/pages/auth/Register'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const DebatePage = lazy(() => import('@/pages/ai/debate/DebatePageV2'));
const AISettings = lazy(() => import('@/pages/ai/AISettings'));
const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));
const ResearchPage = lazy(() => import('@/pages/research/ResearchPage'));
const MarketplacePage = lazy(() => import('@/pages/marketplace/Marketplace'));
const RiskSettings = lazy(() => import('@/pages/risk/RiskSettings'));

const antLocaleMap: Record<SupportedLanguage, typeof zhCN> = {
  'zh-CN': zhCN, 'zh-TW': zhTW, en: enUS, ja: jaJP, vi: viVN,
};

function App() {
  const { user, fetchUser } = useAuthStore();
  const [locale, setLocale] = useState<SupportedLanguage>('zh-CN');

  useEffect(() => { fetchUser(); }, []);
  useEffect(() => {
    const lang = normalizeLanguage(i18n.language);
    setLocale(lang);
    dayjs.locale(lang === 'zh-CN' ? 'zh-cn' : lang === 'zh-TW' ? 'zh-tw' : lang);
  }, []);

  if (!user) {
    return (
      <ConfigProvider locale={antLocaleMap[locale]}>
        <Spin spinning={true}>
          <div style={{ minHeight: '100vh' }} />
        </Spin>
      </ConfigProvider>
    );
  }

  return (
    <ConfigProvider locale={antLocaleMap[locale]}>
      <BrowserRouter>
        <Suspense fallback={<Spin spinning><div style={{ minHeight: '100vh' }} /></Spin>}>
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/" element={<MainLayout />}>
              <Route index element={<Dashboard />} />
              <Route path="ai" element={<AIAssistantLayout />}>
                <Route path="debate" element={<DebatePage />} />
                <Route path="settings" element={<AISettings />} />
                <Route path="agents" element={<SystemAI />} />
              </Route>
              <Route path="marketplace" element={<MarketplacePage />} />
              <Route path="research" element={<ResearchPage />} />
              <Route path="risk" element={<RiskSettings />} />
            </Route>
            <Route path="/admin" element={<AdminLayout />}>
              <Route index element={<Dashboard />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </ConfigProvider>
  );
}

export default App;
