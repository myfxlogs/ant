import { useState, useEffect } from 'react';
import { Table, Tag, Typography } from 'antd';
import { LinkOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface ShareItem {
  token: string;
  shareUrl: string;
  userId: string;
  description: string;
  viewCount: number;
  expiresAt: string;
  createdAt: string;
}

function getAccessToken(): string | null {
  try {
    const raw = localStorage.getItem('auth-storage');
    if (!raw) return null;
    return JSON.parse(raw)?.state?.accessToken ?? null;
  } catch { return null; }
}

export default function ShareManagement() {
  const { t } = useTranslation();
  const [data, setData] = useState<ShareItem[]>([]);
  const [loading, setLoading] = useState(false);

  const fetchData = async () => {
    const token = getAccessToken();
    if (!token) return;
    setLoading(true);
    try {
      const resp = await fetch('/api/admin/shares/list', {
        headers: { 'Authorization': `Bearer ${token}` },
      });
      const json = await resp.json();
      if (Array.isArray(json)) setData(json);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  useEffect(() => { fetchData(); }, []);

  const columns = [
    {
      title: t('share.userId', { defaultValue: '用户' }),
      dataIndex: 'userId', key: 'user', width: 120,
      render: (v: string) => <Typography.Text ellipsis style={{ maxWidth: 100 }}>{v.slice(0, 8)}</Typography.Text>,
    },
    {
      title: t('share.token', { defaultValue: '分享链接' }),
      dataIndex: 'shareUrl', key: 'url',
      render: (url: string) => (
        <Typography.Text copyable ellipsis style={{ maxWidth: 200 }}>
          <LinkOutlined /> {url}
        </Typography.Text>
      ),
    },
    {
      title: t('share.views', { defaultValue: '浏览量' }),
      dataIndex: 'viewCount', key: 'views', width: 80,
    },
    {
      title: t('share.expires', { defaultValue: '过期' }),
      dataIndex: 'expiresAt', key: 'expires', width: 110,
      render: (v: string) => {
        const expired = new Date(v) < new Date();
        return <Tag color={expired ? 'red' : 'green'}>{new Date(v).toLocaleDateString()}</Tag>;
      },
    },
    {
      title: t('share.createdAt', { defaultValue: '创建时间' }),
      dataIndex: 'createdAt', key: 'created', width: 110,
      render: (v: string) => new Date(v).toLocaleDateString(),
    },
  ];

  return (
    <div>
      <h2 className="text-xl font-bold mb-4" style={{ color: 'var(--color-text)' }}>
        {t('admin.sidebar.shareManagement', { defaultValue: '分享统计' })}
      </h2>
      <Table
        dataSource={data}
        columns={columns}
        rowKey="token"
        loading={loading}
        size="small"
        pagination={{ pageSize: 20 }}
      />
    </div>
  );
}
