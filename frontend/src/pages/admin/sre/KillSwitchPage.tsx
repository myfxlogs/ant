import { useState, useEffect, useCallback } from 'react';
import { Card, Button, Input, Modal, Space, Typography, Alert, Descriptions, Tag } from 'antd';
import { StopOutlined, PlayCircleOutlined, ReloadOutlined } from '@ant-design/icons';
import { sreApi, type KillSwitchStatus } from './sreApi';

const { Text, Title } = Typography;

export default function KillSwitchPage() {
  const [status, setStatus] = useState<KillSwitchStatus | null>(null);
  const [loading, setLoading] = useState(false);
  const [actionLoading, setActionLoading] = useState(false);
  const [reason, setReason] = useState('');
  const [modalOpen, setModalOpen] = useState(false);

  const fetchStatus = useCallback(async () => {
    setLoading(true);
    try { setStatus(await sreApi.killSwitchStatus()); } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchStatus(); }, [fetchStatus]);

  const handleEngage = async () => {
    if (!reason.trim()) return;
    setActionLoading(true);
    try { setStatus(await sreApi.killSwitchEngage(reason, 'admin')); setModalOpen(false); setReason(''); }
    finally { setActionLoading(false); }
  };

  const handleDisengage = async () => {
    setActionLoading(true);
    try { setStatus(await sreApi.killSwitchDisengage()); }
    finally { setActionLoading(false); }
  };

  return (
    <div style={{ maxWidth: 720 }}>
      <Title level={4}><StopOutlined style={{ marginRight: 8 }} />Kill Switch</Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        一键停止所有交易 — 紧急情况下使用，需二次确认
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
          </>
        )}
        <Space style={{ marginTop: 16 }}>
          <Button icon={<ReloadOutlined />} onClick={fetchStatus} loading={loading}>刷新</Button>
          {status?.engaged ? (
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleDisengage} loading={actionLoading}>
              解除 Kill Switch
            </Button>
          ) : (
            <Button type="primary" danger icon={<StopOutlined />} onClick={() => setModalOpen(true)} loading={actionLoading}>
              激活 Kill Switch
            </Button>
          )}
        </Space>
      </Card>

      <Modal title="确认激活 Kill Switch" open={modalOpen}
        onOk={handleEngage} onCancel={() => setModalOpen(false)}
        confirmLoading={actionLoading} okText="确认激活" okButtonProps={{ danger: true }}
      >
        <Text type="secondary" style={{ display: 'block', marginBottom: 12 }}>
          此操作将立即停止所有账户的所有交易活动，包括挂单。请确认后输入原因。
        </Text>
        <Input.TextArea rows={3} value={reason} onChange={e => setReason(e.target.value)}
          placeholder="激活原因（必填）" />
      </Modal>
    </div>
  );
}
