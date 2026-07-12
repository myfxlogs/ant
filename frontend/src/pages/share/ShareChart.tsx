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

  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={chartData} margin={{ top: 4, right: 8, bottom: 4, left: 4 }}>
        <defs>
          <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <YAxis hide domain={['auto', 'auto']} />
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
          formatter={(v: number) => v.toFixed(2)}
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
