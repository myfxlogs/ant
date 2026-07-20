import { Typography, Button, Row, Col, Card, Space } from 'antd';
import {
  BarChartOutlined, ThunderboltOutlined, RobotOutlined, SafetyOutlined,
  ShopOutlined, DashboardOutlined, RightOutlined, GlobalOutlined,
  SyncOutlined, LineChartOutlined, TeamOutlined,
} from '@ant-design/icons';
import Seo from '@/components/common/Seo';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';

const { Title, Text, Paragraph } = Typography;

const GOLD = '#D4AF37';

const featureCards = [
  {
    icon: <BarChartOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'Strategy Backtesting',
    desc: 'Backtest MQL4/MQL5 strategies on historical data with realistic execution, commission modeling, and slippage simulation.',
  },
  {
    icon: <RobotOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'AI Optimization',
    desc: 'Leverage AI to optimize strategy parameters, analyze performance, and suggest improvements with out-of-sample validation.',
  },
  {
    icon: <ThunderboltOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'Automated Trading',
    desc: 'Deploy strategies to live MT4/MT5 accounts with real-time SSE monitoring and automatic execution.',
  },
  {
    icon: <SafetyOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'Risk Management',
    desc: 'Built-in risk controls: max drawdown limits, position sizing, trade direction filters, and circuit breakers.',
  },
  {
    icon: <ShopOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'Strategy Marketplace',
    desc: 'Browse, purchase, and deploy trading strategies created by the community. Backtest before you buy.',
  },
  {
    icon: <DashboardOutlined style={{ fontSize: 32, color: GOLD }} />,
    title: 'Performance Analytics',
    desc: 'Comprehensive analytics: equity curves, monthly breakdowns, win rates, Sharpe ratio, and trade-by-trade history.',
  },
];

const platformCards = [
  {
    icon: <GlobalOutlined style={{ fontSize: 28, color: GOLD }} />,
    title: 'Multi-Account Management',
    desc: 'Connect and manage multiple MT4/MT5 accounts from a single dashboard.',
  },
  {
    icon: <SyncOutlined style={{ fontSize: 28, color: GOLD }} />,
    title: 'Real-Time Streaming',
    desc: 'Live price feeds, position updates, and account metrics via SSE push — no polling.',
  },
  {
    icon: <LineChartOutlined style={{ fontSize: 28, color: GOLD }} />,
    title: 'Custom Indicators',
    desc: 'Built-in technical indicator library. Compose custom indicators with MQL/Python.',
  },
  {
    icon: <TeamOutlined style={{ fontSize: 28, color: GOLD }} />,
    title: 'AI Agent Assistant',
    desc: 'Natural language strategy generation. Describe your idea, let AI build and backtest it.',
  },
];

// Rendered only for unauthenticated users by AppRoutes conditional routing.
export default function LandingPage() {
  return (
    <>
      <Seo
        title="AI-Powered MT4/MT5 Strategy Backtesting & Automated Trading Platform"
        description="Create, backtest, and deploy MT4/MT5 trading strategies with AI optimization, risk management, and live execution. Connect your broker accounts and automate your trading."
        path="/"
      />
      <div style={{ background: 'var(--color-bg-primary, #f6f8fa)', minHeight: '100vh' }}>
        {/* ── Hero ── */}
        <section
          style={{
            background: `linear-gradient(135deg, #1a1a2e 0%, #16213e 50%, #0f3460 100%)`,
            padding: '80px 24px 64px',
            textAlign: 'center',
            color: '#fff',
          }}
        >
          <div style={{ maxWidth: 720, margin: '0 auto' }}>
            <Title
              level={1}
              style={{
                color: '#fff', fontSize: 40, fontWeight: 800, marginBottom: 16,
              }}
            >
              AI-Powered MT4/MT5{' '}
              <span style={{ background: PRIMARY_GRADIENT, WebkitBackgroundClip: 'text', WebkitTextFillColor: 'transparent' }}>
                Strategy Platform
              </span>
            </Title>
            <Paragraph style={{ color: 'rgba(255,255,255,0.75)', fontSize: 18, marginBottom: 32, lineHeight: 1.6 }}>
              Create, backtest, and deploy MT4/MT5 trading strategies with AI optimization,
              built-in risk management, and real-time execution. No credit card required.
            </Paragraph>
            <Space size="middle" wrap style={{ justifyContent: 'center' }}>
              <Button
                type="primary"
                size="large"
                onClick={() => navigate('/register')}
                style={{ fontWeight: 600, padding: '0 32px' }}
              >
                Get Started Free
              </Button>
              <Button
                ghost
                size="large"
                onClick={() => navigate('/marketplace')}
                style={{ fontWeight: 600, padding: '0 32px' }}
              >
                Explore Marketplace <RightOutlined />
              </Button>
            </Space>
          </div>
        </section>

        {/* ── Stats ── */}
        <section style={{ padding: '48px 24px', textAlign: 'center' }}>
          <Row gutter={[32, 32]} justify="center" style={{ maxWidth: 800, margin: '0 auto' }}>
            {[
              { value: 'MT4 + MT5', label: 'Platform Support' },
              { value: 'MQL + Python', label: 'Strategy Languages' },
              { value: 'Real-time SSE', label: 'Data Streaming' },
              { value: 'Free to Start', label: 'No Credit Card' },
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
            Everything You Need to Trade Smarter
          </Title>
          <Paragraph type="secondary" style={{ textAlign: 'center', marginBottom: 40, fontSize: 16 }}>
            From backtesting to live deployment — all in one platform.
          </Paragraph>
          <Row gutter={[24, 24]}>
            {featureCards.map(f => (
              <Col xs={24} sm={12} lg={8} key={f.title}>
                <Card hoverable style={{ height: '100%', borderRadius: 12 }}>
                  <div style={{ marginBottom: 12 }}>{f.icon}</div>
                  <Title level={5} style={{ marginBottom: 8 }}>{f.title}</Title>
                  <Text type="secondary">{f.desc}</Text>
                </Card>
              </Col>
            ))}
          </Row>
        </section>

        {/* ── Platform ── */}
        <section style={{ padding: '64px 24px', background: '#fff', borderTop: '1px solid #e8e8e8', borderBottom: '1px solid #e8e8e8' }}>
          <div style={{ maxWidth: 1100, margin: '0 auto' }}>
            <Title level={2} style={{ textAlign: 'center', marginBottom: 8 }}>
              Built for Professional Traders
            </Title>
            <Paragraph type="secondary" style={{ textAlign: 'center', marginBottom: 40, fontSize: 16 }}>
              Enterprise-grade infrastructure, designed for reliability and performance.
            </Paragraph>
            <Row gutter={[24, 24]}>
              {platformCards.map(p => (
                <Col xs={24} sm={12} lg={6} key={p.title}>
                  <div style={{ textAlign: 'center', padding: 16 }}>
                    {p.icon}
                    <Title level={5} style={{ margin: '12px 0 8px' }}>{p.title}</Title>
                    <Text type="secondary">{p.desc}</Text>
                  </div>
                </Col>
              ))}
            </Row>
          </div>
        </section>

        {/* ── CTA ── */}
        <section style={{ padding: '80px 24px', textAlign: 'center', maxWidth: 640, margin: '0 auto' }}>
          <Title level={2}>Ready to Automate Your Trading?</Title>
          <Paragraph type="secondary" style={{ fontSize: 16, marginBottom: 32 }}>
            Join traders who use AlphaForge to build, backtest, and deploy their strategies.
            Start for free — no credit card required.
          </Paragraph>
          <Button
            type="primary"
            size="large"
            onClick={() => navigate('/register')}
            style={{ fontWeight: 600, padding: '0 40px' }}
          >
            Create Free Account
          </Button>
        </section>

        {/* ── Footer ── */}
        <footer style={{ padding: '32px 24px', textAlign: 'center', borderTop: '1px solid #e8e8e8' }}>
          <Text type="secondary">
            AlphaForge © {new Date().getFullYear()} — AI-Powered MT4/MT5 Strategy Platform
          </Text>
        </footer>
      </div>
    </>
  );
}
