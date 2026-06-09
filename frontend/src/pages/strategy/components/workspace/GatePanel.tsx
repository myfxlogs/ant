import { Button, Steps, Alert, Select, Tag } from 'antd';
import { ThunderboltOutlined, CheckCircleFilled, CloseCircleFilled, LoadingOutlined, ClockCircleFilled, MinusCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';

interface Props {
  loading: boolean; gates: GateResult[];
  summary: GatePipelineSummary | null; error: string;
  status: string; canRun: boolean; onRun: () => void;
  runId?: string;
  availableRunIds?: string[];
  onSelectRun?: (runId: string) => void;
}

const GATE_ORDER = ['compliance', 'lookahead', 'walkforward', 'deflated_sharpe', 'paper', 'correlation'];

const GATE_KEY: Record<string, string> = {
  compliance: 'ai.gate.labels.compliance', lookahead: 'ai.gate.labels.lookahead',
  walkforward: 'ai.gate.labels.walkforward', deflated_sharpe: 'ai.gate.labels.deflated_sharpe',
  paper: 'ai.gate.labels.paper', correlation: 'ai.gate.labels.correlation',
};

export default function GatePanel({ loading, gates, summary, error, status, canRun, onRun, runId, availableRunIds, onSelectRun }: Props) {
  const { t } = useTranslation();

  const gateMap = new Map(gates.map(g => [g.gate, g]));

  return (
    <div>
      {/* Run selector + button */}
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
        {availableRunIds && availableRunIds.length > 0 && onSelectRun && (
          <Select size="small" style={{ minWidth: 240 }} placeholder={t('strategy.workspace.backtestRunIdLabel')}
            value={runId || undefined} onChange={onSelectRun}
            options={availableRunIds.map(id => ({ label: id.slice(0, 8) + '...', value: id }))} />
        )}
        <Button type="primary" icon={<ThunderboltOutlined />} loading={loading}
          onClick={onRun} disabled={!canRun || status !== 'completed'}>
          {t('ai.gate.runPipeline', 'Run Gate Evaluation')}
        </Button>
        {!canRun && status === 'idle' && (
          <span style={{ marginLeft: 8, fontSize: 11, color: '#8c8c8c' }}>
            {t('ai.gate.runHint', 'Complete a backtest first')}
          </span>
        )}
      </div>

      {/* Errors */}
      {error && <Alert type="error" message={error} closable style={{ marginBottom: 12 }} />}

      {/* Gate steps */}
      {(gates.length > 0 || loading) && (
        <Steps direction="vertical" size="small"
          status={!loading && summary && !summary.passed ? 'error' : !loading && summary?.passed ? 'finish' : 'process'}
          items={GATE_ORDER.map(gate => {
            const gs = gateMap.get(gate);
            const isCurrent = loading && gates.length === GATE_ORDER.indexOf(gate);
            if (isCurrent) {
              return { title: <span><LoadingOutlined style={{ color: '#1677ff' }} /> <span style={{ marginLeft: 8 }}>{t(GATE_KEY[gate] || gate, gate)}</span></span>,
                description: <span style={{ fontSize: 12, color: '#8c8c8c' }}>{t('ai.gate.evaluating', 'Evaluating...')}</span>, status: 'process' as const };
            }
            if (!gs) {
              return { title: <span><ClockCircleFilled style={{ color: '#d9d9d9' }} /> <span style={{ marginLeft: 8 }}>{t(GATE_KEY[gate] || gate, gate)}</span></span>,
                description: null, status: 'wait' as const };
            }
            if (gs.skipped) {
              return { title: <span><MinusCircleOutlined style={{ color: '#faad14', marginRight: 6 }} />
                {GATE_LABELS[gs.gate] || gs.gate}
                <Tag style={{ marginLeft: 8, fontSize: 10 }}>{gs.gate}</Tag>
              </span>,
                description: <span style={{ fontSize: 12, color: '#8c8c8c' }}>{t('ai.gate.skipped', 'SKIPPED')} — {gs.reason || t('ai.gate.noData', 'no data')}</span>,
                status: 'wait' as const };
            }
            return { title: <span>{gs.passed
                ? <CheckCircleFilled style={{ color: '#52c41a', marginRight: 6 }} />
                : <CloseCircleFilled style={{ color: '#ff4d4f', marginRight: 6 }} />}
              {GATE_LABELS[gs.gate] || gs.gate}
              <Tag style={{ marginLeft: 8, fontSize: 10 }}>{gs.gate}</Tag>
            </span>,
              description: <span style={{ fontSize: 12 }}>
                {gs.passed ? t('ai.gate.pass', 'PASS') : `${t('ai.gate.fail', 'FAIL')} — ${gs.reason || t('ai.gate.unknown', 'unknown')}`}
                {gs.score !== 0 ? ` (score: ${gs.score.toFixed(4)})` : ''}
                {` — ${gs.durationMs}ms`}
              </span>,
              status: gs.passed ? 'finish' as const : 'error' as const };
          })}
        />
      )}

      {/* Summary */}
      {summary && (
        <Alert type={summary.passed ? 'success' : 'error'} showIcon
          style={{ marginTop: 12 }}
          message={summary.passed
            ? t('ai.gate.allPassed', 'All 6 gates passed — strategy eligible for live deployment')
            : t('ai.gate.failed', { defaultValue: `Failed at ${summary.firstFail}` })}
          description={!summary.passed ? summary.summary : undefined} />
      )}
    </div>
  );
}
