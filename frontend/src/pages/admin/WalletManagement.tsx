import { useState, useMemo } from 'react';
import {
  Card, Table, Input, Button, Select, InputNumber, message, Space, Tag, Descriptions, Modal, Divider, Typography
} from 'antd';
import { WalletOutlined, SearchOutlined, PlusOutlined, CalculatorOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { walletApi } from '@/client/wallet';
import { aiGatewayApi } from '@/client/aiGateway';
import { formatDateTime } from '@/utils/date';

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
  // Bidirectional calculator: change one, auto-update the other
  const [calcModel, setCalcModel] = useState<string | undefined>();
  const [calcUSD, setCalcUSD] = useState<string>('0.10');
  const [calcTokens, setCalcTokens] = useState<string>('714285');
  const [editingField, setEditingField] = useState<'usd' | 'tokens'>('usd');

  const { data: systemModels } = useQuery({
    queryKey: ['admin', 'ai-gateway', 'models'],
    queryFn: () => aiGatewayApi.listSystemModels().catch(() => []),
  });

  const modelOptions = useMemo(() =>
    (systemModels || []).map(m => ({
      value: m.id,
      label: `${m.displayName || m.modelName} ($${parseFloat(m.pricePer1mInput).toFixed(2)}/1M)`,
      pricePerToken: (parseFloat(m.pricePer1mInput) + parseFloat(m.pricePer1mOutput)) / 2 / 1000000,
    })),
    [systemModels]);

  const selectedModel = modelOptions.find(m => m.value === calcModel);

  // Sync: when USD changes → compute tokens; when tokens change → compute USD
  const handleUSDChange = (v: string) => {
    setCalcUSD(v);
    setEditingField('usd');
    const usd = parseFloat(v) || 0;
    if (selectedModel && usd > 0) {
      setCalcTokens(Math.round(usd / selectedModel.pricePerToken).toString());
    }
  };

  const handleTokensChange = (v: string) => {
    setCalcTokens(v);
    setEditingField('tokens');
    const tokens = parseInt(v) || 0;
    if (selectedModel && tokens > 0) {
      setCalcUSD((tokens * selectedModel.pricePerToken).toFixed(8));
    }
  };

  const handleModelChange = (modelId: string) => {
    setCalcModel(modelId);
    const m = modelOptions.find(x => x.value === modelId);
    if (m) {
      if (editingField === 'usd') {
        const usd = parseFloat(calcUSD) || 0;
        if (usd > 0) setCalcTokens(Math.round(usd / m.pricePerToken).toString());
      } else {
        const tokens = parseInt(calcTokens) || 0;
        if (tokens > 0) setCalcUSD((tokens * m.pricePerToken).toFixed(8));
      }
    }
  };

  const fillAdjust = () => {
    setAdjustAmount(parseFloat(calcUSD) || 0);
    const t = parseInt(calcTokens) || 0;
    const label = selectedModel?.label?.split(' ')[0] || '';
    setAdjustDesc(`${t >= 1000 ? (t / 1000).toFixed(0) + 'K' : t} tokens (${label})`);
  };

  // Always fetch user list; search filters on the server side
  const { data: userData, isFetching: searching } = useQuery({
    queryKey: ['admin', 'wallet', 'users', search || ''],
    queryFn: () => walletApi.searchUsers(search || ''),
  });

  const users = userData || [];

  // Selected user's wallet
  const { data: wallet, isLoading: walletLoading } = useQuery({
    queryKey: ['admin', 'wallet', 'detail', selectedUserId],
    queryFn: () => walletApi.getWallet(selectedUserId!),
    enabled: !!selectedUserId,
  });

  // Transactions
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

      {/* User list + search */}
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

      {/* Wallet detail */}
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
          {/* Token ↔ USD 双向换算器 */}
          <div style={{ background: '#f6ffed', borderRadius: 8, padding: 12, border: '1px solid #b7eb8f' }}>
            <div style={{ fontSize: 12, color: '#52c41a', marginBottom: 8, fontWeight: 500 }}>
              <CalculatorOutlined /> Token ↔ USD 换算
            </div>
            <Select
              showSearch
              value={calcModel}
              onChange={handleModelChange}
              style={{ width: '100%', marginBottom: 12 }}
              placeholder="选择模型（定价基准）"
              options={modelOptions}
              filterOption={(input, option) =>
                (option?.label as string || '').toLowerCase().includes(input.toLowerCase())
              }
            />
            {selectedModel && (
              <Space style={{ width: '100%' }} size={12}>
                <div style={{ flex: 1 }}>
                  <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>USD 金额</Text>
                  <InputNumber
                    value={calcUSD}
                    onChange={(v) => handleUSDChange(String(v || '0'))}
                    min={0}
                    step={0.01}
                    style={{ width: '100%' }}
                    addonBefore="$"
                    precision={8}
                  />
                </div>
                <div style={{ textAlign: 'center', paddingTop: 18 }}>
                  <Text type="secondary">⇄</Text>
                </div>
                <div style={{ flex: 1 }}>
                  <Text type="secondary" style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>Token 数量</Text>
                  <InputNumber
                    value={calcTokens}
                    onChange={(v) => handleTokensChange(String(v || '0'))}
                    min={0}
                    step={10000}
                    style={{ width: '100%' }}
                    addonAfter="tokens"
                  />
                </div>
              </Space>
            )}
          </div>

          <Divider style={{ margin: 0 }} />

          {/* 实际调整 */}
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
            <Button size="small" icon={<CalculatorOutlined />} onClick={fillAdjust}>
              填入换算结果
            </Button>
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
