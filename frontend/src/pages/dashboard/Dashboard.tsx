import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Col, Row, Spin, Statistic, Tag } from 'antd';
import {
  IconCurrencyDollar,
  IconUsers,
  IconChartLine,
  IconChartBar,
  IconChartPie,
  IconPlus,
  IconX,
  IconArrowUp,
  IconBuildingBank,
} from '@tabler/icons-react';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { useNavigate } from 'react-router-dom';
import { useAccount } from '@/hooks/useAccount';
import { useAuthStore } from '@/stores/authStore';
import { useTradingStore } from '@/stores/tradingStore';
import { useAccountStore } from '@/stores/accountStore';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next';

export default function Dashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { accounts, fetchAccounts } = useAccount();
  const { user } = useAuthStore();
  const accountInfoMap = useTradingStore((state) => state.accountInfoMap);
  const userSummary = useTradingStore((state) => state.userSummary);
  const [localLoading, setLocalLoading] = useState(false);

  const localConnectedCount = useMemo(() => {
    return (accounts || []).filter((a) => !a.isDisabled && a.status === 'connected').length;
  }, [accounts]);
  
  const accountInfoValues = useMemo(() => {
    const result: Record<string, { balance: number; equity: number; profit: number }> = {};
    for (const [key, value] of accountInfoMap.entries()) {
      result[key] = {
        balance: value.balance,
        equity: value.equity,
        profit: value.profit,
      };
    }
    return result;
  }, [accountInfoMap]);

  useEffect(() => {
    const loadData = async () => {
      const currentAccounts = useAccountStore.getState().accounts;
      const hasData = currentAccounts && currentAccounts.length > 0;
      
      if (!hasData) {
        setLocalLoading(true);
        await fetchAccounts();
        setLocalLoading(false);
      }
      // 有数据时，不调用 fetchAccounts，直接使用缓存
    };
    loadData();
  }, [fetchAccounts]); // 只在首次挂载时执行

  const stats = useMemo(() => {
    const backendConnected = (userSummary as any)?.connectedCount;
    const connectedCount =
      typeof backendConnected === 'number' && Number.isFinite(backendConnected)
        ? backendConnected
        : localConnectedCount;
    return {
      totalEquity: userSummary.totalEquity,
      totalBalance: userSummary.totalBalance,
      totalProfit: userSummary.totalProfit,
      accountCount: userSummary.accountCount,
	  connectedCount,
	  pnlToday: userSummary.pnlToday,
	  pnlWeek: userSummary.pnlWeek,
	  pnlMonth: userSummary.pnlMonth,
	  tradesToday: userSummary.tradesToday,
	  tradesWeek: userSummary.tradesWeek,
	  tradesMonth: userSummary.tradesMonth,
	  winRate: userSummary.winRate,
	  profitFactor: userSummary.profitFactor,
	  maxDrawdownPercent: userSummary.maxDrawdownPercent,
	  maxConsecutiveWins: userSummary.maxConsecutiveWins,
	  maxConsecutiveLosses: userSummary.maxConsecutiveLosses,
    };
  }, [userSummary, localConnectedCount]);

  const getStatusTag = (account: Account) => {
    if (account.isDisabled) return <Tag color="red">{t('dashboard.accountStatus.disabled')}</Tag>;
    if (account.status === 'connected') return <Tag color="green">{t('dashboard.accountStatus.connected')}</Tag>;
    if (account.status === 'connecting') return <Tag color="orange">{t('dashboard.accountStatus.connecting')}</Tag>;
    return <Tag color="default">{t('dashboard.accountStatus.disconnected')}</Tag>;
  };

  const getDisplayName = () => {
    if (user?.nickname) return user.nickname;
    if (user?.email) return user.email.split('@')[0];
    return t('topbar.user');
  };

  const quickActions = [
    { key: 'trading', icon: <IconChartLine size={24} stroke={1.5} color="#FFFFFF" />, label: t('dashboard.quickActions.trading'), path: '/trading', color: PRIMARY_GRADIENT },
    { key: 'market', icon: <IconChartBar size={24} stroke={1.5} color="#FFFFFF" />, label: t('dashboard.quickActions.market'), path: '/market', color: 'linear-gradient(135deg, #5A6B75 0%, #3D4A52 100%)' },
    { key: 'analytics', icon: <IconChartPie size={24} stroke={1.5} color="#FFFFFF" />, label: t('dashboard.quickActions.analytics'), path: '/analytics', color: 'linear-gradient(135deg, #9C27B0 0%, #7B1FA2 100%)' },
    { key: 'bind', icon: <IconPlus size={24} stroke={1.5} color="#FFFFFF" />, label: t('dashboard.quickActions.bindAccount'), path: '/accounts/bind', color: 'linear-gradient(135deg, #00A651 0%, #008C44 100%)' },
    { key: 'close', icon: <IconX size={24} stroke={1.5} color="#FFFFFF" />, label: t('dashboard.quickActions.closePosition'), path: '/trading', color: 'linear-gradient(135deg, #E53935 0%, #C62828 100%)' },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>
            {t('dashboard.welcome', { name: getDisplayName() })}
          </h1>
          <p className="mt-1" style={{ color: '#8A9AA5' }}>{t('dashboard.subtitle')}</p>
        </div>
        <Button 
          type="primary" 
          icon={<IconPlus size={16} stroke={1.5} />}
          onClick={() => navigate('/accounts/bind')}
          style={{ background: PRIMARY_GRADIENT, border: 'none' }}
        >
          {t('dashboard.bindAccount')}
        </Button>
      </div>
      
      <div 
        className="rounded-2xl p-6"
        style={{ 
          background: '#FFFFFF',
          boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
        }}
      >
        <h2 className="text-lg font-semibold mb-4" style={{ color: '#141D22' }}>{t('dashboard.accountOverview')}</h2>
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={6}>
            <div className="stat-card group cursor-pointer">
              <div className="flex items-center justify-between mb-3">
                <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: 'rgba(212, 175, 55, 0.1)' }}>
                  <IconCurrencyDollar size={20} stroke={1.5} color="#D4AF37" />
                </div>
                <IconArrowUp size={16} stroke={1.5} color="#00A651" />
              </div>
              <Statistic
                title={<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('dashboard.stats.totalEquity')}</span>}
                value={stats.totalEquity}
                precision={2}
                prefix={<span style={{ color: '#8A9AA5' }}>$</span>}
                styles={{ content: { color: '#141D22', fontSize: '24px', fontWeight: 600 } }}
              />
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card group cursor-pointer">
              <div className="flex items-center justify-between mb-3">
                <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: 'rgba(0, 166, 81, 0.1)' }}>
                  <IconChartLine size={20} stroke={1.5} color="#00A651" />
                </div>
              </div>
              <Statistic
                title={<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('dashboard.stats.connected')}</span>}
                value={stats.connectedCount}
                styles={{ content: { color: '#00A651', fontSize: '24px', fontWeight: 600 } }}
              />
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card group cursor-pointer">
              <div className="flex items-center justify-between mb-3">
                <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: 'rgba(90, 107, 117, 0.1)' }}>
                  <IconUsers size={20} stroke={1.5} color="#5A6B75" />
                </div>
              </div>
              <Statistic
                title={<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('dashboard.stats.accountCount')}</span>}
                value={stats.accountCount}
                styles={{ content: { color: '#141D22', fontSize: '24px', fontWeight: 600 } }}
              />
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card group cursor-pointer">
              <div className="flex items-center justify-between mb-3">
                <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: 'rgba(0, 166, 81, 0.1)' }}>
                  <IconChartLine size={20} stroke={1.5} color="#00A651" />
                </div>
                {stats.totalProfit >= 0 ? (
                  <IconArrowUp size={16} stroke={1.5} color="#00A651" />
                ) : (
                  <IconArrowUp size={16} stroke={1.5} color="#E53935" style={{ transform: 'rotate(180deg)' }} />
                )}
              </div>
              <Statistic
                title={<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('dashboard.stats.totalProfit')}</span>}
                value={stats.totalProfit}
                precision={2}
                prefix={<span style={{ color: '#8A9AA5' }}>$</span>}
                styles={{ 
                  content: { 
                    color: stats.totalProfit >= 0 ? '#00A651' : '#E53935',
                    fontSize: '24px',
                    fontWeight: 600
                  }
                }}
              />
            </div>
          </Col>
        </Row>
      </div>

      <Row gutter={[16, 16]}>
        <Col xs={24} lg={16}>
          <Card 
            title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('dashboard.accountList')}</span>}
            className="glass-card"
          >
            <Spin spinning={localLoading}>
              <div className="space-y-3">
                {(accounts || []).slice(0, 4).map((item) => {
                  const live = accountInfoValues[item.id];
                  const rowBalance = live?.balance ?? item.balance;
                  const rowEquity = live?.equity ?? item.equity;
                  // Prefer Connect/stream accountInfo (same order as balance/equity). REST list often sends profit: 0;
                  // `item.profit ?? live` would keep 0 and never show streamed floating P/L.
                  const rowFloating = live?.profit ?? item.profit ?? 0;
                  return (
                  <div 
                    key={item.id}
                    onClick={() => navigate(`/accounts/${item.id}`)}
                    className="flex items-center justify-between p-4 rounded-xl cursor-pointer transition-all"
                    style={{ 
                      background: '#F5F7F9', 
                      border: '1px solid rgba(0, 0, 0, 0.05)',
                    }}
                    onMouseEnter={(e) => {
                      e.currentTarget.style.background = '#E8ECF0';
                      e.currentTarget.style.borderColor = 'rgba(212, 175, 55, 0.2)';
                    }}
                    onMouseLeave={(e) => {
                      e.currentTarget.style.background = '#F5F7F9';
                      e.currentTarget.style.borderColor = 'rgba(0, 0, 0, 0.05)';
                    }}
                  >
                    <div className="flex items-center gap-4">
                      <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: item.mtType === 'MT4' ? 'rgba(33, 150, 243, 0.1)' : 'rgba(212, 175, 55, 0.1)' }}>
                        <IconBuildingBank size={20} stroke={1.5} color={item.mtType === 'MT4' ? '#2196F3' : '#D4AF37'} />
                      </div>
                      <div>
                        <div className="flex items-center gap-2">
                          <span style={{ color: '#141D22', fontWeight: 500 }}>{item.login}</span>
                          <Tag color={item.mtType === 'MT4' ? 'blue' : 'gold'} className="!text-xs">
                            {item.mtType}
                          </Tag>
                          {getStatusTag(item)}
                        </div>
                        <div className="text-sm mt-1" style={{ color: '#8A9AA5' }}>{item.brokerCompany}</div>
                      </div>
                    </div>
                    <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 text-right">
                      <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-start">
                        <span className="text-xs sm:hidden" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.balance')}</span>
                        <div className="hidden sm:block text-xs mb-1" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.balance')}</div>
                        <div className="font-medium" style={{ color: '#141D22' }}>
                          {rowBalance != null && Number.isFinite(rowBalance) ? rowBalance.toFixed(2) : '0.00'}
                        </div>
                      </div>
                      <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-start">
                        <span className="text-xs sm:hidden" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.equity')}</span>
                        <div className="hidden sm:block text-xs mb-1" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.equity')}</div>
                        <div className="font-semibold" style={{ color: '#141D22' }}>
                          {rowEquity != null && Number.isFinite(rowEquity) ? rowEquity.toFixed(2) : '0.00'}
                        </div>
                      </div>
                      <div className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-start">
                        <span className="text-xs sm:hidden" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.floating')}</span>
                        <div className="hidden sm:block text-xs mb-1" style={{ color: '#8A9AA5' }}>{t('dashboard.fields.floating')}</div>
                        <div className="font-medium" style={{ color: rowFloating >= 0 ? '#00A651' : '#E53935' }}>
                          {rowFloating.toFixed(2)}
                        </div>
                      </div>
                      <div className="text-xs hidden sm:block" style={{ color: '#8A9AA5' }}>
                        {item.currency || 'USD'}
                      </div>
                    </div>
                  </div>
                );
                })}
                {(!accounts || accounts.length === 0) && (
                  <div className="text-center py-8" style={{ color: '#8A9AA5' }}>
                    <IconBuildingBank size={40} stroke={1.5} />
                    <p className="mt-3">{t('dashboard.noAccounts')}</p>
                  </div>
                )}
              </div>
            </Spin>
          </Card>
        </Col>

        <Col xs={24} lg={8}>
          <Card 
            title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('dashboard.quickActions.title')}</span>}
            className="glass-card h-full"
          >
            <div className="grid grid-cols-2 gap-3">
              {quickActions.map((action) => (
                <div
                  key={action.key}
                  onClick={() => navigate(action.path)}
                  className="flex flex-col items-center justify-center p-4 rounded-xl cursor-pointer transition-all"
                  style={{ 
                    background: '#F5F7F9', 
                    border: '1px solid rgba(0, 0, 0, 0.05)',
                  }}
                  onMouseEnter={(e) => {
                    e.currentTarget.style.background = '#E8ECF0';
                    e.currentTarget.style.borderColor = 'rgba(212, 175, 55, 0.2)';
                  }}
                  onMouseLeave={(e) => {
                    e.currentTarget.style.background = '#F5F7F9';
                    e.currentTarget.style.borderColor = 'rgba(0, 0, 0, 0.05)';
                  }}
                >
                  <div className="w-12 h-12 rounded-xl flex items-center justify-center mb-3" style={{ background: action.color }}>
                    {action.icon}
                  </div>
                  <span style={{ color: '#141D22', fontWeight: 500 }}>{action.label}</span>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  );
}
