import { useMemo } from 'react';
import { Tag, Button, Spin, Dropdown, Modal, Input, Tooltip } from 'antd';
import type { MenuProps } from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, PauseCircleOutlined,
  CaretRightOutlined, MoreOutlined, WalletOutlined, LineChartOutlined,
  RiseOutlined, FallOutlined, DollarOutlined, PercentageOutlined,
  WarningOutlined, DeleteOutlined, FileTextOutlined, ClockCircleOutlined,
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

  const disabled = !!currentAccount?.isDisabled;
  const { balance, equity, margin, freeMargin, marginLevel, profit, profitPercent, credit } = financials;

  const formatCurrency = (value: number) => {
    if (disabled) return '--';
    const isNegative = value < 0;
    return `${isNegative ? '-' : ''}${Math.abs(value).toFixed(2)} ${currentAccount?.currency || 'USD'}`;
  };

  const statusConfig = useMemo(() => {
    if (!currentAccount) return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t('common.unknown') };
    if (currentAccount.isDisabled) return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t('accounts.detail.status.disabled') };
    switch (currentAccount.status) {
      case 'connected': return { color: 'var(--color-success)', bg: 'var(--color-success-bg)', text: t('accounts.detail.status.connected') };
      case 'connecting': return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', text: t('accounts.detail.status.connecting') };
      case 'disconnected': return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', text: t('accounts.detail.status.disconnected') };
      case 'error': return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', text: t('accounts.detail.status.error') };
      default: return { color: 'var(--color-text-muted)', bg: 'var(--color-bg-tertiary)', text: t('common.unknown') };
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
      icon: <DeleteOutlined style={{ color: 'var(--color-danger)' }} />,
      onClick: () => setDeleteModalOpen(true),
      danger: true,
    },
  ], [currentAccount?.isDisabled, togglePending, disabling, handleToggleStatus, t]);

  const displayName = currentAccount?.alias || currentAccount?.login;
  const hasAlias = !!currentAccount?.alias;
  const lastConnectedText = currentAccount?.connectedAt
    ? t('accounts.detail.lastConnected', { time: new Date(currentAccount.connectedAt).toLocaleString() })
    : null;

  if (!currentAccount) {
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
                    {currentAccount.accountType === 'real' ? t('accounts.detail.accountType.real') : t('accounts.detail.accountType.demo')}
                  </Tag>
                )}
                <Tag style={{
                  borderRadius: 6,
                  background: currentAccount.isInvestor ? 'var(--color-warning-bg)' : 'var(--color-success-bg)',
                  color: currentAccount.isInvestor ? 'var(--color-warning)' : 'var(--color-success)',
                  border: 'none',
                }}>
                  {currentAccount.isInvestor ? t('accounts.detail.mode.investor') : t('accounts.detail.mode.trader')}
                </Tag>
              </div>
              <div className="flex items-center gap-3 mt-0.5 flex-wrap" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>
                <span>{currentAccount.brokerCompany} · {currentAccount.brokerServer}</span>
                <span>·</span>
                <span>{t('accounts.detail.leverage', { leverage: currentAccount.leverage })}</span>
                {lastConnectedText && (
                  <>
                    <span>·</span>
                    <Tooltip title={lastConnectedText}>
                      <span style={{ cursor: 'default' }}>
                        <ClockCircleOutlined style={{ marginRight: 2 }} />
                        {t('accounts.detail.connected')}
                      </span>
                    </Tooltip>
                  </>
                )}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2 flex-shrink-0">
            <Button icon={<FileTextOutlined />} onClick={() => navigate(`/accounts/${id}/report`)}>
              {t('accounts.report.titleShort')}
            </Button>
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

        {/* ── Status tag + quick connect ── */}
        <div className="flex items-center gap-3 mb-4">
          <Tag
            style={{
              background: statusConfig.bg, color: statusConfig.color, border: 'none', borderRadius: 6,
              cursor: (currentAccount.status === 'disconnected' || currentAccount.status === 'error') ? 'pointer' : 'default',
              padding: '2px 12px', fontSize: 13,
            }}
            onClick={() => { if (currentAccount.status === 'disconnected' || currentAccount.status === 'error') handleConnect(); }}
          >
            {connecting ? t('accounts.detail.status.connecting') : statusConfig.text}
          </Tag>
        </div>

        {/* ── Primary metrics: Equity · P&L · Margin Level ── */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
          {/* Equity */}
          <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
            <div className="flex items-center gap-2 mb-1">
              <LineChartOutlined style={{ color: 'var(--color-text-muted)', fontSize: 14 }} />
              <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t('accounts.detail.cards.equity')}</span>
            </div>
            {isStreamLoading
              ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
              : <div className="text-xl font-bold" style={{ color: 'var(--color-text)' }}>{formatCurrency(equity)}</div>
            }
            <div style={{ color: 'var(--color-text-muted)', fontSize: 11, marginTop: 2 }}>
              {t('accounts.detail.cards.balance')}: {formatCurrency(balance)}
            </div>
          </div>

          {/* Floating P&L */}
          <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
            <div className="flex items-center gap-2 mb-1">
              {profit >= 0 ? <RiseOutlined style={{ color: 'var(--color-success)', fontSize: 14 }} /> : <FallOutlined style={{ color: 'var(--color-danger)', fontSize: 14 }} />}
              <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t('accounts.detail.cards.floatingProfit')}</span>
            </div>
            {isStreamLoading
              ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
              : <>
                <div className="text-xl font-bold" style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
                  {profit >= 0 ? '+' : ''}{formatCurrency(profit)}
                </div>
                <div style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)', fontSize: 12, marginTop: 2 }}>
                  {profitPercent >= 0 ? '+' : ''}{profitPercent.toFixed(2)}%
                </div>
              </>
            }
          </div>

          {/* Margin Level */}
          <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
            <div className="flex items-center gap-2 mb-1">
              <PercentageOutlined style={{ color: 'var(--color-text-muted)', fontSize: 14 }} />
              <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t('accounts.detail.cards.marginLevel')}</span>
            </div>
            {isStreamLoading
              ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
              : <>
                <div className="text-xl font-bold" style={{
                  color: margin > 0 && (marginLevel || 0) < 100 ? 'var(--color-danger)' : 'var(--color-text)',
                }}>
                  {margin > 0 ? `${(marginLevel || 0).toFixed(2)}%` : '--'}
                </div>
                <div style={{ color: 'var(--color-text-muted)', fontSize: 11, marginTop: 2 }}>
                  {t('accounts.detail.cards.marginUsed')}: {formatCurrency(margin)}
                </div>
              </>
            }
          </div>
        </div>

        {/* ── Secondary metrics: Balance · Margin · Free Margin · Credit ── */}
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
          <SmallInfoCard
            icon={<WalletOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
            label={t('accounts.detail.cards.balance')}
            value={formatCurrency(balance)}
            loading={isStreamLoading}
          />
          <SmallInfoCard
            icon={<DollarOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
            label={t('accounts.detail.cards.marginUsed')}
            value={formatCurrency(margin)}
            loading={isStreamLoading}
          />
          <SmallInfoCard
            icon={<DollarOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
            label={t('accounts.detail.cards.marginFree')}
            value={formatCurrency(freeMargin)}
            loading={isStreamLoading}
          />
          <SmallInfoCard
            icon={<WarningOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
            label={t('accounts.detail.cards.credit')}
            value={formatCurrency(credit)}
            loading={isStreamLoading}
          />
        </div>

        {/* ── Trade tabs ── */}
        <div className="rounded-xl overflow-hidden mb-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
          {disabled ? (
            <div className="text-center py-12" style={{ color: 'var(--color-text-muted)' }}>
              <PauseCircleOutlined style={{ fontSize: 48, opacity: 0.3 }} />
              <p className="mt-4">{t('accounts.detail.status.disabled')}</p>
            </div>
          ) : (
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
          )}
        </div>

        {/* ── Analytics ── */}
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
          <div style={{ marginBottom: 16, color: 'var(--color-danger)' }}>{t('accounts.detail.actions.deleteWarning')}</div>
          <div style={{ marginBottom: 8, color: 'var(--color-text-muted)' }}>{t('accounts.detail.actions.deletePasswordHint')}</div>
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
