import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Descriptions, Button, Alert, message, Tooltip, Spin } from 'antd';
import { WalletOutlined, TransactionOutlined, CopyOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { QRCodeSVG } from 'qrcode.react';
import { walletApi } from '@/client/wallet';
import { depositApi } from '@/client/deposit';
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

  const { data: depositAddr, isLoading: addrLoading } = useQuery({
    queryKey: queryKeys.deposit.address,
    queryFn: () => depositApi.getDepositAddress(),
  });

  const { data: myDeposits } = useQuery({
    queryKey: queryKeys.deposit.myDeposits,
    queryFn: () => depositApi.listMyDeposits(1, 20),
  });

  const copyAddress = () => {
    if (depositAddr?.address) {
      navigator.clipboard.writeText(depositAddr.address);
      message.success(t('wallet.deposit.addressCopied'));
    }
  };

  const columns = useMemo(() => [
    {
      title: t('wallet.table.type'),
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
      title: t('wallet.table.amount'),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (v: string) => {
        const num = parseFloat(v);
        return <span style={{ color: num >= 0 ? '#00A651' : '#E53935', fontWeight: 500 }}>{num >= 0 ? '+' : ''}{formatAmount(v)}</span>;
      },
    },
    {
      title: t('wallet.table.balanceAfter'),
      dataIndex: 'balanceAfter',
      key: 'balanceAfter',
      width: 120,
      render: (v: string) => formatAmount(v),
    },
    {
      title: t('wallet.table.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('wallet.table.time'),
      dataIndex: 'createdAtTsMs',
      key: 'createdAtTsMs',
      width: 180,
      render: (v: string) => new Date(Number(v)).toLocaleString(),
    },
  ], [t]);

  const depositColumns = useMemo(() => [
    {
      title: t('wallet.deposit.table.amount'),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
      render: (v: string) => <span style={{ color: '#00A651', fontWeight: 500 }}>+{formatAmount(v)}</span>,
    },
    {
      title: t('wallet.deposit.table.txHash'),
      dataIndex: 'txHash',
      key: 'txHash',
      ellipsis: true,
      width: 200,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 20)}...</span> : '-',
    },
    {
      title: t('wallet.deposit.table.confirmations'),
      dataIndex: 'confirmations',
      key: 'confirmations',
      width: 120,
    },
    {
      title: t('wallet.deposit.table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => {
        const colors: Record<string, string> = { CONFIRMED: 'green', MANUAL_REVIEW: 'orange' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('wallet.deposit.table.time'),
      dataIndex: 'confirmedAt',
      key: 'confirmedAt',
      width: 180,
      render: (v: any) => v ? new Date(v.seconds * 1000).toLocaleString() : '-',
    },
  ], [t]);

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        <WalletOutlined style={{ marginRight: 8, color: '#D4AF37' }} />
        {t('wallet.title')}
      </Title>

      <StatusResult loading={isLoading} error={error instanceof Error ? error.message : null} onRetry={refetch}>
        {wallet && (
          <Card size="small" style={{ marginBottom: 24 }}>
            <Descriptions column={3} size="small">
              <Descriptions.Item label={t('wallet.accountNumber')}>
                <span style={{ fontFamily: 'monospace', fontSize: 18, fontWeight: 700, color: '#D4AF37' }}>
                  {wallet.accountNumber || '-'}
                </span>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.balance')}>
                <span style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-text)' }}>
                  {formatAmount(wallet.balance)} {wallet.currency}
                </span>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.frozen')}>
                <span style={{ fontSize: 16, color: 'var(--color-text)' }}>
                  {formatAmount(wallet.frozenBalance)} {wallet.currency}
                </span>
              </Descriptions.Item>
            </Descriptions>
          </Card>
        )}
      </StatusResult>

      {/* USDT Deposit Section — HD wallet per-user address */}
      <Card
        size="small"
        style={{ marginBottom: 24, borderColor: '#D4AF37', borderWidth: 1 }}
        title={<span style={{ color: '#D4AF37' }}>USDT {t('wallet.deposit.title')}</span>}
      >
        <Spin spinning={addrLoading}>
          {depositAddr?.address ? (
            <div style={{ display: 'flex', gap: 24, alignItems: 'flex-start', flexWrap: 'wrap' }}>
              <div style={{ flexShrink: 0 }}>
                <QRCodeSVG
                  value={depositAddr.address}
                  size={160}
                  level="M"
                  includeMargin
                />
              </div>
              <div style={{ flex: 1, minWidth: 250 }}>
                <Descriptions column={1} size="small">
                  <Descriptions.Item label={t('wallet.deposit.network')}>
                    <Tag color="gold">{depositAddr.network || 'TRC20'}</Tag>
                  </Descriptions.Item>
                  <Descriptions.Item label={t('wallet.deposit.address')}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontFamily: 'monospace', fontSize: 14, wordBreak: 'break-all' }}>
                        {depositAddr.address}
                      </span>
                      <Tooltip title={t('wallet.deposit.copy')}>
                        <Button type="text" size="small" icon={<CopyOutlined />} onClick={copyAddress} />
                      </Tooltip>
                    </div>
                  </Descriptions.Item>
                </Descriptions>
                <Alert
                  type="warning"
                  message={t('wallet.deposit.notice')}
                  style={{ marginTop: 12 }}
                  showIcon
                />
              </div>
            </div>
          ) : !addrLoading && (
            <Alert
              type="info"
              message={t('wallet.deposit.notConfigured')}
              showIcon
            />
          )}
        </Spin>
      </Card>

      {/* Deposit History */}
      {myDeposits?.deposits?.length > 0 && (
        <Card
          title={t('wallet.deposit.history')}
          size="small"
          style={{ marginBottom: 24 }}
        >
          <Table
            columns={depositColumns}
            dataSource={myDeposits.deposits}
            rowKey="id"
            pagination={{ pageSize: 10, size: 'small' }}
            size="small"
          />
        </Card>
      )}

      <Card
        title={<span><TransactionOutlined style={{ marginRight: 8 }} />{t('wallet.transactions')}</span>}
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
