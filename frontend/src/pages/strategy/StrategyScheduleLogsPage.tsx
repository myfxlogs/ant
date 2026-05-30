import { useCallback, useEffect, useMemo, useState } from 'react';
import { Card, Tabs, Table } from 'antd';
import { useParams } from 'react-router-dom';
import { Typography } from 'antd';
import { scheduleHealthApi } from '@/client/scheduleHealth';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import type { OrderHistoryRecord } from '@/gen/ant/v1/log_order_pb';
import {
  formatLogTime, buildExecColumns, buildOrderColumns,
} from './scheduleLogColumns';

const { Title, Text } = Typography;

export default function StrategyScheduleLogsPage() {
  const { t } = useTranslation();
  const { id: scheduleId } = useParams<{ id: string }>();
  const [activeTab, setActiveTab] = useState('exec');
  const [execLogs, setExecLogs] = useState<ScheduleRunLog[]>([]);
  const [orderLogs, setOrderLogs] = useState<OrderHistoryRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshExec = useCallback(async () => {
    if (!scheduleId) return;
    try {
      const resp = await scheduleHealthApi.getScheduleRunLogs({ scheduleId, page: 1, pageSize: 200 });
      setExecLogs((resp?.logs || []) as ScheduleRunLog[]);
    } catch (e: unknown) {
      setError(String((e as { message?: string })?.message || e));
    }
  }, [scheduleId]);

  const refreshOrders = useCallback(async () => {
    if (!scheduleId) return;
    try {
      const resp = await scheduleHealthApi.getScheduleOrders({ scheduleId, page: 1, pageSize: 200 });
      setOrderLogs((resp?.orders || []) as OrderHistoryRecord[]);
    } catch { /* non-critical */ }
  }, [scheduleId]);

  const refresh = useCallback(async () => {
    setLoading(true); setError(null);
    try { await refreshExec(); } catch (e: unknown) {
      setError(String((e as { message?: string })?.message || e));
    }
    setLoading(false);
  }, [refreshExec]);

  useEffect(() => { void refresh(); }, [refresh]);
  useEffect(() => { if (activeTab === 'exec') void refreshExec(); }, [activeTab, refreshExec]);
  useEffect(() => { if (activeTab === 'orders') void refreshOrders(); }, [activeTab, refreshOrders]);

  const colOpts = { t, formatTime: formatLogTime };
  const execColumns = useMemo(() => buildExecColumns(colOpts), [t]);
  const orderColumns = useMemo(() => buildOrderColumns(colOpts), [t]);

  return (
    <div className="p-6">
      <Card>
        <div className="flex items-center justify-between mb-4">
          <Title level={4} style={{ margin: 0 }}>{t('strategy.scheduleLogs.title')}</Title>
          <Text type="secondary">{scheduleId}</Text>
        </div>
        <Tabs activeKey={activeTab} onChange={setActiveTab}
          items={[
            { key: 'exec', label: t('strategy.scheduleLogs.tabs.execLogs') },
            { key: 'orders', label: t('strategy.scheduleLogs.tabs.orderLogs') },
          ]} />
        <StatusResult loading={loading} error={error} onRetry={refresh}
          empty={!loading && !error && activeTab === 'exec' && execLogs.length === 0}
          emptyText={t('common.noData')}>
          {activeTab === 'exec' ? (
            <Table columns={execColumns} dataSource={execLogs} rowKey="id"
              scroll={{ x: 'max-content' }}
              pagination={false} />
          ) : (
            <Table columns={orderColumns} dataSource={orderLogs} rowKey="id"
              scroll={{ x: 'max-content' }}
              pagination={false} />
          )}
        </StatusResult>
      </Card>
    </div>
  );
}
