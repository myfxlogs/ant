import { Alert, Card, Descriptions, Progress, Tag } from 'antd';
import { Line, LineChart, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { BacktestRunStatus } from '@/gen/backtest_run_pb';
import { useTranslation } from 'react-i18next';

function backtestStatusLabel(status: string, t: (k: string) => string): string {
  const s = String(status || '').trim().toUpperCase();
  const n = Number(s);
  if (Number.isFinite(n)) {
    if (n === BacktestRunStatus.SUCCEEDED) return t('ai.backtestScoreCard.status.succeeded');
    if (n === BacktestRunStatus.RUNNING) return t('ai.backtestScoreCard.status.running');
    if (n === BacktestRunStatus.PENDING) return t('ai.backtestScoreCard.status.pending');
    if (n === BacktestRunStatus.FAILED) return t('ai.backtestScoreCard.status.failed');
    if (n === BacktestRunStatus.CANCEL_REQUESTED) return t('ai.backtestScoreCard.status.cancelRequested');
    if (n === BacktestRunStatus.CANCELED) return t('ai.backtestScoreCard.status.canceled');
  }
  if (s === 'SUCCEEDED') return t('ai.backtestScoreCard.status.succeeded');
  if (s === 'RUNNING') return t('ai.backtestScoreCard.status.running');
  if (s === 'PENDING') return t('ai.backtestScoreCard.status.pending');
  if (s === 'FAILED') return t('ai.backtestScoreCard.status.failed');
  if (s === 'CANCEL_REQUESTED') return t('ai.backtestScoreCard.status.cancelRequested');
  if (s === 'CANCELED') return t('ai.backtestScoreCard.status.canceled');
  return s || '-';
}

function num(v: any): number | undefined {
  if (v === null || v === undefined) return undefined;
  if (typeof v === 'number' && Number.isFinite(v)) return v;
  if (typeof v === 'string' && v.trim() !== '') {
    const n = Number(v);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
}

function pickMetric(metrics: any, keys: string[]): number | undefined {
  for (const k of keys) {
    const v = num(metrics?.[k]);
    if (typeof v === 'number') return v;
  }
  return undefined;
}

export default function BacktestScoreCard(props: {
  runId: string;
  status: string;
  error: string;
  metrics?: any;
  equityPoints?: number;
  equityCurve?: number[];
  risk?: {
    loading?: boolean;
    score?: number;
    level?: string;
    isReliable?: boolean;
    reasons?: string[];
    warnings?: string[];
  };
}) {
  const { t } = useTranslation();
  const { runId: _runId, status, error, metrics, equityPoints, equityCurve, risk } = props;

  const totalReturn = pickMetric(metrics, ['totalReturn', 'total_return']);
  const annualReturn = pickMetric(metrics, ['annualReturn', 'annual_return']);
  const maxDrawdown = pickMetric(metrics, ['maxDrawdown', 'max_drawdown']);
  const sharpe = pickMetric(metrics, ['sharpeRatio', 'sharpe_ratio']);
  const winRate = pickMetric(metrics, ['winRate', 'win_rate']);
  const totalTrades = pickMetric(metrics, ['totalTrades', 'total_trades']);

  const chartData = (equityCurve || []).map((v, i) => ({ i, equity: v }));

  return (
    <Card size="small" title={t('ai.backtestScoreCard.title')}>
      <div className="text-sm text-gray-600">
        {t('ai.backtestScoreCard.stateLabel')}：{backtestStatusLabel(status, t)}
      </div>
      {error ? <Alert className="mt-2" type="error" showIcon message={error} /> : null}

      <div className="mt-3">
        <div className="text-sm text-gray-700">{t('ai.backtestScoreCard.backendRiskScore.title')}</div>
        {risk?.loading ? (
          <div className="text-xs text-gray-500 mt-1">{t('ai.backtestScoreCard.backendRiskScore.loading')}</div>
        ) : risk && typeof risk.score === 'number' ? (
          <div className="mt-2">
            <div className="flex items-center gap-2">
              <Progress percent={risk.score} size="small" />
              <Tag color={risk.level === 'low' ? 'green' : risk.level === 'medium' ? 'orange' : 'red'}>
                {risk.level || t('ai.backtestScoreCard.backendRiskScore.unknown')}
              </Tag>
              <Tag color={risk.isReliable ? 'green' : 'red'}>
                {risk.isReliable
                  ? t('ai.backtestScoreCard.backendRiskScore.reliable')
                  : t('ai.backtestScoreCard.backendRiskScore.unreliable')}
              </Tag>
            </div>
            {(risk.reasons?.length || risk.warnings?.length) ? (
              <div className="mt-2">
                {risk.reasons?.length ? (
                  <div className="text-xs text-gray-700 whitespace-pre-wrap">
                    {t('ai.backtestScoreCard.backendRiskScore.reasons')}：{risk.reasons.slice(0, 6).join('\n')}
                  </div>
                ) : null}
                {risk.warnings?.length ? (
                  <div className="text-xs text-gray-500 whitespace-pre-wrap mt-1">
                    {t('ai.backtestScoreCard.backendRiskScore.warnings')}：{risk.warnings.slice(0, 6).join('\n')}
                  </div>
                ) : null}
              </div>
            ) : null}
          </div>
        ) : (
          <div className="text-xs text-gray-500 mt-1">{t('ai.backtestScoreCard.backendRiskScore.empty')}</div>
        )}
      </div>

      {metrics ? (
        <Descriptions className="mt-3" size="small" column={2} bordered>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.totalReturn')}>
            {typeof totalReturn === 'number' ? `${(totalReturn * 100).toFixed(2)}%` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.annualReturn')}>
            {typeof annualReturn === 'number' ? `${(annualReturn * 100).toFixed(2)}%` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.maxDrawdown')}>
            {typeof maxDrawdown === 'number' ? `${(maxDrawdown * 100).toFixed(2)}%` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.sharpe')}>
            {typeof sharpe === 'number' ? sharpe.toFixed(3) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.winRate')}>
            {typeof winRate === 'number' ? `${(winRate * 100).toFixed(2)}%` : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.totalTrades')}>
            {typeof totalTrades === 'number' ? Math.round(totalTrades) : '-'}
          </Descriptions.Item>
          <Descriptions.Item label={t('ai.backtestScoreCard.metrics.equityPoints')}>{equityPoints ?? '-'}</Descriptions.Item>
        </Descriptions>
      ) : null}

      {chartData.length >= 2 ? (
        <div className="mt-3" style={{ width: '100%', height: 220, minWidth: 0, minHeight: 220 }}>
          <div className="text-sm text-gray-700 mb-2">{t('ai.backtestScoreCard.chart.title')}</div>
          <ResponsiveContainer width="100%" height={220} minWidth={0} minHeight={220}>
            <LineChart data={chartData} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
              <XAxis dataKey="i" tick={false} />
              <YAxis tick={{ fontSize: 10 }} width={48} domain={['auto', 'auto']} />
              <Tooltip formatter={(v: any) => (typeof v === 'number' ? v.toFixed(2) : String(v))} />
              <Line type="monotone" dataKey="equity" stroke="#1677ff" dot={false} strokeWidth={2} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      ) : null}
    </Card>
  );
}
