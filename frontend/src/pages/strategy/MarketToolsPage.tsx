import { lazy } from 'react';
import { Tabs, Typography } from 'antd';
import { BulbOutlined, RadarChartOutlined } from '@ant-design/icons';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { PageWrapper } from '@/components/common/PageWrapper';

const AssetAnalysisPage = lazy(() => import('@/pages/strategy/AssetAnalysis'));
const MarketRegimePage = lazy(() => import('@/pages/strategy/MarketRegimePage'));

const { Title } = Typography;

export default function MarketToolsPage() {
  const { t } = useTranslation();
  const [searchParams, setSearchParams] = useSearchParams();
  const tabParam = searchParams.get('tab');
  const activeTab = tabParam === 'regime' ? 'regime' : 'symbol';

  return (
    <PageWrapper>
      <div style={{ maxWidth: 960, margin: '0 auto', padding: '24px 16px' }}>
        <Title level={3}>
          <RadarChartOutlined style={{ marginRight: 8 }} />
          {t('menu.marketTools')}
        </Title>

        <Tabs
          activeKey={activeTab}
          onChange={(key) => setSearchParams({ tab: key }, { replace: true })}
          items={[
            {
              key: 'symbol',
              label: (
                <span>
                  <BulbOutlined />
                  {t('strategy.assetAnalysis.title')}
                </span>
              ),
              children: <AssetAnalysisPage />,
            },
            {
              key: 'regime',
              label: (
                <span>
                  <RadarChartOutlined />
                  {t('strategy.marketRegime.title')}
                </span>
              ),
              children: <MarketRegimePage />,
            },
          ]}
        />
      </div>
    </PageWrapper>
  );
}
