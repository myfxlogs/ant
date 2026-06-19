import { memo } from 'react';
import { Card, Row, Col, Statistic } from 'antd';
import { useTranslation } from 'react-i18next'
import { SUMMARY_CARDS_RISK_METRICS_KEY, SUMMARY_CARDS_TRADE_STATS_KEY, SUMMARY_RISK_MAX_DRAWDOWN_PCT_KEY, SUMMARY_RISK_SHARPE_KEY, SUMMARY_RISK_SORTINO_KEY, SUMMARY_RISK_VOLATILITY_KEY, SUMMARY_TRADE_STATS_AVG_HOLDING_KEY, SUMMARY_TRADE_STATS_AVG_LOSS_KEY, SUMMARY_TRADE_STATS_AVG_PROFIT_KEY, SUMMARY_TRADE_STATS_AVG_VOLUME_KEY, SUMMARY_TRADE_STATS_LOSSES_KEY, SUMMARY_TRADE_STATS_MAX_CONSECUTIVE_LOSSES_KEY, SUMMARY_TRADE_STATS_MAX_CONSECUTIVE_WINS_KEY, SUMMARY_TRADE_STATS_MAX_HOLDING_KEY, SUMMARY_TRADE_STATS_PROFIT_FACTOR_KEY, SUMMARY_TRADE_STATS_TOTAL_TRADES_KEY, SUMMARY_TRADE_STATS_WINS_KEY, SUMMARY_TRADE_STATS_WIN_RATE_KEY } from '@/gen/ant/v1/i18n/analytics_keys';

;
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
      <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_TRADE_STATS_KEY)}</span>} className="glass-card mt-6">
        <Row gutter={[16, 16]}>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_TOTAL_TRADES_KEY)}</span>} value={total} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_WINS_KEY)}</span>} value={wins} suffix={<span style={{ color: 'var(--color-text-muted)', fontSize: '14px' }}> ({winRate.toFixed(0)}%)</span>} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_LOSSES_KEY)}</span>} value={losses} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_WIN_RATE_KEY)}</span>} value={tradeStats?.winRate || 0} suffix="%" valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_PROFIT_FACTOR_KEY)}</span>} value={tradeStats?.profitFactor || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_AVG_HOLDING_KEY)}</span>} value={formatHoldingTime(tradeStats?.averageHoldingTime) || '-'} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
        </Row>
        <Row gutter={[16, 16]} className="mt-4">
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_MAX_CONSECUTIVE_WINS_KEY)}</span>} value={tradeStats?.maxConsecutiveWins || 0} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_MAX_CONSECUTIVE_LOSSES_KEY)}</span>} value={tradeStats?.maxConsecutiveLosses || 0} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_MAX_HOLDING_KEY)}</span>} value={'-'} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_AVG_VOLUME_KEY)}</span>} value={tradeStats?.averageVolume || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_AVG_PROFIT_KEY)}</span>} value={tradeStats?.averageProfit || 0} prefix="$" precision={2} valueStyle={{ color: '#00A651', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={8} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_TRADE_STATS_AVG_LOSS_KEY)}</span>} value={tradeStats?.averageLoss || 0} prefix="$" precision={2} valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
        </Row>
      </Card>

      <Card title={<span style={{ color: 'var(--color-text)', fontWeight: 500 }}>{t(SUMMARY_CARDS_RISK_METRICS_KEY)}</span>} className="glass-card mt-6">
        <Row gutter={16}>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_RISK_MAX_DRAWDOWN_PCT_KEY)}</span>} value={Math.abs(riskMetrics?.maxDrawdownPercent || 0)} precision={2} suffix="%" valueStyle={{ color: '#E53935', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_RISK_SHARPE_KEY)}</span>} value={riskMetrics?.sharpeRatio || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_RISK_SORTINO_KEY)}</span>} value={riskMetrics?.sortinoRatio || 0} precision={2} valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
          <Col xs={12} sm={6} md={4}><Statistic title={<span style={{ color: 'var(--color-text-muted)' }}>{t(SUMMARY_RISK_VOLATILITY_KEY)}</span>} value={riskMetrics?.volatility || 0} precision={2} suffix="%" valueStyle={{ color: 'var(--color-text)', fontSize: '20px' }} /></Col>
        </Row>
      </Card>
    </>
  );
}

export default memo(SummaryMetricsCards);
