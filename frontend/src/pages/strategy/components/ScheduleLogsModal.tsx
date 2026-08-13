import { useCallback, useEffect, useMemo, useState } from 'react';
import { Modal, Tabs, Table } from 'antd';
import { useTranslation } from 'react-i18next';
import { logApi } from '@/client/log';
import { StatusResult } from '@/components/common/StatusResult';
import type { ScheduleRunLog } from '@/gen/ant/v1/log_schedule_pb';
import type { OrderHistoryRecord } from '@/gen/ant/v1/log_order_pb';
import { formatLogTime, buildExecColumns, buildOrderColumns } from '../scheduleLogColumns';
import { TABS_EXEC_LOGS_KEY, TABS_ORDER_LOGS_KEY, TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_schedule_logs_keys';

type Props = {
  open: boolean;
  scheduleId: string | null;
  onClose: () => void;
};

export default function ScheduleLogsModal({ open, scheduleId, onClose }: Props) {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('exec');
  const [execLogs, setExecLogs] = useState<ScheduleRunLog[]>([]);
  const [orderLogs, setOrderLogs] = useState<OrderHistoryRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshExec = useCallback(async () => {
    if (!scheduleId) return;
    try {
      const resp = await logApi.getScheduleRunLogs({ scheduleId, page: 1, pageSize: 200 });
      setExecLogs((resp?.logs || []) as ScheduleRunLog[]);
    } catch (e: unknown) {
      setError(String((e as { message?: string })?.message || e));
    }
  }, [scheduleId]);

  const refreshOrders = useCallback(async () => {
    if (!scheduleId) return;
    try {
      const resp = await logApi.getOrderHistory({ scheduleId, page: 1, pageSize: 200 });
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

  useEffect(() => { if (open) { void refresh(); void refreshOrders(); } }, [open, refresh, refreshOrders]);
  useEffect(() => { if (open && activeTab === 'orders') void refreshOrders(); }, [activeTab, open, refreshOrders]);

  const colOpts = useMemo(() => ({ t, formatTime: formatLogTime }), [t]);
  const execColumns = useMemo(() => buildExecColumns(colOpts), [colOpts]);
  const orderColumns = useMemo(() => buildOrderColumns(colOpts), [colOpts]);

  return (
    <Modal
      title={t(TITLE_KEY)}
      open={open}
      onCancel={onClose}
      footer={null}
      width={900}
      destroyOnClose
    >
      <Tabs activeKey={activeTab} onChange={setActiveTab}
        items={[
          { key: 'exec', label: t(TABS_EXEC_LOGS_KEY) },
          { key: 'orders', label: t(TABS_ORDER_LOGS_KEY) },
        ]} />
      <StatusResult loading={loading} error={error} onRetry={refresh}
        empty={!loading && !error && activeTab === 'exec' && execLogs.length === 0}
        emptyText={t('common.noData')}>
        {activeTab === 'exec' ? (
          <Table columns={execColumns} dataSource={execLogs} rowKey="id"
            scroll={{ x: 'max-content' }} pagination={false} size="small" />
        ) : (
          <Table columns={orderColumns} dataSource={orderLogs} rowKey="id"
            scroll={{ x: 'max-content' }} pagination={false} size="small" />
        )}
      </StatusResult>
    </Modal>
  );
}
