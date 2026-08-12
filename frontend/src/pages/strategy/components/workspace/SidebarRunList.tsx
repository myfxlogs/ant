import { useState, useCallback, useRef, useEffect } from 'react';
import { Button, Typography, Spin, Popconfirm, Checkbox, Input } from 'antd';
import { DeleteOutlined, CloseOutlined, CheckOutlined, EditOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { SIDEBAR_DOUBLE_CLICK_RENAME_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

interface BacktestRun {
  id: string;
  startedAt?: string;
  totalReturn?: number;
  totalTrades?: number;
  templateName?: string;
  templateId?: string;
  name?: string;
}

interface Props {
  runs: BacktestRun[];
  loading: boolean;
  onOpenHistory: (runId?: string) => void;
  onDeleteRun?: (runId: string) => void;
  onBatchDeleteRuns?: (runIds: string[]) => void;
  onRenameRun?: (runId: string, name: string) => void;
}

function fmtReturn(v: number | undefined): string {
  if (v == null) return '—';
  return `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`;
}

export default function SidebarRunList({
  runs, loading, onOpenHistory, onDeleteRun, onBatchDeleteRuns, onRenameRun,
}: Props) {
  const { t } = useTranslation();
  const [checked, setChecked] = useState<Set<string>>(new Set());
  const [renamingId, setRenamingId] = useState<string | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const renameInputRef = useRef<{ focus: () => void; select: () => void }>(null);

  const startRename = useCallback((id: string, name: string) => {
    setRenamingId(id);
    setRenameValue(name);
  }, []);

  const commitRename = useCallback(() => {
    if (renamingId && renameValue.trim() && onRenameRun) {
      onRenameRun(renamingId, renameValue.trim());
    }
    setRenamingId(null);
  }, [renamingId, renameValue, onRenameRun]);

  useEffect(() => {
    if (renamingId) renameInputRef.current?.focus();
  }, [renamingId]);

  const toggleCheck = useCallback((id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setChecked(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }, []);

  const clearChecks = useCallback(() => setChecked(new Set()), []);

  const handleBatchDelete = useCallback(() => {
    if (checked.size === 0 || !onBatchDeleteRuns) return;
    onBatchDeleteRuns([...checked]);
    setChecked(new Set());
  }, [checked, onBatchDeleteRuns]);

  if (loading) return <Spin size="small" style={{ margin: '8px 0' }} />;
  if (runs.length === 0) return (
    <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
      {t('strategy.workspace.sidebar.noRuns', { defaultValue: 'No backtest runs yet' })}
    </Typography.Text>
  );

  return (
    <>
      {checked.size > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 4, padding: '4px 6px', background: 'var(--ant-color-fill-quaternary)', borderRadius: 4, flexShrink: 0 }}>
          <span style={{ fontSize: 11, flex: 1 }}>{checked.size} {t('common.selected', { defaultValue: 'selected' })}</span>
          <Popconfirm
            title={t('strategy.workspace.sidebar.batchDeleteRunsConfirm', { defaultValue: 'Delete selected runs?' })}
            onConfirm={handleBatchDelete}
            okText={t('common.yes', { defaultValue: 'Yes' })}
            cancelText={t('common.no', { defaultValue: 'No' })}
          >
            <Button type="text" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
          <Button type="text" size="small" icon={<CloseOutlined />} onClick={clearChecks} />
        </div>
      )}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 6, overflowY: 'auto', flex: '1 1 auto', minHeight: 0 }}>
        {runs.slice(0, 10).map(r => (
          <div
            key={r.id}
            className="sidebar-item"
            onClick={() => onOpenHistory(r.id)}
            onDoubleClick={(e) => { e.stopPropagation(); if (onRenameRun) startRename(r.id, r.name || r.templateName || ''); }}
            style={{
              padding: '5px 10px', borderRadius: 6, cursor: 'pointer', fontSize: 11,
              background: checked.has(r.id) ? 'var(--ant-color-primary-bg)' : 'transparent',
              border: checked.has(r.id) ? '1px solid var(--ant-color-primary-border)' : '1px solid transparent',
              display: 'flex', alignItems: 'center', gap: 4,
            }}
          >
            <span className="sidebar-check" style={{ flexShrink: 0, opacity: checked.size > 0 ? 1 : undefined, transition: 'opacity 0.15s' }}
              onClick={(e) => onBatchDeleteRuns ? toggleCheck(r.id, e) : undefined}>
              <Checkbox checked={checked.has(r.id)} />
            </span>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                {renamingId === r.id ? (
                  <Input
                    ref={renameInputRef as never}
                    size="small"
                    value={renameValue}
                    onChange={e => setRenameValue(e.target.value)}
                    onPressEnter={commitRename}
                    onBlur={commitRename}
                    onKeyDown={e => { if (e.key === 'Escape') setRenamingId(null); }}
                    style={{ flex: 1, fontSize: 11 }}
                  />
                ) : (
                  <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}
                    title={t(SIDEBAR_DOUBLE_CLICK_RENAME_KEY)}>
                    {r.name || r.templateName || r.id?.slice(0, 8)}
                  </span>
                )}
                {renamingId !== r.id && <span style={{
                  fontWeight: 700, fontSize: 11, flexShrink: 0, marginLeft: 6,
                  color: (r.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149',
                }}>
                  {r.totalReturn != null ? fmtReturn(r.totalReturn) : (
                    <Button type="text" size="small" icon={<EditOutlined />}
                      onClick={(e) => { e.stopPropagation(); if (onRenameRun) startRename(r.id, r.name || r.templateName || ''); }}
                      style={{ flexShrink: 0, padding: '0 2px', fontSize: 11 }} className="sidebar-item-action" />
                  )}
                </span>}
              </div>
              {r.totalTrades != null && (
                <div style={{ color: 'var(--ant-color-text-tertiary)', fontSize: 10 }}>
                  {r.totalTrades} {t('strategy.workspace.sidebar.trades', { defaultValue: 'trades' })}
                </div>
              )}
            </div>
            {renamingId === r.id && (
              <Button type="text" size="small" icon={<CheckOutlined />} onClick={(e) => { e.stopPropagation(); commitRename(); }} style={{ flexShrink: 0, padding: '0 2px' }} />
            )}
            {renamingId !== r.id && onDeleteRun && (
              <Popconfirm
                title={t('strategy.workspace.sidebar.deleteRunConfirm', { defaultValue: 'Delete this backtest run?' })}
                onConfirm={(e) => { e?.stopPropagation(); onDeleteRun(r.id); }}
                okText={t('common.yes', { defaultValue: 'Yes' })}
                cancelText={t('common.no', { defaultValue: 'No' })}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />}
                  onClick={(e) => e.stopPropagation()} style={{ flexShrink: 0, padding: '0 2px' }} className="sidebar-item-action" />
              </Popconfirm>
            )}
          </div>
        ))}
        {runs.length > 10 && (
          <Button size="small" type="link" onClick={() => onOpenHistory()} style={{ fontSize: 11, padding: 0 }}>
            {t('strategy.workspace.sidebar.viewAll', { defaultValue: 'View all' })}
          </Button>
        )}
      </div>
    </>
  );
}
