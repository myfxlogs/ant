import { Button, Input, Tag, Typography, Card, Space } from 'antd';
import { CodeOutlined, ThunderboltOutlined, CheckCircleOutlined, CloseCircleOutlined, EditOutlined, LoadingOutlined, SendOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import {
  EXEC_TITLE_KEY, EXEC_RUNNING_KEY, EXEC_DONE_KEY, EXEC_BACK_TO_PLAN_KEY,
  EXEC_PLAN_LABEL_KEY, EXEC_COMPLIANCE_TOOL_KEY, EXEC_BACKTEST_TOOL_KEY,
  EXEC_FEEDBACK_TITLE_KEY, EXEC_FEEDBACK_HINT_KEY,
  EXEC_FEEDBACK_PLACEHOLDER_KEY, EXEC_CHIP_LOWER_DD_KEY, EXEC_CHIP_RAISE_RETURN_KEY,
  EXEC_CHIP_TIGHTEN_SL_KEY, EXEC_CHIP_LONG_ONLY_KEY, EXEC_SEND_FEEDBACK_KEY,
  EXEC_CLEAR_KEY, EXEC_APPLY_CODE_KEY,
  PLACEHOLDER_KEY, FEEDBACK_HEADING_KEY,
} from '@/gen/ant/v1/i18n/strategy_gen_keys';
import StepProgress from './StepProgress';
import DiffView from './DiffView';
import { useExecutionPanel } from './useExecutionPanel';

const { TextArea } = Input;

interface Props {
  plan: string;
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  previousCode?: string;
  onApply: (code: string, previousCode?: string) => void;
  onReset: () => void;
}

export default function ExecutionPanel({ plan, symbol, timeframe, sessionId, previousCode, onApply, onReset }: Props) {
  const { t } = useTranslation();
  const {
    phase, currentPhase, streamCode, code, prevCode, toolResults,
    error, feedback, setFeedback, analysis, discussionReply, metrics,
    diagnosis, pendingFeedback,
    handleFeedback, handleConfirmDiagnosis, handleRetryFeedback,
    busy, setDiagnosis, setPendingFeedback,
  } = useExecutionPanel({ plan, symbol, timeframe, sessionId, previousCode });

  const toolLabel = (name: string) => {
    switch (name) {
      case 'compliance_check': return t(EXEC_COMPLIANCE_TOOL_KEY, 'Compliance Check');
      case 'backtest': return t(EXEC_BACKTEST_TOOL_KEY, 'Backtest');
      default: return name;
    }
  };

  const toolIcon = (name: string) => {
    switch (name) {
      case 'compliance_check': return <CodeOutlined />;
      case 'backtest': return <ThunderboltOutlined />;
      default: return <CodeOutlined />;
    }
  };

  const chips = [
    { key: 'lower_dd', label: t(EXEC_CHIP_LOWER_DD_KEY, 'Lower Drawdown') },
    { key: 'raise_return', label: t(EXEC_CHIP_RAISE_RETURN_KEY, 'Raise Returns') },
    { key: 'tighten_sl', label: t(EXEC_CHIP_TIGHTEN_SL_KEY, 'Tighten Stop') },
    { key: 'long_only', label: t(EXEC_CHIP_LONG_ONLY_KEY, 'Long Only') },
  ];

  const planSteps = plan.split('\n').filter(line => /^\d+[\.\)]\s/.test(line.trim()));
  const hasSymbol = !!(symbol && timeframe);

  return (
    <div style={{ padding: 10, border: '1px solid #f0f0f0', borderRadius: 6, background: '#fafafa' }}>
      {hasSymbol && (
        <Tag color="blue" style={{ fontSize: 11, marginBottom: 6 }}>📊 {symbol} · {timeframe}</Tag>
      )}
      <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
        <ThunderboltOutlined style={{ color: '#faad14' }} />
        <Typography.Text strong style={{ fontSize: 13 }}>{t(EXEC_TITLE_KEY, 'AI Executing')}</Typography.Text>
        {busy && <Tag icon={<LoadingOutlined />} color="processing">{t(EXEC_RUNNING_KEY, 'Running')}</Tag>}
        {phase === 'done' && <Tag color="success">{t(EXEC_DONE_KEY, 'Done')}</Tag>}
        {phase === 'error' && <Tag color="error">{t(PLACEHOLDER_KEY, 'Error')}</Tag>}
        <Button size="small" type="link" onClick={onReset} style={{ marginLeft: 'auto' }}>{t(EXEC_BACK_TO_PLAN_KEY, 'Back to Plan')}</Button>
      </div>

      <Card size="small" style={{ marginBottom: 8, background: '#f6ffed', borderColor: '#b7eb8f' }}>
        <Typography.Text style={{ fontSize: 11, color: '#8c8c8c' }}>{t(EXEC_PLAN_LABEL_KEY, 'Execution Plan')}</Typography.Text>
        {planSteps.length > 0 ? (
          <div style={{ marginTop: 4 }}>
            {planSteps.map((step, i) => (
              <div key={i} style={{ fontSize: 12, padding: '1px 0', color: '#595959' }}>
                <CheckCircleOutlined style={{ color: '#52c41a', marginRight: 4, fontSize: 11 }} />
                {step.replace(/^\d+[\.\)]\s*/, '')}
              </div>
            ))}
          </div>
        ) : (
          <Typography.Paragraph ellipsis={{ rows: 2 }} style={{ fontSize: 12, margin: 0 }}>{plan}</Typography.Paragraph>
        )}
      </Card>

      <StepProgress phase={busy ? currentPhase : phase === 'done' ? 'done' : 'idle'} plan={undefined} />

      {discussionReply && (
        <div style={{ padding: '8px 10px', marginBottom: 6, borderRadius: 4, background: '#f5f5f5', border: '1px solid #d9d9d9', fontSize: 12, whiteSpace: 'pre-wrap', color: '#595959' }}>
          💬 {discussionReply}
        </div>
      )}

      {analysis && (
        <div style={{ padding: '8px 10px', marginBottom: 6, borderRadius: 4, background: '#e6f4ff', border: '1px solid #91caff', fontSize: 12, whiteSpace: 'pre-wrap', color: '#1677ff' }}>
          💡 {analysis}
        </div>
      )}

      {toolResults.map((tr, i) => (
        <div key={i} style={{
          padding: '6px 10px', marginBottom: 4, borderRadius: 4, fontSize: 11,
          background: tr.name === 'backtest' ? '#f0f5ff' : (tr.success ? '#f6ffed' : '#fff2f0'),
          border: `1px solid ${tr.name === 'backtest' ? '#d6e4ff' : (tr.success ? '#b7eb8f' : '#ffa39e')}`,
        }}>
          <Space>
            {tr.name === 'backtest'
              ? <span style={{ color: '#2f54eb' }}>📊</span>
              : (tr.success ? <CheckCircleOutlined style={{ color: '#52c41a' }} /> : <CloseCircleOutlined style={{ color: '#ff4d4f' }} />)
            }
            <span>{toolIcon(tr.name)} <b>{toolLabel(tr.name)}</b></span>
            {tr.error && <span style={{ color: '#cf1322' }}>{tr.error}</span>}
          </Space>
        </div>
      ))}

      {metrics && (
        <div style={{ padding: '6px 10px', marginBottom: 6, borderRadius: 4, background: '#f6ffed', border: '1px solid #b7eb8f', fontSize: 11 }}>
          <b>📊 {t(FEEDBACK_HEADING_KEY, 'Backtest Results')}</b> · Sharpe {metrics.sharpeRatio?.toFixed(2)} · {(t as any)('strategy.gen.metrics.maxDrawdown', 'Max DD')}: {((metrics.maxDrawdown ?? 0) * 100).toFixed(1)}% · {(t as any)('strategy.gen.metrics.winRate', 'Win')}: {((metrics.winRate ?? 0) * 100).toFixed(0)}% · {(t as any)('strategy.gen.metrics.trades', 'Trades')}: {metrics.totalTrades}
        </div>
      )}

      {prevCode && code && <DiffView oldCode={prevCode} newCode={code} />}

      {streamCode && (
        <div style={{ maxHeight: 200, overflow: 'auto', padding: 8, marginBottom: 8,
          background: '#fff', borderRadius: 4, border: '1px solid #f0f0f0',
          fontSize: 11, fontFamily: 'monospace', whiteSpace: 'pre-wrap' }}>{streamCode}</div>
      )}

      {error && (
        <div style={{ padding: 6, marginBottom: 8, background: '#fff2f0', borderRadius: 4, fontSize: 11, color: '#cf1322' }}>{error}</div>
      )}

      {diagnosis && phase === 'done' && (
        <div style={{ marginBottom: 8, padding: 10, background: '#fffbe6', borderRadius: 6, border: '1px solid #ffe58f' }}>
          <Typography.Text strong style={{ fontSize: 12, display: 'block', marginBottom: 4 }}>
            {t(EXEC_TITLE_KEY)}
          </Typography.Text>
          <div style={{ fontSize: 12, whiteSpace: 'pre-wrap', color: '#595959', marginBottom: 8 }}>
            {diagnosis}
          </div>
          <Space>
            <Button size="small" type="primary" icon={<CheckCircleOutlined />}
              onClick={handleConfirmDiagnosis}>
              {t(EXEC_APPLY_CODE_KEY)}
            </Button>
            <Button size="small" icon={<EditOutlined />}
              onClick={handleRetryFeedback}>
              {t(EXEC_SEND_FEEDBACK_KEY)}
            </Button>
            <Button size="small" type="text"
              onClick={() => { setDiagnosis(''); setPendingFeedback(''); }}>
              {t(EXEC_CLEAR_KEY)}
            </Button>
          </Space>
        </div>
      )}

      {phase === 'done' && code && (
        <div style={{ marginTop: 8, padding: 10, background: '#f0f5ff', borderRadius: 6, border: '1px solid #adc6ff' }}>
          <Typography.Text strong style={{ fontSize: 11, display: 'block', marginBottom: 4 }}>
            {t(EXEC_FEEDBACK_TITLE_KEY, 'Continue AI Conversation')}
          </Typography.Text>
          <Typography.Text type="secondary" style={{ fontSize: 10, display: 'block', marginBottom: 6 }}>
            {t(EXEC_FEEDBACK_HINT_KEY, 'Use natural language to guide the AI.')}
          </Typography.Text>
          <TextArea rows={3} value={feedback} onChange={e => setFeedback(e.target.value)}
            placeholder={t(EXEC_FEEDBACK_PLACEHOLDER_KEY, 'Try saying: "tighten stop to 1%"')}
            disabled={busy} style={{ fontSize: 13, marginBottom: 6 }}
          />
          <Space style={{ marginBottom: 6 }}>
            {chips.map(chip => (
              <Tag key={chip.key} color="purple" style={{ cursor: 'pointer', fontSize: 11 }}
                onClick={() => { setFeedback(chip.label); }}>
                💬 {chip.label}
              </Tag>
            ))}
          </Space>
          <div style={{ display: 'flex', gap: 8 }}>
            <Button type="primary" size="small" icon={<SendOutlined />} block
              onClick={handleFeedback} disabled={!feedback.trim()} loading={busy}>
              {t(EXEC_SEND_FEEDBACK_KEY, 'Send to AI')}
            </Button>
            <Button size="small" onClick={() => { setFeedback(''); }} disabled={!feedback.trim()}>
              {t(EXEC_CLEAR_KEY, 'Clear')}
            </Button>
          </div>
        </div>
      )}

      {phase === 'done' && code && (
        <Button type="primary" size="small" icon={<EditOutlined />} block
          onClick={() => onApply(code, prevCode)} style={{ marginTop: 8 }}>
          {t(EXEC_APPLY_CODE_KEY, 'Apply to Editor')}
        </Button>
      )}
    </div>
  );
}
