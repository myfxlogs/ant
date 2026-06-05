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
  code?: string; onApplyToCode?: (code: string) => void;
}

const OPTIMIZER_ICONS: Partial<Record<TuneMethod, React.ReactNode>> = {
  grid: <ThunderboltOutlined />, random: <ThunderboltOutlined />,
  de: <ExperimentOutlined />, tpe: <ExperimentOutlined />,
  ags: <ExperimentOutlined />, ai: <RobotOutlined />,
};

const gradeColors: Record<string, string> = { A: 'green', B: 'cyan', C: 'blue', D: 'orange', E: 'red' };

export default function SmartTuningPanel({
  tuneMethod, onTuneMethodChange,
  sweepDimensions = [], onToggleDimension,
  enabledSweepDims = [], cartesianSize = 0,
  tuningRunning, canRun, onRunTuning,
  code, onApplyToCode,
}: Props) {
  const [candidates, setCandidates] = useState<StrategyExperimentCandidate[]>([]);
  const [experimentId, setExperimentId] = useState('');
  const [watching, setWatching] = useState(false);
  const [showPreview, setShowPreview] = useState(false);

  const applyParamsToCode = useCallback((candidate: StrategyExperimentCandidate) => {
    if (!code || !onApplyToCode) return;
    const params = candidate.parameters as Record<string, any> | undefined;
    if (!params) return;
    let modified = code;
    for (const [key, value] of Object.entries(params)) {
      // Escape param name for safe regex use. Match both plain values and range=(...) syntax.
      const escaped = key.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
      const re = new RegExp(`(@param\\s+${escaped}\\s+)([\\d.-]+|range\\([^)]+\\))`, 'g');
      modified = modified.replace(re, `$1${String(value)}`);
    }
    onApplyToCode(modified);
  }, [code, onApplyToCode]);

  // Submit tuning job → SSE watch.
  const handleRunTuning = useCallback(async () => {
    if (!canRun || tuningRunning) return;
    setCandidates([]); setExperimentId(''); setWatching(true); onRunTuning();
  }, [canRun, tuningRunning, onRunTuning]);

  // SSE watch for experiment completion (via useRef-based polling from parent).
  useEffect(() => {
    if (!watching || !experimentId) return;
    const ctrl = new AbortController();
    (async () => {
      for await (const event of strategyExperimentApi.watchExperiment(experimentId)) {
        if (event.status === 'COMPLETED') {
          setCandidates(event.candidates || []);
          setWatching(false); break;
        }
        if (event.status === 'FAILED') { setWatching(false); break; }
      }
    })();
    return () => { ctrl.abort(); };
  }, [watching, experimentId]);

  // Preview rows: Cartesian product of enabled dimensions (max 8 shown).
  const previewRows = (() => {
    if (enabledSweepDims.length === 0) return [];
    const rows: Record<string, number>[] = [];
    const recurse = (idx: number, acc: Record<string, number>) => {
      if (idx >= enabledSweepDims.length) { rows.push({ ...acc }); return; }
      const d = enabledSweepDims[idx];
      for (const v of d.values) { acc[d.key] = v; if (rows.length < 8) recurse(idx + 1, acc); }
    };
    recurse(0, {});
    return rows;
  })();
  const previewTruncated = cartesianSize > 8;

  return (
    <div>
      {/* Optimizer selection */}
      <div style={{ marginBottom: 12 }}>
        <Typography.Text type="secondary" style={{ fontSize: 10, marginBottom: 4, display: 'block' }}>
          Optimizer method
        </Typography.Text>
        <Radio.Group value={tuneMethod} onChange={e => onTuneMethodChange(e.target.value)} size="small"
          buttonStyle="solid" style={{ display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {Object.entries(OPTIMIZER_INFO).map(([key, info]) => (
            <Tooltip key={key} title={info.desc}>
              <Radio.Button value={key} style={{ fontSize: 10, padding: '2px 8px' }}>
                {OPTIMIZER_ICONS[key as TuneMethod]} {info.label}
              </Radio.Button>
            </Tooltip>
          ))}
        </Radio.Group>
      </div>

      {/* Run button + AI hint */}
      <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <Button size="small" type="primary" loading={tuningRunning} disabled={!canRun || enabledSweepDims.length === 0}
          onClick={handleRunTuning}>{tuningRunning ? 'Tuning…' : `Run (${cartesianSize.toLocaleString()})`}</Button>
        {tuneMethod === 'ai' && <span style={{ fontSize: 10, color: '#fa8c16' }}>Requires AI provider configured</span>}
      </div>

      {/* Sweep dimensions (hidden for AI optimizer) */}
      {tuneMethod !== 'ai' && sweepDimensions.length > 0 && (
        <div style={{ marginBottom: 12, padding: 10, borderRadius: 6, border: '1px solid #e8e8e8', background: '#fcfcfd' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 6 }}>
            <Typography.Text type="secondary" style={{ fontSize: 10 }}>Parameter dimensions</Typography.Text>
            <span style={{ fontSize: 9, color: '#8c8c8c' }}>
              {enabledSweepDims.length} enabled · {cartesianSize.toLocaleString()} combinations
              {cartesianSize > 0 && (
                <Button type="link" size="small" style={{ fontSize: 10, padding: '0 4px' }}
                  onClick={() => setShowPreview(!showPreview)}>
                  {showPreview ? 'Hide' : 'Preview'}
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
                {d.source.toUpperCase()}</Tag>
              <span style={{ color: '#1890ff', fontWeight: 700 }}>×{d.values.length}</span>
              <span style={{ color: '#bfbfbf', fontSize: 9 }}>{d.values.slice(0, 5).join(', ')}{d.values.length > 5 ? '…' : ''}</span>
            </label>
          ))}
          {showPreview && previewRows.length > 0 && (
            <div style={{ marginTop: 8, padding: 8, borderRadius: 4, background: '#f6f8fa', border: '1px solid #e1e4e8' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span style={{ fontSize: 9, fontWeight: 600, color: '#595959' }}>
                  Preview ({previewRows.length} of {cartesianSize.toLocaleString()})</span>
                {previewTruncated && <Tag color="orange" style={{ fontSize: 8, lineHeight: '14px' }}>TRUNCATED</Tag>}
              </div>
              <Table dataSource={previewRows} rowKey={(_, i) => String(i)} size="small" pagination={false}
                scroll={{ x: 300 }}
                columns={Object.keys(previewRows[0] || {}).map(k => ({
                  title: k, dataIndex: k, width: 80,
                  render: (v: number) => <span style={{ fontSize: 10 }}>{v}</span>,
                }))} />
            </div>
          )}
        </div>
      )}

      {/* Results table */}
      {candidates.length > 0 && (
        <div style={{ marginTop: 12, padding: 10, borderRadius: 6, border: '1px solid #e8e8e8', background: '#fcfcfd' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <TrophyOutlined style={{ color: '#faad14' }} />
            <Typography.Text strong style={{ fontSize: 12 }}>Results ({candidates.length})</Typography.Text>
          </div>
          <Table dataSource={candidates} rowKey="id" size="small" pagination={false} scroll={{ x: 400 }}
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
                    return Object.entries(p as Record<string, unknown>).map(([k, v]) => `${k}=${v}`).join(', ');
                  } catch { return String(p); }
                }},
              { title: 'Summary', dataIndex: 'summary', ellipsis: true, width: 150,
                render: (s: string) => s || '-' },
              ...(onApplyToCode ? [{
                title: '', width: 60,
                render: (_: any, record: StrategyExperimentCandidate) => (
                  <Button size="small" type="link" style={{ fontSize: 10 }}
                    onClick={() => applyParamsToCode(record)}>Apply</Button>
                ),
              }] : []),
            ]} />
        </div>
      )}

      {watching && (
        <div style={{ textAlign: 'center', padding: 4, fontSize: 10, color: '#8c8c8c' }}>
          Waiting for experiment... (SSE auto-refresh)
        </div>
      )}
    </div>
  );
}
