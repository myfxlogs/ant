import React from 'react';
import {
  Button,
  List,
  Typography,
  Space,
  Tag,
  Empty,
} from 'antd';
import {
  DeleteOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  CodeOutlined,
  SettingOutlined,
  RocketOutlined,
  RobotOutlined,
  StopOutlined,
} from '@ant-design/icons';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
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
    case 'strategy_version_update':
      return <RocketOutlined style={{ color: '#13c2c2' }} />;
    case 'auto_fix_started':
      return <RobotOutlined style={{ color: '#1677ff' }} />;
    case 'auto_fix_stopped':
      return <StopOutlined style={{ color: '#fa8c16' }} />;
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
    strategy_version_update: { color: 'cyan', labelKey: 'notifications.types.strategy_version_update' },
    auto_fix_started: { color: 'blue', labelKey: 'notifications.types.auto_fix_started' },
    auto_fix_stopped: { color: 'orange', labelKey: 'notifications.types.auto_fix_stopped' },
    system: { color: 'default', labelKey: 'notifications.types.system' },
  };
  return typeMap[type] || typeMap.system;
};

interface NotificationListProps {
  notifications: Notification[];
  onNotificationClick: (notification: Notification) => void;
  onRemove: (id: string) => void;
}

export const NotificationList: React.FC<NotificationListProps> = ({
  notifications, onNotificationClick, onRemove,
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
