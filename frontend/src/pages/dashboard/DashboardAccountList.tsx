import { Card, Tag } from 'antd';
import { BankOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';
import type { Account } from '@/types/account';

interface Props {
  accounts: Account[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}

function getStatusTag(item: Account, t: (key: string) => string) {
  if (item?.isDisabled) {
    return <Tag color="default">{t('accounts.card.status.disabled')}</Tag>;
  }
  const s = item?.status || item?.accountStatus;
  if (!s || s === 'unknown') return null;
  const colors: Record<string, string> = { connected: 'green', disconnected: 'red', error: 'red', connecting: 'blue' };
  const labels: Record<string, string> = {
    connected: t('accounts.card.status.connected'),
    connecting: t('accounts.card.status.connecting'),
    disconnected: t('accounts.card.status.disconnected'),
    error: t('accounts.card.status.error'),
    disabled: t('accounts.card.status.disabled'),
  };
  return <Tag color={colors[s] || 'default'}>{labels[s] || s}</Tag>;
}

function AccountCard({ item, navigate, t }: { item: Account; navigate: (path: string) => void; t: (key: string) => string }) {
  const isDisabled = item.isDisabled;
  const balance = item.balance;
  const equity = item.equity;
  const floating = item.profit ?? 0;
  const isMT4 = item.mtType === 'MT4';
  const fmt = (v: number, prefix = '') => {
    if (isDisabled) return '--';
    return `${prefix}${Number.isFinite(v) ? v.toFixed(2) : '0.00'}`;
  };

  return (
    <div
      role="button"
      tabIndex={0}
      onClick={() => navigate(`/accounts/${item.id}`)}
      onKeyDown={(e) => { if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); navigate(`/accounts/${item.id}`); } }}
      className="p-4 rounded-xl cursor-pointer transition-all focus-visible:outline focus-visible:outline-2 focus-visible:outline-[#D4AF37]"
      style={{ background: 'var(--color-bg-secondary)', border: '1px solid rgba(0,0,0,0.05)' }}
      onMouseEnter={(e) => {
        e.currentTarget.style.background = '#E8ECF0';
        e.currentTarget.style.borderColor = 'rgba(212,175,55,0.2)';
      }}
      onMouseLeave={(e) => {
        e.currentTarget.style.background = '#F5F7F9';
        e.currentTarget.style.borderColor = 'rgba(0,0,0,0.05)';
      }}
    >
      {/* Header: icon + login + platform tag + status */}
      <div className="flex items-center gap-2 mb-2">
        <div
          className="w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0"
          style={{ background: isMT4 ? 'rgba(33,150,243,0.1)' : 'rgba(212,175,55,0.1)' }}
        >
          <BankOutlined style={{ fontSize: 16, color: isMT4 ? '#2196F3' : '#D4AF37' }} />
        </div>
        <span style={{ color: 'var(--color-text)', fontWeight: 600, fontSize: 14 }}>{item.login}</span>
        <Tag color={isMT4 ? 'blue' : 'gold'} className="!text-xs !m-0">{item.mtType}</Tag>
        <div className="flex-1" />
        {getStatusTag(item, t)}
      </div>

      {/* Server — more specific than company name */}
      <div className="text-xs mb-3 truncate" style={{ color: 'var(--color-text-muted)' }}>{item.brokerServer || '—'}</div>

      {/* Financials — 3 columns */}
      <div className="grid grid-cols-3 gap-2 text-center">
        <div>
          <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>{t('dashboard.fields.balance')}</div>
          <div className="text-sm font-semibold" style={{ color: isDisabled ? 'var(--color-text-muted)' : 'var(--color-text)' }}>{fmt(balance, '$')}</div>
        </div>
        <div>
          <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>{t('dashboard.fields.equity')}</div>
          <div className="text-sm font-semibold" style={{ color: isDisabled ? 'var(--color-text-muted)' : 'var(--color-text)' }}>{fmt(equity, '$')}</div>
        </div>
        {isDisabled ? (
          <div>
            <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>{t('dashboard.fields.floating')}</div>
            <div className="text-sm font-semibold" style={{ color: 'var(--color-text-muted)' }}>--</div>
          </div>
        ) : (
        <div>
          <div className="text-xs mb-0.5" style={{ color: 'var(--color-text-muted)' }}>{t('dashboard.fields.floating')}</div>
          <div
            className="text-sm font-semibold flex items-center justify-center gap-1"
            style={{ color: (Number.isFinite(floating) ? floating : 0) >= 0 ? '#00A651' : '#E53935' }}
          >
            {(Number.isFinite(floating) ? floating : 0) >= 0 ? (
              <RiseOutlined style={{ fontSize: 12 }} />
            ) : (
              <FallOutlined style={{ fontSize: 12 }} />
            )}
            {(Number.isFinite(floating) ? floating : 0) >= 0 ? '+' : ''}{(Number.isFinite(floating) ? floating : 0).toFixed(2)}
          </div>
        </div>
        )}
      </div>
    </div>
  );
}

export default function DashboardAccountList({ accounts, loading, error, onRetry }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <Card
      title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t('dashboard.accountList')}</span>}
      className="glass-card"
    >
      <StatusResult
        loading={loading}
        error={error}
        onRetry={onRetry}
        empty={!loading && !error && (!accounts || accounts.length === 0)}
        emptyText={t('dashboard.noAccounts')}
      >
        <div style={{ maxHeight: 520, overflowY: 'auto' }}>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
            {(accounts || []).map((item) => (
              <AccountCard key={item.id} item={item} navigate={navigate} t={t} />
            ))}
          </div>
        </div>
      </StatusResult>
    </Card>
  );
}
