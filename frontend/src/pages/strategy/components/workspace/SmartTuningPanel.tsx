import { useState, useCallback, useEffect, useRef } from 'react';
import { Radio, Button, Tag, Table, Typography, Tooltip } from 'antd';
import { ExperimentOutlined, TrophyOutlined, ThunderboltOutlined, RobotOutlined } from '@ant-design/icons';
import type { SweepDimension, TuneMethod } from '../../hooks/useBacktestParams';
import { OPTIMIZER_INFO } from '../../hooks/useBacktestParams';
import { strategyExperimentApi } from '@/client/strategyExperiment';
import type { StrategyExperimentCandidate } from '@/gen/ant/v1/strategy_experiment_pb';

interface Props {
  tuneMethod: TuneMethod; onTuneMethodChange: (m: TuneMethod) => void;
  sweepDimensions: SweepDimension[]; onToggleDimension: (key: string) => void;
  enabledSweepDims: SweepDimension[]; cartesianSize: number;
  tuningRunning: boolean; canRun: boolean; onRunTuning: () => void;
  code?: string;
}

const OPTIMIZER_ICONS: Partial<Record<TuneMethod, React.ReactNode>> = {
  grid: <ThunderboltOutlined />, random: <ThunderboltOutlined />,
  de: <ExperimentOutlined />, tpe: <ExperimentOutlined />,
  ags: <ExperimentOutlined />, ai: <RobotOutlined />,
};

export default function SmartTuningPanel({
  tuneMethod, onTuneMethodChange,
  sweepDimensions = [], onToggleDimension,
  enabledSweepDims = [], cartesianSize = 0,
  tuningRunning, canRun, onRunTuning,
  code,
}: Props) {
  const [candidates, setCandidates] = useState<StrategyExperimentCandidate[]>([]);
  const [experimentId, setExperimentId] = useState('');
  const [watching, setWatching] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  const handleRunTuning = useCallback(async () => {
    setCandidates([]); setWatching(true);
    try {
      const paramSpace: Record<string, number[]> = {};
      if (enabledSweepDims.length === 0 && code) {
        // AI optimizer: no explicit param space needed, LLM proposes
      } else {
        for (const dim of enabledSweepDims) {
          if (dim.values?.length) paramSpace[dim.key] = dim.values as number[];
        }
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
    } catch { setWatching(false); }
  }, [enabledSweepDims, tuneMethod, cartesianSize, onRunTuning, code]);

  // SSE streaming
  const abortRef = useRef<AbortController | null>(null);
  useEffect(() => {
    if (!watching || !experimentId) return;
    const ctrl = new AbortController();
    abortRef.current = ctrl;
    (async () => {
      try {
        for await (const event of strategyExperimentApi.watchExperiment(experimentId)) {
          if (ctrl.signal.aborted) break;
          if (event.status === 'COMPLETED') {
            setCandidates(event.candidates || []);
            setWatching(false);
            break;
          }
          if (event.status === 'FAILED') { setWatching(false); break; }
        }
      } catch { setWatching(false); }
    })();
    return () => { ctrl.abort(); };
  }, [watching, experimentId]);

  const gradeColors: Record<string, string> = { A: 'green', B: 'blue', C: 'gold', D: 'orange', E: 'red' };
  const optInfo = OPTIMIZER_INFO[tuneMethod];

  // Compute preview: representative sample of the full Cartesian product.
  // Always show total count; show first N rows with truncation warning.
  const MAX_PREVIEW = 15;
  const previewRows = (() => {
    if (!showPreview || enabledSweepDims.length === 0) return [] as Record<string, number>[];
    let rows: Record<string, number>[] = [{}];
    for (const dim of enabledSweepDims) {
      const vals = dim.values.slice(0, 8); // cap per-dim values for preview
      const next: Record<string, number>[] = [];
      for (const row of rows) {
        for (const v of vals) {
          next.push({ ...row, [dim.key]: v });
          if (next.length >= MAX_PREVIEW * 2) break;
        }
        if (next.length >= MAX_PREVIEW * 2) break;
      }
      rows = next;
    }
    return rows.slice(0, MAX_PREVIEW);
  })();
  const previewTruncated = (() => {
    if (!showPreview) return false;
    let total = 1;
    for (const dim of enabledSweepDims) total *= dim.values.length;
    return total > previewRows.length;
  })();

  return (
    <div style={{ padding: '0 4px' }}>
      {/* Header */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12,
        padding: '10px 12px', borderRadius: 6, background: 'linear-gradient(180deg, #f0f7ff 0%, #e6f4ff 100%)',
        border: '1px solid #bae0ff' }}>
        {OPTIMIZER_ICONS[tuneMethod] || <ExperimentOutlined style={{ fontSize: 18, color: '#1890ff' }} />}
        <div>
          <div style={{ fontSize: 13, fontWeight: 700, color: '#262626' }}>Smart Tuning</div>
          <div style={{ fontSize: 10, color: '#8c8c8c', marginTop: 2 }}>
            {optInfo?.desc || 'Automatically search the optimal strategy parameters'}
          </div>
        </div>
      </div>

      {/* Optimizer selection */}
      <div style={{ padding: 12, borderRadius: 6, border: '1px solid #e8e8e8', background: '#fcfcfd' }}>
        <div style={{ fontSize: 11, fontWeight: 600, color: '#595959', marginBottom: 8 }}>
          Search Method
        </div>
        {/* Basic: one-shot, no iteration */}
        <div style={{ fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 4 }}>
          Basic (one-shot)
        </div>
        <Radio.Group value={tuneMethod} onChange={e => onTuneMethodChange(e.target.value as TuneMethod)}
          size="small" buttonStyle="solid" style={{ marginBottom: 8 }}>
          {(['grid', 'random'] as TuneMethod[]).map(m => (
            <Tooltip key={m} title={OPTIMIZER_INFO[m]?.desc}>
              <Radio.Button value={m} style={{ fontSize: 10 }}>
                {OPTIMIZER_INFO[m]?.label || m.toUpperCase()}
              </Radio.Button>
            </Tooltip>
          ))}
        </Radio.Group>

        {/* Advanced: iterative ask/tell with convergence */}
        <div style={{ fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 4 }}>
          Advanced (iterative convergence)
        </div>
        <Radio.Group value={tuneMethod} onChange={e => onTuneMethodChange(e.target.value as TuneMethod)}
          size="small" buttonStyle="solid" style={{ marginBottom: 10 }}>
          {(['de', 'tpe', 'ags', 'ai'] as TuneMethod[]).map(m => (
            <Tooltip key={m} title={OPTIMIZER_INFO[m]?.desc}>
              <Radio.Button value={m} style={{ fontSize: 10 }}>
                {OPTIMIZER_INFO[m]?.label || m.toUpperCase()}
              </Radio.Button>
            </Tooltip>
          ))}
        </Radio.Group>

        {/* Run button */}
        <div style={{ marginBottom: tuneMethod !== 'ai' ? 12 : 0 }}>
          <Button type="primary" size="small" loading={tuningRunning || watching}
            disabled={!canRun && tuneMethod !== 'ai'}
            onClick={handleRunTuning} style={{ borderRadius: 6, fontWeight: 600 }}>
            {(tuningRunning || watching) ? 'Running...' : `▶ Run ${optInfo?.label || 'Tuning'}`}
          </Button>
          {tuneMethod === 'ai' && (
            <span style={{ fontSize: 10, color: '#8c8c8c', marginLeft: 8 }}>
              Requires AI provider configured in ⚙ Settings
            </span>
          )}
        </div>

        {/* Sweep dimensions (not shown for AI optimizer) */}
        {tuneMethod !== 'ai' && (
          <div>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
              <span style={{ fontSize: 10, fontWeight: 600, color: '#595959' }}>📐 Parameters</span>
              <span style={{ fontSize: 10, color: '#8c8c8c' }}>
                {enabledSweepDims.length}/{sweepDimensions.length} enabled · {cartesianSize.toLocaleString()} combos
                {cartesianSize > 0 && (
                  <Button type="link" size="small" style={{ fontSize: 10, padding: '0 4px' }}
                    onClick={() => setShowPreview(!showPreview)}>
                    {showPreview ? 'Hide preview' : 'Preview'}
                  </Button>
                )}
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
                <span style={{ color: '#bfbfbf', fontSize: 9 }}>{d.values.slice(0, 5).join(', ')}{d.values.length > 5 ? '…' : ''}</span>
              </label>
            ))}

            {/* Preview matrix */}
            {showPreview && previewRows.length > 0 && (
              <div style={{ marginTop: 8, padding: 8, borderRadius: 4, background: '#f6f8fa', border: '1px solid #e1e4e8' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                  <span style={{ fontSize: 9, fontWeight: 600, color: '#595959' }}>
                    Preview (first {previewRows.length} of {cartesianSize.toLocaleString()} combinations)
                  </span>
                  {previewTruncated && (
                    <span style={{ fontSize: 9, color: '#faad14' }}>
                      ⚠ showing {previewRows.length}/{cartesianSize.toLocaleString()}
                    </span>
                  )}
                </div>
                <Table
                  dataSource={previewRows.map((r, i) => ({ ...r, _key: i }))}
                  rowKey="_key"
                  size="small"
                  pagination={false}
                  scroll={{ x: 300 }}
                  columns={Object.keys(previewRows[0] || {}).map(k => ({
                    title: k, dataIndex: k, width: 80,
                    render: (v: number) => <span style={{ fontSize: 10 }}>{v}</span>,
                  }))}
                />
              </div>
            )}
          </div>
        )}
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
                    const obj = p as Record<string, unknown>;
                    return Object.entries(obj).map(([k, v]) => `${k}=${v}`).join(', ');
                  } catch { return String(p); }
                }},
              { title: 'Summary', dataIndex: 'summary', ellipsis: true, width: 150,
                render: (s: string) => s || '-' },
            ]}
          />
        </div>
      )}

      {watching && (
        <div style={{ textAlign: 'center', padding: 4, fontSize: 10, color: '#8c8c8c' }}>
          Waiting for experiment to complete... (auto-refreshing via SSE)
        </div>
      )}
    </div>
  );
}
