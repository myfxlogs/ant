import { useState, useCallback, useRef, useEffect } from 'react';
import { Button, Typography, Spin, Popconfirm, Input, Checkbox } from 'antd';
import { DeleteOutlined, CheckOutlined, CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { SIDEBAR_DOUBLE_CLICK_RENAME_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

interface StrategyItem {
  id: string;
  name: string;
}

interface Props {
  templates: StrategyItem[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  onDeleteTemplate?: (id: string) => void;
  onRenameTemplate?: (id: string, name: string) => void;
  onBatchDeleteTemplates?: (ids: string[]) => void;
}

export default function SidebarStrategyList({
  templates, loading, selectedId, onSelect, onDeleteTemplate, onRenameTemplate, onBatchDeleteTemplates,
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
    if (renamingId && renameValue.trim() && onRenameTemplate) {
      onRenameTemplate(renamingId, renameValue.trim());
    }
    setRenamingId(null);
  }, [renamingId, renameValue, onRenameTemplate]);

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
    if (checked.size === 0 || !onBatchDeleteTemplates) return;
    onBatchDeleteTemplates([...checked]);
    setChecked(new Set());
  }, [checked, onBatchDeleteTemplates]);

  if (loading) return <Spin size="small" style={{ margin: '8px 0' }} />;
  if (templates.length === 0) return (
    <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
      {t('strategy.workspace.sidebar.noStrategies', { defaultValue: 'No strategies yet' })}
    </Typography.Text>
  );

  return (
    <>
      {checked.size > 0 && (
        <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 4, padding: '4px 6px', background: 'var(--ant-color-fill-quaternary)', borderRadius: 4, flexShrink: 0 }}>
          <span style={{ fontSize: 11, flex: 1 }}>{checked.size} {t('common.selected', { defaultValue: 'selected' })}</span>
          <Popconfirm
            title={t('strategy.workspace.sidebar.batchDeleteConfirm', { defaultValue: 'Delete selected strategies?' })}
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
        {templates.map(tpl => (
          <div
            key={tpl.id}
            className="sidebar-item"
            onClick={() => onSelect(tpl.id)}
            onDoubleClick={(e) => { e.stopPropagation(); if (onRenameTemplate) startRename(tpl.id, tpl.name || ''); }}
            style={{
              padding: '6px 10px', borderRadius: 6, cursor: 'pointer', fontSize: 12,
              background: checked.has(tpl.id) ? 'var(--ant-color-primary-bg)' : (tpl.id === selectedId ? '#e6f4ff' : 'transparent'),
              border: checked.has(tpl.id) ? '1px solid var(--ant-color-primary-border)' : (tpl.id === selectedId ? '1px solid #91caff' : '1px solid transparent'),
              fontWeight: tpl.id === selectedId ? 600 : 400,
              display: 'flex', alignItems: 'center', gap: 4,
            }}
          >
            <span className="sidebar-check" style={{ flexShrink: 0, opacity: checked.size > 0 ? 1 : undefined, transition: 'opacity 0.15s' }}
              onClick={(e) => onBatchDeleteTemplates ? toggleCheck(tpl.id, e) : undefined}>
              <Checkbox checked={checked.has(tpl.id)} size="small" />
            </span>
            {renamingId === tpl.id ? (
              <Input
                ref={renameInputRef as never}
                size="small"
                value={renameValue}
                onChange={e => setRenameValue(e.target.value)}
                onPressEnter={commitRename}
                onBlur={commitRename}
                onKeyDown={e => { if (e.key === 'Escape') setRenamingId(null); }}
                style={{ flex: 1, fontSize: 12 }}
              />
            ) : (
              <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}
                title={t(SIDEBAR_DOUBLE_CLICK_RENAME_KEY)}>
                {tpl.name || tpl.id}
              </span>
            )}
            {renamingId === tpl.id && (
              <Button type="text" size="small" icon={<CheckOutlined />} onClick={(e) => { e.stopPropagation(); commitRename(); }} style={{ flexShrink: 0, padding: '0 2px' }} />
            )}
            {renamingId !== tpl.id && onDeleteTemplate && (
              <Popconfirm
                title={t('strategy.workspace.sidebar.deleteStrategyConfirm', { defaultValue: 'Delete this strategy?' })}
                onConfirm={(e) => { e?.stopPropagation(); onDeleteTemplate(tpl.id); }}
                okText={t('common.yes', { defaultValue: 'Yes' })}
                cancelText={t('common.no', { defaultValue: 'No' })}
              >
                <Button type="text" size="small" danger icon={<DeleteOutlined />}
                  onClick={(e) => e.stopPropagation()} style={{ flexShrink: 0, padding: '0 2px' }} className="sidebar-item-action" />
              </Popconfirm>
            )}
          </div>
        ))}
      </div>
    </>
  );
}
