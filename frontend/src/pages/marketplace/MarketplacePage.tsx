import { Tabs, Typography, Grid } from 'antd';
import { ShopOutlined, BookOutlined, UserOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useMarketplace } from './hooks/useMarketplace';
import MarketTab from './components/MarketTab';
import PurchaseTab from './components/PurchaseTab';
import AuthorTab from './components/AuthorTab';
import StrategyDetailModal from './components/StrategyDetailModal';

const { Title, Text } = Typography;

export default function MarketplacePage() {
  const { t } = useTranslation();
  const m = useMarketplace();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.sm;

  return (
    <div style={{ padding: isMobile ? '16px 12px' : '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
      <div className="max-w-7xl mx-auto">
        {/* Header */}
        <div style={{ marginBottom: 20 }}>
          <Title level={3} style={{ margin: 0 }}>
            <ShopOutlined style={{ marginRight: 8 }} />{t('marketplace.title')}
          </Title>
          <Text type="secondary">{t('marketplace.subtitle')}</Text>
        </div>

        {/* Tabs */}
        <Tabs
          activeKey={m.activeTab}
          onChange={k => m.setActiveTab(k as any)}
          items={[
            {
              key: 'market',
              label: <span><ShopOutlined /> {t('marketplace.tabs.marketplace')}</span>,
              children: (
                <MarketTab
                  strategies={m.strategies}
                  loading={m.loading}
                  error={m.error}
                  searchText={m.searchText}
                  onSearchChange={m.setSearchText}
                  priceFilter={m.priceFilter}
                  onPriceFilterChange={m.setPriceFilter}
                  sortBy={m.sortBy}
                  onSortChange={m.setSortBy}
                  onRefresh={m.refetch}
                  isPurchased={m.isPurchased}
                  onOpenDetail={m.openDetail}
                  onGetFree={m.handleGetFree}
                />
              ),
            },
            {
              key: 'purchases',
              label: <span><BookOutlined /> {t('marketplace.tabs.purchases', '我的购买')}</span>,
              children: (
                <PurchaseTab
                  purchases={m.purchases}
                  loading={false}
                  onViewDetail={(id) => {
                    const s = m.strategies.find(s => s.strategyId === id);
                    if (s) m.openDetail(s);
                  }}
                  onOpenInWorkspace={(id) => {
                    window.open(`/strategy/workspace?templateId=${id}`, '_blank');
                  }}
                />
              ),
            },
            {
              key: 'author',
              label: <span><UserOutlined /> {t('marketplace.tabs.author', '作者中心')}</span>,
              children: (
                <AuthorTab
                  published={m.myPublished}
                  stats={m.authorStats}
                />
              ),
            },
          ]}
        />

        {/* Detail modal */}
        <StrategyDetailModal
          strategy={m.detailStrategy}
          open={m.detailOpen}
          isPurchased={m.detailStrategy ? m.isPurchased(m.detailStrategy.strategyId) : false}
          onClose={m.closeDetail}
          onGetFree={m.handleGetFree}
          onBuy={m.handleBuy}
        />
      </div>
    </div>
  );
}
