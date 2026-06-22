import { Input, Select, Row, Col, Pagination } from 'antd';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';
import { useMarketplaceCtx } from '../MarketplaceContext';
import StrategyMarketCard from './StrategyMarketCard';
import type { PriceFilter, SortBy } from '../hooks/useMarketplace';

export default function MarketTab() {
  const { t } = useTranslation();
  const m = useMarketplaceCtx();

  return (
    <div>
      {/* Toolbar */}
      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Input
            placeholder={t('marketplace.searchPlaceholder')}
            value={m.searchText}
            onChange={e => m.setSearchText(e.target.value)}
          />
        </Col>
        <Col xs={12} sm={4}>
          <Select value={m.priceFilter} onChange={v => m.setPriceFilter(v as PriceFilter)} style={{ width: '100%' }}
            options={[
              { value: 'all', label: t('marketplace.filter.all') },
              { value: 'free', label: t('marketplace.filter.free') },
              { value: 'paid', label: t('marketplace.filter.paid') },
            ]} />
        </Col>
        <Col xs={12} sm={6}>
          <Select value={m.sortBy} onChange={v => m.setSortBy(v as SortBy)} style={{ width: '100%' }}
            options={[
              { value: 'score', label: t('marketplace.sort.score') },
              { value: 'newest', label: t('marketplace.sort.newest') },
              { value: 'popular', label: t('marketplace.sort.popular') },
              { value: 'rating', label: t('marketplace.sort.rating') },
              { value: 'price_asc', label: t('marketplace.sort.priceAsc') },
              { value: 'price_desc', label: t('marketplace.sort.priceDesc') },
            ]} />
        </Col>
      </Row>

      <StatusResult loading={m.loading} error={m.error instanceof Error ? m.error.message : undefined}
        onRetry={m.refetch}
        empty={!m.loading && m.strategies.length === 0}
        emptyText={t('marketplace.empty')}>
        <Row gutter={[12, 12]}>
          {m.strategies.map(s => (
            <Col key={s.publishId || s.strategyId} xs={24} sm={12} md={8} lg={6}>
              <StrategyMarketCard
                strategy={s}
                isPurchased={m.isPurchased(s.strategyId)}
                isOwner={m.isOwner(s.strategyId)}
                onOpenDetail={m.openDetail}
                onGetFree={m.handleGetFree}
              />
            </Col>
          ))}
        </Row>

        {m.total > m.pageSize && (
          <div style={{ textAlign: 'center', marginTop: 20 }}>
            <Pagination
              current={m.page} pageSize={m.pageSize} total={m.total}
              showSizeChanger showQuickJumper
              showTotal={(t) => `Total ${t} strategies`}
              onChange={(p, ps) => { m.setPage(p); m.setPageSize(ps); }}
            />
          </div>
        )}
      </StatusResult>
    </div>
  );
}
