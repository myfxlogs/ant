import { Tag, Typography } from 'antd';
import {
  BarChartOutlined, PieChartOutlined, LineChartOutlined,
  RiseOutlined, FallOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { ANALYTICS_EMPTY_SYMBOL_DISTRIBUTION_KEY, ANALYTICS_STATS_NET_PROFIT_KEY, ANALYTICS_STATS_TOTAL_TRADES_KEY, ANALYTICS_STATS_WIN_RATE_KEY, REPORT_DIRECTION_KEY, REPORT_DIRECTION_LONG_KEY, REPORT_DIRECTION_SHORT_KEY, REPORT_DRAWDOWN_EVENTS_KEY, REPORT_DRAWDOWN_OVERLAY_KEY, REPORT_RECOVERED_KEY, REPORT_SYMBOL_PN_L_KEY, REPORT_TRADE_DISTRIBUTION_KEY, REPORT_WIN_RATE_TREND_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import {
  Bar, CartesianGrid, ComposedChart, Cell, Line,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts';
import { StatusResult } from '@/components/common/StatusResult';
import MonthlyAnalysisCard from './MonthlyAnalysisCard';
import { HourlyDailyChart } from './HourlyDailyChart';
import type { MonthlyAnalysisPoint } from './MonthlyAnalysisCard.shared';
import type {
  AttributionAnalysisData, RollingMetricsData,
} from '@/client/analytics';

const { Title, Text } = Typography;

type Props = {
  accountId: string | undefined;
  currency: string;
  attribution: AttributionAnalysisData | undefined;
  rolling: RollingMetricsData | undefined;
  attributionLoading: boolean;
  attributionError: string | null;
  rollingLoading: boolean;
  rollingError: string | null;
  analyticsLoading: boolean;
  analyticsError: string | null;
  monthlyAnalysisLoading: boolean;
  monthlyAnalysisError: string | null;
  monthlyAnalysisYears: number[];
  monthlyAnalysisData: MonthlyAnalysisPoint[];
  monthlyWinRates: RollingMetricsData['monthlyWinRates'];
  hourlyData: { hourLabel: string }[];
  dailyPnLData: Record<string, unknown>[];
};

export default function ReportChartPanels({
  accountId, currency, attribution, rolling,
  attributionLoading, attributionError,
  rollingLoading, rollingError,
  analyticsLoading, analyticsError,
  monthlyAnalysisLoading, monthlyAnalysisError,
  monthlyAnalysisYears, monthlyAnalysisData, monthlyWinRates,
  hourlyData, dailyPnLData,
}: Props) {
  const { t } = useTranslation();

  return (
    <>
      {/* ═══ Attribution Charts ═══ */}
      <StatusResult loading={attributionLoading} error={attributionError ?? undefined}>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Symbol P&L ranking */}
          <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
            <Title level={5}><BarChartOutlined /> {t(REPORT_SYMBOL_PN_L_KEY)}</Title>
            {attribution?.symbolPnls?.length ? (() => {
              const data = attribution.symbolPnls.slice(0, 8);
              const maxSymbolLen = Math.max(...data.map((s: { symbol: string }) => s.symbol.length), 3);
              const yWidth = Math.min(Math.round(maxSymbolLen * 6.8 + 10), 90);
              const yFontSize = maxSymbolLen > 12 ? 8 : maxSymbolLen > 8 ? 9 : 10;
              return (
              <ResponsiveContainer width="100%" height={Math.max(data.length * 36, 140)}>
                <ComposedChart layout="vertical" data={data} barCategoryGap="8%">
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" horizontal={false} />
                  <XAxis type="number" stroke="var(--color-text-muted)" fontSize={11} />
                  <YAxis type="category" dataKey="symbol" width={yWidth} stroke="var(--color-text-muted)" fontSize={yFontSize} />
                  <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px var(--color-shadow)' }} />
                  <Bar dataKey="netProfit" radius={[0, 4, 4, 0]} maxBarSize={30} isAnimationActive={false} cursor="pointer">
                    {data.map((_: unknown, i: number) => (
                      <Cell key={i} fill={(data[i]?.netProfit ?? 0) >= 0 ? 'var(--color-success)' : 'var(--color-danger)'} />
                    ))}
                  </Bar>
                </ComposedChart>
              </ResponsiveContainer>
              );
            })() : <div className="flex items-center justify-center h-[280px]" style={{ color: 'var(--color-text-muted)' }}>{t(ANALYTICS_EMPTY_SYMBOL_DISTRIBUTION_KEY)}</div>}
          </div>

          {/* Direction breakdown */}
          <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
            <Title level={5}><PieChartOutlined /> {t(REPORT_DIRECTION_KEY)}</Title>
            {attribution?.direction ? (
              <div className="grid grid-cols-2 gap-4 mt-4">
                {[
                  { label: t(REPORT_DIRECTION_LONG_KEY), color: 'var(--color-success)', d: { profit: attribution.direction.longProfit ?? 0, trades: attribution.direction.longTrades ?? 0, winRate: attribution.direction.longWinRate ?? 0 } },
                  { label: t(REPORT_DIRECTION_SHORT_KEY), color: 'var(--color-danger)', d: { profit: attribution.direction.shortProfit ?? 0, trades: attribution.direction.shortTrades ?? 0, winRate: attribution.direction.shortWinRate ?? 0 } },
                ].map(({ label, color, d }) => (
                  <div key={label} className="rounded-xl p-4" style={{ border: '1px solid var(--color-border)' }}>
                    <Text strong style={{ color }}>{label}</Text>
                    <div className="mt-2 space-y-1" style={{ fontSize: 13, color: 'var(--color-text-secondary)' }}>
                      <div>{t(ANALYTICS_STATS_NET_PROFIT_KEY)}: <strong style={{ color: d.profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>{d.profit >= 0 ? '+' : ''}{d.profit.toFixed(2)}</strong></div>
                      <div>{t(ANALYTICS_STATS_TOTAL_TRADES_KEY)}: {d.trades}</div>
                      <div>{t(ANALYTICS_STATS_WIN_RATE_KEY)}: {d.winRate.toFixed(1)}%</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className="flex items-center justify-center h-[200px]" style={{ color: 'var(--color-text-muted)' }}>{t(ANALYTICS_EMPTY_SYMBOL_DISTRIBUTION_KEY)}</div>
            )}
          </div>

          {/* Trade profit distribution */}
          {attribution?.tradeDistribution?.profitBuckets?.length ? (
            <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
              <Title level={5}><BarChartOutlined /> {t(REPORT_TRADE_DISTRIBUTION_KEY)}</Title>
              <ResponsiveContainer width="100%" height={250}>
                <ComposedChart data={attribution.tradeDistribution.profitBuckets} barCategoryGap="10%">
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                  <XAxis dataKey="label" stroke="var(--color-text-muted)" fontSize={10} angle={-30} textAnchor="end" height={60} />
                  <YAxis stroke="var(--color-text-muted)" fontSize={11} />
                  <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px var(--color-shadow)' }} />
                  <Bar dataKey="count" fill="var(--color-info)" radius={[4, 4, 0, 0]} maxBarSize={50} isAnimationActive={false} cursor="pointer" />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          ) : null}
        </div>
      </StatusResult>

      {/* ═══ Monthly Analysis + Hourly/Daily ═══ */}
      <StatusResult loading={analyticsLoading || monthlyAnalysisLoading} error={(analyticsError || monthlyAnalysisError) ?? undefined}>
        {monthlyAnalysisYears.length > 0 && (
          <MonthlyAnalysisCard
            accountId={accountId}
            years={monthlyAnalysisYears}
            data={monthlyAnalysisData}
            winRateData={monthlyWinRates}
            currency={currency}
          />
        )}
        {hourlyData.length > 0 && (
          <HourlyDailyChart hourlyData={hourlyData} dailyPnLData={dailyPnLData} currency={currency || 'USD'} />
        )}
      </StatusResult>

      {/* ═══ Rolling Metrics ═══ */}
      <StatusResult loading={rollingLoading} error={rollingError ?? undefined}>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-6">
          {/* Equity + Drawdown overlay */}
          {rolling?.equityCurve?.length ? (
            <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
              <Title level={5}><LineChartOutlined /> {t(REPORT_DRAWDOWN_OVERLAY_KEY)}</Title>
              <ResponsiveContainer width="100%" height={280}>
                <ComposedChart data={rolling.equityCurve.map((p, i) => ({
                  ...p,
                  drawdown: rolling.drawdownCurve?.[i]?.drawdownPercent ?? 0,
                }))}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                  <XAxis dataKey="date" stroke="var(--color-text-muted)" fontSize={10} />
                  <YAxis yAxisId="left" stroke="var(--color-text-muted)" fontSize={11} />
                  <YAxis yAxisId="right" orientation="right" stroke="var(--color-danger)" fontSize={11} domain={[100, 0]} />
                  <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px var(--color-shadow)' }} />
                  <Line yAxisId="left" type="monotone" dataKey="equity" stroke="var(--color-info)" strokeWidth={2} dot={false} name="Equity" isAnimationActive={false} />
                  <Line yAxisId="right" type="monotone" dataKey="drawdown" stroke="var(--color-danger)" strokeWidth={1} dot={false} name="DD%" isAnimationActive={false} fillOpacity={0.1} fill="var(--color-danger)" />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          ) : null}

          {/* Drawdown events */}
          {rolling?.drawdownEvents?.length ? (
            <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
              <Title level={5}><FallOutlined /> {t(REPORT_DRAWDOWN_EVENTS_KEY)}</Title>
              <div className="space-y-2 max-h-[280px] overflow-y-auto">
                {rolling.drawdownEvents.map((e, i) => (
                  <div key={i} className="flex items-center justify-between rounded-lg p-3" style={{ border: '1px solid var(--color-border)' }}>
                    <div>
                      <Text style={{ color: 'var(--color-text)', fontSize: 13 }}>{e.startDate} → {e.endDate || '...'}</Text>
                      {e.recoveryDate && <Text style={{ color: 'var(--color-text-muted)', fontSize: 12, marginLeft: 8 }}>{t(REPORT_RECOVERED_KEY)}: {e.recoveryDate}</Text>}
                    </div>
                    <Tag color="error">{(e.depthPercent ?? 0).toFixed(1)}%</Tag>
                  </div>
                ))}
              </div>
            </div>
          ) : null}

          {/* Monthly win rate trend */}
          {rolling?.monthlyWinRates?.length ? (
            <div className="rounded-2xl p-5" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
              <Title level={5}><RiseOutlined /> {t(REPORT_WIN_RATE_TREND_KEY)}</Title>
              <ResponsiveContainer width="100%" height={250}>
                <ComposedChart data={rolling.monthlyWinRates}>
                  <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" />
                  <XAxis dataKey="month" stroke="var(--color-text-muted)" fontSize={10} />
                  <YAxis stroke="var(--color-text-muted)" fontSize={11} domain={[0, 100]} />
                  <Tooltip contentStyle={{ background: 'var(--color-bg-card)', border: 'none', borderRadius: '8px', boxShadow: '0 4px 12px var(--color-shadow)' }} />
                  <Line type="monotone" dataKey="winRate" stroke="var(--color-success)" strokeWidth={2} dot isAnimationActive={false} />
                </ComposedChart>
              </ResponsiveContainer>
            </div>
          ) : null}
        </div>
      </StatusResult>
    </>
  );
}
