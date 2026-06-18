import { useState, useEffect, useCallback } from 'react';
import { Button, Modal, Table, message, Tag, Space, Typography } from 'antd';
import { ShareAltOutlined, CopyOutlined, LinkOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

function getAccessToken(): string | null {
  try {
    const raw = localStorage.getItem('auth-storage');
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return parsed?.state?.accessToken ?? null;
  } catch {
    return null;
  }
}

interface ShareItem {
  token: string;
  shareUrl: string;
  description: string;
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

  const fetchList = useCallback(async () => {
    const token = getAccessToken();
    if (!token) return;
    setLoading(true);
    try {
      const resp = await fetch('/api/shares/list', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      const data = await resp.json();
      if (Array.isArray(data)) setItems(data);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (open) fetchList();
  }, [open, fetchList]);

  const handleCreate = async () => {
    const token = getAccessToken();
    if (!token) return;
    try {
      const resp = await fetch('/ant.v1.ShareService/CreateShareToken', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` },
        body: JSON.stringify({ account_id: accountId, expire_days: 7 }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.message);
      const url = `${window.location.origin}${data.shareUrl}`;
      await navigator.clipboard.writeText(url);
      message.success(t('accounts.messages.shareLinkCopied', { defaultValue: '分享链接已复制到剪贴板' }));
      fetchList();
    } catch {
      message.error(t('accounts.messages.shareLinkFailed', { defaultValue: '创建分享链接失败' }));
    }
  };

  const handleCopy = async (shareUrl: string) => {
    const url = `${window.location.origin}${shareUrl}`;
    await navigator.clipboard.writeText(url);
    message.success(t('accounts.messages.shareLinkCopied', { defaultValue: '已复制' }));
  };

  const columns = [
    { title: t('share.token', { defaultValue: '分享链接' }), dataIndex: 'shareUrl', key: 'url',
      render: (url: string) => (
        <Space>
          <LinkOutlined />
          <Typography.Text copyable={{ text: `${window.location.origin}${url}` }} ellipsis style={{ maxWidth: 200 }}>
            {url}
          </Typography.Text>
        </Space>
      ),
    },
    { title: t('share.views', { defaultValue: '浏览量' }), dataIndex: 'viewCount', key: 'views', width: 80 },
    { title: t('share.expires', { defaultValue: '过期时间' }), dataIndex: 'expiresAt', key: 'expires', width: 120,
      render: (v: string) => {
        const d = new Date(v);
        const expired = d < new Date();
        return <Tag color={expired ? 'red' : 'green'}>{d.toLocaleDateString()}</Tag>;
      },
    },
    { title: t('share.actions', { defaultValue: '操作' }), key: 'actions', width: 80,
      render: (_: unknown, row: ShareItem) => (
        <Button size="small" icon={<CopyOutlined />} onClick={() => handleCopy(row.shareUrl)}>
          {t('common.copy', { defaultValue: '复制' })}
        </Button>
      ),
    },
  ];

  return (
    <>
      <Button icon={<ShareAltOutlined />} onClick={() => setOpen(true)}>
        {t('strategy.share', { defaultValue: '分享' })}
      </Button>
      <Modal
        title={t('share.title', { defaultValue: '分享管理' })}
        open={open}
        onCancel={() => setOpen(false)}
        width={640}
        footer={null}
      >
        <div className="mb-4">
          <Button type="primary" icon={<LinkOutlined />} onClick={handleCreate}>
            {t('share.createNew', { defaultValue: '创建新分享链接' })}
          </Button>
        </div>
        <Table
          dataSource={items}
          columns={columns}
          rowKey="token"
          loading={loading}
          size="small"
          pagination={false}
          locale={{ emptyText: t('share.empty', { defaultValue: '暂无分享链接' }) }}
        />
      </Modal>
    </>
  );
}
