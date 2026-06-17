import { ResponsiveContainer, AreaChart, Area, Tooltip, YAxis } from 'recharts';

export default function ShareChart({ data }: { data: number[] }) {
  const chartData = data.map((v, i) => ({ x: i, value: v }));
  return (
    <ResponsiveContainer width="100%" height={260}>
      <AreaChart data={chartData} margin={{ top: 4, right: 4, bottom: 4, left: 4 }}>
        <defs>
          <linearGradient id="grad" x1="0" y1="0" x2="0" y2="1">
            <stop offset="0%" stopColor="#1677ff" stopOpacity={0.3} />
            <stop offset="100%" stopColor="#1677ff" stopOpacity={0} />
          </linearGradient>
        </defs>
        <YAxis hide domain={['auto', 'auto']} />
        <Tooltip formatter={(v: number) => v.toFixed(2)} labelFormatter={() => ''} />
        <Area type="monotone" dataKey="value" stroke="#1677ff" strokeWidth={2} fill="url(#grad)" dot={false} />
      </AreaChart>
    </ResponsiveContainer>
  );
}
