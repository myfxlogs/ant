import { useState, useRef, useEffect, useCallback } from 'react';
import {
  Card, Form, Input, InputNumber, Button, Steps, Space, Typography,
  Descriptions, Collapse, Divider, Alert, Tag,
} from 'antd';
import {
  PlayCircleOutlined, ReloadOutlined, CheckCircleFilled,
  CloseCircleFilled, ClockCircleFilled, LoadingOutlined, ThunderboltOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAuthStore } from '@/stores/authStore';
import { apiBaseUrl } from '@/client/transport';

const { Text, Title } = Typography;

interface GateStatus {
  gate: string; passed: boolean; reason?: string; score?: number; duration_ms: number;
}
interface PipelineSummary { passed: boolean; summary: string; first_fail: string; }

const GATE_ORDER = ['compliance', 'lookahead', 'walkforward', 'deflated_sharpe', 'paper', 'correlation'];

function useGateI18n() {
  const { t } = useTranslation();
  return {
    label: (gate: string) => t(`ai.gate.labels.${gate}`),
    description: (gate: string) => t(`ai.gate.descriptions.${gate}`),
    evaluating: t('ai.gate.status.evaluating', { defaultValue: 'Evaluating...' }),
    pipelineDesc: t('ai.gate.pipelineDesc', { defaultValue: '6-stage Gate pipeline: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation' }),
    strategyParams: t('ai.gate.strategyParams', { defaultValue: 'Strategy Parameters' }),
    dslExpression: t('ai.gate.dslExpression', { defaultValue: 'DSL Expression' }),
    dailyReturns: t('ai.gate.dailyReturns', { defaultValue: 'Daily Returns (comma or newline separated)' }),
    numAttempts: t('ai.gate.numAttempts', { defaultValue: 'Strategy Attempts' }),
    paperMetrics: t('ai.gate.paperMetrics', { defaultValue: 'Paper Trading Metrics' }),
    paperDays: t('ai.gate.paperDays', { defaultValue: 'Paper Days' }),
    paperNetPnL: t('ai.gate.paperNetPnL', { defaultValue: 'Paper Net P&L' }),
    paperNetReturn: t('ai.gate.paperNetReturn', { defaultValue: 'Paper Net Return' }),
    paperTradeCount: t('ai.gate.paperTradeCount', { defaultValue: 'Paper Trade Count' }),
    backtestNetReturn: t('ai.gate.backtestNetReturn', { defaultValue: 'Backtest Net Return' }),
    backtestGrossReturn: t('ai.gate.backtestGrossReturn', { defaultValue: 'Backtest Gross Return' }),
    runPipeline: t('ai.gate.runPipeline', { defaultValue: 'Run Gate Pipeline' }),
    retry: t('ai.gate.retry', { defaultValue: 'Retry' }),
    gateProgress: t('ai.gate.gateProgress', { defaultValue: 'Gate Evaluation Progress' }),
    pipelineResult: t('ai.gate.pipelineResult', { defaultValue: 'Pipeline Result' }),
    allPassed: t('ai.gate.allPassed', { defaultValue: 'All 6 gates passed — strategy eligible for PromoteToLive evaluation' }),
    failed: (gate: string) => t('ai.gate.failed', { defaultValue: `Failed: ${gate}` }),
    details: t('ai.gate.details', { defaultValue: 'Details' }),
  };
}

function buildGateIcon(idx: number, gates: GateStatus[], loading: boolean, i18n: ReturnType<typeof useGateI18n>): [React.ReactNode, string, 'wait' | 'process' | 'finish' | 'error'] {
  const gs = gates[idx];
  const isCurrent = loading && idx === gates.length;
  if (isCurrent) return [<LoadingOutlined style={{ color: '#1677ff' }} />, 'process', i18n.evaluating];
  if (!gs) return [<ClockCircleFilled style={{ color: '#d9d9d9' }} />, 'wait', i18n.description(GATE_ORDER[idx])];
  if (gs.passed) return [<CheckCircleFilled style={{ color: '#52c41a' }} />, 'finish', buildDesc(gs, i18n)];
  return [<CloseCircleFilled style={{ color: '#ff4d4f' }} />, 'error', buildDesc(gs, i18n)];
}

function buildDesc(gs: GateStatus, i18n: ReturnType<typeof useGateI18n>): string {
  const parts = [i18n.description(gs.gate)];
  if (gs.score !== undefined && gs.score !== 0) parts.push(`Score: ${gs.score.toFixed(4)}`);
  parts.push(`${gs.duration_ms}ms`);
  if (!gs.passed && gs.reason) parts.push(`❌ ${gs.reason}`);
  return parts.join(' · ');
}

const INITIAL_VALUES = {
  expression: 'close[1] > close[2] * 1.01 and volume[1] > volume[2]',
  dailyReturns: '0.01, -0.005, 0.02, 0.008, -0.003, 0.015, 0.007, -0.01, 0.012, 0.005, -0.002, 0.018, 0.009, -0.004, 0.011',
  numAttempts: 5, paperDays: 14, paperNetPnL: 500, paperNetReturn: 0.08,
  backtestNetReturn: 0.12, backtestGrossReturn: 0.15, paperTradeCount: 10,
};

export default function GateProgressPage() {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);
  const [gates, setGates] = useState<GateStatus[]>([]);
  const [summary, setSummary] = useState<PipelineSummary | null>(null);
  const [error, setError] = useState<string | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const i18n = useGateI18n();

  useEffect(() => {
    return () => { abortRef.current?.abort(); };
  }, []);

  const handleRun = useCallback(async () => {
    abortRef.current?.abort();
    const values = await form.validateFields();
    setLoading(true); setError(null); setGates([]); setSummary(null);
    const token = useAuthStore.getState().accessToken;
    if (!token) { setError('Not authenticated'); setLoading(false); return; }
    const controller = new AbortController();
    abortRef.current = controller;

    const dailyReturnsStr: string = values.dailyReturns || '';
    const dailyReturns = dailyReturnsStr.split(/[\n,]+/).map(s => s.trim()).filter(s => s.length > 0).map(Number).filter(n => !isNaN(n));

    const input = {
      expression: values.expression || '', daily_returns: dailyReturns, num_attempts: values.numAttempts || 1,
      paper_metrics: {
        paper_days: values.paperDays || 14, paper_net_pnl: values.paperNetPnL || 0,
        paper_net_return: values.paperNetReturn || 0, paper_trade_count: values.paperTradeCount || 0,
        backtest_net_return: values.backtestNetReturn || 0, backtest_gross_return: values.backtestGrossReturn || 0,
      },
      new_signals: [], existing_signals: {},
    };

    try {
      const response = await fetch(`${apiBaseUrl}/sse/ai/gate-progress`, {
        method: 'POST', headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${token}` }, body: JSON.stringify(input), signal: controller.signal,
      });
      if (!response.ok) {
        let errText = `HTTP ${response.status}`;
        try { errText = await response.text() || errText; } catch { /* body unreadable */ }
        setError(errText); setLoading(false); return;
      }
      const reader = response.body?.getReader();
      if (!reader) { setError('Streaming not supported'); setLoading(false); return; }

      const decoder = new TextDecoder(); let buffer = '';
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split('\n'); buffer = lines.pop() || '';
        let eventType = '';
        for (const line of lines) {
          if (line.startsWith('event: ')) { eventType = line.slice(7).trim(); }
          else if (line.startsWith('data: ')) {
            try {
              const parsed = JSON.parse(line.slice(6));
              if (eventType === 'gate') setGates(prev => [...prev, parsed as GateStatus]);
              else if (eventType === 'completed') setSummary(parsed as PipelineSummary);
            } catch (parseErr) { console.debug('gate-progress SSE: unparseable frame', parseErr); }
          }
        }
      }
    } catch (err: unknown) {
      if (err instanceof DOMException && err.name === 'AbortError') return;
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally { setLoading(false); abortRef.current = null; }
  }, [form]);

  const handleRetry = () => { abortRef.current?.abort(); handleRun(); };
  const currentStep = gates.length > 0 ? gates.length - 1 : 0;

  return (
    <div style={{ padding: 16, maxWidth: 960, margin: '0 auto' }}>
      <Title level={4} style={{ marginBottom: 0 }}>
        <ThunderboltOutlined style={{ marginRight: 8 }} />
        {t('ai.gate.title', { defaultValue: 'AI Gate Progress' })}
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        {i18n.pipelineDesc}
      </Text>

      <Card size="small" title={i18n.strategyParams} style={{ marginBottom: 16 }}>
        <Form form={form} layout="vertical" size="small" initialValues={INITIAL_VALUES}>
          <Form.Item name="expression" label={i18n.dslExpression}>
            <Input.TextArea rows={2} placeholder="close[1] > close[2] * 1.01" />
          </Form.Item>
          <Form.Item name="dailyReturns" label={i18n.dailyReturns}>
            <Input.TextArea rows={3} placeholder="0.01, -0.005, 0.02, ..." />
          </Form.Item>
          <Form.Item name="numAttempts" label={i18n.numAttempts}>
            <InputNumber min={1} max={1000} />
          </Form.Item>
          <Divider plain style={{ fontSize: 13 }}>{i18n.paperMetrics}</Divider>
          <Space wrap>
            <Form.Item name="paperDays" label={i18n.paperDays}><InputNumber min={1} max={365} /></Form.Item>
            <Form.Item name="paperNetPnL" label={i18n.paperNetPnL}><InputNumber style={{ width: 120 }} /></Form.Item>
            <Form.Item name="paperNetReturn" label={i18n.paperNetReturn}><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
            <Form.Item name="paperTradeCount" label={i18n.paperTradeCount}><InputNumber min={0} max={10000} /></Form.Item>
            <Form.Item name="backtestNetReturn" label={i18n.backtestNetReturn}><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
            <Form.Item name="backtestGrossReturn" label={i18n.backtestGrossReturn}><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
          </Space>
          <div style={{ marginTop: 12 }}>
            <Space>
              <Button type="primary" icon={loading ? <LoadingOutlined /> : <PlayCircleOutlined />} onClick={handleRun} loading={loading}>
                {i18n.runPipeline}
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleRetry} disabled={!loading && gates.length === 0}>{i18n.retry}</Button>
            </Space>
          </div>
        </Form>
      </Card>

      {error && <Alert type="error" message={error} closable style={{ marginBottom: 16 }} onClose={() => setError(null)} />}

      {(gates.length > 0 || loading) && (
        <Card size="small" title={i18n.gateProgress} style={{ marginBottom: 16 }}>
          <Steps direction="vertical" size="small"
            current={loading && gates.length === 0 ? -1 : currentStep}
            status={!loading && summary && !summary.passed ? 'error' : !loading && summary?.passed ? 'finish' : 'process'}
            items={GATE_ORDER.map((gate, idx) => {
              const [icon, status, desc] = buildGateIcon(idx, gates, loading, i18n);
              return {
                title: <span>{icon}<span style={{ marginLeft: 8, fontWeight: 600 }}>{i18n.label(gate)}</span><Tag style={{ marginLeft: 8, fontSize: 11 }}>{gate}</Tag></span>,
                description: <Text type={status === 'error' ? 'danger' : 'secondary'} style={{ fontSize: 12 }}>{desc}</Text>,
                status,
              };
            })}
          />
        </Card>
      )}

      {summary && (
        <Card size="small" title={i18n.pipelineResult}>
          {summary.passed
            ? <Alert type="success" message={i18n.allPassed} showIcon />
            : <Alert type="error" message={i18n.failed(summary.first_fail)} description={summary.summary} showIcon />
          }
          {gates.length > 0 && (
            <Collapse size="small" style={{ marginTop: 12 }} items={[{
              key: 'details', label: i18n.details,
              children: (
                <Descriptions size="small" column={1}>
                  {gates.map(gs => (
                    <Descriptions.Item key={gs.gate}
                      label={<span>{gs.passed ? <CheckCircleFilled style={{ color: '#52c41a', marginRight: 4 }} /> : <CloseCircleFilled style={{ color: '#ff4d4f', marginRight: 4 }} />}{i18n.label(gs.gate)}</span>}
                    >
                      {gs.passed ? 'PASS' : `FAIL — ${gs.reason || 'unknown'}`}
                      {gs.score !== undefined && gs.score !== 0 ? ` (score: ${gs.score.toFixed(4)})` : ''}
                      {' — '}{gs.duration_ms}ms
                    </Descriptions.Item>
                  ))}
                </Descriptions>
              ),
            }]}
            />
          )}
        </Card>
      )}
    </div>
  );
}
