
import { ANALYTICS_EMPTY_MONTHLY_PROFIT_KEY, ANALYTICS_MONTHLY_DETAIL_HOLDING_TITLE_KEY, ANALYTICS_MONTHLY_DETAIL_LONG_KEY, ANALYTICS_MONTHLY_DETAIL_POPULARITY_TITLE_KEY, ANALYTICS_MONTHLY_DETAIL_RISK_REWARD_TITLE_KEY, ANALYTICS_MONTHLY_DETAIL_SHORT_KEY } from '@/gen/ant/v1/i18n/accounts_keys';
import React, { useMemo } from 'react'
;
import {
  Bar, BarChart, CartesianGrid, Cell, LabelList, Legend, Pie, PieChart,
  ResponsiveContainer, Tooltip, XAxis, YAxis,
} from 'recharts';
import { PIE_COLORS } from './MonthlyAnalysisCard.shared';

const sectionStyle = {
  background: 'var(--color-bg-secondary)',
  border: '1px solid var(--color-border)',
  borderRadius: 8,
  padding: 10,
} as const;

const titleStyle = {
  color: 'var(--color-text)',
  fontSize: 12,
  fontWeight: 600,
  marginBottom: 6,
} as const;

const tooltipStyle = {
  background: 'var(--color-bg-card)',
  border: 'none',
  borderRadius: 6,
  boxShadow: '0 2px 8px var(--color-shadow)',
  fontSize: 11,
};

// Adaptive YAxis: width & font size scale with the longest symbol name.
const useAdaptiveYAxis = (symbols: string[]) =>
  useMemo(() => {
    const maxLen = Math.max(...symbols.map((s) => s.length), 3);
    // ~6.5px per char at fontSize 10, plus tick mark + padding
    const width = Math.min(Math.round(maxLen * 6.8 + 10), 90);
    const fontSize = maxLen > 12 ? 8 : maxLen > 8 ? 9 : 10;
    return { width, fontSize };
  }, [symbols]);

/* ── Panel 1: Risk/Reward ratio per symbol (horizontal bar chart) ── */
const RiskRewardPanel = React.memo(({ risks, t }: {
  risks: Array<{ symbol: string; riskRatio: number }>;
  t: (k: string) => string;
}) => {
  const chartData = useMemo(() => {
    if (!risks.length) return [];
    return risks.slice(0, 10).map((r) => ({
      symbol: r.symbol,
      riskRatio: Math.min(r.riskRatio, 5), // clamp visual, raw value in tooltip
      rawRatio: r.riskRatio,
    }));
  }, [risks]);

  const symbols = useMemo(() => chartData.map((d) => d.symbol), [chartData]);
  const yAxis = useAdaptiveYAxis(symbols);

  if (!chartData.length) {
    return (
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_RISK_REWARD_TITLE_KEY)}</h4>
        <div className="flex items-center justify-center h-[100px]" style={{ color: 'var(--color-text-muted)', fontSize: 11 }}>
          {t(ANALYTICS_EMPTY_MONTHLY_PROFIT_KEY)}
        </div>
      </div>
    );
  }

  return (
    <div style={sectionStyle}>
      <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_RISK_REWARD_TITLE_KEY)}</h4>
      <ResponsiveContainer width="100%" height={Math.max(chartData.length * 36, 140)}>
        <BarChart layout="vertical" data={chartData} margin={{ top: 0, right: 28, bottom: 0, left: 0 }} barCategoryGap="8%">
          <CartesianGrid strokeDasharray="2 2" stroke="var(--color-border)" horizontal={false} />
          <XAxis type="number" stroke="var(--color-text-muted)" fontSize={9}
            tickFormatter={(v) => v.toFixed(1)} />
          <YAxis type="category" dataKey="symbol" width={yAxis.width} stroke="var(--color-text-muted)" fontSize={yAxis.fontSize} />
          <Tooltip contentStyle={tooltipStyle}
            formatter={(_v: number, _n: string, props: { payload?: { rawRatio?: number } }) =>
              [`${props?.payload?.rawRatio?.toFixed(2) ?? '—'}`, 'Reward:Risk']
            } />
          <Bar dataKey="riskRatio" radius={[0, 3, 3, 0]} maxBarSize={30} cursor="pointer" isAnimationActive={false}>
            <LabelList
              dataKey="rawRatio"
              position="right"
              formatter={(v: number) => Number.isFinite(v) ? v.toFixed(2) : ''}
              style={{ fontSize: 10, fill: 'var(--color-text-muted)', fontWeight: 500 }}
            />
            {chartData.map((_, i) => (
              <Cell key={i} fill="rgba(119, 189, 243, 0.7)" />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
});

/* ── Panel 2: Currency/Symbol Popularity pie chart ── */
const PopularityPanel = React.memo(({ popularity, t }: {
  popularity: Array<{ symbol: string; trades: number; sharePercent: number }>;
  t: (k: string) => string;
}) => {
  const pieData = useMemo(() => {
    if (!popularity.length) return [];
    return popularity.map((s, i) => ({
      name: s.symbol,
      value: s.sharePercent,
      trades: s.trades,
      fill: PIE_COLORS[i % PIE_COLORS.length],
    }));
  }, [popularity]);

  // Adaptive dimensions: more symbols → taller chart + proportionally larger pie.
  const { pieHeight, innerR, outerR } = useMemo(() => {
    const n = Math.max(pieData.length, 1);
    const h = Math.min(Math.round(120 + n * 32), 240);
    const or = Math.round(h * 0.38);
    const ir = Math.round(or * 0.55);
    return { pieHeight: h, innerR: ir, outerR: or };
  }, [pieData.length]);

  if (!pieData.length) {
    return (
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_POPULARITY_TITLE_KEY)}</h4>
        <div className="flex items-center justify-center h-[140px]" style={{ color: 'var(--color-text-muted)', fontSize: 11 }}>
          {t(ANALYTICS_EMPTY_MONTHLY_PROFIT_KEY)}
        </div>
      </div>
    );
  }

  return (
    <div style={sectionStyle}>
      <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_POPULARITY_TITLE_KEY)}</h4>
      <ResponsiveContainer width="100%" height={pieHeight}>
        <PieChart>
          <Pie
            data={pieData}
            cx="50%"
            cy="50%"
            innerRadius={innerR}
            outerRadius={outerR}
            paddingAngle={2}
            dataKey="value"
            isAnimationActive={false}
            label={({ name, value }) => `${name} ${value.toFixed(1)}%`}
            labelLine={{ stroke: 'var(--color-text-muted)', strokeWidth: 1 }}
          >
            {pieData.map((d, i) => (
              <Cell key={i} fill={d.fill} stroke="#fff" strokeWidth={1} />
            ))}
          </Pie>
          <Tooltip contentStyle={tooltipStyle}
            formatter={(v: number, _n: string, props: { payload?: { trades?: number } }) =>
              [`${v.toFixed(1)}% (${props?.payload?.trades ?? 0} trades)`, '']
            } />
        </PieChart>
      </ResponsiveContainer>
    </div>
  );
});

/* ── Panel 3: Holding time split — bulls (long) vs short (horizontal bars) ── */
const HoldingSplitPanel = React.memo(({ holdingSplit, t }: {
  holdingSplit: Array<{ symbol: string; bullsSeconds: number; shortTermSeconds: number }>;
  t: (k: string) => string;
}) => {
  // Transform to stacked-data shape for grouped horizontal bars
  const chartData = useMemo(() => {
    if (!holdingSplit.length) return [];
    // myfxbook uses milliseconds in the chart — convert seconds to ms
    return holdingSplit.slice(0, 10).map((h) => ({
      symbol: h.symbol,
      long: h.bullsSeconds * 1000,
      short: h.shortTermSeconds * 1000,
    }));
  }, [holdingSplit]);

  if (!chartData.length) {
    return (
      <div style={sectionStyle}>
        <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_HOLDING_TITLE_KEY)}</h4>
        <div className="flex items-center justify-center h-[100px]" style={{ color: 'var(--color-text-muted)', fontSize: 11 }}>
          {t(ANALYTICS_EMPTY_MONTHLY_PROFIT_KEY)}
        </div>
      </div>
    );
  }

  const maxVal = Math.max(...chartData.flatMap((d) => [d.long, d.short]), 1);
  const holdingSymbols = useMemo(() => chartData.map((d) => d.symbol), [chartData]);
  const yAxis = useAdaptiveYAxis(holdingSymbols);

  // Format milliseconds to human-readable
  const fmtMs = (ms: number): string => {
    if (ms <= 0) return '0';
    const sec = ms / 1000;
    if (sec < 60) return `${Math.round(sec)}s`;
    if (sec < 3600) return `${(sec / 60).toFixed(1)}m`;
    if (sec < 86400) return `${(sec / 3600).toFixed(1)}h`;
    return `${(sec / 86400).toFixed(1)}d`;
  };

  return (
    <div style={sectionStyle}>
      <h4 style={titleStyle}>{t(ANALYTICS_MONTHLY_DETAIL_HOLDING_TITLE_KEY)}</h4>
      <ResponsiveContainer width="100%" height={Math.max(chartData.length * 44, 150)}>
        <BarChart layout="vertical" data={chartData} margin={{ top: 0, right: 0, bottom: 0, left: 0 }}
          barGap={1} barCategoryGap="6%">
          <CartesianGrid strokeDasharray="2 2" stroke="var(--color-border)" horizontal={false} />
          <XAxis type="number" stroke="var(--color-text-muted)" fontSize={9}
            domain={[0, maxVal * 1.15]}
            tickFormatter={fmtMs} />
          <YAxis type="category" dataKey="symbol" width={yAxis.width} stroke="var(--color-text-muted)" fontSize={yAxis.fontSize} />
          <Tooltip contentStyle={tooltipStyle}
            formatter={(v: number) => [fmtMs(Number(v)), '']} />
          <Bar dataKey="long" radius={[0, 3, 3, 0]} maxBarSize={22} cursor="pointer" isAnimationActive={false}
            fill="rgba(83, 243, 2, 0.7)" name={t(ANALYTICS_MONTHLY_DETAIL_LONG_KEY)} />
          <Bar dataKey="short" radius={[0, 3, 3, 0]} maxBarSize={22} cursor="pointer" isAnimationActive={false}
            fill="rgba(255, 0, 0, 0.7)" name={t(ANALYTICS_MONTHLY_DETAIL_SHORT_KEY)} />
          <Legend
            wrapperStyle={{ fontSize: 10, color: 'var(--color-text-muted)', paddingTop: 4 }}
            iconSize={10}
          />
        </BarChart>
      </ResponsiveContainer>
    </div>
  );
});

/* ── Named exports — caller lays out panels in myfxbook grid ── */
export { RiskRewardPanel, PopularityPanel, HoldingSplitPanel };
