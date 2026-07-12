import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Typography, Tag, Popconfirm } from 'antd';
import { ReloadOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { sreApi, type BreakerStatus } from './sreApi';

const { Text, Title } = Typography;

const stateColor: Record<string, string> = { closed: 'green', open: 'red', half_open: 'orange' };

export default function BreakersPage() {
  const { t } = useTranslation();
  const stateLabel: Record<string, string> = {
    closed: t('sre.breakers.stateClosed', { defaultValue: 'Normal' }),
    open: t('sre.breakers.stateOpen', { defaultValue: 'Tripped' }),
    half_open: t('sre.breakers.stateHalfOpen', { defaultValue: 'Half-Open (probing)' }),
  };
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
    { title: t('sre.breakers.columns.strategyId', { defaultValue: 'Strategy ID' }), dataIndex: 'strategy_id', key: 'id', width: 200, render: (v: string) => <Text code>{v}</Text> },
    {
      title: t('sre.breakers.columns.state', { defaultValue: 'State' }), dataIndex: 'state', key: 'state', width: 130,
      render: (v: string) => <Tag color={stateColor[v] || 'default'}>{stateLabel[v] || v}</Tag>,
    },
    { title: t('sre.breakers.columns.totalPnl', { defaultValue: 'Total P&L' }), dataIndex: 'total_pnl', key: 'pnl', width: 100, render: (v: number) => (v ?? 0).toFixed(2) },
    { title: t('sre.breakers.columns.lossPercent', { defaultValue: 'Loss %' }), dataIndex: 'loss_percent', key: 'loss', width: 100, render: (v: number) => `${(v ?? 0).toFixed(2)}%` },
    { title: t('sre.breakers.columns.tradeCount', { defaultValue: 'Trades' }), dataIndex: 'trade_count', key: 'count', width: 80 },
    { title: t('sre.breakers.columns.trippedAt', { defaultValue: 'Tripped At' }), dataIndex: 'tripped_at', key: 'tripped', width: 160, render: (v: string) => v || '-' },
    { title: t('sre.breakers.columns.tripReason', { defaultValue: 'Trip Reason' }), dataIndex: 'trip_reason', key: 'reason', render: (v: string) => v || '-' },
    {
      title: '', key: 'actions', width: 100,
      render: (_: unknown, record: BreakerStatus) =>
        record.state !== 'closed' ? (
          <Popconfirm title={t('sre.breakers.confirmReset', { defaultValue: 'Reset this breaker?' })} onConfirm={() => handleReset(record.strategy_id)} okText={t('common.confirm', { defaultValue: 'Confirm' })} cancelText={t('common.cancel', { defaultValue: 'Cancel' })}>
            <Button size="small" type="link">{t('common.reset', { defaultValue: 'Reset' })}</Button>
          </Popconfirm>
        ) : null,
    },
  ];

  return (
    <div style={{ maxWidth: 960 }}>
      <Title level={4}><ThunderboltOutlined style={{ marginRight: 8 }} />Strategy Breakers</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        {t('sre.breakers.description', { defaultValue: 'Strategy breaker status overview — auto-detects abnormal losses and trips' })}
      </Text>

      <Card size="small" extra={<Button icon={<ReloadOutlined />} onClick={fetchBreakers} loading={loading}>{t('common.refresh', { defaultValue: 'Refresh' })}</Button>}>
        <Table dataSource={breakers} columns={columns} rowKey="strategy_id"
          loading={loading} size="small" pagination={false}
          locale={{ emptyText: t('sre.breakers.noBreakers', { defaultValue: 'No registered breakers' }) }}
        />
      </Card>
    </div>
  );
}
