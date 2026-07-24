import { Typography, Row, Col, Card, Tag, Space, Button } from 'antd';
import { GlobalOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Seo from '@/components/common/Seo';

const { Title, Text, Paragraph } = Typography;

interface Broker {
  name: string;
  mt4: boolean;
  mt5: boolean;
  description: string;
}

const BROKERS: Broker[] = [
  { name: 'IC Markets', mt4: true, mt5: true, description: 'Raw spreads from 0.0 pips, ASIC & CySEC regulated' },
  { name: 'Pepperstone', mt4: true, mt5: true, description: 'Tight spreads, fast execution, multi-jurisdiction' },
  { name: 'XM', mt4: true, mt5: true, description: 'No requotes, flexible leverage up to 1:1000' },
  { name: 'Exness', mt4: true, mt5: true, description: 'Instant withdrawals, zero spread accounts' },
  { name: 'OANDA', mt4: true, mt5: true, description: 'Regulated in 8 jurisdictions, transparent pricing' },
  { name: 'FXTM', mt4: true, mt5: true, description: 'FXTM Trader app, educational resources' },
  { name: 'FBS', mt4: true, mt5: true, description: 'Leverage up to 1:3000, multiple account types' },
  { name: 'OctaFX', mt4: true, mt5: true, description: 'Copy trading, low minimum deposit' },
  { name: 'HotForex', mt4: true, mt5: true, description: 'Multiple account types, zero spread option' },
  { name: 'Alpari', mt4: true, mt5: true, description: 'Established broker, ECN accounts available' },
  { name: 'RoboForex', mt4: true, mt5: true, description: 'CopyFX platform, tight spreads' },
  { name: 'LiteFinance', mt4: true, mt5: true, description: 'Zero spread, social trading' },
  { name: 'AMarkets', mt4: true, mt5: true, description: 'Fast execution, multiple platforms' },
  { name: 'Tickmill', mt4: true, mt5: true, description: 'Raw spreads, pro accounts' },
  { name: 'Vantage', mt4: true, mt5: true, description: 'ASIC regulated, raw ECN' },
  { name: 'Axi', mt4: true, mt5: true, description: 'ASIC regulated, tight spreads' },
  { name: 'Admirals', mt4: true, mt5: true, description: 'Multi-asset, educational tools' },
  { name: 'FXCM', mt4: true, mt5: false, description: 'Established broker, no dealing desk' },
  { name: 'Forex.com', mt4: true, mt5: true, description: 'Global access, research tools' },
  { name: 'ThinkMarkets', mt4: true, mt5: true, description: 'Fast execution, multi-jurisdiction' },
  { name: 'FxPro', mt4: true, mt5: true, description: 'No dealing desk, multiple platforms' },
  { name: 'Deriv', mt4: true, mt5: true, description: 'Synthetic indices, low deposits' },
  { name: 'InstaForex', mt4: true, mt5: true, description: 'Copy trading, bonuses' },
  { name: 'SuperForex', mt4: true, mt5: true, description: 'No requotes, high leverage' },
  { name: 'JustMarkets', mt4: true, mt5: true, description: 'Low spreads, copy trading' },
  { name: 'BlackBull Markets', mt4: true, mt5: true, description: 'ECN pricing, fast execution' },
  { name: 'Blueberry Markets', mt4: true, mt5: true, description: 'ASIC regulated, tight spreads' },
  { name: 'Vantage FX', mt4: true, mt5: true, description: 'Raw ECN, multiple platforms' },
  { name: 'GO Markets', mt4: true, mt5: true, description: 'ASIC regulated, tight spreads' },
  { name: 'Eightcap', mt4: true, mt5: true, description: 'Raw spreads, multi-platform' },
];

export default function BrokersPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <>
      <Seo
        title={t('landing.brokersSeoTitle')}
        description={t('landing.brokersSeoDesc')}
        path="/brokers"
        keywords={[
          'MT4 broker', 'MT5 broker', 'MetaTrader broker', 'IC Markets', 'Pepperstone', 'XM',
          'Exness', 'OANDA', 'FXTM', 'FBS', 'OctaFX', 'HotForex', 'Alpari', 'RoboForex',
          'forex broker', 'automated trading broker', 'EA broker', 'copy trading broker',
        ]}
      />
      <div style={{ padding: '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-7xl mx-auto">
          <div style={{ marginBottom: 24, textAlign: 'center' }}>
            <Title level={2}>
              <GlobalOutlined style={{ marginRight: 8 }} />
              Supported Brokers
            </Title>
            <Paragraph type="secondary" style={{ maxWidth: 600, margin: '0 auto' }}>
              AlphaForge connects to 30+ MT4/MT5 brokers via the MetaTrader API. Link your existing
              broker account to start backtesting, optimizing, and deploying automated strategies.
            </Paragraph>
          </div>

          <Row gutter={[16, 16]}>
            {BROKERS.map(b => (
              <Col key={b.name} xs={24} sm={12} md={8} lg={6}>
                <Card hoverable size="small" style={{ borderRadius: 12, height: '100%' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
                    <Text strong style={{ fontSize: 15 }}>{b.name}</Text>
                    <Space size={4}>
                      {b.mt4 && <Tag color="blue" style={{ fontSize: 10 }}>MT4</Tag>}
                      {b.mt5 && <Tag color="purple" style={{ fontSize: 10 }}>MT5</Tag>}
                    </Space>
                  </div>
                  <Text type="secondary" style={{ fontSize: 12 }}>{b.description}</Text>
                  <div style={{ marginTop: 8 }}>
                    <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 12, marginRight: 4 }} />
                    <Text style={{ fontSize: 11, color: '#8c8c8c' }}>{t('landing.brokersCompatible')}</Text>
                  </div>
                </Card>
              </Col>
            ))}
          </Row>

          <div style={{ textAlign: 'center', marginTop: 40 }}>
            <Button type="primary" size="large" onClick={() => navigate('/marketplace')}>
              {t('marketplace.title', { defaultValue: 'Explore Strategies' })}
            </Button>
          </div>
        </div>
      </div>
    </>
  );
}
