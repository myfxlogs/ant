import { lazy } from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { Spin } from 'antd';
import { useAuthStore } from '@/stores/authStore';
import { StreamProvider } from '@/providers/StreamProvider';
import { PageWrapper } from '@/components/common/PageWrapper';
import { PrivateRoute, PublicRoute, AdminRoute } from '@/components/auth/RouteGuards';
import MainLayout from '@/components/layout/MainLayout';
import AdminLayout from '@/components/layout/AdminLayout';

// ── Lazy page imports ──
const Login = lazy(() => import('@/pages/auth/Login'));
const Register = lazy(() => import('@/pages/auth/Register'));
const ForgotPassword = lazy(() => import('@/pages/auth/ForgotPassword'));
const ResetPassword = lazy(() => import('@/pages/auth/ResetPassword'));
const Dashboard = lazy(() => import('@/pages/dashboard/Dashboard'));
const LandingPage = lazy(() => import('@/pages/landing/LandingPage'));
const BrokersPage = lazy(() => import('@/pages/landing/BrokersPage'));
const AccountDetail = lazy(() => import('@/pages/accounts/AccountDetail'));
const BindAccount = lazy(() => import('@/pages/accounts/BindAccount'));
const AccountReport = lazy(() => import('@/pages/accounts/AccountReport'));
const StrategyScheduleLogsPage = lazy(() => import('@/pages/strategy/StrategyScheduleLogsPage'));
const StrategyWorkspacePage = lazy(() => import('@/pages/strategy/StrategyWorkspacePage'));
const LiveStrategyPage = lazy(() => import('@/pages/strategy/LiveStrategyPage'));
const MarketplacePage = lazy(() => import('@/pages/marketplace/MarketplacePage'));
const ProfilePage = lazy(() => import('@/pages/profile/ProfilePage'));
const WalletPage = lazy(() => import('@/pages/wallet/WalletPage'));
const SubscriptionPage = lazy(() => import('@/pages/subscription/SubscriptionPage'));
const LogManagement = lazy(() => import('@/pages/logs/LogManagement'));
const AutoTradingSettings = lazy(() => import('@/pages/auto-trading/AutoTradingSettings'));
const MarketToolsPage = lazy(() => import('@/pages/strategy/MarketToolsPage'));
const AlgoDashboard = lazy(() => import('@/pages/trading/AlgoDashboard'));
const AnalyticsSummary = lazy(() => import('@/pages/analytics/Summary'));
const AdminDashboard = lazy(() => import('@/pages/admin/Dashboard'));
const UserManagement = lazy(() => import('@/pages/admin/UserManagement'));
const WalletManagement = lazy(() => import('@/pages/admin/WalletManagement'));
const AccountManagement = lazy(() => import('@/pages/admin/AccountManagement'));
const TradingMonitor = lazy(() => import('@/pages/admin/TradingMonitor'));
const OperationLogs = lazy(() => import('@/pages/admin/OperationLogs'));
const SystemConfig = lazy(() => import('@/pages/admin/SystemConfig'));
const JurisdictionGate = lazy(() => import('@/pages/admin/JurisdictionGate'));
const SREKillSwitch = lazy(() => import('@/pages/admin/sre/KillSwitchPage'));
const StrategyManagement = lazy(() => import('@/pages/admin/StrategyManagement'));
const SREBreakers = lazy(() => import('@/pages/admin/sre/BreakersPage'));
const ShareManagement = lazy(() => import('@/pages/admin/ShareManagement'));
const SRECanary = lazy(() => import('@/pages/admin/sre/CanaryPage'));
const AIGatewayManagement = lazy(() => import('@/pages/admin/AIGatewayManagement'));
const AdminSettingsPage = lazy(() => import('@/pages/admin/AdminSettingsPage'));
const AutoGenTaskReview = lazy(() => import('@/pages/admin/AutoGenTaskReview'));
const MarketplaceManagement = lazy(() => import('@/pages/admin/MarketplaceManagement'));
const RefundManagement = lazy(() => import('@/pages/admin/RefundManagement'));
const MarketplaceAnalyticsPage = lazy(() => import('@/pages/admin/MarketplaceAnalytics'));
const CouponManagement = lazy(() => import('@/pages/admin/CouponManagement'));
const BillingManagement = lazy(() => import('@/pages/admin/BillingManagement'));
const DepositManagement = lazy(() => import('@/pages/admin/DepositManagement'));
const MonitoringPage = lazy(() => import('@/pages/admin/MonitoringPage'));
const SRELayout = lazy(() => import('@/pages/admin/sre/SRELayout'));
const SharePerformancePage = lazy(() => import('@/pages/share/SharePerformancePage'));
const StrategySharePage = lazy(() => import('@/pages/marketplace/StrategySharePage'));

// ── Route helpers ──
const wrap = (el: React.ReactNode) => <PageWrapper>{el}</PageWrapper>;

const VerifyEmail = lazy(() => import('@/pages/auth/VerifyEmail'));

// ── Public routes ──
const publicRoutes = (
  <>
    <Route path="/login" element={<PublicRoute>{wrap(<Login />)}</PublicRoute>} />
    <Route path="/register" element={<PublicRoute>{wrap(<Register />)}</PublicRoute>} />
    <Route path="/verify-email" element={<PublicRoute>{wrap(<VerifyEmail />)}</PublicRoute>} />
    <Route path="/forgot-password" element={<PublicRoute>{wrap(<ForgotPassword />)}</PublicRoute>} />
    <Route path="/reset-password" element={<PublicRoute>{wrap(<ResetPassword />)}</PublicRoute>} />
  </>
);

// ── Main app routes ──
const mainRoutes = (
  <Route path="/" element={<PrivateRoute><MainLayout /></PrivateRoute>}>
    <Route index element={wrap(<Dashboard />)} />
    <Route path="accounts/:id" element={wrap(<AccountDetail />)} />
    <Route path="accounts/:id/report" element={wrap(<AccountReport />)} />
    <Route path="accounts/bind" element={wrap(<BindAccount />)} />
    <Route path="profile" element={wrap(<ProfilePage />)} />
    <Route path="wallet" element={wrap(<WalletPage />)} />
    <Route path="subscription" element={wrap(<SubscriptionPage />)} />
    <Route path="strategy/templates" element={<Navigate to="/strategy/workspace" replace />} />
    <Route path="strategy/schedules" element={<Navigate to="/strategy/live" replace />} />
    <Route path="strategy/schedules/:id/logs" element={wrap(<StrategyScheduleLogsPage />)} />
    <Route path="strategy/library" element={<Navigate to="/strategy/workspace" replace />} />
    <Route path="strategy/workspace" element={wrap(<StrategyWorkspacePage />)} />
    <Route path="strategy/live" element={wrap(<LiveStrategyPage />)} />
    <Route path="strategy/indicator-catalog" element={<Navigate to="/strategy/workspace" replace />} />
    <Route path="marketplace" element={wrap(<MarketplacePage />)} />
    <Route path="strategy/experiments" element={<Navigate to="/strategy/workspace" replace />} />
    <Route path="strategy/market-tools" element={wrap(<MarketToolsPage />)} />
    <Route path="strategy/analysis" element={<Navigate to="/strategy/market-tools?tab=symbol" replace />} />
    <Route path="strategy/market-regime" element={<Navigate to="/strategy/market-tools?tab=regime" replace />} />
    <Route path="strategy/memory" element={<Navigate to="/strategy/workspace" replace />} />
    <Route path="logs" element={wrap(<LogManagement />)} />
    <Route path="auto-trading" element={wrap(<AutoTradingSettings />)} />
    <Route path="trading/algos" element={wrap(<AlgoDashboard />)} />
    <Route path="analytics" element={wrap(<AnalyticsSummary />)} />
  </Route>
);

// ── Admin routes ──
const adminRoutes = (
  <Route path="/admin" element={<AdminRoute><AdminLayout /></AdminRoute>}>
    <Route index element={wrap(<AdminDashboard />)} />
    <Route path="users" element={wrap(<UserManagement />)} />
    <Route path="wallet" element={wrap(<WalletManagement />)} />
    <Route path="billing" element={wrap(<BillingManagement />)} />
    <Route path="deposits" element={wrap(<DepositManagement />)} />
    <Route path="accounts" element={wrap(<AccountManagement />)} />
    <Route path="trading" element={wrap(<TradingMonitor />)} />
    <Route path="logs" element={wrap(<OperationLogs />)} />
    <Route path="config" element={wrap(<SystemConfig />)} />
    <Route path="jurisdiction" element={wrap(<JurisdictionGate />)} />
    <Route path="strategies" element={wrap(<StrategyManagement />)} />
    <Route path="shares" element={wrap(<ShareManagement />)} />
    <Route path="ai-gateway" element={wrap(<AIGatewayManagement />)} />
    <Route path="monitoring" element={wrap(<MonitoringPage />)} />
    <Route path="agent-settings" element={wrap(<AdminSettingsPage />)} />
    <Route path="autogen-tasks" element={wrap(<AutoGenTaskReview />)} />
    <Route path="marketplace" element={wrap(<MarketplaceManagement />)} />
    <Route path="refunds" element={wrap(<RefundManagement />)} />
    <Route path="analytics" element={wrap(<MarketplaceAnalyticsPage />)} />
    <Route path="coupons" element={wrap(<CouponManagement />)} />
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
  const { _hasHydrated, isAuthenticated } = useAuthStore();
  if (!_hasHydrated) {
    return <div className="min-h-screen flex items-center justify-center"><Spin size="large" /></div>;
  }
  return (
    <Routes>
      {/* Public share page — standalone, no SSE, no auth */}
      <Route path="/share/:token" element={<SharePerformancePage />} />
      {/* Public landing page — unauthenticated only; authenticated falls through to * */}
      {!isAuthenticated && <Route path="/" element={<LandingPage />} />}
      {/* Public marketplace — unauthenticated only; authenticated uses mainRoutes version with layout */}
      {!isAuthenticated && <Route path="/marketplace" element={wrap(<MarketplacePage />)} />}
      {/* Public brokers page — SEO landing page, always accessible */}
      <Route path="/brokers" element={<BrokersPage />} />
      {/* Everything else inside StreamProvider */}
      <Route path="*" element={
        <StreamProvider>
          <Routes>
            {publicRoutes}
            {mainRoutes}
            {adminRoutes}
            {/* Strategy share landing page — must be after mainRoutes so known paths (workspace, live, etc.) match first */}
            <Route path="/strategy/:strategyId" element={<StrategySharePage />} />
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </StreamProvider>
      } />
    </Routes>
  );
}
