import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Typography, Tag, Popconfirm } from 'antd';
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { sreApi, type BreakerStatus } from './sreApi';

const { Text, Title } = Typography;

const stateColor: Record<string, string> = { closed: 'green', open: 'red', half_open: 'orange' };
const stateLabel: Record<string, string> = { closed: '正常', open: '已熔断', half_open: '半开(探测中)' };

export default function BreakersPage() {
  const [breakers, setBreakers] = useState<BreakerStatus[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchBreakers = useCallback(async () => {
    setLoading(true);
    try { setBreakers(await sreApi.breakersList()); } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchBreakers(); }, [fetchBreakers]);

  const handleReset = async (strategyId: string) => {
    await sreApi.breakerReset(strategyId);
    fetchBreakers();
  };

  const columns = [
    { title: '策略 ID', dataIndex: 'strategy_id', key: 'id', width: 200, render: (v: string) => <Text code>{v}</Text> },
    {
      title: '状态', dataIndex: 'state', key: 'state', width: 130,
      render: (v: string) => <Tag color={stateColor[v] || 'default'}>{stateLabel[v] || v}</Tag>,
    },
    { title: '总 P&L', dataIndex: 'total_pnl', key: 'pnl', width: 100, render: (v: number) => v.toFixed(2) },
    { title: '亏损 %', dataIndex: 'loss_percent', key: 'loss', width: 100, render: (v: number) => `${v.toFixed(2)}%` },
    { title: '交易数', dataIndex: 'trade_count', key: 'count', width: 80 },
    { title: '熔断时间', dataIndex: 'tripped_at', key: 'tripped', width: 160, render: (v: string) => v || '-' },
    { title: '熔断原因', dataIndex: 'trip_reason', key: 'reason', render: (v: string) => v || '-' },
    {
      title: '', key: 'actions', width: 100,
      render: (_: unknown, record: BreakerStatus) =>
        record.state !== 'closed' ? (
          <Popconfirm title="确认重置此熔断器？" onConfirm={() => handleReset(record.strategy_id)} okText="确认" cancelText="取消">
            <Button size="small" type="link">重置</Button>
          </Popconfirm>
        ) : null,
    },
  ];

  return (
    <div style={{ maxWidth: 960 }}>
      <Title level={4}><ThunderboltOutlined style={{ marginRight: 8 }} />Strategy Breakers</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        策略熔断器状态总览 — 自动检测异常亏损并熔断
      </Text>

      <Card size="small" extra={<Button icon={<ReloadOutlined />} onClick={fetchBreakers} loading={loading}>刷新</Button>}>
        <Table dataSource={breakers} columns={columns} rowKey="strategy_id"
          loading={loading} size="small" pagination={false}
          locale={{ emptyText: '暂无已注册的熔断器' }}
        />
      </Card>
    </div>
  );
}
