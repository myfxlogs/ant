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

// ── Lazy page imports ──
const Login = lazy(() => import('@/pages/auth/Login'));
const Register = lazy(() => import('@/pages/auth/Register'));
const ForgotPassword = lazy(() => import('@/pages/auth/ForgotPassword'));
const TermsOfService = lazy(() => import('@/pages/legal/TermsOfService'));
const PrivacyPolicy = lazy(() => import('@/pages/legal/PrivacyPolicy'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const AccountDetail = lazy(() => import('@/pages/accounts/AccountDetail'));
const BindAccount = lazy(() => import('@/pages/accounts/BindAccount'));
const SystemAI = lazy(() => import('@/pages/ai/SystemAI'));
const StrategyTemplatePage = lazy(() => import('@/pages/strategy/StrategyTemplatePage'));
const StrategyAssetPage = lazy(() => import('@/pages/strategy/StrategyAssetPage'));
const StrategySchedulePage = lazy(() => import('@/pages/strategy/StrategySchedulePage'));
const StrategyScheduleLogsPage = lazy(() => import('@/pages/strategy/StrategyScheduleLogsPage'));
const IndicatorCatalogPage = lazy(() => import('@/pages/strategy/IndicatorCatalogPage'));
const StrategyWorkspacePage = lazy(() => import('@/pages/strategy/StrategyWorkspacePage'));
const MarketplacePage = lazy(() => import('@/pages/marketplace/Marketplace'));
const AssetAnalysisPage = lazy(() => import('@/pages/strategy/AssetAnalysis'));
const ProfilePage = lazy(() => import('@/pages/profile/ProfilePage'));
const LogManagement = lazy(() => import('@/pages/logs/LogManagement'));
const AutoTradingSettings = lazy(() => import('@/pages/auto-trading/AutoTradingSettings'));
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
    <Route path="strategy/templates" element={wrap(<StrategyTemplatePage />)} />
    <Route path="strategy/workspace" element={wrap(<StrategyWorkspacePage />)} />
    <Route path="strategy/assets" element={wrap(<StrategyAssetPage />)} />
    <Route path="strategy/schedules" element={wrap(<StrategySchedulePage />)} />
    <Route path="strategy/schedules/:id/logs" element={wrap(<StrategyScheduleLogsPage />)} />
    <Route path="strategy/indicator-catalog" element={wrap(<IndicatorCatalogPage />)} />
    <Route path="marketplace" element={wrap(<MarketplacePage />)} />
    <Route path="strategy/analysis" element={wrap(<AssetAnalysisPage />)} />
    <Route path="logs" element={wrap(<LogManagement />)} />
    <Route path="auto-trading" element={wrap(<AutoTradingSettings />)} />
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
