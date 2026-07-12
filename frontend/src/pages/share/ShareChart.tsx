import { ResponsiveContainer, AreaChart, Area, Tooltip, YAxis, XAxis, ReferenceLine } from 'recharts';

interface ShareChartProps {
  data: number[];
  timesMs?: number[];
}

export default function ShareChart({ data, timesMs }: ShareChartProps) {
  const chartData = data.map((v, i) => ({
    x: timesMs && timesMs[i] ? timesMs[i] : i,
    value: typeof v === 'bigint' ? Number(v) : v,
  }));
  const lastValue = chartData.length > 0 ? chartData[chartData.length - 1].value : 0;
  const firstValue = chartData.length > 0 ? chartData[0].value : 0;
  const isPositive = lastValue >= firstValue;
  const color = isPositive ? '#52c41a' : '#ff4d4f';
  const gradId = isPositive ? 'grad-pos' : 'grad-neg';

  const hasTimeAxis = !!timesMs && timesMs.length > 0;

  // Handle flat equity curve: if all values are the same, expand the YAxis domain
  // so the chart doesn't render as a flat line at the top/bottom edge.
  const allSame = chartData.length > 0 && chartData.every(d => d.value === firstValue);
  const padding = allSame ? Math.max(Math.abs(firstValue) * 0.01, 1) : 0;
  const yDomain: [number | string, number | string] = allSame
    ? [firstValue - padding, firstValue + padding]
    : ['auto', 'auto'];

  // Single data point: recharts needs at least 2 points to draw an area.
  const renderData = chartData.length === 1
    ? [chartData[0], { ...chartData[0], x: chartData[0].x + 1 }]
    : chartData;

  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={renderData} margin={{ top: 4, right: 8, bottom: 4, left: 4 }}>
        <defs>
          <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <YAxis hide domain={yDomain} />
        {hasTimeAxis && (
          <XAxis
            dataKey="x"
            scale="time"
            type="number"
            domain={['dataMin', 'dataMax']}
            tickFormatter={(v: number) => {
              const d = new Date(v);
              return `${d.getMonth() + 1}/${d.getDate()}`;
            }}
            tick={{ fontSize: 10, fill: '#8c8c8c' }}
            tickLine={false}
            axisLine={{ stroke: '#e8e8e8' }}
            minTickGap={40}
          />
        )}
        <Tooltip
          formatter={(v: number | undefined) => (v ?? 0).toFixed(2)}
          labelFormatter={(label: unknown) => {
            if (hasTimeAxis && typeof label === 'number') {
              return new Date(label).toLocaleDateString();
            }
            return '';
          }}
          contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e8e8e8' }}
        />
        <ReferenceLine y={firstValue} stroke="#d9d9d9" strokeDasharray="3 3" />
        <Area type="monotone" dataKey="value" stroke={color} strokeWidth={2} fill={`url(#${gradId})`} dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  );
}
