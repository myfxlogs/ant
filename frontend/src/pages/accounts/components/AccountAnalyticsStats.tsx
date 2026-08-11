import { BarChartOutlined, TrophyOutlined, DownOutlined, RightOutlined } from '@ant-design/icons';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Bar, CartesianGrid, ComposedChart, Cell, Pie, PieChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import {
  ANALYTICS_ADVANCED_STATS_TITLE_KEY, ANALYTICS_STATS_AVG_DAILY_RETURN_KEY,
  ANALYTICS_STATS_AVG_HOLDING_KEY, ANALYTICS_STATS_AVG_LOSS_KEY, ANALYTICS_STATS_AVG_PROFIT_KEY,
  ANALYTICS_STATS_CALMAR_KEY, ANALYTICS_STATS_CONSECUTIVE_WINS_LOSSES_KEY, ANALYTICS_STATS_LARGEST_LOSS_KEY,
  ANALYTICS_STATS_LARGEST_WIN_KEY, ANALYTICS_STATS_MAX_DRAWDOWN_KEY, ANALYTICS_STATS_NET_DEPOSIT_KEY,
  ANALYTICS_STATS_NET_PROFIT_KEY, ANALYTICS_STATS_PROFIT_FACTOR_KEY, ANALYTICS_STATS_SHARPE_KEY,
  ANALYTICS_STATS_SORTINO_KEY, ANALYTICS_STATS_TOTAL_DEPOSIT_KEY, ANALYTICS_STATS_TOTAL_TRADES_KEY,
  ANALYTICS_STATS_TOTAL_WITHDRAWAL_KEY, ANALYTICS_STATS_VOLATILITY_KEY, ANALYTICS_STATS_WIN_RATE_KEY,
  ANALYTICS_EMPTY_SYMBOL_DISTRIBUTION_KEY,
  REPORT_SYMBOL_PN_L_KEY,
} from '@/gen/ant/v1/i18n/accounts_keys';
import { CHART_COLORS } from '@/constants/performance';
import { formatHoldingTime } from '@/utils/date';
import { StatusResult } from '@/components/common/StatusResult';
import type { AttributionAnalysisData } from '@/client/analytics';

const StatCell = React.memo(({ label, value, color = 'var(--color-text)' }: { label: string; value: string; color?: string }) => (
  <div style={{ minWidth: 0 }}>
    <div style={{ color: 'var(--color-text-muted)', fontSize: 10, lineHeight: 1.4 }}>{label}</div>
    <div style={{ color, fontSize: 13, fontWeight: 600, lineHeight: 1.4 }}>{value}</div>
  </div>
));

const cardStyle = {
  background: 'var(--color-bg-card)',
  border: '1px solid var(--color-border)',
  borderRadius: 12,
  boxShadow: '0 1px 3px var(--color-shadow)',
} as const;

interface StatsProps {
  tradeStats: Record<string, number>;
  riskMetrics: Record<string, number>;
  symbolDistributionData: Record<string, unknown>[];
  attributionLoading: boolean;
  attributionError?: string;
  attr?: AttributionAnalysisData | null;
  dirCards: Array<{ key: string; label: string; color: string; profit: number; trades: number; winRate: number }>;
}

export default function AccountAnalyticsStats({
  tradeStats, riskMetrics, symbolDistributionData,
  attributionLoading, attributionError, attr, dirCards,
}: StatsProps) {
  const { t } = useTranslation();
  const [statsExpanded, setStatsExpanded] = useState(true);
  const stats = tradeStats;
  const risks = riskMetrics;

  return (
    <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 mb-4">
      <div className="p-4" style={cardStyle}>
        <button
          onClick={() => setStatsExpanded((v) => !v)}
          className="w-full flex items-center justify-between mb-3 cursor-pointer"
          style={{ background: 'none', border: 'none', padding: 0, color: 'var(--color-text)' }}
        >
          <h2 className="text-base font-semibold flex items-center gap-2 m-0" style={{ color: 'var(--color-text)' }}>
            <TrophyOutlined />{t(ANALYTICS_ADVANCED_STATS_TITLE_KEY)}
          </h2>
          {statsExpanded
            ? <DownOutlined style={{ fontSize: 12, color: 'var(--color-text-muted)' }} />
            : <RightOutlined style={{ fontSize: 12, color: 'var(--color-text-muted)' }} />}
        </button>
        {statsExpanded && (
          <div className="grid grid-cols-3 gap-x-3 gap-y-2">
            <StatCell label={t(ANALYTICS_STATS_WIN_RATE_KEY)} value={`${(stats.winRate || 0).toFixed(1)}%`} color="var(--color-success)" />
            <StatCell label={t(ANALYTICS_STATS_PROFIT_FACTOR_KEY)} value={`${(stats.profitFactor || 0).toFixed(2)}`} color="var(--color-primary)" />
            <StatCell label={t(ANALYTICS_STATS_MAX_DRAWDOWN_KEY)} value={`${(risks.maxDrawdownPercent || 0).toFixed(2)}%`} color="var(--color-danger)" />
            <StatCell label={t(ANALYTICS_STATS_TOTAL_TRADES_KEY)} value={`${stats.totalTrades || 0}`} />
            <StatCell label={t(ANALYTICS_STATS_AVG_PROFIT_KEY)} value={`+${(stats.averageProfit || 0).toFixed(2)}`} color="var(--color-success)" />
            <StatCell label={t(ANALYTICS_STATS_AVG_LOSS_KEY)} value={`${(stats.averageLoss || 0).toFixed(2)}`} color="var(--color-danger)" />
            <StatCell label={t(ANALYTICS_STATS_AVG_HOLDING_KEY)} value={formatHoldingTime(String(stats.averageHoldingTime ?? '')) || '-'} />
            <StatCell label={t(ANALYTICS_STATS_CONSECUTIVE_WINS_LOSSES_KEY)} value={`${stats.maxConsecutiveWins || 0}/${stats.maxConsecutiveLosses || 0}`} />
            <StatCell label={t(ANALYTICS_STATS_SHARPE_KEY)} value={`${(risks.sharpeRatio || 0).toFixed(2)}`} color="var(--color-success)" />
            <StatCell label={t(ANALYTICS_STATS_SORTINO_KEY)} value={`${(risks.sortinoRatio || 0).toFixed(2)}`} />
            <StatCell label={t(ANALYTICS_STATS_CALMAR_KEY)} value={`${(risks.calmarRatio || 0).toFixed(2)}`} />
            <StatCell label={t(ANALYTICS_STATS_LARGEST_WIN_KEY)} value={`+${(stats.largestWin || 0).toFixed(2)}`} color="var(--color-success)" />
            <StatCell label={t(ANALYTICS_STATS_LARGEST_LOSS_KEY)} value={`${(stats.largestLoss || 0).toFixed(2)}`} color="var(--color-danger)" />
            <StatCell label={t(ANALYTICS_STATS_AVG_DAILY_RETURN_KEY)} value={`${(risks.averageDailyReturn || 0).toFixed(2)}`} />
            <StatCell label={t(ANALYTICS_STATS_VOLATILITY_KEY)} value={`${(risks.volatility || 0).toFixed(2)}`} color="var(--color-info)" />
            <StatCell label={t(ANALYTICS_STATS_NET_PROFIT_KEY)} value={`${(stats.netProfit || 0).toFixed(2)}`} color={(stats.netProfit || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
            <StatCell label={t(ANALYTICS_STATS_TOTAL_DEPOSIT_KEY)} value={`+${(stats.totalDeposit || 0).toFixed(2)}`} />
            <StatCell label={t(ANALYTICS_STATS_TOTAL_WITHDRAWAL_KEY)} value={`-${(stats.totalWithdrawal || 0).toFixed(2)}`} />
            <StatCell label={t(ANALYTICS_STATS_NET_DEPOSIT_KEY)} value={`${(stats.netDeposit || 0).toFixed(2)}`} color={(stats.netDeposit || 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
          </div>
        )}
      </div>

      <div className="p-4" style={cardStyle}>
        <h2 className="text-base font-semibold flex items-center gap-2 mb-3" style={{ color: 'var(--color-text)' }}>
          <BarChartOutlined />{t(REPORT_SYMBOL_PN_L_KEY)}
        </h2>
        <StatusResult loading={attributionLoading} error={attributionError}>
          {attr?.symbolPnls?.length ? (
            <div className="space-y-3">
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

              {dirCards.length > 0 && (
                <div className="grid grid-cols-2 gap-2">
                  {dirCards.map(({ key, label, color, profit, trades, winRate }) => (
                    <div key={key} className="rounded-lg p-2"
                      style={{ border: '1px solid var(--color-border)', background: 'var(--color-bg-secondary)' }}>
                      <div className="text-xs font-semibold" style={{ color }}>{label}</div>
                      <div className="text-xs" style={{ color: 'var(--color-text-secondary)', lineHeight: 1.6 }}>
                        <div>{t('analytics.pnl', { defaultValue: 'P&L:' })} <strong style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>{profit >= 0 ? '+' : ''}{profit.toFixed(2)}</strong></div>
                        <div>{t(ANALYTICS_STATS_TOTAL_TRADES_KEY)}: {trades}</div>
                        <div>{t(ANALYTICS_STATS_WIN_RATE_KEY)}: {winRate.toFixed(1)}%</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : (
            <div className="flex items-center justify-center h-[200px]" style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>
              {t(ANALYTICS_EMPTY_SYMBOL_DISTRIBUTION_KEY)}
            </div>
          )}
        </StatusResult>
      </div>
    </div>
  );
}
