import { useState, useEffect, useCallback } from 'react';
import { Button, Modal, Table, message, Tag, Space, Typography, Popconfirm, Switch } from 'antd';
import { ShareAltOutlined, CopyOutlined, LinkOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { SHARE_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';
import { useAuthStore } from '@/stores/authStore';

interface ShareItem {
  token: string;
  shareUrl: string;
  description: string;
  showPositions: boolean;
  viewCount: number;
  expiresAt: string;
  createdAt: string;
}

interface Props {
  accountId: string;
}

export default function ShareAccountButton({ accountId }: Props) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [items, setItems] = useState<ShareItem[]>([]);
  const [loading, setLoading] = useState(false);
  const accessToken = useAuthStore(s => s.accessToken);
  const authHeaders = useCallback(() => {
    return accessToken ? { 'Content-Type': 'application/json' as const, 'Authorization': `Bearer ${accessToken}` } : null;
  }, [accessToken]);

  const fetchList = useCallback(async () => {
    const h = authHeaders();
    if (!h) return;
    setLoading(true);
    try {
      const resp = await fetch('/api/shares/list', { headers: h });
      const data = await resp.json();
      if (Array.isArray(data)) setItems(data);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [authHeaders]);

  useEffect(() => {
    if (open) fetchList();
  }, [open, fetchList]);

  const handleCreate = async () => {
    const h = authHeaders();
    if (!h) return;
    try {
      const resp = await fetch('/api/shares/create', {
        method: 'POST',
        headers: h,
        body: JSON.stringify({ account_id: accountId, expire_days: 7 }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.message);
      const url = `${window.location.origin}${data.shareUrl}`;
      await navigator.clipboard.writeText(url);
      message.success(t('accounts.messages.shareLinkCopied', { defaultValue: 'Share link copied to clipboard' }));
      fetchList();
    } catch {
      message.error(t('accounts.messages.shareLinkFailed', { defaultValue: 'Failed to create share link' }));
    }
  };

  const handleCopy = async (shareUrl: string) => {
    await navigator.clipboard.writeText(`${window.location.origin}${shareUrl}`);
    message.success(t('common.copied', { defaultValue: 'Copied' }));
  };

  const handleDelete = async (shareToken: string) => {
    const h = authHeaders();
    if (!h) return;
    try {
      const resp = await fetch('/api/shares/delete', {
        method: 'POST',
        headers: h,
        body: JSON.stringify({ token: shareToken }),
      });
      if (!resp.ok) throw new Error('Failed');
      message.success(t('common.deleted', { defaultValue: 'Deleted' }));
      fetchList();
    } catch {
      message.error(t('common.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  };

  const handleTogglePositions = async (shareToken: string, show: boolean) => {
    const h = authHeaders();
    if (!h) return;
    try {
      const resp = await fetch('/api/shares/update', {
        method: 'POST',
        headers: h,
        body: JSON.stringify({ token: shareToken, show_positions: show }),
      });
      if (!resp.ok) throw new Error('Failed');
      setItems(prev => prev.map(item => item.token === shareToken ? { ...item, showPositions: show } : item));
    } catch {
      message.error(t('common.operationFailed', { defaultValue: 'Operation failed' }));
    }
  };

  const columns = [
    { title: t('share.token', { defaultValue: 'Share Link' }), dataIndex: 'shareUrl', key: 'url',
      render: (url: string) => (
        <Space>
          <LinkOutlined />
          <Typography.Text copyable={{ text: `${window.location.origin}${url}` }} ellipsis style={{ maxWidth: 200 }}>
            {url}
          </Typography.Text>
        </Space>
      ),
    },
    { title: t('share.views', { defaultValue: 'Views' }), dataIndex: 'viewCount', key: 'views', width: 70 },
    {
      title: t('share.positions', { defaultValue: 'Positions' }), dataIndex: 'showPositions', key: 'pos', width: 70,
      render: (v: boolean, row: ShareItem) => (
        <Switch size="small" checked={v} onChange={(checked: boolean) => handleTogglePositions(row.token, checked)} />
      ),
    },
    { title: t('share.expires', { defaultValue: 'Expires' }), dataIndex: 'expiresAt', key: 'expires', width: 110,
      render: (v: string) => {
        const d = new Date(v);
        const expired = d < new Date();
        return <Tag color={expired ? 'red' : 'green'}>{d.toLocaleDateString()}</Tag>;
      },
    },
    { title: t('share.actions', { defaultValue: 'Actions' }), key: 'actions', width: 100,
      render: (_: unknown, row: ShareItem) => (
        <Space size="small">
          <Button size="small" icon={<CopyOutlined />} onClick={() => handleCopy(row.shareUrl)} />
          <Popconfirm
            title={t('share.deleteConfirm', { defaultValue: 'Delete this share link?' })}
            onConfirm={() => handleDelete(row.token)}
            okText={t('common.confirm', { defaultValue: 'OK' })}
            cancelText={t('common.cancel', { defaultValue: 'Cancel' })}
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <>
      <Button icon={<ShareAltOutlined />} onClick={() => setOpen(true)}>
        {t(SHARE_KEY, { defaultValue: 'Share' })}
      </Button>
      <Modal
        title={t('share.title', { defaultValue: 'Share Management' })}
        open={open}
        onCancel={() => setOpen(false)}
        width={700}
        footer={null}
      >
        <div className="mb-4">
          <Button type="primary" icon={<LinkOutlined />} onClick={handleCreate}>
            {t('share.createNew', { defaultValue: 'Create New Share Link' })}
          </Button>
        </div>
        <Table
          dataSource={items}
          columns={columns}
          rowKey="token"
          loading={loading}
          size="small"
          pagination={false}
          locale={{ emptyText: t('share.empty', { defaultValue: 'No share links yet' }) }}
        />
      </Modal>
    </>
  );
}
