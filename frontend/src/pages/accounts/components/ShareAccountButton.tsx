import { useState, useEffect, useCallback } from 'react';
import { Button, Modal, Table, message, Tag, Space, Typography, Popconfirm, Switch } from 'antd';
import { ShareAltOutlined, CopyOutlined, LinkOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { SHARE_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';
import { shareClient } from '@/client/connect';

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

  const fetchList = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await shareClient.listShareTokens({});
      setItems(resp.items.map(item => ({
        token: item.token,
        shareUrl: item.shareUrl,
        description: item.description,
        showPositions: item.showPositions,
        viewCount: item.viewCount,
        expiresAt: item.expiresAt,
        createdAt: item.createdAt,
      })));
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => {
    if (open) fetchList();
  }, [open, fetchList]);

  const handleCreate = async () => {
    try {
      const resp = await shareClient.createShareToken({ accountId, expireDays: 7 });
      const url = `${window.location.origin}${resp.shareUrl}`;
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
    try {
      await shareClient.deleteShareToken({ token: shareToken });
      message.success(t('common.deleted', { defaultValue: 'Deleted' }));
      fetchList();
    } catch {
      message.error(t('common.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  };

  const handleTogglePositions = async (shareToken: string, show: boolean) => {
    try {
      await shareClient.updateShareToken({ token: shareToken, showPositions: show });
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
        width="90vw"
        style={{ maxWidth: 700 }}
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
