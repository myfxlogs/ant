import { useCallback, useMemo } from 'react';
import { Tag, Button, Spin, Dropdown } from 'antd';
import type { MenuProps } from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, PauseCircleOutlined,
  CaretRightOutlined, MoreOutlined,
  WarningOutlined, DeleteOutlined, FileTextOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next'
import { DETAIL_ACCOUNT_TYPE_DEMO_KEY, DETAIL_ACCOUNT_TYPE_REAL_KEY, DETAIL_ACTIONS_DELETE_ACCOUNT_KEY, DETAIL_ACTIONS_DISABLE_ACCOUNT_KEY, DETAIL_ACTIONS_ENABLE_ACCOUNT_KEY, DETAIL_LEVERAGE_KEY, DETAIL_MESSAGES_FETCH_ACCOUNT_FAILED_KEY, DETAIL_MODE_INVESTOR_KEY, DETAIL_MODE_TRADER_KEY, DETAIL_STATUS_CONNECTED_KEY, DETAIL_STATUS_CONNECTING_KEY, DETAIL_STATUS_DISABLED_KEY, DETAIL_STATUS_DISCONNECTED_KEY, DETAIL_STATUS_ERROR_KEY, REPORT_TITLE_SHORT_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import AccountTradeTabs from './components/AccountTradeTabs';
import AccountMetricsCards from './components/AccountMetricsCards';
import AccountAnalyticsSection from './components/AccountAnalyticsSection';
import AccountDeleteModal from './components/AccountDeleteModal';
import ShareAccountButton from './components/ShareAccountButton';
import { useAccountDetailData } from './AccountDetail/useAccountDetailData';

export default function AccountDetail() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const {
    currentAccount, isStreamLoading, accountLoadError, financials,
    positions, pendingOrders,
    historyTrades, historyTotal, historyPage, historyLoading,
    setHistoryTrades, setHistoryTotal, setHistoryPage,
    connecting, disabling,
    handleConnect, handleToggleStatus, handleRefresh, handleRetry,
    deleteModalOpen, setDeleteModalOpen,
    deletePassword, setDeletePassword,
    deleting, handleDelete,
    togglePending,
    // analytics
    chartType, setChartType, chartPeriod, setChartPeriod,
    analyticsLoading, analyticsError,
    equityChartData, profitByMonthData, symbolDistributionData,
    dailyPnLData, hourlyData, tradeStats, riskMetrics,
    monthlyAnalysisYears, monthlyAnalysisData,
  } = useAccountDetailData(id);

  const s = (currentAccount?.status || currentAccount?.accountStatus || '').toLowerCase();
  const disabled = s === 'disconnected' || s === 'frozen' || s === 'circuit_open' || currentAccount?.isDisabled === true;
  const { balance, equity, margin, freeMargin, marginLevel, profit, profitPercent, credit } = financials;

  const formatCurrency = useCallback((value: number) => {
    if (disabled) return '--';
    const isNegative = value < 0;
    return `${isNegative ? '-' : ''}${Math.abs(value).toFixed(2)} ${currentAccount?.currency || 'USD'}`;
  }, [disabled, currentAccount?.currency]);

  const statusConfig = useMemo(() => {
    if (!currentAccount) return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t('common.unknown') };
    if (currentAccount.isDisabled) return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t(DETAIL_STATUS_DISABLED_KEY) };
    switch (currentAccount.status) {
      case 'connected': return { color: 'var(--color-success)', bg: 'var(--color-success-bg)', text: t(DETAIL_STATUS_CONNECTED_KEY) };
      case 'connecting': return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', text: t(DETAIL_STATUS_CONNECTING_KEY) };
      case 'disconnected': return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', text: t(DETAIL_STATUS_DISCONNECTED_KEY) };
      case 'error': return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', text: t(DETAIL_STATUS_ERROR_KEY) };
      case 'circuit_open': return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', text: t('accounts.status.circuit_open', 'Circuit Open') };
      case 'circuit_half_open': return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', text: t('accounts.status.circuit_half_open', 'Circuit Testing') };
      default: return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t('common.unknown') };
    }
  }, [currentAccount, t]);

  const menuItems: MenuProps['items'] = useMemo(() => [
    {
      key: 'toggle',
      label: currentAccount?.isDisabled
        ? t(DETAIL_ACTIONS_ENABLE_ACCOUNT_KEY)
        : t(DETAIL_ACTIONS_DISABLE_ACCOUNT_KEY),
      icon: togglePending ? <Spin size="small" />
        : currentAccount?.isDisabled ? <CaretRightOutlined /> : <PauseCircleOutlined />,
      onClick: handleToggleStatus,
      disabled: disabling,
    },
    {
      key: 'delete',
      label: t(DETAIL_ACTIONS_DELETE_ACCOUNT_KEY),
      icon: <DeleteOutlined style={{ color: 'var(--color-danger)' }} />,
      onClick: () => setDeleteModalOpen(true),
      danger: true,
    },
  ], [currentAccount?.isDisabled, togglePending, disabling, handleToggleStatus, setDeleteModalOpen, t]);

  const displayName = currentAccount?.alias || currentAccount?.login;
  const hasAlias = !!currentAccount?.alias;
  if (!currentAccount) {
    if (accountLoadError) {
      return (
        <div className="p-4 flex flex-col justify-center items-center h-64 gap-4">
          <div style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>
            {t(DETAIL_MESSAGES_FETCH_ACCOUNT_FAILED_KEY)}
          </div>
          <Button onClick={handleRetry}>{t('common.retry')}</Button>
        </div>
      );
    }
    return <div className="p-4 flex justify-center items-center h-64"><Spin size="large" /></div>;
  }

  return (
    <div className="min-h-screen" style={{ background: 'var(--color-bg-secondary)' }}>
      <div className="max-w-7xl mx-auto p-4">
        {/* ── Header ── */}
        <div className="flex items-start justify-between mb-4">
          <div className="flex items-center gap-3 min-w-0">
            <Button type="text" icon={<ArrowLeftOutlined />} onClick={() => navigate('/')}
              style={{ color: 'var(--color-text-muted)', flexShrink: 0 }} />
            <div className="min-w-0">
              <div className="flex items-center gap-2 flex-wrap">
                <h1 className="text-xl font-bold truncate" style={{ color: 'var(--color-text)' }}>
                  {displayName}
                </h1>
                {hasAlias && (
                  <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>
                    ({currentAccount.login})
                  </span>
                )}
                <Tag color={currentAccount.mtType === 'MT4' ? 'blue' : 'purple'}>{currentAccount.mtType}</Tag>
                {currentAccount.accountType && (
                  <Tag style={{
                    borderRadius: 6,
                    background: currentAccount.accountType === 'real' ? 'var(--color-danger-bg)' : 'var(--color-info-bg)',
                    color: currentAccount.accountType === 'real' ? 'var(--color-danger)' : 'var(--color-info)',
                    border: 'none',
                  }}>
                    {currentAccount.accountType === 'real' ? t(DETAIL_ACCOUNT_TYPE_REAL_KEY) : t(DETAIL_ACCOUNT_TYPE_DEMO_KEY)}
                  </Tag>
                )}
                <Tag style={{
                  borderRadius: 6,
                  background: currentAccount.isInvestor ? 'var(--color-warning-bg)' : 'var(--color-success-bg)',
                  color: currentAccount.isInvestor ? 'var(--color-warning)' : 'var(--color-success)',
                  border: 'none',
                }}>
                  {currentAccount.isInvestor ? t(DETAIL_MODE_INVESTOR_KEY) : t(DETAIL_MODE_TRADER_KEY)}
                </Tag>
              </div>
              <div className="flex items-center gap-3 mt-0.5 flex-wrap" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>
                <span>{currentAccount.brokerCompany} · {currentAccount.brokerServer}</span>
                <span>·</span>
                <span>{t(DETAIL_LEVERAGE_KEY, { leverage: currentAccount.leverage })}</span>
                <span>·</span>
                <Tag
                  style={{
                    background: statusConfig.bg, color: statusConfig.color, border: 'none', borderRadius: 6,
                    cursor: (currentAccount.status === 'disconnected' || currentAccount.status === 'error') ? 'pointer' : 'default',
                    padding: '0 8px', fontSize: 12, margin: 0,
                  }}
                  onClick={() => { if (currentAccount.status === 'disconnected' || currentAccount.status === 'error') handleConnect(); }}
                >
                  {connecting ? t(DETAIL_STATUS_CONNECTING_KEY) : statusConfig.text}
                </Tag>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            <Button icon={<FileTextOutlined />} onClick={() => navigate(`/accounts/${id}/report`)}>
              {t(REPORT_TITLE_SHORT_KEY)}
            </Button>
            {id && <ShareAccountButton accountId={id} />}
            <Button icon={<ReloadOutlined />} onClick={handleRefresh} loading={analyticsLoading}>
              {t('common.refresh')}
            </Button>
            <Dropdown menu={{ items: menuItems }} trigger={['click']}>
              <Button icon={<MoreOutlined />} />
            </Dropdown>
          </div>
        </div>

        {/* ── Error banner ── */}
        {currentAccount.status === 'error' && currentAccount.lastError && (
          <div className="rounded-lg p-3 mb-4 flex items-center gap-2"
            style={{ background: 'var(--color-danger-bg-subtle)', border: '1px solid var(--color-danger-bg)' }}>
            <WarningOutlined style={{ color: 'var(--color-danger)' }} />
            <span style={{ color: 'var(--color-danger)', fontSize: 13 }}>{currentAccount.lastError}</span>
          </div>
        )}

        {/* ── Primary + Secondary metrics ── */}
        <AccountMetricsCards
          isStreamLoading={isStreamLoading}
          disabled={disabled}
          formatCurrency={formatCurrency}
          balance={balance} equity={equity}
          profit={profit} profitPercent={profitPercent}
          margin={margin} freeMargin={freeMargin}
          marginLevel={marginLevel} credit={credit}
        />

        {/* ── Trade tabs ── */}
        <div className="rounded-xl overflow-hidden mb-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
          {disabled ? (
            <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
              <PauseCircleOutlined style={{ fontSize: 48, opacity: 0.3 }} />
              <p className="mt-4">{t(DETAIL_STATUS_DISABLED_KEY)}</p>
            </div>
          ) : (
            <AccountTradeTabs
              id={id ?? ''}
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
          )}
        </div>

        {/* ── Account-level Analytics ── */}
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

        {/* ── Delete modal ── */}
        <AccountDeleteModal
          open={deleteModalOpen}
          deletePassword={deletePassword}
          deleting={deleting}
          onDelete={handleDelete}
          onCancel={() => setDeleteModalOpen(false)}
          onPasswordChange={setDeletePassword}
        />
      </div>
    </div>
  );
}
