import { Card, List, Button, Space, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { useRpcQuery } from '@/hooks/useRpcQuery';
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
  const userId = localStorage.getItem('userId') || '';

  const { data: strategies = [], isLoading, error, refetch } = useRpcQuery(
    ['marketplace', 'published', userId],
    async () => {
      const resp = await marketplaceClient.listPublished({ userId, limit: 50 });
      return (resp.strategies || []) as unknown as Strategy[];
    },
  );

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
      refetch();
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : 'Publish failed');
    }
  }

  const errorMessage = error instanceof Error ? error.message : undefined;

  return (
    <Card title="Strategy Marketplace">
      <StatusResult
        loading={isLoading}
        error={errorMessage}
        empty={strategies.length === 0}
        emptyText={t('common.noData', { defaultValue: 'No strategies published' })}
        onRetry={refetch}
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
