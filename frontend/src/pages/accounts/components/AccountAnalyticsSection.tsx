import { Segmented, Switch, Tag } from 'antd';
import { FallOutlined } from '@ant-design/icons';
import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next'
import { ANALYTICS_CHART_PERIOD_ALL_KEY, ANALYTICS_CHART_PERIOD_DAY_KEY, ANALYTICS_CHART_PERIOD_MONTH_KEY, ANALYTICS_CHART_PERIOD_WEEK_KEY, REPORT_DIRECTION_LONG_KEY, REPORT_DIRECTION_SHORT_KEY, REPORT_DRAWDOWN_EVENTS_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import { useQuery } from '@tanstack/react-query';
import type { AttributionAnalysisData, RollingMetricsData } from '@/client/analytics';
import { queryKeys } from '@/queries/queryKeys';
import { analyticsApi } from '@/client/analytics';
import { StatusResult } from '@/components/common/StatusResult';
import MonthlyAnalysisCard from './MonthlyAnalysisCard';
import type { MonthlyAnalysisPoint } from './MonthlyAnalysisCard.shared';
import { EquityChart } from './EquityChart';
import { HourlyDailyChart } from './HourlyDailyChart';
import AccountAnalyticsStats from './AccountAnalyticsStats';

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

  const dirCards = useMemo(() => {
    if (!attr?.direction) return [];
    return [
      { key: 'long', label: t(REPORT_DIRECTION_LONG_KEY), color: 'var(--color-success)',
        profit: attr.direction.longProfit ?? 0, trades: attr.direction.longTrades ?? 0, winRate: attr.direction.longWinRate ?? 0 },
      { key: 'short', label: t(REPORT_DIRECTION_SHORT_KEY), color: 'var(--color-danger)',
        profit: attr.direction.shortProfit ?? 0, trades: attr.direction.shortTrades ?? 0, winRate: attr.direction.shortWinRate ?? 0 },
    ];
  }, [attr?.direction, t]);

  return (
    <StatusResult loading={analyticsLoading} error={analyticsError} onRetry={onRetryAnalytics}>
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
                { label: t(ANALYTICS_CHART_PERIOD_DAY_KEY), value: 'day' },
                { label: t(ANALYTICS_CHART_PERIOD_WEEK_KEY), value: 'week' },
                { label: t(ANALYTICS_CHART_PERIOD_MONTH_KEY), value: 'month' },
                { label: t(ANALYTICS_CHART_PERIOD_ALL_KEY), value: 'all' },
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
              <FallOutlined className="mr-1" />{t(REPORT_DRAWDOWN_EVENTS_KEY)}
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

      <MonthlyAnalysisCard
        accountId={accountId}
        years={monthlyAnalysisYears}
        data={monthlyAnalysisData as unknown as MonthlyAnalysisPoint[]}
        winRateData={roll?.monthlyWinRates}
        currency={currency}
      />

      <HourlyDailyChart hourlyData={hourlyData} dailyPnLData={dailyPnLData} currency={currency || 'USD'} />

      <AccountAnalyticsStats
        tradeStats={tradeStats}
        riskMetrics={riskMetrics}
        symbolDistributionData={symbolDistributionData}
        attributionLoading={attributionQ.isLoading}
        attributionError={attributionQ.error?.message}
        attr={attr}
        dirCards={dirCards}
      />
    </StatusResult>
  );
}

export default React.memo(AccountAnalyticsSection);
