import { lazy } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '@/stores/authStore';
import { ConnectProvider } from '@/providers/ConnectProvider';
import { SSEQueryBridge } from '@/bridge/SSEQueryBridge';
import { PageWrapper } from '@/components/common/PageWrapper';
import { PrivateRoute, PublicRoute, AdminRoute } from '@/components/auth/RouteGuards';
import MainLayout from '@/components/layout/MainLayout';
import AdminLayout from '@/components/layout/AdminLayout';
import AIAssistantLayout from '@/pages/ai/AIAssistantLayout';
import RequireAIConfig from '@/pages/ai/components/RequireAIConfig';

// ── Lazy page imports ──
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

// ── Route helpers ──
const wrap = (el: React.ReactNode) => <PageWrapper>{el}</PageWrapper>;

// ── Public routes ──
const publicRoutes = (
  <>
    <Route path="/login" element={<PublicRoute>{wrap(<Login />)}</PublicRoute>} />
    <Route path="/register" element={<PublicRoute>{wrap(<Register />)}</PublicRoute>} />
    <Route path="/forgot-password" element={<PublicRoute>{wrap(<ForgotPassword />)}</PublicRoute>} />
    <Route path="/terms" element={wrap(<TermsOfService />)} />
    <Route path="/privacy" element={wrap(<PrivacyPolicy />)} />
  </>
);

// ── Main app routes ──
const mainRoutes = (
  <Route path="/" element={<PrivateRoute><MainLayout /></PrivateRoute>}>
    <Route index element={wrap(<Dashboard />)} />
    <Route path="accounts/:id" element={wrap(<AccountDetail />)} />
    <Route path="accounts/bind" element={wrap(<BindAccount />)} />
    <Route path="profile" element={wrap(<ProfilePage />)} />
    <Route path="ai" element={<AIAssistantLayout />}>
      <Route index element={<Navigate to="/ai/debate" replace />} />
      <Route path="debate" element={<RequireAIConfig>{wrap(<DebatePage />)}</RequireAIConfig>} />
      <Route path="settings" element={wrap(<SystemAI />)} />
      <Route path="agents" element={wrap(<AISettings mode="agents" />)} />
      <Route path="gate" element={wrap(<GateProgressPage />)} />
    </Route>
    <Route path="strategy/templates" element={wrap(<StrategyTemplatePage />)} />
    <Route path="strategy/assets" element={wrap(<StrategyAssetPage />)} />
    <Route path="strategy/schedules" element={wrap(<StrategySchedulePage />)} />
    <Route path="strategy/schedules/:id/logs" element={wrap(<StrategyScheduleLogsPage />)} />
    <Route path="strategy/indicator-catalog" element={wrap(<IndicatorCatalogPage />)} />
    <Route path="logs" element={wrap(<LogManagement />)} />
  </Route>
);

// ── Admin routes ──
const adminRoutes = (
  <Route path="/admin" element={<AdminRoute><AdminLayout /></AdminRoute>}>
    <Route index element={wrap(<AdminDashboard />)} />
    <Route path="users" element={wrap(<UserManagement />)} />
    <Route path="accounts" element={wrap(<AccountManagement />)} />
    <Route path="trading" element={wrap(<TradingMonitor />)} />
    <Route path="logs" element={wrap(<OperationLogs />)} />
    <Route path="config" element={wrap(<SystemConfig />)} />
    <Route path="jurisdiction" element={wrap(<JurisdictionGate />)} />
    <Route path="sre" element={<SRELayout />}>
      <Route index element={<Navigate to="/admin/sre/killswitch" replace />} />
      <Route path="killswitch" element={wrap(<SREKillSwitch />)} />
      <Route path="breakers" element={wrap(<SREBreakers />)} />
      <Route path="canary" element={wrap(<SRECanary />)} />
    </Route>
  </Route>
);

// ── App content ──
export function AppRoutes() {
  const { _hasHydrated } = useAuthStore();
  if (!_hasHydrated) {
    return <div className="min-h-screen flex items-center justify-center"><Spin size="large" /></div>;
  }
  return (
    <ConnectProvider>
      <SSEQueryBridge />
      <Routes>
        {publicRoutes}
        {mainRoutes}
        {adminRoutes}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ConnectProvider>
  );
}
