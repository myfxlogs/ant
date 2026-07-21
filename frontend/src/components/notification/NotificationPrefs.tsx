import { useState, useEffect, useCallback } from 'react';
import { Modal, Switch, Space, Typography, message } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { notificationClient } from '@/client/connect';

const { Text } = Typography;

interface Props {
  open: boolean;
  onClose: () => void;
}

export default function NotificationPrefs({ open, onClose }: Props) {
  const { t } = useTranslation();
  const [prefs, setPrefs] = useState({
    newStrategyEnabled: true,
    priceChangeEnabled: true,
    subExpiringEnabled: true,
    performanceAlertEnabled: true,
    newRatingEnabled: true,
  });
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!open) return;
    notificationClient.getNotificationPrefs({}).then(resp => {
      setPrefs({
        newStrategyEnabled: resp.newStrategyEnabled,
        priceChangeEnabled: resp.priceChangeEnabled,
        subExpiringEnabled: resp.subExpiringEnabled,
        performanceAlertEnabled: resp.performanceAlertEnabled,
        newRatingEnabled: resp.newRatingEnabled,
      });
    }).catch(() => {});
  }, [open]);

  const handleSave = useCallback(async (key: keyof typeof prefs, value: boolean) => {
    const prev = prefs;
    setPrefs({ ...prev, [key]: value });
    setSaving(true);
    try {
      await notificationClient.setNotificationPrefs({ ...prev, [key]: value });
    } catch {
      setPrefs(prev); // revert on failure
      message.error(t('notifications.prefs.saveFailed', { defaultValue: 'Failed to save preferences' }));
    } finally {
      setSaving(false);
    }
  }, [prefs, t]);

  const items: { key: keyof typeof prefs; label: string }[] = [
    { key: 'newStrategyEnabled', label: t('notifications.prefs.newStrategy', { defaultValue: 'New strategy published' }) },
    { key: 'priceChangeEnabled', label: t('notifications.prefs.priceChange', { defaultValue: 'Strategy price changed' }) },
    { key: 'subExpiringEnabled', label: t('notifications.prefs.subExpiring', { defaultValue: 'Subscription expiring soon' }) },
    { key: 'performanceAlertEnabled', label: t('notifications.prefs.performance', { defaultValue: 'Strategy performance anomaly' }) },
    { key: 'newRatingEnabled', label: t('notifications.prefs.newRating', { defaultValue: 'New rating or comment received' }) },
  ];

  return (
    <Modal
      title={<span><SettingOutlined /> {t('notifications.prefs.title', { defaultValue: 'Notification Preferences' })}</span>}
      open={open}
      onCancel={onClose}
      footer={null}
      width={480}
    >
      <Space direction="vertical" style={{ width: '100%' }} size="middle">
        {items.map(item => (
          <div key={item.key} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <Text>{item.label}</Text>
            <Switch
              checked={prefs[item.key]}
              loading={saving}
              onChange={(v) => handleSave(item.key, v)}
            />
          </div>
        ))}
      </Space>
    </Modal>
  );
}
