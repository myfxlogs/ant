import { useState, useEffect } from 'react';
import { Card, List, Button, Space, Tag, message } from 'antd';

interface Strategy {
  publishId: string;
  strategyId: string;
  strategyName: string;
  publisherUserId: string;
  publishedAt?: string;
}

const API = '/ant.v1.MarketplaceService';

export default function MarketplacePage() {
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [loading, setLoading] = useState(false);
  const userId = localStorage.getItem('userId') || '';

  useEffect(() => { fetchPublished(); }, []);

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
    if (!userId) { message.warning('No user ID set'); return; }
    try {
      const res = await fetch(`${API}/Subscribe`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, publisherUserId, strategyId, kind: 'copy_trade' }),
      });
      if (res.ok) message.success('Subscribed!');
      else { const err = await res.json(); message.error(err.message || 'Subscribe failed'); }
    } catch {
      message.error('Network error');
    }
  }

  async function handlePublish(strategyId: string) {
    if (!userId) { message.warning('No user ID set'); return; }
    try {
      await fetch(`${API}/PublishStrategy`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ userId, strategyId }),
      });
      message.success('Published!');
      fetchPublished();
    } catch {
      message.error('Publish failed');
    }
  }

  return (
    <Card title="Strategy Marketplace">
      <List
        loading={loading}
        dataSource={strategies}
        renderItem={(s: Strategy) => (
          <List.Item
            actions={[
              <Space key="actions">
                <Button size="small" type="primary" onClick={() => handleSubscribe(s.publisherUserId, s.strategyId)}>
                  Subscribe
                </Button>
                <Button size="small" onClick={() => handlePublish(s.strategyId)}>
                  Publish
                </Button>
              </Space>
            ]}
          >
            <List.Item.Meta
              title={s.strategyName || s.strategyId}
              description={`by ${s.publisherUserId ? s.publisherUserId.slice(0, 8) : 'unknown'}... | ${s.strategyId}`}
            />
          </List.Item>
        )}
      />
    </Card>
  );
}
