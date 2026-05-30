import { Card, Row, Col } from 'antd';
import { PieChart, Pie, Cell, Tooltip, Legend, ResponsiveContainer, BarChart, Bar, XAxis, YAxis, CartesianGrid } from 'recharts';
import { useTranslation } from 'react-i18next';

interface PieDataItem { name: string; value: number; color: string; [key: string]: unknown; }
interface SymbolStatItem { symbol: string; profit: number; [key: string]: unknown; }

interface Props {
  symbolStats: SymbolStatItem[];
  symbolPieData: PieDataItem[];
  directionPieData: PieDataItem[];
  profitPieData: PieDataItem[];
}

export default function SummaryPieGrid({ symbolStats, symbolPieData, directionPieData, profitPieData }: Props) {
  const { t } = useTranslation();
  return (
    <Row gutter={[16, 16]} className="mt-6">
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.symbolPnlCompare')}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <BarChart data={(symbolStats || []).slice(0, 5)} layout="vertical">
              <CartesianGrid strokeDasharray="3 3" stroke="#E8ECF0" />
              <XAxis type="number" stroke="#8A9AA5" fontSize={12} />
              <YAxis dataKey="symbol" type="category" stroke="#8A9AA5" fontSize={12} width={60} />
              <Tooltip
                contentStyle={{ background: '#FFFFFF', border: '1px solid rgba(0, 0, 0, 0.1)', borderRadius: '8px' }}
                formatter={(value: number | undefined) => [`$${(value || 0).toFixed(2)}`, t('analytics.summary.labels.pnl')]}
              />
              <Bar dataKey="profit" fill="#D4AF37" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.symbolTradeShare')}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={symbolPieData} cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2} dataKey="value">
                {symbolPieData.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
              </Pie>
              <Tooltip /><Legend />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.directionShare')}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={directionPieData} cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2} dataKey="value">
                {directionPieData.map((entry, index) => <Cell key={`cell-${index}`} fill={entry.color} />)}
              </Pie>
              <Tooltip /><Legend />
            </PieChart>
          </ResponsiveContainer>
        </Card>
      </Col>
      <Col xs={24} lg={12}>
        <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.pnlShare')}</span>} className="glass-card">
          <ResponsiveContainer width="100%" height={200}>
            <PieChart>
              <Pie data={profitPieData} cx="50%" cy="50%" innerRadius={50} outerRadius={80} paddingAngle={2} dataKey="value">
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
