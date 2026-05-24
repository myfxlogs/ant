import { useState, useEffect } from 'react';
import { Card, List, Button, Space, Tag, message } from 'antd';
import { useTranslation } from 'react-i18next';

interface Strategy {
  publishId: string;
  strategyId: string;
  strategyName: string;
  publisherUserId: string;
  publishedAt?: string;
}

const API = '/ant.v1.MarketplaceService';

export default function MarketplacePage() {
  const { t } = useTranslation();
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [loading, setLoading] = useState(false);

  const [userId] = useState(() => localStorage.getItem('userId') || 'default');

  useEffect(() => {
    fetchPublished();
  }, []);

  async function fetchPublished() {
    setLoading(true);
    try {
      const res = await fetch(`${API}/ListPublished`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, limit: 50 }),
      });
      const data = await res.json();
      setStrategies(data.strategies || []);
    } catch {
      message.error('Failed to load strategies');
    } finally {
      setLoading(false);
    }
  }

  async function handleSubscribe(publisherUserId: string, strategyId: string) {
    try {
      const res = await fetch(`${API}/Subscribe`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, publisherUserId, strategyId, kind: 'copy_trade' }),
      });
      if (res.ok) message.success('Subscribed!');
      else message.error('Subscribe failed');
    } catch {
      message.error('Network error');
    }
  }

  return (
    <Card title={t('marketplace.title', 'Strategy Marketplace')}>
      <List
        loading={loading}
        dataSource={strategies}
        renderItem={(s: Strategy) => (
          <List.Item
            actions={[
              <Space key="actions">
                <Tag color="blue">{s.publisherUserId === userId ? 'Yours' : 'Public'}</Tag>
                {s.publisherUserId !== userId && (
                  <Button size="small" type="primary" onClick={() => handleSubscribe(s.publisherUserId, s.strategyId)}>
                    Subscribe
                  </Button>
                )}
              </Space>
            ]}
          >
            <List.Item.Meta
              title={s.strategyName || s.strategyId}
              description={`by ${s.publisherUserId} | ${s.strategyId}`}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
