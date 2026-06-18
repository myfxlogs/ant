import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Spin, Tag, Typography, Empty, Row, Col, Table } from 'antd';
import { RiseOutlined, FallOutlined, ShareAltOutlined, EyeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { create } from '@bufbuild/protobuf';
import { shareClient } from '@/client/connect';
import { GetSharedPerformanceRequestSchema } from '@/gen/ant/v1/share_pb';
import type { GetSharedPerformanceResponse } from '@/gen/ant/v1/share_pb';
import ShareChart from './ShareChart';

const { Title, Text } = Typography;

function fmt(n: number, d = 2) { return Number.isFinite(n) ? n.toFixed(d) : '-'; }

export default function SharePerformancePage() {
  const { token } = useParams<{ token: string }>();
  const { t } = useTranslation();
  const [data, setData] = useState<GetSharedPerformanceResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    const req = create(GetSharedPerformanceRequestSchema, { token });
    shareClient.getSharedPerformance(req)
      .then(setData)
      .catch(() => setError(t('share.page.loadFailed', { defaultValue: 'Failed to load shared performance' })))
      .finally(() => setLoading(false));
  }, [token, t]);

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Spin size="large" /></div>;
  if (error || !data) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={error || t('share.page.notFound', { defaultValue: 'Not found' })} /></div>;
  if (data.expired) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={t('share.page.expired', { defaultValue: 'This share link has expired' })} /></div>;

  const isPositive = (data.totalReturn || 0) >= 0;
  const trades = data.trades || [];

  const kpiCards = [
    { label: t('share.page.totalReturn', { defaultValue: 'Total Return' }), value: `${isPositive ? '+' : ''}${fmt(data.totalReturn || 0, 2)}%`, color: isPositive ? '#52c41a' : '#ff4d4f', icon: isPositive ? <RiseOutlined /> : <FallOutlined /> },
    { label: t('share.page.winRate', { defaultValue: 'Win Rate' }), value: `${fmt(data.winRate || 0, 1)}%`, color: '#1677ff', icon: null },
    { label: t('share.page.maxDrawdown', { defaultValue: 'Max Drawdown' }), value: `${fmt(data.maxDrawdown || 0, 2)}%`, color: '#fa8c16', icon: null },
    { label: t('share.page.totalTrades', { defaultValue: 'Total Trades' }), value: String(data.totalTrades || 0), color: '#722ed1', icon: null },
  ];

  const columns = [
    { title: t('share.page.symbol', { defaultValue: 'Symbol' }), dataIndex: 'symbol', key: 'symbol', width: 80, responsive: ['xs' as const, 'sm' as const] },
    { title: t('share.page.side', { defaultValue: 'Side' }), dataIndex: 'side', key: 'side', width: 60,
      render: (v: string) => <Tag color={v?.toLowerCase() === 'buy' ? 'green' : 'red'}>{v}</Tag> },
    { title: t('share.page.volume', { defaultValue: 'Volume' }), dataIndex: 'volume', key: 'volume', width: 70, render: (v: number) => v?.toFixed(2) },
    { title: t('share.page.profit', { defaultValue: 'Profit' }), dataIndex: 'profit', key: 'profit', width: 90,
      render: (v: number) => <span style={{ color: v >= 0 ? '#52c41a' : '#ff4d4f', fontWeight: 500 }}>{fmt(v, 2)}</span> },
    { title: t('share.page.closeTime', { defaultValue: 'Close' }), dataIndex: 'closeTimeMs', key: 'closeTimeMs',
      render: (v: number) => v ? new Date(v).toLocaleDateString() : '-' },
  ];

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: 'clamp(12px, 3vw, 24px)' }}>
      {/* Header */}
      <div style={{ textAlign: 'center', marginBottom: 20 }}>
        <Title level={3} style={{ marginBottom: 4, fontSize: 'clamp(18px, 5vw, 24px)' }}>
          <ShareAltOutlined /> {t('share.page.title', { defaultValue: 'Trading Performance' })}
        </Title>
        <div style={{ display: 'flex', justifyContent: 'center', gap: 16, flexWrap: 'wrap', color: '#8c8c8c', fontSize: 13 }}>
          <span><Text>{data.userName || '-'}</Text></span>
          {data.totalTrades ? <span><EyeOutlined /> {data.totalTrades} {t('share.page.views', { defaultValue: 'views' })}</span> : null}
        </div>
      </div>

      {/* KPI cards */}
      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        {kpiCards.map(({ label, value, color, icon }, i) => (
          <Col xs={12} sm={6} key={i}>
            <Card size="small" style={{ textAlign: 'center', borderRadius: 10 }}>
              <div style={{ fontSize: 11, color: '#8c8c8c', marginBottom: 4 }}>{label}</div>
              <Title level={4} style={{ margin: 0, color, fontSize: 'clamp(16px, 4vw, 22px)' }}>
                {icon}{value}
              </Title>
            </Card>
          </Col>
        ))}
      </Row>

      {/* Equity curve */}
      {(data.equityCurve?.length || 0) > 0 && (
        <Card size="small" title={t('share.page.equityCurve', { defaultValue: 'Equity Curve' })} style={{ marginBottom: 16, borderRadius: 10 }}>
          <ShareChart data={data.equityCurve} />
        </Card>
      )}

      {/* Trade list */}
      <Card size="small" title={`${t('share.page.tradeRecords', { defaultValue: 'Trade Records' })} (${trades.length})`} style={{ borderRadius: 10 }}>
        {trades.length === 0 ? (
          <Empty description={t('share.page.noTrades', { defaultValue: 'No trade records yet' })} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            dataSource={trades}
            columns={columns}
            rowKey={(_, i) => String(i)}
            size="small"
            pagination={{ pageSize: 20, size: 'small' }}
            scroll={{ x: 400 }}
          />
        )}
      </Card>

      {/* Footer */}
      <div style={{ textAlign: 'center', marginTop: 24, fontSize: 11, color: '#bbb', padding: '0 8px' }}>
        {t('share.page.footer', { defaultValue: 'Generated by AntTrader · This link will expire' })}
      </div>
    </div>
  );
}
