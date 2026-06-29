import { useState } from 'react';
import {
  Card, Table, Input, Button, Select, InputNumber, message, Space, Tag, Descriptions, Modal, Typography
} from 'antd';
import { WalletOutlined, SearchOutlined, PlusOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { walletApi } from '@/client/wallet';
import { formatDateTime } from '@/utils/date';
import WalletCalculator from './WalletCalculator';

const { Title, Text } = Typography;

export default function WalletManagement() {
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
      if (!selectedUserId) throw new Error('未选择用户');
      const amount = adjustType === 'add' ? String(adjustAmount) : `-${adjustAmount}`;
      return walletApi.adjustBalance(selectedUserId, amount, adjustDesc);
    },
    onSuccess: () => {
      message.success('余额调整成功');
      setAdjustOpen(false);
      setAdjustAmount(0);
      setAdjustDesc('');
      queryClient.invalidateQueries({ queryKey: ['admin', 'wallet'] });
    },
    onError: (err: Error) => {
      message.error(err.message || '调整失败');
    },
  });

  const selectUser = (u: { id: string; email: string; accountNumber?: string }) => {
    setSelectedUserId(u.id);
    setSelectedUserInfo({ email: u.email, accountNumber: u.accountNumber });
  };

  const handleFillAdjust = (usd: string, tokens: string, modelLabel: string) => {
    setAdjustAmount(parseFloat(usd) || 0);
    const t = parseInt(tokens) || 0;
    setAdjustDesc(`${t >= 1000 ? (t / 1000).toFixed(0) + 'K' : t} tokens (${modelLabel})`);
  };

  const userColumns = [
    {
      title: '钱包号',
      dataIndex: 'accountNumber',
      key: 'accountNumber',
      width: 120,
      render: (v: string | undefined) =>
        v ? <Tag color="blue" style={{ fontFamily: 'monospace' }}>{v}</Tag> : <Tag color="default">未分配</Tag>,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      ellipsis: true,
      render: (v: string) => <Text>{v}</Text>,
    },
    {
      title: '昵称',
      dataIndex: 'nickname',
      key: 'nickname',
      width: 120,
      render: (v: string | undefined) => <Text type="secondary">{v || '—'}</Text>,
    },
  ];

  const txColumns = [
    { title: '类型', dataIndex: 'txType', key: 'txType', width: 100,
      render: (v: string) => {
        const m: Record<string, string> = { deposit: 'green', withdrawal: 'red', adjustment: 'blue', ai_usage: 'purple', fee: 'orange' };
        return <Tag color={m[v] || 'default'}>{v}</Tag>;
      },
    },
    { title: '金额', dataIndex: 'amount', key: 'amount', width: 140,
      render: (v: string) => <span style={{ color: v.startsWith('-') ? '#ef4444' : '#22c55e', fontWeight: 500 }}>{v.startsWith('-') ? v : `+${v}`}</span>,
    },
    { title: '变动后余额', dataIndex: 'balanceAfter', key: 'balanceAfter', width: 140 },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '时间', dataIndex: 'createdAtTsMs', key: 'createdAtTsMs', width: 170,
      render: (v: unknown) => formatDateTime(String(v || '')),
    },
  ];

  return (
    <div className="space-y-4">
      <Title level={4}>
        <WalletOutlined /> 钱包管理
      </Title>

      <Card size="small" title="用户列表">
        <Input
          prefix={<SearchOutlined />}
          placeholder="搜索钱包号 / 邮箱 / 昵称"
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
          pagination={{ pageSize: 15, showTotal: (t) => `${t} 个用户` }}
          onRow={(record) => ({
            onClick: () => selectUser(record),
            style: {
              cursor: 'pointer',
              background: selectedUserId === record.id ? '#e6f4ff' : undefined,
            },
          })}
          locale={{ emptyText: search ? '未找到匹配用户' : '暂无用户' }}
        />
      </Card>

      {selectedUserId && (
        <>
          <Card
            size="small"
            title={`钱包详情: ${selectedUserInfo.accountNumber ? selectedUserInfo.accountNumber + ' · ' : ''}${selectedUserInfo.email}`}
            loading={walletLoading}
          >
            <Descriptions column={4} size="small">
              <Descriptions.Item label="钱包号">
                {wallet?.accountNumber ? <Tag color="blue">{wallet.accountNumber}</Tag> : '—'}
              </Descriptions.Item>
              <Descriptions.Item label="余额">
                <Text strong style={{ color: '#22c55e', fontSize: 16 }}>{wallet?.balance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="冻结">
                <Text type="secondary">{wallet?.frozenBalance || '0'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="币种">
                {wallet?.currency || 'USD'}
              </Descriptions.Item>
            </Descriptions>
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => setAdjustOpen(true)}
              style={{ marginTop: 12 }}
            >
              调整余额（赠送 / 扣除）
            </Button>
          </Card>

          <Card size="small" title="交易记录">
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
        title="调整余额"
        onCancel={() => setAdjustOpen(false)}
        onOk={() => adjustMutation.mutate()}
        confirmLoading={adjustMutation.isPending}
        okText="确认"
        cancelText="取消"
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
                { label: '赠送', value: 'add' },
                { label: '扣除', value: 'deduct' },
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
            placeholder="调整原因"
            value={adjustDesc}
            onChange={(e) => setAdjustDesc(e.target.value)}
          />
        </Space>
      </Modal>
    </div>
  );
}
