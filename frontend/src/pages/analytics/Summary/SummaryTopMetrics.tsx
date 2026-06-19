import { memo } from 'react';
import { Row, Col } from 'antd';
import { RiseOutlined, LineChartOutlined, AimOutlined, PieChartOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { SUMMARY_METRICS_BALANCE_KEY, SUMMARY_METRICS_EQUITY_KEY, SUMMARY_METRICS_EQUITY_VALUE_KEY, SUMMARY_METRICS_NET_PROFIT_KEY } from '@/gen/ant/v1/i18n/analytics_keys';

;

interface Props {
  netProfit: number;
  latestEquity: number;
  latestBalance: number;
}

function SummaryTopMetrics({ netProfit, latestEquity, latestBalance }: Props) {
  const { t } = useTranslation();
  const cards = [
    { icon: <RiseOutlined />, color: '#00A651', label: t(SUMMARY_METRICS_NET_PROFIT_KEY), value: `$${Number(netProfit || 0).toFixed(2)}`, valueColor: netProfit >= 0 ? '#00A651' : '#E53935' },
    { icon: <LineChartOutlined />, color: '#2196F3', label: t(SUMMARY_METRICS_EQUITY_KEY), value: `$${latestEquity.toFixed(2)}`, valueColor: 'var(--color-text)' },
    { icon: <AimOutlined />, color: '#D4AF37', label: t(SUMMARY_METRICS_BALANCE_KEY), value: `$${Number(latestBalance).toFixed(2)}`, valueColor: 'var(--color-text)' },
    { icon: <PieChartOutlined />, color: '#9C27B0', label: t(SUMMARY_METRICS_EQUITY_VALUE_KEY), value: `$${latestEquity.toFixed(2)}`, valueColor: 'var(--color-text)' },
  ];
  return (
    <Row gutter={[16, 16]} className="mt-6">
      {cards.map((c, i) => (
        <Col xs={12} sm={6} key={i}>
          <div className="stat-card">
            <div className="flex items-center gap-2 mb-2">
              {c.icon}
              <span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}>{c.label}</span>
            </div>
            <div className="text-2xl font-semibold" style={{ color: c.valueColor }}>{c.value}</div>
          </div>
        </Col>
      ))}
    </Row>
  );
}

export default memo(SummaryTopMetrics);
