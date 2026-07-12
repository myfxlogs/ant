import { useState, useEffect, useCallback } from 'react';
import { Card, Button, Input, Modal, Space, Typography, Alert, Descriptions, Tag } from 'antd';
import { StopOutlined, PlayCircleOutlined, ReloadOutlined, UndoOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { sreApi, type KillSwitchStatus } from './sreApi';

const { Text, Title } = Typography;
const UNDO_WINDOW_MS = 5 * 60 * 1000; // 5-minute undo window

export default function KillSwitchPage() {
  const { t } = useTranslation();
  const [status, setStatus] = useState<KillSwitchStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [reason, setReason] = useState('');
  const [confirmText, setConfirmText] = useState('');
  const [modalOpen, setModalOpen] = useState(false);
  const [undoRemaining, setUndoRemaining] = useState(0);
  const [undoLoading, setUndoLoading] = useState(false);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    try { setStatus(await sreApi.killSwitchStatus()); } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchStatus(); }, [fetchStatus]);

  // Undo countdown timer
  useEffect(() => {
    if (!status?.engaged || !status.engaged_at) return;
    const engagedMs = Date.parse(status.engaged_at);
    if (isNaN(engagedMs)) return;
    const elapsed = Date.now() - engagedMs;
    if (elapsed >= UNDO_WINDOW_MS) { setUndoRemaining(0); return; }
    setUndoRemaining(Math.ceil((UNDO_WINDOW_MS - elapsed) / 1000));
    const interval = setInterval(() => {
      const remaining = Math.ceil((UNDO_WINDOW_MS - (Date.now() - engagedMs)) / 1000);
      if (remaining <= 0) { setUndoRemaining(0); clearInterval(interval); }
      else { setUndoRemaining(remaining); }
    }, 1000);
    return () => clearInterval(interval);
  }, [status?.engaged, status?.engaged_at]);

  const handleEngage = async () => {
    if (!reason.trim() || confirmText !== 'KILL') return;
    setActionLoading(true);
    try {
      const s = await sreApi.killSwitchEngage(reason, 'admin');
      setStatus(s);
      setModalOpen(false);
      setReason('');
      setConfirmText('');
      setUndoRemaining(Math.ceil(UNDO_WINDOW_MS / 1000));
    } finally { setActionLoading(false); }
  };

  const handleDisengage = async () => {
    setActionLoading(true);
    try { setStatus(await sreApi.killSwitchDisengage()); setUndoRemaining(0); }
    finally { setActionLoading(false); }
  };

  const handleUndo = async () => {
    setUndoLoading(true);
    try { setStatus(await sreApi.killSwitchDisengage()); setUndoRemaining(0); }
    finally { setUndoLoading(false); }
  };

  const canConfirm = reason.trim().length > 0 && confirmText === 'KILL';

  return (
    <div style={{ maxWidth: 720 }}>
      <Title level={4}><StopOutlined style={{ marginRight: 8 }} />Kill Switch</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        {t('sre.killSwitch.description', { defaultValue: 'One-click stop all trading — requires KILL confirmation; undo within 5 minutes' })}
      </Text>

      <Card size="small" loading={loading}>
        {status && (
          <>
            {status.engaged
              ? <Alert type="error" message={t('sre.killSwitch.engaged', { defaultValue: 'Kill Switch engaged — all trading stopped' })} showIcon style={{ marginBottom: 16 }} />
              : <Alert type="success" message={t('sre.killSwitch.disarmed', { defaultValue: 'Kill Switch disarmed — trading normal' })} showIcon style={{ marginBottom: 16 }} />
            }
            <Descriptions size="small" column={2}>
              <Descriptions.Item label={t('sre.killSwitch.status', { defaultValue: 'Status' })}>
                <Tag color={status.engaged ? 'red' : 'green'}>{status.engaged ? 'ENGAGED' : 'DISARMED'}</Tag>
              </Descriptions.Item>
              {status.engaged && (
                <>
                  <Descriptions.Item label={t('sre.killSwitch.reason', { defaultValue: 'Reason' })}>{status.reason}</Descriptions.Item>
                  <Descriptions.Item label={t('sre.killSwitch.operator', { defaultValue: 'Operator' })}>{status.operator}</Descriptions.Item>
                  <Descriptions.Item label={t('sre.killSwitch.engagedAt', { defaultValue: 'Engaged At' })}>{status.engaged_at}</Descriptions.Item>
                </>
              )}
            </Descriptions>
            {undoRemaining > 0 && (
              <Alert
                type="warning"
                showIcon
                style={{ marginTop: 12 }}
                message={t('sre.killSwitch.undoWindow', { m: Math.floor(undoRemaining / 60), s: undoRemaining % 60, defaultValue: `Undo window: ${Math.floor(undoRemaining / 60)}m ${undoRemaining % 60}s remaining` })}
                action={
                  <Button size="small" icon={<UndoOutlined />} onClick={handleUndo} loading={undoLoading}>
                    {t('sre.killSwitch.undo', { defaultValue: 'Undo Kill Switch' })}
                  </Button>
                }
              />
            )}
          </>
        )}
        <Space style={{ marginTop: 16 }}>
          <Button icon={<ReloadOutlined />} onClick={fetchStatus} loading={loading}>{t('common.refresh', { defaultValue: 'Refresh' })}</Button>
          {status?.engaged ? (
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleDisengage} loading={actionLoading}>
              {t('sre.killSwitch.disengage', { defaultValue: 'Disengage Kill Switch' })}
            </Button>
          ) : (
            <Button type="primary" danger icon={<StopOutlined />} onClick={() => { setConfirmText(''); setReason(''); setModalOpen(true); }} loading={actionLoading}>
              {t('sre.killSwitch.engage', { defaultValue: 'Engage Kill Switch' })}
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title={t('sre.killSwitch.confirmTitle', { defaultValue: 'Engage Kill Switch — Confirmation' })}
        open={modalOpen}
        onOk={handleEngage}
        onCancel={() => { setModalOpen(false); setConfirmText(''); }}
        confirmLoading={actionLoading}
        okText={t('sre.killSwitch.confirmEngage', { defaultValue: 'Confirm Engage' })}
        okButtonProps={{ danger: true, disabled: !canConfirm }}
      >
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message={t('sre.killSwitch.confirmWarning', { defaultValue: 'This will immediately stop all trading activity for all accounts, including pending and submitted orders. Enter a reason and type KILL to confirm.' })}
        />
        <Text strong style={{ display: 'block', marginBottom: 8 }}>{t('sre.killSwitch.reasonLabel', { defaultValue: 'Reason (required)' })}</Text>
        <Input.TextArea
          rows={3}
          value={reason}
          onChange={e => setReason(e.target.value)}
          placeholder={t('sre.killSwitch.reasonPlaceholder', { defaultValue: 'e.g.: Detected abnormal market volatility, emergency stop all trading' })}
          style={{ marginBottom: 12 }}
        />
        <Text strong style={{ display: 'block', marginBottom: 8 }}>{t('sre.killSwitch.typeKill', { defaultValue: 'Type KILL to confirm' })}</Text>
        <Input
          value={confirmText}
          onChange={e => setConfirmText(e.target.value)}
          placeholder={t('sre.killSwitch.typeKillPlaceholder', { defaultValue: 'Type KILL (uppercase)' })}
          status={confirmText.length > 0 && confirmText !== 'KILL' ? 'error' : undefined}
        />
      </Modal>
    </div>
  );
}
