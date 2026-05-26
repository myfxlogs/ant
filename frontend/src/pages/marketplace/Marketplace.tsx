import { useState, useEffect, useCallback } from 'react';
import { Card, List, Button, Space, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { StatusResult } from '@/components/common/StatusResult';

interface Strategy {
  publishId: string;
  strategyId: string;
  strategyName: string;
  publisherUserId: string;
  publishedAt?: string;
}

export default function MarketplacePage() {
  const { t } = useTranslation();
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const userId = localStorage.getItem('userId') || '';

  const fetchPublished = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await marketplaceClient.listPublished({ userId, limit: 50 });
      setStrategies((resp.strategies || []) as unknown as Strategy[]);
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : t('common.loadingFailed');
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, [userId, t]);

  useEffect(() => { fetchPublished(); }, [fetchPublished]);

  async function handleSubscribe(publisherUserId: string, strategyId: string) {
    if (!userId) { message.warning('No user ID set'); return; }
    try {
      await marketplaceClient.subscribe({ userId, publisherUserId, strategyId, kind: 'copy_trade' });
      message.success('Subscribed!');
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : 'Subscribe failed');
    }
  }

  async function handlePublish(strategyId: string) {
    if (!userId) { message.warning('No user ID set'); return; }
    try {
      await marketplaceClient.publishStrategy({ userId, strategyId });
      message.success('Published!');
      fetchPublished();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : 'Publish failed');
    }
  }

  return (
    <Card title="Strategy Marketplace">
      <StatusResult
        loading={loading}
        error={error}
        empty={strategies.length === 0}
        emptyText={t('common.noData', { defaultValue: 'No strategies published' })}
        onRetry={fetchPublished}
      >
        <List
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
                </Space>,
              ]}
            >
              <List.Item.Meta
                title={s.strategyName || s.strategyId}
                description={`by ${s.publisherUserId ? s.publisherUserId.slice(0, 8) : 'unknown'}... | ${s.strategyId}`}
              />
            </List.Item>
          )}
        />
      </StatusResult>
    </Card>
  );
}
