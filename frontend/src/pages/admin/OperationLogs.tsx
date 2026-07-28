import { useCallback, useEffect, useState } from 'react';
import { Card, Table, Select, DatePicker, message } from 'antd';
import { adminApi, type AdminLog, type LogListParams } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';

const { RangePicker } = DatePicker;

export default function OperationLogs() {
  const { t } = useTranslation();
  const [logs, setLogs] = useState<AdminLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<LogListParams>({ page: 1, pageSize: 20 });

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await adminApi.listLogs(params);
      setLogs(result.logs);
      setTotal(result.total);
    } catch (err) {
      const msg = getErrorMessage(err, t('admin.logs.errors.loadFailed', { defaultValue: 'Failed to load logs' }));
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  }, [params, t]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleDateChange = (dates: [Date | null, Date | null] | null) => {
    if (dates && dates[0] && dates[1]) {
      const startDate = dates[0].toISOString().split('T')[0];
      const endDate = dates[1].toISOString().split('T')[0];
      setParams({ ...params, startDate, endDate, page: 1 });
    } else {
      setParams({ ...params, startDate: undefined, endDate: undefined, page: 1 });
    }
  };

  const moduleMap: Record<string, string> = {
    user_management: t('admin.logs.modules.userManagement', { defaultValue: 'User Management' }),
    account_management: t('admin.logs.modules.accountManagement', { defaultValue: 'Account Management' }),
    trading: t('admin.logs.modules.trading', { defaultValue: 'Trading' }),
    system_config: t('admin.logs.modules.systemConfig', { defaultValue: 'System Config' }),
  };

  const columns = [
    {
      title: t('admin.logs.columns.time', { defaultValue: 'Time' }),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_text: unknown, record: AdminLog) => formatDateTime(record.createdAt),
    },
    {
      title: t('admin.logs.columns.ip', { defaultValue: 'IP Address' }),
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
    },
    {
      title: t('admin.logs.columns.action', { defaultValue: 'Action' }),
      dataIndex: 'action',
      key: 'action',
      width: 140,
      render: (text: string) => moduleMap[text] || text,
    },
    {
      title: t('admin.logs.columns.details', { defaultValue: 'Details' }),
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{t('admin.logs.title', { defaultValue: 'Operation Logs' })}</h1>
      </div>

      <Card>
        <div className="mb-4 flex gap-4 flex-wrap">
          <RangePicker onChange={(dates) => handleDateChange(dates as [Date | null, Date | null] | null)} />
          <Select
            placeholder={t('admin.logs.filterModule', { defaultValue: 'Filter by module' })}
            allowClear
            style={{ width: 120 }}
            onChange={(value) => setParams({ ...params, module: value, page: 1 })}
            options={[
              { label: t('admin.logs.modules.userManagement', { defaultValue: 'User Management' }), value: 'user_management' },
              { label: t('admin.logs.modules.accountManagement', { defaultValue: 'Account Management' }), value: 'account_management' },
              { label: t('admin.logs.modules.trading', { defaultValue: 'Trading' }), value: 'trading' },
              { label: t('admin.logs.modules.systemConfig', { defaultValue: 'System Config' }), value: 'system_config' },
            ]}
          />
          <Select
            placeholder={t('admin.logs.filterAction', { defaultValue: 'Filter by action' })}
            allowClear
            style={{ width: 120 }}
            onChange={(value) => setParams({ ...params, actionType: value, page: 1 })}
            options={[
              { label: t('admin.logs.actions.create', { defaultValue: 'Create' }), value: 'create' },
              { label: t('admin.logs.actions.update', { defaultValue: 'Update' }), value: 'update' },
              { label: t('admin.logs.actions.delete', { defaultValue: 'Delete' }), value: 'delete' },
              { label: t('admin.logs.actions.disable', { defaultValue: 'Disable' }), value: 'disable' },
              { label: t('admin.logs.actions.enable', { defaultValue: 'Enable' }), value: 'enable' },
              { label: t('admin.logs.actions.freeze', { defaultValue: 'Freeze' }), value: 'freeze' },
              { label: t('admin.logs.actions.unfreeze', { defaultValue: 'Unfreeze' }), value: 'unfreeze' },
            ]}
          />
        </div>

        <StatusResult error={error} onRetry={fetchLogs}>
        <Table
          scroll={{ x: "max-content" }}
          columns={columns}
          dataSource={logs}
          rowKey="id"
          loading={loading}
          pagination={{
            current: params.page,
            pageSize: params.pageSize,
            total,
            showSizeChanger: true,
            showTotal: (total) => t('common.total', { total, defaultValue: `${total} total` }),
            onChange: (page, pageSize) => setParams({ ...params, page, pageSize }),
          }}
        />
        </StatusResult>
      </Card>
    </div>
  );
}
