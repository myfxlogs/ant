import { Typography, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  ComposedChart, Line, Bar, XAxis, YAxis, CartesianGrid, Tooltip,
  ResponsiveContainer, Legend, Area, AreaChart,
} from 'recharts';
import type { RevenueTrendPoint, SubscriberTrendPoint } from '@/gen/ant/v1/marketplace_service_pb';

const { Text } = Typography;

function formatDate(ms: number | bigint): string {
  const d = new Date(Number(ms));
  return `${d.getMonth() + 1}/${d.getDate()}`;
}

export function RevenueTrendChart({ data }: { data: RevenueTrendPoint[] }) {
  const { t } = useTranslation();
  if (!data || data.length === 0) {
    return <Empty description={t('marketplace.author.noRevenueData')} />;
  }
  const chartData = data.map(p => ({
    date: formatDate(p.dateMs),
    sale: Number(p.saleRevenue || 0),
    subscription: Number(p.subscriptionRevenue || 0),
  }));
  return (
    <ResponsiveContainer width="100%" height={200}>
      <ComposedChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
        <XAxis dataKey="date" tick={{ fontSize: 11 }} />
        <YAxis tick={{ fontSize: 11 }} />
        <Tooltip />
        <Legend />
        <Bar dataKey="sale" name={t('marketplace.author.saleRevenue')} fill="#52c41a" />
        <Bar dataKey="subscription" name={t('marketplace.author.subRevenue')} fill="#1890ff" />
      </ComposedChart>
    </ResponsiveContainer>
  );
}

export function SubscriberTrendChart({ data }: { data: SubscriberTrendPoint[] }) {
  const { t } = useTranslation();
  if (!data || data.length === 0) {
    return <Empty description={t('marketplace.author.noSubscriberData')} />;
  }
  const chartData = data.map(p => ({
    date: formatDate(p.dateMs),
    new: Number(p.newSubscribers || 0),
    churned: Number(p.churned || 0),
    active: Number(p.active || 0),
  }));
  return (
    <ResponsiveContainer width="100%" height={200}>
      <ComposedChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" stroke="#f0f0f0" />
        <XAxis dataKey="date" tick={{ fontSize: 11 }} />
        <YAxis tick={{ fontSize: 11 }} />
        <Tooltip />
        <Legend />
        <Area dataKey="active" name={t('marketplace.author.activeSubs')} fill="#1890ff" fillOpacity={0.1} stroke="#1890ff" />
        <Bar dataKey="new" name={t('marketplace.author.newSubs')} fill="#52c41a" />
        <Bar dataKey="churned" name={t('marketplace.author.churned')} fill="#ff4d4f" />
      </ComposedChart>
    </ResponsiveContainer>
  );
}
