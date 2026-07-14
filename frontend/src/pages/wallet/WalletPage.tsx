import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Descriptions, Button, Modal, Input, InputNumber, Alert, message, Tooltip } from 'antd';
import { WalletOutlined, TransactionOutlined, PlusOutlined, CopyOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { walletApi } from '@/client/wallet';
import { depositApi } from '@/client/deposit';
import { queryKeys } from '@/queries/queryKeys';
import { StatusResult } from '@/components/common/StatusResult';
import { formatAmount } from '@/utils/amount';
import { useMemo, useState } from 'react';

const { Title, Text } = Typography;

export default function WalletPage() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [depositModalOpen, setDepositModalOpen] = useState(false);
  const [depositAmount, setDepositAmount] = useState<number | null>(null);
  const [depositTxHash, setDepositTxHash] = useState('');

  const { data: wallet, isLoading, error, refetch } = useQuery({
    queryKey: queryKeys.wallet.all,
    queryFn: () => walletApi.getWallet(),
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: queryKeys.wallet.transactions(),
    queryFn: () => walletApi.listTransactions(1, 50),
  });

  const { data: depositInfo } = useQuery({
    queryKey: queryKeys.deposit.info,
    queryFn: () => depositApi.getDepositInfo(),
  });

  const { data: myDeposits } = useQuery({
    queryKey: queryKeys.deposit.myDeposits,
    queryFn: () => depositApi.listMyDeposits(1, 20),
  });

  const createDepositMutation = useMutation({
    mutationFn: () => depositApi.createDeposit(String(depositAmount || 0), depositTxHash),
    onSuccess: () => {
      message.success(t('wallet.deposit.success', { defaultValue: 'Deposit request submitted. Please wait for admin review.' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.deposit.myDeposits });
      setDepositModalOpen(false);
      setDepositAmount(null);
      setDepositTxHash('');
    },
    onError: (err: Error) => {
      message.error(err.message || t('wallet.deposit.failed', { defaultValue: 'Failed to submit deposit request.' }));
    },
  });

  const copyAddress = () => {
    if (depositInfo?.receivingAddress) {
      navigator.clipboard.writeText(depositInfo.receivingAddress);
      message.success(t('wallet.deposit.addressCopied', { defaultValue: 'Address copied to clipboard' }));
    }
  };

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

  const depositColumns = useMemo(() => [
    {
      title: t('wallet.deposit.table.amount', { defaultValue: 'USDT Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
    },
    {
      title: t('wallet.deposit.table.amountUsd', { defaultValue: 'USD Credit' }),
      dataIndex: 'amountUsd',
      key: 'amountUsd',
      width: 120,
      render: (v: string) => <span style={{ color: '#00A651', fontWeight: 500 }}>+{formatAmount(v)}</span>,
    },
    {
      title: t('wallet.deposit.table.txHash', { defaultValue: 'Tx Hash' }),
      dataIndex: 'txHash',
      key: 'txHash',
      ellipsis: true,
      width: 200,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 20)}...</span> : '-',
    },
    {
      title: t('wallet.deposit.table.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { PENDING: 'orange', APPROVED: 'green', REJECTED: 'red' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('wallet.deposit.table.time', { defaultValue: 'Time' }),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: any) => v ? new Date(v.seconds * 1000).toLocaleString() : '-',
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

      {/* USDT Deposit Section */}
      <Card
        size="small"
        style={{ marginBottom: 24, borderColor: '#D4AF37', borderWidth: 1 }}
        title={<span style={{ color: '#D4AF37' }}>USDT {t('wallet.deposit.title', { defaultValue: 'Deposit' })}</span>}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => setDepositModalOpen(true)} disabled={!depositInfo?.receivingAddress}>
            {t('wallet.deposit.button', { defaultValue: 'New Deposit' })}
          </Button>
        }
      >
          <Descriptions column={2} size="small">
            <Descriptions.Item label={t('wallet.deposit.network', { defaultValue: 'Network' })}>
              <Tag color="gold">{depositInfo?.network || 'TRC20'}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label={t('wallet.deposit.exchangeRate', { defaultValue: 'Exchange Rate' })}>
              1 USDT = {depositInfo?.exchangeRate || '1'} USD
            </Descriptions.Item>
            <Descriptions.Item label={t('wallet.deposit.address', { defaultValue: 'Receiving Address' })} span={2}>
              {depositInfo?.receivingAddress ? (
                <>
                  <span style={{ fontFamily: 'monospace', fontSize: 14, wordBreak: 'break-all' }}>
                    {depositInfo.receivingAddress}
                  </span>
                  <Tooltip title={t('wallet.deposit.copy', { defaultValue: 'Copy' })}>
                    <Button type="text" size="small" icon={<CopyOutlined />} onClick={copyAddress} style={{ marginLeft: 8 }} />
                  </Tooltip>
                </>
              ) : (
                <Alert
                  type="info"
                  message={t('wallet.deposit.notConfigured', { defaultValue: 'USDT deposit is not yet configured. Please contact support.' })}
                  showIcon
                  style={{ marginTop: 0 }}
                />
              )}
            </Descriptions.Item>
          </Descriptions>
          {depositInfo?.receivingAddress && (
            <Alert
              type="warning"
              message={t('wallet.deposit.notice', { defaultValue: 'Only send USDT via the specified network. Sending other tokens or using a different network may result in permanent loss. After sending, submit a deposit request with the amount and optional tx hash for admin review.' })}
              style={{ marginTop: 12 }}
              showIcon
            />
          )}
      </Card>

      {/* Deposit History */}
      {myDeposits?.deposits?.length > 0 && (
        <Card
          title={t('wallet.deposit.history', { defaultValue: 'Deposit History' })}
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

      {/* Deposit Modal */}
      <Modal
        title={t('wallet.deposit.modalTitle', { defaultValue: 'Submit Deposit Request' })}
        open={depositModalOpen}
        onCancel={() => setDepositModalOpen(false)}
        onOk={() => createDepositMutation.mutate()}
        confirmLoading={createDepositMutation.isPending}
        okText={t('wallet.deposit.submit', { defaultValue: 'Submit' })}
      >
        <Descriptions column={1} size="small" style={{ marginBottom: 16 }}>
          <Descriptions.Item label={t('wallet.deposit.network', { defaultValue: 'Network' })}>
            <Tag color="gold">{depositInfo?.network || 'TRC20'}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('wallet.deposit.address', { defaultValue: 'Receiving Address' })}>
            <Text style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}>
              {depositInfo?.receivingAddress || '-'}
            </Text>
          </Descriptions.Item>
        </Descriptions>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500 }}>
            {t('wallet.deposit.amountLabel', { defaultValue: 'USDT Amount' })}
          </label>
          <InputNumber
            value={depositAmount}
            onChange={setDepositAmount}
            min={0.01}
            step={0.01}
            precision={8}
            style={{ width: '100%' }}
            placeholder="0.00"
          />
        </div>
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500 }}>
            {t('wallet.deposit.txHashLabel', { defaultValue: 'Transaction Hash (optional)' })}
          </label>
          <Input
            value={depositTxHash}
            onChange={(e) => setDepositTxHash(e.target.value)}
            placeholder="0x..."
            style={{ fontFamily: 'monospace' }}
          />
        </div>
        {depositAmount && depositInfo?.exchangeRate && (
          <Alert
            type="info"
            message={`${t('wallet.deposit.willCredit', { defaultValue: 'Will credit' })}: +$${(depositAmount * parseFloat(depositInfo.exchangeRate)).toFixed(2)} USD`}
            style={{ marginTop: 8 }}
          />
        )}
      </Modal>
    </div>
  );
}
