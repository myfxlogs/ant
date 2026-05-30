import { Alert, Descriptions, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import type { BacktestTrade, BacktestTradeSummary } from '@/client/backtestRuns';

const fmt = (n: number | null | undefined, digits = 4): string =>
  n === null || n === undefined || Number.isNaN(n) ? '-' : Number(n).toFixed(digits);

const fmtTs = (ms: number | undefined): string =>
  !ms || ms <= 0 ? '-' : dayjs(ms).format('YYYY-MM-DD HH:mm:ss');

interface ContentProps {
  watched: {
    run?: { status?: number; error?: string } | null;
    metrics?: { totalReturn?: string; annualReturn?: string } | null;
    loading?: boolean;
    error?: string | null;
    isTerminal?: boolean;
  };
  statusText: string;
  trades: BacktestTrade[];
  summary: string | null;
  tradesLoading: boolean;
  tradesError: string | null;
  columns: ColumnsType<BacktestTrade>;
}

export default function BacktestRunDrawerContent({ watched, statusText, trades, summary, tradesLoading, tradesError, columns }: ContentProps) {
  const { t } = useTranslation();
  const isCompleted = isSucceededRun(watched.run);
  
  return (
    <>
      {watched.loading ? (
        <Alert type="info" title={t('common.loading')} />
      ) : watched.error ? (
        <Alert type="error" title={watched.error} />
      ) : (
        <>
          {watched.run?.status === 1 ? <Alert type="info" title={t('strategy.backtestRun.hints.queued')} /> : null}
          {watched.run?.status === 2 ? <Alert type="info" title={t('strategy.backtestRun.hints.running')} /> : null}
          {watched.run?.status === 5 ? <Alert type="warning" title={t('strategy.backtestRun.hints.canceling')} /> : null}
          <Descriptions size="small" column={1} bordered>
            <Descriptions.Item label={t('strategy.backtestRun.fields.status')}>{statusText}</Descriptions.Item>
            <Descriptions.Item label={t('strategy.backtestRun.fields.error')}>{watched.run?.error || '-'}</Descriptions.Item>
          </Descriptions>
          <div className="mt-4" />
          <Descriptions size="small" column={2} bordered>
            <Descriptions.Item label={t('strategy.backtestRun.metrics.totalReturn')}>{isCompleted ? watched.metrics?.totalReturn ?? '-' : '-'}</Descriptions.Item>
            <Descriptions.Item label={t('strategy.backtestRun.metrics.annualReturn')}>{isCompleted ? watched.metrics?.annualReturn ?? '-' : '-'}</Descriptions.Item>
            <Descriptions.Item label={t('strategy.backtestRun.fields.maxDrawdown')}>-</Descriptions.Item>
            <Descriptions.Item label={t('strategy.backtestRun.fields.sharpe')}>-</Descriptions.Item>
          </Descriptions>
          <div className="mt-4" />
          <Typography.Text strong>{t('strategy.backtestRun.trades.title')}</Typography.Text>
          {summary && <div className="text-xs mt-1 mb-2" style={{ color: '#8A9AA5' }}>{summary}</div>}
          {tradesError ? (
            <Alert type="error" title={tradesError} />
          ) : (
            <Table<BacktestTrade> rowKey="ticket" size="small" bordered loading={tradesLoading} dataSource={trades} columns={columns} scroll={{ x: 'max-content' }} pagination={false} />
          )}
        </>
      )}
    </>
  );
}
