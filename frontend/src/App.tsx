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
import { ConnectProvider } from '@/providers/ConnectProvider';
import i18n, { normalizeLanguage, type SupportedLanguage } from '@/i18n';
import { useEffect, useState, Suspense, lazy } from 'react';

import MainLayout from '@/components/layout/MainLayout';
import AdminLayout from '@/components/layout/AdminLayout';
import AIAssistantLayout from '@/pages/ai/AIAssistantLayout';
const Login = lazy(() => import('@/pages/auth/Login'));
const Register = lazy(() => import('@/pages/auth/Register'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const AccountDetail = lazy(() => import('@/pages/accounts/AccountDetail'));
const BindAccount = lazy(() => import('@/pages/accounts/BindAccount'));
const Summary = lazy(() => import('@/pages/analytics/Summary'));
const DebatePage = lazy(() => import('@/pages/ai/debate/DebatePageV2'));
const AISettings = lazy(() => import('@/pages/ai/AISettings'));
const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));
import RequireAIConfig from '@/pages/ai/components/RequireAIConfig';
const StrategyTemplatePage = lazy(() => import('@/pages/strategy/StrategyTemplatePage'));
const StrategyExperimentPage = lazy(() => import('@/pages/strategy/StrategyExperimentPage'));
const MarketRegimePage = lazy(() => import('@/pages/strategy/MarketRegimePage'));
const StrategyAssetPage = lazy(() => import('@/pages/strategy/StrategyAssetPage'));
const StrategySchedulePage = lazy(() => import('@/pages/strategy/StrategySchedulePage'));
const StrategyScheduleLogsPage = lazy(() => import('@/pages/strategy/StrategyScheduleLogsPage'));
const LogManagement = lazy(() => import('@/pages/logs/LogManagement'));
const ResearchPage = lazy(() => import('@/pages/research/ResearchPage'));
const AdminDashboard = lazy(() => import('@/pages/admin/Dashboard'));
const UserManagement = lazy(() => import('@/pages/admin/UserManagement'));
const AccountManagement = lazy(() => import('@/pages/admin/AccountManagement'));
const TradingMonitor = lazy(() => import('@/pages/admin/TradingMonitor'));
const OperationLogs = lazy(() => import('@/pages/admin/OperationLogs'));
const SystemConfig = lazy(() => import('@/pages/admin/SystemConfig'));

function PrivateRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();
  return isAuthenticated ? <>{children}</> : <Navigate to="/login" replace />;
}

function PublicRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated } = useAuthStore();
  return isAuthenticated ? <Navigate to="/" replace /> : <>{children}</>;
}

function AdminRoute({ children }: { children: React.ReactNode }) {
  const { isAuthenticated, user } = useAuthStore();
  const adminRoles = ['super_admin', 'operation', 'customer_service', 'audit'];
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (!user?.role || !adminRoles.includes(user.role)) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}

function AppContent() {
  const { _hasHydrated } = useAuthStore();

  if (!_hasHydrated) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <ConnectProvider>
      <Routes>
        <Route
          path="/login"
          element={
            <PublicRoute>
              <Suspense fallback={<div className="min-h-screen flex items-center justify-center"><Spin size="large" /></div>}>
                <Login />
              </Suspense>
            </PublicRoute>
          }
        />
        <Route
          path="/register"
          element={
            <PublicRoute>
              <Suspense fallback={<div className="min-h-screen flex items-center justify-center"><Spin size="large" /></div>}>
                <Register />
              </Suspense>
            </PublicRoute>
          }
        />
        <Route
          path="/"
          element={
            <PrivateRoute>
              <MainLayout />
            </PrivateRoute>
          }
        >
          <Route
            index
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <Dashboard />
              </Suspense>
            }
          />
          <Route
            path="accounts/:id"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <AccountDetail />
              </Suspense>
            }
          />
          <Route
            path="accounts/bind"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <BindAccount />
              </Suspense>
            }
          />
          <Route
            path="analytics"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <Summary />
              </Suspense>
            }
          />
          <Route path="ai" element={<AIAssistantLayout />}>
            <Route index element={<Navigate to="/ai/debate" replace />} />
            <Route
              path="debate"
              element={
                <RequireAIConfig>
                  <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                    <DebatePage />
                  </Suspense>
                </RequireAIConfig>
              }
            />
            <Route
              path="settings"
              element={
                <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                  <SystemAI />
                </Suspense>
              }
            />
            <Route
              path="agents"
              element={
                <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                  <AISettings mode="agents" />
                </Suspense>
              }
            />
          </Route>
          <Route
            path="strategy/templates"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <StrategyTemplatePage />
              </Suspense>
            }
          />
          <Route
            path="strategy/experiments"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <StrategyExperimentPage />
              </Suspense>
            }
          />
          <Route
            path="strategy/market-regime"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <MarketRegimePage />
              </Suspense>
            }
          />
          <Route
            path="strategy/assets"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <StrategyAssetPage />
              </Suspense>
            }
          />
          <Route
            path="strategy/schedules"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <StrategySchedulePage />
              </Suspense>
            }
          />
          <Route
            path="strategy/schedules/:id/logs"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <StrategyScheduleLogsPage />
              </Suspense>
            }
          />
          <Route
            path="logs"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <LogManagement />
              </Suspense>
            }
          />
          <Route
            path="research"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <ResearchPage />
              </Suspense>
            }
          />
        </Route>
        <Route
          path="/admin"
          element={
            <AdminRoute>
              <AdminLayout />
            </AdminRoute>
          }
        >
          <Route
            index
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <AdminDashboard />
              </Suspense>
            }
          />
          <Route
            path="users"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <UserManagement />
              </Suspense>
            }
          />
          <Route
            path="accounts"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <AccountManagement />
              </Suspense>
            }
          />
          <Route
            path="trading"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <TradingMonitor />
              </Suspense>
            }
          />
          <Route
            path="logs"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <OperationLogs />
              </Suspense>
            }
          />
          <Route
            path="config"
            element={
              <Suspense fallback={<div className="flex items-center justify-center py-10"><Spin size="large" /></div>}>
                <SystemConfig />
              </Suspense>
            }
          />
        </Route>
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ConnectProvider>
  );
}

export default function App() {
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(i18n.language));

  useEffect(() => {
    const handler = (lng: string) => setLang(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => {
      i18n.off('languageChanged', handler);
    };
  }, []);

  useEffect(() => {
    const dayjsLocale =
      lang === 'zh-cn'
        ? 'zh-cn'
        : lang === 'zh-tw'
          ? 'zh-tw'
          : lang === 'ja'
            ? 'ja'
            : lang === 'vi'
              ? 'vi'
              : 'en';
    dayjs.locale(dayjsLocale);
  }, [lang]);

  const antdLocale =
    lang === 'zh-cn'
      ? zhCN
      : lang === 'zh-tw'
        ? zhTW
        : lang === 'ja'
          ? jaJP
          : lang === 'vi'
            ? viVN
            : enUS;

  return (
    <ConfigProvider locale={antdLocale}>
      <BrowserRouter>
        <AppContent />
      </BrowserRouter>
    </ConfigProvider>
  );
}
