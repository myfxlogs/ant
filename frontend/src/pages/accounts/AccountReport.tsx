import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button, Select, Spin, Tag } from 'antd';
import { ArrowLeftOutlined, ReloadOutlined } from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next'
import { DETAIL_CARDS_BALANCE_KEY, DETAIL_CARDS_EQUITY_KEY, DETAIL_CARDS_FLOATING_PROFIT_KEY, DETAIL_CARDS_MARGIN_FREE_KEY, DETAIL_CARDS_MARGIN_LEVEL_KEY, DETAIL_CARDS_MARGIN_USED_KEY, REPORT_GENERATE_KEY, REPORT_PERIODS_MONTH_KEY, REPORT_PERIODS_QUARTER_KEY, REPORT_PERIODS_WEEK_KEY, REPORT_PERIODS_YEAR_KEY, REPORT_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import { useQuery } from '@tanstack/react-query';
import { analyticsApi } from '@/client/analytics';
import type { AttributionAnalysisData, RollingMetricsData, AccountAnalyticsData } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import { useAccountDetailQuery } from '@/queries/useAccountDetailQuery';
import { useAccountFinancials } from '@/queries/useAccountFinancials';
import ReportNarrative from './components/ReportNarrative';
import ReportChartPanels from './components/ReportChartPanels';
import type { MonthlyAnalysisPoint } from './components/MonthlyAnalysisCard.shared';

type Period = 'week' | 'month' | 'quarter' | 'year';

export default function AccountReport() {
  const { t, i18n } = useTranslation();
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

  const attributionQ = useQuery<AttributionAnalysisData>({
    queryKey: queryKeys.analytics.attribution(id!),
    queryFn: () => analyticsApi.getAttributionAnalysis(id!),
    enabled: !!id, staleTime: 5 * 60_000,
  });

  const rollingQ = useQuery<RollingMetricsData>({
    queryKey: queryKeys.analytics.rolling(id!),
    queryFn: () => analyticsApi.getRollingMetrics(id!),
    enabled: !!id, staleTime: 5 * 60_000,
  });

  const analyticsQ = useQuery<AccountAnalyticsData>({
    queryKey: queryKeys.analytics.detail(id!, period),
    queryFn: () => analyticsApi.getAccountAnalytics(id!, period),
    enabled: !!id, staleTime: 5 * 60_000,
  });

  const monthlyAnalysisQ = useQuery<{ years: number[]; data: unknown[] }>({
    queryKey: queryKeys.analytics.monthlyAnalysis(id!),
    queryFn: () => analyticsApi.getMonthlyAnalysis(id!),
    enabled: !!id, staleTime: 5 * 60_000,
  });

  useEffect(() => () => abortRef.current?.(), []);

  const handleGenerate = useCallback(() => {
    if (!id) return;
    abortRef.current?.();
    setGenerating(true);
    setNarrative('');
    setSections({});
    setReportError(null);
    const abort = analyticsApi.generateReportStream(id, period, i18n.language, {
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
  }, [id, period, i18n.language]);

  const { balance = 0, equity = 0, profit = 0, profitPercent = 0, margin = 0, freeMargin = 0, marginLevel = 0 } = financials;
  const analytics = analyticsQ.data;

  const derived = useMemo(() => ({
    hourlyData: (analytics?.hourlyStats || []).map((h) => ({
      ...h, hourLabel: `${String(h.hour).padStart(2, '0')}:00`,
    })),
    dailyPnLData: (analytics?.dailyPnl || []).map((d) => ({
      day: d.day, date: d.date, profit: d.pnl, trades: d.trades, lots: d.lots,
      balance: d.balance, profitFactor: d.profitFactor,
      maxFloatingLossAmount: d.maxFloatingLossAmount, maxFloatingLossRatio: d.maxFloatingLossRatio,
      maxFloatingProfitAmount: d.maxFloatingProfitAmount, maxFloatingProfitRatio: d.maxFloatingProfitRatio,
    })),
  }), [analytics]);

  const monthlyAnalysisYears = monthlyAnalysisQ.data?.years ?? [];
  const raw = monthlyAnalysisQ.data?.data;
  const monthlyAnalysisData: MonthlyAnalysisPoint[] = Array.isArray(raw) ? raw as MonthlyAnalysisPoint[] : [];

  const periodLabels: Record<Period, string> = {
    week: t(REPORT_PERIODS_WEEK_KEY), month: t(REPORT_PERIODS_MONTH_KEY),
    quarter: t(REPORT_PERIODS_QUARTER_KEY), year: t(REPORT_PERIODS_YEAR_KEY),
  };

  if (!currentAccount) {
    return <div className="p-4 flex justify-center items-center h-64"><Spin size="large" /></div>;
  }

  return (
    <div className="min-h-screen" style={{ background: 'var(--color-bg-secondary)' }}>
      <div className="max-w-7xl mx-auto p-4">
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button type="text" icon={<ArrowLeftOutlined />}
              onClick={() => navigate(`/accounts/${id}`)} style={{ color: 'var(--color-text-muted)' }} />
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{t(REPORT_TRADING_TITLE_KEY)}</h1>
                <Tag color={currentAccount.mtType === 'MT4' ? 'blue' : 'purple'}>{currentAccount.mtType}</Tag>
              </div>
              <p style={{ color: 'var(--color-text-muted)', fontSize: 14 }}>{currentAccount.login} · {currentAccount.brokerCompany}</p>
            </div>
          </div>
          <div className="flex items-center gap-3">
            <Select value={period} onChange={(v) => setPeriod(v)} style={{ width: 120 }}
              options={Object.entries(periodLabels).map(([k, v]) => ({ value: k, label: v }))} />
            <Button type="primary" icon={<ReloadOutlined />} loading={generating} onClick={handleGenerate}>
              {t(REPORT_GENERATE_KEY)}
            </Button>
          </div>
        </div>

        {/* Account snapshot cards */}
        <div className="grid grid-cols-2 lg:grid-cols-6 gap-3 mb-6">
          {[
            { label: t(DETAIL_CARDS_TRADING_BALANCE_KEY), value: `${balance.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t(DETAIL_CARDS_TRADING_EQUITY_KEY), value: `${equity.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t(DETAIL_CARDS_FLOATING_TRADING_PROFIT_KEY), value: `${profit >= 0 ? '+' : ''}${profit.toFixed(2)} (${profitPercent >= 0 ? '+' : ''}${profitPercent.toFixed(2)}%)`, color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' },
            { label: t(DETAIL_CARDS_MARGIN_USED_KEY), value: `${margin.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t(DETAIL_CARDS_MARGIN_FREE_KEY), value: `${freeMargin.toFixed(2)} ${currentAccount.currency || 'USD'}` },
            { label: t(DETAIL_CARDS_TRADING_MARGIN_LEVEL_KEY), value: `${marginLevel.toFixed(2)}%` },
          ].map((card, i) => (
            <div key={i} className="rounded-xl p-3" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
              <div style={{ color: 'var(--color-text-muted)', fontSize: 12 }}>{card.label}</div>
              <div className="text-lg font-bold mt-1" style={{ color: card.color || 'var(--color-text)' }}>{card.value}</div>
            </div>
          ))}
        </div>

        {/* AI Report Narrative */}
        <ReportNarrative
          narrative={narrative}
          sections={sections}
          reportError={reportError}
          generating={generating}
          onNavigateAISettings={() => navigate('/strategy/workspace')}
        />

        {/* Charts */}
        <ReportChartPanels
          accountId={id}
          currency={currentAccount.currency || 'USD'}
          attribution={attributionQ.data}
          rolling={rollingQ.data}
          attributionLoading={attributionQ.isLoading}
          attributionError={attributionQ.error?.message ?? null}
          rollingLoading={rollingQ.isLoading}
          rollingError={rollingQ.error?.message ?? null}
          analyticsLoading={analyticsQ.isLoading}
          analyticsError={analyticsQ.error?.message ?? null}
          monthlyAnalysisLoading={monthlyAnalysisQ.isLoading}
          monthlyAnalysisError={monthlyAnalysisQ.error?.message ?? null}
          monthlyAnalysisYears={monthlyAnalysisYears}
          monthlyAnalysisData={monthlyAnalysisData}
          monthlyWinRates={rollingQ.data?.monthlyWinRates}
          hourlyData={derived.hourlyData}
          dailyPnLData={derived.dailyPnLData}
        />
      </div>
    </div>
  );
}
