import { useState, useRef, useCallback } from 'react';
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

const GATE_LABELS: Record<string, string> = {
  compliance: '合规检查', lookahead: '前视偏差', walkforward: 'Walk-Forward',
  deflated_sharpe: 'Deflated Sharpe', paper: '模拟交易', correlation: '相关性',
};

const GATE_DESCRIPTIONS: Record<string, string> = {
  compliance: 'DSL 表达式非空验证',
  lookahead: '扫描未来函数引用 (close[t+N], ref 负偏移)',
  walkforward: 'Purged Walk-Forward 交叉验证',
  deflated_sharpe: 'Lopez de Prado 紧缩夏普比率',
  paper: '≥14 天模拟交易验证',
  correlation: '与现有策略信号相关性检查',
};

function buildGateIcon(idx: number, gates: GateStatus[], loading: boolean): [React.ReactNode, string, 'wait' | 'process' | 'finish' | 'error'] {
  const gs = gates[idx];
  const isCurrent = loading && idx === gates.length;
  if (isCurrent) return [<LoadingOutlined style={{ color: '#1677ff' }} />, 'process', '评估中...'];
  if (!gs) return [<ClockCircleFilled style={{ color: '#d9d9d9' }} />, 'wait', GATE_DESCRIPTIONS[GATE_ORDER[idx]]];
  if (gs.passed) return [<CheckCircleFilled style={{ color: '#52c41a' }} />, 'finish', buildDesc(gs)];
  return [<CloseCircleFilled style={{ color: '#ff4d4f' }} />, 'error', buildDesc(gs)];
}

function buildDesc(gs: GateStatus): string {
  const parts = [GATE_DESCRIPTIONS[gs.gate] || ''];
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

  const handleRun = useCallback(async () => {
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
      const response = await fetch(`${apiBaseUrl}/sse/ai/gate-progress?access_token=${encodeURIComponent(token)}`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(input), signal: controller.signal,
      });
      if (!response.ok) { setError(await response.text() || `HTTP ${response.status}`); setLoading(false); return; }
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
            } catch { /* skip unparseable frames */ }
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
        {t('ai.gate.title', { defaultValue: 'AI Gate 进度面板' })}
      </Title>
      <Text type="secondary" style={{ display: 'block', marginBottom: 16 }}>
        6 级 Gate 管道: Compliance → LookAhead → Walk-Forward → DeflatedSharpe → Paper → Correlation
      </Text>

      <Card size="small" title="策略参数" style={{ marginBottom: 16 }}>
        <Form form={form} layout="vertical" size="small" initialValues={INITIAL_VALUES}>
          <Form.Item name="expression" label="DSL 表达式">
            <Input.TextArea rows={2} placeholder="close[1] > close[2] * 1.01" />
          </Form.Item>
          <Form.Item name="dailyReturns" label="日收益率 (逗号或换行分隔)">
            <Input.TextArea rows={3} placeholder="0.01, -0.005, 0.02, ..." />
          </Form.Item>
          <Form.Item name="numAttempts" label="策略尝试次数">
            <InputNumber min={1} max={1000} />
          </Form.Item>
          <Divider plain style={{ fontSize: 13 }}>模拟交易指标</Divider>
          <Space wrap>
            <Form.Item name="paperDays" label="模拟天数"><InputNumber min={1} max={365} /></Form.Item>
            <Form.Item name="paperNetPnL" label="模拟 Net P&L"><InputNumber style={{ width: 120 }} /></Form.Item>
            <Form.Item name="paperNetReturn" label="模拟净收益"><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
            <Form.Item name="paperTradeCount" label="模拟交易数"><InputNumber min={0} max={10000} /></Form.Item>
            <Form.Item name="backtestNetReturn" label="回测净收益"><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
            <Form.Item name="backtestGrossReturn" label="回测毛收益"><InputNumber min={-1} max={10} step={0.01} /></Form.Item>
          </Space>
          <div style={{ marginTop: 12 }}>
            <Space>
              <Button type="primary" icon={loading ? <LoadingOutlined /> : <PlayCircleOutlined />} onClick={handleRun} loading={loading}>
                运行 Gate 管道
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleRetry} disabled={!loading && gates.length === 0}>重试</Button>
            </Space>
          </div>
        </Form>
      </Card>

      {error && <Alert type="error" message={error} closable style={{ marginBottom: 16 }} onClose={() => setError(null)} />}

      {(gates.length > 0 || loading) && (
        <Card size="small" title="Gate 评估进度" style={{ marginBottom: 16 }}>
          <Steps direction="vertical" size="small"
            current={loading && gates.length === 0 ? -1 : currentStep}
            status={!loading && summary && !summary.passed ? 'error' : !loading && summary?.passed ? 'finish' : 'process'}
            items={GATE_ORDER.map((gate, idx) => {
              const [icon, status, desc] = buildGateIcon(idx, gates, loading);
              return {
                title: <span>{icon}<span style={{ marginLeft: 8, fontWeight: 600 }}>{GATE_LABELS[gate]}</span><Tag style={{ marginLeft: 8, fontSize: 11 }}>{gate}</Tag></span>,
                description: <Text type={status === 'error' ? 'danger' : 'secondary'} style={{ fontSize: 12 }}>{desc}</Text>,
                status,
              };
            })}
          />
        </Card>
      )}

      {summary && (
        <Card size="small" title="管道结果">
          {summary.passed
            ? <Alert type="success" message="所有 6 个 Gate 通过，策略可进入 PromoteToLive 评估" showIcon />
            : <Alert type="error" message={`未通过: ${GATE_LABELS[summary.first_fail] || summary.first_fail}`} description={summary.summary} showIcon />
          }
          {gates.length > 0 && (
            <Collapse size="small" style={{ marginTop: 12 }} items={[{
              key: 'details', label: '详细结果',
              children: (
                <Descriptions size="small" column={1}>
                  {gates.map(gs => (
                    <Descriptions.Item key={gs.gate}
                      label={<span>{gs.passed ? <CheckCircleFilled style={{ color: '#52c41a', marginRight: 4 }} /> : <CloseCircleFilled style={{ color: '#ff4d4f', marginRight: 4 }} />}{GATE_LABELS[gs.gate] || gs.gate}</span>}
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
