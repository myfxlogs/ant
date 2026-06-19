import { Row, Col, Skeleton, Statistic } from 'antd';
import { DollarOutlined, LineChartOutlined, TeamOutlined, ArrowUpOutlined, AccountBookOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { STATS_ACCOUNT_COUNT_KEY, STATS_CONNECTED_KEY, STATS_TOTAL_BALANCE_KEY, STATS_TOTAL_EQUITY_KEY, STATS_TOTAL_PROFIT_KEY } from '@/gen/ant/v1/i18n/dashboard_keys';

;

interface Stats {
  totalBalance: number;
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
    return <Row gutter={[16, 16]}>{[1, 2, 3, 4, 5].map((i) => <Col xs={12} sm={6} key={i}><Skeleton active paragraph={{ rows: 1 }} title={{ width: '60%' }} /></Col>)}</Row>;
  }
  const cards = [
    { icon: <AccountBookOutlined size={20} />, bg: 'rgba(33,150,243,0.1)', color: '#2196F3', title: t(STATS_TOTAL_BALANCE_KEY), value: stats.totalBalance, valueColor: 'var(--color-text)', prefix: '$', precision: 2 },
    { icon: <DollarOutlined size={20} />, bg: 'rgba(212,175,55,0.1)', color: '#D4AF37', title: t(STATS_TOTAL_EQUITY_KEY), value: stats.totalEquity, valueColor: 'var(--color-text)', prefix: '$', precision: 2 },
    { icon: <LineChartOutlined size={20} />, bg: 'rgba(0,166,81,0.1)', color: '#00A651', title: t(STATS_CONNECTED_KEY), value: stats.connectedCount, valueColor: '#00A651', prefix: undefined, precision: 0 },
    { icon: <TeamOutlined size={20} />, bg: 'rgba(90,107,117,0.1)', color: 'var(--color-text-secondary)', title: t(STATS_ACCOUNT_COUNT_KEY), value: stats.accountCount, valueColor: 'var(--color-text)', prefix: undefined, precision: 0 },
    { icon: <LineChartOutlined size={20} />, bg: 'rgba(0,166,81,0.1)', color: '#00A651', title: t(STATS_TOTAL_PROFIT_KEY), value: stats.totalProfit, valueColor: stats.totalProfit >= 0 ? '#00A651' : '#E53935', prefix: '$', precision: 2 },
  ];
  return (
    <Row gutter={[16, 16]}>
      {cards.map((c, i) => (
        <Col xs={12} sm={6} key={i}>
          <div className="stat-card group cursor-pointer">
            <div className="flex items-center justify-between mb-3">
              <div className="w-10 h-10 rounded-xl flex items-center justify-center" style={{ background: c.bg }}>{c.icon}</div>
              {i === 4 ? <ArrowUpOutlined size={16} style={{ color: c.valueColor, transform: stats.totalProfit < 0 ? 'rotate(180deg)' : undefined }} /> : i === 0 || i === 1 ? <ArrowUpOutlined size={16} color="#00A651" /> : null}
            </div>
            <Statistic title={<span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>{c.title}</span>} value={c.value} precision={c.precision} prefix={c.prefix} styles={{ content: { color: c.valueColor, fontSize: '24px', fontWeight: 600 } }} />
          </div>
        </Col>
      ))}
    </Row>
  );
}
