import { useContext, useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Row } from 'antd';
import {
  BarChartOutlined, PieChartOutlined, PlusOutlined,
  WifiOutlined, DisconnectOutlined,
} from '@ant-design/icons';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useNavigate } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { useAuthStore } from '@/stores/authStore';
import { useQuery } from '@tanstack/react-query';
import { queryKeys } from '@/queries/queryKeys';
import { ConnectContext } from '@/providers/connectContext';
import { useTranslation } from 'react-i18next';
import type { Account } from '@/types/account';
import type { UserSummaryData } from '@/bridge/bridgeUserSummary';
import DashboardStatCards from './DashboardStatCards';
import DashboardAccountList from './DashboardAccountList';

const quickActions = [
  { key: 'bind', label: 'dashboard.quickActions.bindAccount', path: '/accounts/bind', icon: <PlusOutlined size={22} />, color: 'rgba(212,175,55,0.1)' },
  { key: 'templates', label: 'dashboard.quickActions.templates', path: '/strategy/templates', icon: <PieChartOutlined size={22} />, color: 'rgba(0,166,81,0.1)' },
  { key: 'analytics', label: 'dashboard.quickActions.analytics', path: '/analytics', icon: <BarChartOutlined size={22} />, color: 'rgba(33,150,243,0.1)' },
  { key: 'logs', label: 'dashboard.quickActions.logs', path: '/logs', icon: <BarChartOutlined size={22} />, color: 'rgba(156,39,176,0.1)' },
];

export default function Dashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { accounts, fetchAccounts } = useAccount();
  const { user } = useAuthStore();
  const [localLoading, setLocalLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const connectCtx = useContext(ConnectContext);
  const streamConnected = connectCtx?.isConnected ?? false;

  // Aggregated totals come from backend SubscribeUserSummary SSE (pre-computed).
  const { data: summary } = useQuery<UserSummaryData>({
    queryKey: queryKeys.userSummary.all,
    staleTime: 1000,
  });

  // Live per-account financials from SSE via TanStack Query.
  const { data: liveAccounts } = useQuery<Account[]>({
    queryKey: queryKeys.accounts.list(),
    staleTime: Infinity,
  });

  // Merge Zustand metadata with TanStack Query live financials — display only, no math.
  const merged = useMemo(() => {
    const liveMap = new Map<string, Account>();
    if (liveAccounts) for (const a of liveAccounts) liveMap.set(a.id, a);
    return (accounts || []).map((a) => {
      const live = liveMap.get(a.id);
      if (!live) return a;
      return { ...a, balance: live.balance ?? a.balance, equity: live.equity ?? a.equity, profit: live.profit ?? a.profit, status: live.status || a.status };
    });
  }, [accounts, liveAccounts]);

  useEffect(() => {
    let cancelled = false;
    setLocalLoading(true);
    setLoadError(null);
    fetchAccounts()
      .catch((e) => { if (!cancelled) setLoadError(String(e?.message || e)); })
      .finally(() => { if (!cancelled) setLocalLoading(false); });
    return () => { cancelled = true; };
  }, [fetchAccounts]);

  const stats = {
    totalEquity: summary?.totalEquity ?? 0,
    totalProfit: summary?.totalProfit ?? 0,
    connectedCount: summary?.connectedCount ?? 0,
    accountCount: summary?.accountCount ?? (accounts || []).length,
  };

  const getDisplayName = () => user?.email?.split('@')[0] || user?.username || t('dashboard.defaultName');

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>
            {t('dashboard.welcome', { name: getDisplayName() })}
          </h1>
          <p className="mt-1" style={{ color: '#8A9AA5' }}>
            {t('dashboard.subtitle')}
            <span className="ml-3 inline-flex items-center gap-1" style={{ fontSize: 12, color: streamConnected ? '#00A651' : '#E53935' }}>
              {streamConnected ? <WifiOutlined size={14} /> : <DisconnectOutlined size={14} />}
              {streamConnected ? t('dashboard.streamLive') : t('dashboard.streamOffline')}
            </span>
          </p>
        </div>
        <Button type="primary" icon={<PlusOutlined size={16} />} onClick={() => navigate('/accounts/bind')}
          style={{ background: PRIMARY_GRADIENT, border: 'none' }}>{t('dashboard.bindAccount')}</Button>
      </div>

      <div className="rounded-2xl p-6" style={{ background: '#FFFFFF', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
        <h2 className="text-lg font-semibold mb-4" style={{ color: '#141D22' }}>{t('dashboard.accountOverview')}</h2>
        <DashboardStatCards stats={stats} loading={localLoading} />
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <DashboardAccountList accounts={merged} loading={localLoading} error={loadError} onRetry={() => fetchAccounts()} />
        </Col>
        <Col xs={24} lg={8}>
          <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('dashboard.quickActions.title')}</span>} className="glass-card h-full">
            <div className="grid grid-cols-2 gap-3">
              {quickActions.map((action) => (
                <div key={action.key} onClick={() => navigate(action.path)}
                  className="flex flex-col items-center justify-center p-4 rounded-xl cursor-pointer transition-all"
                  style={{ background: '#F5F7F9', border: '1px solid rgba(0,0,0,0.05)' }}
                  onMouseEnter={(e) => { e.currentTarget.style.background = '#E8ECF0'; e.currentTarget.style.borderColor = 'rgba(212,175,55,0.2)'; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = '#F5F7F9'; e.currentTarget.style.borderColor = 'rgba(0,0,0,0.05)'; }}>
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-3" style={{ background: action.color }}>{action.icon}</div>
                  <span style={{ color: '#141D22', fontWeight: 500, fontSize: '13px' }}>{t(action.label)}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
