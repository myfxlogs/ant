import { useState, useCallback, useEffect, useRef } from 'react';
import { Radio, Button, Tag, Table, Typography } from 'antd';
import { ExperimentOutlined, TrophyOutlined } from '@ant-design/icons';
import type { SweepDimension } from '../../hooks/useBacktestParams';
import { strategyExperimentApi } from '@/client/strategyExperiment';
import type { StrategyExperimentCandidate } from '@/gen/ant/v1/strategy_experiment_pb';

interface Props {
  tuneMethod: 'grid' | 'random'; onTuneMethodChange: (m: string) => void;
  sweepDimensions: SweepDimension[]; onToggleDimension: (key: string) => void;
  enabledSweepDims: SweepDimension[]; cartesianSize: number;
  tuningRunning: boolean; canRun: boolean; onRunTuning: () => void;
}

export default function SmartTuningPanel({
  tuneMethod, onTuneMethodChange,
  sweepDimensions = [], onToggleDimension,
  enabledSweepDims = [], cartesianSize = 0,
  tuningRunning, canRun, onRunTuning,
}: Props) {
  const [candidates, setCandidates] = useState<StrategyExperimentCandidate[]>([]);
  const [experimentId, setExperimentId] = useState('');
  const [pollingRunning, setPollingRunning] = useState(false);

  const handleRunTuning = useCallback(async () => {
    setCandidates([]); setPollingRunning(true);
    try {
      const paramSpace: Record<string, number[]> = {};
      for (const dim of enabledSweepDims) {
        if (dim.values?.length) paramSpace[dim.key] = dim.values as number[];
      }
      const resp = await strategyExperimentApi.submit({
        baseTemplateId: '',
        parameterSpace: paramSpace as Record<string, unknown>,
        searchMethod: tuneMethod,
        maxCandidates: Math.min(cartesianSize || 24, 48),
        objective: 'balanced',
      });
      setExperimentId(resp.experiment?.id || '');
      onRunTuning();
    } catch { setPollingRunning(false); }
  }, [enabledSweepDims, tuneMethod, cartesianSize, onRunTuning]);

  const handlePoll = useCallback(async () => {
    if (!experimentId) return;
    const exps = await strategyExperimentApi.list();
    const exp = exps.find(e => e.id === experimentId);
    if (exp?.status === 'COMPLETED' || exp?.status === 'FAILED') {
      setPollingRunning(false);
      if (exp.status === 'COMPLETED') {
        const cands = await strategyExperimentApi.listCandidates(experimentId);
        setCandidates(cands);
      }
    }
  }, [experimentId]);

  // Auto-poll every 5s when running (useEffect with cleanup)
  const pollRef = useRef<ReturnType<typeof setInterval>>();
  useEffect(() => {
    if (!pollingRunning || !experimentId) return;
    pollRef.current = setInterval(handlePoll, 5000);
    return () => { if (pollRef.current) clearInterval(pollRef.current); };
  }, [pollingRunning, experimentId, handlePoll]);

  const gradeColors: Record<string, string> = { A: 'green', B: 'blue', C: 'gold', D: 'orange', E: 'red' };

  return (
    <div style={{ padding: '0 4px' }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12,
        padding: '10px 12px', borderRadius: 6, background: 'linear-gradient(180deg, #f0f7ff 0%, #e6f4ff 100%)',
        border: '1px solid #bae0ff' }}>
        <ExperimentOutlined style={{ fontSize: 18, color: '#1890ff' }} />
        <div>
          <div style={{ fontSize: 13, fontWeight: 700, color: '#262626' }}>Smart Tuning</div>
          <div style={{ fontSize: 10, color: '#8c8c8c', marginTop: 2 }}>
            Automatically search the optimal strategy parameters
          </div>
        </div>
      </div>

      <div style={{ padding: 12, borderRadius: 6, border: '1px solid #e8e8e8', background: '#fcfcfd' }}>
        <div style={{ fontSize: 11, fontWeight: 600, color: '#595959', marginBottom: 10 }}>
          Structured scan (no LLM)
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12 }}>
          <Radio.Group value={tuneMethod} onChange={e => onTuneMethodChange(e.target.value)}
            size="small" buttonStyle="solid">
            <Radio.Button value="grid">Grid</Radio.Button>
            <Radio.Button value="random">Random</Radio.Button>
          </Radio.Group>
          <Button type="primary" size="small" loading={tuningRunning || pollingRunning}
            disabled={!canRun} onClick={handleRunTuning} style={{ borderRadius: 6, fontWeight: 600 }}>
            {(tuningRunning || pollingRunning) ? 'Running...' : '▶ Run Smart Tuning'}
          </Button>
        </div>

        {/* Sweep Dimensions */}
        <div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
            <span style={{ fontSize: 10, fontWeight: 600, color: '#595959' }}>📐 Sweep Dimensions</span>
            <span style={{ fontSize: 10, color: '#8c8c8c' }}>
              {enabledSweepDims.length}/{sweepDimensions.length} enabled · {cartesianSize.toLocaleString()} combos
            </span>
          </div>
          {sweepDimensions.map(d => (
            <label key={d.key} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '3px 8px',
              fontSize: 10, borderRadius: 4, marginBottom: 2, cursor: 'pointer',
              background: d.enabled ? 'rgba(24,144,255,0.06)' : 'transparent', opacity: d.enabled ? 1 : 0.4 }}>
              <input type="checkbox" checked={d.enabled} onChange={() => onToggleDimension(d.key)} />
              <span style={{ flex: 1, fontWeight: 500, color: '#262626' }}>{d.label}</span>
              <Tag color={d.source === 'code' ? 'blue' : 'orange'} style={{ fontSize: 8, lineHeight: '14px', margin: 0 }}>
                {d.source.toUpperCase()}
              </Tag>
              <span style={{ color: '#1890ff', fontWeight: 700 }}>×{d.values.length}</span>
              <span style={{ color: '#bfbfbf', fontSize: 9 }}>{d.values.join(', ')}</span>
            </label>
          ))}
        </div>
      </div>

      {/* Results table */}
      {candidates.length > 0 && (
        <div style={{ marginTop: 12, padding: 10, borderRadius: 6, border: '1px solid #e8e8e8', background: '#fcfcfd' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <TrophyOutlined style={{ color: '#faad14' }} />
            <Typography.Text strong style={{ fontSize: 12 }}>Results ({candidates.length} candidates)</Typography.Text>
          </div>
          <Table
            dataSource={candidates}
            rowKey="id"
            size="small"
            pagination={false}
            scroll={{ x: 400 }}
            columns={[
              { title: '#', dataIndex: 'rank', width: 40, render: (v: number) => v || '-' },
              { title: 'Grade', dataIndex: 'grade', width: 60,
                render: (g: string) => <Tag color={gradeColors[g] || 'default'}>{g || 'C'}</Tag> },
              { title: 'Score', dataIndex: 'score', width: 60,
                render: (s: number) => s > 0 ? s.toFixed(1) : '-' },
              { title: 'Parameters', dataIndex: 'parameters', ellipsis: true,
                render: (p: unknown) => {
                  if (!p) return '-';
                  try {
                    const obj = p;
                    return Object.entries(obj as Record<string, unknown>).map(
                      ([k, v]) => `${k}=${v}`).join(', ');
                  } catch { return String(p); }
                }},
              { title: 'Summary', dataIndex: 'summary', ellipsis: true, width: 150,
                render: (s: string) => s || '-' },
            ]}
          />
        </div>
      )}

      {/* Polling indicator */}
      {pollingRunning && (
        <div style={{ textAlign: 'center', padding: 4, fontSize: 10, color: '#8c8c8c' }}>
          Waiting for experiment to complete... (auto-refreshing every 5s)
        </div>
      )}
    </div>
  );
}
