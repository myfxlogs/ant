import { Segmented, Spin, Tag } from 'antd';
import {
  Area,
  AreaChart,
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  Legend,
  Line,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import {
  IconChartBar,
  IconChartPie,
  IconTrophy,
} from '@tabler/icons-react';

import { CHART_COLORS } from '@/constants/performance';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { formatHoldingTime } from '@/utils/date';

import { StatCard } from './AccountDetail.shared';
import MonthlyAnalysisCard from './MonthlyAnalysisCard';

type Props = {
  analyticsLoading: boolean;
  chartType: 'equity' | 'balance' | 'profit';
  setChartType: (value: 'equity' | 'balance' | 'profit') => void;
  chartPeriod: 'day' | 'week' | 'month' | 'all';
  setChartPeriod: (value: 'day' | 'week' | 'month' | 'all') => void;
  selectedYear: number;
  setSelectedYear: (value: number) => void;
  equityChartData: any[];
  profitByMonthData: any[];
  symbolDistributionData: any[];
  dailyPnLData: any[];
  hourlyData: any[];
  tradeStats: any;
  riskMetrics: any;
  monthlyAnalysisYears: number[];
  monthlyAnalysisData: any[];
  currency?: string;
  accountId?: string;
};

export default function AccountAnalyticsSection({
  analyticsLoading,
  chartType,
  setChartType,
  chartPeriod,
  setChartPeriod,
  selectedYear,
  setSelectedYear,
  equityChartData,
  profitByMonthData,
  symbolDistributionData,
  dailyPnLData,
  hourlyData,
  tradeStats,
  riskMetrics,
  monthlyAnalysisYears,
  monthlyAnalysisData,
  currency,
  accountId,
}: Props) {
  const { t } = useTranslation();
  const [timeView, setTimeView] = useState<'hourly' | 'daily'>('hourly');
  const [selectedHourlyIndex, setSelectedHourlyIndex] = useState(0);
  const [selectedDailyIndex, setSelectedDailyIndex] = useState(0);

  // 数据长度变化导致旧 index 越界时，直接在渲染期 clamp，
  // 避免 useEffect 内 setState 触发额外渲染（react-hooks/set-state-in-effect）。
  const safeHourlyIndex =
    selectedHourlyIndex < hourlyData.length ? selectedHourlyIndex : 0;
  const safeDailyIndex =
    selectedDailyIndex < dailyPnLData.length ? selectedDailyIndex : 0;

  const selectedHourly = hourlyData[safeHourlyIndex] || null;
  const selectedDaily = dailyPnLData[safeDailyIndex] || null;
  const selectedTimePoint = timeView === 'hourly' ? selectedHourly : selectedDaily;

  const formatMoney = (value: number) => `${value >= 0 ? '+' : ''}${Number(value || 0).toFixed(2)} ${currency || 'USD'}`;
  const formatRatio = (value: number) => `${Number(value || 0).toFixed(2)}%`;
  const preventChartFocus = (e: React.MouseEvent) => e.preventDefault();
  type RechartsMouseState = { activeTooltipIndex?: number | string };
  const pickTooltipIndex = (state: RechartsMouseState, len: number): number | null => {
    const idx = state?.activeTooltipIndex;
    if (typeof idx !== 'number') return null;
    if (idx < 0 || idx >= len) return null;
    return idx;
  };

  if (analyticsLoading) {
    return (
      <div className="flex justify-center items-center py-12">
        <Spin size="large" />
      </div>
    );
  }

  return (
    <>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
        <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <div className="flex items-center justify-between mb-4">
            <div className="flex gap-1 p-1 rounded-lg" style={{ background: '#F5F7F9' }}>
              {[{ key: 'equity', label: t('accounts.analytics.chartType.equity') }, { key: 'balance', label: t('accounts.analytics.chartType.balance') }, { key: 'profit', label: t('accounts.analytics.chartType.profit') }].map((item) => (
                <button
                  key={item.key}
                  onClick={() => setChartType(item.key as typeof chartType)}
                  className="px-4 py-1.5 rounded-md text-sm font-medium transition-all"
                  style={{
                    background: chartType === item.key ? '#FFFFFF' : 'transparent',
                    color: chartType === item.key ? '#141D22' : '#8A9AA5',
                    boxShadow: chartType === item.key ? '0 1px 3px rgba(0, 0, 0, 0.1)' : 'none',
                  }}
                >
                  {item.label}
                </button>
              ))}
            </div>
            <Segmented
              value={chartPeriod}
              onChange={(v) => setChartPeriod(v as typeof chartPeriod)}
              options={[
                { label: t('accounts.analytics.chartPeriod.day'), value: 'day' },
                { label: t('accounts.analytics.chartPeriod.week'), value: 'week' },
                { label: t('accounts.analytics.chartPeriod.month'), value: 'month' },
                { label: t('accounts.analytics.chartPeriod.all'), value: 'all' },
              ]}
              size="small"
            />
          </div>
          {equityChartData.length > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <AreaChart data={equityChartData}>
                <defs>
                  <linearGradient id="colorEquityGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#D4AF37" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#D4AF37" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorBalanceGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#2196F3" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#2196F3" stopOpacity={0} />
                  </linearGradient>
                  <linearGradient id="colorProfitGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#00A651" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#00A651" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                <XAxis dataKey="date" stroke="#8A9AA5" fontSize={11} />
                <YAxis stroke="#8A9AA5" fontSize={11} />
                <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }} />
                {chartType === 'equity' && (
                  <Area type="monotone" dataKey="equity" stroke="#D4AF37" strokeWidth={2} fillOpacity={1} fill="url(#colorEquityGradient)" name={t('accounts.analytics.chartSeries.equity')} isAnimationActive={false} />
                )}
                {chartType === 'balance' && (
                  <Area type="monotone" dataKey="balance" stroke="#2196F3" strokeWidth={2} fillOpacity={1} fill="url(#colorBalanceGradient)" name={t('accounts.analytics.chartSeries.balance')} isAnimationActive={false} />
                )}
                {chartType === 'profit' && (
                  <Area type="monotone" dataKey="profit" stroke="#00A651" strokeWidth={2} fillOpacity={1} fill="url(#colorProfitGradient)" name={t('accounts.analytics.chartSeries.profit')} isAnimationActive={false} />
                )}
              </AreaChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[280px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.equityCurve')}</div>
          )}
        </div>

        <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2" style={{ color: '#141D22' }}>
              <IconChartBar size={18} stroke={1.5} />
              {t('accounts.analytics.monthlyProfitTitle')}
            </h2>
            <div className="flex items-center gap-2">
              {[2024, 2025, 2026].map((year) => (
                <Tag
                  key={year}
                  onClick={() => setSelectedYear(year)}
                  style={{
                    cursor: 'pointer',
                    borderRadius: '6px',
                    padding: '2px 12px',
                    background: selectedYear === year ? '#D4AF37' : '#F5F7F9',
                    color: selectedYear === year ? '#FFFFFF' : '#8A9AA5',
                    border: 'none',
                    fontWeight: selectedYear === year ? 600 : 400,
                  }}
                >
                  {year}
                </Tag>
              ))}
            </div>
          </div>
          {profitByMonthData.length > 0 ? (
            <ResponsiveContainer width="100%" height={280}>
              <ComposedChart data={profitByMonthData}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                <XAxis dataKey="month" type="category" stroke="#8A9AA5" fontSize={11} />
                <YAxis yAxisId="left" stroke="#8A9AA5" fontSize={11} />
                <YAxis yAxisId="right" orientation="right" stroke="#8A9AA5" fontSize={11} />
                <Tooltip contentStyle={{ background: '#FFFFFF', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)' }} />
                <Legend />
                <Bar yAxisId="left" dataKey="profit" fill="#D4AF37" radius={[4, 4, 0, 0]} name={t('accounts.analytics.chartSeries.profit')} isAnimationActive={false} />
                <Line yAxisId="right" type="monotone" dataKey="trades" stroke="#2196F3" strokeWidth={2} name={t('accounts.analytics.chartSeries.tradeCount')} isAnimationActive={false} />
              </ComposedChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex items-center justify-center h-[280px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.monthlyProfit')}</div>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6 mb-6">
        <div className="lg:col-span-2 rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4" style={{ color: '#141D22' }}>
            <IconTrophy size={18} stroke={1.5} />
            {t('accounts.analytics.advancedStatsTitle')}
          </h2>
          <div className="grid grid-cols-3 sm:grid-cols-4 lg:grid-cols-6 gap-2">
            <StatCard icon="🏆" label={t('accounts.analytics.stats.winRate')} value={`${(tradeStats.winRate || 0).toFixed(1)}%`} valueColor="#00A651" />
            <StatCard icon="🎯" label={t('accounts.analytics.stats.profitFactor')} value={`${(tradeStats.profitFactor || 0).toFixed(2)}`} valueColor="#D4AF37" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.maxDrawdown')} value={`${(riskMetrics.maxDrawdownPercent || 0).toFixed(2)}%`} valueColor="#E53935" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.totalTrades')} value={`${tradeStats.totalTrades || 0}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.avgProfit')} value={`+${(tradeStats.averageProfit || 0).toFixed(2)}`} valueColor="#00A651" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.avgLoss')} value={`${(tradeStats.averageLoss || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="⏱️" label={t('accounts.analytics.stats.avgHolding')} value={formatHoldingTime(tradeStats.averageHoldingTime) || '-'} valueColor="#9C27B0" />
            <StatCard icon="🔥" label={t('accounts.analytics.stats.consecutiveWinsLosses')} value={`${tradeStats.maxConsecutiveWins || 0}/${tradeStats.maxConsecutiveLosses || 0}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.sharpe')} value={`${(riskMetrics.sharpeRatio || 0).toFixed(2)}`} valueColor="#00A651" />
            <StatCard icon="📉" label={t('accounts.analytics.stats.sortino')} value={`${(riskMetrics.sortinoRatio || 0).toFixed(2)}`} valueColor="#D4AF37" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.calmar')} value={`${(riskMetrics.calmarRatio || 0).toFixed(2)}`} valueColor="#FF9800" />
            <StatCard icon="✨" label={t('accounts.analytics.stats.largestWin')} value={`+${(tradeStats.largestWin || 0).toFixed(2)}`} valueColor="#00A651" background="rgba(212, 175, 55, 0.1)" />
            <StatCard icon="💥" label={t('accounts.analytics.stats.largestLoss')} value={`${(tradeStats.largestLoss || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="📅" label={t('accounts.analytics.stats.avgDailyReturn')} value={`${(riskMetrics.averageDailyReturn || 0).toFixed(2)}`} />
            <StatCard icon="📈" label={t('accounts.analytics.stats.volatility')} value={`${(riskMetrics.volatility || 0).toFixed(2)}`} valueColor="#2196F3" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.netProfit')} value={`${(tradeStats.netProfit || 0).toFixed(2)}`} valueColor={(tradeStats.netProfit || 0) >= 0 ? '#00A651' : '#E53935'} />
            <StatCard icon="💰" label={t('accounts.analytics.stats.totalDeposit')} value={`+${(tradeStats.totalDeposit || 0).toFixed(2)}`} valueColor="#D4AF37" background="rgba(212, 175, 55, 0.1)" />
            <StatCard icon="💸" label={t('accounts.analytics.stats.totalWithdrawal')} value={`-${(tradeStats.totalWithdrawal || 0).toFixed(2)}`} valueColor="#E53935" />
            <StatCard icon="📊" label={t('accounts.analytics.stats.netDeposit')} value={`${(tradeStats.netDeposit || 0).toFixed(2)}`} valueColor={(tradeStats.netDeposit || 0) >= 0 ? '#D4AF37' : '#E53935'} />
          </div>
        </div>

        <div className="rounded-2xl p-5" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
          <h2 className="text-lg font-semibold flex items-center gap-2 mb-4" style={{ color: '#141D22' }}>
            <IconChartPie size={18} stroke={1.5} />
            {t('accounts.analytics.symbolDistributionTitle')}
          </h2>
          {symbolDistributionData.length > 0 ? (
            <div className="flex items-center gap-3">
              <ResponsiveContainer width={120} height={120}>
                <PieChart>
                  <Pie data={symbolDistributionData} cx={60} cy={60} innerRadius={35} outerRadius={50} paddingAngle={2} dataKey="value" isAnimationActive={false}>
                    {symbolDistributionData.map((_: any, index: any) => (
                      <Cell key={`cell-${index}`} fill={CHART_COLORS[index % CHART_COLORS.length]} />
                    ))}
                  </Pie>
                </PieChart>
              </ResponsiveContainer>
              <div className="flex-1">
                {symbolDistributionData.map((item: any, index: any) => (
                  <div key={item.name} className="flex items-center justify-between mb-1.5">
                    <div className="flex items-center gap-2">
                      <div className="w-2.5 h-2.5 rounded-full" style={{ background: CHART_COLORS[index % CHART_COLORS.length] }} />
                      <span style={{ color: '#141D22', fontSize: '12px' }}>{item.name}</span>
                    </div>
                    <span style={{ color: '#8A9AA5', fontSize: '12px' }}>{item.value}%</span>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-center h-[120px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.symbolDistribution')}</div>
          )}
        </div>
      </div>

      <div className="rounded-2xl p-5 mb-6" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold" style={{ color: '#141D22' }}>
            {t('accounts.analytics.hourlyTitle')} / {t('accounts.analytics.dailyPnLTitle')}
          </h2>
          <div className="inline-flex rounded-lg p-1" style={{ background: '#F5F7F9' }}>
            {([
              { key: 'hourly', label: t('accounts.analytics.advancedTabs.hourly') },
              { key: 'daily', label: t('accounts.analytics.advancedTabs.daily') },
            ] as const).map((tab) => (
              <button
                key={tab.key}
                onClick={() => setTimeView(tab.key)}
                className="px-3 py-1.5 rounded text-xs font-semibold transition-all"
                style={{
                  background: timeView === tab.key ? '#FFFFFF' : 'transparent',
                  color: timeView === tab.key ? '#141D22' : '#8A9AA5',
                  boxShadow: timeView === tab.key ? '0 1px 3px rgba(0, 0, 0, 0.08)' : 'none',
                }}
              >
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
          <div className="xl:col-span-2 min-w-0">
            {timeView === 'hourly' ? (
              hourlyData.length > 0 ? (
                <div
                  className="outline-none [&_.recharts-wrapper]:!outline-none [&_.recharts-surface]:outline-none"
                  onMouseDown={preventChartFocus}
                >
                <ResponsiveContainer width="100%" height={250}>
                  <ComposedChart
                    data={hourlyData}
                    onMouseMove={(state) => {
                      const idx = pickTooltipIndex(state as RechartsMouseState, hourlyData.length);
                      if (idx != null) setSelectedHourlyIndex(idx);
                    }}
                    onClick={(state) => {
                      const idx = pickTooltipIndex(state as RechartsMouseState, hourlyData.length);
                      if (idx != null) setSelectedHourlyIndex(idx);
                    }}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis dataKey="hourLabel" stroke="#8A9AA5" fontSize={10} />
                    <YAxis yAxisId="left" stroke="#8A9AA5" fontSize={10} />
                    <YAxis yAxisId="right" orientation="right" stroke="#8A9AA5" fontSize={10} />
                    <Tooltip
                      cursor={false}
                      wrapperStyle={{ pointerEvents: 'none' }}
                      contentStyle={{ background: '#FFFFFF', border: '1px solid #E5E7EB', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)' }}
                    />
                    <Bar
                      yAxisId="left"
                      dataKey="trades"
                      radius={[3, 3, 0, 0]}
                      barSize={18}
                      isAnimationActive={false}
                    >
                      {hourlyData.map((row: any, index: number) => (
                        <Cell
                          key={`hour-${index}`}
                          fill={index === safeHourlyIndex ? '#2B6CB0' : '#64B5F6'}
                          style={{ cursor: 'pointer' }}
                          onClick={() => setSelectedHourlyIndex(index)}
                        />
                      ))}
                    </Bar>
                    <Line
                      yAxisId="right"
                      type="monotone"
                      dataKey="profit"
                      stroke="#FF9800"
                      strokeWidth={2}
                      dot={false}
                      activeDot={false}
                      style={{ pointerEvents: 'none' }}
                      isAnimationActive={false}
                    />
                  </ComposedChart>
                </ResponsiveContainer>
                </div>
              ) : (
                <div className="flex items-center justify-center h-[250px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.hourly')}</div>
              )
            ) : (
              dailyPnLData.length > 0 ? (
                <div
                  className="outline-none [&_.recharts-wrapper]:!outline-none [&_.recharts-surface]:outline-none"
                  onMouseDown={preventChartFocus}
                >
                <ResponsiveContainer width="100%" height={250}>
                  <ComposedChart
                    data={dailyPnLData}
                    onMouseMove={(state) => {
                      const idx = pickTooltipIndex(state as RechartsMouseState, dailyPnLData.length);
                      if (idx != null) setSelectedDailyIndex(idx);
                    }}
                    onClick={(state) => {
                      const idx = pickTooltipIndex(state as RechartsMouseState, dailyPnLData.length);
                      if (idx != null) setSelectedDailyIndex(idx);
                    }}
                  >
                    <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                    <XAxis dataKey="date" stroke="#8A9AA5" fontSize={10} />
                    <YAxis yAxisId="left" stroke="#8A9AA5" fontSize={10} />
                    <YAxis yAxisId="right" orientation="right" stroke="#8A9AA5" fontSize={10} />
                    <Tooltip
                      cursor={false}
                      wrapperStyle={{ pointerEvents: 'none' }}
                      contentStyle={{ background: '#FFFFFF', border: '1px solid #E5E7EB', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)' }}
                    />
                    <Bar
                      yAxisId="left"
                      dataKey="trades"
                      radius={[3, 3, 0, 0]}
                      barSize={24}
                      isAnimationActive={false}
                    >
                      {dailyPnLData.map((row: any, index: number) => (
                        <Cell
                          key={`day-${index}`}
                          fill={index === safeDailyIndex ? '#2B6CB0' : '#4DB6AC'}
                          style={{ cursor: 'pointer' }}
                          onClick={() => setSelectedDailyIndex(index)}
                        />
                      ))}
                    </Bar>
                    <Line
                      yAxisId="right"
                      type="monotone"
                      dataKey="profit"
                      stroke="#FF9800"
                      strokeWidth={2}
                      dot={false}
                      activeDot={false}
                      style={{ pointerEvents: 'none' }}
                      isAnimationActive={false}
                    />
                  </ComposedChart>
                </ResponsiveContainer>
                </div>
              ) : (
                <div className="flex items-center justify-center h-[250px]" style={{ color: '#8A9AA5' }}>{t('accounts.analytics.empty.dailyPnL')}</div>
              )
            )}
          </div>

          <div className="rounded-xl p-3 border" style={{ borderColor: '#E5E7EB', background: '#F8FAFC' }}>
            <div className="text-sm font-semibold mb-2" style={{ color: '#1F2937' }}>
              {timeView === 'hourly'
                ? (selectedTimePoint?.hourLabel || '--')
                : `${selectedTimePoint?.date || '--'} ${selectedTimePoint?.day || ''}`}
            </div>
            <div className="text-xs space-y-1.5" style={{ color: '#475467' }}>
              <div>{t('accounts.analytics.timeDetail.lots')}: <span className="font-semibold">{Number(selectedTimePoint?.lots || 0).toFixed(2)}</span></div>
              <div>{t('accounts.analytics.timeDetail.trades')}: <span className="font-semibold">{Number(selectedTimePoint?.trades || 0)}</span></div>
              <div>{t('accounts.analytics.timeDetail.profitAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.profit || 0))}</span></div>
              <div>{t('accounts.analytics.timeDetail.balance')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.balance || 0))}</span></div>
              <div>{t('accounts.analytics.timeDetail.profitFactor')}: <span className="font-semibold">{Number(selectedTimePoint?.profitFactor || 0).toFixed(2)}</span></div>
              <div>{t('accounts.analytics.timeDetail.maxFloatingLossAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.maxFloatingLossAmount || 0))}</span></div>
              <div>{t('accounts.analytics.timeDetail.maxFloatingLossRatio')}: <span className="font-semibold">{formatRatio(Number(selectedTimePoint?.maxFloatingLossRatio || 0))}</span></div>
              <div>{t('accounts.analytics.timeDetail.maxFloatingProfitAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.maxFloatingProfitAmount || 0))}</span></div>
              <div>{t('accounts.analytics.timeDetail.maxFloatingProfitRatio')}: <span className="font-semibold">{formatRatio(Number(selectedTimePoint?.maxFloatingProfitRatio || 0))}</span></div>
            </div>
          </div>
        </div>
      </div>

      <MonthlyAnalysisCard
        accountId={accountId}
        years={monthlyAnalysisYears}
        data={monthlyAnalysisData}
        currency={currency}
      />

    </>
  );
}
