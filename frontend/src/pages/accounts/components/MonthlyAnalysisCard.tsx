import type { MouseEvent as ReactMouseEvent } from 'react';
import { useCallback, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next'
import { ANALYTICS_MONTHLY_ANALYSIS_CHART_MAIN_TITLE_KEY, ANALYTICS_MONTHLY_ANALYSIS_METRICS_CHANGE_KEY, ANALYTICS_MONTHLY_ANALYSIS_METRICS_LOTS_KEY, ANALYTICS_MONTHLY_ANALYSIS_METRICS_PIPS_KEY, ANALYTICS_MONTHLY_ANALYSIS_METRICS_PROFIT_KEY, ANALYTICS_MONTHLY_ANALYSIS_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import { useQuery } from '@tanstack/react-query';
import { Spin } from 'antd';

import MonthlyAnalysisMainChart from './MonthlyAnalysisMainChart';
import { RiskRewardPanel, PopularityPanel, HoldingSplitPanel } from './MonthlyDrillDown';
import {
  type MonthlyAnalysisCardProps,
  type MonthlyAnalysisPoint,
  type MonthlyBarRow,
  monthFromBarClick,
  getMonthLabels,
} from './MonthlyAnalysisCard.shared';
import { analyticsApi } from '@/client/analytics';
import type { MonthlyDetailData } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';

type MetricType = 'change' | 'profit' | 'lots' | 'pips';

const METRIC_OPTIONS: { key: MetricType; labelKey: string }[] = [
  { key: 'change', labelKey: 'accounts.analytics.monthlyAnalysis.metrics.change' },
  { key: 'profit', labelKey: 'accounts.analytics.monthlyAnalysis.metrics.profit' },
  { key: 'lots', labelKey: 'accounts.analytics.monthlyAnalysis.metrics.lots' },
  { key: 'pips', labelKey: 'accounts.analytics.monthlyAnalysis.metrics.pips' },
];

export default function MonthlyAnalysisCard({ accountId, years, data, winRateData, currency = 'USD' }: MonthlyAnalysisCardProps) {
  const { t } = useTranslation();
  const [selectedYear, setSelectedYear] = useState<number>(years[years.length - 1] || new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState<number | null>(null);
  const [hoverMonth, setHoverMonth] = useState<number | null>(null);
  const [selectedMetric, setSelectedMetric] = useState<MetricType>('change');
  const displayMonth = hoverMonth ?? selectedMonth;

  // Fetch drill-down detail for the selected month.
  const monthlyDetailQ = useQuery<MonthlyDetailData>({
    queryKey: queryKeys.analytics.monthlyDetail(accountId!, selectedYear, selectedMonth!),
    queryFn: () => analyticsApi.getMonthlyDetail(accountId!, selectedYear, selectedMonth!),
    enabled: !!accountId && selectedMonth != null,
    staleTime: 10 * 60_000,
  });

  // Merge winRate data into year data points.
  const winRateMap = useMemo(() => {
    const m = new Map<number, number>();
    if (!winRateData) return m;
    for (const w of winRateData) {
      const parts = w.month.split('-');
      if (parts.length >= 2) {
        const monthNum = parseInt(parts[1], 10);
        if (monthNum >= 1 && monthNum <= 12) m.set(monthNum, w.winRate);
      }
    }
    return m;
  }, [winRateData]);

  const yearData = useMemo(() => {
    const monthMap = new Map<number, MonthlyAnalysisPoint>();
    data
      .filter((item) => item.year === selectedYear)
      .forEach((item) => monthMap.set(item.month, item));
    return Array.from({ length: 12 }, (_, index) => {
      const month = index + 1;
      const existing = monthMap.get(month);
      return {
        year: selectedYear,
        month,
        change: existing?.change ?? 0,
        profit: existing?.profit ?? 0,
        lots: existing?.lots ?? 0,
        pips: existing?.pips ?? 0,
        trades: existing?.trades ?? 0,
        winRate: winRateMap.get(month) ?? 0,
      };
    });
  }, [data, selectedYear, winRateMap]);

  const metricTitleMap: Record<MetricType, string> = {
    change: t(ANALYTICS_MONTHLY_ANALYSIS_METRICS_CHANGE_KEY),
    profit: t(ANALYTICS_MONTHLY_ANALYSIS_METRICS_TRADING_PROFIT_KEY),
    lots: t(ANALYTICS_MONTHLY_ANALYSIS_METRICS_LOTS_KEY),
    pips: t(ANALYTICS_MONTHLY_ANALYSIS_METRICS_PIPS_KEY),
  };

  const monthLabels = useMemo(() => getMonthLabels(t), [t]);

  const series: MonthlyBarRow[] = useMemo(
    () =>
      yearData.map((item) => {
        const isActive = item.month === displayMonth;
        return {
          ...item,
          monthAxisLabel: monthLabels[item.month - 1],
          value: (item as Record<string, number>)[selectedMetric] ?? 0,
          isActive,
        };
      }),
    [yearData, selectedMetric, displayMonth, monthLabels]
  );

  const seriesRef = useRef<MonthlyBarRow[]>(series);
  seriesRef.current = series;

  const syncHoverFromTooltipIndex = useCallback((activeTooltipIndex: number | string | undefined) => {
    if (activeTooltipIndex == null || typeof activeTooltipIndex !== 'number') return;
    const row = seriesRef.current[activeTooltipIndex];
    if (!row || row.month < 1 || row.month > 12) return;
    setHoverMonth((prev) => (prev === row.month ? prev : row.month));
  }, []);

  const handleMainChartMouseMove = useCallback(
    (activeTooltipIndex: number | string | undefined) => {
      if (activeTooltipIndex == null) return;
      syncHoverFromTooltipIndex(activeTooltipIndex);
    },
    [syncHoverFromTooltipIndex]
  );

  const handleMainChartMouseLeave = useCallback(() => {
    setHoverMonth(null);
  }, []);

  const suppressChartFocus = useCallback((e: ReactMouseEvent) => {
    e.preventDefault();
  }, []);

  const commitMonthClick = useCallback((data: unknown, index: number) => {
    const m = monthFromBarClick(data, index, seriesRef.current);
    if (m != null) {
      setSelectedMonth(m);
      setHoverMonth(null);
    }
  }, []);

  const commitMonthByTooltipIndex = useCallback((activeTooltipIndex: number | string | undefined) => {
    if (activeTooltipIndex == null) return;
    const rows = seriesRef.current;
    let row: MonthlyBarRow | undefined;
    if (typeof activeTooltipIndex === 'number') {
      row = rows[activeTooltipIndex];
    } else {
      // recharts v3: activeTooltipIndex can be a string label (e.g. "Jan", "Feb")
      row = rows.find(r => r.monthAxisLabel === activeTooltipIndex);
    }
    if (!row) return;
    setSelectedMonth(row.month);
    setHoverMonth(null);
  }, []);

  const isLoading = monthlyDetailQ.isLoading;
  const detail = monthlyDetailQ.data;

  // Y-axis formatter per metric
  const yAxisFormatter = useCallback((v: number) => {
    if (selectedMetric === 'change') return `${v >= 0 ? '+' : ''}${v.toFixed(2)}`;
    if (selectedMetric === 'profit') return `${v >= 0 ? '+' : ''}${v.toFixed(2)}`;
    if (selectedMetric === 'lots') return v.toFixed(2);
    return `${v >= 0 ? '+' : ''}${v.toFixed(1)}`;
  }, [selectedMetric]);

  return (
    <div
      className="rounded-xl p-4 mb-4"
      style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', boxShadow: '0 1px 3px var(--color-shadow)' }}
    >
      {/* Header — myfxbook tabs style */}
      <div className="flex items-center justify-between gap-3 mb-3 flex-wrap">
        <div className="flex items-center gap-1">
          <span className="text-sm font-semibold px-3 py-1" style={{ color: 'var(--color-text-muted)' }}>
            {t(ANALYTICS_MONTHLY_ANALYSIS_TRADING_TITLE_KEY)}
          </span>
          <div className="flex items-center gap-0.5 rounded-md p-0.5" style={{ background: 'var(--color-bg-secondary)' }}>
            {years.map((year) => (
              <button
                key={year}
                onClick={() => { setSelectedYear(year); setSelectedMonth(1); setHoverMonth(null); }}
                className="px-3 py-1 rounded text-xs font-semibold transition-colors"
                style={{
                  background: selectedYear === year ? 'var(--color-info)' : 'transparent',
                  color: selectedYear === year ? '#FFFFFF' : 'var(--color-text-muted)',
                }}
              >
                {year}
              </button>
            ))}
          </div>
        </div>

        {/* Metric filter dropdown — myfxbook style */}
        <div className="relative">
          <div className="flex rounded-md overflow-hidden" style={{ border: '1px solid var(--color-border)' }}>
            {METRIC_OPTIONS.map((opt) => (
              <button
                key={opt.key}
                onClick={() => setSelectedMetric(opt.key)}
                className="px-2.5 py-0.5 text-xs font-medium transition-colors"
                style={{
                  background: selectedMetric === opt.key ? 'var(--color-info)' : 'transparent',
                  color: selectedMetric === opt.key ? '#FFFFFF' : 'var(--color-text-muted)',
                  borderRight: opt.key !== 'pips' ? '1px solid var(--color-border)' : 'none',
                }}
              >
                {t(opt.labelKey)}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Chart title */}
      <div className="text-center text-xs font-semibold mb-1" style={{ color: 'var(--color-text-secondary)' }}>
        {t(ANALYTICS_MONTHLY_ANALYSIS_CHART_MAIN_TRADING_TITLE_KEY, { metric: metricTitleMap[selectedMetric] })}
      </div>

      {/* myfxbook 2×2 grid: Row1=Chart+RiskReward, Row2=Popularity+HoldingSplit */}
      <div className="flex flex-col gap-3">
        {/* Row 1: Main chart (left) | Risk/Reward (right) */}
        <div className="flex flex-col lg:grid lg:grid-cols-12 gap-3">
          <div className="lg:col-span-7">
            <MonthlyAnalysisMainChart
              series={series}
              selectedMetric={selectedMetric}
              currency={currency}
              monthLabels={monthLabels}
              yAxisFormatter={yAxisFormatter}
              onMouseDown={suppressChartFocus}
              onMouseMove={handleMainChartMouseMove}
              onMouseLeave={handleMainChartMouseLeave}
              onCommitByTooltipIndex={commitMonthByTooltipIndex}
              onCommitMonthClick={commitMonthClick}
            />
          </div>
          <div className="lg:col-span-5">
            {isLoading ? (
              <div className="text-center py-8" style={{ color: 'var(--color-text-muted)' }}>
                <Spin size="small" />
              </div>
            ) : detail?.bonus ? (
              <RiskRewardPanel risks={detail.bonus.symbolRisks} t={t} />
            ) : null}
          </div>
        </div>

        {/* Row 2: Currency Popularity (left) | Holding Split (right) */}
        {!isLoading && detail?.bonus && (
          <div className="flex flex-col lg:grid lg:grid-cols-12 gap-3">
            <div className="lg:col-span-7">
              <PopularityPanel popularity={detail.bonus.symbolPopularity} t={t} />
            </div>
            <div className="lg:col-span-5">
              <HoldingSplitPanel holdingSplit={detail.bonus.symbolHoldingSplit} t={t} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
