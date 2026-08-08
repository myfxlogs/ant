import { useState } from 'react';
import { Button, Typography, Spin, Popconfirm } from 'antd';
import { PlusOutlined, ImportOutlined, FileTextOutlined, HistoryOutlined, CaretLeftOutlined, DownOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

interface StrategyItem {
  id: string;
  name: string;
}

interface BacktestRun {
  id: string;
  startedAt?: string;
  totalReturn?: number;
  totalTrades?: number;
  templateName?: string;
  templateId?: string;
}

interface Props {
  templates: StrategyItem[];
  loading: boolean;
  selectedId: string;
  onSelect: (id: string) => void;
  onDeleteTemplate?: (id: string) => void;
  backtestRuns: BacktestRun[];
  runsLoading: boolean;
  onOpenHistory: (templateId?: string) => void;
  onDeleteRun?: (runId: string) => void;
  onImport: () => void;
  onNew: () => void;
  collapsed: boolean;
  onToggle: () => void;
  autoExpandHistory?: boolean;
}

function fmtReturn(v: number | undefined): string {
  if (v == null) return '—';
  return `${v >= 0 ? '+' : ''}${v.toFixed(1)}%`;
}

export default function WorkspaceSidebar({
  templates, loading, selectedId, onSelect, onDeleteTemplate,
  backtestRuns, runsLoading, onOpenHistory, onDeleteRun,
  onImport, onNew,
  collapsed, onToggle,
  autoExpandHistory,
}: Props) {
  const { t } = useTranslation();
  const [strategiesExpanded, setStrategiesExpanded] = useState(true);
  const [historyExpanded, setHistoryExpanded] = useState(false);

  // Auto-expand history section when triggered by external event (e.g. backtest completion)
  const [prevAutoExpand, setPrevAutoExpand] = useState(false);
  if (autoExpandHistory && !prevAutoExpand) {
    setPrevAutoExpand(true);
    setStrategiesExpanded(false);
    setHistoryExpanded(true);
  } else if (!autoExpandHistory && prevAutoExpand) {
    setPrevAutoExpand(false);
  }

  return (
    <div style={{
      width: collapsed ? 36 : 240, flexShrink: 0, overflow: 'hidden',
      borderRight: '1px solid var(--ant-color-border)',
      background: 'var(--ant-color-bg-container)',
      display: 'flex', flexDirection: 'column',
      transition: 'width 0.2s',
    }}>
      {/* Header */}
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: collapsed ? '8px 6px' : '8px 12px',
        borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0,
      }}>
        {!collapsed && (
          <span style={{ fontSize: 12, fontWeight: 600 }}>
            {t('strategy.workspace.sidebar.title', { defaultValue: 'Workspace' })}
          </span>
        )}
        <Button size="small" type="text" icon={<CaretLeftOutlined style={{ transform: collapsed ? 'rotate(180deg)' : undefined }} />}
          onClick={onToggle} style={{ padding: '0 4px' }} />
      </div>

      {!collapsed && (
        <div style={{ flex: '1 1 0', overflow: 'hidden', display: 'flex', flexDirection: 'column', padding: '8px 10px' }}>
          {/* My Strategies — fixed-height section */}
          <div style={{ flex: '0 0 auto', maxHeight: '40%', display: 'flex', flexDirection: 'column', marginBottom: 8, overflow: 'hidden' }}>
            <Button
              size="small"
              icon={<FileTextOutlined />}
              onClick={() => setStrategiesExpanded(v => !v)}
              block
              style={{ justifyContent: 'space-between', display: 'flex', alignItems: 'center', flexShrink: 0 }}
            >
              <span>{t('strategy.workspace.sidebar.myStrategies', { defaultValue: 'My Strategies' })}</span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {!strategiesExpanded && templates.length > 0 && <span style={{ fontSize: 10 }}>{templates.length}</span>}
                <DownOutlined style={{ fontSize: 10, transform: strategiesExpanded ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
              </span>
            </Button>
            {strategiesExpanded && (
              loading ? (
                <Spin size="small" style={{ margin: '8px 0' }} />
              ) : templates.length === 0 ? (
                <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
                  {t('strategy.workspace.sidebar.noStrategies', { defaultValue: 'No strategies yet' })}
                </Typography.Text>
              ) : (
                <div style={{
                  display: 'flex', flexDirection: 'column', gap: 2,
                  marginTop: 6, overflowY: 'auto', flex: '1 1 auto', minHeight: 0,
                }}>
                  {templates.map(tpl => (
                    <div
                      key={tpl.id}
                      onClick={() => onSelect(tpl.id)}
                      style={{
                        padding: '6px 10px', borderRadius: 6, cursor: 'pointer', fontSize: 12,
                        background: tpl.id === selectedId ? '#e6f4ff' : 'transparent',
                        border: tpl.id === selectedId ? '1px solid #91caff' : '1px solid transparent',
                        fontWeight: tpl.id === selectedId ? 600 : 400,
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 4,
                      }}
                    >
                      <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                        {tpl.name || tpl.id}
                      </span>
                      {onDeleteTemplate && (
                        <Popconfirm
                          title={t('strategy.workspace.sidebar.deleteStrategyConfirm', { defaultValue: 'Delete this strategy?' })}
                          onConfirm={(e) => { e?.stopPropagation(); onDeleteTemplate(tpl.id); }}
                          okText={t('common.yes', { defaultValue: 'Yes' })}
                          cancelText={t('common.no', { defaultValue: 'No' })}
                        >
                          <Button
                            type="text" size="small" danger
                            icon={<DeleteOutlined />}
                            onClick={(e) => e.stopPropagation()}
                            style={{ flexShrink: 0, padding: '0 2px' }}
                          />
                        </Popconfirm>
                      )}
                    </div>
                  ))}
                </div>
              )
            )}
          </div>

          {/* Backtest History — fixed-height section */}
          <div style={{ flex: '1 1 auto', display: 'flex', flexDirection: 'column', overflow: 'hidden', minHeight: 0 }}>
            <Button
              size="small"
              icon={<HistoryOutlined />}
              onClick={() => setHistoryExpanded(v => !v)}
              block
              style={{ justifyContent: 'space-between', display: 'flex', alignItems: 'center', flexShrink: 0 }}
            >
              <span>{t('strategy.workspace.sidebar.backtestHistory', { defaultValue: 'Backtest History' })}</span>
              <span style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
                {!historyExpanded && backtestRuns.length > 0 && <span style={{ fontSize: 10 }}>{backtestRuns.length}</span>}
                <DownOutlined style={{ fontSize: 10, transform: historyExpanded ? 'rotate(180deg)' : 'none', transition: 'transform 0.15s' }} />
              </span>
            </Button>
            {historyExpanded && (
              runsLoading ? (
                <Spin size="small" style={{ margin: '8px 0' }} />
              ) : backtestRuns.length === 0 ? (
                <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 6 }}>
                  {t('strategy.workspace.sidebar.noRuns', { defaultValue: 'No backtest runs yet' })}
                </Typography.Text>
              ) : (
                <div style={{ display: 'flex', flexDirection: 'column', gap: 2, marginTop: 6, overflowY: 'auto', flex: '1 1 auto', minHeight: 0 }}>
                  {backtestRuns.slice(0, 10).map(r => (
                    <div
                      key={r.id}
                      onClick={() => onOpenHistory(r.templateId)}
                      style={{
                        padding: '5px 10px', borderRadius: 6, cursor: 'pointer', fontSize: 11,
                        background: 'transparent', border: '1px solid transparent',
                        display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 4,
                      }}
                    >
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                          <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', flex: 1 }}>
                            {r.templateName || r.id?.slice(0, 8)}
                          </span>
                          <span style={{
                            fontWeight: 700, fontSize: 11, flexShrink: 0, marginLeft: 6,
                            color: (r.totalReturn ?? 0) >= 0 ? '#3fb950' : '#f85149',
                          }}>
                            {fmtReturn(r.totalReturn)}
                          </span>
                        </div>
                        {r.totalTrades != null && (
                          <div style={{ color: 'var(--ant-color-text-tertiary)', fontSize: 10 }}>
                            {r.totalTrades} {t('strategy.workspace.sidebar.trades', { defaultValue: 'trades' })}
                          </div>
                        )}
                      </div>
                      {onDeleteRun && (
                        <Popconfirm
                          title={t('strategy.workspace.sidebar.deleteRunConfirm', { defaultValue: 'Delete this backtest run?' })}
                          onConfirm={(e) => { e?.stopPropagation(); onDeleteRun(r.id); }}
                          okText={t('common.yes', { defaultValue: 'Yes' })}
                          cancelText={t('common.no', { defaultValue: 'No' })}
                        >
                          <Button
                            type="text" size="small" danger
                            icon={<DeleteOutlined />}
                            onClick={(e) => e.stopPropagation()}
                            style={{ flexShrink: 0, padding: '0 2px' }}
                          />
                        </Popconfirm>
                      )}
                    </div>
                  ))}
                  {backtestRuns.length > 10 && (
                    <Button size="small" type="link" onClick={() => onOpenHistory()}
                      style={{ fontSize: 11, padding: 0 }}>
                      {t('strategy.workspace.sidebar.viewAll', { defaultValue: 'View all' })}
                    </Button>
                  )}
                </div>
              )
            )}
          </div>
        </div>
      )}

      {/* Action buttons */}
      <div style={{
        padding: collapsed ? '6px 4px' : '8px 10px',
        borderTop: '1px solid var(--ant-color-border)', flexShrink: 0,
        display: 'flex', flexDirection: 'column', gap: 4,
      }}>
        {collapsed ? (
          <>
            <Button size="small" type="text" icon={<PlusOutlined />} onClick={onNew}
              title={t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' })} />
            <Button size="small" type="text" icon={<ImportOutlined />} onClick={onImport}
              title={t('strategy.workspace.importMql', { defaultValue: 'Import MQL' })} />
          </>
        ) : (
          <>
            <Button size="small" icon={<PlusOutlined />} onClick={onNew} block>
              {t('strategy.workspace.sidebar.newStrategy', { defaultValue: 'New Strategy' })}
            </Button>
            <Button size="small" icon={<ImportOutlined />} onClick={onImport} block>
              {t('strategy.workspace.importMql', { defaultValue: 'Import MQL' })}
            </Button>
          </>
        )}
      </div>
    </div>
  );
}
