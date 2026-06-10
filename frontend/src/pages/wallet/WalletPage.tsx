import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Descriptions } from 'antd';
import { WalletOutlined, TransactionOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { walletApi } from '@/client/wallet';
import { queryKeys } from '@/queries/queryKeys';
import { StatusResult } from '@/components/common/StatusResult';
import { formatAmount } from '@/utils/amount';
import { useMemo } from 'react';

const { Title } = Typography;

export default function WalletPage() {
  const { t } = useTranslation();

  const { data: wallet, isLoading, error, refetch } = useQuery({
    queryKey: queryKeys.wallet.all,
    queryFn: () => walletApi.getWallet(),
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: queryKeys.wallet.transactions(),
    queryFn: () => walletApi.listTransactions(1, 50),
  });

  const columns = useMemo(() => [
    {
      title: t('wallet.table.type', { defaultValue: 'Type' }),
      dataIndex: 'txType',
      key: 'txType',
      width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { deposit: 'green', withdrawal: 'red', adjustment: 'blue', fee: 'orange', reversal: 'orange' };
        const label = t(`wallet.txType.${v}`, { defaultValue: v });
        return <Tag color={colors[v] || 'default'}>{label}</Tag>;
      },
    },
    {
      title: t('wallet.table.amount', { defaultValue: 'Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (v: string) => {
        const num = parseFloat(v);
        return <span style={{ color: num >= 0 ? '#00A651' : '#E53935', fontWeight: 500 }}>{num >= 0 ? '+' : ''}{formatAmount(v)}</span>;
      },
    },
    {
      title: t('wallet.table.balanceAfter', { defaultValue: 'Balance After' }),
      dataIndex: 'balanceAfter',
      key: 'balanceAfter',
      width: 120,
      render: (v: string) => formatAmount(v),
    },
    {
      title: t('wallet.table.description', { defaultValue: 'Description' }),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('wallet.table.time', { defaultValue: 'Time' }),
      dataIndex: 'createdAtTsMs',
      key: 'createdAtTsMs',
      width: 180,
      render: (v: string) => new Date(Number(v)).toLocaleString(),
    },
  ], [t]);

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        <WalletOutlined style={{ marginRight: 8, color: '#D4AF37' }} />
        {t('wallet.title', { defaultValue: 'My Wallet' })}
      </Title>

      <StatusResult loading={isLoading} error={error instanceof Error ? error.message : null} onRetry={refetch}>
        {wallet && (
          <Card size="small" style={{ marginBottom: 24 }}>
            <Descriptions column={3} size="small">
              <Descriptions.Item label={t('wallet.accountNumber', { defaultValue: 'Account' })}>
                <span style={{ fontFamily: 'monospace', fontSize: 18, fontWeight: 700, color: '#D4AF37' }}>
                  {wallet.accountNumber || '-'}
                </span>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.balance', { defaultValue: 'Balance' })}>
                <span style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-text)' }}>
                  {formatAmount(wallet.balance)} {wallet.currency}
                </span>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.frozen', { defaultValue: 'Frozen' })}>
                <span style={{ fontSize: 16, color: 'var(--color-text)' }}>
                  {formatAmount(wallet.frozenBalance)} {wallet.currency}
                </span>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}
      </StatusResult>

      <Card
        title={<span><TransactionOutlined style={{ marginRight: 8 }} />{t('wallet.transactions', { defaultValue: 'Transactions' })}</span>}
        size="small"
      >
        <StatusResult loading={txLoading} empty={!txData?.transactions?.length}>
          <Table
            columns={columns}
            dataSource={txData?.transactions || []}
            rowKey="id"
            pagination={{ pageSize: 20, size: 'small' }}
            size="small"
          />
        </StatusResult>
      </Card>
    </div>
  );
}
