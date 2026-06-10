import { useCallback, useEffect, useRef, useState } from 'react';
import { Button, Select, Spin, Tag, Typography } from 'antd';
import {
  ArrowLeftOutlined, ReloadOutlined, TrophyOutlined,
  PieChartOutlined, LineChartOutlined, BarChartOutlined,
  RiseOutlined, FallOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import {
  Bar, CartesianGrid, ComposedChart, Cell, Line, Pie, PieChart,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts';
import { CHART_COLORS } from '@/constants/performance';
import { analyticsApi } from '@/client/analytics';
import type {
  AttributionAnalysisData, RollingMetricsData, TradeBucket,
} from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import { useAccountDetailQuery } from '@/queries/useAccountDetailQuery';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import { StatusResult } from '@/components/common/StatusResult';

const { Title, Text, Paragraph } = Typography;

type Period = 'week' | 'month' | 'quarter' | 'year';

export default function AccountReport() {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const abortRef = useRef<() => void>();

  const [period, setPeriod] = useState<Period>('month');
  const [generating, setGenerating] = useState(false);
  const [narrative, setNarrative] = useState('');
  const [sections, setSections] = useState<{ summary?: string; findings?: string; recommendations?: string }>({});
  const [reportError, setReportError] = useState<string | null>(null);

  const accountQ = useAccountDetailQuery(id ?? '');
  const financialsQ = useAccountFinancials(id ?? '');
  const currentAccount = accountQ.data;
  const financials = financialsQ.data ?? { balance: 0, equity: 0, profit: 0, profitPercent: 0, margin: 0, freeMargin: 0, marginLevel: 0, credit: 0 };

  // Attribution data
  const attributionQ = useQuery<AttributionAnalysisData>({
    queryKey: queryKeys.analytics.attribution(id!),
    queryFn: () => analyticsApi.getAttributionAnalysis(id!),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });

  // Rolling metrics data
  const rollingQ = useQuery<RollingMetricsData>({
    queryKey: queryKeys.analytics.rolling(id!),
    queryFn: () => analyticsApi.getRollingMetrics(id!),
    enabled: !!id,
    staleTime: 5 * 60_000,
  });

  useEffect(() => () => abortRef.current?.(), []);

  const handleGenerate = useCallback(() => {
    if (!id) return;
    abortRef.current?.();
    setGenerating(true);
    setNarrative('');
    setSections({});
    setReportError(null);

    const abort = analyticsApi.generateReportStream(id, period, {
      onPhase: () => {},
      onDelta: (delta) => setNarrative((p) => p + delta),
      onSection: () => {},
      onSummary: (text) => setSections((p) => ({ ...p, summary: text })),
      onFindings: (text) => setSections((p) => ({ ...p, findings: text })),
      onRecommendations: (text) => setSections((p) => ({ ...p, recommendations: text })),
      onError: (err) => setReportError(err),
      onDone: () => setGenerating(false),
    });
    abortRef.current = abort;
  }, [id, period]);

  const { balance = 0, equity = 0, profit = 0, profitPercent = 0 } = financials;
  const attribution = attributionQ.data;
  const rolling = rollingQ.data;

  const periodLabels: Record<Period, string> = {
    week: t('accounts.report.periods.week'),
    month: t('accounts.report.periods.month'),
    quarter: t('accounts.report.periods.quarter'),
    year: t('accounts.report.periods.year'),
  };

  if (!currentAccount) {
    return <div className="p-4 flex justify-center items-center h-64"><Spin size="large" /></div>;
  }

  return (
    <div className="min-h-screen" style={{ background: '#F5F7F9' }}>
      <div className="max-w-7xl mx-auto p-4">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button type="text" icon={<ArrowLeftOutlined />}
              onClick={() => navigate(`/accounts/${id}`)} style={{ color: '#8A9AA5' }} />
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>
                  {t('accounts.report.title')}
                </h1>
                <Tag color={currentAccount.mtType === 'MT4' ? 'blue' : 'purple'}>{currentAccount.mtType}</Tag>
              </div>
              <p style={{ color: '#8A9AA5', fontSize: 14 }}>
                {currentAccount.login} · {currentAccount.brokerCompany}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Select value={period} onChange={(v) => setPeriod(v)} style={{ width: 120 }}
              options={Object.entries(periodLabels).map(([k, v]) => ({ value: k, label: v }))} />
            <Button type="primary" icon={<ReloadOutlined />} loading={generating}
              onClick={handleGenerate}>
              {t('accounts.report.generate')}
            </Button>
          </div>
        </div>

        {/* Account snapshot cards */}
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-4 mb-6">
          {[
            { label: t('accounts.detail.cards.balance'), value: `${balance.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t('accounts.detail.cards.equity'), value: `${equity.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t('accounts.detail.cards.floatingProfit'), value: `${profit >= 0 ? '+' : ''}${profit.toFixed(2)} (${profitPercent >= 0 ? '+' : ''}${profitPercent.toFixed(2)}%)`, color: profit >= 0 ? '#00A651' : '#E53935' },
          ].map((card, i) => (
            <div key={i} className="rounded-xl p-4" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
              <Text style={{ color: '#8A9AA5', fontSize: 13 }}>{card.label}</Text>
              <div className="text-xl font-bold mt-1" style={{ color: card.color || '#141D22' }}>{card.value}</div>
            </div>
          ))}
        </div>

        {/* AI Narrative */}
        {(narrative || generating) && (
          <div className="rounded-2xl p-6 mb-6" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
            <Title level={5} style={{ marginBottom: 16 }}>
              <TrophyOutlined /> {t('accounts.report.aiAnalysis')}
            </Title>
            {reportError && (
              <div className="rounded-lg p-3 mb-4" style={{ background: '#FFF2F0', border: '1px solid #FFCCC7', color: '#E53935' }}>
                {reportError}
              </div>
            )}
            {/* Structured sections */}
            {sections.summary && (
              <div className="mb-4">
                <Text strong style={{ color: '#141D22' }}>{t('accounts.report.sections.summary')}</Text>
                <Paragraph style={{ color: '#475467', marginTop: 4 }}>{sections.summary}</Paragraph>
              </div>
            )}
            {sections.findings && (
              <div className="mb-4">
                <Text strong style={{ color: '#141D22' }}>{t('accounts.report.sections.findings')}</Text>
                <Paragraph style={{ color: '#475467', marginTop: 4, whiteSpace: 'pre-wrap' }}>{sections.findings}</Paragraph>
              </div>
            )}
            {sections.recommendations && (
              <div className="mb-4">
                <Text strong style={{ color: '#141D22' }}>{t('accounts.report.sections.recommendations')}</Text>
                <Paragraph style={{ color: '#475467', marginTop: 4, whiteSpace: 'pre-wrap' }}>{sections.recommendations}</Paragraph>
              </div>
            )}
            {/* Raw streaming text if sections not parsed yet */}
            {!sections.summary && narrative && (
              <div className="rounded-lg p-4" style={{ background: '#F8FAFC', whiteSpace: 'pre-wrap', color: '#475467', fontSize: 14, lineHeight: 1.8 }}>
                {narrative}
                {generating && <span className="inline-block w-2 h-4 ml-1 animate-pulse" style={{ background: '#D4AF37' }} />}
              </div>
            )}
          </div>
        )}

        {/* Attribution Charts */}
        <StatusResult loading={attributionQ.isLoading} error={attributionQ.error?.message}>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            {/* Symbol P&L ranking */}
            <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
              <Title level={5}><BarChartOutlined /> {t('accounts.report.symbolPnL')}</Title>
              {attribution?.symbolPnls?.length ? (
                <ResponsiveContainer width="100%" height={280}>
                  <ComposedChart layout="vertical" data={attribution.symbolPnls.slice(0, 8)}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis type="number" stroke="#8A9AA5" fontSize={11} />
                    <YAxis type="category" dataKey="symbol" width={70} stroke="#8A9AA5" fontSize={11} />
                    <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                    <Bar dataKey="netProfit" radius={[0, 4, 4, 0]} isAnimationActive={false}>
                      {attribution.symbolPnls.slice(0, 8).map((_: unknown, i: number) => (
                        <Cell key={i} fill={(attribution.symbolPnls?.[i]?.netProfit ?? 0) >= 0 ? '#00A651' : '#E53935'} />
                      ))}
                    </Bar>
                  </ComposedChart>
                </ResponsiveContainer>
              ) : <div className="flex items-center justify-center h-[280px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.monthlyProfit')}</div>}
            </div>

            {/* Direction breakdown */}
            <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
              <Title level={5}><PieChartOutlined /> {t('accounts.report.direction')}</Title>
              {attribution?.direction ? (
                <div className="grid grid-cols-2 gap-4 mt-4">
                  {[
                    { label: t('accounts.report.directionLong'), color: '#00A651', d: { profit: attribution.direction.longProfit, trades: attribution.direction.longTrades, winRate: attribution.direction.longWinRate } },
                    { label: t('accounts.report.directionShort'), color: '#E53935', d: { profit: attribution.direction.shortProfit, trades: attribution.direction.shortTrades, winRate: attribution.direction.shortWinRate } },
                  ].map(({ label, color, d }) => (
                    <div key={label} className="rounded-xl p-4" style={{ border: '1px solid #E5E7EB' }}>
                      <Text strong style={{ color }}>{label}</Text>
                      <div className="mt-2 space-y-1" style={{ fontSize: 13, color: '#475467' }}>
                        <div>{t('accounts.analytics.stats.netProfit')}: <Text strong style={{ color: d.profit >= 0 ? '#00A651' : '#E53935' }}>{d.profit >= 0 ? '+' : ''}{d.profit.toFixed(2)}</Text></div>
                        <div>{t('accounts.analytics.stats.totalTrades')}: {d.trades}</div>
                        <div>{t('accounts.analytics.stats.winRate')}: {d.winRate.toFixed(1)}%</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="flex items-center justify-center h-[200px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.monthlyProfit')}</div>
              )}
            </div>

            {/* Trade profit distribution */}
            {attribution?.tradeDistribution?.profitBuckets?.length ? (
              <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                <Title level={5}><BarChartOutlined /> {t('accounts.report.tradeDistribution')}</Title>
                <ResponsiveContainer width="100%" height={250}>
                  <ComposedChart data={attribution.tradeDistribution.profitBuckets}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis dataKey="label" stroke="#8A9AA5" fontSize={10} angle={-30} textAnchor="end" height={60} />
                    <YAxis stroke="#8A9AA5" fontSize={11} />
                    <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                    <Bar dataKey="count" fill="#2196F3" radius={[4, 4, 0, 0]} isAnimationActive={false} />
                  </ComposedChart>
                </ResponsiveContainer>
              </div>
            ) : null}
          </div>
        </StatusResult>

        {/* Rolling Metrics */}
        <StatusResult loading={rollingQ.isLoading} error={rollingQ.error?.message}>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
            {/* Equity + Drawdown overlay */}
            {rolling?.equityCurve?.length ? (
              <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                <Title level={5}><LineChartOutlined /> {t('accounts.report.drawdownOverlay')}</Title>
                <ResponsiveContainer width="100%" height={280}>
                  <ComposedChart data={rolling.equityCurve.map((p, i) => ({
                    ...p,
                    drawdown: rolling.drawdownCurve?.[i]?.drawdownPercent ?? 0,
                  }))}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis dataKey="date" stroke="#8A9AA5" fontSize={10} />
                    <YAxis yAxisId="left" stroke="#8A9AA5" fontSize={11} />
                    <YAxis yAxisId="right" orientation="right" stroke="#E53935" fontSize={11} domain={[100, 0]} />
                    <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                    <Line yAxisId="left" type="monotone" dataKey="equity" stroke="#2196F3" strokeWidth={2} dot={false} name="Equity" isAnimationActive={false} />
                    <Line yAxisId="right" type="monotone" dataKey="drawdown" stroke="#E53935" strokeWidth={1} dot={false} name="DD%" isAnimationActive={false} fillOpacity={0.1} fill="#E53935" />
                  </ComposedChart>
                </ResponsiveContainer>
              </div>
            ) : null}

            {/* Drawdown events */}
            {rolling?.drawdownEvents?.length ? (
              <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                <Title level={5}><FallOutlined /> {t('accounts.report.drawdownEvents')}</Title>
                <div className="space-y-2 max-h-[280px] overflow-y-auto">
                  {rolling.drawdownEvents.map((e, i) => (
                    <div key={i} className="flex items-center justify-between rounded-lg p-3" style={{ border: '1px solid #E5E7EB' }}>
                      <div>
                        <Text style={{ color: '#141D22', fontSize: 13 }}>{e.startDate} → {e.endDate || '...'}</Text>
                        {e.recoveryDate && <Text style={{ color: '#8A9AA5', fontSize: 12, marginLeft: 8 }}>{t('accounts.report.recovered')}: {e.recoveryDate}</Text>}
                      </div>
                      <Tag color="error">{e.depthPercent.toFixed(1)}%</Tag>
                    </div>
                  ))}
                </div>
              </div>
            ) : null}

            {/* Monthly win rate trend */}
            {rolling?.monthlyWinRates?.length ? (
              <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0,0,0,0.06)' }}>
                <Title level={5}><RiseOutlined /> {t('accounts.report.winRateTrend')}</Title>
                <ResponsiveContainer width="100%" height={250}>
                  <ComposedChart data={rolling.monthlyWinRates}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis dataKey="month" stroke="#8A9AA5" fontSize={10} />
                    <YAxis stroke="#8A9AA5" fontSize={11} domain={[0, 100]} />
                    <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                    <Line type="monotone" dataKey="winRate" stroke="#00A651" strokeWidth={2} dot isAnimationActive={false} />
                  </ComposedChart>
                </ResponsiveContainer>
              </div>
            ) : null}
          </div>
        </StatusResult>
      </div>
    </div>
  );
}
