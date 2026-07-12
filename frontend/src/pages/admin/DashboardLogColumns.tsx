import { Tag } from 'antd';
import type { AdminLog } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import i18n from '@/i18n';

export function getLogColumns() {
  const t = i18n.t.bind(i18n);
  return [
    { title: t('admin.logs.columns.time', { defaultValue: 'Time' }), dataIndex: 'createdAt', key: 'createdAt', width: 180,
      render: (_text: unknown, record: AdminLog) => formatDateTime(record.createdAt) },
    { title: t('admin.logs.columns.module', { defaultValue: 'Module' }), dataIndex: 'module', key: 'module', width: 120,
      render: (text: string) => {
        const m: Record<string, string> = {
          user_management: t('admin.logs.modules.userManagement', { defaultValue: 'User Management' }),
          account_management: t('admin.logs.modules.accountManagement', { defaultValue: 'Account Management' }),
          trading: t('admin.logs.modules.trading', { defaultValue: 'Trading' }),
          system_config: t('admin.logs.modules.systemConfig', { defaultValue: 'System Config' }),
        };
        return m[text] || text;
      } },
    { title: t('admin.logs.columns.actionType', { defaultValue: 'Action Type' }), dataIndex: 'actionType', key: 'actionType', width: 100 },
    { title: t('admin.logs.columns.target', { defaultValue: 'Target' }), dataIndex: 'targetId', key: 'targetId', width: 200, ellipsis: true },
    { title: t('admin.logs.columns.status', { defaultValue: 'Status' }), dataIndex: 'success', key: 'success', width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'success' : 'error'}>
          {success ? t('common.success', { defaultValue: 'Success' }) : t('common.failed', { defaultValue: 'Failed' })}
        </Tag>
      ) },
  ];
}
