import { Drawer, Descriptions } from 'antd';
import { useTranslation } from 'react-i18next';
import { formatDateTime } from '@/utils/date';
import type { UserWithAccounts } from '@/client/admin';

interface Props {
  visible: boolean;
  user: UserWithAccounts | null;
  onClose: () => void;
}

export default function UserDetailDrawer({ visible, user, onClose }: Props) {
  const { t } = useTranslation();
  return (
    <Drawer title={t('admin.userManagement.drawer.title')} placement="right" width={500} onClose={onClose} open={visible}>
      {user && (
        <Descriptions column={1} bordered>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.id')}>{user.id}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.email')}>{user.email}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.nickname')}>{user?.nickname}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.role')}>{user.role}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.status')}>{user.status}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.mtAccountCount')}>{user?.mtAccountCount}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.lastLogin')}>{formatDateTime(user?.lastLoginAt)}</Descriptions.Item>
          <Descriptions.Item label={t('admin.userManagement.drawer.labels.createdAt')}>{formatDateTime(user.createdAt)}</Descriptions.Item>
        </Descriptions>
      )}
    </Drawer>
  );
}
