import { useState } from 'react';
import {
  Card, Table, Input, Button, Select, InputNumber, message, Space, Tag, Descriptions, Modal
} from 'antd';
import { WalletOutlined, SearchOutlined, PlusOutlined } from '@ant-design/icons';
import { Typography } from 'antd';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { walletApi } from '@/client/wallet';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '@/utils/date';

const { Title, Text } = Typography;

export default function WalletManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [search, setSearch] = useState('');
  const [selectedUserId, setSelectedUserId] = useState<string | null>(null);
  const [selectedUserEmail, setSelectedUserEmail] = useState<string>('');
  const [adjustOpen, setAdjustOpen] = useState(false);
  const [adjustAmount, setAdjustAmount] = useState<number>(0);
  const [adjustDesc, setAdjustDesc] = useState('');
  const [adjustType, setAdjustType] = useState<'add' | 'deduct'>('add');

  // Search users
  const { data: users, isFetching: searching } = useQuery({
    queryKey: ['admin', 'wallet', 'search', search],
    queryFn: () => walletApi.searchUsers(search),
    enabled: search.length >= 2,
  });

  // Selected user's wallet (only when a user is selected)
  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ['admin', 'wallet', 'detail', selectedUserId],
    queryFn: () => walletApi.getWallet(selectedUserId!),
    enabled: !!selectedUserId,
  });

  // Transactions for selected user
  const { data: txData, isLoading: txLoading } = useQuery({
    queryKey: ['admin', 'wallet', 'transactions', selectedUserId],
    queryFn: () => walletApi.listTransactions(1, 50, selectedUserId!),
    enabled: !!selectedUserId,
  });

  const adjustMutation = useMutation({
    mutationFn: async () => {
      if (!selectedUserId) throw new Error('No user selected');
      const amount = adjustType === 'add' ? String(adjustAmount) : `-${adjustAmount}`;
      return walletApi.adjustBalance(selectedUserId, amount, adjustDesc);
    },
    onSuccess: () => {
      message.success(t('admin.wallet.adjustSuccess', { defaultValue: 'Balance adjusted' }));
      setAdjustOpen(false);
      setAdjustAmount(0);
      setAdjustDesc('');
      queryClient.invalidateQueries({ queryKey: ['admin', 'wallet'] });
    },
    onError: (err: Error) => {
      message.error(err.message || t('admin.wallet.adjustFailed', { defaultValue: 'Adjustment failed' }));
    },
  });

  const txColumns = [
    {
      title: t('wallet.table.type', { defaultValue: 'Type' }),
      dataIndex: 'txType',
      key: 'txType',
      width: 100,
      render: (v: string) => {
        const colorMap: Record<string, string> = {
          deposit: 'green', withdrawal: 'red', adjustment: 'blue', fee: 'orange',
        };
        return <Tag color={colorMap[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('wallet.table.amount', { defaultValue: 'Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 140,
      render: (v: string) => (
        <span style={{ color: v.startsWith('-') ? '#ef4444' : '#22c55e', fontWeight: 500 }}>
          {v.startsWith('-') ? v : `+${v}`}
        </span>
      ),
    },
    {
      title: t('wallet.table.balanceAfter', { defaultValue: 'Balance After' }),
      dataIndex: 'balanceAfter',
      key: 'balanceAfter',
      width: 140,
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
      render: (v: string) => formatDateTime(v),
    },
  ];

  const selectUser = (user: { id: string; email: string }) => {
    setSelectedUserId(user.id);
    setSelectedUserEmail(user.email);
  };

  return (
    <div className="space-y-4">
      <Title level={4}>
        <WalletOutlined size={18} /> {t('admin.wallet.title', { defaultValue: 'Wallet Management' })}
      </Title>

      {/* User search */}
      <Card size="small">
        <Input
          prefix={<SearchOutlined />}
          placeholder={t('admin.wallet.searchPlaceholder', { defaultValue: 'Search by email or account number...' })}
          value={search}
          onChange={(e) => { setSearch(e.target.value); setSelectedUserId(null); }}
          style={{ width: 360 }}
          allowClear
        />

        {search.length >= 2 && (
          <div className="mt-3">
            {searching ? (
              <Text type="secondary">{t('common.searching')}</Text>
            ) : users && users.length > 0 ? (
              <Space direction="vertical" style={{ width: '100%' }}>
                {users.map((u: any) => (
                  <Card
                    key={u.id}
                    size="small"
                    hoverable
                    onClick={() => selectUser(u)}
                    style={{
                      cursor: 'pointer',
                      borderColor: selectedUserId === u.id ? '#1677ff' : undefined,
                      borderWidth: selectedUserId === u.id ? 2 : 1,
                    }}
                  >
                    <Space>
                      <Text strong>{u.email}</Text>
                      {u.nickname && <Text type="secondary">({u.nickname})</Text>}
                      {u.accountNumber && <Tag color="blue">{u.accountNumber}</Tag>}
                    </Space>
                  </Card>
                ))}
              </Space>
            ) : (
              <Text type="secondary">{t('admin.wallet.noUsers', { defaultValue: 'No users found' })}</Text>
            )}
          </div>
        )}
      </Card>

      {/* Wallet detail */}
      {selectedUserId && (
        <>
          <Card
            size="small"
            title={`${t('admin.wallet.walletFor', { defaultValue: 'Wallet for' })}: ${selectedUserEmail}`}
            loading={walletLoading}
          >
            <Descriptions column={4} size="small">
              <Descriptions.Item label={t('admin.wallet.accountNumber', { defaultValue: 'Account' })}>
                {wallet?.accountNumber ? <Tag color="blue">{wallet.accountNumber}</Tag> : '—'}
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.balance', { defaultValue: 'Balance' })}>
                <Text strong style={{ color: '#22c55e', fontSize: 16 }}>{wallet?.balance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.frozenBalance', { defaultValue: 'Frozen' })}>
                <Text type="secondary">{wallet?.frozenBalance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label={t('wallet.currency', { defaultValue: 'Currency' })}>
                {wallet?.currency || 'USD'}
              </Descriptions.Item>
            </Descriptions>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setAdjustOpen(true)}
              style={{ marginTop: 12 }}
            >
              {t('admin.wallet.adjustBalance', { defaultValue: 'Adjust Balance' })}
            </Button>
          </Card>

          {/* Transaction history */}
          <Card size="small" title={t('wallet.transactions', { defaultValue: 'Transactions' })}>
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

      {/* Adjust modal */}
      <Modal
        open={adjustOpen}
        title={t('admin.wallet.adjustBalance', { defaultValue: 'Adjust Balance' })}
        onCancel={() => setAdjustOpen(false)}
        onOk={() => adjustMutation.mutate()}
        confirmLoading={adjustMutation.isPending}
        okText={t('common.confirm')}
        cancelText={t('common.cancel')}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
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
            />
          </Space>
          <Input
            placeholder={t('admin.wallet.reason', { defaultValue: 'Reason for adjustment...' })}
            value={adjustDesc}
            onChange={(e) => setAdjustDesc(e.target.value)}
          />
        </Space>
      </Modal>
    </div>
  );
}
