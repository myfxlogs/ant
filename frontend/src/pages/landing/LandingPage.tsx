import { Typography, Button, Row, Col, Card, Space } from 'antd';
import {
  BarChartOutlined, ThunderboltOutlined, RobotOutlined, SafetyOutlined,
  ShopOutlined, DashboardOutlined, RightOutlined, GlobalOutlined,
  SyncOutlined, LineChartOutlined, TeamOutlined,
} from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import Seo from '@/components/common/Seo';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';

const { Title, Text, Paragraph } = Typography;
const GOLD = '#D4AF37';

interface CardIcon { icon: React.ReactNode; tKey: string }

const featureIcons: CardIcon[] = [
  { icon: <BarChartOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'Backtest' },
  { icon: <RobotOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'AI' },
  { icon: <ThunderboltOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'Auto' },
  { icon: <SafetyOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'Risk' },
  { icon: <ShopOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'Market' },
  { icon: <DashboardOutlined style={{ fontSize: 32, color: GOLD }} />, tKey: 'Analytics' },
];

const platformIcons: CardIcon[] = [
  { icon: <GlobalOutlined style={{ fontSize: 28, color: GOLD }} />, tKey: 'Multi' },
  { icon: <SyncOutlined style={{ fontSize: 28, color: GOLD }} />, tKey: 'SSE' },
  { icon: <LineChartOutlined style={{ fontSize: 28, color: GOLD }} />, tKey: 'Indicator' },
  { icon: <TeamOutlined style={{ fontSize: 28, color: GOLD }} />, tKey: 'Agent' },
];

// Rendered only for unauthenticated users by AppRoutes conditional routing.
export default function LandingPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <>
      <Seo title={t('landing.heroTitle')} description={t('landing.heroSubtitle')} path="/" />
      <div style={{ background: 'var(--color-bg-primary, #f6f8fa)', minHeight: '100vh' }}>
        {/* ── Hero ── */}
        <section
          style={{
            background: `linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)`,
            padding: '80px 24px 64px', textAlign: 'center', color: '#fff',
          }}
        >
          <div style={{ maxWidth: 720, margin: '0 auto' }}>
            <Title level={1} style={{ color: '#fff', fontSize: 40, fontWeight: 800, marginBottom: 16 }}>
              <span style={{ background: PRIMARY_GRADIENT, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                {t('landing.heroTitle')}
              </span>
            </Title>
            <Paragraph style={{ color: 'rgba(255,255,255,0.75)', fontSize: 18, marginBottom: 32, lineHeight: 1.6 }}>
              {t('landing.heroSubtitle')}
            </Paragraph>
            <Space size="middle" wrap style={{ justifyContent: 'center' }}>
              <Button type="primary" size="large" onClick={() => navigate('/register')}
                style={{ fontWeight: 600, padding: '0 32px' }}>
                {t('landing.heroCTA')}
              </Button>
              <Button ghost size="large" onClick={() => navigate('/marketplace')}
                style={{ fontWeight: 600, padding: '0 32px' }}>
                {t('landing.heroSecondary')} <RightOutlined />
              </Button>
            </Space>
          </div>
        </section>

        {/* ── Stats ── */}
        <section style={{ padding: '48px 24px', textAlign: 'center' }}>
          <Row gutter={[32, 32]} justify="center" style={{ maxWidth: 800, margin: '0 auto' }}>
            {[
              { value: t('landing.statPlatforms'), label: t('landing.statPlatformsLabel') },
              { value: t('landing.statLanguages'), label: t('landing.statLanguagesLabel') },
              { value: t('landing.statStreaming'), label: t('landing.statStreamingLabel') },
              { value: t('landing.statFree'), label: t('landing.statFreeLabel') },
            ].map(s => (
              <Col xs={12} sm={6} key={s.label}>
                <div style={{ fontSize: 24, fontWeight: 700, color: GOLD }}>{s.value}</div>
                <Text type="secondary">{s.label}</Text>
              </Col>
            ))}
          </Row>
        </section>

        {/* ── Features ── */}
        <section style={{ padding: '64px 24px', maxWidth: 1100, margin: '0 auto' }}>
          <Title level={2} style={{ textAlign: 'center', marginBottom: 8 }}>
            {t('landing.featuresTitle')}
          </Title>
          <Paragraph type="secondary" style={{ textAlign: 'center', marginBottom: 40, fontSize: 16 }}>
            {t('landing.featuresSubtitle')}
          </Paragraph>
          <Row gutter={[24, 24]}>
            {featureIcons.map(f => (
              <Col xs={24} sm={12} lg={8} key={f.tKey}>
                <Card hoverable style={{ height: '100%', borderRadius: 12 }}>
                  <div style={{ marginBottom: 12 }}>{f.icon}</div>
                  <Title level={5} style={{ marginBottom: 8 }}>
                    {t(`landing.feature${f.tKey}Title`)}
                  </Title>
                  <Text type="secondary">{t(`landing.feature${f.tKey}Desc`)}</Text>
                </Card>
              </Col>
            ))}
          </Row>
        </section>

        {/* ── Platform ── */}
        <section style={{ padding: '64px 24px', background: '#fff', borderTop: '1px solid #e8e8e8', borderBottom: '1px solid #e8e8e8' }}>
          <div style={{ maxWidth: 1100, margin: '0 auto' }}>
            <Title level={2} style={{ textAlign: 'center', marginBottom: 8 }}>
              {t('landing.platformTitle')}
            </Title>
            <Paragraph type="secondary" style={{ textAlign: 'center', marginBottom: 40, fontSize: 16 }}>
              {t('landing.platformSubtitle')}
            </Paragraph>
            <Row gutter={[24, 24]}>
              {platformIcons.map(p => (
                <Col xs={24} sm={12} lg={6} key={p.tKey}>
                  <div style={{ textAlign: 'center', padding: 16 }}>
                    {p.icon}
                    <Title level={5} style={{ margin: '12px 0 8px' }}>
                      {t(`landing.platform${p.tKey}Title`)}
                    </Title>
                    <Text type="secondary">{t(`landing.platform${p.tKey}Desc`)}</Text>
                  </div>
                </Col>
              ))}
            </Row>
          </div>
        </section>

        {/* ── CTA ── */}
        <section style={{ padding: '80px 24px', textAlign: 'center', maxWidth: 640, margin: '0 auto' }}>
          <Title level={2}>{t('landing.ctaTitle')}</Title>
          <Paragraph type="secondary" style={{ fontSize: 16, marginBottom: 32 }}>
            {t('landing.ctaSubtitle')}
          </Paragraph>
          <Button type="primary" size="large" onClick={() => navigate('/register')}
            style={{ fontWeight: 600, padding: '0 40px' }}>
            {t('landing.ctaButton')}
          </Button>
        </section>

        {/* ── Footer ── */}
        <footer style={{ padding: '32px 24px', textAlign: 'center', borderTop: '1px solid #e8e8e8' }}>
          <Text type="secondary">
            AlphaForge &copy; {new Date().getFullYear()} &mdash; {t('landing.footer')}
          </Text>
        </footer>
      </div>
    </>
  );
}
