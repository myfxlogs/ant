import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { Card, Spin, Tag, Typography, Empty, Row, Col, Table, Statistic, Select, Progress, Divider } from 'antd';
import { RiseOutlined, FallOutlined, TrophyOutlined, ClockCircleOutlined, BarChartOutlined, GlobalOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { normalizeLanguage, setLanguage, SUPPORTED_LANGUAGES, type SupportedLanguage } from '@/i18n';
import { PRIMARY_GRADIENT } from '@/components/common/GradientButton';
import { PieChart, Pie, Cell, ResponsiveContainer, Tooltip as RechartsTooltip } from 'recharts';
import ShareChart from './ShareChart';

const { Text } = Typography;

const LANGUAGE_LABELS: Record<SupportedLanguage, string> = {
  'zh-cn': '简体中文',
  'zh-tw': '繁體中文',
  en: 'English',
  ja: '日本語',
  vi: 'Tiếng Việt',
};

function BrandLogo({ name }: { name: string }) {
  return (
    <div style={{ display: 'inline-flex', alignItems: 'center', gap: 10 }}>
      <span style={{ width: 40, height: 40, borderRadius: 12, background: PRIMARY_GRADIENT, display: 'inline-flex', alignItems: 'center', justifyContent: 'center' }}>
        <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="#fff" strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">
          <path d="M13 7h8m0 0v8m0-8l-8 8-4-4-6 6" />
        </svg>
      </span>
      <span style={{ fontWeight: 700, fontSize: 'clamp(16px, 4vw, 20px)', fontFamily: 'Poppins, sans-serif' }}>{name}</span>
    </div>
  );
}

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
  const [lang, setLang] = useState<SupportedLanguage>(normalizeLanguage(navigator.language));

  // Default to the viewer's device/browser language (the viewer is usually a
  // third party, not the share-link creator). They can override via the selector.
  useEffect(() => {
    const dev = normalizeLanguage(navigator.language);
    setLang(dev);
    setLanguage(dev);
  }, []);

  const changeLang = (l: SupportedLanguage) => { setLang(l); setLanguage(l); };

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

  const appName = t('app.name', { defaultValue: 'AntTrader' });
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

  const trades = data.trades || [];
  const netProfit = toNum(data.totalReturn);
  const isPositive = netProfit >= 0;
  const green = '#52c41a', red = '#ff4d4f';

  // Derived metrics computed client-side from the trade list.
  const profits = trades.map(tr => toNum(tr.profit));
  const winningProfits = profits.filter(p => p > 0);
  const losingProfits = profits.filter(p => p < 0);
  const winningTrades = winningProfits.length;
  const losingTrades = losingProfits.length;
  const bestTrade = profits.length ? Math.max(...profits) : 0;
  const worstTrade = profits.length ? Math.min(...profits) : 0;
  const avgWin = winningTrades ? winningProfits.reduce((a, b) => a + b, 0) / winningTrades : 0;
  const avgLoss = losingTrades ? losingProfits.reduce((a, b) => a + b, 0) / losingTrades : 0;
  const winPct = (winningTrades + losingTrades) > 0 ? Math.round(winningTrades / (winningTrades + losingTrades) * 100) : 0;

  // Real peak-to-trough max drawdown (%) from the equity curve when available;
  // falls back to the backend-provided value otherwise.
  const equity = data.equityCurve || [];
  let maxDrawdownPct = toNum(data.maxDrawdown);
  if (equity.length > 1) {
    let peak = -Infinity, maxDD = 0;
    for (const raw of equity) {
      const e = toNum(raw);
      if (e > peak) peak = e;
      if (peak > 0) { const dd = (peak - e) / peak * 100; if (dd > maxDD) maxDD = dd; }
    }
    maxDrawdownPct = maxDD;
  }

  // Per-symbol aggregation (count + net profit), sorted by net profit.
  const symbolMap = new Map<string, { symbol: string; count: number; net: number }>();
  for (const tr of trades) {
    const k = tr.symbol || '-';
    const cur = symbolMap.get(k) || { symbol: k, count: 0, net: 0 };
    cur.count += 1;
    cur.net += toNum(tr.profit);
    symbolMap.set(k, cur);
  }
  const bySymbol = Array.from(symbolMap.values()).sort((a, b) => b.net - a.net);

  const money = (n: number, d = 2) => Number.isFinite(n)
    ? n.toLocaleString(i18n.language, { minimumFractionDigits: d, maximumFractionDigits: d })
    : '-';
  const signed = (n: number) => `${n >= 0 ? '+' : ''}${money(n)}`;

  const kpiCards = [
    { label: t('sharePage.winRate'), value: `${fmt(toNum(data.winRate), 1)}%`, color: '#1677ff', icon: null },
    { label: t('sharePage.profitFactor'), value: fmt(toNum(data.profitFactor), 2), color: '#eb2f96', icon: <TrophyOutlined /> },
    { label: t('sharePage.maxDrawdown'), value: `${fmt(maxDrawdownPct, 2)}%`, color: '#fa8c16', icon: null },
    { label: t('sharePage.sharpeRatio'), value: fmt(toNum(data.sharpeRatio), 2), color: '#a0d911', icon: <BarChartOutlined /> },
    { label: t('sharePage.totalTrades'), value: String(data.totalTrades || 0), color: '#722ed1', icon: null },
    { label: t('sharePage.totalVolume'), value: fmt(toNum(data.totalVolume), 1), color: '#13c2c2', icon: null },
    { label: t('sharePage.avgHolding'), value: avgHoldingText(toNum(data.avgHoldingMs)), color: '#2f54eb', icon: <ClockCircleOutlined /> },
    { label: t('sharePage.bestTrade'), value: signed(bestTrade), color: green, icon: <RiseOutlined /> },
    { label: t('sharePage.worstTrade'), value: signed(worstTrade), color: red, icon: <FallOutlined /> },
    { label: t('sharePage.avgWin'), value: signed(avgWin), color: green, icon: null },
    { label: t('sharePage.avgLoss'), value: signed(avgLoss), color: red, icon: null },
    { label: `${t('sharePage.winningTrades')} / ${t('sharePage.losingTrades')}`, value: `${winningTrades} / ${losingTrades}`, color: '#1677ff', icon: null },
  ];

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

  return (
    <div style={{ maxWidth: 960, margin: '0 auto', padding: 'clamp(10px, 3vw, 24px)', fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif' }}>
      {/* Top bar: brand logo + language selector */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8, flexWrap: 'wrap', marginBottom: 'clamp(14px, 3vw, 20px)' }}>
        <BrandLogo name={appName} />
        {langSelector}
      </div>

      {/* Hero — net profit + win/loss split */}
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
      {equity.length > 0 && (
        <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.equityCurve')}</span>} style={{ marginBottom: 16, borderRadius: 10 }}>
          <ShareChart data={equity} />
        </Card>
      )}

      {/* Performance by symbol — pie chart */}
      {bySymbol.length > 0 && (
        <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.bySymbol')}</span>} style={{ marginBottom: 16, borderRadius: 10 }}>
          <ResponsiveContainer width="100%" height={280}>
            <PieChart>
              <Pie data={bySymbol} dataKey="net" nameKey="symbol" cx="50%" cy="50%" outerRadius="60%" innerRadius="35%" paddingAngle={2}>
                {bySymbol.map((_, i) => <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />)}
              </Pie>
              <RechartsTooltip formatter={(v: number) => signed(v)} />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      )}

      {/* Trade list */}
      <Card size="small" title={<span style={{ fontSize: 'clamp(12px, 2.5vw, 14px)' }}>{t('sharePage.tradeRecords')} ({trades.length})</span>} style={{ borderRadius: 10 }}>
        {trades.length === 0 ? (
          <Empty description={t('sharePage.noTrades')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <Table
            dataSource={trades}
            columns={columns}
            rowKey={(_, i) => String(i)}
            size="small"
            pagination={{ pageSize: 20, size: 'small', showSizeChanger: true, pageSizeOptions: ['10', '20', '50'] }}
          />
        )}
      </Card>

      {/* Footer */}
      <Divider style={{ margin: '24px 0 12px' }} />
      <div style={{ textAlign: 'center', fontSize: 'clamp(10px, 2vw, 12px)', color: '#bbb', padding: '0 8px' }}>
        <div>{t('sharePage.footer')} · {appName}</div>
        <div style={{ marginTop: 4 }}>{t('sharePage.disclaimer')}</div>
      </div>
    </div>
  );
}
