import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Spin, Tag, Typography, Empty, Row, Col, Table, Statistic, Select, Progress, Divider } from 'antd';
import { RiseOutlined, FallOutlined, GlobalOutlined, EyeOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { normalizeLanguage, setLanguage, SUPPORTED_LANGUAGES, type SupportedLanguage } from '@/i18n';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip } from 'recharts';
import ShareChart from './ShareChart';
import Seo from '@/components/common/Seo';
import { sharePublicClient } from '@/client/connect';
import {
  LANGUAGE_LABELS, BrandLogo, toNum, fmt,
  type ShareData,
} from './SharePerformancePageHelpers';
import { computeTradeStats, buildKpiCards } from './SharePerformancePageStats';

const { Text } = Typography;

export default function SharePerformancePage() {
  const { token } = useParams<{ token: string }>();
  const { t, i18n } = useTranslation();
  const [data, setData] = useState<ShareData | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(navigator.language));
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    const dev = normalizeLanguage(navigator.language);
    setLang(dev);
    setLanguage(dev);
  }, []);

  useEffect(() => {
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    setIsDark(mq.matches);
    const handler = (e: MediaQueryListEvent) => setIsDark(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  const changeLang = (l: SupportedLanguage) => { setLang(l); setLanguage(l); };

  useEffect(() => {
    if (!token) return;
    setLoading(true);
    sharePublicClient.getSharedPerformance({ token })
      .then(resp => {
        if (resp.expired) { setError('expired'); return; }
        setData({
          userName: resp.userName,
          totalReturn: resp.totalReturn,
          winRate: resp.winRate,
          maxDrawdown: resp.maxDrawdown,
          totalTrades: resp.totalTrades,
          totalVolume: resp.totalVolume,
          profitFactor: resp.profitFactor,
          avgHoldingMs: resp.avgHoldingMs,
          sharpeRatio: resp.sharpeRatio,
          equityCurve: resp.equityCurve,
          equityTimesMs: resp.equityTimesMs,
          trades: resp.trades.map(t => ({
            symbol: t.symbol, side: t.side, volume: t.volume,
            profit: t.profit, closeTimeMs: t.closeTimeMs,
          })),
          positions: resp.positions.map(p => ({
            symbol: p.symbol, type: p.type, volume: p.volume,
            openPrice: p.openPrice, profit: p.profit,
          })),
          showPositions: resp.showPositions,
        });
      })
      .catch(() => setError('loadFailed'))
      .finally(() => setLoading(false));
  }, [token]);

  const appName = t('app.name', { defaultValue: 'AlphaForge' });
  const langSelector = (
    <Select
      size="small"
      value={lang}
      onChange={changeLang}
      suffixIcon={<GlobalOutlined />}
      style={{ minWidth: 130 }}
      options={SUPPORTED_LANGUAGES.map(l => ({ value: l, label: LANGUAGE_LABELS[l] }))}
    />
  );

  if (loading) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Spin size="large" /></div>;
  if (error === 'expired') return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={t('sharePage.expired')} /></div>;
  if (error || !data) return <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', minHeight: '60vh' }}><Empty description={t('sharePage.notFound')} /></div>;

  const stats = computeTradeStats(data);
  const { winningTrades, losingTrades, winPct, netProfit, isPositive, bySymbol } = stats;
  const green = '#52c41a', red = '#ff4d4f';

  const equity = data.equityCurve || [];

  const money = (n: number, d = 2) => Number.isFinite(n)
    ? n.toLocaleString(i18n.language, { minimumFractionDigits: d, maximumFractionDigits: d })
    : '-';
  const signed = (n: number) => `${n >= 0 ? '+' : ''}${money(n)}`;

  const kpiCards = buildKpiCards(t, data, stats);

  const PIE_COLORS = ['#1677ff', '#52c41a', '#fa8c16', '#722ed1', '#eb2f96', '#13c2c2', '#a0d911', '#f5222d', '#2f54eb', '#faad14'];

  const columns = [
    { title: t('sharePage.symbol'), dataIndex: 'symbol', key: 'symbol', ellipsis: true },
    { title: t('sharePage.side'), dataIndex: 'side', key: 'side',
      render: (v: string) => <Tag color={v?.toLowerCase() === 'buy' ? 'green' : 'red'}>{v}</Tag> },
    { title: t('sharePage.volume'), dataIndex: 'volume', key: 'volume', render: (v: unknown) => toNum(v).toFixed(2) },
    { title: t('sharePage.profit'), dataIndex: 'profit', key: 'profit',
      render: (v: unknown) => { const n = toNum(v); return <span style={{ color: n >= 0 ? green : red, fontWeight: 500 }}>{signed(n)}</span>; } },
    { title: t('sharePage.closeTime'), dataIndex: 'closeTimeMs', key: 'closeTimeMs',
      render: (v: unknown) => { const ms = toNum(v); return ms ? new Date(ms).toLocaleDateString(i18n.language) : '-'; } },
  ];

  const pageBg = isDark ? '#141414' : '#fff';
  const pageColor = isDark ? '#e0e0e0' : '#333';
  const cardBg = isDark ? '#1f1f1f' : '#fff';

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: 'clamp(10px, 3vw, 24px)', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif', background: pageBg, color: pageColor, minHeight: '100vh' }}>
      <Seo
        title={data ? `${data.userName}'s Trading Performance` : 'Shared Performance Report'}
        description={data
          ? `${data.userName}: ${signed(toNum(data.totalReturn))} total return, ${fmt(toNum(data.winRate), 1)}% win rate, ${data.totalTrades} trades on AlphaForge.`
          : 'View shared trading performance report on AlphaForge.'}
        path={`/share/${token}`}
        ogImage={`/share/${token}/og-image`}
        keywords={data ? [data.userName, 'trading performance', 'verified track record', 'AlphaForge', 'MT4', 'MT5'] : undefined}
      />
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginBottom: 'clamp(14px, 3vw, 20px)' }}>
        <BrandLogo name={appName} dark={isDark} />
        {langSelector}
      </div>

      <Card style={{ borderRadius: 14, marginBottom: 16, textAlign: 'center', border: 'none', background: 'linear-gradient(135deg, rgba(212,175,55,0.10), rgba(184,150,11,0.03))' }}>
        <Text type="secondary" style={{ fontSize: 'clamp(11px, 2.5vw, 13px)' }}>{t('sharePage.title')} · {data.userName || '-'}</Text>
        <div style={{ fontSize: 'clamp(30px, 9vw, 52px)', fontWeight: 800, color: isPositive ? green : red, lineHeight: 1.1, margin: '6px 0' }}>
          {isPositive ? <RiseOutlined /> : <FallOutlined />} {signed(netProfit)}
        </div>
        <Text type="secondary" style={{ fontSize: 'clamp(11px, 2.5vw, 13px)' }}>{t('sharePage.netProfit')}</Text>
        <div style={{ maxWidth: 380, margin: '14px auto 0' }}>
          <Progress percent={winPct} strokeColor={green} trailColor={red} showInfo={false} size="small" />
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 11, color: '#8c8c8c', marginTop: 2 }}>
            <span style={{ color: green }}>{t('sharePage.winningTrades')}: {winningTrades}</span>
            <span style={{ color: red }}>{t('sharePage.losingTrades')}: {losingTrades}</span>
          </div>
        </div>
      </Card>

      <Row gutter={[8, 8]} style={{ marginBottom: 16 }}>
        {kpiCards.map(({ label, value, color, icon }, i) => (
          <Col xs={12} sm={6} md={6} key={i}>
            <Card size="small" style={{ textAlign: 'center', borderRadius: 10, background: cardBg }}>
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

      {equity.length > 0 && (
        <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.equityCurve')}</span>} style={{ marginBottom: 16, borderRadius: 10, background: cardBg }}>
          <ShareChart data={equity} timesMs={data.equityTimesMs} />
        </Card>
      )}

      {data.showPositions && data.positions != null ? (
        data.positions.length > 0 ? (
          <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}><EyeOutlined /> {t('sharePage.positions', { defaultValue: 'Open Positions' })} ({data.positions.length})</span>} style={{ marginBottom: 16, borderRadius: 10, background: cardBg }}>
            <Table
              dataSource={data.positions}
              columns={[
                { title: t('sharePage.symbol'), dataIndex: 'symbol', key: 'symbol', ellipsis: true },
                { title: t('sharePage.side'), dataIndex: 'type', key: 'type',
                  render: (v: string) => <Tag color={v === 'BUY' ? 'green' : 'red'}>{v}</Tag> },
                { title: t('sharePage.volume'), dataIndex: 'volume', key: 'volume', render: (v: unknown) => toNum(v).toFixed(2) },
                { title: t('sharePage.openPrice', { defaultValue: 'Open' }), dataIndex: 'openPrice', key: 'openPrice', render: (v: unknown) => toNum(v).toFixed(5) },
                { title: t('sharePage.profit'), dataIndex: 'profit', key: 'profit',
                  render: (v: unknown) => { const n = toNum(v); return <span style={{ color: n >= 0 ? green : red, fontWeight: 500 }}>{signed(n)}</span>; } },
              ]}
              rowKey={(_, i) => String(i)}
              size="small"
              pagination={false}
              scroll={{ x: 400 }}
            />
          </Card>
        ) : (
          <Card size="small" style={{ marginBottom: 16, borderRadius: 10, textAlign: 'center', background: cardBg }}>
            <EyeOutlined style={{ fontSize: 18, color: '#8c8c8c', marginBottom: 4 }} />
            <div style={{ fontSize: 12, color: '#8c8c8c' }}>{t('sharePage.noPositions', { defaultValue: 'No open positions' })}</div>
          </Card>
        )
      ) : (
        <Card size="small" style={{ marginBottom: 16, borderRadius: 10, textAlign: 'center', opacity: 0.6, background: cardBg }}>
          <LockOutlined style={{ fontSize: 24, color: '#bbb', marginBottom: 4 }} />
          <div style={{ fontSize: 12, color: '#bbb' }}>{t('sharePage.positionsLocked', { defaultValue: 'Positions hidden by creator' })}</div>
        </Card>
      )}

      {bySymbol.length > 0 && (
        <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.bySymbol')}</span>} style={{ marginBottom: 16, borderRadius: 10, background: cardBg }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 16, flexWrap: 'wrap' }}>
            <ResponsiveContainer width={110} height={110}>
              <PieChart>
                <Pie data={bySymbol} dataKey="net" nameKey="symbol" cx="50%" cy="50%" innerRadius={30} outerRadius={48} paddingAngle={2} isAnimationActive={false}>
                  {bySymbol.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
                </Pie>
                <Tooltip formatter={(v: number, n: string) => [signed(v), n]} />
              </PieChart>
            </ResponsiveContainer>
            <div style={{ flex: 1, minWidth: 160, fontSize: 'clamp(11px, 2vw, 13px)' }}>
              {bySymbol.slice(0, 8).map((s, i) => (
                <div key={s.symbol} style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 2 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                    <div style={{ width: 8, height: 8, borderRadius: '50%', flexShrink: 0, background: PIE_COLORS[i % PIE_COLORS.length] }} />
                    <span style={{ color: pageColor }}>{s.symbol}</span>
                    <span style={{ color: '#8c8c8c', fontSize: '0.9em' }}>{s.count}{t('sharePage.countUnit', { defaultValue: '笔' })}</span>
                  </div>
                  <span style={{ fontWeight: 500, color: s.net >= 0 ? green : red }}>{signed(s.net)}</span>
                </div>
              ))}
              {bySymbol.length > 8 && <div style={{ color: '#8c8c8c', fontSize: 11 }}>+{bySymbol.length - 8} more</div>}
            </div>
          </div>
        </Card>
      )}

      <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.tradeRecords')} ({trades.length})</span>} style={{ borderRadius: 10, background: cardBg }}>
        {trades.length === 0 ? (
          <Empty description={t('sharePage.noTrades')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            dataSource={trades}
            columns={columns}
            rowKey={(_, i) => String(i)}
            size="small"
            scroll={{ x: 500 }}
            pagination={{ pageSize: 20, size: 'small', showSizeChanger: true, pageSizeOptions: ['10', '20', '50'], simple: true }}
          />
        )}
      </Card>

      <Divider style={{ margin: '24px 0 12px' }} />
      <div style={{ textAlign: 'center', fontSize: 'clamp(10px, 2vw, 12px)', color: '#bbb', padding: '0 8px' }}>
        <div>{t('sharePage.footer')} · {appName}</div>
        <div style={{ marginTop: 4 }}>{t('sharePage.disclaimer')}</div>
      </div>
    </div>
  );
}
