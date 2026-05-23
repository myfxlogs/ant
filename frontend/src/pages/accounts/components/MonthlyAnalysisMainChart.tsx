import type { MouseEvent as ReactMouseEvent } from 'react';
import {
  Bar,
  CartesianGrid,
  Cell,
  ComposedChart,
  LabelList,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';

import {
  type MetricType,
  type MonthlyAnalysisPoint,
  type MonthlyBarRow,
  barCellFill,
  monthShortLabels,
} from './MonthlyAnalysisCard.shared';

type RechartsMouseState = {
  isTooltipActive?: boolean;
  activeTooltipIndex?: number | string;
};

type Props = {
  series: MonthlyBarRow[];
  selectedMetric: MetricType;
  metricTitleMap: Record<MetricType, string>;
  formatValue: (metric: MetricType, value: number) => string;
  renderMetricValue: (metric: MetricType, value: number) => JSX.Element;
  onMouseDown: (e: ReactMouseEvent) => void;
  onMouseMove: (state: RechartsMouseState) => void;
  onMouseLeave: () => void;
  onCommitByTooltipIndex: (activeTooltipIndex: number | string | undefined) => void;
  onCommitMonthClick: (data: unknown, index: number) => void;
};

export default function MonthlyAnalysisMainChart({
  series,
  selectedMetric,
  metricTitleMap,
  formatValue,
  renderMetricValue,
  onMouseDown,
  onMouseMove,
  onMouseLeave,
  onCommitByTooltipIndex,
  onCommitMonthClick,
}: Props) {
  return (
    <div
      className="outline-none [&_.recharts-wrapper]:!outline-none [&_.recharts-wrapper]:ring-0 [&_.recharts-surface]:outline-none"
      onMouseDown={onMouseDown}
    >
      <ResponsiveContainer width="100%" height={250}>
        <ComposedChart
          data={series}
          margin={{ top: 22, right: 6, left: 0, bottom: 4 }}
          onMouseMove={onMouseMove}
          onMouseLeave={onMouseLeave}
          onClick={(state: RechartsMouseState) => onCommitByTooltipIndex(state.activeTooltipIndex)}
        >
          <CartesianGrid strokeDasharray="3 3" stroke="#E6EDF5" vertical={false} />
          <XAxis
            dataKey="monthAxisLabel"
            stroke="#94A3B8"
            fontSize={10}
            tickLine={false}
            axisLine={{ stroke: '#E6EDF5' }}
            interval={0}
            angle={-32}
            textAnchor="end"
            height={46}
          />
          <YAxis stroke="#94A3B8" fontSize={11} tickLine={false} axisLine={{ stroke: '#E6EDF5' }} />
          <Tooltip
            wrapperStyle={{ pointerEvents: 'none' }}
            cursor={false}
            content={({ active, payload }) => {
              if (!active || !payload?.length) return null;
              const point = payload[0]?.payload as MonthlyAnalysisPoint | undefined;
              if (!point) return null;
              return (
                <div
                  style={{
                    background: '#FFFFFF',
                    border: '1px solid #D9E2EC',
                    borderRadius: '6px',
                    boxShadow: '0 4px 10px rgba(15, 23, 42, 0.12)',
                    padding: '8px 10px',
                    minWidth: 190,
                    fontSize: 12,
                    pointerEvents: 'none',
                  }}
                >
                  <div style={{ fontWeight: 700, color: '#1F2937', marginBottom: 6 }}>
                    {monthShortLabels[(point.month || 1) - 1]} {point.year}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#475467', marginBottom: 2 }}>
                    <span>{metricTitleMap.change}</span>
                    {renderMetricValue('change', point.change || 0)}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#475467', marginBottom: 2 }}>
                    <span>{metricTitleMap.profit}</span>
                    {renderMetricValue('profit', point.profit || 0)}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#475467', marginBottom: 2 }}>
                    <span>{metricTitleMap.lots}</span>
                    {renderMetricValue('lots', point.lots || 0)}
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', color: '#475467' }}>
                    <span>{metricTitleMap.pips}</span>
                    {renderMetricValue('pips', point.pips || 0)}
                  </div>
                </div>
              );
            }}
          />
          <Bar dataKey="value" radius={[2, 2, 0, 0]} minPointSize={8} isAnimationActive={false} style={{ cursor: 'pointer' }} onClick={onCommitMonthClick}>
            <LabelList
              dataKey="value"
              position="top"
              formatter={(v: number) => (Math.abs(Number(v)) < 1e-12 ? '' : formatValue(selectedMetric, v))}
              style={{ fontSize: 10, fill: '#475467', fontWeight: 600, pointerEvents: 'none' }}
            />
            {series.map((item) => (
              <Cell key={`${item.year}-${item.month}`} fill={barCellFill(item)} />
            ))}
          </Bar>
        </ComposedChart>
      </ResponsiveContainer>
    </div>
  );
}
