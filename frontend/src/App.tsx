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
import { QueryProvider } from '@/providers/QueryProvider';
import i18n, { normalizeLanguage, type SupportedLanguage } from '@/i18n';
import { useEffect, useState, lazy } from 'react';
import { PageWrapper } from '@/components/common/PageWrapper';

import MainLayout from '@/components/layout/MainLayout';
import AdminLayout from '@/components/layout/AdminLayout';
import AIAssistantLayout from '@/pages/ai/AIAssistantLayout';
const Login = lazy(() => import('@/pages/auth/Login'));
const Register = lazy(() => import('@/pages/auth/Register'));
const ForgotPassword = lazy(() => import('@/pages/auth/ForgotPassword'));
const TermsOfService = lazy(() => import('@/pages/legal/TermsOfService'));
const PrivacyPolicy = lazy(() => import('@/pages/legal/PrivacyPolicy'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const AccountDetail = lazy(() => import('@/pages/accounts/AccountDetail'));
const AccountsList = lazy(() => import('@/pages/accounts/AccountsList'));
const BindAccount = lazy(() => import('@/pages/accounts/BindAccount'));
const Trading = lazy(() => import('@/pages/trading/Trading'));
const Market = lazy(() => import('@/pages/market/Market'));
const Marketplace = lazy(() => import('@/pages/marketplace/Marketplace'));
const Summary = lazy(() => import('@/pages/analytics/Summary'));
const DebatePage = lazy(() => import('@/pages/ai/debate/DebatePageV2'));
const AISettings = lazy(() => import('@/pages/ai/AISettings'));
const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));
const GateProgressPage = lazy(() => import('@/pages/ai/gate/GateProgressPage'));
import RequireAIConfig from '@/pages/ai/components/RequireAIConfig';
const StrategyTemplatePage = lazy(() => import('@/pages/strategy/StrategyTemplatePage'));
const StrategyExperimentPage = lazy(() => import('@/pages/strategy/StrategyExperimentPage'));
const MarketRegimePage = lazy(() => import('@/pages/strategy/MarketRegimePage'));
const StrategyAssetPage = lazy(() => import('@/pages/strategy/StrategyAssetPage'));
const StrategySchedulePage = lazy(() => import('@/pages/strategy/StrategySchedulePage'));
const StrategyScheduleLogsPage = lazy(() => import('@/pages/strategy/StrategyScheduleLogsPage'));
const LogManagement = lazy(() => import('@/pages/logs/LogManagement'));
const AdminDashboard = lazy(() => import('@/pages/admin/Dashboard'));
const UserManagement = lazy(() => import('@/pages/admin/UserManagement'));
const AccountManagement = lazy(() => import('@/pages/admin/AccountManagement'));
const TradingMonitor = lazy(() => import('@/pages/admin/TradingMonitor'));
const OperationLogs = lazy(() => import('@/pages/admin/OperationLogs'));
const SystemConfig = lazy(() => import('@/pages/admin/SystemConfig'));
const JurisdictionGate = lazy(() => import('@/pages/admin/JurisdictionGate'));
const SREKillSwitch = lazy(() => import('@/pages/admin/sre/KillSwitchPage'));
const SREBreakers = lazy(() => import('@/pages/admin/sre/BreakersPage'));
const SRECanary = lazy(() => import('@/pages/admin/sre/CanaryPage'));
const SRELayout = lazy(() => import('@/pages/admin/sre/SRELayout'));

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
  if (!isAuthenticated) {
    return <Navigate to="/login" replace />;
  }
  if (!user?.permissions?.includes('admin:view')) {
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
              <PageWrapper>
                <Login />
              </PageWrapper>
            </PublicRoute>
          }
        />
        <Route
          path="/register"
          element={
            <PublicRoute>
              <PageWrapper>
                <Register />
              </PageWrapper>
            </PublicRoute>
          }
        />
        <Route
          path="/forgot-password"
          element={
            <PublicRoute>
              <PageWrapper>
                <ForgotPassword />
              </PageWrapper>
            </PublicRoute>
          }
        />
        <Route
          path="/terms"
          element={
            <PageWrapper>
              <TermsOfService />
            </PageWrapper>
          }
        />
        <Route
          path="/privacy"
          element={
            <PageWrapper>
              <PrivacyPolicy />
            </PageWrapper>
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
              <PageWrapper>
                <Dashboard />
              </PageWrapper>
            }
          />
          <Route
            path="accounts"
            element={
              <PageWrapper>
                <AccountsList />
              </PageWrapper>
            }
          />
          <Route
            path="accounts/:id"
            element={
              <PageWrapper>
                <AccountDetail />
              </PageWrapper>
            }
          />
          <Route
            path="accounts/bind"
            element={
              <PageWrapper>
                <BindAccount />
              </PageWrapper>
            }
          />
          <Route
            path="trading"
            element={
              <PageWrapper>
                <Trading />
              </PageWrapper>
            }
          />
          <Route
            path="market"
            element={
              <PageWrapper>
                <Market />
              </PageWrapper>
            }
          />
          <Route
            path="marketplace"
            element={
              <PageWrapper>
                <Marketplace />
              </PageWrapper>
            }
          />
          <Route
            path="analytics"
            element={
              <PageWrapper>
                <Summary />
              </PageWrapper>
            }
          />
          <Route path="ai" element={<AIAssistantLayout />}>
            <Route index element={<Navigate to="/ai/debate" replace />} />
            <Route
              path="debate"
              element={
                <RequireAIConfig>
                  <PageWrapper>
                    <DebatePage />
                  </PageWrapper>
                </RequireAIConfig>
              }
            />
            <Route
              path="settings"
              element={
                <PageWrapper>
                  <SystemAI />
                </PageWrapper>
              }
            />
            <Route
              path="agents"
              element={
                <PageWrapper>
                  <AISettings mode="agents" />
                </PageWrapper>
              }
            />
            <Route
              path="gate"
              element={
                <PageWrapper>
                  <GateProgressPage />
                </PageWrapper>
              }
            />
          </Route>
          <Route
            path="strategy/templates"
            element={
              <PageWrapper>
                <StrategyTemplatePage />
              </PageWrapper>
            }
          />
          <Route
            path="strategy/experiments"
            element={
              <PageWrapper>
                <StrategyExperimentPage />
              </PageWrapper>
            }
          />
          <Route
            path="strategy/market-regime"
            element={
              <PageWrapper>
                <MarketRegimePage />
              </PageWrapper>
            }
          />
          <Route
            path="strategy/assets"
            element={
              <PageWrapper>
                <StrategyAssetPage />
              </PageWrapper>
            }
          />
          <Route
            path="strategy/schedules"
            element={
              <PageWrapper>
                <StrategySchedulePage />
              </PageWrapper>
            }
          />
          <Route
            path="strategy/schedules/:id/logs"
            element={
              <PageWrapper>
                <StrategyScheduleLogsPage />
              </PageWrapper>
            }
          />
          <Route
            path="logs"
            element={
              <PageWrapper>
                <LogManagement />
              </PageWrapper>
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
              <PageWrapper>
                <AdminDashboard />
              </PageWrapper>
            }
          />
          <Route
            path="users"
            element={
              <PageWrapper>
                <UserManagement />
              </PageWrapper>
            }
          />
          <Route
            path="accounts"
            element={
              <PageWrapper>
                <AccountManagement />
              </PageWrapper>
            }
          />
          <Route
            path="trading"
            element={
              <PageWrapper>
                <TradingMonitor />
              </PageWrapper>
            }
          />
          <Route
            path="logs"
            element={
              <PageWrapper>
                <OperationLogs />
              </PageWrapper>
            }
          />
          <Route
            path="config"
            element={
              <PageWrapper>
                <SystemConfig />
              </PageWrapper>
            }
          />
          <Route
            path="jurisdiction"
            element={
              <PageWrapper>
                <JurisdictionGate />
              </PageWrapper>
            }
          />
          <Route path="sre" element={<SRELayout />}>
            <Route index element={<Navigate to="/admin/sre/killswitch" replace />} />
            <Route path="killswitch" element={<PageWrapper><SREKillSwitch /></PageWrapper>} />
            <Route path="breakers" element={<PageWrapper><SREBreakers /></PageWrapper>} />
            <Route path="canary" element={<PageWrapper><SRECanary /></PageWrapper>} />
          </Route>
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
      <QueryProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </QueryProvider>
    </ConfigProvider>
  );
}
