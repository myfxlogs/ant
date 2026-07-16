import { useState, useEffect } from 'react';
import { Table, Tag, Typography } from 'antd';
import { LinkOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { shareClient } from '@/client/connect';

interface ShareItem {
  token: string;
  shareUrl: string;
  userId: string;
  description: string;
  viewCount: number;
  expiresAt: string;
  createdAt: string;
}

export default function ShareManagement() {
  const { t } = useTranslation();
  const [data, setData] = useState<ShareItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);

  const fetchData = async (page = 1, pageSize = 20) => {
    setLoading(true);
    try {
      const resp = await shareClient.listAllShareTokens({ page, pageSize });
      setData(resp.items.map(item => ({
        token: item.token,
        shareUrl: item.shareUrl,
        userId: item.userId,
        description: item.description,
        viewCount: item.viewCount,
        expiresAt: item.expiresAt,
        createdAt: item.createdAt,
      })));
      setTotal(resp.total || 0);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, []);

  const columns = [
    {
      title: t('share.userId', { defaultValue: 'User' }),
      dataIndex: 'userId', key: 'user', width: 120,
      render: (v: string) => <Typography.Text ellipsis style={{ maxWidth: 100 }}>{v.slice(0, 8)}</Typography.Text>,
    },
    {
      title: t('share.token', { defaultValue: 'Share Link' }),
      dataIndex: 'shareUrl', key: 'url',
      render: (url: string) => (
        <Typography.Text copyable ellipsis style={{ maxWidth: 200 }}>
          <LinkOutlined /> {url}
        </Typography.Text>
      ),
    },
    {
      title: t('share.views', { defaultValue: 'Views' }),
      dataIndex: 'viewCount', key: 'views', width: 80,
    },
    {
      title: t('share.expires', { defaultValue: 'Expires' }),
      dataIndex: 'expiresAt', key: 'expires', width: 110,
      render: (v: string) => {
        const expired = new Date(v) < new Date();
        return <Tag color={expired ? 'red' : 'green'}>{new Date(v).toLocaleDateString()}</Tag>;
      },
    },
    {
      title: t('share.createdAt', { defaultValue: 'Created' }),
      dataIndex: 'createdAt', key: 'created', width: 110,
      render: (v: string) => new Date(v).toLocaleDateString(),
    },
  ];

  return (
    <div>
      <h2 className="text-xl font-bold mb-4" style={{ color: 'var(--color-text)' }}>
        {t('admin.sidebar.shareManagement', { defaultValue: 'Share Analytics' })}
      </h2>
      <Table
        dataSource={data}
        columns={columns}
        rowKey="token"
        loading={loading}
        size="small"
        pagination={{ total, pageSize: 20, onChange: (p) => fetchData(p) }}
      />
    </div>
  );
}
