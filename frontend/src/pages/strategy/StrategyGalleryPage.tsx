import { useState, useCallback, useDeferredValue } from 'react';
import { Typography, Button, Input, Segmented, Select, Row, Col, Spin, Empty, Alert, Space, message } from 'antd';
import { PlusOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { strategyApi } from '@/client/strategy';
import { queryKeys } from '@/queries/queryKeys';
import Seo from '@/components/common/Seo';
import { StrategyCard } from './components/StrategyCard';

const { Title } = Typography;

type FilterKey = 'all' | 'mine' | 'preset';
type SortKey = 'recent' | 'return' | 'risk' | 'usage';

export default function StrategyGalleryPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [search, setSearch] = useState('');
  const deferredSearch = useDeferredValue(search);
  const [filter, setFilter] = useState<FilterKey>('all');
  const [sort, setSort] = useState<SortKey>('recent');
  const [creating, setCreating] = useState(false);

  const { data, isLoading, isError } = useQuery({
    queryKey: queryKeys.strategyCards.list({ filter, sort, search: deferredSearch }),
    queryFn: () => strategyApi.listStrategyCards({ filter, sort, search: deferredSearch }),
  });
  const cards = data?.cards ?? [];

  const handleNew = useCallback(async () => {
    setCreating(true);
    try {
      const draft = await strategyApi.createTemplateDraft({ name: 'Untitled Strategy' });
      if (!draft.id) throw new Error('Draft creation returned empty id');
      navigate(`/strategy/${draft.id}/edit`);
    } catch (e: unknown) {
      message.error(e instanceof Error ? e.message : t('strategy.gallery.createFailed', { defaultValue: 'Failed to create strategy' }));
    } finally {
      setCreating(false);
    }
  }, [navigate, t]);

  return (
    <>
      <Seo title={t('strategy.gallery.title', { defaultValue: 'Strategies' })} path="/strategy" />
      <div style={{ padding: '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-7xl mx-auto">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
            <Title level={3} style={{ margin: 0 }}>{t('strategy.gallery.title', { defaultValue: 'Strategies' })}</Title>
            <Space>
              <Button type="primary" icon={<PlusOutlined />} loading={creating} onClick={handleNew}>
                {t('strategy.gallery.create', { defaultValue: 'New Strategy' })}
              </Button>
              <Button icon={<RobotOutlined />} onClick={() => navigate('/strategy/workspace?ai=1')}>
                {t('strategy.gallery.aiGenerate', { defaultValue: 'AI Generate' })}
              </Button>
            </Space>
          </div>

          <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
            <Input
              placeholder={t('strategy.gallery.searchPlaceholder', { defaultValue: 'Search strategies...' })}
              allowClear
              value={search}
              onChange={e => setSearch(e.target.value)}
              style={{ width: 280 }}
            />
            <Segmented
              value={filter}
              onChange={v => setFilter(v as FilterKey)}
              options={[
                { value: 'all', label: t('strategy.gallery.filterAll', { defaultValue: 'All' }) },
                { value: 'mine', label: t('strategy.gallery.filterMine', { defaultValue: 'Mine' }) },
                { value: 'preset', label: t('strategy.gallery.filterSystem', { defaultValue: 'System' }) },
              ]}
            />
            <Select
              value={sort}
              onChange={v => setSort(v)}
              style={{ width: 140 }}
              options={[
                { value: 'recent', label: t('strategy.gallery.sortRecent', { defaultValue: 'Recent' }) },
                { value: 'return', label: t('strategy.gallery.sortReturn', { defaultValue: 'Return' }) },
                { value: 'risk', label: t('strategy.gallery.sortRisk', { defaultValue: 'Risk' }) },
                { value: 'usage', label: t('strategy.gallery.sortUsage', { defaultValue: 'Usage' }) },
              ]}
            />
          </div>

          {isError ? (
            <Alert type="error" showIcon message={t('strategy.gallery.loadError', { defaultValue: 'Failed to load strategies' })} />
          ) : isLoading ? (
            <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
          ) : cards.length === 0 ? (
            <Empty description={t('strategy.gallery.empty', { defaultValue: 'No strategies found' })} />
          ) : (
            <Row gutter={[16, 16]}>
              {cards.map(card => (
                <Col key={card.id} xs={24} sm={12} md={8} lg={6}>
                  <StrategyCard card={card} />
                </Col>
              ))}
            </Row>
          )}
        </div>
      </div>
    </>
  );
}
