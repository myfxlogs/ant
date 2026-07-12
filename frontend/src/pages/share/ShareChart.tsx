import { ResponsiveContainer, AreaChart, Area, Tooltip, YAxis, ReferenceLine } from 'recharts';

export default function ShareChart({ data }: { data: number[] }) {
  const chartData = data.map((v, i) => ({ x: i, value: typeof v === 'bigint' ? Number(v) : v }));
  const lastValue = chartData.length > 0 ? chartData[chartData.length - 1].value : 0;
  const firstValue = chartData.length > 0 ? chartData[0].value : 0;
  const isPositive = lastValue >= firstValue;
  const color = isPositive ? '#52c41a' : '#ff4d4f';
  const gradId = isPositive ? 'grad-pos' : 'grad-neg';

  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={chartData} margin={{ top: 4, right: 4, bottom: 4, left: 4 }}>
        <defs>
          <linearGradient id={gradId} x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor={color} stopOpacity={0.3} />
            <stop offset="100%" stopColor={color} stopOpacity={0} />
          </linearGradient>
        </defs>
        <YAxis hide domain={['auto', 'auto']} />
        <Tooltip
          formatter={(v: number) => v.toFixed(2)}
          labelFormatter={() => ''}
          contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e8e8e8' }}
        />
        <ReferenceLine y={firstValue} stroke="#d9d9d9" strokeDasharray="3 3" />
        <Area type="monotone" dataKey="value" stroke={color} strokeWidth={2} fill={`url(#${gradId})`} dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  );
}
