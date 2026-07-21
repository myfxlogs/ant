import React, { useState, useMemo, useCallback } from 'react';
import {
  Badge,
  Button,
  Dropdown,
  Space,
  Tabs,
  Popconfirm,
} from 'antd';
import {
  BellOutlined,
  CheckOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import { useNotificationStore } from '@/stores/notificationStore';
import { useNotificationListener } from '@/hooks/useNotificationListener';
import { useTranslation } from 'react-i18next';
import { NotificationList } from './NotificationList';
import NotificationPrefs from './NotificationPrefs';
import type { Notification } from '@/types/notification';

interface NotificationCenterProps {
  className?: string;
}

const NotificationCenter: React.FC<NotificationCenterProps> = ({ className }) => {
  const { t } = useTranslation();
  const {
    notifications,
    unreadCount,
    markAsRead,
    markAllAsRead,
    removeNotification,
    clearAll,
  } = useNotificationStore();

  const [tab, setTab] = useState<string>('all');
  const [open, setOpen] = useState(false);
  const [prefsOpen, setPrefsOpen] = useState(false);

  const filteredNotifications = useMemo(() => {
    if (tab === 'unread') return notifications.filter((n) => !n.read);
    return notifications;
  }, [notifications, tab]);

  const handleNotificationClick = useCallback((notification: Notification) => {
    markAsRead(notification.id);
    setOpen(false);
  }, [markAsRead]);

  const dropdownContent = (
    <div style={{ width: 380 }}>
      <div style={{
        display: 'flex',
        justifyContent: 'space-between',
        alignItems: 'center',
        padding: '8px 12px',
        borderBottom: '1px solid #f0f0f0',
      }}>
        <Space>
          <Tabs
            activeKey={tab}
            onChange={setTab}
            size="small"
            items={[
              { key: 'all', label: t('notifications.all') },
              { key: 'unread', label: t('notifications.unread') },
            ]}
          />
        </Space>
        <Space>
          <Button
            type="text"
            size="small"
            icon={<SettingOutlined />}
            onClick={() => setPrefsOpen(true)}
          />
          {unreadCount > 0 && (
            <Button
              type="link"
              size="small"
              icon={<CheckOutlined />}
              onClick={() => markAllAsRead()}
            >
              {t('notifications.markAllRead')}
            </Button>
          )}
          {filteredNotifications.length > 0 && (
            <Popconfirm
              title={t('notifications.confirmClearAll')}
              onConfirm={clearAll}
              okText={t('common.confirm')}
              cancelText={t('common.cancel')}
            >
              <Button type="link" size="small" danger>
                {t('notifications.clearAll')}
              </Button>
            </Popconfirm>
          )}
        </Space>
      </div>
      <NotificationList
        notifications={filteredNotifications}
        onNotificationClick={handleNotificationClick}
        onRemove={removeNotification}
      />
    </div>
  );

  return (
    <Dropdown
      open={open}
      onOpenChange={setOpen}
      dropdownRender={() => dropdownContent}
      trigger={['click']}
      placement="bottomRight"
    >
      <Badge count={unreadCount} size="small" offset={[-2, 2]}>
        <Button
          type="text"
          icon={<BellOutlined style={{ fontSize: 18 }} />}
          className={className}
        />
      </Badge>
      <NotificationPrefs open={prefsOpen} onClose={() => setPrefsOpen(false)} />
    </Dropdown>
  );
};

export default NotificationCenter;
