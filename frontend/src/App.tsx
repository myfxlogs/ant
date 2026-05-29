import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { ConfigProvider, Spin } from 'antd';
import dayjs from 'dayjs';
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
const BindAccount = lazy(() => import('@/pages/accounts/BindAccount'));
const DebatePage = lazy(() => import('@/pages/ai/debate/DebatePageV2'));
const AISettings = lazy(() => import('@/pages/ai/AISettings'));
const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));
const GateProgressPage = lazy(() => import('@/pages/ai/gate/GateProgressPage'));
import RequireAIConfig from '@/pages/ai/components/RequireAIConfig';
const StrategyTemplatePage = lazy(() => import('@/pages/strategy/StrategyTemplatePage'));
const StrategyAssetPage = lazy(() => import('@/pages/strategy/StrategyAssetPage'));
const StrategySchedulePage = lazy(() => import('@/pages/strategy/StrategySchedulePage'));
const StrategyScheduleLogsPage = lazy(() => import('@/pages/strategy/StrategyScheduleLogsPage'));
const IndicatorCatalogPage = lazy(() => import('@/pages/strategy/IndicatorCatalogPage'));
const ProfilePage = lazy(() => import('@/pages/profile/ProfilePage'));
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
            path="profile"
            element={
              <PageWrapper>
                <ProfilePage />
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
            path="strategy/indicator-catalog"
            element={
              <PageWrapper>
                <IndicatorCatalogPage />
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

// Locale cache: dynamically loaded antd locales keyed by locale code (e.g. "zh_CN").
const antdLocaleCache: Record<string, unknown> = {};

// Statically-analyzable locale loaders — each import() path must be a literal so
// Vite can discover and bundle the chunks at build time. Template-literal dynamic
// imports (e.g. import(`dayjs/locale/${dl}`)) are NOT statically analyzable and
// will fail in production builds.

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

// dayjsLocale maps our SupportedLanguage to loader keys.
const dayjsLocaleMap: Record<string, string> = {
  'zh-cn': 'zh-cn',
  'zh-tw': 'zh-tw',
  ja: 'ja',
  vi: 'vi',
};

// antdLocaleKey maps our SupportedLanguage to loader keys.
const antdLocaleKeyMap: Record<string, string> = {
  'zh-cn': 'zh_CN',
  'zh-tw': 'zh_TW',
  ja: 'ja_JP',
  vi: 'vi_VN',
};

export default function App() {
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(i18n.language));
  const [antdLocale, setAntdLocale] = useState<unknown>(null);

  useEffect(() => {
    const handler = (lng: string) => setLang(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => {
      i18n.off('languageChanged', handler);
    };
  }, []);

  // Proactive token-lifecycle: schedules background refresh when the user is
  // active + the page is visible, and refreshes on tab visibility change.
  // The transport interceptor also calls ensureFreshToken() per request, so
  // even right after a hard reload the first authed RPC will not 401.
  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const { startTokenScheduler, ensureFreshToken } = await import('@/utils/tokenLifecycle');
      if (cancelled) return;
      // Boot-time preflight: the persisted token may already be expired.
      await ensureFreshToken();
      startTokenScheduler();
    })();
    return () => { cancelled = true; };
  }, []);

  useEffect(() => {
    const dl = dayjsLocaleMap[lang] || 'en';
    const ak = antdLocaleKeyMap[lang] || 'en_US';

    // Lazy-load dayjs locale (side-effect module registers the locale).
    if (dl !== 'en') {
      const dayjsLoader = dayjsLocaleLoaders[dl];
      if (dayjsLoader) {
        dayjsLoader().then(() => {
          dayjs.locale(dl);
        });
      }
    } else {
      dayjs.locale('en');
    }

    // Lazy-load antd locale with module-level cache.
    if (antdLocaleCache[ak]) {
      setAntdLocale(antdLocaleCache[ak]);
    } else {
      const antdLoader = antdLocaleLoaders[ak];
      if (antdLoader) {
        antdLoader().then((m) => {
          antdLocaleCache[ak] = m.default;
          setAntdLocale(m.default);
        });
      }
    }
  }, [lang]);

  return (
    <ConfigProvider locale={antdLocale || undefined}>
      <QueryProvider>
        <BrowserRouter>
          <AppContent />
        </BrowserRouter>
      </QueryProvider>
    </ConfigProvider>
  );
}
