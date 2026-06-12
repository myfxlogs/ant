import { Segmented, Switch, Tag } from 'antd';
import { Bar, CartesianGrid, ComposedChart, Cell, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { BarChartOutlined, TrophyOutlined, FallOutlined, DownOutlined, RightOutlined } from '@ant-design/icons';
import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { CHART_COLORS } from '@/constants/performance';
import { formatHoldingTime } from '@/utils/date';
import { StatusResult } from '@/components/common/StatusResult';
import { analyticsApi } from '@/client/analytics';
import type { AttributionAnalysisData, RollingMetricsData } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import MonthlyAnalysisCard from './MonthlyAnalysisCard';
import type { MonthlyAnalysisPoint } from './MonthlyAnalysisCard.shared';
import { EquityChart } from './EquityChart';
import { HourlyDailyChart } from './HourlyDailyChart';

type Props = {
  analyticsLoading: boolean;
  analyticsError?: string | null;
  onRetryAnalytics?: () => void;
  chartType: 'equity' | 'balance' | 'profit';
  setChartType: (value: 'equity' | 'balance' | 'profit') => void;
  chartPeriod: 'day' | 'week' | 'month' | 'all';
  setChartPeriod: (value: 'day' | 'week' | 'month' | 'all') => void;
  equityChartData: Record<string, unknown>[];
  profitByMonthData: Record<string, unknown>[];
  symbolDistributionData: Record<string, unknown>[];
  dailyPnLData: Record<string, unknown>[];
  hourlyData: Record<string, unknown>[];
  tradeStats: Record<string, number>;
  riskMetrics: Record<string, number>;
  monthlyAnalysisYears: number[];
  monthlyAnalysisData: Record<string, unknown>[];
  currency?: string;
  accountId?: string;
};

const StatCell = React.memo(({ label, value, color = 'var(--color-text)' }: { label: string; value: string; color?: string }) => (
  <div style={{ minWidth: 0 }}>
    <div style={{ color: 'var(--color-text-muted)', fontSize: 10, lineHeight: 1.4 }}>{label}</div>
    <div style={{ color, fontSize: 13, fontWeight: 600, lineHeight: 1.4 }}>{value}</div>
  </div>
));

/* ── shared card style ── */
const cardStyle = {
  background: 'var(--color-bg-card)',
  border: '1px solid var(--color-border)',
  borderRadius: 12,
  boxShadow: '0 1px 3px var(--color-shadow)',
} as const;

function AccountAnalyticsSection(props: Props) {
  const {
    analyticsLoading, analyticsError, onRetryAnalytics,
    chartType, setChartType, chartPeriod, setChartPeriod,
    equityChartData, symbolDistributionData, dailyPnLData, hourlyData,
    tradeStats, riskMetrics, monthlyAnalysisYears, monthlyAnalysisData,
    currency, accountId,
  } = props;

  const { t } = useTranslation();
  const [showDrawdown, setShowDrawdown] = useState(false);
  const [statsExpanded, setStatsExpanded] = useState(true);

  const stats = tradeStats as Record<string, number>;
  const risks = riskMetrics as Record<string, number>;

  const attributionQ = useQuery<AttributionAnalysisData>({
    queryKey: queryKeys.analytics.attribution(accountId!),
    queryFn: () => analyticsApi.getAttributionAnalysis(accountId!),
    enabled: !!accountId,
    staleTime: 5 * 60_000,
  });
  const rollingQ = useQuery<RollingMetricsData>({
    queryKey: queryKeys.analytics.rolling(accountId!),
    queryFn: () => analyticsApi.getRollingMetrics(accountId!),
    enabled: !!accountId,
    staleTime: 5 * 60_000,
  });
  const attr = attributionQ.data;
  const roll = rollingQ.data;

  const equityData = useMemo(() => {
    if (!showDrawdown || !roll?.drawdownCurve?.length) return equityChartData;
    return equityChartData.map((p, i) => ({
      ...p,
      drawdown: roll.drawdownCurve?.[i]?.drawdownPercent ?? 0,
    }));
  }, [equityChartData, roll?.drawdownCurve, showDrawdown]);

  // ── Direction cards — filter to only sides with trades ──
  const dirCards = useMemo(() => {
    if (!attr?.direction) return [];
    return [
      { key: 'long', label: t('accounts.report.directionLong'), color: 'var(--color-success)',
        profit: attr.direction.longProfit ?? 0, trades: attr.direction.longTrades ?? 0, winRate: attr.direction.longWinRate ?? 0 },
      { key: 'short', label: t('accounts.report.directionShort'), color: 'var(--color-danger)',
        profit: attr.direction.shortProfit ?? 0, trades: attr.direction.shortTrades ?? 0, winRate: attr.direction.shortWinRate ?? 0 },
    ];
  }, [attr?.direction, t]);

  return (
    <StatusResult loading={analyticsLoading} error={analyticsError} onRetry={onRetryAnalytics}>

      {/* ═══ 1. Equity Curve + Drawdown ═══ */}
      <div className="p-4 mb-4" style={cardStyle}>
        <div className="flex items-center justify-between mb-3 flex-wrap gap-2">
          <div className="flex gap-1 p-1 rounded-lg" style={{ background: 'var(--color-bg-secondary)' }}>
            {(['equity', 'balance', 'profit'] as const).map((key) => (
              <button key={key} onClick={() => setChartType(key)}
                className="px-3 py-1 rounded-md text-sm font-medium transition-all"
                style={{
                  background: chartType === key ? 'var(--color-bg-card)' : 'transparent',
                  color: chartType === key ? 'var(--color-text)' : 'var(--color-text-muted)',
                  boxShadow: chartType === key ? '0 1px 3px var(--color-shadow)' : 'none',
                }}>
                {t(`accounts.analytics.chartType.${key}`)}
              </button>
            ))}
          </div>
          <div className="flex items-center gap-3">
            <Segmented value={chartPeriod} onChange={(v) => setChartPeriod(v as typeof chartPeriod)}
              options={[
                { label: t('accounts.analytics.chartPeriod.day'), value: 'day' },
                { label: t('accounts.analytics.chartPeriod.week'), value: 'week' },
                { label: t('accounts.analytics.chartPeriod.month'), value: 'month' },
                { label: t('accounts.analytics.chartPeriod.all'), value: 'all' },
              ]} size="small" />
            {roll?.drawdownCurve?.length ? (
              <span style={{ fontSize: 12, color: 'var(--color-text-muted)', whiteSpace: 'nowrap' }}>
                <Switch size="small" checked={showDrawdown} onChange={setShowDrawdown} />
                <span style={{ marginLeft: 4 }}>DD%</span>
              </span>
            ) : null}
          </div>
        </div>
        <EquityChart chartType={chartType} chartPeriod={chartPeriod} data={equityData} />
        {showDrawdown && roll?.drawdownEvents?.length ? (
          <div className="mt-3 pt-3" style={{ borderTop: '1px solid var(--color-border)' }}>
            <div className="text-xs font-semibold mb-2" style={{ color: 'var(--color-text-secondary)' }}>
              <FallOutlined className="mr-1" />{t('accounts.report.drawdownEvents')}
            </div>
            <div className="flex flex-wrap gap-2">
              {roll.drawdownEvents.slice(0, 6).map((e, i) => (
                <Tag key={i} color="error" style={{ margin: 0, fontSize: 11 }}>
                  {e.startDate} → {e.endDate || '...'} ({e.depthPercent.toFixed(1)}%)
                </Tag>
              ))}
            </div>
          </div>
        ) : null}
      </div>

      {/* ═══ 2. Monthly Analysis ═══ */}
      <MonthlyAnalysisCard
        accountId={accountId}
        years={monthlyAnalysisYears}
        data={monthlyAnalysisData as unknown as MonthlyAnalysisPoint[]}
        winRateData={roll?.monthlyWinRates}
        currency={currency}
      />

      {/* ═══ 3. Hourly / Daily ═══ */}
      <HourlyDailyChart hourlyData={hourlyData} dailyPnLData={dailyPnLData} currency={currency || 'USD'} />

      {/* ═══ 4. Trading Stats + Symbol P&L ═══ */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">

        {/* Stats — collapsible */}
        <div className="p-4" style={cardStyle}>
          <button
            onClick={() => setStatsExpanded((v) => !v)}
            className="w-full flex items-center justify-between mb-3 cursor-pointer"
            style={{ background: 'none', border: 'none', padding: 0, color: 'var(--color-text)' }}
          >
            <h2 className="text-base font-semibold flex items-center gap-2 m-0" style={{ color: 'var(--color-text)' }}>
              <TrophyOutlined />{t('accounts.analytics.advancedStatsTitle')}
            </h2>
            {statsExpanded
              ? <DownOutlined style={{ fontSize: 12, color: 'var(--color-text-muted)' }} />
              : <RightOutlined style={{ fontSize: 12, color: 'var(--color-text-muted)' }} />
            }
          </button>
          {statsExpanded && (
            <div className="grid grid-cols-3 gap-x-3 gap-y-2">
              <StatCell label={t('accounts.analytics.stats.winRate')} value={`${(stats.winRate || 0).toFixed(1)}%`} color="var(--color-success)" />
              <StatCell label={t('accounts.analytics.stats.profitFactor')} value={`${(stats.profitFactor || 0).toFixed(2)}`} color="var(--color-primary)" />
              <StatCell label={t('accounts.analytics.stats.maxDrawdown')} value={`${(risks.maxDrawdownPercent || 0).toFixed(2)}%`} color="var(--color-danger)" />
              <StatCell label={t('accounts.analytics.stats.totalTrades')} value={`${stats.totalTrades || 0}`} />
              <StatCell label={t('accounts.analytics.stats.avgProfit')} value={`+${(stats.averageProfit || 0).toFixed(2)}`} color="var(--color-success)" />
              <StatCell label={t('accounts.analytics.stats.avgLoss')} value={`${(stats.averageLoss || 0).toFixed(2)}`} color="var(--color-danger)" />
              <StatCell label={t('accounts.analytics.stats.avgHolding')} value={formatHoldingTime(stats.averageHoldingTime) || '-'} />
              <StatCell label={t('accounts.analytics.stats.consecutiveWinsLosses')} value={`${stats.maxConsecutiveWins || 0}/${stats.maxConsecutiveLosses || 0}`} />
              <StatCell label={t('accounts.analytics.stats.sharpe')} value={`${(risks.sharpeRatio || 0).toFixed(2)}`} color="var(--color-success)" />
              <StatCell label={t('accounts.analytics.stats.sortino')} value={`${(risks.sortinoRatio || 0).toFixed(2)}`} />
              <StatCell label={t('accounts.analytics.stats.calmar')} value={`${(risks.calmarRatio || 0).toFixed(2)}`} />
              <StatCell label={t('accounts.analytics.stats.largestWin')} value={`+${(stats.largestWin || 0).toFixed(2)}`} color="var(--color-success)" />
              <StatCell label={t('accounts.analytics.stats.largestLoss')} value={`${(stats.largestLoss || 0).toFixed(2)}`} color="var(--color-danger)" />
              <StatCell label={t('accounts.analytics.stats.avgDailyReturn')} value={`${(risks.averageDailyReturn || 0).toFixed(2)}`} />
              <StatCell label={t('accounts.analytics.stats.volatility')} value={`${(risks.volatility || 0).toFixed(2)}`} color="var(--color-info)" />
              <StatCell label={t('accounts.analytics.stats.netProfit')} value={`${(stats.netProfit || 0).toFixed(2)}`} color={(stats.netProfit || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
              <StatCell label={t('accounts.analytics.stats.totalDeposit')} value={`+${(stats.totalDeposit || 0).toFixed(2)}`} />
              <StatCell label={t('accounts.analytics.stats.totalWithdrawal')} value={`-${(stats.totalWithdrawal || 0).toFixed(2)}`} />
              <StatCell label={t('accounts.analytics.stats.netDeposit')} value={`${(stats.netDeposit || 0).toFixed(2)}`} color={(stats.netDeposit || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
            </div>
          )}
        </div>

        {/* Symbol P&L — pie chart + ranking + direction */}
        <div className="p-4" style={cardStyle}>
          <h2 className="text-base font-semibold flex items-center gap-2 mb-3" style={{ color: 'var(--color-text)' }}>
            <BarChartOutlined />{t('accounts.report.symbolPnL')}
          </h2>
          <StatusResult loading={attributionQ.isLoading} error={attributionQ.error?.message}>
            {attr?.symbolPnls?.length ? (
              <div className="space-y-3">
                {/* Pie chart */}
                {symbolDistributionData.length > 0 && (
                  <div className="flex items-center gap-3">
                    <ResponsiveContainer width={90} height={90}>
                      <PieChart>
                        <Pie data={symbolDistributionData} cx={45} cy={45} innerRadius={25} outerRadius={38} paddingAngle={2} dataKey="value" isAnimationActive={false}>
                          {symbolDistributionData.map((_: unknown, i: number) => (
                            <Cell key={`c-${i}`} fill={CHART_COLORS[i % CHART_COLORS.length]} />
                          ))}
                        </Pie>
                      </PieChart>
                    </ResponsiveContainer>
                    <div className="flex-1 text-xs" style={{ color: 'var(--color-text-secondary)' }}>
                      {symbolDistributionData.slice(0, 6).map((item: Record<string, unknown>, i: number) => (
                        <div key={String(item.name)} className="flex items-center justify-between mb-0.5">
                          <div className="flex items-center gap-1.5">
                            <div className="w-2 h-2 rounded-full flex-shrink-0" style={{ background: CHART_COLORS[i % CHART_COLORS.length] }} />
                            <span>{String(item.name)}</span>
                          </div>
                          <span style={{ color: 'var(--color-text-muted)' }}>{String(item.value)}%</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                {/* P&L ranking bar chart */}
                {(() => {
                  const data = attr.symbolPnls.slice(0, 6);
                  const maxSymbolLen = Math.max(...data.map((s: { symbol: string }) => s.symbol.length), 3);
                  const yWidth = Math.min(Math.round(maxSymbolLen * 6.8 + 10), 90);
                  const yFontSize = maxSymbolLen > 12 ? 8 : maxSymbolLen > 8 ? 9 : 10;
                  return (
                <ResponsiveContainer width="100%" height={Math.max(data.length * 28, 100)}>
                  <ComposedChart layout="vertical" data={data} barCategoryGap="6%">
                    <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" horizontal={false} />
                    <XAxis type="number" stroke="var(--color-text-muted)" fontSize={10} />
                    <YAxis type="category" dataKey="symbol" width={yWidth} stroke="var(--color-text-muted)" fontSize={yFontSize} />
                    <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: 8, boxShadow: '0 4px 12px var(--color-shadow)' }} />
                    <Bar dataKey="netProfit" radius={[0, 3, 3, 0]} maxBarSize={22} isAnimationActive={false} cursor="pointer">
                      {data.map((_: unknown, i: number) => (
                        <Cell key={i} fill={(data[i]?.netProfit ?? 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
                      ))}
                    </Bar>
                  </ComposedChart>
                </ResponsiveContainer>
                  );
                })()}

                {/* Direction — only sides with trades */}
                {dirCards.length > 0 && (
                  <div className="grid grid-cols-2 gap-2">
                    {dirCards.map(({ key, label, color, profit, trades, winRate }) => (
                      <div key={key} className="rounded-lg p-2"
                        style={{ border: '1px solid var(--color-border)', background: 'var(--color-bg-secondary)' }}>
                        <div className="text-xs font-semibold" style={{ color }}>{label}</div>
                        <div className="text-xs" style={{ color: 'var(--color-text-secondary)', lineHeight: 1.6 }}>
                          <div>P&L: <strong style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>{profit >= 0 ? '+' : ''}{profit.toFixed(2)}</strong></div>
                          <div>{t('accounts.analytics.stats.totalTrades')}: {trades}</div>
                          <div>{t('accounts.analytics.stats.winRate')}: {winRate.toFixed(1)}%</div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <div className="flex items-center justify-center h-[200px]" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>
                {t('accounts.analytics.empty.monthlyProfit')}
              </div>
            )}
          </StatusResult>
        </div>
      </div>

    </StatusResult>
  );
}

export default React.memo(AccountAnalyticsSection);
