import { Tabs, Typography, Grid } from 'antd';
import { ShopOutlined, BookOutlined, UserOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useMarketplace } from './hooks/useMarketplace';
import { MarketplaceProvider } from './MarketplaceContext';
import MarketTab from './components/MarketTab';
import PurchaseTab from './components/PurchaseTab';
import AuthorTab from './components/AuthorTab';
import StrategyDetailModal from './components/StrategyDetailModal';

const { Title, Text } = Typography;

function MarketplaceTabs() {
  const { t } = useTranslation();
  const m = useMarketplace();

  return (
    <Tabs
      activeKey={m.activeTab}
      onChange={k => m.setActiveTab(k as any)}
      items={[
        {
          key: 'market',
          label: <span><ShopOutlined /> {t('marketplace.tabs.marketplace')}</span>,
          children: <MarketTab />,
        },
        {
          key: 'purchases',
          label: <span><BookOutlined /> {t('marketplace.tabs.purchases', 'My Purchases')}</span>,
          children: <PurchaseTab />,
        },
        {
          key: 'author',
          label: <span><UserOutlined /> {t('marketplace.tabs.author', 'Author Center')}</span>,
          children: <AuthorTab />,
        },
      ]}
    />
  );
}

function MarketplaceModals() {
  const m = useMarketplace();
  return (
    <StrategyDetailModal
      strategy={m.detailStrategy}
      open={m.detailOpen}
      isPurchased={m.detailStrategy ? m.isPurchased(m.detailStrategy.strategyId) : false}
      onClose={m.closeDetail}
      onGetFree={m.handleGetFree}
      onBuy={m.handleBuy}
    />
  );
}

export default function MarketplacePage() {
  const { t } = useTranslation();
  const m = useMarketplace();
  const screens = Grid.useBreakpoint();
  const isMobile = !screens.sm;

  return (
    <MarketplaceProvider value={m}>
      <div style={{ padding: isMobile ? '16px 12px' : '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-7xl mx-auto">
          <div style={{ marginBottom: 20 }}>
            <Title level={3} style={{ margin: 0 }}>
              <ShopOutlined style={{ marginRight: 8 }} />{t('marketplace.title')}
            </Title>
            <Text type="secondary">{t('marketplace.subtitle')}</Text>
          </div>
          <MarketplaceTabs />
          <MarketplaceModals />
        </div>
      </div>
    </MarketplaceProvider>
  );
}
