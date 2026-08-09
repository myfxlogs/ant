import { useState, useCallback, useEffect } from 'react';
import { Radio, Button, Checkbox, Tag, Table, Typography, Alert, Tooltip } from 'antd';
import { ExperimentOutlined, TrophyOutlined, ThunderboltOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation, Trans } from 'react-i18next';
import { APPLY_KEY, DEGRADATION_KEY, ENABLED_COMBINATIONS_KEY, GRADE_KEY, GRID_WARNING_KEY, HIDE_KEY, OOS_FOOTNOTE_KEY, OOS_SCORE_KEY, OPTIMIZER_METHOD_KEY, OVERFIT_KEY, OVERFIT_WARNING_KEY, PARAMETERS_KEY, PARAMETER_DIMENSIONS_KEY, PREVIEW_KEY, PREVIEW_TITLE_KEY, RANK_KEY, REQUIRES_A_I_KEY, RESULTS_KEY, RUN_KEY, SCORE_KEY, SUMMARY_KEY, SWITCH_TO_D_E_KEY, TRUNCATED_KEY, TUNING_KEY, WAITING_KEY } from '@/gen/ant/v1/i18n/strategy_tuning_keys';
import type { SweepDimension, TuneMethod } from '../../hooks/useBacktestParams';
import { OPTIMIZER_INFO } from '../../hooks/useBacktestParams';
import { strategyExperimentApi } from '@/client/strategyExperiment';
import type { StrategyExperimentCandidate } from '@/gen/ant/v1/strategy_experiment_pb';

interface Props {
  tuneMethod: TuneMethod; onTuneMethodChange: (m: TuneMethod) => void;
  sweepDimensions: SweepDimension[]; onToggleDimension: (key: string) => void;
  enabledSweepDims: SweepDimension[]; cartesianSize: number;
  tuningRunning: boolean; canRun: boolean; onRunTuning: () => Promise<string>;
  code?: string; onApplyToCode?: (code: string) => void;
}

const OPTIMIZER_ICONS: Partial<Record<TuneMethod, React.ReactNode>> = { grid: <ThunderboltOutlined />, random: <ThunderboltOutlined />, de: <ExperimentOutlined />, tpe: <ExperimentOutlined />, ags: <ExperimentOutlined />, ai: <RobotOutlined /> };
const gradeColors: Record<string, string> = { A: 'green', B: 'cyan', C: 'blue', D: 'orange', E: 'red' };

function optLabel(t: (k: string) => string, key: TuneMethod): string { return t(`strategy.tuning.optimizer.${key}`); }
function optDesc(t: (k: string) => string, key: TuneMethod): string { return t(`strategy.tuning.optimizer.${key}Desc`); }

export default function SmartTuningPanel({
  tuneMethod, onTuneMethodChange,
  sweepDimensions = [], onToggleDimension,
  enabledSweepDims = [], cartesianSize = 0,
  tuningRunning, canRun, onRunTuning,
  code, onApplyToCode,
}: Props) {
  const { t } = useTranslation();
  const [candidates, setCandidates] = useState<StrategyExperimentCandidate[]>([]);
  const [experimentId, setExperimentId] = useState('');
  const [watching, setWatching] = useState(false);
  const [showPreview, setShowPreview] = useState(false);
  const applyParamsToCode = useCallback((candidate: StrategyExperimentCandidate) => {
    if (!code || !onApplyToCode) return;
    const params = candidate.parameters as Record<string, unknown> | undefined;
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

  // Submit tuning job → receive experiment ID → SSE watch.
  const handleRunTuning = useCallback(async () => {
    if (!canRun || tuningRunning) return;
    setCandidates([]); setExperimentId(''); setWatching(true);
    try {
      const eid = await onRunTuning();
      if (eid) { setExperimentId(eid); } else { setWatching(false); }
    } catch {
      setWatching(false);
    }
  }, [canRun, tuningRunning, onRunTuning]);

  // SSE watch for experiment completion.
  useEffect(() => {
    if (!watching || !experimentId) return;
    const ctrl = new AbortController();
    (async () => {
      try {
        const { stream } = strategyExperimentApi.watchExperiment(experimentId);
        for await (const event of stream) {
          if (event.status === 'COMPLETED') { setCandidates(event.candidates || []); setWatching(false); break; }
          if (event.status === 'FAILED') { setWatching(false); break; }
        }
      } catch (e: unknown) {
        if ((e as { name?: string })?.name !== 'AbortError') { setWatching(false); }
      }
    })();
    return () => { ctrl.abort(); };
  }, [watching, experimentId]);

  // Preview rows: Cartesian product of enabled dimensions (max 8 shown).
  const previewRows: Record<string, number>[] = [];
  if (enabledSweepDims.length > 0) {
    const recurse = (idx: number, acc: Record<string, number>) => {
      if (idx >= enabledSweepDims.length) { previewRows.push({ ...acc }); return; }
      const d = enabledSweepDims[idx];
      for (const v of d.values) { acc[d.key] = v; if (previewRows.length < 8) recurse(idx + 1, acc); }
    };
    recurse(0, {});
  }
  const previewTruncated = cartesianSize > 8;

  return (
    <div style={{ fontSize: 13 }}>
      {/* No @param guidance */}
      {sweepDimensions.length === 0 && (
        <Alert type="info" showIcon style={{ marginBottom: 12, fontSize: 13 }}
          message={t('strategy.tuning.noParams.title', { defaultValue: 'No tunable parameters detected' })}
          description={<span style={{ fontSize: 12 }}>{t('strategy.tuning.noParams.desc', { defaultValue: 'Add @param annotations to your strategy code to enable Smart Tuning. Example: // @param fastPeriod 14 range=5:30:5' })}</span>} />
      )}

      {/* Optimizer selection */}
      <div style={{ marginBottom: 12 }}>
        <Typography.Text type="secondary" style={{ fontSize: 13, marginBottom: 6, display: 'block', fontWeight: 600 }}>
          {t(OPTIMIZER_METHOD_KEY)}
        </Typography.Text>
        <Radio.Group value={tuneMethod} onChange={e => onTuneMethodChange(e.target.value)} size="middle"
          buttonStyle="solid" style={{ flexWrap: 'wrap' }}>
          {Object.keys(OPTIMIZER_INFO).map((key) => (
            <Radio.Button key={key} value={key} style={{ fontSize: 12, padding: '4px 12px' }}
              title={optDesc(t, key as TuneMethod)}>
              {OPTIMIZER_ICONS[key as TuneMethod]} {optLabel(t, key as TuneMethod)}
            </Radio.Button>
          ))}
        </Radio.Group>
      </div>

      {/* Budget auto-suggest: Grid → DE when Cartesian product exceeds 48 */}
      {tuneMethod === 'grid' && cartesianSize > 48 && (
        <Alert
          type="info" showIcon style={{ marginBottom: 12, padding: '8px 12px', fontSize: 12 }}
          message={
            <span style={{ fontSize: 12 }}>
              <Trans i18nKey={GRID_WARNING_KEY} components={{ b: <b /> }} values={{ count: cartesianSize.toLocaleString() }} />
            </span>
          }
          action={
            <Button size="small" type="primary" onClick={() => onTuneMethodChange('de')}>
              {t(SWITCH_TO_D_E_KEY)}
            </Button>
          }
        />
      )}

      {/* Run button + AI hint */}
      <div style={{ marginBottom: 12, display: 'flex', alignItems: 'center', gap: 8 }}>
        <Tooltip title={!canRun ? t('strategy.tuning.disabledHint', { defaultValue: 'Need strategy code and symbol. Select a strategy from the sidebar or run a backtest first.' }) : enabledSweepDims.length === 0 ? t('strategy.tuning.noDimsHint', { defaultValue: 'Enable at least one parameter dimension below.' }) : ''}>
          <Button size="middle" type="primary" loading={tuningRunning} disabled={!canRun || enabledSweepDims.length === 0}
            onClick={handleRunTuning} style={{ fontSize: 13 }}>{tuningRunning ? t(TUNING_KEY) : t(RUN_KEY, { count: cartesianSize.toLocaleString() })}</Button>
        </Tooltip>
        {tuneMethod === 'ai' && <span style={{ fontSize: 12, color: '#fa8c16' }}>{t(REQUIRES_A_I_KEY)}</span>}
      </div>
      {!canRun && (
        <div style={{ marginBottom: 12, padding: '6px 10px', borderRadius: 4, background: '#fffbe6', border: '1px solid #ffe58f', fontSize: 12, color: '#8c6d1f' }}>
          {t('strategy.tuning.disabledHint', { defaultValue: 'Need strategy code and symbol. Select a strategy from the sidebar or run a backtest first.' })}
        </div>
      )}

      {/* Sweep dimensions (hidden for AI optimizer) */}
      {tuneMethod !== 'ai' && sweepDimensions.length > 0 && (
        <div style={{ marginBottom: 12, padding: 12, borderRadius: 6, border: '1px solid #d9d9d9', background: '#fafbfc' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 8 }}>
            <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: 600 }}>{t(PARAMETER_DIMENSIONS_KEY)}</Typography.Text>
            <span style={{ fontSize: 12, color: '#595959' }}>
              {t(ENABLED_COMBINATIONS_KEY, { enabled: enabledSweepDims.length, combos: cartesianSize.toLocaleString() })}
              {cartesianSize > 0 && (
                <Button type="link" size="small" style={{ fontSize: 12, padding: '0 4px' }}
                  onClick={() => setShowPreview(!showPreview)}>
                  {showPreview ? t(HIDE_KEY) : t(PREVIEW_KEY)}
                </Button>
              )}
            </span>
          </div>
          {sweepDimensions.map(d => (
            <label key={d.key} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '5px 10px',
              fontSize: 13, borderRadius: 4, marginBottom: 3, cursor: 'pointer',
              background: d.enabled ? 'rgba(24,144,255,0.06)' : 'transparent', opacity: d.enabled ? 1 : 0.4 }}>
              <Checkbox checked={d.enabled} onChange={() => onToggleDimension(d.key)} />
              <span style={{ flex: 1, fontWeight: 500, color: '#262626' }}>{d.label}</span>
              <Tag color={d.source === 'code' ? 'blue' : 'orange'} style={{ fontSize: 10, lineHeight: '16px', margin: 0 }}>
                {d.source.toUpperCase()}</Tag>
              <span style={{ color: '#1890ff', fontWeight: 700, fontSize: 13 }}>×{d.values.length}</span>
              <span style={{ color: '#8c8c8c', fontSize: 11 }}>{d.values.slice(0, 5).join(', ')}{d.values.length > 5 ? '…' : ''}</span>
            </label>
          ))}
          {showPreview && previewRows.length > 0 && (
            <div style={{ marginTop: 8, padding: 8, borderRadius: 4, background: '#f6f8fa', border: '1px solid #e1e4e8' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
                <span style={{ fontSize: 12, fontWeight: 600, color: '#595959' }}>
                  {t(PREVIEW_TITLE_KEY, { shown: previewRows.length, total: cartesianSize.toLocaleString() })}</span>
                {previewTruncated && <Tag color="orange" style={{ fontSize: 10, lineHeight: '16px' }}>{t(TRUNCATED_KEY)}</Tag>}
              </div>
              <Table dataSource={previewRows} rowKey={(_, i) => String(i)} size="small" pagination={false}
                scroll={{ x: 300 }}
                columns={Object.keys(previewRows[0] || {}).map(k => ({
                  title: k, dataIndex: k, width: 80,
                  render: (v: number) => <span style={{ fontSize: 12 }}>{v}</span>,
                }))} />
            </div>
          )}
        </div>
      )}

      {/* Results table */}
      {candidates.length > 0 && (
        <div style={{ marginTop: 12, padding: 12, borderRadius: 6, border: '1px solid #d9d9d9', background: '#fafbfc' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6, marginBottom: 8 }}>
            <TrophyOutlined style={{ color: '#faad14', fontSize: 16 }} />
            <Typography.Text strong style={{ fontSize: 14 }}>{t(RESULTS_KEY, { count: candidates.length })}</Typography.Text>
          </div>
          <Table dataSource={candidates} rowKey="id" size="small" pagination={false} scroll={{ x: 700 }}
            columns={[
              { title: t(RANK_KEY), dataIndex: 'rank', width: 40, render: (v: number) => v || '-' },
              { title: t(GRADE_KEY), dataIndex: 'grade', width: 60, render: (g: string) => <Tag color={gradeColors[g] || 'default'}>{g || 'C'}</Tag> },
              { title: t(SCORE_KEY), dataIndex: 'score', width: 60, render: (s: number) => s > 0 ? s.toFixed(1) : '-' },
              { title: t(PARAMETERS_KEY), dataIndex: 'parameters', ellipsis: true,
                render: (p: unknown) => {
                  if (!p) return '-';
                  try { return Object.entries(p as Record<string, unknown>).map(([k, v]) => `${k}=${v}`).join(', '); }
                  catch { return String(p); }
                }},
              { title: t(SUMMARY_KEY), dataIndex: 'summary', ellipsis: true, width: 150, render: (s: string) => s || '-' },
              { title: t(OOS_SCORE_KEY), dataIndex: 'oosScore', width: 70, render: (s: number | undefined) => s != null ? s.toFixed(1) : '-' },
              { title: t(DEGRADATION_KEY), dataIndex: 'degradationPct', width: 90, render: (pct: number | undefined) => {
                  if (pct == null) return '-';
                  const c = pct < 20 ? 'green' : pct < 40 ? 'orange' : 'red';
                  return <Tag color={c} style={{ fontSize: 11, margin: 0 }}>{pct.toFixed(1)}%</Tag>;
                }},
              { title: t(OVERFIT_KEY), dataIndex: 'isOverfit', width: 70, render: (v: boolean) => v ? <Tag color="red" style={{ fontSize: 11, margin: 0 }}>{t(OVERFIT_WARNING_KEY)}</Tag> : <span style={{ color: '#bfbfbf', fontSize: 12 }}>-</span> },
              ...(onApplyToCode ? [{
                title: '', width: 120,
                render: (_: unknown, record: StrategyExperimentCandidate) => (
                  <div style={{ display: 'flex', gap: 4 }}>
                    <Button size="small" type="link" style={{ fontSize: 12 }}
                      onClick={() => applyParamsToCode(record)}>{t(APPLY_KEY)}</Button>
                    {record.backtestRunId && (
                      <Button size="small" type="link" style={{ fontSize: 12 }}
                        onClick={() => window.open(`/strategy?runId=${record.backtestRunId}`, '_blank')}>
                        {t('strategy.tuning.qualityGate', { defaultValue: 'Gate' })}
                      </Button>
                    )}
                  </div>
                ),
              }] : []),
            ]} />
            {candidates.some(c => c.oosScore != null) && (
              <Typography.Text type="secondary" style={{ fontSize: 11, display: 'block', marginTop: 4 }}>{t(OOS_FOOTNOTE_KEY)}</Typography.Text>
            )}
        </div>
      )}

      {watching && (
        <div style={{ textAlign: 'center', padding: 8, fontSize: 13, color: '#8c8c8c' }}>
          {t(WAITING_KEY)}
        </div>
      )}
    </div>
  );
}
