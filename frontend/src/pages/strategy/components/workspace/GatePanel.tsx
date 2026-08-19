import { Button, Steps, Alert, Select, Tag } from 'antd';
import { ThunderboltOutlined, CheckCircleFilled, CloseCircleFilled, LoadingOutlined, ClockCircleFilled, MinusCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { BACKTEST_RUN_ID_LABEL_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { GATE_ALL_PASSED_KEY, GATE_EVALUATING_KEY, GATE_FAIL_KEY, GATE_FAILED_KEY, GATE_NO_DATA_KEY, GATE_PASS_KEY, GATE_RUN_HINT_KEY, GATE_RUN_PIPELINE_KEY, GATE_SKIPPED_KEY, GATE_UNKNOWN_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';

;
import type { GateResult, GatePipelineSummary } from '@/gen/ant/v1/ai_gate_pb';

interface Props {
  loading: boolean; gates: GateResult[];
  summary: GatePipelineSummary | null; error: string;
  status: string; canRun: boolean; onRun: () => void;
  runId?: string;
  availableRunIds?: string[];
  onSelectRun?: (runId: string) => void;
  fixDepth?: number;
}

const GATE_ORDER = ['compliance', 'lookahead', 'walkforward', 'deflated_sharpe', 'monte_carlo', 'paper', 'correlation'];

const GATE_KEY: Record<string, string> = {
  compliance: 'ai.gate.labels.compliance', lookahead: 'ai.gate.labels.lookahead',
  walkforward: 'ai.gate.labels.walkforward', deflated_sharpe: 'ai.gate.labels.deflated_sharpe',
  monte_carlo: 'ai.gate.labels.monte_carlo',
  paper: 'ai.gate.labels.paper', correlation: 'ai.gate.labels.correlation',
};

export default function GatePanel({ loading, gates, summary, error, status, canRun, onRun, runId, availableRunIds, onSelectRun, fixDepth }: Props) {
  const { t } = useTranslation();

  const gateMap = new Map(gates.map(g => [g.gate, g]));

  return (
    <div>
      {/* Run selector + button */}
      <div style={{ marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
        {availableRunIds && availableRunIds.length > 0 && onSelectRun && (
          <Select size="small" style={{ minWidth: 240 }} placeholder={t(BACKTEST_RUN_ID_LABEL_KEY)}
            value={runId || undefined} onChange={onSelectRun}
            options={availableRunIds.map(id => ({ label: id.slice(0, 8) + '...', value: id }))} />
        )}
        <Button type="primary" icon={<ThunderboltOutlined />} loading={loading}
          onClick={onRun} disabled={!canRun || status !== 'completed'}>
          {t(GATE_RUN_PIPELINE_KEY, 'Run Gate Evaluation')}
        </Button>
        {!canRun && status === 'idle' && (
          <span style={{ marginLeft: 8, fontSize: 11, color: 'var(--color-text-muted)' }}>
            {t(GATE_RUN_HINT_KEY, 'Complete a backtest first')}
          </span>
        )}
        {fixDepth && fixDepth > 0 && (
          <Tag color="blue" icon={<ThunderboltOutlined />} style={{ marginLeft: 'auto' }}>
            {t('ai.gate.autoFix', { defaultValue: 'Auto-Fix #{{n}}/3', n: fixDepth })}
          </Tag>
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
              return { title: <span><LoadingOutlined style={{ color: 'var(--color-info)' }} /> <span style={{ marginLeft: 8 }}>{t(GATE_KEY[gate] || gate, gate)}</span></span>,
                description: <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>{t(GATE_EVALUATING_KEY, 'Evaluating...')}</span>, status: 'process' as const };
            }
            if (!gs) {
              return { title: <span><ClockCircleFilled style={{ color: 'var(--color-text-muted)' }} /> <span style={{ marginLeft: 8 }}>{t(GATE_KEY[gate] || gate, gate)}</span></span>,
                description: null, status: 'wait' as const };
            }
            if (gs.skipped) {
              return { title: <span><MinusCircleOutlined style={{ color: 'var(--color-warning)', marginRight: 6 }} />
                {t(GATE_KEY[gs.gate] || gs.gate, gs.gate)}
                <Tag style={{ marginLeft: 8, fontSize: 10 }}>{gs.gate}</Tag>
              </span>,
                description: <span style={{ fontSize: 12, color: 'var(--color-text-muted)' }}>{t(GATE_SKIPPED_KEY, 'SKIPPED')} — {gs.reason || t(GATE_NO_DATA_KEY, 'no data')}</span>,
                status: 'wait' as const };
            }
            return { title: <span>{gs.passed
                ? <CheckCircleFilled style={{ color: 'var(--color-success)', marginRight: 6 }} />
                : <CloseCircleFilled style={{ color: 'var(--color-danger)', marginRight: 6 }} />}
              {t(GATE_KEY[gs.gate] || gs.gate, gs.gate)}
              <Tag style={{ marginLeft: 8, fontSize: 10 }}>{gs.gate}</Tag>
            </span>,
              description: <span style={{ fontSize: 12 }}>
                {gs.passed ? t(GATE_PASS_KEY, 'PASS') : `${t(GATE_FAIL_KEY, 'FAIL')} — ${gs.reason || t(GATE_UNKNOWN_KEY, 'unknown')}`}
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
            ? t(GATE_ALL_PASSED_KEY, 'All 7 gates passed — strategy eligible for live deployment')
            : t(GATE_FAILED_KEY, { defaultValue: `Failed at ${summary.firstFail}` })}
          description={!summary.passed ? summary.summary : undefined} />
      )}
    </div>
  );
}
