import { Alert, Descriptions, Table, Typography } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next'
import { FIELDS_ERROR_KEY, FIELDS_MAX_DRAWDOWN_KEY, FIELDS_SHARPE_KEY, FIELDS_STATUS_KEY, HINTS_CANCELING_KEY, HINTS_QUEUED_KEY, HINTS_RUNNING_KEY, METRICS_ANNUAL_RETURN_KEY, METRICS_TOTAL_RETURN_KEY, TRADES_TITLE_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_run_keys';

;
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import type { BacktestTrade } from '@/client/backtestRuns';

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
          {watched.run?.status === 1 ? <Alert type="info" title={t(HINTS_QUEUED_KEY)} /> : null}
          {watched.run?.status === 2 ? <Alert type="info" title={t(HINTS_RUNNING_KEY)} /> : null}
          {watched.run?.status === 5 ? <Alert type="warning" title={t(HINTS_CANCELING_KEY)} /> : null}
          <Descriptions size="small" column={1} bordered>
            <Descriptions.Item label={t(FIELDS_STATUS_KEY)}>{statusText}</Descriptions.Item>
            <Descriptions.Item label={t(FIELDS_ERROR_KEY)}>{watched.run?.error || '-'}</Descriptions.Item>
          </Descriptions>
          <div className="mt-4" />
          <Descriptions size="small" column={2} bordered>
            <Descriptions.Item label={t(METRICS_TOTAL_RETURN_KEY)}>{isCompleted ? watched.metrics?.totalReturn ?? '-' : '-'}</Descriptions.Item>
            <Descriptions.Item label={t(METRICS_ANNUAL_RETURN_KEY)}>{isCompleted ? watched.metrics?.annualReturn ?? '-' : '-'}</Descriptions.Item>
            <Descriptions.Item label={t(FIELDS_MAX_DRAWDOWN_KEY)}>{isCompleted ? watched.metrics?.maxDrawdown ?? '-' : '-'}</Descriptions.Item>
            <Descriptions.Item label={t(FIELDS_SHARPE_KEY)}>{isCompleted ? watched.metrics?.sharpeRatio ?? '-' : '-'}</Descriptions.Item>
          </Descriptions>
          <div className="mt-4" />
          <Typography.Text strong>{t(TRADES_TITLE_KEY)}</Typography.Text>
          {summary && <div className="text-xs mt-1 mb-2" style={{ color: 'var(--color-text-muted)' }}>{summary}</div>}
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
