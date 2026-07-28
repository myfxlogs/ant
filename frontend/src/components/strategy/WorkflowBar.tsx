import { useState, useCallback, useEffect } from 'react';
import { Modal, Input, Tooltip } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined, ThunderboltOutlined, SafetyOutlined, SaveOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { codeAssistApi, type ValidateExtendedResult } from '@/client/codeAssist';
import {
  WORKFLOW_BACKTEST_DONE_KEY, WORKFLOW_BACKTEST_FAIL_KEY, WORKFLOW_BACKTEST_HINT_DONE_KEY,
  WORKFLOW_BACKTEST_HINT_NEED_ACCOUNT_KEY, WORKFLOW_BACKTEST_HINT_NEED_REVIEW_KEY,
  WORKFLOW_BACKTEST_HINT_NEED_SYMBOL_KEY, WORKFLOW_BACKTEST_KEY, WORKFLOW_BACKTESTING_KEY,
  WORKFLOW_BACKTEST_ERROR_KEY, WORKFLOW_DIRECTIVES_LABEL_KEY,
  WORKFLOW_PARAMS_LABEL_KEY, WORKFLOW_REVIEW_ERROR_KEY, WORKFLOW_REVIEW_HINT_DONE_KEY,
  WORKFLOW_REVIEW_HINT_NEED_CODE_KEY, WORKFLOW_REVIEW_KEY, WORKFLOW_REVIEWING_KEY,
  WORKFLOW_SAVE_BTN_KEY, WORKFLOW_SAVE_FAIL_KEY, WORKFLOW_SAVE_HINT_DONE_KEY,
  WORKFLOW_SAVE_HINT_NEED_BACKTEST_KEY, WORKFLOW_SAVE_KEY, WORKFLOW_SAVE_NAME_PLACEHOLDER_KEY,
  WORKFLOW_SAVE_TITLE_KEY, WORKFLOW_SAVED_KEY, WORKFLOW_SECURITY_FAIL_KEY,
  WORKFLOW_SECURITY_PASS_KEY, WORKFLOW_SUGGESTION_LABEL_KEY, WORKFLOW_SWEEP_LABEL_KEY,
} from '@/gen/ant/v1/i18n/strategy_code_assist_keys';

type StepKey = 'check' | 'backtest' | 'save';
type StepStatus = 'idle' | 'running' | 'done' | 'failed';

interface TemplateInfo { id: string; name: string; code: string }

interface Props {
  codeRef: React.MutableRefObject<string>;
  busy: boolean;
  hasSymbol: boolean;
  accountId?: string;
  symbol?: string;
  timeframe?: string;
  templates: TemplateInfo[];
  codeGenKey: number;
  addMsg: (role: 'ai', extra: { text: string }) => void;
  fetchTemplates: () => void;
  onValidateResult?: (result: ValidateExtendedResult) => void;
  onRunBacktest?: () => void;
  backtestStatus?: string;
}

const iconStyle = { fontSize: 11 };

export default function WorkflowBar({ codeRef, busy, accountId, hasSymbol, symbol, timeframe, templates, codeGenKey, addMsg, fetchTemplates, onValidateResult, onRunBacktest, backtestStatus }: Props) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<Record<StepKey, StepStatus>>({ check: 'idle', backtest: 'idle', save: 'idle' });

  useEffect(() => {
    setStatus({ check: 'idle', backtest: 'idle', save: 'idle' });
  }, [codeGenKey]);
  const [saveOpen, setSaveOpen] = useState(false);
  const [saveName, setSaveName] = useState('');
  const [saveDup, setSaveDup] = useState(false);

  const setStep = (k: StepKey, s: StepStatus) => setStatus(prev => ({ ...prev, [k]: s }));

  // eslint-disable-next-line react-hooks/exhaustive-deps -- codeRef is a ref, not a reactive dependency
  const getCode = useCallback(() => codeRef.current, []);

  const runCheck = useCallback(async () => {
    const code = getCode(); if (!code) return;
    setStep('check', 'running'); addMsg('ai', { text: t(WORKFLOW_REVIEWING_KEY) });
    try {
      const r = await codeAssistApi.validateExtended(code);
      const parts: string[] = [];
      if (r.valid && r.errors.length === 0) {
        parts.push(t(WORKFLOW_SECURITY_PASS_KEY));
      } else {
        setStep('check', 'failed');
        parts.push(t(WORKFLOW_SECURITY_FAIL_KEY));
        r.errors.forEach(e => parts.push(`  • ${e}`));
        r.warnings.forEach(w => parts.push(`  ⚠ ${w}`));
        addMsg('ai', { text: parts.join('\n') });
        return;
      }
      if (onValidateResult) onValidateResult(r);
      if (r.parameters.length > 0) parts.push(`${t(WORKFLOW_PARAMS_LABEL_KEY)} ${r.parameters.map(p => `${p.key}${p.required ? '*' : ''}`).join(', ')}`);
      if (r.strategyDirectives.length > 0) parts.push(`${t(WORKFLOW_DIRECTIVES_LABEL_KEY)} ${r.strategyDirectives.map(d => `${d.key}=${d.value}`).join(', ')}`);
      if (r.sweepDimensions.length > 0) parts.push(`${t(WORKFLOW_SWEEP_LABEL_KEY)} ${r.sweepDimensions.length}`);
      if (r.qualityHints.length > 0) {
        const warns = r.qualityHints.filter(h => h.severity === 'warn');
        if (warns.length > 0) parts.push(`${t(WORKFLOW_SUGGESTION_LABEL_KEY)} ${warns.map(h => h.message).join('; ')}`);
      }
      setStep('check', 'done'); setStep('backtest', 'idle');
      addMsg('ai', { text: parts.join('\n') });
    } catch (e: unknown) { setStep('check', 'failed'); addMsg('ai', { text: `${t(WORKFLOW_REVIEW_ERROR_KEY)} ${e instanceof Error ? e.message : String(e)}` }); }
  }, [addMsg, t, onValidateResult, getCode]);

  const runBacktest = useCallback(async () => {
    const code = getCode(); if (!code || !accountId || !hasSymbol) return;
    setStep('backtest', 'running'); addMsg('ai', { text: t(WORKFLOW_BACKTESTING_KEY) });
    if (onRunBacktest) { onRunBacktest(); return; }
    // Fallback: direct backtest if not wired to BacktestPanel.
    try {
      const r = await strategyRuntimeApi.backtest({ code, accountId, symbol: symbol!, timeframe: timeframe!, initialCapital: 10000 });
      if (r.success && r.metrics) {
        setStep('backtest', 'done'); setStep('save', 'idle');
        addMsg('ai', { text: `${t(WORKFLOW_BACKTEST_DONE_KEY)} ${r.metrics.sharpeRatio?.toFixed(2)??'-'} | DD: ${((r.metrics.maxDrawdown??0)*100).toFixed(1)}% | WR: ${((r.metrics.winRate??0)*100).toFixed(0)}% | Trades: ${r.metrics.totalTrades??0}` });
      } else { setStep('backtest', 'failed'); addMsg('ai', { text: `${t(WORKFLOW_BACKTEST_FAIL_KEY)} ${r.error || ''}` }); }
    } catch (e: unknown) { setStep('backtest', 'failed'); addMsg('ai', { text: `${t(WORKFLOW_BACKTEST_ERROR_KEY)} ${e instanceof Error ? e.message : String(e)}` }); }
  }, [accountId, hasSymbol, symbol, timeframe, addMsg, t, onRunBacktest, getCode]);

  // Watch BacktestPanel runner status to advance workflow state.
  useEffect(() => {
    if (backtestStatus === 'completed' && status.backtest === 'running') {
      setStep('backtest', 'done'); setStep('save', 'idle');
      addMsg('ai', { text: t(WORKFLOW_BACKTEST_DONE_KEY) });
    } else if (backtestStatus === 'error' && status.backtest === 'running') {
      setStep('backtest', 'failed');
    }
  }, [backtestStatus, status.backtest, addMsg, t]);

  const openSave = useCallback(() => {
    setSaveName(''); setSaveDup(false); setSaveOpen(true);
  }, []);

  const handleSaveNameChange = useCallback((v: string) => {
    setSaveName(v);
    setSaveDup(templates.some(t => t.name === v.trim()));
  }, [templates]);

  const handleSaveConfirm = useCallback(async () => {
    const name = saveName.trim();
    if (!name || saveDup) return;
    setSaveOpen(false); setStep('save', 'running');
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      await strategyTemplateApi.create({ name, code: getCode() });
      setStep('save', 'done');
      await fetchTemplates();
      addMsg('ai', { text: `${t(WORKFLOW_SAVED_KEY)} ${name}` });
    } catch { setStep('save', 'failed'); addMsg('ai', { text: t(WORKFLOW_SAVE_FAIL_KEY) }); }
  }, [saveName, saveDup, addMsg, fetchTemplates, t, getCode]);

  const disabled = busy || !getCode();
  const canBacktest = status.check === 'done' && !disabled && hasSymbol && !!accountId;
  const canSave = status.backtest === 'done' && !disabled;

  const StatusIcon = ({ s }: { s: StepStatus }) => {
    if (s === 'running') return <LoadingOutlined style={{ color: '#1677ff', ...iconStyle }} />;
    if (s === 'done') return <CheckCircleOutlined style={{ color: '#52c41a', ...iconStyle }} />;
    if (s === 'failed') return <CloseCircleOutlined style={{ color: '#ff4d4f', ...iconStyle }} />;
    return <span style={{ color: '#d9d9d9', fontSize: 11 }}>○</span>;
  };
  const stepConfig = [
    { key: 'check' as StepKey, label: t(WORKFLOW_REVIEW_KEY), icon: <SafetyOutlined />, canRun: !disabled && status.check !== 'done', action: runCheck,
      hint: disabled ? t(WORKFLOW_REVIEW_HINT_NEED_CODE_KEY) : status.check === 'done' ? t(WORKFLOW_REVIEW_HINT_DONE_KEY) : '' },
    { key: 'backtest' as StepKey, label: t(WORKFLOW_BACKTEST_KEY), icon: <ThunderboltOutlined />, canRun: canBacktest && status.backtest !== 'done', action: runBacktest,
      hint: status.check !== 'done' ? t(WORKFLOW_BACKTEST_HINT_NEED_REVIEW_KEY)
        : !hasSymbol ? t(WORKFLOW_BACKTEST_HINT_NEED_SYMBOL_KEY)
        : !accountId ? t(WORKFLOW_BACKTEST_HINT_NEED_ACCOUNT_KEY)
        : status.backtest === 'done' ? t(WORKFLOW_BACKTEST_HINT_DONE_KEY) : '' },
    { key: 'save' as StepKey, label: t(WORKFLOW_SAVE_KEY), icon: <SaveOutlined />, canRun: canSave && status.save !== 'done', action: openSave,
      hint: status.backtest !== 'done' ? t(WORKFLOW_SAVE_HINT_NEED_BACKTEST_KEY) : status.save === 'done' ? t(WORKFLOW_SAVE_HINT_DONE_KEY) : '' },
  ];

  return (
    <>
      <div style={{ display: 'flex', gap: 8, marginTop: 6, flexWrap: 'wrap' }}>
        {stepConfig.map(s => (
          <Tooltip key={s.key} title={s.hint || undefined}>
            <div
              onClick={s.canRun ? s.action : undefined}
              style={{ display: 'flex', alignItems: 'center', gap: 5,
                padding: '4px 12px', borderRadius: 6, fontSize: 12,
                cursor: s.canRun ? 'pointer' : 'default',
                background: status[s.key] === 'done' ? '#f6ffed' : status[s.key] === 'failed' ? '#fff2f0' : '#fafafa',
                border: `1px solid ${status[s.key] === 'done' ? '#b7eb8f' : status[s.key] === 'failed' ? '#ffccc7' : '#e8e8e8'}`,
                transition: 'all 0.15s', userSelect: 'none' as const,
              }}>
              <StatusIcon s={status[s.key]} />
              {s.canRun ? <span style={{ color: '#1677ff', fontWeight: 600 }}>{s.label}</span> : <span style={{ color: '#262626', fontWeight: 500 }}>{s.label}</span>}
            </div>
          </Tooltip>
        ))}
      </div>
      <Modal title={t(WORKFLOW_SAVE_TITLE_KEY)} open={saveOpen} onOk={handleSaveConfirm} onCancel={() => setSaveOpen(false)}
        centered okText={t(WORKFLOW_SAVE_BTN_KEY)} cancelText={t('common.cancel')}
        okButtonProps={{ disabled: !saveName.trim() || saveDup }}>
        <div style={{ marginBottom: 8 }}>
          <Input placeholder={t(WORKFLOW_SAVE_NAME_PLACEHOLDER_KEY)} value={saveName} onChange={e => handleSaveNameChange(e.target.value)}
            onPressEnter={handleSaveConfirm} autoFocus style={{ fontSize: 13 }} />
        </div>
        {saveDup && <span style={{ color: '#ff4d4f', fontSize: 12 }}>⚠ {t('common.duplicateName', 'Name already exists')}</span>}
      </Modal>
    </>
  );
}
