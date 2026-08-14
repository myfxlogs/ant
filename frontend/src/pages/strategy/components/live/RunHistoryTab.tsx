import { useState, useEffect, useCallback } from 'react';
import { Table, Typography, Button, Card, Empty } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyRunsApi } from '@/client/strategy';
import { accountApi } from '@/client/account';
import type { Account } from '@/types/account';
import type { StrategyRun } from '@/gen/ant/v1/strategy_runtime_pb';
import { formatTime, shortId } from '../../LiveStrategyPageSignalDrawer';

const { Text } = Typography;

export default function RunHistoryTab() {
  const { t } = useTranslation();
  const [runs, setRuns] = useState<StrategyRun[]>([]);
  const [loading, setLoading] = useState(false);
  const [accounts, setAccounts] = useState<Account[]>([]);

  useEffect(() => { void accountApi.list().then(setAccounts).catch(() => {}); }, []);

  const fmtAccount = useCallback((id: string) => {
    const a = accounts.find(x => x.id === id);
    return a?.login ? `${a.login} (${a.mtType})` : id;
  }, [accounts]);

  const fetchRuns = useCallback(async () => {
    setLoading(true);
    try {
      const r = await strategyRunsApi.listRuns({ limit: 100 });
      setRuns(r as StrategyRun[]);
    } catch { setRuns([]); }
    setLoading(false);
  }, []);

  useEffect(() => { void fetchRuns(); }, [fetchRuns]);

  const columns = [
    { title: t('strategy.live.runId', { defaultValue: 'Run ID' }), dataIndex: 'id', width: 100, render: (v: string) => <Text code copyable>{shortId(v)}</Text> },
    { title: t('strategy.live.account', { defaultValue: 'Account' }), dataIndex: 'accountId', width: 120, render: (v: string) => <Text style={{ fontSize: 12 }}>{fmtAccount(v)}</Text> },
    { title: t('strategy.live.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', width: 80 },
    { title: t('strategy.live.timeframe', { defaultValue: 'TF' }), dataIndex: 'timeframe', width: 60 },
    { title: t('strategy.live.mode', { defaultValue: 'Mode' }), dataIndex: 'mode', width: 70, render: (v: string) => <Text>{v}</Text> },
    { title: t('strategy.live.status', { defaultValue: 'Status' }), dataIndex: 'status', width: 90, render: (v: string) => <Text>{v}</Text> },
    { title: t('strategy.live.totalSignals', { defaultValue: 'Total Signals' }), dataIndex: 'totalSignals', width: 90, render: (v: number) => <Text strong>{v}</Text> },
    { title: t('strategy.live.startedAt', { defaultValue: 'Started' }), dataIndex: 'startedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.stoppedAt', { defaultValue: 'Stopped' }), dataIndex: 'stoppedAt', width: 140, render: (v: { seconds?: bigint; nanos?: number } | null) => <Text style={{ fontSize: 12 }}>{formatTime(v)}</Text> },
    { title: t('strategy.live.error', { defaultValue: 'Error' }), dataIndex: 'error', ellipsis: true, render: (v: string) => v ? <Text type="danger" style={{ fontSize: 12 }}>{v}</Text> : <Text type="secondary">-</Text> },
  ];

  return (
    <Card size="small">
      <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: 12 }}>
        <Button icon={<ReloadOutlined />} onClick={fetchRuns} loading={loading}>
          {t('common.refresh', { defaultValue: 'Refresh' })}
        </Button>
      </div>
      <Table
        size="small"
        dataSource={runs}
        rowKey="id"
        loading={loading}
        columns={columns}
        pagination={{ pageSize: 20, showSizeChanger: false }}
        locale={{ emptyText: <Empty description={t('strategy.live.noRuns', { defaultValue: 'No strategy runs' })} /> }}
      />
    </Card>
  );
}
