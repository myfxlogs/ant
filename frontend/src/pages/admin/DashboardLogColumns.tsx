import { Tag } from 'antd';
import type { AdminLog } from '@/client/admin';
import { formatDateTime } from '@/utils/date';

export function getLogColumns() {
  return [
    { title: '时间', dataIndex: 'createdAt', key: 'createdAt', width: 180,
      render: (_text: unknown, record: AdminLog) => formatDateTime(record.createdAt) },
    { title: '模块', dataIndex: 'module', key: 'module', width: 120,
      render: (text: string) => {
        const m: Record<string, string> = {
          user_management: '用户管理',
          account_management: '账户管理',
          trading: '交易',
          system_config: '系统配置',
        };
        return m[text] || text;
      } },
    { title: '操作类型', dataIndex: 'actionType', key: 'actionType', width: 100 },
    { title: '目标', dataIndex: 'targetId', key: 'targetId', width: 200, ellipsis: true },
    { title: '状态', dataIndex: 'success', key: 'success', width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'success' : 'error'}>
          {success ? '成功' : '失败'}
        </Tag>
      ) },
  ];
}
