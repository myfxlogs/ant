import type { MouseEvent as ReactMouseEvent } from 'react';
import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Spin } from 'antd';
import {
  ResponsiveContainer,
  CartesianGrid,
  XAxis,
  YAxis,
  Tooltip,
  Bar,
  Cell,
  BarChart,
  Legend,
  PieChart,
  Pie,
} from 'recharts';
import { useTranslation } from 'react-i18next';

import { analyticsApi } from '@/client/analytics';
import MonthlyAnalysisMainChart from './MonthlyAnalysisMainChart';
import { formatMonthLongName } from '@/utils/date';
import {
  type BonusPayload,
  type MetricType,
  type MonthlyAnalysisCardProps,
  type MonthlyAnalysisPoint,
  type MonthlyBarRow,
  PIE_COLORS,
  formatSecondsAxis,
  monthFromBarClick,
  monthShortLabels,
  riskBarVisual,
} from './MonthlyAnalysisCard.shared';

export default function MonthlyAnalysisCard({ accountId, years, data, currency = 'USD' }: MonthlyAnalysisCardProps) {
  const { t } = useTranslation();
  const [selectedYear, setSelectedYear] = useState<number>(years[years.length - 1] || new Date().getFullYear());
  /** Committed selection: drives bonus API and persists after pointer leaves the chart. */
  const [selectedMonth, setSelectedMonth] = useState<number>(new Date().getMonth() + 1);
  /** Hover preview: follows Recharts tooltip index so summary/highlight match the tooltip without extra API calls. */
  const [hoverMonth, setHoverMonth] = useState<number | null>(null);
  const [selectedMetric, setSelectedMetric] = useState<MetricType>('change');
  const [bonusLoading, setBonusLoading] = useState(false);
  const [bonus, setBonus] = useState<BonusPayload | null>(null);

  const displayMonth = hoverMonth ?? selectedMonth;

  useLayoutEffect(() => {
    if (!accountId) {
      setBonusLoading(false);
      return;
    }
    setBonusLoading(true);
  }, [accountId, selectedYear, selectedMonth]);

  useEffect(() => {
    if (!accountId) {
      setBonus(null);
      return;
    }
    let cancelled = false;
    (async () => {
      try {
        const b = await analyticsApi.getMonthlyAnalysisBonus(accountId, selectedYear, selectedMonth);
        if (!cancelled) setBonus(b);
      } catch {
        if (!cancelled) setBonus(null);
      } finally {
        if (!cancelled) setBonusLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [accountId, selectedYear, selectedMonth]);

  const yearData = useMemo(() => {
    const monthMap = new Map<number, MonthlyAnalysisPoint>();
    data
      .filter((item) => item.year === selectedYear)
      .forEach((item) => monthMap.set(item.month, item));
    return Array.from({ length: 12 }, (_, index) => {
      const month = index + 1;
      return monthMap.get(month) || { year: selectedYear, month, change: 0, profit: 0, lots: 0, pips: 0, trades: 0 };
    });
  }, [data, selectedYear]);

  const focused = useMemo(
    () => yearData.find((item) => item.month === displayMonth) || yearData[0],
    [yearData, displayMonth]
  );

  const metricTitleMap: Record<MetricType, string> = {
    change: t('accounts.analytics.monthlyAnalysis.metrics.change'),
    profit: t('accounts.analytics.monthlyAnalysis.metrics.profit'),
    lots: t('accounts.analytics.monthlyAnalysis.metrics.lots'),
    pips: t('accounts.analytics.monthlyAnalysis.metrics.pips'),
  };

  const formatValue = (metric: MetricType, value: number) => {
    if (metric === 'change') return `${value >= 0 ? '+' : ''}${value.toFixed(2)}%`;
    if (metric === 'profit') return `${value >= 0 ? '+' : ''}${value.toFixed(2)} ${currency}`;
    if (metric === 'lots') return `${value.toFixed(2)} lots`;
    return `${value >= 0 ? '+' : ''}${value.toFixed(1)} pips`;
  };

  const renderMetricValue = (metric: MetricType, value: number) => (
    <span style={{ color: value > 0 ? '#2E7D32' : value < 0 ? '#C62828' : '#607D8B', fontWeight: 600 }}>
      {formatValue(metric, value)}
    </span>
  );

  const series: MonthlyBarRow[] = useMemo(
    () =>
      yearData.map((item) => {
        const isActive = item.month === displayMonth;
        return {
          ...item,
          monthAxisLabel: `${monthShortLabels[item.month - 1]} ${selectedYear}`,
          value: item[selectedMetric],
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
  const monthLongBonus = formatMonthLongName(selectedMonth);
  const selectedPeriodLabel = `${monthLong} ${selectedYear}`;

  const pieData = useMemo(() => {
    if (!bonus?.symbolPopularity?.length) return [];
    return bonus.symbolPopularity.map((row, i) => ({
      name: row.symbol === 'Other' ? t('accounts.analytics.monthlyAnalysis.bonus.sliceOther') : row.symbol,
      rawSymbol: row.symbol,
      value: row.sharePercent,
      trades: row.trades,
      fill: PIE_COLORS[i % PIE_COLORS.length],
    }));
  }, [bonus, t]);

  const riskChartData = useMemo(() => {
    if (!bonus?.symbolRisks?.length) return [];
    return [...bonus.symbolRisks]
      .sort((a, b) => a.symbol.localeCompare(b.symbol))
      .map((r) => ({
        symbol: r.symbol,
        risk: riskBarVisual(r.riskRatio),
        riskRaw: r.riskRatio,
      }));
  }, [bonus]);

  const holdingChartData = useMemo(() => {
    if (!bonus?.symbolHoldingSplit?.length) return [];
    return [...bonus.symbolHoldingSplit].sort((a, b) => a.symbol.localeCompare(b.symbol));
  }, [bonus]);

  const chartTitleMain = t('accounts.analytics.monthlyAnalysis.chartMainTitle', {
    metric: metricTitleMap[selectedMetric],
  });
  const chartTitleRisk = t('accounts.analytics.monthlyAnalysis.bonus.chartRiskTitle', { month: monthLongBonus });
  const chartTitlePopular = t('accounts.analytics.monthlyAnalysis.bonus.chartPopularTitle', { month: monthLongBonus });
  const chartTitleHolding = t('accounts.analytics.monthlyAnalysis.bonus.chartHoldingTitle', { month: monthLongBonus });

  return (
    <div
      className="rounded-xl p-4 mb-6"
      style={{ background: '#FFFFFF', border: '1px solid #D9E2EC', boxShadow: '0 1px 2px rgba(15, 23, 42, 0.04)' }}
    >
      <div className="flex items-center justify-between gap-3 mb-3 flex-wrap">
        <h2 className="text-base font-semibold" style={{ color: '#1F2937' }}>
          {t('accounts.analytics.monthlyAnalysis.title')}
        </h2>
        <div className="flex items-center gap-1 rounded-md p-1" style={{ background: '#F4F7FA' }}>
          {years.map((year) => (
            <button
              key={year}
              onClick={() => {
                setSelectedYear(year);
                setSelectedMonth(1);
                setHoverMonth(null);
              }}
              className="px-3 py-1 rounded text-xs font-semibold transition-colors"
              style={{
                background: selectedYear === year ? '#2B6CB0' : 'transparent',
                color: selectedYear === year ? '#FFFFFF' : '#64748B',
              }}
            >
              {year}
            </button>
          ))}
        </div>
      </div>

      <div className="flex gap-3 mb-2 border-b" style={{ borderColor: '#E5EAF0' }}>
        {(['change', 'profit', 'lots', 'pips'] as MetricType[]).map((metric) => (
          <button
            key={metric}
            onClick={() => setSelectedMetric(metric)}
            className="pb-2 text-sm font-medium transition-colors"
            style={{
              color: selectedMetric === metric ? '#2B6CB0' : '#667085',
              borderBottom: selectedMetric === metric ? '2px solid #2B6CB0' : '2px solid transparent',
              marginBottom: '-1px',
            }}
          >
            {metricTitleMap[metric]}
          </button>
        ))}
      </div>

      <div
        className="mb-2 px-2 py-1.5 rounded-md flex flex-wrap items-center gap-x-3 gap-y-1"
        style={{ background: '#F8FAFC', border: '1px solid #E6EDF5', fontSize: '11px' }}
      >
        <span style={{ color: '#475467', fontWeight: 600 }}>
          {t('accounts.analytics.monthlyAnalysis.focusedValue', {
            period: selectedPeriodLabel,
            metric: metricTitleMap[selectedMetric],
            value: formatValue(selectedMetric, focused?.[selectedMetric] || 0),
          })}
        </span>
        <span className="text-slate-400 hidden sm:inline">|</span>
        <span style={{ color: '#64748B' }} className="flex flex-wrap gap-x-3 gap-y-0.5">
          <span>{metricTitleMap.change}: {renderMetricValue('change', focused?.change || 0)}</span>
          <span>{metricTitleMap.profit}: {renderMetricValue('profit', focused?.profit || 0)}</span>
          <span>{metricTitleMap.lots}: {renderMetricValue('lots', focused?.lots || 0)}</span>
          <span>{metricTitleMap.pips}: {renderMetricValue('pips', focused?.pips || 0)}</span>
        </span>
      </div>

      {accountId ? (
        <div className="relative">
          <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
            <div className="min-w-0">
              <div className="text-center text-xs font-semibold mb-1" style={{ color: '#475467' }}>
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
              />            </div>

            <div className="min-w-0">
              <div className="text-center text-xs font-semibold mb-1" style={{ color: '#475467', fontStyle: 'italic' }}>
                {chartTitleRisk}
              </div>
              <ResponsiveContainer width="100%" height={240}>
                {riskChartData.length > 0 ? (
                  <BarChart layout="vertical" data={riskChartData} margin={{ left: 8, right: 12, top: 4, bottom: 4 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E6EDF5" horizontal />
                    <XAxis type="number" stroke="#94A3B8" fontSize={10} tickLine={false} axisLine={{ stroke: '#E6EDF5' }} />
                    <YAxis type="category" dataKey="symbol" width={76} stroke="#94A3B8" fontSize={10} tickLine={false} axisLine={{ stroke: '#E6EDF5' }} />
                    <Tooltip
                      cursor={false}
                      content={({ active, payload }) => {
                        if (!active || !payload?.length) return null;
                        const row = payload[0]?.payload as { symbol?: string; risk?: number; riskRaw?: number };
                        if (!row?.symbol) return null;
                        const raw = row.riskRaw ?? 0;
                        const text = raw >= 999.98 ? '>99.99' : raw.toFixed(2);
                        return (
                          <div
                            className="rounded border bg-white px-2 py-1 text-xs shadow"
                            style={{ borderColor: '#D9E2EC' }}
                          >
                            <span className="font-semibold text-slate-700">{row.symbol}</span>
                            <span className="text-slate-600">
                              {' '}
                              · {t('accounts.analytics.stats.profitFactor')}: {text}
                            </span>
                          </div>
                        );
                      }}
                    />
                    <Bar dataKey="risk" fill="#7BA7D9" radius={[0, 2, 2, 0]} barSize={12} isAnimationActive={false} />
                  </BarChart>
                ) : (
                  <div className="flex h-full items-center justify-center text-xs text-slate-400">
                    {t('accounts.analytics.monthlyAnalysis.bonus.emptyCharts')}
                  </div>
                )}
              </ResponsiveContainer>
            </div>

            <div className="min-w-0">
              <div className="text-center text-xs font-semibold mb-1" style={{ color: '#475467', fontStyle: 'italic' }}>
                {chartTitlePopular}
              </div>
              <ResponsiveContainer width="100%" height={240}>
                {pieData.length > 0 ? (
                  <PieChart>
                    <Tooltip
                      cursor={false}
                      formatter={(value: number, _name, item) => {
                        const p = item?.payload as { trades?: number };
                        return [
                          `${Number(value).toFixed(1)}% (${p?.trades ?? 0} ${t('accounts.analytics.chartSeries.tradeCount')})`,
                          t('accounts.analytics.monthlyAnalysis.bonus.popularityShare'),
                        ];
                      }}
                    />
                    <Pie
                      data={pieData}
                      dataKey="value"
                      nameKey="name"
                      cx="50%"
                      cy="50%"
                      innerRadius={0}
                      outerRadius={78}
                      paddingAngle={1}
                      label={({ name, percent }) => `${name} ${((percent ?? 0) * 100).toFixed(1)}%`}
                      isAnimationActive={false}
                    >
                      {pieData.map((entry, index) => (
                        <Cell key={`cell-${entry.rawSymbol}-${index}`} fill={entry.fill} stroke="#fff" strokeWidth={1} />
                      ))}
                    </Pie>
                  </PieChart>
                ) : (
                  <div className="flex h-full items-center justify-center text-xs text-slate-400">
                    {t('accounts.analytics.monthlyAnalysis.bonus.emptyCharts')}
                  </div>
                )}
              </ResponsiveContainer>
            </div>

            <div className="min-w-0">
              <div className="text-center text-xs font-semibold mb-1" style={{ color: '#475467', fontStyle: 'italic' }}>
                {chartTitleHolding}
              </div>
              <ResponsiveContainer width="100%" height={240}>
                {holdingChartData.length > 0 ? (
                  <BarChart layout="vertical" data={holdingChartData} margin={{ left: 8, right: 12, top: 4, bottom: 20 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#E6EDF5" horizontal />
                    <XAxis type="number" stroke="#94A3B8" fontSize={10} tickFormatter={formatSecondsAxis} tickLine={false} axisLine={{ stroke: '#E6EDF5' }} />
                    <YAxis type="category" dataKey="symbol" width={76} stroke="#94A3B8" fontSize={10} tickLine={false} axisLine={{ stroke: '#E6EDF5' }} />
                    <Tooltip
                      cursor={false}
                      formatter={(value: number) => [formatSecondsAxis(value), '']}
                    />
                    <Legend
                      wrapperStyle={{ fontSize: 11 }}
                      formatter={(value) =>
                        value === 'bullsSeconds'
                          ? t('accounts.analytics.monthlyAnalysis.bonus.legendBulls')
                          : t('accounts.analytics.monthlyAnalysis.bonus.legendShortTerm')
                      }
                    />
                    <Bar dataKey="bullsSeconds" stackId="hold" fill="#00A651" name="bullsSeconds" barSize={12} isAnimationActive={false} />
                    <Bar dataKey="shortTermSeconds" stackId="hold" fill="#E53935" name="shortTermSeconds" barSize={12} isAnimationActive={false} />
                  </BarChart>
                ) : (
                  <div className="flex h-full items-center justify-center text-xs text-slate-400">
                    {t('accounts.analytics.monthlyAnalysis.bonus.emptyCharts')}
                  </div>
                )}
              </ResponsiveContainer>
            </div>
          </div>

          {bonusLoading ? (
            <div className="absolute top-2 right-2 z-10 flex items-center gap-2 rounded bg-white/90 px-2 py-1 shadow border border-slate-200/80 pointer-events-none">
              <Spin size="small" />
            </div>
          ) : null}
        </div>
      ) : (
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
              />      )}
    </div>
  );
}
