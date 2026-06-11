import { Input, Select, Row, Col, Pagination } from 'antd';
import { SearchOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';
import StrategyMarketCard from './StrategyMarketCard';
import type { PublishedStrategy } from '@/gen/ant/v1/marketplace_service_pb';
import type { PriceFilter, SortBy } from '../hooks/useMarketplace';

interface Props {
  strategies: PublishedStrategy[];
  loading: boolean;
  error: unknown;
  searchText: string;
  onSearchChange: (v: string) => void;
  priceFilter: PriceFilter;
  onPriceFilterChange: (v: PriceFilter) => void;
  sortBy: SortBy;
  onSortChange: (v: SortBy) => void;
  onRefresh: () => void;
  isPurchased: (id: string) => boolean;
  onOpenDetail: (s: PublishedStrategy) => void;
  onGetFree: (s: PublishedStrategy) => void;
}

export default function MarketTab(props: Props) {
  const { t } = useTranslation();
  const { strategies, loading, searchText, onSearchChange, priceFilter, onPriceFilterChange, sortBy, onSortChange, onRefresh } = props;

  return (
    <div>
      {/* Toolbar */}
      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8}>
          <Input prefix={<SearchOutlined />} placeholder={t('marketplace.searchPlaceholder')}
            value={searchText} onChange={e => onSearchChange(e.target.value)} allowClear />
        </Col>
        <Col xs={12} sm={4}>
          <Select value={priceFilter} onChange={v => onPriceFilterChange(v as PriceFilter)} style={{ width: '100%' }}
            options={[
              { value: 'all', label: t('marketplace.filter.all') },
              { value: 'free', label: t('marketplace.filter.free') },
              { value: 'paid', label: t('marketplace.filter.paid') },
            ]} />
        </Col>
        <Col xs={12} sm={6}>
          <Select value={sortBy} onChange={v => onSortChange(v as SortBy)} style={{ width: '100%' }}
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

      <StatusResult loading={loading} error={props.error instanceof Error ? props.error.message : undefined}
        onRetry={onRefresh}
        empty={!loading && strategies.length === 0}
        emptyText={t('marketplace.empty')}>
        <Row gutter={[12, 12]}>
          {strategies.map(s => (
            <Col key={s.publishId || s.strategyId} xs={24} sm={12} md={8} lg={6}>
              <StrategyMarketCard
                strategy={s}
                isPurchased={props.isPurchased(s.strategyId)}
                onOpenDetail={props.onOpenDetail}
                onGetFree={props.onGetFree}
              />
            </Col>
          ))}
        </Row>
      </StatusResult>
    </div>
  );
}
