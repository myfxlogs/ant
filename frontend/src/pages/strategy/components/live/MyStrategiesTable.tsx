import { useMemo, useState, useCallback } from 'react';
import { Table, Tag, Typography, Space, Button, Tooltip, Popconfirm, Empty, Badge } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlayCircleOutlined, PauseCircleOutlined, ThunderboltOutlined, EditOutlined, FileTextOutlined, HeartOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ScheduleRow, TemplateOption, AccountRow } from '../../hooks/libraryTypes';
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import { MODE_COLORS } from '../../LiveStrategyPageSignalDrawer';
import ScheduleExpandedRow from './ScheduleExpandedRow';
import OrphanRunsTable from './OrphanRunsTable';
import { secondsSince, formatAgo } from './timeHelpers';
import { isLogButtonDisabled, isHealthButtonDisabled } from './strategyJoin';
import type { JoinedRow } from './strategyJoin';

const { Text } = Typography;

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
  onDelete: (row: ScheduleRow) => void;
  onShowLogs: (scheduleId: string) => void;
  onHealthCheck: (row: ScheduleRow) => void;
  onStop: (runId: string) => void;
  stopping: string | null;
}

export default function MyStrategiesTable({
  schedules, orphanRuns, templates, accounts, loading, activeVersion,
  highlightScheduleId, onToggleActive, onManualTrigger, onEdit, onDelete,
  onShowLogs, onHealthCheck, onStop, stopping,
}: Props) {
  const { t } = useTranslation();
  const [expandedKeys, setExpandedKeys] = useState<string[]>([]);

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

  const columns: ColumnsType<JoinedRow> = [
    {
      title: t('strategy.live.strategyName', { defaultValue: 'Strategy' }),
      dataIndex: 'name', key: 'name', width: 140,
      render: (v: string, row: JoinedRow) => (
        <Space direction="vertical" size={0}>
          <Text strong>{v || templateById.get(row.templateId)?.name || row.templateId}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{row.id}</Text>
        </Space>
      ),
    },
    {
      title: t('strategy.live.symbol', { defaultValue: 'Symbol' }),
      key: 'symbolTf', width: 100,
      render: (_: unknown, row: JoinedRow) => (
        <Space direction="vertical" size={0}>
          <Text>{row.symbol}</Text>
          <Text type="secondary" style={{ fontSize: 11 }}>{row.timeframe}</Text>
        </Space>
      ),
    },
    {
      title: t('strategy.live.account', { defaultValue: 'Account' }),
      dataIndex: 'accountId', key: 'accountId', width: 110,
      render: (v: string) => <Text style={{ fontSize: 12 }}>{fmtAccount(v)}</Text>,
    },
    {
      title: t('strategy.live.status', { defaultValue: 'Status' }),
      key: 'status', width: 100,
      render: (_: unknown, row: JoinedRow) => {
        if (row.isRunning && row.active) {
          return <Space><Badge status="success" /><Text>{t('strategy.schedules.status.running', { defaultValue: 'Running' })}</Text></Space>;
        }
        if (row.isActive) {
          return <Space><Badge status="warning" /><Text>{t('strategy.schedules.status.idle', { defaultValue: 'Idle' })}</Text></Space>;
        }
        return <Space><Badge status="default" /><Text type="secondary">{t('strategy.schedules.status.disabled', { defaultValue: 'Disabled' })}</Text></Space>;
      },
    },
    {
      title: t('strategy.live.price', { defaultValue: 'Price' }),
      key: 'price', width: 100,
      render: (_: unknown, row: JoinedRow) => {
        const a = row.active;
        if (!a || (!a.bid && !a.ask)) return <Text type="secondary">-</Text>;
        const stale = a.lastTickAt ? secondsSince(a.lastTickAt) > 60 : false;
        const spread = a.bid && a.ask ? ` / ${a.ask}` : '';
        return (
          <Space direction="vertical" size={0}>
            <Text style={{ fontSize: 12, fontVariantNumeric: 'tabular-nums' }} type={stale ? 'secondary' : undefined}>
              {a.bid}{spread}
            </Text>
            {stale && <Tag color="default" style={{ fontSize: 10 }}>{t('strategy.live.stale', { defaultValue: 'stale' })}</Tag>}
          </Space>
        );
      },
    },
    {
      title: t('strategy.live.signals', { defaultValue: 'Signals' }),
      key: 'signals', width: 110,
      render: (_: unknown, row: JoinedRow) => {
        const a = row.active;
        if (!a) return <Text type="secondary">-</Text>;
        const s = secondsSince(a.lastSignalAt);
        let type: 'secondary' | 'warning' | undefined = undefined;
        if (s > 15 * 60) type = 'secondary';
        else if (s > 300) type = 'warning';
        return (
          <Space direction="vertical" size={0}>
            <Text strong>{a.signalCount}</Text>
            <Text style={{ fontSize: 11 }} type={type}>{formatAgo(a.lastSignalAt)}</Text>
          </Space>
        );
      },
    },
    {
      title: t('strategy.live.pnl', { defaultValue: 'PnL' }),
      key: 'pnl', width: 90,
      render: (_: unknown, row: JoinedRow) => {
        const a = row.active;
        if (!a || !a.pnl) return <Text type="secondary">-</Text>;
        const n = Number(a.pnl);
        const color = n >= 0 ? 'success' : 'danger';
        return <Text style={{ fontSize: 12 }} type={color}>{n >= 0 ? `+${a.pnl}` : a.pnl}</Text>;
      },
    },
    {
      title: t('strategy.live.errors', { defaultValue: 'Errors' }),
      key: 'errors', width: 70,
      render: (_: unknown, row: JoinedRow) => {
        const a = row.active;
        if (!a || a.errorCount === 0) return <Text type="secondary">0</Text>;
        return (
          <Tooltip title={a.lastError || t('strategy.live.unknownError', { defaultValue: 'Unknown error' })}>
            <Tag color="red">{a.errorCount}</Tag>
          </Tooltip>
        );
      },
    },
    {
      title: t('strategy.live.mode', { defaultValue: 'Mode' }),
      key: 'mode', width: 60,
      render: (_: unknown, row: JoinedRow) => row.active
        ? <Tag color={MODE_COLORS[row.active.mode] || 'default'}>{row.active.mode}</Tag>
        : <Text type="secondary">-</Text>,
    },
    {
      title: '', key: 'actions', width: 200,
      render: (_: unknown, row: JoinedRow) => (
        <Space size="small">
          <Tooltip title={row.isActive ? t('common.disable', { defaultValue: 'Disable' }) : t('common.enable', { defaultValue: 'Enable' })}>
            <Button size="small" type="text" icon={row.isActive ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
              onClick={() => onToggleActive(row, !row.isActive)} />
          </Tooltip>
          <Tooltip title={t('strategy.schedules.actions.runNow', { defaultValue: 'Run Now' })}>
            <Button size="small" type="text" icon={<ThunderboltOutlined />} disabled={!row.isActive}
              onClick={() => onManualTrigger(row)} />
          </Tooltip>
          <Tooltip title={t('common.edit', { defaultValue: 'Edit' })}>
            <Button size="small" type="text" icon={<EditOutlined />} onClick={() => onEdit(row)} />
          </Tooltip>
          <Tooltip title={t('strategy.live.logs', { defaultValue: 'Logs' })}>
            <Button size="small" type="text" icon={<FileTextOutlined />} disabled={isLogButtonDisabled(row.id)}
              onClick={() => onShowLogs(row.id)} />
          </Tooltip>
          <Tooltip title={t('strategy.live.health', { defaultValue: 'Health' })}>
            <Button size="small" type="text" icon={<HeartOutlined />} disabled={isHealthButtonDisabled(row.id)}
              onClick={() => onHealthCheck(row)} />
          </Tooltip>
          {row.active && (
            <Popconfirm title={t('strategy.live.confirmStop', { defaultValue: 'Stop this strategy?' })}
              onConfirm={() => onStop(row.active!.runId)}>
              <Button size="small" type="text" danger loading={stopping === row.active.runId}
                icon={<PauseCircleOutlined />} />
            </Popconfirm>
          )}
          <Popconfirm title={t('strategy.schedules.deleteConfirm.title', { defaultValue: 'Delete this schedule?' })}
            onConfirm={() => onDelete(row)}>
            <Button size="small" type="text" danger icon={<DeleteOutlined />} disabled={!row.id} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Table<JoinedRow>
        size="small"
        dataSource={schedules}
        rowKey="id"
        loading={loading}
        columns={columns}
        pagination={false}
        rowClassName={(row) => row.id === highlightScheduleId ? 'schedule-row-highlight' : ''}
        expandable={{
          expandedRowKeys: expandedKeys,
          onExpand: (expanded, row) => setExpandedKeys(expanded ? [...expandedKeys, row.id] : expandedKeys.filter(k => k !== row.id)),
          expandedRowRender: (row) => <ScheduleExpandedRow row={row} activeVersion={activeVersion} />,
          rowExpandable: (row) => !!row.id,
        }}
        locale={{ emptyText: (
          <Empty description={t('strategy.live.noActive', { defaultValue: 'No active strategies' })} />
        ) }}
      />
      <OrphanRunsTable orphanRuns={orphanRuns} onStop={onStop} stopping={stopping} />
    </div>
  );
}
