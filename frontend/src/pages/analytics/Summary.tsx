import { useEffect, useState, useMemo } from 'react';
import { Select, Space, Affix } from 'antd';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import { useAccount } from '@/hooks/useAccount';
import { analyticsApi } from '@/client/analytics';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import type { EconomicCalendarEvent, EconomicIndicator } from '@/gen/ant/v1/economic_data_pb';
import { periodOptions } from './Summary.constants';
import {
  getDirectionPieData,
  getEquityCurveData,
  getMonthlyData,
  getProfitPieData,
  getSymbolPieData,
  getYearOptions,
} from './Summary.helpers';
import type { DirectionBreakdownLike } from './Summary.helpers';
import SummaryCharts from './Summary/SummaryCharts';
import SummaryPieGrid from './Summary/SummaryPieGrid';
import SummaryMetricsCards from './Summary/SummaryMetricsCards';
import SummaryTopMetrics from './Summary/SummaryTopMetrics';
import EconomicCalendarSection from './Summary/EconomicCalendarSection';

export default function Summary() {
  const { t } = useTranslation();
  const { accounts, fetchAccounts } = useAccount();
  const [selectedAccount, setSelectedAccount] = useState<string | null>(null);
  const [selectedPeriod, setSelectedPeriod] = useState('month');
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());

  // Fetch accounts on mount — the Analytics page is a direct navigation target
  // and cannot rely on another page having already populated the Zustand store.
  useEffect(() => {
    fetchAccounts();
  }, [fetchAccounts]);

  useEffect(() => {
    if (!selectedAccount && accounts.length > 0) {
      setSelectedAccount(accounts[0].id);
    }
  }, [accounts, selectedAccount]);

  const { data: analytics, isLoading: loading, error: queryError, refetch: refetchAnalytics } = useRpcQuery(
    ['analytics', 'summary', selectedAccount || '', selectedPeriod],
    async () => {
      if (!selectedAccount) return null;
      const [accountAnalytics] = await Promise.all([
        analyticsApi.getAccountAnalytics(selectedAccount, selectedPeriod as 'day' | 'week' | 'month' | 'all'),
      ]);
      return { ...accountAnalytics };
    },
  );

  const error = queryError instanceof Error ? queryError.message : null;

  const { data: calendarEvents = [] } = useRpcQuery(
    ['economicCalendar'],
    async () => {
      const events = await analyticsApi.getEconomicCalendar();
      return (Array.isArray(events) ? events.slice(0, 50) : []) as EconomicCalendarEvent[];
    },
  );

  const { data: keyIndicators = [] } = useRpcQuery(
    ['economicIndicators'],
    async () => {
      const indicators = await analyticsApi.getEconomicIndicators();
      return (Array.isArray(indicators) ? indicators : []) as EconomicIndicator[];
    },
  );

  // Fetch attribution analysis for direction share (long/short breakdown).
  const { data: attribution } = useRpcQuery(
    ['analytics', 'attribution', selectedAccount || ''],
    async () => {
      if (!selectedAccount) return null;
      return analyticsApi.getAttributionAnalysis(selectedAccount);
    },
  );

  const tradeStats = analytics?.tradeStats || null;
  const riskMetrics = analytics?.riskMetrics || null;
  const symbolStats = analytics?.symbolStats || [];

  const equityCurveData = useMemo(() => getEquityCurveData(analytics?.equityCurve || []), [analytics?.equityCurve]);
  const monthlyData = useMemo(() => getMonthlyData(analytics?.dailyPnl || []), [analytics?.dailyPnl]);
  const symbolPieData = useMemo(() => getSymbolPieData(symbolStats), [symbolStats]);
  const directionPieData = useMemo(() => getDirectionPieData(t, attribution?.direction as DirectionBreakdownLike | null | undefined), [t, attribution?.direction]);
  const profitPieData = useMemo(() => getProfitPieData(t, tradeStats), [t, tradeStats]);
  const yearOptions = useMemo(() => getYearOptions(t), [t]);

  // Derive equity/balance from equity curve (last data point).
  const lastCurvePoint = equityCurveData.length > 0 ? equityCurveData[equityCurveData.length - 1] : null;
  const latestEquity = lastCurvePoint?.equity ?? 0;
  const latestBalance = lastCurvePoint?.balance ?? 0;

  return (
    <div className="space-y-6">
      <Affix offsetTop={0} target={() => window}>
        <div className="flex items-center justify-between" style={{ zIndex: 10, background: 'var(--color-bg)', padding: '8px 0' }}>
          <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: 'var(--color-text)', margin: 0 }}>
            {t('analytics.summary.title')}
          </h1>
          <Space>
            <Select value={selectedAccount} onChange={setSelectedAccount} style={{ width: 200 }} placeholder={t('analytics.summary.placeholders.selectAccount')}>
              {(accounts || []).map(a => <Select.Option key={a.id} value={a.id}>{a.login} · {a.brokerServer}</Select.Option>)}
            </Select>
            <Select value={selectedPeriod} onChange={setSelectedPeriod} options={periodOptions(t)} style={{ width: 120 }} />
          </Space>
        </div>
      </Affix>

      <StatusResult loading={loading && !analytics} error={error} onRetry={refetchAnalytics}>
        <SummaryCharts
          equityCurveData={equityCurveData}
          monthlyData={monthlyData}
          selectedYear={selectedYear}
          yearOptions={yearOptions}
          onYearChange={setSelectedYear}
        />

        <SummaryTopMetrics
          netProfit={Number(tradeStats?.netProfit || 0)}
          latestEquity={latestEquity}
          latestBalance={Number(latestBalance)}
        />

        <SummaryPieGrid
          symbolStats={symbolStats}
          symbolPieData={symbolPieData}
          directionPieData={directionPieData}
          profitPieData={profitPieData}
        />

        <SummaryMetricsCards tradeStats={tradeStats} riskMetrics={riskMetrics} />
      </StatusResult>

      {/* Economic calendar — independent section with its own loading/error state */}
      <StatusResult loading={false} error={null}>
        <EconomicCalendarSection calendarEvents={calendarEvents} keyIndicators={keyIndicators} />
      </StatusResult>
    </div>
  );
}
