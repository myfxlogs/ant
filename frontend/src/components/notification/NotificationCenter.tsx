import React, { useState, useMemo, useCallback } from 'react';
import {
  Badge,
  Button,
  Dropdown,
  List,
  Typography,
  Space,
  Tag,
  Empty,
  Tabs,
  Popconfirm,
} from 'antd';
import {
  BellOutlined,
  CheckOutlined,
  DeleteOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  CodeOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { useNotificationStore } from '@/stores/notificationStore';
import { useNotificationListener } from '@/hooks/useNotificationListener';
import type { Notification } from '@/types/notification';
import { useTranslation } from 'react-i18next';

dayjs.extend(relativeTime);

const { Text } = Typography;

const getTypeIcon = (type: Notification['type']) => {
  switch (type) {
    case 'trade':
      return <ThunderboltOutlined style={{ color: '#1890ff' }} />;
    case 'signal':
      return <CodeOutlined style={{ color: '#52c41a' }} />;
    case 'risk_alert':
      return <WarningOutlined style={{ color: '#faad14' }} />;
    case 'strategy_execution':
      return <CheckCircleOutlined style={{ color: '#722ed1' }} />;
    default:
      return <SettingOutlined style={{ color: '#8c8c8c' }} />;
  }
};

const getTypeTag = (type: Notification['type']) => {
  const typeMap: Record<Notification['type'], { color: string; labelKey: string }> = {
    trade: { color: 'blue', labelKey: 'notifications.types.trade' },
    signal: { color: 'green', labelKey: 'notifications.types.signal' },
    risk_alert: { color: 'orange', labelKey: 'notifications.types.risk_alert' },
    strategy_execution: { color: 'purple', labelKey: 'notifications.types.strategy_execution' },
    system: { color: 'default', labelKey: 'notifications.types.system' },
  };
  return typeMap[type] || typeMap.system;
};

interface NotificationListProps {
  notifications: Notification[];
  onNotificationClick: (notification: Notification) => void;
  onRemove: (id: string) => void;
}

const NotificationList: React.FC<NotificationListProps> = ({
  notifications,
  onNotificationClick,
  onRemove,
}) => {
  const { t } = useTranslation();
  if (notifications.length === 0) {
    return (
      <Empty
        image={Empty.PRESENTED_IMAGE_SIMPLE}
        description={t('notifications.empty')}
        style={{ padding: '20px 0' }}
      />
    );
  }

  return (
    <List
      dataSource={notifications}
      style={{ maxHeight: 400, overflow: 'auto' }}
      renderItem={(item) => (
        <List.Item
          key={item.id}
          onClick={() => onNotificationClick(item)}
          style={{
            cursor: 'pointer',
            backgroundColor: item.read ? 'transparent' : '#f6ffed',
            padding: '8px 12px',
            borderBottom: '1px solid #f0f0f0',
          }}
        >
          <List.Item.Meta
            avatar={getTypeIcon(item.type)}
            title={
              <Space>
                <Text strong={!item.read}>{item.title}</Text>
                {(() => {
                  const cfg = getTypeTag(item.type);
                  return <Tag color={cfg.color}>{t(cfg.labelKey)}</Tag>;
                })()}
              </Space>
            }
            description={
              <Space direction="vertical" size={0}>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  {item.message}
                </Text>
                <Text type="secondary" style={{ fontSize: 11 }}>
                  {dayjs(item.created_at).fromNow()}
                </Text>
              </Space>
            }
          />
          <Button
            type="text"
            size="small"
            icon={<DeleteOutlined />}
            onClick={(e) => {
              e.stopPropagation();
              onRemove(item.id);
            }}
          />
        </List.Item>
      )}
    />
  );
};

interface NotificationCenterProps {
  className?: string;
}

const NotificationCenter: React.FC<NotificationCenterProps> = ({ className }) => {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const {
    notifications,
    unreadCount,
    markAsRead,
    markAllAsRead,
    removeNotification,
    clearAll,
  } = useNotificationStore();

  useNotificationListener();

  const handleNotificationClick = useCallback(
    (notification: Notification) => {
      if (!notification.read) {
        markAsRead(notification.id);
      }
    },
    [markAsRead]
  );

  const filterByType = useCallback(
    (type: string | 'all') => {
      if (type === 'all') return notifications;
      if (type === 'unread') return notifications.filter((n) => !n.read);
      return notifications.filter((n) => n.type === type);
    },
    [notifications]
  );

  const tabItems = useMemo(
    () => [
      {
        key: 'all',
        label: t('notifications.tabs.all', { count: notifications.length }),
        children: (
          <NotificationList
            notifications={filterByType('all')}
            onNotificationClick={handleNotificationClick}
            onRemove={removeNotification}
          />
        ),
      },
      {
        key: 'unread',
        label: t('notifications.tabs.unread', { count: unreadCount }),
        children: (
          <NotificationList
            notifications={filterByType('unread')}
            onNotificationClick={handleNotificationClick}
            onRemove={removeNotification}
          />
        ),
      },
      {
        key: 'trade',
        label: t('notifications.types.trade'),
        children: (
          <NotificationList
            notifications={filterByType('trade')}
            onNotificationClick={handleNotificationClick}
            onRemove={removeNotification}
          />
        ),
      },
      {
        key: 'risk_alert',
        label: t('notifications.types.risk_alert'),
        children: (
          <NotificationList
            notifications={filterByType('risk_alert')}
            onNotificationClick={handleNotificationClick}
            onRemove={removeNotification}
          />
        ),
      },
    ],
    [notifications, unreadCount, removeNotification, filterByType, handleNotificationClick, t]
  );

  const dropdownContent = (
    <div style={{ width: 360, backgroundColor: '#fff', borderRadius: 8, boxShadow: '0 2px 8px rgba(0,0,0,0.15)' }}>
      <div style={{ padding: '8px 12px', borderBottom: '1px solid #f0f0f0', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Text strong>{t('notifications.title')}</Text>
        <Space>
          <Button
            type="link"
            size="small"
            icon={<CheckOutlined />}
            onClick={markAllAsRead}
            disabled={unreadCount === 0}
          >
            {t('notifications.actions.markAllAsRead')}
          </Button>
          <Popconfirm
            title={t('notifications.actions.clearAllConfirm')}
            onConfirm={clearAll}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
          >
            <Button type="link" size="small" danger>
              {t('notifications.actions.clearAll')}
            </Button>
          </Popconfirm>
        </Space>
      </div>
      <Tabs
        defaultActiveKey="all"
        items={tabItems}
        size="small"
        style={{ padding: '0 8px' }}
      />
    </div>
  );

  return (
    <Dropdown
      popupRender={() => dropdownContent}
      trigger={['click']}
      open={visible}
      onOpenChange={setVisible}
      placement="bottomRight"
    >
      <Badge count={unreadCount} size="small" offset={[-2, 2]}>
        <Button
          type="text"
          className={className}
          icon={<BellOutlined style={{ fontSize: 18 }} />}
          style={{ color: 'inherit' }}
        />
      </Badge>
    </Dropdown>
  );
};

export default NotificationCenter;
