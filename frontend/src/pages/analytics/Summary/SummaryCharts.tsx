import { Select } from 'antd';
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  BarChart, Bar,
} from 'recharts';
import { useTranslation } from 'react-i18next';

interface ChartDataPoint {
  date?: string;
  month?: string;
  equity?: number;
  profit?: number;
  [key: string]: unknown;
}

interface Props {
  equityCurveData: ChartDataPoint[];
  monthlyData: ChartDataPoint[];
  selectedYear: number;
  yearOptions: { value: number; label: string }[];
  onYearChange: (year: number) => void;
}

export default function SummaryCharts({
  equityCurveData,
  monthlyData,
  selectedYear,
  yearOptions,
  onYearChange,
}: Props) {
  const { t } = useTranslation();
  return (
    <>
      <div className="rounded-2xl p-6" style={{ background: '#FFFFFF', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
        <h2 className="text-lg font-semibold mb-4" style={{ color: '#141D22' }}>{t('analytics.summary.sections.equityCurve')}</h2>
        <ResponsiveContainer width="100%" height={250}>
          <LineChart data={equityCurveData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
            <XAxis dataKey="date" stroke="#8A9AA5" fontSize={12} />
            <YAxis stroke="#8A9AA5" fontSize={12} />
            <Tooltip contentStyle={{ background: '#FFFFFF', border: '1px solid rgba(0, 0, 0, 0.1)', borderRadius: '8px' }} />
            <Line type="monotone" dataKey="equity" stroke="#D4AF37" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>

      <div className="rounded-2xl p-6 mt-6" style={{ background: '#FFFFFF', boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)' }}>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold" style={{ color: '#141D22' }}>{t('analytics.summary.sections.monthlyStats')}</h2>
          <Select value={selectedYear} onChange={onYearChange} options={yearOptions} style={{ width: 100 }} />
        </div>
        <ResponsiveContainer width="100%" height={200}>
          <BarChart data={monthlyData}>
            <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
            <XAxis dataKey="month" stroke="#8A9AA5" fontSize={12} />
            <YAxis stroke="#8A9AA5" fontSize={12} />
            <Tooltip
              contentStyle={{ background: '#FFFFFF', border: '1px solid rgba(0, 0, 0, 0.1)', borderRadius: '8px' }}
              formatter={(value: number | undefined) => [`$${(value || 0).toFixed(2)}`, t('analytics.summary.labels.pnl')]}
            />
            <Bar dataKey="profit" fill="#D4AF37" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </>
  );
}
