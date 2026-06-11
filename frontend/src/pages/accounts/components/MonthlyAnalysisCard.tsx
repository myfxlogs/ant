import type { MouseEvent as ReactMouseEvent } from 'react';
import { useCallback, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { Spin } from 'antd';

import MonthlyAnalysisMainChart from './MonthlyAnalysisMainChart';
import MonthlyDrillDown from './MonthlyDrillDown';
import { formatMonthLongName } from '@/utils/date';
import {
  type MetricType,
  type MonthlyAnalysisCardProps,
  type MonthlyAnalysisPoint,
  type MonthlyBarRow,
  monthFromBarClick,
  monthShortLabels,
} from './MonthlyAnalysisCard.shared';
import { analyticsApi } from '@/client/analytics';
import type { MonthlyDetailData } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';

const ALL_METRICS: MetricType[] = ['change', 'profit', 'lots', 'pips', 'winRate'];

export default function MonthlyAnalysisCard({ accountId, years, data, winRateData, currency = 'USD' }: MonthlyAnalysisCardProps) {
  const { t } = useTranslation();
  const [selectedYear, setSelectedYear] = useState<number>(years[years.length - 1] || new Date().getFullYear());
  const [selectedMonth, setSelectedMonth] = useState<number>(new Date().getMonth() + 1);
  const [hoverMonth, setHoverMonth] = useState<number | null>(null);
  const [selectedMetric, setSelectedMetric] = useState<MetricType>('change');
  const displayMonth = hoverMonth ?? selectedMonth;

  // Fetch drill-down detail for the selected month.
  const monthlyDetailQ = useQuery<MonthlyDetailData>({
    queryKey: queryKeys.analytics.monthlyDetail(accountId!, selectedYear, selectedMonth),
    queryFn: () => analyticsApi.getMonthlyDetail(accountId!, selectedYear, selectedMonth),
    enabled: !!accountId,
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

  const focused = useMemo(
    () => yearData.find((item) => item.month === displayMonth) || yearData[0],
    [yearData, displayMonth]
  );

  const metricTitleMap: Record<MetricType, string> = {
    change: t('accounts.analytics.monthlyAnalysis.metrics.change'),
    profit: t('accounts.analytics.monthlyAnalysis.metrics.profit'),
    lots: t('accounts.analytics.monthlyAnalysis.metrics.lots'),
    pips: t('accounts.analytics.monthlyAnalysis.metrics.pips'),
    winRate: t('accounts.analytics.stats.winRate'),
  };

  const formatValue = (metric: MetricType, value: number) => {
    if (metric === 'change') return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
    if (metric === 'profit') return `${value >= 0 ? '+' : ''}${value.toFixed(2)} ${currency}`;
    if (metric === 'lots') return `${value.toFixed(2)} lots`;
    if (metric === 'winRate') return `${value.toFixed(1)}%`;
    return `${value >= 0 ? '+' : ''}${value.toFixed(1)} pips`;
  };

  const renderMetricValue = (metric: MetricType, value: number) => {
    const color = metric === 'winRate'
      ? 'var(--color-text)'
      : value > 0 ? 'var(--color-success)' : value < 0 ? 'var(--color-danger)' : 'var(--color-text-muted)';
    return <span style={{ color, fontWeight: 600 }}>{formatValue(metric, value)}</span>;
  };

  const series: MonthlyBarRow[] = useMemo(
    () =>
      yearData.map((item) => {
        const isActive = item.month === displayMonth;
        return {
          ...item,
          monthAxisLabel: `${monthShortLabels[item.month - 1]} ${selectedYear}`,
          value: (item as Record<string, number>)[selectedMetric] ?? 0,
          isActive,
        };
      }),
    [yearData, selectedMetric, selectedYear, displayMonth]
  );

  const seriesRef = useRef<MonthlyBarRow[]>(series);
  seriesRef.current = series;

  const syncHoverFromTooltipIndex = useCallback((activeTooltipIndex: number | string | undefined) => {
    if (activeTooltipIndex == null || typeof activeTooltipIndex !== 'number') return;
    const row = seriesRef.current[activeTooltipIndex];
    if (!row || row.month < 1 || row.month > 12) return;
    setHoverMonth((prev) => (prev === row.month ? prev : row.month));
  }, []);

  type RechartsMouseState = {
    isTooltipActive?: boolean;
    activeTooltipIndex?: number | string;
  };

  const handleMainChartMouseMove = useCallback(
    (state: RechartsMouseState) => {
      if (!state.isTooltipActive) return;
      syncHoverFromTooltipIndex(state.activeTooltipIndex);
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
    if (typeof activeTooltipIndex !== 'number') return;
    const row = seriesRef.current[activeTooltipIndex];
    if (!row) return;
    setSelectedMonth(row.month);
    setHoverMonth(null);
  }, []);

  const monthLong = formatMonthLongName(displayMonth);
  const selectedPeriodLabel = `${monthLong} ${selectedYear}`;

  const chartTitleMain = t('accounts.analytics.monthlyAnalysis.chartMainTitle', {
    metric: metricTitleMap[selectedMetric],
  });

  return (
    <div
      className="rounded-xl p-4 mb-4"
      style={{ background: 'var(--color-bg-card)', border: '1px solid var(--color-border)', boxShadow: '0 1px 3px var(--color-shadow)' }}
    >
      <div className="flex items-center justify-between gap-3 mb-3 flex-wrap">
        <h2 className="text-base font-semibold" style={{ color: 'var(--color-text)' }}>
          {t('accounts.analytics.monthlyAnalysis.title')}
        </h2>
        <div className="flex items-center gap-1 rounded-md p-1" style={{ background: 'var(--color-bg-secondary)' }}>
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

      <div className="flex gap-3 mb-2 border-b" style={{ borderColor: 'var(--color-border)' }}>
        {ALL_METRICS.map((metric) => (
          <button
            key={metric}
            onClick={() => setSelectedMetric(metric)}
            className="pb-2 text-sm font-medium transition-colors"
            style={{
              color: selectedMetric === metric ? 'var(--color-info)' : 'var(--color-text-muted)',
              borderBottom: selectedMetric === metric ? '2px solid var(--color-info)' : '2px solid transparent',
              marginBottom: '-1px',
            }}
          >
            {metricTitleMap[metric]}
          </button>
        ))}
      </div>

      <div
        className="mb-2 px-2 py-1.5 rounded-md flex flex-wrap items-center gap-x-3 gap-y-1"
        style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)', fontSize: 11 }}
      >
        <span style={{ color: 'var(--color-text-secondary)', fontWeight: 600 }}>
          {t('accounts.analytics.monthlyAnalysis.focusedValue', {
            period: selectedPeriodLabel,
            metric: metricTitleMap[selectedMetric],
            value: formatValue(selectedMetric, focused?.[selectedMetric as keyof typeof focused] as number || 0),
          })}
        </span>
        <span className="hidden sm:inline" style={{ color: 'var(--color-text-muted)' }}>|</span>
        <span style={{ color: 'var(--color-text-secondary)' }} className="flex flex-wrap gap-x-3 gap-y-0.5">
          {ALL_METRICS.map((m) => (
            <span key={m}>{metricTitleMap[m]}: {renderMetricValue(m, (focused as Record<string, number>)[m] || 0)}</span>
          ))}
        </span>
      </div>

      <div className="relative">
        <div className="text-center text-xs font-semibold mb-1" style={{ color: 'var(--color-text-secondary)' }}>
          {chartTitleMain}
        </div>
        <MonthlyAnalysisMainChart
          series={series}
          selectedMetric={selectedMetric}
          metricTitleMap={metricTitleMap}
          formatValue={formatValue}
          renderMetricValue={renderMetricValue}
          onMouseDown={suppressChartFocus}
          onMouseMove={handleMainChartMouseMove}
          onMouseLeave={handleMainChartMouseLeave}
          onCommitByTooltipIndex={commitMonthByTooltipIndex}
          onCommitMonthClick={commitMonthClick}
        />
      </div>

      {/* Drill-down sub-panels for the selected month */}
      {monthlyDetailQ.isLoading ? (
        <div className="mt-4 text-center py-4" style={{ color: 'var(--color-text-muted)' }}>
          <Spin size="small" />
        </div>
      ) : monthlyDetailQ.data ? (
        <MonthlyDrillDown detail={monthlyDetailQ.data} currency={currency} />
      ) : null}
    </div>
  );
}
