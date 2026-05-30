import { useMemo } from 'react';
import { Tag, Button, Spin, Dropdown, Modal, Input } from 'antd';
import type { MenuProps } from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, PauseCircleOutlined,
  CaretRightOutlined, MoreOutlined, WalletOutlined, LineChartOutlined,
  RiseOutlined, FallOutlined, DollarOutlined, PercentageOutlined,
  WarningOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import AccountTradeTabs from './components/AccountTradeTabs';
import AccountAnalyticsSection from './components/AccountAnalyticsSection';
import { InfoCard, SmallInfoCard } from './components/AccountDetail.shared';
import { useAccountDetailData } from './AccountDetail/useAccountDetailData';

export default function AccountDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const {
    currentAccount, isStreamLoading, financials,
    positions, pendingOrders,
    analyticsLoading, analyticsError,
    equityChartData, profitByMonthData, symbolDistributionData,
    dailyPnLData, hourlyData, tradeStats, riskMetrics,
    monthlyAnalysisYears, monthlyAnalysisData,
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    chartType, setChartType, chartPeriod, setChartPeriod,
    connecting, disabling,
    handleConnect, handleToggleStatus, handleRefresh, handleRetry,
    deleteModalOpen, setDeleteModalOpen,
    deletePassword, setDeletePassword,
    deleting, handleDelete,
    togglePending,
  } = useAccountDetailData(id);

  const { balance, equity, margin, freeMargin, marginLevel, profit, profitPercent, credit } = financials;

  const formatCurrency = (value: number) => {
    const isNegative = value < 0;
    return `${isNegative ? '-' : ''}${Math.abs(value).toFixed(2)} ${currentAccount?.currency || 'USD'}`;
  };

  const statusConfig = useMemo(() => {
    if (!currentAccount) return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('common.unknown') };
    if (currentAccount.isDisabled) return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('accounts.detail.status.disabled') };
    switch (currentAccount.status) {
      case 'connected': return { color: '#00A651', bg: 'rgba(0, 166, 81, 0.1)', text: t('accounts.detail.status.connected') };
      case 'connecting': return { color: '#FF9800', bg: 'rgba(255, 152, 0, 0.1)', text: t('accounts.detail.status.connecting') };
      case 'disconnected': return { color: '#E53935', bg: 'rgba(229, 57, 53, 0.1)', text: t('accounts.detail.status.disconnected') };
      case 'error': return { color: '#E53935', bg: 'rgba(229, 57, 53, 0.1)', text: t('accounts.detail.status.error') };
      default: return { color: '#8A9AA5', bg: 'rgba(138, 154, 165, 0.1)', text: t('common.unknown') };
    }
  }, [currentAccount, t]);

  const menuItems: MenuProps['items'] = useMemo(() => [
    {
      key: 'toggle',
      label: currentAccount?.isDisabled
        ? t('accounts.detail.actions.enableAccount')
        : t('accounts.detail.actions.disableAccount'),
      icon: togglePending ? <Spin size="small" />
        : currentAccount?.isDisabled ? <CaretRightOutlined /> : <PauseCircleOutlined />,
      onClick: handleToggleStatus,
      disabled: disabling,
    },
    {
      key: 'delete',
      label: t('accounts.detail.actions.deleteAccount'),
      icon: <DeleteOutlined style={{ color: '#E53935' }} />,
      onClick: () => setDeleteModalOpen(true),
      danger: true,
    },
  ], [currentAccount?.isDisabled, togglePending, disabling, handleToggleStatus, t]);

  if (!currentAccount) {
    return <div className="p-4 flex justify-center items-center h-64"><Spin size="large" /></div>;
  }

  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <div className="max-w-7xl mx-auto p-4">
        {/* Header + control bar */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/')} style={{ color: '#8A9AA5' }} />
            <div>
              <div className="flex items-center gap-3">
                <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{currentAccount.login}</h1>
                <Tag color={currentAccount.mtType === 'MT4' ? 'blue' : 'purple'}>{currentAccount.mtType}</Tag>
                {currentAccount.accountType && (
                  <Tag style={{ borderRadius: '6px', background: currentAccount.accountType === 'real' ? 'rgba(229, 57, 53, 0.1)' : 'rgba(33, 150, 243, 0.1)', color: currentAccount.accountType === 'real' ? '#E53935' : '#2196F3', border: 'none' }}>
                    {currentAccount.accountType === 'real' ? t('accounts.detail.accountType.real') : t('accounts.detail.accountType.demo')}
                  </Tag>
                )}
                <Tag style={{ borderRadius: '6px', background: currentAccount.isInvestor ? 'rgba(255, 152, 0, 0.1)' : 'rgba(0, 166, 81, 0.1)', color: currentAccount.isInvestor ? '#FF9800' : '#00A651', border: 'none' }}>
                  {currentAccount.isInvestor ? t('accounts.detail.mode.investor') : t('accounts.detail.mode.trader')}
                </Tag>
                <Tag style={{ background: statusConfig.bg, color: statusConfig.color, border: 'none', borderRadius: '6px', cursor: currentAccount.status === 'disconnected' || currentAccount.status === 'error' ? 'pointer' : 'default' }}
                  onClick={() => { if (currentAccount.status === 'disconnected' || currentAccount.status === 'error') handleConnect(); }}>
                  {connecting ? t('accounts.detail.status.connecting') : statusConfig.text}
                </Tag>
              </div>
              <div className="flex items-center gap-4 mt-1" style={{ color: '#8A9AA5', fontSize: '14px' }}>
                <span>{currentAccount.brokerCompany}</span><span>•</span><span>{currentAccount.brokerServer}</span><span>•</span><span>{t('accounts.detail.leverage', { leverage: currentAccount.leverage })}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={analyticsLoading}>{t('common.refresh')}</Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}><Button icon={<MoreOutlined />} /></Dropdown>
          </div>
        </div>

        {/* Info cards */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
          <InfoCard icon={<WalletOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.balance')} value={formatCurrency(balance)} loading={isStreamLoading} />
          <InfoCard icon={<LineChartOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.equity')} value={formatCurrency(equity)} loading={isStreamLoading} />
          <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
            <div className="flex items-center gap-2 mb-3">
              {profit >= 0 ? <RiseOutlined style={{ color: '#00A651' }} /> : <FallOutlined style={{ color: '#E53935' }} />}
              <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('accounts.detail.cards.floatingProfit')}</span>
            </div>
            {isStreamLoading ? <div className="text-lg" style={{ color: '#8A9AA5' }}>{t('common.loading')}</div>
              : <div className="flex items-baseline gap-2"><span className="text-2xl font-bold" style={{ color: profit >= 0 ? '#00A651' : '#E53935' }}>{profit >= 0 ? '+' : ''}{formatCurrency(profit)}</span><span style={{ color: profit >= 0 ? '#00A651' : '#E53935', fontSize: '14px' }}>({profitPercent >= 0 ? '+' : ''}{profitPercent.toFixed(2)}%)</span></div>
            }
          </div>
        </div>

        {/* Small info cards */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <SmallInfoCard icon={<DollarOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.marginUsed')} value={formatCurrency(margin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<DollarOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.marginFree')} value={formatCurrency(freeMargin)} loading={isStreamLoading} />
          <SmallInfoCard icon={<PercentageOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.marginLevel')} value={margin > 0 ? `${(marginLevel || 0).toFixed(2)}%` : '--'} loading={isStreamLoading} valueColor={margin > 0 && (marginLevel || 0) < 100 ? '#E53935' : '#141D22'} />
          <SmallInfoCard icon={<WarningOutlined style={{ color: '#8A9AA5' }} />} label={t('accounts.detail.cards.credit')} value={formatCurrency(credit)} loading={isStreamLoading} />
        </div>

        {/* Trade tabs */}
        <div className="rounded-2xl overflow-hidden mb-6" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <AccountTradeTabs
            id={id}
            realPositions={positions}
            pendingOrders={pendingOrders}
            historyTrades={historyTrades}
            historyTotal={historyTotal}
            historyPage={historyPage}
            historyPageSize={10}
            onHistoryTradesChange={setHistoryTrades}
            onHistoryTotalChange={setHistoryTotal}
            onHistoryPageChange={setHistoryPage}
            historyLoading={historyLoading}
          />
        </div>

        {/* Analytics */}
        <AccountAnalyticsSection
          analyticsLoading={analyticsLoading}
          analyticsError={analyticsError}
          onRetryAnalytics={handleRetry}
          chartType={chartType}
          setChartType={setChartType}
          chartPeriod={chartPeriod}
          setChartPeriod={setChartPeriod}
          equityChartData={equityChartData}
          profitByMonthData={profitByMonthData}
          symbolDistributionData={symbolDistributionData}
          dailyPnLData={dailyPnLData}
          hourlyData={hourlyData}
          tradeStats={tradeStats}
          riskMetrics={riskMetrics}
          monthlyAnalysisYears={monthlyAnalysisYears}
          monthlyAnalysisData={monthlyAnalysisData}
          currency={currentAccount?.currency || 'USD'}
          accountId={id}
        />

        {/* Delete modal */}
        <Modal
          title={t('accounts.detail.actions.deleteAccount')}
          open={deleteModalOpen}
          onOk={handleDelete}
          onCancel={() => setDeleteModalOpen(false)}
          confirmLoading={deleting}
          okText={t('accounts.detail.actions.deleteConfirm')}
          cancelText={t('common.cancel')}
          okButtonProps={{ danger: true }}
          destroyOnClose
        >
          <div style={{ marginBottom: 16, color: '#E53935' }}>{t('accounts.detail.actions.deleteWarning')}</div>
          <div style={{ marginBottom: 8, color: '#8A9AA5' }}>{t('accounts.detail.actions.deletePasswordHint')}</div>
          <Input
            placeholder={t('accounts.detail.actions.deletePasswordPlaceholder')}
            value={deletePassword}
            onChange={(e) => setDeletePassword(e.target.value)}
            onPressEnter={handleDelete}
            disabled={deleting}
          />
        </Modal>
      </div>
    </div>
  );
}
