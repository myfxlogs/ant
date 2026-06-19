import { useContext, useEffect, useState } from 'react';
import { Button, Card, Col, Row } from 'antd';
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
    totalBalance: summary?.totalBalance ?? 0,
    totalEquity: summary?.totalEquity ?? 0,
    totalProfit: summary?.totalProfit ?? 0,
    connectedCount: summary?.connectedCount ?? 0,
    accountCount: summary?.accountCount ?? accts.length,
  };

  const getDisplayName = () => user?.email?.split('@')[0] || user?.username || t(DEFAULT_NAME_KEY);

  return (
    <div className="space-y-6">
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
          style={{ background: PRIMARY_GRADIENT, border: 'none' }}>{t(BIND_ACCOUNT_KEY)}</Button>
      </div>

      <div className="rounded-2xl p-6" style={{ background: 'var(--color-bg-card)', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
        <h2 className="text-lg font-semibold mb-4" style={{ color: 'var(--color-text)' }}>{t(ACCOUNT_OVERVIEW_KEY)}</h2>
        <DashboardStatCards stats={stats} loading={localLoading} />
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <DashboardAccountList accounts={accts} loading={localLoading} error={loadError} onRetry={fetchAccounts} />
        </Col>
        <Col xs={24} lg={8}>
          <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(QUICK_ACTIONS_TITLE_KEY)}</span>} className="glass-card h-full">
            <div className="grid grid-cols-2 gap-3">
              {quickActions.map((action) => (
                <div key={action.key} onClick={() => navigate(action.path)}
                  className="flex flex-col items-center justify-center p-4 rounded-xl cursor-pointer transition-all"
                  style={{ background: 'var(--color-bg-secondary)', border: '1px solid rgba(0,0,0,0.05)' }}
                  onMouseEnter={(e) => { e.currentTarget.style.background = '#E8ECF0'; e.currentTarget.style.borderColor = 'rgba(212,175,55,0.2)'; }}
                  onMouseLeave={(e) => { e.currentTarget.style.background = '#F5F7F9'; e.currentTarget.style.borderColor = 'rgba(0,0,0,0.05)'; }}>
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-3" style={{ background: action.color }}>{action.icon}</div>
                  <span style={{ color: 'var(--color-text)', fontWeight: 500, fontSize: '13px' }}>{action.label}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
