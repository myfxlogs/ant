import { memo } from 'react';
import { Card, Row, Col } from 'antd';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts';
import { useTranslation } from 'react-i18next'
import { SUMMARY_CARDS_DIRECTION_SHARE_KEY, SUMMARY_CARDS_PNL_SHARE_KEY, SUMMARY_CARDS_SYMBOL_PNL_COMPARE_KEY, SUMMARY_CARDS_SYMBOL_TRADE_SHARE_KEY, SUMMARY_LABELS_PNL_KEY } from '@/gen/ant/v1/i18n/analytics_keys';

;

interface PieDataItem { name: string; value: number; color: string; [key: string]: unknown; }
interface SymbolStatItem { symbol: string; profit: number; [key: string]: unknown; }

interface Props {
  symbolStats: SymbolStatItem[];
  symbolPieData: PieDataItem[];
  directionPieData: PieDataItem[];
  profitPieData: PieDataItem[];
}

function SummaryPieGrid({ symbolStats, symbolPieData, directionPieData, profitPieData }: Props) {
  const { t } = useTranslation();
  return (
    <Row gutter={[16, 16]} className="mt-6">
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_SYMBOL_PNL_COMPARE_KEY)}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={(symbolStats || []).slice(0, 5)} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
              <XAxis type="number" stroke="var(--color-text-muted)" fontSize={12} />
              <YAxis dataKey="symbol" type="category" stroke="var(--color-text-muted)" fontSize={12} width={60} />
              <Tooltip
                contentStyle={{ background: 'var(--color-bg-card)', border: '1px solid rgba(0, 0, 0, 0.1)', borderRadius: '8px' }}
                formatter={(value: number | undefined) => [`$${(value || 0).toFixed(2)}`, t(SUMMARY_LABELS_PNL_KEY)]}
              />
              <Bar dataKey="profit" fill="#D4AF37" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_SYMBOL_TRADE_SHARE_KEY)}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={symbolPieData} cx="50%" cy="50%" innerRadius={18} outerRadius={34} paddingAngle={2} dataKey="value" label={({ name, value }) => `${name} (${value}%)`}>
                {symbolPieData.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
              </Pie>
              <Tooltip /><Legend />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_DIRECTION_SHARE_KEY)}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={directionPieData} cx="50%" cy="50%" innerRadius={18} outerRadius={34} paddingAngle={2} dataKey="value" label={({ name, value }) => `${name} (${value})`}>
                {directionPieData.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
              </Pie>
              <Tooltip /><Legend />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_PNL_SHARE_KEY)}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={profitPieData} cx="50%" cy="50%" innerRadius={18} outerRadius={34} paddingAngle={2} dataKey="value" label={({ name, value }) => `${name} (${value})`}>
                {profitPieData.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
              </Pie>
              <Tooltip /><Legend />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      </Col>
    </Row>
  );
}

export default memo(SummaryPieGrid);
