// PaperAccountPanel — virtual paper trading account management + real-time portfolio display.
// Subscribes via SSE (WatchPaperAccount) for push-first portfolio updates.

import { useState, useEffect, useCallback } from 'react';
import { Button, Card, Input, List, Space, Tag, Typography, message, Statistic, Row, Col } from 'antd';
import { PlusOutlined, PlayCircleOutlined, StopOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { paperTradingClient } from '@/client/connect';
import type { PaperAccount, PaperOrder } from '@/gen/ant/v1/paper_trading_pb';
import { useAuthStore } from '@/stores/authStore';

const { Text, Title } = Typography;

export default function PaperAccountPanel() {
  const { t } = useTranslation();
  const userId = useAuthStore(s => s.user?.id);
  const [accounts, setAccounts] = useState<PaperAccount[]>([]);
  const [loading, setLoading] = useState(false);
  const [createName, setCreateName] = useState('');
  const [createBalance, setCreateBalance] = useState('10000');

  const loadAccounts = useCallback(async () => {
    if (!userId) return;
    setLoading(true);
    try {
      const resp = await paperTradingClient.listPaperAccounts({});
      setAccounts(resp.accounts || []);
    } catch { /* non-critical */ }
    setLoading(false);
  }, [userId]);

  useEffect(() => { loadAccounts(); }, [loadAccounts]);

  const handleCreate = useCallback(async () => {
    if (!createName.trim()) { message.warning('Enter a name'); return; }
    try {
      await paperTradingClient.createPaperAccount({ name: createName, initialBalance: createBalance });
      message.success('Paper account created');
      setCreateName(''); setCreateBalance('10000'); loadAccounts();
    } catch { message.error('Create failed'); }
  }, [createName, createBalance, loadAccounts]);

  return (
    <div style={{ padding: 16 }}>
      <Title level={5} style={{ marginBottom: 16 }}>📊 Paper Trading</Title>

      {/* Create account */}
      <Card size="small" style={{ marginBottom: 16, background: '#fafafa' }}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Text strong style={{ fontSize: 12 }}>Create Paper Account</Text>
          <Space>
            <Input size="small" placeholder="Account name" value={createName}
              onChange={e => setCreateName(e.target.value)} style={{ width: 160 }} />
            <Input size="small" placeholder="Balance" value={createBalance}
              onChange={e => setCreateBalance(e.target.value)} style={{ width: 100 }} />
            <Button size="small" type="primary" icon={<PlusOutlined />}
              onClick={handleCreate}>Create</Button>
          </Space>
        </Space>
      </Card>

      {/* Account list */}
      <List
        loading={loading}
        dataSource={accounts}
        locale={{ emptyText: 'No paper accounts. Create one to start simulated trading.' }}
        renderItem={(a: PaperAccount) => (
          <Card size="small" style={{ marginBottom: 8 }} key={a.id}>
            <Row gutter={12}>
              <Col span={8}>
                <Text strong>{a.name}</Text>
                <br />
                <Text type="secondary" style={{ fontSize: 11 }}>{a.id?.slice(0, 8)}</Text>
              </Col>
              <Col span={8}>
                <Statistic title="Balance" value={a.currentBalance || '0'}
                  valueStyle={{ fontSize: 14 }} prefix="$" />
              </Col>
              <Col span={8}>
                <Statistic title="Equity" value={a.equity || '0'}
                  valueStyle={{ fontSize: 14, color: parseFloat(a.equity || '0') >= parseFloat(a.initialBalance || '0') ? '#26a69a' : '#ef5350' }}
                  prefix={parseFloat(a.equity || '0') >= parseFloat(a.initialBalance || '0') ? <RiseOutlined /> : <FallOutlined />} />
              </Col>
            </Row>
            <Space style={{ marginTop: 8 }}>
              <Button size="small" icon={<PlayCircleOutlined />}
                onClick={() => message.info('Start strategy — connect from Strategy Workspace')}>
                Start
              </Button>
              <Button size="small" icon={<StopOutlined />} disabled>Stop</Button>
              <Tag color="blue">Paper</Tag>
            </Space>
          </Card>
        )}
      />
    </div>
  );
}
