import { Space, Alert, Descriptions, Table, Tag, Typography } from 'antd';
import { useTranslation } from 'react-i18next'
import { HEALTH_FIELDS_CONFIG_KEY_KEY, HEALTH_FIELDS_FAILED_RUNS_KEY, HEALTH_FIELDS_GRADE_KEY, HEALTH_FIELDS_LAST_RUN_AT_KEY, HEALTH_FIELDS_LATEST_ERROR_KEY, HEALTH_FIELDS_LATEST_PROFIT_KEY, HEALTH_FIELDS_LATEST_TICKET_KEY, HEALTH_FIELDS_RULE_KEY, HEALTH_FIELDS_SUCCESS_OVER_TOTAL_KEY, HEALTH_FIELDS_THRESHOLDS_KEY, HEALTH_GRADE_ALERT_KEY, HEALTH_GRADE_HEALTHY_KEY, HEALTH_GRADE_NO_SAMPLE_KEY, HEALTH_GRADE_PENDING_KEY, HEALTH_GRADE_WATCH_KEY, HEALTH_MESSAGES_CLICK_REFRESH_KEY, HEALTH_NOTES_ALERT_KEY, HEALTH_NOTES_HEALTHY_KEY, HEALTH_NOTES_NO_SAMPLE_KEY, HEALTH_NOTES_PENDING_KEY, HEALTH_NOTES_WATCH_KEY, HEALTH_RUN_LOGS_SIGNAL_TYPE_KEY, HEALTH_SECTIONS_ORDERS_KEY, HEALTH_SECTIONS_RUN_LOGS_KEY, HEALTH_SUMMARY_BANNER_KEY, HEALTH_THRESHOLDS_SUMMARY_KEY } from '@/gen/ant/v1/i18n/strategy_schedules_keys';

;

const { Text } = Typography;

interface Props {
  summary: Record<string, unknown> | null;
  loading: boolean;
  formatTime: (v: unknown) => string;
}

function toNumber(v: unknown): number {
  if (typeof v === 'number') return v;
  if (typeof v === 'bigint') return Number(v);
  if (typeof v === 'string') { const n = Number(v); return isNaN(n) ? 0 : n; }
  return 0;
}

function getGrade(t: (key: string, opts?: Record<string, unknown>) => string, summary: Record<string, unknown> | null) {
  if (!summary) return { level: 'unknown', label: t(HEALTH_GRADE_PENDING_KEY), color: 'default', note: t(HEALTH_NOTES_PENDING_KEY) };
  const code = String(summary.gradeNoteCode || 'pending');
  if (code === 'no_sample') return { level: summary.gradeLevel as string, label: t(HEALTH_GRADE_NO_SAMPLE_KEY), color: summary.gradeColor as string, note: t(HEALTH_NOTES_NO_SAMPLE_KEY, { minSampleSize: summary.minSampleSize }) };
  if (code === 'healthy') return { level: summary.gradeLevel as string, label: t(HEALTH_GRADE_HEALTHY_KEY), color: summary.gradeColor as string, note: t(HEALTH_NOTES_HEALTHY_KEY, { greenSuccessRate: summary.greenSuccessRate, greenMaxFailedRuns: summary.greenMaxFailedRuns }) };
  if (code === 'watch') return { level: summary.gradeLevel as string, label: t(HEALTH_GRADE_WATCH_KEY), color: summary.gradeColor as string, note: t(HEALTH_NOTES_WATCH_KEY, { yellowSuccessRate: summary.yellowSuccessRate }) };
  return { level: summary.gradeLevel as string, label: t(HEALTH_GRADE_ALERT_KEY), color: summary.gradeColor as string, note: t(HEALTH_NOTES_ALERT_KEY) };
}

export default function ScheduleHealthContent({ summary, loading, formatTime }: Props) {
  const { t } = useTranslation();
  const grade = getGrade(t, summary);

  return (
    <Space direction="vertical" style={{ width: '100%' }} size={12}>
      <Alert
        type={grade.level === 'red' ? 'error' : grade.level === 'yellow' ? 'warning' : 'success'}
        showIcon
        message={summary ? t(HEALTH_SUMMARY_BANNER_KEY, { grade: grade.label, totalRuns: summary.totalRuns, successRate: Number(summary.successRate).toFixed(1) }) : t(HEALTH_MESSAGES_CLICK_REFRESH_KEY)}
        description={summary ? grade.note : undefined}
      />

      <Descriptions bordered size="small" column={2}>
        <Descriptions.Item label={t(HEALTH_FIELDS_GRADE_KEY)}><Tag color={grade.color}>{grade.label}</Tag></Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_RULE_KEY)}>{summary ? grade.note : '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_THRESHOLDS_KEY)}>
          {summary ? t(HEALTH_THRESHOLDS_SUMMARY_KEY, { minSampleSize: summary.minSampleSize, greenSuccessRate: summary.greenSuccessRate, greenMaxFailedRuns: summary.greenMaxFailedRuns, yellowSuccessRate: summary.yellowSuccessRate }) : '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_CONFIG_KEY_KEY)}><Text code>strategy.schedule.health_grading_config</Text></Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_LAST_RUN_AT_KEY)}>{summary ? formatTime(summary.lastRunAt) : '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_LATEST_TICKET_KEY)}>{summary?.latestOrderTicket as string || '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_SUCCESS_OVER_TOTAL_KEY)}>{summary ? `${summary.successRuns}/${summary.totalRuns}` : '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_FAILED_RUNS_KEY)}>{summary ? (summary.failedRuns as number) : '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_LATEST_PROFIT_KEY)}>{summary?.latestOrderProfit != null ? (summary.latestOrderProfit as number).toFixed(2) : '-'}</Descriptions.Item>
        <Descriptions.Item label={t(HEALTH_FIELDS_LATEST_ERROR_KEY)}>{(summary?.latestError as string) || '-'}</Descriptions.Item>
      </Descriptions>

      <Text strong>{t(HEALTH_SECTIONS_RUN_LOGS_KEY)}</Text>
      <Table scroll={{ x: 'max-content' }} rowKey={(row) => String(row?.id || '')} size="small" loading={loading} pagination={false}
        dataSource={(summary?.runLogs || []) as Record<string, unknown>[]}
        columns={[
          { title: t(EXEC_TABLE_TIME_KEY), key: 'createdAt', width: 180, render: (_: unknown, row: { createdAt?: string | Date }) => formatTime(row?.createdAt) },
          { title: t(EXEC_TABLE_STATUS_KEY), dataIndex: 'status', key: 'status', width: 120 },
          { title: t(HEALTH_RUN_LOGS_SIGNAL_TYPE_KEY), dataIndex: 'signalType', key: 'signalType', width: 120 },
          { title: t(EXEC_TABLE_DURATION_MS_KEY), dataIndex: 'durationMs', key: 'durationMs', width: 110, render: (v: unknown) => toNumber(v) },
          { title: t(EXEC_TABLE_ERROR_KEY), dataIndex: 'errorMessage', key: 'errorMessage', render: (v: unknown) => String(v || '-') },
        ]}
      />

      <Text strong>{t(HEALTH_SECTIONS_ORDERS_KEY)}</Text>
      <Table scroll={{ x: 'max-content' }} rowKey={(row) => String(row?.id || row?.ticket || '')} size="small" loading={loading} pagination={false}
        dataSource={(summary?.orders || []) as Record<string, unknown>[]}
        columns={[
          { title: t(ORDERS_TABLE_TIME_KEY), key: 'time', width: 180, render: (_: unknown, row: Record<string, unknown>) => formatTime(row?.closeTime || row?.openTime) },
          { title: t(ORDERS_TABLE_TICKET_KEY), dataIndex: 'ticket', key: 'ticket', width: 110 },
          { title: t(ORDERS_TABLE_SIDE_KEY), dataIndex: 'orderType', key: 'orderType', width: 110 },
          { title: t(ORDERS_TABLE_SYMBOL_KEY), dataIndex: 'symbol', key: 'symbol', width: 120 },
          { title: t(ORDERS_TABLE_PROFIT_KEY), dataIndex: 'profit', key: 'profit', width: 100, render: (v: unknown) => toNumber(v).toFixed(2) },
        ]}
      />
    </Space>
  );
}
