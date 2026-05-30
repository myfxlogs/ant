import { Card, Row, Col, Statistic } from 'antd';
import { ClockCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { formatHoldingTime } from '@/utils/date';

interface TradeStats {
  totalTrades?: number;
  winningTrades?: number;
  losingTrades?: number;
  winRate?: number;
  profitFactor?: number;
  maxConsecutiveWins?: number;
  maxConsecutiveLosses?: number;
  averageHoldingTime?: string;
  averageVolume?: number;
  averageProfit?: number;
  averageLoss?: number;
}

interface RiskMetrics {
  maxDrawdown?: number;
  maxDrawdownPercent?: number;
  sharpeRatio?: number;
  sortinoRatio?: number;
  volatility?: number;
  valueAtRisk?: number;
}

interface Props {
  tradeStats: TradeStats | null;
  riskMetrics: RiskMetrics | null;
}

export default function SummaryMetricsCards({ tradeStats, riskMetrics }: Props) {
  const { t } = useTranslation();
  return (
    <>
      <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.tradeStats')}</span>} className="glass-card mt-6">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.totalTrades')}</span>} value={tradeStats?.totalTrades || 0} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.wins')}</span>} value={tradeStats?.winningTrades || 0} suffix={<span style={{ color: '#8A9AA5', fontSize: '14px' }}> ({tradeStats?.winRate?.toFixed(0) || 0}%)</span>} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.losses')}</span>} value={tradeStats?.losingTrades || 0} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.winRate')}</span>} value={tradeStats?.winRate || 0} suffix="%" valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.profitFactor')}</span>} value={tradeStats?.profitFactor || 0} precision={2} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={24} sm={12} md={6}>
            <div className="flex items-center gap-2"><ClockCircleOutlined size={16} stroke={1.5} color="#8A9AA5" /><span style={{ color: '#8A9AA5', fontSize: '14px' }}>{t('analytics.summary.tradeStats.avgHolding')}</span></div>
            <div className="text-lg font-semibold mt-1" style={{ color: '#141D22' }}>{formatHoldingTime(tradeStats?.averageHoldingTime) || '-'}</div>
          </Col>
        </Row>
        <Row gutter={[16, 16]} className="mt-4">
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxConsecutiveWins')}</span>} value={tradeStats?.maxConsecutiveWins || 0} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxConsecutiveLosses')}</span>} value={tradeStats?.maxConsecutiveLosses || 0} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.maxHolding')}</span>} value={'-'} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgVolume')}</span>} value={tradeStats?.averageVolume || 0} precision={2} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgProfit')}</span>} value={tradeStats?.averageProfit || 0} prefix="$" valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.tradeStats.avgLoss')}</span>} value={tradeStats?.averageLoss || 0} prefix="$" precision={2} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
        </Row>
      </Card>

      <Card title={<span style={{ color: '#141D22', fontWeight: 500 }}>{t('analytics.summary.cards.riskMetrics')}</span>} className="glass-card mt-6">
        <Row gutter={16}>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.maxDrawdown')}</span>} value={Math.abs(riskMetrics?.maxDrawdown || 0)} precision={2} prefix="$" valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.maxDrawdownPct')}</span>} value={Math.abs(riskMetrics?.maxDrawdownPercent || 0)} precision={2} suffix="%" valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.sharpe')}</span>} value={riskMetrics?.sharpeRatio || 0} precision={2} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.sortino')}</span>} value={riskMetrics?.sortinoRatio || 0} precision={2} valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.volatility')}</span>} value={riskMetrics?.volatility || 0} precision={2} suffix="%" valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: '#8A9AA5' }}>{t('analytics.summary.risk.var95')}</span>} value={riskMetrics?.valueAtRisk || 0} precision={2} prefix="$" valueStyle={{ color: '#141D22', fontSize: '20px' }} /></Col>
        </Row>
      </Card>
    </>
  );
}
