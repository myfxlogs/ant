import { Tag } from 'antd';
import type { AdminLog } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import type { TFunction } from 'i18next';

export function getLogColumns(t: TFunction) {
  return [
    { title: t('admin.dashboard.logs.time'), dataIndex: 'createdAt', key: 'createdAt', width: 180,
      render: (_text: unknown, record: AdminLog) => formatDateTime(record.createdAt) },
    { title: t('admin.dashboard.logs.module'), dataIndex: 'module', key: 'module', width: 120,
      render: (text: string) => {
        const moduleMap: Record<string, string> = {
          user_management: t('admin.dashboard.logs.moduleMap.userManagement'),
          account_management: t('admin.dashboard.logs.moduleMap.accountManagement'),
          trading: t('admin.dashboard.logs.moduleMap.trading'),
          system_config: t('admin.dashboard.logs.moduleMap.systemConfig'),
        };
        return moduleMap[text] || text;
      } },
    { title: t('admin.dashboard.logs.actionType'), dataIndex: 'actionType', key: 'actionType', width: 100 },
    { title: t('admin.dashboard.logs.target'), dataIndex: 'targetId', key: 'targetId', width: 200, ellipsis: true },
    { title: t('admin.dashboard.logs.status'), dataIndex: 'success', key: 'success', width: 80,
      render: (success: boolean) => (
        <Tag color={success ? 'success' : 'error'}>
          {success ? t('admin.dashboard.logs.success') : t('admin.dashboard.logs.failed')}
        </Tag>
      ) },
  ];
}
