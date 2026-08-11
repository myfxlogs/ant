import type { MouseEvent as ReactMouseEvent } from 'react';
import type { XAxisTickContentProps } from 'recharts';
import {
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  LabelList,
  ReferenceArea,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { barCellFill, type MonthlyBarRow } from './MonthlyAnalysisCard.shared';

type MetricType = 'change' | 'profit' | 'lots' | 'pips';

type Props = {
  series: MonthlyBarRow[];
  selectedMetric: MetricType;
  currency: string;
  monthLabels: string[];
  yAxisFormatter: (v: number) => string;
  onMouseDown: (e: ReactMouseEvent) => void;
  onMouseMove: (activeTooltipIndex: number | string | undefined) => void;
  onMouseLeave: () => void;
  onCommitByTooltipIndex: (activeTooltipIndex: number | string | undefined) => void;
  onCommitMonthClick: (data: unknown, index: number) => void;
};

function formatTooltipValue(metric: MetricType, point: MonthlyBarRow, currency: string): string {
  switch (metric) {
    case 'change':
      return `${point.change >= 0 ? '+' : ''}${point.change.toFixed(2)}`;
    case 'profit':
      return `${point.profit >= 0 ? '+' : ''}${point.profit.toFixed(2)} ${currency}`;
    case 'lots':
      return `${point.lots.toFixed(2)} lots`;
    case 'pips':
      return `${point.pips >= 0 ? '+' : ''}${point.pips.toFixed(1)} pips`;
    default:
      return '';
  }
}

export default function MonthlyAnalysisMainChart({
  series,
  selectedMetric,
  currency,
  monthLabels,
  yAxisFormatter,
  onMouseDown,
  onMouseMove,
  onMouseLeave,
  onCommitByTooltipIndex,
  onCommitMonthClick,
}: Props) {
  const maxVal = Math.max(...series.map(s => Math.abs(s.value ?? 0)), 1);
  const chartHeight = Math.max(210, Math.min(500, maxVal * 2 + 80));
  return (
    <div
      className="outline-none [&_.recharts-wrapper]:!outline-none [&_.recharts-wrapper]:ring-0 [&_.recharts-surface]:outline-none"
      onMouseDown={onMouseDown}
    >
      <ResponsiveContainer width="100%" height={chartHeight}>
        <ComposedChart
          data={series}
          margin={{ top: 20, right: 6, left: 0, bottom: 4 }}
          onMouseMove={(state) => onMouseMove(Number(state.activeTooltipIndex))}
          onMouseLeave={onMouseLeave}
          onClick={(state) => onCommitByTooltipIndex(Number(state.activeTooltipIndex ?? state.activeIndex))}
        >
          {/* Alternating month background — myfxbook signature */}
          {series.map((item, i) => {
            if (i % 2 !== 1) return null;
            const next = series[i + 1];
            return (
              <ReferenceArea
                key={`bg-${i}`}
                x1={item.monthAxisLabel}
                x2={next?.monthAxisLabel}
                fill="var(--color-bg-secondary)"
                fillOpacity={0.4}
                ifOverflow="hidden"
              />
            );
          })}
          <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" vertical={false} />
          {/* Zero line */}
          <ReferenceLine y={0} stroke="var(--color-text-muted)" strokeOpacity={0.5} />
          <XAxis
            dataKey="monthAxisLabel"
            stroke="var(--color-text-muted)"
            fontSize={11}
            tickLine={false}
            axisLine={{ stroke: 'var(--color-border)' }}
            interval={0}
            angle={-30}
            textAnchor="end"
            height={42}
            tick={({ x, y, payload, index }: XAxisTickContentProps) => (
              <g transform={`translate(${x},${y})`} style={{ cursor: 'pointer' }}
                onClick={() => onCommitMonthClick(series[index], index)}>
                <text x={0} y={0} dy={4} textAnchor="end" fill="var(--color-text-muted)" fontSize={11}
                  transform="rotate(-30)" fontWeight={series[index]?.isActive ? 700 : 400}>
                  {payload.value}
                </text>
              </g>
            )}
          />
          <YAxis
            stroke="var(--color-text-muted)"
            fontSize={10}
            tickLine={false}
            axisLine={false}
            tickFormatter={yAxisFormatter}
            width={52}
          />
          <Tooltip
            wrapperStyle={{ pointerEvents: 'none' }}
            cursor={false}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              const row = payload[0]?.payload as MonthlyBarRow | undefined;
              if (!row) return null;
              return (
                <div
                  style={{
                    background: 'var(--color-bg-card)',
                    border: '1px solid var(--color-border)',
                    borderRadius: 6,
                    boxShadow: '0 2px 8px var(--color-shadow)',
                    padding: '8px 12px',
                    fontSize: 12,
                    pointerEvents: 'none',
                    lineHeight: 1.6,
                  }}
                >
                  <div style={{ fontWeight: 700, color: 'var(--color-text)', marginBottom: 2 }}>
                    {monthLabels[(row.month || 1) - 1]} {row.year}
                  </div>
                  <div style={{ color: 'var(--color-text-secondary)' }}>
                    {formatTooltipValue(selectedMetric, row, currency)}
                  </div>
                </div>
              );
            }}
          />
          <Bar
            dataKey="value"
            barSize={28}
            maxBarSize={36}
            radius={[2, 2, 0, 0]}
            minPointSize={10}
            isAnimationActive={false}
            style={{ cursor: 'pointer' }}
            onClick={onCommitMonthClick}
          >
            <LabelList
              dataKey="value"
              position="top"
              formatter={(v: string | number | boolean | null | undefined) => {
                const n = Number(v);
                if (!Number.isFinite(n) || Math.abs(n) < 1e-12) return '';
                if (selectedMetric === 'change') return `${n >= 0 ? '+' : ''}${n.toFixed(2)}`;
                if (selectedMetric === 'profit') return `${n >= 0 ? '+' : ''}${n.toFixed(2)}`;
                if (selectedMetric === 'lots') return n.toFixed(2);
                return `${n >= 0 ? '+' : ''}${n.toFixed(1)}`;
              }}
              style={{ fontSize: 9, fill: 'var(--color-text-muted)', fontWeight: 600, pointerEvents: 'none' }}
            />
            {series.map((item) => (
              <Cell
                key={`${item.year}-${item.month}`}
                fill={barCellFill(item)}
              />
            ))}
          </Bar>
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
