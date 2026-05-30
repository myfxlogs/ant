import { useEffect, useState } from 'react';
import { Select, Row, Col, Space } from 'antd';
import { StatusResult } from '@/components/common/StatusResult';
import { useTranslation } from 'react-i18next';
import { ResponsiveContainer, LineChart, Line } from 'recharts';
import {
  RiseOutlined, LineChartOutlined, PieChartOutlined, AimOutlined,
} from '@ant-design/icons';
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
import SummaryCharts from './Summary/SummaryCharts';
import SummaryPieGrid from './Summary/SummaryPieGrid';
import SummaryMetricsCards from './Summary/SummaryMetricsCards';

interface AnalyticsTradeStats {
  netProfit?: number;
  totalTrades?: number;
  winningTrades?: number;
  losingTrades?: number;
  buyTrades?: number;
  sellTrades?: number;
  winRate?: number;
  profitFactor?: number;
  averageHoldingTime?: string;
  maxConsecutiveWins?: number;
  maxConsecutiveLosses?: number;
  averageVolume?: number;
  averageProfit?: number;
  averageLoss?: number;
}

interface AnalyticsRiskMetrics {
  maxDrawdown?: number;
  maxDrawdownPercent?: number;
  sharpeRatio?: number;
  sortinoRatio?: number;
  volatility?: number;
  valueAtRisk?: number;
}

export default function Summary() {
  const { t } = useTranslation();
  const { accounts } = useAccount();
  const [selectedAccount, setSelectedAccount] = useState<string | null>(null);
  const [selectedPeriod, setSelectedPeriod] = useState('month');
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());

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
        analyticsApi.getAccountAnalytics({ accountId: selectedAccount, period: selectedPeriod }),
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

  const tradeStats = analytics?.tradeStats || null;
  const riskMetrics = analytics?.riskMetrics || null;
  const symbolStats = analytics?.symbolStats || [];
  const equityCurveData = getEquityCurveData(analytics?.equityCurve || []);
  const monthlyData = getMonthlyData(analytics?.monthlyPnl || []);
  const symbolPieData = getSymbolPieData(symbolStats);
  const directionPieData = getDirectionPieData(t, tradeStats);
  const profitPieData = getProfitPieData(t, tradeStats);
  const yearOptions = getYearOptions(t);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold" style={{ fontFamily: 'Poppins, sans-serif', color: '#141D22' }}>
          {t('analytics.summary.title')}
        </h1>
        <Space>
          <Select value={selectedAccount} onChange={setSelectedAccount} style={{ width: 200 }} placeholder={t('analytics.summary.placeholders.selectAccount')}>
            {(accounts || []).map(a => <Select.Option key={a.id} value={a.id}>{a.alias}</Select.Option>)}
          </Select>
          <Select value={selectedPeriod} onChange={setSelectedPeriod} options={periodOptions(t)} style={{ width: 120 }} />
        </Space>
      </div>

      <StatusResult loading={loading && !analytics} error={error} onRetry={refetchAnalytics}>
        <SummaryCharts
          equityCurveData={equityCurveData}
          monthlyData={monthlyData}
          selectedYear={selectedYear}
          yearOptions={yearOptions}
          onYearChange={setSelectedYear}
        />

        <Row gutter={[16, 16]} className="mt-6">
          {[
            { icon: <RiseOutlined />, color: '#00A651', label: t('analytics.summary.metrics.netProfit'), value: `$${(Number(analytics?.profit || tradeStats?.netProfit || 0)).toFixed(2)}`, valueColor: (Number(analytics?.profit || tradeStats?.netProfit || 0)) >= 0 ? '#00A651' : '#E53935' },
            { icon: <LineChartOutlined />, color: '#2196F3', label: t('analytics.summary.metrics.equity'), value: `$${(Number(analytics?.equity || 0)).toFixed(2)}`, valueColor: '#141D22' },
            { icon: <AimOutlined />, color: '#D4AF37', label: t('analytics.summary.metrics.balance'), value: `$${(Number(analytics?.balance || 0)).toFixed(2)}`, valueColor: '#141D22' },
            { icon: <PieChartOutlined />, color: '#9C27B0', label: t('analytics.summary.metrics.equityValue'), value: `$${(Number(analytics?.equity || 0)).toFixed(2)}`, valueColor: '#141D22' },
          ].map((s, i) => (
            <Col xs={12} sm={6} key={i}>
              <div className="stat-card">
                <div className="flex items-center gap-2 mb-2">
                  {s.icon}
                  <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{s.label}</span>
                </div>
                <div className="text-2xl font-semibold" style={{ color: s.valueColor }}>{s.value}</div>
              </div>
            </Col>
          ))}
        </Row>

        <SummaryPieGrid
          symbolStats={symbolStats}
          symbolPieData={symbolPieData}
          directionPieData={directionPieData}
          profitPieData={profitPieData}
        />

        <SummaryMetricsCards tradeStats={tradeStats} riskMetrics={riskMetrics} />

        {/* Economic calendar panel */}
        <div className="rounded-2xl p-6 mt-6" style={{ background: '#FFFFFF', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
          <h2 className="text-lg font-semibold mb-4" style={{ color: '#141D22' }}>{t('analytics.summary.cards.economicCalendar')}</h2>
          <Row gutter={16}>
            <Col xs={24} md={14}>
              {calendarEvents.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
              ) : (
                <div className="space-y-2 max-h-64 overflow-auto mt-2">
                  {calendarEvents.map((event, index) => {
                    const key = `${event.timestamp || ''}-${event.event || ''}-${event.country || ''}-${index}`;
                    const dtLabel = event.time ? `${event.date || ''} ${event.time}` : (event.date || '');
                    return (
                      <div key={key} className="flex justify-between gap-3 text-sm py-1 border-b border-gray-100 last:border-b-0">
                        <div className="flex-1 min-w-0">
                          <div className="font-medium truncate" style={{ color: '#141D22' }}>{event.localizedEvent || event.event || '-'}</div>
                          <div className="text-xs mt-1" style={{ color: '#8A9AA5' }}>{dtLabel}{event.country ? ` · ${event.country}` : ''}{event.impact ? ` · ${event.impact}` : ''}</div>
                        </div>
                        <div className="text-right text-xs" style={{ color: '#8A9AA5', minWidth: '120px' }}>
                          {event.actual && <div>{t('analytics.summary.economicCalendar.actual') || 'Actual'}: {event.actual}</div>}
                          {event.previous && <div>{t('analytics.summary.economicCalendar.previous') || 'Previous'}: {event.previous}</div>}
                          {event.estimate && <div>{t('analytics.summary.economicCalendar.estimate') || 'Estimate'}: {event.estimate}</div>}
                        </div>
                      </div>
                    );
                  })}
                </div>
              )}
            </Col>
            <Col xs={24} md={10}>
              <div className="mb-2 text-sm font-medium" style={{ color: '#141D22' }}>{t('analytics.summary.economicCalendar.keyIndicatorsTitle') || 'Key macro indicators'}</div>
              {keyIndicators.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
              ) : (
                <div className="space-y-3 max-h-64 overflow-auto mt-1">
                  {keyIndicators.map((ind) => {
                    const history = Array.isArray(ind.history) ? [...ind.history].reverse() : [];
                    return (
                      <div key={ind.code} className="text-xs p-1.5 rounded-lg" style={{ backgroundColor: '#F7F9FB' }}>
                        <div className="flex items-center justify-between mb-1">
                          <div className="font-medium truncate" style={{ color: '#141D22' }}>
                            {t(`analytics.summary.economicCalendar.indicators.${ind.code}`, { defaultValue: ind.name || ind.code })}
                          </div>
                          <div style={{ color: '#141D22' }}>{ind.latestValue?.toFixed ? ind.latestValue.toFixed(2) : ind.latestValue}{ind.units ? ` ${ind.units}` : ''}</div>
                        </div>
                        {history.length > 1 && (
                          <div style={{ height: 40 }}>
                            <ResponsiveContainer width="100%" height="100%">
                              <LineChart data={history}><Line type="monotone" dataKey="value" stroke="#D4AF37" strokeWidth={1.5} dot={false} /></LineChart>
                            </ResponsiveContainer>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </Col>
          </Row>
        </div>
      </StatusResult>
    </div>
  );
}
