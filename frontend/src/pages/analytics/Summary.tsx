import { useCallback, useEffect, useState } from 'react';
import { Card, Select, Row, Col, Statistic, Spin, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  BarChart, Bar, PieChart, Pie, Cell, Legend,
} from 'recharts';
import {
  IconTrendingUp,
  IconChartLine,
  IconChartPie,
  IconClock,
  IconTarget,
} from '@tabler/icons-react';
import { useAccount } from '@/hooks/useAccount';
import { analyticsApi } from '@/client/analytics';
import { periodOptions } from './Summary.constants';
import {
  getDirectionPieData,
  getEquityCurveData,
  getMonthlyData,
  getProfitPieData,
  getSymbolPieData,
  getYearOptions,
} from './Summary.helpers';
import { formatHoldingTime } from '@/utils/date';

export default function Summary() {
  const { t, i18n } = useTranslation();
  const { accounts, fetchAccounts } = useAccount();
  const [selectedAccount, setSelectedAccount] = useState<string | null>(null);
  const [selectedPeriod, setSelectedPeriod] = useState('month');
  const [selectedYear, setSelectedYear] = useState(new Date().getFullYear());
  const [analytics, setAnalytics] = useState<any>(null);
  const [loading, setLoading] = useState(false);
  const [calendarEvents, setCalendarEvents] = useState<any[]>([]);
  const [calendarLoading, setCalendarLoading] = useState(false);
  const [keyIndicators, setKeyIndicators] = useState<any[]>([]);
  const [keyIndicatorsLoading, setKeyIndicatorsLoading] = useState(false);

  useEffect(() => {
    fetchAccounts();
  }, [fetchAccounts]);

  useEffect(() => {
    if ((accounts || []).length > 0 && !selectedAccount) {
      const firstAccount = accounts[0];
      if (firstAccount && firstAccount.id) {
        setSelectedAccount(firstAccount.id);
      }
    }
  }, [accounts, selectedAccount]);

  const fetchAnalytics = useCallback(async () => {
    if (!selectedAccount) return;
    setLoading(true);
    try {
      const [accountAnalytics, monthlyPnL] = await Promise.all([
        analyticsApi.getAccountAnalytics(selectedAccount),
        analyticsApi.getMonthlyPnL(selectedAccount, selectedYear),
      ]);
      setAnalytics({
        ...(accountAnalytics as any),
        monthlyPnl: (monthlyPnL as any)?.monthlyPnl || (accountAnalytics as any)?.monthlyPnl || [],
      });
    } catch (_error) {
      // 错误处理 - 保持默认值
      setAnalytics(null);
    } finally {
      setLoading(false);
    }
  }, [selectedAccount, selectedYear]);

  useEffect(() => {
    if (selectedAccount) {
      fetchAnalytics();
    }
  }, [selectedAccount, selectedPeriod, fetchAnalytics]);

  useEffect(() => {
    let cancelled = false;
    const loadCalendar = async () => {
      setCalendarLoading(true);
      try {
        const events = await analyticsApi.getEconomicCalendar();
        if (!cancelled) {
          setCalendarEvents(Array.isArray(events) ? events.slice(0, 50) : []);
        }
      } catch (_error) {
        if (!cancelled) {
          setCalendarEvents([]);
        }
      } finally {
        if (!cancelled) {
          setCalendarLoading(false);
        }
      }
    };
    const loadIndicators = async () => {
      setKeyIndicatorsLoading(true);
      try {
        const indicators = await analyticsApi.getEconomicIndicators();
        if (!cancelled) {
          setKeyIndicators(Array.isArray(indicators) ? indicators : []);
        }
      } catch (_error) {
        if (!cancelled) {
          setKeyIndicators([]);
        }
      } finally {
        if (!cancelled) {
          setKeyIndicatorsLoading(false);
        }
      }
    };
    void loadCalendar();
    void loadIndicators();
    return () => {
      cancelled = true;
    };
  }, [i18n.language]);

  const tradeStats = (analytics as any)?.tradeStats || null;
  const riskMetrics = (analytics as any)?.riskMetrics || null;
  const symbolStats = (analytics as any)?.symbolStats || [];

  const equityCurveData = getEquityCurveData((analytics as any)?.equityCurve || []);
  const monthlyData = getMonthlyData((analytics as any)?.monthlyPnl || []);
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
          <Select
            value={selectedAccount}
            onChange={setSelectedAccount}
            style={{ width: 200 }}
            placeholder={t('analytics.summary.placeholders.selectAccount')}
          >
            {(accounts || []).map(a => (
              <Select.Option key={a.id} value={a.id}>
                {a.alias}
              </Select.Option>
            ))}
          </Select>
          <Select
            value={selectedPeriod}
            onChange={setSelectedPeriod}
            options={periodOptions(t)}
            style={{ width: 120 }}
          />
        </Space>
      </div>

      <Spin spinning={loading}>
        <div
          className="rounded-2xl p-6"
          style={{
            background: '#FFFFFF',
            boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
          }}
        >
          <h2 className="text-lg font-semibold mb-4" style={{ color: '#141D22' }}>{t('analytics.summary.sections.equityCurve')}</h2>
          <ResponsiveContainer width="100%" height={250}>
            <LineChart data={equityCurveData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
              <XAxis dataKey="date" stroke="#8A9AA5" fontSize={12} />
              <YAxis stroke="#8A9AA5" fontSize={12} />
              <Tooltip
                contentStyle={{
                  background: '#FFFFFF',
                  border: '1px solid rgba(0, 0, 0, 0.1)',
                  borderRadius: '8px',
                }}
              />
              <Line
                type="monotone"
                dataKey="equity"
                stroke="#D4AF37"
                strokeWidth={2}
                dot={false}
              />
            </LineChart>
          </ResponsiveContainer>
        </div>

        <div
          className="rounded-2xl p-6 mt-6"
          style={{
            background: '#FFFFFF',
            boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
          }}
        >
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold" style={{ color: '#141D22' }}>{t('analytics.summary.sections.monthlyStats')}</h2>
            <Select
              value={selectedYear}
              onChange={setSelectedYear}
              options={yearOptions}
              style={{ width: 100 }}
            />
          </div>
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={monthlyData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
              <XAxis dataKey="month" stroke="#8A9AA5" fontSize={12} />
              <YAxis stroke="#8A9AA5" fontSize={12} />
              <Tooltip
                contentStyle={{
                  background: '#FFFFFF',
                  border: '1px solid rgba(0, 0, 0, 0.1)',
                  borderRadius: '8px',
                }}
                formatter={(value: number | undefined) => [`$${(value || 0).toFixed(2)}`, t('analytics.summary.labels.pnl')]}
              />
              <Bar
                dataKey="profit"
                fill="#D4AF37"
                radius={[4, 4, 0, 0]}
              />
            </BarChart>
          </ResponsiveContainer>
        </div>

        <Row gutter={[16, 16]} className="mt-6">
          <Col xs={12} sm={6}>
            <div className="stat-card">
              <div className="flex items-center gap-2 mb-2">
                <IconTrendingUp size={18} stroke={1.5} color="#00A651" />
                <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.metrics.netProfit')}</span>
              </div>
              <div className="text-2xl font-semibold" style={{ color: (tradeStats?.netProfit || 0) >= 0 ? '#00A651' : '#E53935' }}>
                ${(Number((analytics as any)?.profit || tradeStats?.netProfit || 0)).toFixed(2)}
              </div>
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card">
              <div className="flex items-center gap-2 mb-2">
                <IconChartLine size={18} stroke={1.5} color="#2196F3" />
                <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.metrics.equity')}</span>
              </div>
              <div className="text-2xl font-semibold" style={{ color: '#141D22' }}>
                ${(Number((analytics as any)?.equity || 0)).toFixed(2)}
              </div>
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card">
              <div className="flex items-center gap-2 mb-2">
                <IconTarget size={18} stroke={1.5} color="#D4AF37" />
                <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.metrics.balance')}</span>
              </div>
              <div className="text-2xl font-semibold" style={{ color: '#141D22' }}>
                ${(Number((analytics as any)?.balance || 0)).toFixed(2)}
              </div>
            </div>
          </Col>
          <Col xs={12} sm={6}>
            <div className="stat-card">
              <div className="flex items-center gap-2 mb-2">
                <IconChartPie size={18} stroke={1.5} color="#9C27B0" />
                <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.metrics.equityValue')}</span>
              </div>
              <div className="text-2xl font-semibold" style={{ color: '#141D22' }}>
                ${(Number((analytics as any)?.equity || 0)).toFixed(2)}
              </div>
            </div>
          </Col>
        </Row>

        <Row gutter={[16, 16]} className="mt-6">
          <Col xs={24} lg={12}>
            <Card
              title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.symbolPnlCompare')}</span>}
              className="glass-card"
            >
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={(symbolStats || []).slice(0, 5)} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
                  <XAxis type="number" stroke="#8A9AA5" fontSize={12} />
                  <YAxis dataKey="symbol" type="category" stroke="#8A9AA5" fontSize={12} width={60} />
                  <Tooltip
                    contentStyle={{
                      background: '#FFFFFF',
                      border: '1px solid rgba(0, 0, 0, 0.1)',
                      borderRadius: '8px',
                    }}
                    formatter={(value: number | undefined) => [`$${(value || 0).toFixed(2)}`, t('analytics.summary.labels.pnl')]}
                  />
                  <Bar dataKey="profit" fill="#D4AF37" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card
              title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.symbolTradeShare')}</span>}
              className="glass-card"
            >
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={symbolPieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                  >
                    {symbolPieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>
        </Row>

        <Row gutter={[16, 16]} className="mt-6">
          <Col xs={24} lg={12}>
            <Card
              title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.directionShare')}</span>}
              className="glass-card"
            >
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={directionPieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                  >
                    {directionPieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>

          <Col xs={24} lg={12}>
            <Card
              title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.pnlShare')}</span>}
              className="glass-card"
            >
              <ResponsiveContainer width="100%" height={200}>
                <PieChart>
                  <Pie
                    data={profitPieData}
                    cx="50%"
                    cy="50%"
                    innerRadius={50}
                    outerRadius={80}
                    paddingAngle={2}
                    dataKey="value"
                  >
                    {profitPieData.map((entry, index) => (
                      <Cell key={`cell-${index}`} fill={entry.color} />
                    ))}
                  </Pie>
                  <Tooltip />
                  <Legend />
                </PieChart>
              </ResponsiveContainer>
            </Card>
          </Col>
        </Row>

        <Card
          title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.tradeStats')}</span>}
          className="glass-card mt-6"
        >
          <Row gutter={[16, 16]}>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.totalTrades')}</span>}
                value={tradeStats?.totalTrades || 0}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.wins')}</span>}
                value={tradeStats?.winningTrades || 0}
                suffix={<span style={{ color: '#8A9AA5', fontSize: '14px' }}> ({tradeStats?.winRate?.toFixed(0) || 0}%)</span>}
                valueStyle={{ color: '#00A651', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.losses')}</span>}
                value={tradeStats?.losingTrades || 0}
                valueStyle={{ color: '#E53935', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.winRate')}</span>}
                value={tradeStats?.winRate || 0}
                suffix="%"
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.profitFactor')}</span>}
                value={tradeStats?.profitFactor || 0}
                precision={2}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={24} sm={12} md={6}>
              <div className="flex items-center gap-2">
                <IconClock size={16} stroke={1.5} color="#8A9AA5" />
                <span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.tradeStats.avgHolding')}</span>
              </div>
              <div className="text-lg font-semibold mt-1" style={{ color: '#141D22' }}>{formatHoldingTime(tradeStats?.averageHoldingTime) || '-'}</div>
            </Col>
          </Row>
          <Row gutter={[16, 16]} className="mt-4">
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxConsecutiveWins')}</span>}
                value={tradeStats?.maxConsecutiveWins || 0}
                valueStyle={{ color: '#00A651', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxConsecutiveLosses')}</span>}
                value={tradeStats?.maxConsecutiveLosses || 0}
                valueStyle={{ color: '#E53935', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxHolding')}</span>}
                value={'-'}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgVolume')}</span>}
                value={tradeStats?.averageVolume || 0}
                precision={2}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgProfit')}</span>}
                value={tradeStats?.averageProfit || 0}
                prefix="$"
                valueStyle={{ color: '#00A651', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={8} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgLoss')}</span>}
                value={tradeStats?.averageLoss || 0}
                prefix="$"
                precision={2}
                valueStyle={{ color: '#E53935', fontSize: '20px' }}
              />
            </Col>
          </Row>
        </Card>

        <Card
          title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.riskMetrics')}</span>}
          className="glass-card mt-6"
        >
          <Row gutter={16}>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.maxDrawdown')}</span>}
                value={Math.abs(riskMetrics?.maxDrawdown || 0)}
                precision={2}
                prefix="$"
                valueStyle={{ color: '#E53935', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.maxDrawdownPct')}</span>}
                value={Math.abs(riskMetrics?.maxDrawdownPercent || 0)}
                precision={2}
                suffix="%"
                valueStyle={{ color: '#E53935', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.sharpe')}</span>}
                value={riskMetrics?.sharpeRatio || 0}
                precision={2}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.sortino')}</span>}
                value={riskMetrics?.sortinoRatio || 0}
                precision={2}
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.volatility')}</span>}
                value={riskMetrics?.volatility || 0}
                precision={2}
                suffix="%"
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Statistic
                title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.var95')}</span>}
                value={riskMetrics?.valueAtRisk || 0}
                precision={2}
                prefix="$"
                valueStyle={{ color: '#141D22', fontSize: '20px' }}
              />
            </Col>
          </Row>
        </Card>

        <Card
          title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.economicCalendar')}</span>}
          className="glass-card mt-6"
        >
          <Row gutter={16}>
            <Col xs={24} md={14}>
              {calendarLoading && calendarEvents.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.loading') || 'Loading economic calendar...'}</div>
              ) : null}
              {!calendarLoading && calendarEvents.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
              ) : null}
              {calendarEvents.length > 0 ? (
                <div className="space-y-2 max-h-64 overflow-auto mt-2">
                  {calendarEvents.map((event, index) => {
                    const key = `${event.timestamp || ''}-${event.event || ''}-${event.country || ''}-${index}`;
                    const dateStr = event.date || '';
                    const timeStr = event.time || '';
                    const dtLabel = timeStr ? `${dateStr} ${timeStr}` : dateStr;
                    return (
                      <div key={key} className="flex justify-between gap-3 text-sm py-1 border-b border-gray-100 last:border-b-0">
                        <div className="flex-1 min-w-0">
                          <div className="font-medium truncate" style={{ color: '#141D22' }}>{event.localizedEvent || event.event || '-'}</div>
                          <div className="text-xs mt-1" style={{ color: '#8A9AA5' }}>
                            {dtLabel}
                            {event.country ? ` · ${event.country}` : ''}
                            {event.impact ? ` · ${event.impact}` : ''}
                          </div>
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
              ) : null}
            </Col>
            <Col xs={24} md={10}>
              <div className="mb-2 text-sm font-medium" style={{ color: '#141D22' }}>
                {t('analytics.summary.economicCalendar.keyIndicatorsTitle') || 'Key macro indicators'}
              </div>
              {keyIndicatorsLoading && keyIndicators.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.loading') || 'Loading economic calendar...'}</div>
              ) : null}
              {!keyIndicatorsLoading && keyIndicators.length === 0 ? (
                <div style={{ color: '#8A9AA5' }}>{t('analytics.summary.economicCalendar.empty') || 'No economic events available.'}</div>
              ) : null}
              {keyIndicators.length > 0 ? (
                <div className="space-y-3 max-h-64 overflow-auto mt-1">
                  {keyIndicators.map((ind: any) => {
                    const history = Array.isArray(ind.history) ? [...ind.history].reverse() : [];
                    return (
                      <div key={ind.code} className="text-xs p-1.5 rounded-lg" style={{ backgroundColor: '#F7F9FB' }}>
                        <div className="flex items-center justify-between mb-1">
                          <div className="font-medium truncate" style={{ color: '#141D22' }}>
                            {t(`analytics.summary.economicCalendar.indicators.${ind.code}`, { defaultValue: ind.name || ind.code })}
                          </div>
                          <div style={{ color: '#141D22' }}>
                            {ind.latestValue?.toFixed ? ind.latestValue.toFixed(2) : ind.latestValue}
                            {ind.units ? ` ${ind.units}` : ''}
                          </div>
                        </div>
                        {history.length > 1 ? (
                          <div style={{ height: 40 }}>
                            <ResponsiveContainer width="100%" height="100%">
                              <LineChart data={history}>
                                <Line
                                  type="monotone"
                                  dataKey="value"
                                  stroke="#D4AF37"
                                  strokeWidth={1.5}
                                  dot={false}
                                />
                              </LineChart>
                            </ResponsiveContainer>
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              ) : null}
            </Col>
          </Row>
        </Card>
      </Spin>
    </div>
  );
}
