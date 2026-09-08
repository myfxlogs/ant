import { Button, Empty } from 'antd';
import { useTranslation } from 'react-i18next';

interface RunItem {
  id: string;
  startedAt?: string;
  totalReturn?: number;
  totalTrades?: number;
  templateName?: string;
  name?: string;
}

interface Props {
  runs: RunItem[];
  loading: boolean;
  onOpen: (runId: string) => void;
  onDelete?: (runId: string) => void;
}

// 回测历史主区面板：分区切换后展示，点击一条即加载该次回测。
export default function BacktestHistoryPanel({ runs, loading, onOpen, onDelete }: Props) {
  const { t } = useTranslation();

  if (loading) {
    return <div style={{ flex: '1 1 0', display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--ant-color-text-secondary)' }}>{t('strategy.workspace.history.loading', { defaultValue: '加载中…' })}</div>;
  }
  if (runs.length === 0) {
    return (
      <div style={{ flex: '1 1 0', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <Empty description={t('strategy.workspace.history.empty', { defaultValue: '暂无回测记录' })} />
      </div>
    );
  }

  return (
    <div style={{ flex: '1 1 0', overflow: 'auto', padding: '20px 24px' }}>
      <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--ant-color-text)', marginBottom: 14 }}>
        {t('strategy.workspace.history.title', { defaultValue: '回测历史' })}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 8, maxWidth: 760 }}>
        {runs.map((r) => {
          const ret = r.totalReturn != null ? Number(r.totalReturn) : null;
          return (
            <div key={r.id} data-testid={`history-run-${r.id}`}
              onClick={() => onOpen(r.id)}
              style={{
                display: 'flex', alignItems: 'center', gap: 14, padding: '10px 14px', cursor: 'pointer',
                borderRadius: 8, border: '1px solid var(--ant-color-border)', background: 'var(--ant-color-bg-container)',
              }}>
              <span style={{ flex: '1 1 0', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: 500 }}>
                {r.name || r.templateName || r.id.slice(0, 8)}
              </span>
              <span style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>{r.startedAt || ''}</span>
              {ret != null && (
                <span style={{ fontSize: 13, fontWeight: 600, color: ret >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
                  {ret >= 0 ? '+' : ''}{ret.toFixed(1)}%
                </span>
              )}
              {r.totalTrades != null && (
                <span style={{ fontSize: 12, color: 'var(--ant-color-text-secondary)' }}>{r.totalTrades} {t('strategy.workspace.history.trades', { defaultValue: '笔' })}</span>
              )}
              {onDelete && (
                <Button size="small" type="text" danger onClick={(e) => { e.stopPropagation(); onDelete(r.id); }}>
                  {t('strategy.workspace.history.delete', { defaultValue: '删除' })}
                </Button>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
