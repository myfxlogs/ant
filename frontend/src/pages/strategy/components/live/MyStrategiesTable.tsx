import { useMemo, useState, useCallback } from 'react';
import { Table, Tag, Typography, Space, Button, Tooltip, Empty, Badge, Dropdown, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { PlayCircleOutlined, PauseCircleOutlined, ThunderboltOutlined, EditOutlined, FileTextOutlined, HeartOutlined, DeleteOutlined, MoreOutlined, CopyOutlined, RightOutlined, DownOutlined, CodeOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ScheduleRow, TemplateOption, AccountRow } from '../../hooks/libraryTypes';
import type { ActiveStrategy } from '@/gen/ant/v1/strategy_runtime_pb';
import { MODE_COLORS } from '../../LiveStrategyPageSignalDrawer';
import ScheduleExpandedRow from './ScheduleExpandedRow';
import OrphanRunsTable from './OrphanRunsTable';
import { secondsSince, formatAgo } from './timeHelpers';
import { isLogButtonDisabled, isHealthButtonDisabled } from './strategyJoin';
import type { JoinedRow } from './strategyJoin';
import { formatMode } from './formatMode';


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

  const columns: ColumnsType<JoinedRow> = [
    {
      title: t('strategy.live.strategyName', { defaultValue: 'Strategy' }),
      dataIndex: 'name', key: 'name', width: 140,
      render: (v: string, row: JoinedRow) => (
        <Space direction="vertical" size={0}>
          <Text strong>{v || templateById.get(row.templateId)?.name || row.templateId}</Text>
          <Space size={2}>
            <Text type="secondary" style={{ fontSize: 11 }}>{row.id.slice(0, 8)}</Text>
            <Tooltip title={t('common.copy', { defaultValue: 'Copy' })}>
              <Button type="text" size="small" icon={<CopyOutlined style={{ fontSize: 11 }} />}
                style={{ padding: 0, height: 16, minWidth: 16 }}
                onClick={() => {
                  const text = row.id;
                  if (navigator.clipboard?.writeText) {
                    void navigator.clipboard.writeText(text).then(
                      () => message.success(t('common.copied', { defaultValue: 'Copied' })),
                      () => fallbackCopy(text),
                    );
                  } else {
                    fallbackCopy(text);
                  }
                }} />
            </Tooltip>
          </Space>
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
            {a.signalCount > 0 && <Text style={{ fontSize: 11, textDecoration: 'none' }} type={type}>{formatAgo(a.lastSignalAt)}</Text>}
          </Space>
        );
      },
    },
    {
      title: t('strategy.live.pnl', { defaultValue: 'PnL' }),
      key: 'pnl', width: 90,
      render: (_: unknown, row: JoinedRow) => {
        const a = row.active;
        if (!a) return <Text type="secondary">-</Text>;
        const pnl = a.pnl || '0';
        const n = Number(pnl);
        const color = n >= 0 ? 'success' : 'danger';
        return <Text style={{ fontSize: 12 }} type={color}>{n >= 0 ? `+${pnl}` : pnl}</Text>;
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
        ? <Tag color={MODE_COLORS[row.active.mode] || 'default'}>{formatMode(row.active.mode, t)}</Tag>
        : <Text type="secondary">-</Text>,
    },
    {
      title: t('common.action', { defaultValue: 'Actions' }), key: 'actions', width: 120, fixed: 'right',
      render: (_: unknown, row: JoinedRow) => {
        const menuItems = [
          { key: 'run', icon: <ThunderboltOutlined />, label: t('strategy.schedules.actions.runNow', { defaultValue: 'Run Now' }), disabled: !row.isActive },
          { key: 'editParams', icon: <EditOutlined />, label: t('strategy.live.editParams', { defaultValue: 'Edit Parameters' }) },
          { key: 'editCode', icon: <CodeOutlined />, label: t('strategy.live.editStrategy', { defaultValue: 'Edit Strategy' }) },
          { key: 'logs', icon: <FileTextOutlined />, label: t('strategy.live.logs', { defaultValue: 'Logs' }), disabled: isLogButtonDisabled(row.id) },
          { key: 'health', icon: <HeartOutlined />, label: t('strategy.live.health', { defaultValue: 'Health' }), disabled: isHealthButtonDisabled(row.id) },
          ...(row.active || row.isActive ? [{ type: 'divider' as const }, { key: 'stop', icon: <PauseCircleOutlined />, label: t('strategy.live.stopAndDisable', { defaultValue: 'Stop & Disable' }), danger: true }] : []),
          { type: 'divider' as const },
          { key: 'delete', icon: <DeleteOutlined />, label: t('common.delete', { defaultValue: 'Delete' }), danger: true, disabled: !row.id },
        ];
        const handleMenu = ({ key }: { key: string }) => {
          if (key === 'run') onManualTrigger(row);
          if (key === 'editParams') onEditParams(row);
          else if (key === 'editCode') onEdit(row);
          else if (key === 'logs') onShowLogs(row.id);
          else if (key === 'health') onHealthCheck(row);
          else if (key === 'stop') {
            if (row.active) onStop(row.active.runId);
            if (row.isActive) onToggleActive(row, false);
          }
          else if (key === 'delete') onDelete(row);
        };
        return (
          <Space size="small">
            <Tooltip title={row.isActive ? t('common.disable', { defaultValue: 'Disable' }) : t('common.enable', { defaultValue: 'Enable' })}>
              <Button size="small" type="text" icon={row.isActive ? <PauseCircleOutlined /> : <PlayCircleOutlined />}
                onClick={() => onToggleActive(row, !row.isActive)} />
            </Tooltip>
            <Tooltip title={t('strategy.schedules.actions.runNow', { defaultValue: 'Run Now' })}>
              <Button size="small" type="text" icon={<ThunderboltOutlined />} disabled={!row.isActive}
                onClick={() => onManualTrigger(row)} />
            </Tooltip>
            <Dropdown menu={{ items: menuItems, onClick: handleMenu }} trigger={['click']} placement="bottomRight">
              <Button size="small" type="text" icon={<MoreOutlined />} />
            </Dropdown>
          </Space>
        );
      },
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
        scroll={{ y: 400 }}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: ['10', '20', '50'], showTotal: (total) => `${total}` }}
        rowClassName={(row) => row.id === highlightScheduleId ? 'schedule-row-highlight' : ''}
        expandable={{
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
