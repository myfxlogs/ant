import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Tooltip, Empty, Input, message } from 'antd';
import { PlayCircleOutlined, SaveOutlined, CopyOutlined, EditOutlined } from '@ant-design/icons';
import type { RightTab } from '@/stores/workspaceStore';
import StrategyChat from '@/components/strategy/StrategyChat';
import BacktestResultsCard from '@/components/strategy/BacktestResultsCard';

interface BacktestMetrics {
  totalReturn?: number;
  annualReturn?: number;
  maxDrawdown?: number;
  sharpeRatio?: number;
  winRate?: number;
  totalTrades?: number;
  equityCurve?: Array<{ time: number; equity: number }>;
}

interface Props {
  tab: RightTab;
  onTabChange: (t: RightTab) => void;

  // Chat
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  accountId?: string;
  onApplyCode: (code: string) => void;
  onValidateResult?: (result: import('@/client/codeAssist').ValidateExtendedResult) => void;
  onRunBacktest?: () => void;
  backtestStatus?: string;

  // Results
  metrics: BacktestMetrics | null;
  backtestStatus2: string;

  // Code
  code: string;
  onCodeChange: (code: string) => void;
  onSaveCode: () => void;
  onRunBacktest2: () => void;
}

const TABS: { key: RightTab; icon: string; labelKey: string; fallback: string }[] = [
  { key: 'chat', icon: '💬', labelKey: 'strategy.workspace.tabChat', fallback: 'Chat' },
  { key: 'results', icon: '📊', labelKey: 'strategy.workspace.tabResults', fallback: 'Results' },
  { key: 'code', icon: '📄', labelKey: 'strategy.workspace.tabCode', fallback: 'Code' },
];

export default function RightPanel(props: Props) {
  const { t } = useTranslation();
  const { tab, onTabChange } = props;
  const [editable, setEditable] = useState(false);

  const handleCopy = () => {
    if (!props.code) return;
    navigator.clipboard?.writeText(props.code)
      .then(() => message.success(t('common.copied', '已复制')))
      .catch(() => message.error(t('common.copyFailed', '复制失败')));
  };

  return (
    <div style={{
      width: 380, minWidth: 380, flexShrink: 0,
      background: 'var(--ant-color-bg-container)',
      borderLeft: '1px solid var(--ant-color-border)',
      display: 'flex', flexDirection: 'column', overflow: 'hidden',
    }}>
      {/* Tab bar */}
      <div style={{
        display: 'flex', borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0,
      }}>
        {TABS.map(({ key, icon, labelKey, fallback }) => (
          <div
            key={key}
            onClick={() => onTabChange(key)}
            style={{
              flex: 1, textAlign: 'center', padding: '8px 0',
              fontSize: 11, fontWeight: 600, cursor: 'pointer',
              color: tab === key ? '#58a6ff' : 'var(--ant-color-text-secondary)',
              borderBottom: tab === key ? '2px solid #58a6ff' : '2px solid transparent',
              transition: 'color 0.15s, border-color 0.15s',
            }}
          >
            {icon} {t(labelKey, fallback)}
          </div>
        ))}
      </div>

      {/* Tab content */}
      {tab === 'chat' && (
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          <StrategyChat
            symbol={props.symbol}
            timeframe={props.timeframe}
            sessionId={props.sessionId}
            accountId={props.accountId}
            onApplyCode={props.onApplyCode}
            onValidateResult={props.onValidateResult}
            onRunBacktest={props.onRunBacktest}
            backtestStatus={props.backtestStatus}
          />
        </div>
      )}

      {tab === 'results' && (
        <div style={{ flex: 1, overflowY: 'auto', padding: 12 }}>
          <BacktestResultsCard metrics={props.metrics} status={props.backtestStatus2} />
        </div>
      )}

      {tab === 'code' && (
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
          {/* Code actions bar */}
          <div style={{
            display: 'flex', gap: 4, padding: '6px 10px',
            borderBottom: '1px solid var(--ant-color-border)', flexShrink: 0,
            alignItems: 'center',
          }}>
            <span style={{ fontWeight: 700, fontSize: 11, marginRight: 4 }}>
              {t('strategy.workspace.strategyCode', 'Strategy Code')}
            </span>
            <span style={{ flex: 1 }} />
            <Tooltip title={t('strategy.workspace.save', 'Save Strategy')}>
              <Button size="small" icon={<SaveOutlined />}
                onClick={props.onSaveCode}
                style={{ background: '#58a6ff', borderColor: '#58a6ff', color: '#fff' }}>
                {t('common.save', 'Save')}
              </Button>
            </Tooltip>
            <Tooltip title={t('strategy.workspace.runBacktest', 'Run Backtest')}>
              <Button size="small" type="primary" icon={<PlayCircleOutlined />}
                onClick={props.onRunBacktest2}
                style={{ background: '#3fb950', borderColor: '#3fb950' }}>
                {t('strategy.gen.backtest', 'Backtest')}
              </Button>
            </Tooltip>
            <Tooltip title={t('common.copy', 'Copy Code')}>
              <Button size="small" icon={<CopyOutlined />} onClick={handleCopy} />
            </Tooltip>
            <Tooltip title={editable ? t('common.readOnly', 'Read Only') : t('common.edit', 'Edit')}>
              <Button size="small" type={editable ? 'primary' : 'default'} icon={<EditOutlined />}
                onClick={() => setEditable(e => !e)} />
            </Tooltip>
          </div>

          {/* Code display / editor */}
          {props.code ? (
            editable ? (
              <Input.TextArea
                value={props.code}
                onChange={(e) => props.onCodeChange(e.target.value)}
                style={{
                  flex: 1, borderRadius: 0, resize: 'none',
                  fontFamily: '"SF Mono", "Fira Code", monospace',
                  fontSize: 11, lineHeight: 1.6,
                }}
              />
            ) : (
              <div style={{
                flex: 1, overflowY: 'auto', padding: 10,
                fontFamily: '"SF Mono", "Fira Code", monospace',
                fontSize: 11, lineHeight: 1.6, whiteSpace: 'pre-wrap',
                color: 'var(--ant-color-text)',
              }}>
                {props.code}
              </div>
            )
          ) : (
            <div style={{ display: 'flex', justifyContent: 'center', paddingTop: 40 }}>
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE}
                description={t('strategy.workspace.noCode', 'No code generated yet')} />
            </div>
          )}
        </div>
      )}
    </div>
  );
}
