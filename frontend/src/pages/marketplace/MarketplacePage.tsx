import { memo, useState, useEffect } from 'react';
import { Tabs, Typography, Drawer, Button, Badge, Select } from 'antd';
import { ShopOutlined, BookOutlined, UserOutlined, RobotOutlined, TrophyOutlined, SwapOutlined, GiftOutlined, ThunderboltOutlined, PercentageOutlined, GlobalOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import i18n, { normalizeLanguage, setLanguage, SUPPORTED_LANGUAGES, type SupportedLanguage } from '@/i18n';
import Seo from '@/components/common/Seo';
import { useMarketplace } from './hooks/useMarketplace';
import { MarketplaceProvider } from './MarketplaceContext';
import MarketTab from './components/MarketTab';
import PurchaseTab from './components/PurchaseTab';
import AuthorTab from './components/AuthorTab';
import AutoGeneratePanel from './components/AutoGeneratePanel';
import LeaderboardTab from './components/LeaderboardTab';
import StrategyDetailModal from './components/StrategyDetailModal';
import PaymentModal from './components/PaymentModal';
import ProtectedBacktestPanel from './components/ProtectedBacktestPanel';
import CompareModal, { useCompareSelection } from './components/CompareModal';
import BundleTab from './components/BundleTab';
import OptimizationTab from './components/OptimizationTab';
import FeeTierPanel from './components/FeeTierPanel';

const { Title, Text } = Typography;

const MarketTabMemo = memo(MarketTab);
const PurchaseTabMemo = memo(PurchaseTab);
const AuthorTabMemo = memo(AuthorTab);
const AutoGenerateMemo = memo(AutoGeneratePanel);
const LeaderboardTabMemo = memo(LeaderboardTab);
const BundleTabMemo = memo(BundleTab);
const OptimizationTabMemo = memo(OptimizationTab);
const FeeTierPanelMemo = memo(FeeTierPanel);

function MarketplaceUI() {
  const { t } = useTranslation();
  const m = useMarketplace();
  const compare = useCompareSelection();
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(i18n.language));

  useEffect(() => {
    const handler = (lng: string) => setLang(normalizeLanguage(lng));
    i18n.on('languageChanged', handler);
    return () => { i18n.off('languageChanged', handler); };
  }, []);

  const langSelector = (
    <Select
      size="small"
      value={lang}
      onChange={(l: SupportedLanguage) => { setLang(l); setLanguage(l); }}
      suffixIcon={<GlobalOutlined />}
      style={{ minWidth: 120 }}
      options={SUPPORTED_LANGUAGES.map(l => ({ value: l, label: l.toUpperCase() }))}
    />
  );

  return (
    <MarketplaceProvider value={{ ...m, compareIds: compare.compareIds, toggleCompare: compare.toggleCompare }}>
      <Seo title="Strategy Marketplace" description="Discover and purchase MT4/MT5 trading strategies. Supports IC Markets, Pepperstone, XM and 30+ brokers. AI-assisted strategy generation and optimization. Backtest verified, live performance tracked." path="/marketplace" keywords={[
        'strategy marketplace', 'buy forex EA', 'MT4 strategies', 'MT5 strategies',
        'trading robots', 'AI trading strategies', 'IC Markets', 'Pepperstone', 'XM',
        'Exness', 'OANDA', 'automated trading', 'algorithmic trading',
      ]} />
      <div style={{ padding: '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-7xl mx-auto">
          <div style={{ marginBottom: 20, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
            <div>
              <Title level={3} style={{ margin: 0 }}>
                <ShopOutlined style={{ marginRight: 8 }} />{t('marketplace.title')}
              </Title>
              <Text type="secondary">{t('marketplace.subtitle')}</Text>
            </div>
            {!m.isAuthenticated && <div>{langSelector}</div>}
          </div>
          <Tabs activeKey={m.activeTab} onChange={k => m.setActiveTab(k as any)} items={[
            { key: 'market', label: <span><ShopOutlined /> {t('marketplace.tabs.marketplace')}</span>, children: <MarketTabMemo /> },
            { key: 'leaderboard', label: <span><TrophyOutlined /> {t('marketplace.tabs.leaderboard')}</span>, children: <LeaderboardTabMemo /> },
            ...(m.isAuthenticated ? [
              { key: 'ai', label: <span><RobotOutlined /> {t('marketplace.tabs.ai')}</span>, children: <AutoGenerateMemo /> },
              { key: 'purchases', label: <span><BookOutlined /> {t('marketplace.tabs.purchases')}</span>, children: <PurchaseTabMemo /> },
              { key: 'author', label: <span><UserOutlined /> {t('marketplace.tabs.author')}</span>, children: <AuthorTabMemo /> },
              { key: 'bundles', label: <span><GiftOutlined /> {t('marketplace.tabs.bundles')}</span>, children: <BundleTabMemo /> },
              { key: 'optimization', label: <span><ThunderboltOutlined /> {t('marketplace.tabs.optimization')}</span>, children: <OptimizationTabMemo /> },
              { key: 'fees', label: <span><PercentageOutlined /> {t('marketplace.tabs.fees')}</span>, children: <FeeTierPanelMemo /> },
            ] : []),
          ]} />
          <StrategyDetailModal
            strategy={m.detailStrategy} open={m.detailOpen}
            isPurchased={m.detailStrategy ? m.isPurchased(m.detailStrategy.strategyId) : false}
            isOwner={m.detailStrategy ? m.isOwner(m.detailStrategy.strategyId) : false}
            onClose={m.closeDetail} onGetFree={m.handleGetFree} onBuy={m.handleBuy}
            onRunBacktest={m.handleRunBacktest}
          />
          <Drawer
            title={t('marketplace.backtestTitle')}
            open={m.backtestDrawerOpen}
            onClose={() => m.setBacktestDrawerOpen(false)}
            width={680}
            destroyOnClose
          >
            {m.backtestStrategyId && <ProtectedBacktestPanel strategyId={m.backtestStrategyId} />}
          </Drawer>
          <PaymentModal
            strategy={m.paymentStrategy}
            walletBalance={m.walletBalance}
            open={m.paymentModalOpen}
            loading={m.paymentLoading}
            onConfirm={m.handleConfirmPayment}
            onCancel={m.handleCancelPayment}
          />
          <CompareModal
            open={compare.compareOpen}
            strategyIds={compare.compareIds}
            onClose={() => compare.setCompareOpen(false)}
            onRemove={compare.removeFromCompare}
          />
        </div>
      </div>
      {compare.compareIds.length > 0 && (
        <div style={{ position: 'fixed', bottom: 24, right: 24, zIndex: 1000 }}>
          <Badge count={compare.compareIds.length}>
            <Button
              type="primary"
              size="large"
              icon={<SwapOutlined />}
              onClick={() => compare.setCompareOpen(true)}
            >
              {t('marketplace.compare.button')}
            </Button>
          </Badge>
        </div>
      )}
    </MarketplaceProvider>
  );
}

export default function MarketplacePage() {
  return <MarketplaceUI />;
}
