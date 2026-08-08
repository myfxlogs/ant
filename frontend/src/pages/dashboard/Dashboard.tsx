import { useContext, useEffect, useState } from 'react';
import { Button, Col, Row } from 'antd';
import { PlusOutlined, WifiOutlined, DisconnectOutlined } from '@ant-design/icons';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useNavigate } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { useAuthStore } from '@/stores/authStore';
import { useUserSummaryQuery } from '@/queries/useUserSummaryQuery';
import { ConnectContext } from '@/providers/connectContext';
import { useTranslation } from 'react-i18next'
import { ACCOUNT_OVERVIEW_KEY, BIND_ACCOUNT_KEY, DEFAULT_NAME_KEY, QUICK_ACTIONS_TITLE_KEY, STREAM_LIVE_KEY, STREAM_OFFLINE_KEY, SUBTITLE_KEY, WELCOME_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';

;
import DashboardStatCards from './DashboardStatCards';
import DashboardAccountList from './DashboardAccountList';
import { createQuickActions } from './quickActions';
import Seo from '@/components/common/Seo';
import WelcomeModal from '@/components/onboarding/WelcomeModal';
import GlassCard from '@/components/common/GlassCard';

export default function Dashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { fetchAccounts, accounts } = useAccount();
  const { user } = useAuthStore();
  const [localLoading, setLocalLoading] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const connectCtx = useContext(ConnectContext);
  const streamConnected = connectCtx?.isConnected ?? false;

  // Aggregated totals come from backend SubscribeUserSummary SSE (pre-computed).
  const { data: summary } = useUserSummaryQuery();

  // Initial or forced refresh — populates TQ cache via SSE bridge.
  useEffect(() => {
    let cancelled = false;
    setLocalLoading(true);
    setLoadError(null);
    fetchAccounts()
      .catch((e) => { if (!cancelled) setLoadError(String(e?.message || e)); })
      .finally(() => { if (!cancelled) setLocalLoading(false); });
    return () => { cancelled = true; };
  }, [fetchAccounts]);

  const accts = accounts ?? [];
  const quickActions = createQuickActions(t);

  const stats = {
    totalBalance: Number(summary?.totalBalance) || 0,
    totalEquity: Number(summary?.totalEquity) || 0,
    totalProfit: Number(summary?.totalProfit) || 0,
    connectedCount: summary?.connectedCount ?? 0,
    accountCount: summary?.accountCount ?? accts.length,
  };

  const getDisplayName = () => user?.email?.split('@')[0] || user?.username || t(DEFAULT_NAME_KEY);

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 'var(--space-lg)' }}>
      <Seo title="Dashboard" noindex />
      <WelcomeModal hasAccounts={accts.length > 0} hasStrategies={false} />

      {/* Header — generous whitespace */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: 'var(--color-text)' }}>
            {t(WELCOME_KEY, { name: getDisplayName() })}
          </h1>
          <p className="mt-1" style={{ color: 'var(--color-text-muted)' }}>
            {t(SUBTITLE_KEY)}
            <span className="ml-3 inline-flex items-center gap-1" style={{ fontSize: 12, color: streamConnected ? '#00A651' : '#E53935' }}>
              {streamConnected ? <WifiOutlined size={14} /> : <DisconnectOutlined size={14} />}
              {streamConnected ? t(STREAM_LIVE_KEY) : t(STREAM_OFFLINE_KEY)}
            </span>
          </p>
        </div>
        <Button type="primary" icon={<PlusOutlined size={16} />} onClick={() => navigate('/accounts/bind')}
          style={{ background: PRIMARY_GRADIENT, border: 'none', borderRadius: 'var(--radius-sm)' }}>{t(BIND_ACCOUNT_KEY)}</Button>
      </div>

      {/* Account Overview — liquid glass card */}
      <GlassCard hover={false} style={{ padding: 'var(--space-lg)' }}>
        <h2 className="text-lg font-semibold mb-4" style={{ color: 'var(--color-text)' }}>{t(ACCOUNT_OVERVIEW_KEY)}</h2>
        <DashboardStatCards stats={stats} loading={localLoading} />
      </GlassCard>

      {/* Main grid — 12-col grid alignment, generous gutter */}
      <Row gutter={[24, 24]}>
        <Col xs={24} lg={16}>
          <DashboardAccountList accounts={accts} loading={localLoading} error={loadError} onRetry={fetchAccounts} />
        </Col>
        <Col xs={24} lg={8}>
          <GlassCard hover={false} style={{ padding: 'var(--space-md)', height: '100%' }}>
            <h3 className="text-base font-semibold mb-4" style={{ color: 'var(--color-text)' }}>{t(QUICK_ACTIONS_TITLE_KEY)}</h3>
            <div className="grid grid-cols-2 gap-3">
              {quickActions.map((action) => (
                <GlassCard
                  key={action.key}
                  onClick={() => navigate(action.path)}
                  style={{ padding: 'var(--space-md)' }}
                >
                  <div className="flex flex-col items-center justify-center">
                    <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-3" style={{ background: action.color }}>{action.icon}</div>
                    <span style={{ color: 'var(--color-text)', fontWeight: 500, fontSize: '13px' }}>{action.label}</span>
                  </div>
                </GlassCard>
              ))}
            </div>
          </GlassCard>
        </Col>
      </Row>
    </div>
  );
}
