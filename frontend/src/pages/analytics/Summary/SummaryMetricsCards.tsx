import { memo } from 'react';
import { Card, Row, Col, Statistic } from 'antd';
import { useTranslation } from 'react-i18next';
import { formatHoldingTime } from '@/utils/date';

interface TradeStats {
  totalTrades?: number;
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
  maxDrawdownPercent?: number;
  sharpeRatio?: number;
  sortinoRatio?: number;
  volatility?: number;
}

interface Props {
  tradeStats: TradeStats | null;
  riskMetrics: RiskMetrics | null;
}

function SummaryMetricsCards({ tradeStats, riskMetrics }: Props) {
  const { t } = useTranslation();
  const total = Number(tradeStats?.totalTrades || 0);
  const winRate = Number(tradeStats?.winRate || 0);
  const wins = Math.round(total * winRate / 100);
  const losses = total - wins;
  return (
    <>
      <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t('analytics.summary.cards.tradeStats')}</span>} className="glass-card mt-6">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.totalTrades')}</span>} value={total} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.wins')}</span>} value={wins} suffix={<span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}> ({winRate.toFixed(0)}%)</span>} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.losses')}</span>} value={losses} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.winRate')}</span>} value={tradeStats?.winRate || 0} suffix="%" valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.profitFactor')}</span>} value={tradeStats?.profitFactor || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.avgHolding')}</span>} value={formatHoldingTime(tradeStats?.averageHoldingTime) || '-'} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
        </Row>
        <Row gutter={[16, 16]} className="mt-4">
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.maxConsecutiveWins')}</span>} value={tradeStats?.maxConsecutiveWins || 0} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.maxConsecutiveLosses')}</span>} value={tradeStats?.maxConsecutiveLosses || 0} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.maxHolding')}</span>} value={'-'} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.avgVolume')}</span>} value={tradeStats?.averageVolume || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.avgProfit')}</span>} value={tradeStats?.averageProfit || 0} prefix="$" precision={2} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.tradeStats.avgLoss')}</span>} value={tradeStats?.averageLoss || 0} prefix="$" precision={2} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
        </Row>
      </Card>

      <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t('analytics.summary.cards.riskMetrics')}</span>} className="glass-card mt-6">
        <Row gutter={16}>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.risk.maxDrawdownPct')}</span>} value={Math.abs(riskMetrics?.maxDrawdownPercent || 0)} precision={2} suffix="%" valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.risk.sharpe')}</span>} value={riskMetrics?.sharpeRatio || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.risk.sortino')}</span>} value={riskMetrics?.sortinoRatio || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t('analytics.summary.risk.volatility')}</span>} value={riskMetrics?.volatility || 0} precision={2} suffix="%" valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
        </Row>
      </Card>
    </>
  );
}

export default memo(SummaryMetricsCards);
