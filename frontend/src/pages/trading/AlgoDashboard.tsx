import { useState, useCallback } from 'react';
import { Card, Table, Tag, Button, Space, Typography, Empty } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ColumnsType } from 'antd/es/table';
import { executionAlgoClient } from '@/client/connect';
import AlgoSubmitForm from '@/components/trade/AlgoSubmitForm';

const { Title } = Typography;

interface ActiveAlgo {
  executionId: string;
  state: string;
  algo: string;
  submittedSlices: number;
  totalSlices: number;
  symbol: string;
  side: string;
  volume: number;
}

export default function AlgoDashboard() {
  const { t } = useTranslation();
  const [algos, setAlgos] = useState<ActiveAlgo[]>([]);
  const [loading, setLoading] = useState(false);

  const refreshStatus = useCallback(async (executionId: string) => {
    try {
      const resp = await executionAlgoClient.getAlgoStatus({ executionId });
      return resp;
    } catch {
      return null;
    }
  }, []);

  const refreshAll = useCallback(async () => {
    setLoading(true);
    const updated = await Promise.all(
      algos.map(async (a) => {
        const status = await refreshStatus(a.executionId);
        if (!status) return a;
        return {
          ...a,
          state: status.state,
          submittedSlices: status.submittedSlices,
          totalSlices: status.totalSlices,
        };
      })
    );
    setAlgos(updated.filter(a => a.state !== 'completed' && a.state !== 'cancelled'));
    setLoading(false);
  }, [algos, refreshStatus]);

  const handleStarted = useCallback((executionId: string) => {
    setAlgos(prev => [...prev, {
      executionId,
      state: 'running',
      algo: 'twap',
      submittedSlices: 0,
      totalSlices: 0,
      symbol: '',
      side: '',
      volume: 0,
    }]);
    // Fetch initial status.
    refreshStatus(executionId).then(status => {
      if (status) {
        setAlgos(prev => prev.map(a =>
          a.executionId === executionId ? {
            ...a,
            state: status.state,
            algo: status.algo,
            submittedSlices: status.submittedSlices,
            totalSlices: status.totalSlices,
            symbol: status.parentSymbol,
            side: status.parentSide,
            volume: status.parentVolume,
          } : a
        ));
      }
    });
  }, [refreshStatus]);

  const handleCancel = useCallback(async (executionId: string) => {
    try {
      await executionAlgoClient.cancelAlgo({ executionId });
      setAlgos(prev => prev.map(a =>
        a.executionId === executionId ? { ...a, state: 'cancelled' } : a
      ));
    } catch (_e: unknown) {
      // ignore — refresh will pick up state
    }
  }, []);

  const columns: ColumnsType<ActiveAlgo> = [
    { title: t('algo.table.executionId'), dataIndex: 'executionId', key: 'id', width: 100, render: (v: string) => <Typography.Text code>{v.slice(0, 8)}</Typography.Text> },
    { title: t('algo.table.algo'), dataIndex: 'algo', key: 'algo', width: 100, render: (v: string) => <Tag>{v?.toUpperCase()}</Tag> },
    { title: t('algo.table.symbol'), dataIndex: 'symbol', key: 'symbol', width: 90 },
    {
      title: t('algo.table.side'), dataIndex: 'side', key: 'side', width: 70,
      render: (v: string) => <Tag color={v === 'buy' ? 'green' : 'red'}>{v?.toUpperCase()}</Tag>,
    },
    { title: t('algo.table.volume'), dataIndex: 'volume', key: 'volume', width: 80, render: (v: number) => v?.toFixed(2) || '-' },
    {
      title: t('algo.table.progress'), key: 'progress', width: 150,
      render: (_: unknown, r: ActiveAlgo) => {
        const pct = r.totalSlices > 0 ? Math.round((r.submittedSlices / r.totalSlices) * 100) : 0;
        return <span>{r.submittedSlices}/{r.totalSlices} ({pct}%)</span>;
      },
    },
    {
      title: t('algo.table.state'), dataIndex: 'state', key: 'state', width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { running: 'blue', paused: 'orange', completed: 'green', cancelled: 'default', failed: 'red' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('algo.table.actions'), key: 'actions', width: 80,
      render: (_: unknown, r: ActiveAlgo) =>
        r.state === 'running' || r.state === 'paused' ? (
          <Button size="small" danger onClick={() => handleCancel(r.executionId)}>{t('algo.actions.cancel')}</Button>
        ) : null,
    },
  ];

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16 }}>
        <Title level={4} style={{ margin: 0 }}>{t('algo.dashboard.title')}</Title>
        <Button icon={<ReloadOutlined />} onClick={refreshAll} loading={loading}>{t('common.refresh')}</Button>
      </div>

      <Space direction="vertical" size="large" style={{ width: '100%' }}>
        <AlgoSubmitForm onStarted={handleStarted} />

        <Card title={t('algo.dashboard.activeExecutions')} size="small">
          {algos.length === 0 ? (
            <Empty description={t('algo.dashboard.noActive')} />
          ) : (
            <Table<ActiveAlgo>
              columns={columns}
              dataSource={algos}
              rowKey="executionId"
              pagination={false}
              size="small"
            />
          )}
        </Card>
      </Space>
    </div>
  );
}
