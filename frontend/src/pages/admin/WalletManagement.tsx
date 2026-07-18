import { useState } from 'react';
import {
  Card, Table, Input, Button, Select, InputNumber, message, Space, Tag, Descriptions, Modal, Typography, Tabs
} from 'antd';
import { WalletOutlined, SearchOutlined, PlusOutlined } from '@ant-design/icons';
import DepositAddressesTab from './DepositAddressesTab';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { walletApi } from '@/client/wallet';
import { formatDateTime } from '@/utils/date';
import WalletCalculator from './WalletCalculator';

const { Title, Text } = Typography;

export default function WalletManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedUserInfo, setSelectedUserInfo] = useState<{ email: string; accountNumber?: string }>({ email: '' });
  const [adjustOpen, setAdjustOpen] = useState(false);
  const [adjustAmount, setAdjustAmount] = useState<number>(0);
  const [adjustDesc, setAdjustDesc] = useState('');
  const [adjustType, setAdjustType] = useState<'add' | 'deduct'>('add');

  const { data: userData, isFetching: searching } = useQuery({
    queryKey: ['admin', 'wallet', 'users', search || ''],
    queryFn: () => walletApi.searchUsers(search || ''),
  });

  const users = userData || [];

  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ['admin', 'wallet', 'detail', selectedUserId],
    queryFn: () => walletApi.getWallet(selectedUserId!),
    enabled: !!selectedUserId,
  });

  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: ['admin', 'wallet', 'transactions', selectedUserId],
    queryFn: () => walletApi.listTransactions(1, 50, selectedUserId!),
    enabled: !!selectedUserId,
  });

  const adjustMutation = useMutation({
    mutationFn: async () => {
      if (!selectedUserId) throw new Error(t('admin.wallet.errors.noUserSelected', { defaultValue: 'No user selected' }));
      const amount = adjustType === 'add' ? String(adjustAmount) : `-${adjustAmount}`;
      return walletApi.adjustBalance(selectedUserId, amount, adjustDesc);
    },
    onSuccess: () => {
      message.success(t('admin.wallet.messages.adjustSuccess', { defaultValue: 'Balance adjusted successfully' }));
      setAdjustOpen(false);
      setAdjustAmount(0);
      setAdjustDesc('');
      queryClient.invalidateQueries({ queryKey: ['admin', 'wallet'] });
    },
    onError: (err: Error) => {
      message.error(err.message || t('admin.wallet.messages.adjustFailed', { defaultValue: 'Adjustment failed' }));
    },
  });

  const selectUser = (u: { id: string; email: string; accountNumber?: string }) => {
    setSelectedUserId(u.id);
    setSelectedUserInfo({ email: u.email, accountNumber: u.accountNumber });
  };

  const handleFillAdjust = (usd: string, tokens: string, modelLabel: string) => {
    setAdjustAmount(parseFloat(usd) || 0);
    const tokenCount = parseInt(tokens) || 0;
    setAdjustDesc(`${tokenCount >= 1000 ? (tokenCount / 1000).toFixed(0) + 'K' : tokenCount} tokens (${modelLabel})`);
  };

  const userColumns = [
    {
      title: t('admin.wallet.columns.walletNumber', { defaultValue: 'Wallet No.' }),
      dataIndex: 'accountNumber',
      key: 'accountNumber',
      width: 120,
      render: (v: string | undefined) =>
        v ? <Tag color="blue" style={{ fontFamily: 'monospace' }}>{v}</Tag> : <Tag color="default">{t('admin.wallet.unassigned', { defaultValue: 'Unassigned' })}</Tag>,
    },
    {
      title: t('admin.wallet.columns.email', { defaultValue: 'Email' }),
      dataIndex: 'email',
      key: 'email',
      ellipsis: true,
      render: (v: string) => <Text>{v}</Text>,
    },
    {
      title: t('admin.wallet.columns.nickname', { defaultValue: 'Nickname' }),
      dataIndex: 'nickname',
      key: 'nickname',
      width: 120,
      render: (v: string | undefined) => <Text type="secondary">{v || '—'}</Text>,
    },
  ];

  const txColumns = [
    { title: t('admin.wallet.columns.type', { defaultValue: 'Type' }), dataIndex: 'txType', key: 'txType', width: 100,
      render: (v: string) => {
        const m: Record<string, string> = { deposit: 'green', withdrawal: 'red', adjustment: 'blue', ai_usage: 'purple', fee: 'orange' };
        return <Tag color={m[v] || 'default'}>{v}</Tag>;
      },
    },
    { title: t('admin.wallet.columns.amount', { defaultValue: 'Amount' }), dataIndex: 'amount', key: 'amount', width: 140,
      render: (v: string) => <span style={{ color: v.startsWith('-') ? '#ef4444' : '#22c55e', fontWeight: 500 }}>{v.startsWith('-') ? v : `+${v}`}</span>,
    },
    { title: t('admin.wallet.columns.balanceAfter', { defaultValue: 'Balance After' }), dataIndex: 'balanceAfter', key: 'balanceAfter', width: 140 },
    { title: t('admin.wallet.columns.description', { defaultValue: 'Description' }), dataIndex: 'description', key: 'description', ellipsis: true },
    { title: t('admin.wallet.columns.time', { defaultValue: 'Time' }), dataIndex: 'createdAtTsMs', key: 'createdAtTsMs', width: 170,
      render: (v: unknown) => formatDateTime(String(v || '')),
    },
  ];

  return (
    <div className="space-y-4">
      <Title level={4}>
        <WalletOutlined /> {t('admin.wallet.title', { defaultValue: 'Wallet Management' })}
      </Title>

      <Tabs
        defaultActiveKey="wallets"
        items={[
          {
            key: 'wallets',
            label: t('admin.wallet.tabWallets', { defaultValue: 'User Wallets' }),
            children: (
              <>
      <Card size="small" title={t('admin.wallet.userList', { defaultValue: 'User List' })}>
        <Input
          prefix={<SearchOutlined />}
          placeholder={t('admin.wallet.searchPlaceholder', { defaultValue: 'Search wallet / email / nickname' })}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setSelectedUserId(null); }}
          style={{ width: 360, marginBottom: 12 }}
          allowClear
        />
        <Table
          columns={userColumns}
          dataSource={users}
          rowKey="id"
          loading={searching}
          size="small"
          pagination={{ pageSize: 15, showTotal: (total) => t('admin.wallet.totalUsers', { total, defaultValue: `${total} users` }) }}
          onRow={(record) => ({
            onClick: () => selectUser(record),
            style: {
              cursor: 'pointer',
              background: selectedUserId === record.id ? '#e6f4ff' : undefined,
            },
          })}
          locale={{ emptyText: search ? t('admin.wallet.noMatch', { defaultValue: 'No matching users' }) : t('admin.wallet.noUsers', { defaultValue: 'No users' }) }}
        />
      </Card>

      {selectedUserId && (
        <>
          <Card
            size="small"
            title={t('admin.wallet.walletDetail', { defaultValue: 'Wallet Detail' }) + `: ${selectedUserInfo.accountNumber ? selectedUserInfo.accountNumber + ' · ' : ''}${selectedUserInfo.email}`}
            loading={walletLoading}
          >
            <Descriptions column={4} size="small">
              <Descriptions.Item label={t('admin.wallet.columns.walletNumber', { defaultValue: 'Wallet No.' })}>
                {wallet?.accountNumber ? <Tag color="blue">{wallet.accountNumber}</Tag> : '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('admin.wallet.columns.balance', { defaultValue: 'Balance' })}>
                <Text strong style={{ color: '#22c55e', fontSize: 16 }}>{wallet?.balance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('admin.wallet.columns.frozen', { defaultValue: 'Frozen' })}>
                <Text type="secondary">{wallet?.frozenBalance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('admin.wallet.columns.currency', { defaultValue: 'Currency' })}>
                {wallet?.currency || 'USD'}
              </Descriptions.Item>
            </Descriptions>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setAdjustOpen(true)}
              style={{ marginTop: 12 }}
            >
              {t('admin.wallet.adjustBalance', { defaultValue: 'Adjust Balance (Add / Deduct)' })}
            </Button>
          </Card>

          <Card size="small" title={t('admin.wallet.transactions', { defaultValue: 'Transactions' })}>
            <Table
              columns={txColumns}
              dataSource={txData?.transactions || []}
              rowKey="id"
              loading={txLoading}
              size="small"
              pagination={{ pageSize: 20 }}
            />
          </Card>
        </>
      )}

      <Modal
        open={adjustOpen}
        title={t('admin.wallet.adjustBalance', { defaultValue: 'Adjust Balance' })}
        onCancel={() => setAdjustOpen(false)}
        onOk={() => adjustMutation.mutate()}
        confirmLoading={adjustMutation.isPending}
        okText={t('common.confirm', { defaultValue: 'Confirm' })}
        cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
        width={520}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <WalletCalculator onFillAdjust={handleFillAdjust} />
          <Space>
            <Select
              value={adjustType}
              onChange={(v) => setAdjustType(v)}
              style={{ width: 100 }}
              options={[
                { label: t('admin.wallet.add', { defaultValue: 'Add' }), value: 'add' },
                { label: t('admin.wallet.deduct', { defaultValue: 'Deduct' }), value: 'deduct' },
              ]}
            />
            <InputNumber
              value={adjustAmount}
              onChange={(v) => setAdjustAmount(v || 0)}
              min={0}
              precision={8}
              style={{ width: 200 }}
              placeholder="0.00"
              addonAfter="USD"
            />
          </Space>
          <Input
            placeholder={t('admin.wallet.adjustReason', { defaultValue: 'Reason' })}
            value={adjustDesc}
            onChange={(e) => setAdjustDesc(e.target.value)}
          />
        </Space>
      </Modal>
              </>
            ),
          },
          {
            key: 'deposit-addresses',
            label: t('admin.wallet.tabDepositAddresses', { defaultValue: 'Deposit Addresses' }),
            children: <DepositAddressesTab />,
          },
        ]}
      />
    </div>
  );
}
