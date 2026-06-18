import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Spin, Tag, Typography, Empty, Row, Col, Table, Statistic } from 'antd';
import { RiseOutlined, FallOutlined, ShareAltOutlined, TrophyOutlined, ClockCircleOutlined, BarChartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import ShareChart from './ShareChart';

const { Title, Text } = Typography;

function toNum(v: unknown): number {
  if (typeof v === 'bigint') return Number(v);
  if (typeof v === 'number') return v;
  return 0;
}

interface ShareData {
  userName: string;
  totalReturn: number;
  winRate: number;
  maxDrawdown: number;
  totalTrades: number;
  totalVolume: number;
  profitFactor: number;
  avgHoldingMs: number;
  sharpeRatio: number;
  equityCurve: number[];
  trades: Array<{ symbol: string; side: string; volume: number; profit: number; closeTimeMs: number }>;
  expired?: boolean;
}

function fmt(n: number, d = 2) { return Number.isFinite(n) ? n.toFixed(d) : '-'; }

function avgHoldingText(ms: number) {
  if (ms <= 0) return '-';
  const h = ms / 3600000;
  if (h < 1) return `${Math.round(ms / 60000)}m`;
  if (h < 24) return `${h.toFixed(1)}h`;
  return `${(h / 24).toFixed(1)}d`;
}

export default function SharePerformancePage() {
  const { token } = useParams<{ token: string }>();
  const { t, i18n } = useTranslation();
  const [data, setData] = useState<ShareData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    fetch(`/api/share/performance?token=${encodeURIComponent(token)}`)
      .then(r => r.json())
      .then(d => {
        if (d.expired) setError('expired');
        else setData(d);
      })
      .catch(() => setError('loadFailed'))
      .finally(() => setLoading(false));
  }, [token]);

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Spin size="large" /></div>;
  if (error === 'expired') return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={t('share.page.expired', { defaultValue: 'This share link has expired' })} /></div>;
  if (error || !data) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={t('share.page.notFound', { defaultValue: 'Not found' })} /></div>;

  const isPositive = toNum(data.totalReturn) >= 0;
  const trades = data.trades || [];

  const kpiCards = [
    { label: t('share.page.totalReturn'), value: `${isPositive ? '+' : ''}${fmt(toNum(data.totalReturn), 2)}%`, color: isPositive ? '#52c41a' : '#ff4d4f', icon: isPositive ? <RiseOutlined /> : <FallOutlined /> },
    { label: t('share.page.winRate'), value: `${fmt(toNum(data.winRate), 1)}%`, color: '#1677ff', icon: null },
    { label: t('share.page.maxDrawdown'), value: `${fmt(toNum(data.maxDrawdown), 2)}%`, color: '#fa8c16', icon: null },
    { label: t('share.page.totalTrades'), value: String(data.totalTrades || 0), color: '#722ed1', icon: null },
    { label: t('share.page.totalVolume'), value: fmt(toNum(data.totalVolume), 1), color: '#13c2c2', icon: null },
    { label: t('share.page.profitFactor'), value: fmt(toNum(data.profitFactor), 2), color: '#eb2f96', icon: <TrophyOutlined /> },
    { label: t('share.page.avgHolding'), value: avgHoldingText(toNum(data.avgHoldingMs)), color: '#2f54eb', icon: <ClockCircleOutlined /> },
    { label: t('share.page.sharpeRatio'), value: fmt(toNum(data.sharpeRatio), 2), color: '#a0d911', icon: <BarChartOutlined /> },
  ];

  const columns = [
    { title: t('share.page.symbol'), dataIndex: 'symbol', key: 'symbol', width: 80 },
    { title: t('share.page.side'), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v?.toLowerCase() === 'buy' ? 'green' : 'red'}>{v}</Tag> },
    { title: t('share.page.volume'), dataIndex: 'volume', key: 'volume', width: 70, render: (v: unknown) => toNum(v).toFixed(2) },
    { title: t('share.page.profit'), dataIndex: 'profit', key: 'profit', width: 90,
      render: (v: unknown) => { const n = toNum(v); return <span style={{ color: n >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 500 }}>{fmt(n, 2)}</span>; } },
    { title: t('share.page.closeTime'), dataIndex: 'closeTimeMs', key: 'closeTimeMs',
      render: (v: unknown) => { const ms = toNum(v); return ms ? new Date(ms).toLocaleDateString(i18n.language) : '-'; } },
  ];

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: 'clamp(10px, 3vw, 24px)', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif' }}>
      {/* Header */}
      <div style={{ textAlign: 'center', marginBottom: 'clamp(14px, 3vw, 24px)' }}>
        <Title level={3} style={{ marginBottom: 4, fontSize: 'clamp(16px, 5vw, 24px)' }}>
          <ShareAltOutlined /> {t('share.page.title', { defaultValue: 'Trading Performance' })}
        </Title>
        <Text type="secondary" style={{ fontSize: 'clamp(12px, 3vw, 14px)' }}>{data.userName || '-'}</Text>
      </div>

      {/* KPI cards — responsive 2/4 columns */}
      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        {kpiCards.map(({ label, value, color, icon }, i) => (
          <Col xs={12} sm={6} md={6} key={i}>
            <Card size="small" style={{ textAlign: 'center', borderRadius: 10 }}>
              <Statistic
                title={<span style={{ fontSize: 'clamp(10px, 2vw, 12px)', color: '#8c8c8c' }}>{label}</span>}
                value={value}
                valueStyle={{ color, fontSize: 'clamp(14px, 3.5vw, 20px)', fontWeight: 600 }}
                prefix={icon}
              />
            </Card>
          </Col>
        ))}
      </Row>

      {/* Equity curve */}
      {(data.equityCurve?.length || 0) > 0 && (
        <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('share.page.equityCurve', { defaultValue: 'Equity Curve' })}</span>} style={{ marginBottom: 16, borderRadius: 10 }}>
          <ShareChart data={data.equityCurve} />
        </Card>
      )}

      {/* Trade list */}
      <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('share.page.tradeRecords', { defaultValue: 'Trade Records' })} ({trades.length})</span>} style={{ borderRadius: 10 }}>
        {trades.length === 0 ? (
          <Empty description={t('share.page.noTrades', { defaultValue: 'No trade records yet' })} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            dataSource={trades}
            columns={columns}
            rowKey={(_, i) => String(i)}
            size="small"
            pagination={{ pageSize: 20, size: 'small' }}
            scroll={{ x: 'max-content' }}
          />
        )}
      </Card>

      {/* Footer */}
      <div style={{ textAlign: 'center', marginTop: 24, fontSize: 'clamp(10px, 2vw, 12px)', color: '#bbb', padding: '0 8px' }}>
        {t('share.page.footer', { defaultValue: 'Generated by AntTrader · This link will expire' })}
      </div>
    </div>
  );
}
