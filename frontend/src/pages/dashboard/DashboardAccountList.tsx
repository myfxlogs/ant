import { Card, Row, Col, Tag } from 'antd';
import { BankOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';
import type { Account } from '@/types/account';

interface AccountInfo { balance?: number; equity?: number; profit?: number; }

interface Props {
  accounts: Account[];
  accountInfoValues: Record<string, AccountInfo>;
  loading: boolean;
  error: string | null;
  onRetry: () => void;
}

function getStatusTag(item: Account) {
  const s = item?.status || item?.accountStatus;
  if (!s || s === 'unknown') return null;
  const colors: Record<string, string> = { connected: 'green', disconnected: 'red', error: 'red' };
  const labels: Record<string, string> = {};
  return <Tag color={colors[s] || 'default'}>{labels[s] || s}</Tag>;
}

export default function DashboardAccountList({ accounts, accountInfoValues, loading, error, onRetry }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  return (
    <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('dashboard.accountList')}</span>} className="glass-card">
      <StatusResult loading={loading} error={error} onRetry={onRetry} empty={!loading && !error && (!accounts || accounts.length === 0)} emptyText={t('dashboard.noAccounts')}>
        <div className="space-y-3">
          {(accounts || []).slice(0, 4).map((item) => {
            const live = accountInfoValues[item.id];
            const rowBalance = live?.balance ?? item.balance;
            const rowEquity = live?.equity ?? item.equity;
            const rowFloating = live?.profit ?? item.profit ?? 0;
            return (
              <div key={item.id} onClick={() => navigate(`/accounts/${item.id}`)}
                className="flex items-center justify-between p-4 rounded-xl cursor-pointer transition-all"
                style={{ background: '#F5F7F9', border: '1px solid rgba(0,0,0,0.05)' }}
                onMouseEnter={(e) => { e.currentTarget.style.background = '#E8ECF0'; e.currentTarget.style.borderColor = 'rgba(212,175,55,0.2)'; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = '#F5F7F9'; e.currentTarget.style.borderColor = 'rgba(0,0,0,0.05)'; }}>
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: item.mtType === 'MT4' ? 'rgba(33,150,243,0.1)' : 'rgba(212,175,55,0.1)' }}>
                    <BankOutlined size={20} color={item.mtType === 'MT4' ? '#2196F3' : '#D4AF37'} />
                  </div>
                  <div>
                    <div className="flex items-center gap-2">
                      <span style={{ color: '#141D22', fontWeight: 500 }}>{item.login}</span>
                      <Tag color={item.mtType === 'MT4' ? 'blue' : 'gold'} className="!text-xs">{item.mtType}</Tag>
                      {getStatusTag(item)}
                    </div>
                    <div className="text-sm mt-1" style={{ color: '#8A9AA5' }}>{item.brokerCompany}</div>
                  </div>
                </div>
                <div className="flex flex-col sm:flex-row sm:items-center gap-2 sm:gap-4 text-right">
                  {[
                    { label: t('dashboard.fields.balance'), value: rowBalance },
                    { label: t('dashboard.fields.equity'), value: rowEquity },
                    { label: t('dashboard.fields.floating'), value: rowFloating, color: rowFloating >= 0 ? '#00A651' : '#E53935' },
                  ].map((f, i) => (
                    <div key={i} className="flex sm:flex-col items-center sm:items-end justify-between sm:justify-start">
                      <span className="text-xs sm:hidden" style={{ color: '#8A9AA5' }}>{f.label}</span>
                      <div className="hidden sm:block text-xs mb-1" style={{ color: '#8A9AA5' }}>{f.label}</div>
                      <div className="font-medium" style={{ color: f.color || '#141D22' }}>
                        {f.value != null && Number.isFinite(f.value) ? f.value.toFixed(2) : '0.00'}
                      </div>
                    </div>
                  ))}
                  <div className="text-xs hidden sm:block" style={{ color: '#8A9AA5' }}>{item.currency || 'USD'}</div>
                </div>
              </div>
            );
          })}
        </div>
      </StatusResult>
    </Card>
  );
}
