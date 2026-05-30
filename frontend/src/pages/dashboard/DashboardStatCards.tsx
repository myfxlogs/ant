import { Row, Col, Skeleton, Statistic } from 'antd';
import { DollarOutlined, LineChartOutlined, TeamOutlined, ArrowUpOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface Stats {
  totalEquity: number;
  connectedCount: number;
  accountCount: number;
  totalProfit: number;
}

interface Props {
  stats: Stats;
  loading?: boolean;
}

export default function DashboardStatCards({ stats, loading }: Props) {
  const { t } = useTranslation();
  if (loading) {
    return <Row gutter={[16, 16]}>{[1, 2, 3, 4].map((i) => <Col xs={12} sm={6} key={i}><Skeleton active paragraph={{ rows: 1 }} title={{ width: '60%' }} /></Col>)}</Row>;
  }
  const cards = [
    { icon: <DollarOutlined size={20} />, bg: 'rgba(212,175,55,0.1)', color: '#D4AF37', title: t('dashboard.stats.totalEquity'), value: stats.totalEquity, valueColor: '#141D22' },
    { icon: <LineChartOutlined size={20} />, bg: 'rgba(0,166,81,0.1)', color: '#00A651', title: t('dashboard.stats.connected'), value: stats.connectedCount, valueColor: '#00A651' },
    { icon: <TeamOutlined size={20} />, bg: 'rgba(90,107,117,0.1)', color: '#5A6B75', title: t('dashboard.stats.accountCount'), value: stats.accountCount, valueColor: '#141D22' },
    { icon: <LineChartOutlined size={20} />, bg: 'rgba(0,166,81,0.1)', color: '#00A651', title: t('dashboard.stats.totalProfit'), value: stats.totalProfit, valueColor: stats.totalProfit >= 0 ? '#00A651' : '#E53935' },
  ];
  return (
    <Row gutter={[16, 16]}>
      {cards.map((c, i) => (
        <Col xs={12} sm={6} key={i}>
          <div className="stat-card group cursor-pointer">
            <div className="flex items-center justify-between mb-3">
              <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: c.bg }}>{c.icon}</div>
              {i === 3 ? <ArrowUpOutlined size={16} style={{ color: c.valueColor, transform: stats.totalProfit < 0 ? 'rotate(180deg)' : undefined }} /> : i === 0 ? <ArrowUpOutlined size={16} color="#00A651" /> : null}
            </div>
            <Statistic title={<span style={{ color: '#8A9AA5', fontSize: '14px' }}>{c.title}</span>} value={c.value} precision={2} prefix="$" styles={{ content: { color: c.valueColor, fontSize: '24px', fontWeight: 600 } }} />
          </div>
        </Col>
      ))}
    </Row>
  );
}
