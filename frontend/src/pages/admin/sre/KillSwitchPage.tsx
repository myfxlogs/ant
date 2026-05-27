import { useState, useEffect, useCallback } from 'react';
import { Card, Button, Input, Modal, Space, Typography, Alert, Descriptions, Tag } from 'antd';
import { StopOutlined, PlayCircleOutlined, ReloadOutlined, UndoOutlined } from '@ant-design/icons';
import { sreApi, type KillSwitchStatus } from './sreApi';

const { Text, Title } = Typography;
const UNDO_WINDOW_MS = 5 * 60 * 1000; // 5-minute undo window

export default function KillSwitchPage() {
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
        一键停止所有交易 — 激活需输入 KILL 确认；激活后 5 分钟内可撤销
      </Text>

      <Card size="small" loading={loading}>
        {status && (
          <>
            {status.engaged
              ? <Alert type="error" message="Kill Switch 已激活 — 所有交易已停止" showIcon style={{ marginBottom: 16 }} />
              : <Alert type="success" message="Kill Switch 未激活 — 交易正常运行" showIcon style={{ marginBottom: 16 }} />
            }
            <Descriptions size="small" column={2}>
              <Descriptions.Item label="状态">
                <Tag color={status.engaged ? 'red' : 'green'}>{status.engaged ? 'ENGAGED' : 'DISARMED'}</Tag>
              </Descriptions.Item>
              {status.engaged && (
                <>
                  <Descriptions.Item label="原因">{status.reason}</Descriptions.Item>
                  <Descriptions.Item label="操作人">{status.operator}</Descriptions.Item>
                  <Descriptions.Item label="激活时间">{status.engaged_at}</Descriptions.Item>
                </>
              )}
            </Descriptions>
            {undoRemaining > 0 && (
              <Alert
                type="warning"
                showIcon
                style={{ marginTop: 12 }}
                message={`撤销窗口：${Math.floor(undoRemaining / 60)}分${undoRemaining % 60}秒后关闭`}
                action={
                  <Button size="small" icon={<UndoOutlined />} onClick={handleUndo} loading={undoLoading}>
                    撤销 Kill Switch
                  </Button>
                }
              />
            )}
          </>
        )}
        <Space style={{ marginTop: 16 }}>
          <Button icon={<ReloadOutlined />} onClick={fetchStatus} loading={loading}>刷新</Button>
          {status?.engaged ? (
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleDisengage} loading={actionLoading}>
              解除 Kill Switch
            </Button>
          ) : (
            <Button type="primary" danger icon={<StopOutlined />} onClick={() => { setConfirmText(''); setReason(''); setModalOpen(true); }} loading={actionLoading}>
              激活 Kill Switch
            </Button>
          )}
        </Space>
      </Card>

      <Modal
        title="激活 Kill Switch — 二次确认"
        open={modalOpen}
        onOk={handleEngage}
        onCancel={() => { setModalOpen(false); setConfirmText(''); }}
        confirmLoading={actionLoading}
        okText="确认激活"
        okButtonProps={{ danger: true, disabled: !canConfirm }}
      >
        <Alert
          type="error"
          showIcon
          style={{ marginBottom: 16 }}
          message="此操作将立即停止所有账户的所有交易活动，包括挂单和已提交订单。请确认后输入原因并键入 KILL。"
        />
        <Text strong style={{ display: 'block', marginBottom: 8 }}>激活原因（必填）</Text>
        <Input.TextArea
          rows={3}
          value={reason}
          onChange={e => setReason(e.target.value)}
          placeholder="例如：检测到异常行情波动，紧急停止所有交易"
          style={{ marginBottom: 12 }}
        />
        <Text strong style={{ display: 'block', marginBottom: 8 }}>键入 KILL 确认激活</Text>
        <Input
          value={confirmText}
          onChange={e => setConfirmText(e.target.value)}
          placeholder='请输入 KILL（大写）'
          status={confirmText.length > 0 && confirmText !== 'KILL' ? 'error' : undefined}
        />
      </Modal>
    </div>
  );
}
