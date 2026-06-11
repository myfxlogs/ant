import { Segmented, Tag, Collapse } from 'antd';
import { Bar, CartesianGrid, ComposedChart, Cell, Legend, Line, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { BarChartOutlined, PieChartOutlined, TrophyOutlined, RiseOutlined, FallOutlined } from '@ant-design/icons';
import { CHART_COLORS } from '@/constants/performance';
import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { formatHoldingTime } from '@/utils/date';
import { StatusResult } from '@/components/common/StatusResult';
import { analyticsApi } from '@/client/analytics';
import type { AttributionAnalysisData, RollingMetricsData } from '@/client/analytics';
import { getErrorMessage } from '@/utils/error';
import { queryKeys } from '@/queries/queryKeys';
import { StatCard } from './AccountDetail.shared';
import MonthlyAnalysisCard from './MonthlyAnalysisCard';
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

function AccountAnalyticsSection(props: Props) {
  const {
    analyticsLoading, analyticsError, onRetryAnalytics,
    chartType, setChartType, chartPeriod, setChartPeriod,
    equityChartData, symbolDistributionData, dailyPnLData, hourlyData,
    tradeStats, riskMetrics, monthlyAnalysisYears, monthlyAnalysisData,
    currency, accountId,
  } = props;

  const { t } = useTranslation();
  const currentYear = new Date().getFullYear();
  const [selectedYear, setSelectedYear] = useState(currentYear);
  const [monthlyData, setMonthlyData] = useState<Record<string, unknown>[] | null>(
    () => props.profitByMonthData.length > 0 ? props.profitByMonthData : null,
  );
  const [monthlyError, setMonthlyError] = useState<string | null>(null);

  useEffect(() => {
    if (!accountId) return;
    let cancelled = false;
    setMonthlyError(null);
    analyticsApi.getMonthlyPnL(accountId, selectedYear)
      .then((data) => { if (!cancelled) { setMonthlyData(data?.monthlyPnl || []); setMonthlyError(null); } })
      .catch((err) => { if (!cancelled) { setMonthlyError(getErrorMessage(err, 'Failed to load monthly PnL')); setMonthlyData([]); } });
    return () => { cancelled = true; };
  }, [accountId, selectedYear]);

  const profitByMonthData = (monthlyData || props.profitByMonthData || [])
    .map((m: Record<string, unknown>) => ({
      month: String(m?.month ?? m?.monthNum ?? m?.month_num ?? ''),
      profit: m.profit, trades: Number(m.trades),
    }))
    .filter((m: Record<string, unknown>) => m.month);

  const stats = tradeStats as Record<string, number>;
  const risks = riskMetrics as Record<string, number>;

  // ── Attribution + Rolling (on-demand data) ──
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

  return (
    <StatusResult loading={analyticsLoading} error={analyticsError} onRetry={onRetryAnalytics}>
      {/* Equity + Monthly Profit */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <div className="flex items-center justify-between mb-4">
            <div className="flex gap-1 p-1 rounded-lg" style={{ background: 'var(--color-bg-secondary)' }}>
              {(['equity', 'balance', 'profit'] as const).map((key) => (
                <button key={key} onClick={() => setChartType(key)}
                  className="px-4 py-1.5 rounded-md text-sm font-medium transition-all"
                  style={{ background: chartType === key ? 'var(--color-bg-card)' : 'transparent', color: chartType === key ? 'var(--color-text)' : 'var(--color-text-muted)', boxShadow: chartType === key ? '0 1px 3px rgba(0, 0, 0, 0.1)' : 'none' }}>
                  {t(`accounts.analytics.chartType.${key}`)}
                </button>
              ))}
            </div>
            <Segmented value={chartPeriod} onChange={(v) => setChartPeriod(v as typeof chartPeriod)}
              options={[
                { label: t('accounts.analytics.chartPeriod.day'), value: 'day' },
                { label: t('accounts.analytics.chartPeriod.week'), value: 'week' },
                { label: t('accounts.analytics.chartPeriod.month'), value: 'month' },
                { label: t('accounts.analytics.chartPeriod.all'), value: 'all' },
              ]} size="small" />
          </div>
          <EquityChart chartType={chartType} chartPeriod={chartPeriod} data={equityChartData} />
        </div>

        <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2" style={{ color: 'var(--color-text)' }}>
              <BarChartOutlined />{t('accounts.analytics.monthlyProfitTitle')}
            </h2>
            <div className="flex items-center gap-2">
              {[currentYear - 2, currentYear - 1, currentYear].map((year) => (
                <Tag key={year} onClick={() => setSelectedYear(year)}
                  style={{ cursor: 'pointer', borderRadius: '6px', padding: '2px 12px', background: selectedYear === year ? '#D4AF37' : 'var(--color-bg-secondary)', color: selectedYear === year ? '#FFFFFF' : 'var(--color-text-muted)', border: 'none', fontWeight: selectedYear === year ? 600 : 400 }}>
                  {year}
                </Tag>
              ))}
            </div>
          </div>
          {profitByMonthData.length > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <ComposedChart data={profitByMonthData}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                <XAxis dataKey="month" type="category" stroke="var(--color-text-muted)" fontSize={11} />
                <YAxis yAxisId="left" stroke="var(--color-text-muted)" fontSize={11} />
                <YAxis yAxisId="right" orientation="right" stroke="var(--color-text-muted)" fontSize={11} />
                <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }} />
                <Legend />
                <Bar yAxisId="left" dataKey="profit" fill="#D4AF37" radius={[4, 4, 0, 0]} name={t('accounts.analytics.chartSeries.profit')} isAnimationActive={false} />
                <Line yAxisId="right" type="monotone" dataKey="trades" stroke="#2196F3" strokeWidth={2} name={t('accounts.analytics.chartSeries.tradeCount')} isAnimationActive={false} />
              </ComposedChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[280px]" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.analytics.empty.monthlyProfit')}</div>
          )}
        </div>
      </div>

      {/* Stats + Symbol Distribution */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="lg:col-span-2 rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4" style={{ color: 'var(--color-text)' }}>
            <TrophyOutlined />{t('accounts.analytics.advancedStatsTitle')}
          </h2>
          <div className="grid grid-cols-3 sm:grid-cols-4 lg:grid-cols-6 gap-2">
            <StatCard icon="🏆" label={t('accounts.analytics.stats.winRate')} value={`${(stats.winRate || 0).toFixed(1)}%`} valueColor="#00A651" />
            <StatCard icon="🎯" label={t('accounts.analytics.stats.profitFactor')} value={`${(stats.profitFactor || 0).toFixed(2)}`} valueColor="#D4AF37" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.maxDrawdown')} value={`${(risks.maxDrawdownPercent || 0).toFixed(2)}%`} valueColor="#E53935" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.totalTrades')} value={`${stats.totalTrades || 0}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.avgProfit')} value={`+${(stats.averageProfit || 0).toFixed(2)}`} valueColor="#00A651" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.avgLoss')} value={`${(stats.averageLoss || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="⏱️" label={t('accounts.analytics.stats.avgHolding')} value={formatHoldingTime(stats.averageHoldingTime) || '-'} valueColor="#9C27B0" />
            <StatCard icon="🔥" label={t('accounts.analytics.stats.consecutiveWinsLosses')} value={`${stats.maxConsecutiveWins || 0}/${stats.maxConsecutiveLosses || 0}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.sharpe')} value={`${(risks.sharpeRatio || 0).toFixed(2)}`} valueColor="#00A651" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.sortino')} value={`${(risks.sortinoRatio || 0).toFixed(2)}`} valueColor="#D4AF37" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.calmar')} value={`${(risks.calmarRatio || 0).toFixed(2)}`} valueColor="#FF9800" />
            <StatCard icon="✨" label={t('accounts.analytics.stats.largestWin')} value={`+${(stats.largestWin || 0).toFixed(2)}`} valueColor="#00A651" background="rgba(212, 175, 55, 0.1)" />
            <StatCard icon="💥" label={t('accounts.analytics.stats.largestLoss')} value={`${(stats.largestLoss || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="📅" label={t('accounts.analytics.stats.avgDailyReturn')} value={`${(risks.averageDailyReturn || 0).toFixed(2)}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.volatility')} value={`${(risks.volatility || 0).toFixed(2)}`} valueColor="#2196F3" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.netProfit')} value={`${(stats.netProfit || 0).toFixed(2)}`} valueColor={(stats.netProfit || 0) >= 0 ? '#00A651' : '#E53935'} />
            <StatCard icon="💰" label={t('accounts.analytics.stats.totalDeposit')} value={`+${(stats.totalDeposit || 0).toFixed(2)}`} valueColor="#D4AF37" background="rgba(212, 175, 55, 0.1)" />
            <StatCard icon="💸" label={t('accounts.analytics.stats.totalWithdrawal')} value={`-${(stats.totalWithdrawal || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.netDeposit')} value={`${(stats.netDeposit || 0).toFixed(2)}`} valueColor={(stats.netDeposit || 0) >= 0 ? '#D4AF37' : '#E53935'} />
          </div>
        </div>

        <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4" style={{ color: 'var(--color-text)' }}>
            <PieChartOutlined />{t('accounts.analytics.symbolDistributionTitle')}
          </h2>
          {symbolDistributionData.length > 0 ? (
            <div className="flex items-center gap-3">
              <ResponsiveContainer width={120} height={120}>
                <PieChart>
                  <Pie data={symbolDistributionData} cx={60} cy={60} innerRadius={35} outerRadius={50} paddingAngle={2} dataKey="value" isAnimationActive={false}>
                    {symbolDistributionData.map((_: unknown, i: number) => <Cell key={`c-${i}`} fill={CHART_COLORS[i % CHART_COLORS.length]} />)}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
              <div className="flex-1">
                {symbolDistributionData.map((item: Record<string, unknown>, i: number) => (
                  <div key={String(item.name)} className="flex items-center justify-between mb-1.5">
                    <div className="flex items-center gap-2">
                      <div className="w-2.5 h-2.5 rounded-full" style={{ background: CHART_COLORS[i % CHART_COLORS.length] }} />
                      <span style={{ color: 'var(--color-text)', fontSize: '12px' }}>{String(item.name)}</span>
                    </div>
                    <span style={{ color: 'var(--color-text-muted)', fontSize: '12px' }}>{String(item.value)}%</span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-[120px]" style={{ color: 'var(--color-text-muted)' }}>{t('accounts.analytics.empty.symbolDistribution')}</div>
          )}
        </div>
      </div>

      {/* Hourly / Daily chart */}
      <HourlyDailyChart hourlyData={hourlyData} dailyPnLData={dailyPnLData} currency={currency || 'USD'} />

      <MonthlyAnalysisCard accountId={accountId} years={monthlyAnalysisYears} data={monthlyAnalysisData} currency={currency} />

      {/* ── Attribution Analysis (collapsed) ── */}
      <Collapse ghost className="mb-6" items={[{
        key: 'attribution',
        label: <span className="text-lg font-semibold" style={{ color: 'var(--color-text)' }}><PieChartOutlined className="mr-2" />{t('accounts.report.symbolPnL')}</span>,
        children: (
          <StatusResult loading={attributionQ.isLoading} error={attributionQ.error?.message}>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Symbol P&L ranking */}
              <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.symbolPnL')}</h3>
                {attr?.symbolPnls?.length ? (
                  <ResponsiveContainer width="100%" height={220}>
                    <ComposedChart layout="vertical" data={attr.symbolPnls.slice(0, 6)}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis type="number" stroke="var(--color-text-muted)" fontSize={10} />
                      <YAxis type="category" dataKey="symbol" width={60} stroke="var(--color-text-muted)" fontSize={10} />
                      <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                      <Bar dataKey="netProfit" radius={[0, 3, 3, 0]} isAnimationActive={false}>
                        {attr.symbolPnls.slice(0, 6).map((_: unknown, i: number) => (
                          <Cell key={i} fill={(attr.symbolPnls?.[i]?.netProfit ?? 0) >= 0 ? '#00A651' : '#E53935'} />
                        ))}
                      </Bar>
                    </ComposedChart>
                  </ResponsiveContainer>
                ) : <div className="flex items-center justify-center h-[220px]" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t('accounts.analytics.empty.monthlyProfit')}</div>}
              </div>
              {/* Direction breakdown */}
              <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.direction')}</h3>
                {attr?.direction ? (
                  <div className="grid grid-cols-2 gap-3">
                    {[
                      { label: t('accounts.report.directionLong'), color: '#00A651', d: { profit: attr.direction.longProfit, trades: attr.direction.longTrades, winRate: attr.direction.longWinRate } },
                      { label: t('accounts.report.directionShort'), color: '#E53935', d: { profit: attr.direction.shortProfit, trades: attr.direction.shortTrades, winRate: attr.direction.shortWinRate } },
                    ].map(({ label, color, d }) => (
                      <div key={label} className="rounded-lg p-3" style={{ border: '1px solid var(--color-border)', background: 'var(--color-bg-card)' }}>
                        <span className="text-sm font-semibold" style={{ color }}>{label}</span>
                        <div className="mt-1 text-xs" style={{ color: 'var(--color-text-secondary)', lineHeight: 1.8 }}>
                          <div>P&L: <strong style={{ color: (d.profit ?? 0) >= 0 ? '#00A651' : '#E53935' }}>{(d.profit ?? 0) >= 0 ? '+' : ''}{(d.profit ?? 0).toFixed(2)}</strong></div>
                          <div>{t('accounts.analytics.stats.totalTrades')}: {d.trades ?? 0}</div>
                          <div>{t('accounts.analytics.stats.winRate')}: {(d.winRate ?? 0).toFixed(1)}%</div>
                        </div>
                      </div>
                    ))}
                  </div>
                ) : <div className="flex items-center justify-center h-[120px]" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t('accounts.analytics.empty.monthlyProfit')}</div>}
              </div>
              {/* Trade profit histogram */}
              {attr?.tradeDistribution?.profitBuckets?.length ? (
                <div className="rounded-xl p-4 lg:col-span-2" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                  <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.tradeDistribution')}</h3>
                  <ResponsiveContainer width="100%" height={200}>
                    <ComposedChart data={attr.tradeDistribution.profitBuckets}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis dataKey="label" stroke="var(--color-text-muted)" fontSize={9} angle={-20} textAnchor="end" height={50} />
                      <YAxis stroke="var(--color-text-muted)" fontSize={10} />
                      <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                      <Bar dataKey="count" fill="#2196F3" radius={[3, 3, 0, 0]} isAnimationActive={false} />
                    </ComposedChart>
                  </ResponsiveContainer>
                </div>
              ) : null}
            </div>
          </StatusResult>
        ),
      }]} />

      {/* ── Time-Series Depth (collapsed) ── */}
      <Collapse ghost className="mb-6" items={[{
        key: 'rolling',
        label: <span className="text-lg font-semibold" style={{ color: 'var(--color-text)' }}><RiseOutlined className="mr-2" />{t('accounts.report.drawdownOverlay')}</span>,
        children: (
          <StatusResult loading={rollingQ.isLoading} error={rollingQ.error?.message}>
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              {/* Equity + drawdown overlay */}
              {roll?.equityCurve?.length ? (
                <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                  <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.drawdownOverlay')}</h3>
                  <ResponsiveContainer width="100%" height={240}>
                    <ComposedChart data={roll.equityCurve.map((p, i) => ({
                      ...p,
                      drawdown: roll.drawdownCurve?.[i]?.drawdownPercent ?? 0,
                    }))}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis dataKey="date" stroke="var(--color-text-muted)" fontSize={9} />
                      <YAxis yAxisId="left" stroke="var(--color-text-muted)" fontSize={10} />
                      <YAxis yAxisId="right" orientation="right" stroke="#E53935" fontSize={10} reversed />
                      <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                      <Line yAxisId="left" type="monotone" dataKey="equity" stroke="#2196F3" strokeWidth={2} dot={false} name="Equity" isAnimationActive={false} />
                      <Line yAxisId="right" type="monotone" dataKey="drawdown" stroke="#E53935" strokeWidth={1} dot={false} name="DD%" isAnimationActive={false} />
                    </ComposedChart>
                  </ResponsiveContainer>
                </div>
              ) : null}
              {/* Drawdown events */}
              {roll?.drawdownEvents?.length ? (
                <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                  <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.drawdownEvents')}</h3>
                  <div className="space-y-2 max-h-[240px] overflow-y-auto">
                    {roll.drawdownEvents.map((e, i) => (
                      <div key={i} className="flex items-center justify-between rounded-lg p-2 text-xs" style={{ border: '1px solid var(--color-border)', background: 'var(--color-bg-card)' }}>
                        <span style={{ color: 'var(--color-text)' }}>{e.startDate} → {e.endDate || '...'}</span>
                        <Tag color="error" style={{ margin: 0 }}>{e.depthPercent.toFixed(1)}%</Tag>
                      </div>
                    ))}
                  </div>
                </div>
              ) : null}
              {/* Monthly win rate trend */}
              {roll?.monthlyWinRates?.length ? (
                <div className="rounded-xl p-4 lg:col-span-2" style={{ background: 'var(--color-bg-secondary)', border: '1px solid var(--color-border)' }}>
                  <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-text-secondary)' }}>{t('accounts.report.winRateTrend')}</h3>
                  <ResponsiveContainer width="100%" height={200}>
                    <ComposedChart data={roll.monthlyWinRates}>
                      <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                      <XAxis dataKey="month" stroke="var(--color-text-muted)" fontSize={9} />
                      <YAxis stroke="var(--color-text-muted)" fontSize={10} domain={[0, 100]} />
                      <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.1)' }} />
                      <Line type="monotone" dataKey="winRate" stroke="#00A651" strokeWidth={2} dot isAnimationActive={false} />
                    </ComposedChart>
                  </ResponsiveContainer>
                </div>
              ) : null}
            </div>
          </StatusResult>
        ),
      }]} />
    </StatusResult>
  );
}

export default React.memo(AccountAnalyticsSection);
