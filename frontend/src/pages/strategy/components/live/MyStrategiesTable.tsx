import { useMemo, useState, useCallback } from 'react';
import { Table, Empty, message } from 'antd';
import type { TemplateOption, AccountRow, ScheduleRow } from '../../hooks/libraryTypes';
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import ScheduleExpandedRow from './ScheduleExpandedRow';
import OrphanRunsTable from './OrphanRunsTable';
import type { JoinedRow } from './strategyJoin';
import { LIVE_EXPAND_COL_WIDTH } from './strategyJoin';
import { useMyStrategiesColumns } from './myStrategiesColumns';
import { RightOutlined, DownOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface Props {
  schedules: JoinedRow[];
  orphanRuns: ActiveStrategy[];
  templates: TemplateOption[];
  accounts: AccountRow[];
  loading: boolean;
  activeVersion: number;
  highlightScheduleId?: string | null;
  onToggleActive: (row: ScheduleRow, next: boolean) => void;
  onManualTrigger: (row: ScheduleRow) => void;
  onEdit: (row: ScheduleRow) => void;
  onEditParams: (row: ScheduleRow) => void;
  onDelete: (row: ScheduleRow) => void;
  onShowLogs: (scheduleId: string) => void;
  onHealthCheck: (row: ScheduleRow) => void;
  onStop: (runId: string) => void;
  stopping: string | null;
}

export default function MyStrategiesTable({
  schedules, orphanRuns, templates, accounts, loading, activeVersion,
  highlightScheduleId, onToggleActive, onManualTrigger, onEdit, onEditParams, onDelete,
  onShowLogs, onHealthCheck, onStop, stopping,
}: Props) {
  const { t } = useTranslation();
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);

  const fallbackCopy = useCallback((text: string) => {
    const ta = document.createElement('textarea');
    ta.value = text;
    ta.style.position = 'fixed';
    ta.style.opacity = '0';
    document.body.appendChild(ta);
    ta.select();
    try {
      document.execCommand('copy');
      message.success(t('common.copied', { defaultValue: 'Copied' }));
    } catch {
      message.error(t('common.copyFailed', { defaultValue: 'Copy failed' }));
    }
    document.body.removeChild(ta);
  }, [t]);

  const templateById = useMemo(() => {
    const m = new Map<string, TemplateOption>();
    (templates || []).forEach(item => { if (item?.id) m.set(item.id, item); });
    return m;
  }, [templates]);

  const accountById = useMemo(() => {
    const m = new Map<string, AccountRow>();
    (accounts || []).forEach(item => { if (item?.id) m.set(item.id, item); });
    return m;
  }, [accounts]);

  const fmtAccount = useCallback((id: string) => {
    const a = accountById.get(id);
    return a?.login ? `${a.login} (${a.mtType || ''})` : id;
  }, [accountById]);

  const columns = useMyStrategiesColumns({
    templateById, fmtAccount, onToggleActive, onManualTrigger, onEdit, onEditParams,
    onDelete, onShowLogs, onHealthCheck, onStop, fallbackCopy,
  });

  return (
    <div>
      <Table<JoinedRow>
        size="small"
        dataSource={schedules}
        rowKey="id"
        loading={loading}
        columns={columns}
        scroll={{ y: 400 }}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: ['10', '20', '50'], showTotal: (total) => `${total}` }}
        rowClassName={(row) => row.id === highlightScheduleId ? 'schedule-row-highlight' : ''}
        expandable={{
          columnWidth: LIVE_EXPAND_COL_WIDTH,
          expandedRowKeys: expandedKeys,
          onExpand: (expanded, row) => setExpandedKeys(expanded ? [row.id] : []),
          expandedRowRender: (row) => <ScheduleExpandedRow row={row} activeVersion={activeVersion} liveBid={row.active?.bid} liveAsk={row.active?.ask} />,
          rowExpandable: (row) => !!row.id,
          expandIcon: ({ expanded, onExpand, record }) => (
            <span role="img" aria-label={expanded ? 'Collapse' : 'Expand'}
              onClick={(e) => onExpand(record, e)}
              style={{ cursor: 'pointer', color: 'var(--color-text-secondary)', display: 'inline-flex', alignItems: 'center', justifyContent: 'center', width: 16, height: 16 }}>
              {expanded ? <DownOutlined style={{ fontSize: 10 }} /> : <RightOutlined style={{ fontSize: 10 }} />}
            </span>
          ),
        }}
        locale={{ emptyText: (
          <Empty description={t('strategy.live.noActive', { defaultValue: 'No active strategies' })} />
        ) }}
      />
      <OrphanRunsTable orphanRuns={orphanRuns} onStop={onStop} stopping={stopping} />
    </div>
  );
}
