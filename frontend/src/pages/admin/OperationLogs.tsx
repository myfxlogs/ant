import { useCallback, useEffect, useState } from 'react';
import { Card, Table, Button, Select, DatePicker, message } from 'antd';
import { IconDownload } from '@tabler/icons-react';
import { adminApi, type AdminLog, type LogListParams } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';

const { RangePicker } = DatePicker;

export default function OperationLogs() {
  const [logs, setLogs] = useState<AdminLog[]>([]);
  const [loading, setLoading] = useState(true);
  const [total, setTotal] = useState(0);
  const [params, setParams] = useState<LogListParams>({ page: 1, pageSize: 20 });

  const fetchLogs = useCallback(async () => {
    setLoading(true);
    try {
      const result = await adminApi.listLogs(params);
      setLogs(result.logs);
      setTotal(result.total);
    } catch (error) {
      message.error(getErrorMessage(error, '加载日志失败'));
    } finally {
      setLoading(false);
    }
  }, [params]);

  useEffect(() => {
    fetchLogs();
  }, [fetchLogs]);

  const handleExport = async () => {
    try {
      const blob = await adminApi.exportLogs({
        userId: params.userId,
        action: params.action,
      });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `admin_logs_${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);
      message.success('导出成功');
    } catch (error) {
      message.error(getErrorMessage(error, '导出失败'));
    }
  };

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
    user_management: '用户管理',
    account_management: '账户管理',
    trading: '交易',
    system_config: '系统配置',
  };

  const columns = [
    {
      title: '时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (_text: any, record: AdminLog) => formatDateTime(record.createdAt),
    },
    {
      title: 'IP地址',
      dataIndex: 'ip',
      key: 'ip',
      width: 130,
    },
    {
      title: '操作',
      dataIndex: 'action',
      key: 'action',
      width: 140,
      render: (text: string) => moduleMap[text] || text,
    },
    {
      title: '详情',
      dataIndex: 'details',
      key: 'details',
      ellipsis: true,
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>操作日志</h1>
        <Button
          icon={<IconDownload size={16} />}
          onClick={handleExport}
        >
          导出
        </Button>
      </div>

      <Card>
        <div className="mb-4 flex gap-4 flex-wrap">
          <RangePicker onChange={(dates) => handleDateChange(dates as [Date | null, Date | null] | null)} />
          <Select
            placeholder="模块筛选"
            allowClear
            style={{ width: 120 }}
            onChange={(value) => setParams({ ...params, module: value, page: 1 })}
            options={[
              { label: '用户管理', value: 'user_management' },
              { label: '账户管理', value: 'account_management' },
              { label: '交易', value: 'trading' },
              { label: '系统配置', value: 'system_config' },
            ]}
          />
          <Select
            placeholder="操作类型"
            allowClear
            style={{ width: 120 }}
            onChange={(value) => setParams({ ...params, actionType: value, page: 1 })}
            options={[
              { label: '创建', value: 'create' },
              { label: '更新', value: 'update' },
              { label: '删除', value: 'delete' },
              { label: '禁用', value: 'disable' },
              { label: '启用', value: 'enable' },
              { label: '冻结', value: 'freeze' },
              { label: '解冻', value: 'unfreeze' },
            ]}
          />
        </div>

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
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, pageSize) => setParams({ ...params, page, pageSize }),
          }}
        />
      </Card>
    </div>
  );
}
