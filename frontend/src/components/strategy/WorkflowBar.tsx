import { useState, useCallback, useEffect } from 'react';
import { Modal, Input, Tooltip } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined, ThunderboltOutlined, SafetyOutlined, SaveOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { codeAssistApi } from '@/client/codeAssist';
import type { BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';

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
  setMetrics: (m: BacktestMetricsMsg | null) => void;
  fetchTemplates: () => void;
}

const iconStyle = { fontSize: 11 };

export default function WorkflowBar({ codeRef, busy, accountId, hasSymbol, symbol, timeframe, templates, codeGenKey, addMsg, setMetrics, fetchTemplates }: Props) {
  const [status, setStatus] = useState<Record<StepKey, StepStatus>>({ check: 'idle', backtest: 'idle', save: 'idle' });

  // Reset workflow when code changes (agent regenerated after failed check)
  useEffect(() => {
    setStatus({ check: 'idle', backtest: 'idle', save: 'idle' });
  }, [codeGenKey]);
  const [saveOpen, setSaveOpen] = useState(false);
  const [saveName, setSaveName] = useState('');
  const [saveDup, setSaveDup] = useState(false);

  const setStep = (k: StepKey, s: StepStatus) => setStatus(prev => ({ ...prev, [k]: s }));

  const getCode = () => codeRef.current;

  const runCheck = useCallback(async () => {
    const code = getCode(); if (!code) return;
    setStep('check', 'running'); addMsg('ai', { text: '🔍 策略审查中...' });
    try {
      const r = await codeAssistApi.validateExtended(code);
      const parts: string[] = [];
      if (r.valid && r.errors.length === 0) {
        parts.push('✅ 安全检测通过');
      } else {
        setStep('check', 'failed');
        parts.push('❌ 安全检测未通过:');
        r.errors.forEach(e => parts.push(`  • ${e}`));
        r.warnings.forEach(w => parts.push(`  ⚠ ${w}`));
        addMsg('ai', { text: parts.join('\n') });
        return;
      }
      if (r.parameters.length > 0) parts.push(`📐 可调参数: ${r.parameters.map(p => `${p.key}${p.required ? '*' : ''}`).join(', ')}`);
      if (r.strategyDirectives.length > 0) parts.push(`⚙ 策略指令: ${r.strategyDirectives.map(d => `${d.key}=${d.value}`).join(', ')}`);
      if (r.sweepDimensions.length > 0) parts.push(`🔀 扫参维度: ${r.sweepDimensions.length} 维`);
      if (r.qualityHints.length > 0) {
        const warns = r.qualityHints.filter(h => h.severity === 'warn');
        if (warns.length > 0) parts.push(`💡 建议: ${warns.map(h => h.message).join('; ')}`);
      }
      setStep('check', 'done'); setStep('backtest', 'idle');
      addMsg('ai', { text: parts.join('\n') });
    } catch (e: any) { setStep('check', 'failed'); addMsg('ai', { text: `❌ 审查异常: ${e?.message || e}` }); }
  }, [addMsg]);

  const runBacktest = useCallback(async () => {
    const code = getCode(); if (!code || !accountId || !hasSymbol) return;
    setStep('backtest', 'running'); addMsg('ai', { text: '⚡ 正在启动回测...' });
    try {
      const r = await pythonStrategyApi.backtest({ code, accountId, symbol: symbol!, timeframe: timeframe!, initialCapital: 10000 });
      if (r.success && r.metrics) {
        setMetrics({ totalReturn: r.metrics.totalReturn, sharpeRatio: r.metrics.sharpeRatio, maxDrawdown: r.metrics.maxDrawdown, winRate: r.metrics.winRate, totalTrades: r.metrics.totalTrades, profitFactor: r.metrics.profitFactor });
        setStep('backtest', 'done'); setStep('save', 'idle');
        addMsg('ai', { text: `✅ 回测完成 | Sharpe: ${r.metrics.sharpeRatio?.toFixed(2)??'-'} | 回撤: ${((r.metrics.maxDrawdown??0)*100).toFixed(1)}% | 胜率: ${((r.metrics.winRate??0)*100).toFixed(0)}% | 交易: ${r.metrics.totalTrades??0}次` });
      } else { setStep('backtest', 'failed'); addMsg('ai', { text: `❌ 回测失败: ${r.error || '未知错误'}` }); }
    } catch (e: any) { setStep('backtest', 'failed'); addMsg('ai', { text: `❌ 回测异常: ${e?.message || e}` }); }
  }, [accountId, hasSymbol, symbol, timeframe, addMsg, setMetrics]);

  // Open save modal — validate name uniqueness
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
      addMsg('ai', { text: `✅ 已保存策略: ${name}` });
    } catch { setStep('save', 'failed'); addMsg('ai', { text: '❌ 保存失败' }); }
  }, [saveName, saveDup, addMsg, fetchTemplates]);

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
    { key: 'check' as StepKey, label: '策略审查', icon: <SafetyOutlined />, canRun: !disabled && status.check !== 'done', action: runCheck,
      hint: disabled ? '需要代码' : status.check === 'done' ? '已完成' : '' },
    { key: 'backtest' as StepKey, label: '运行回测', icon: <ThunderboltOutlined />, canRun: canBacktest && status.backtest !== 'done', action: runBacktest,
      hint: status.check !== 'done' ? '请先完成策略审查'
        : !hasSymbol ? '请选择交易品种和周期'
        : !accountId ? '请选择交易账户'
        : status.backtest === 'done' ? '已完成' : '' },
    { key: 'save' as StepKey, label: '保存策略', icon: <SaveOutlined />, canRun: canSave && status.save !== 'done', action: openSave,
      hint: status.backtest !== 'done' ? '请先完成回测' : status.save === 'done' ? '已保存' : '' },
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
      <Modal title="保存策略" open={saveOpen} onOk={handleSaveConfirm} onCancel={() => setSaveOpen(false)}
        centered okText="保存" cancelText="取消"
        okButtonProps={{ disabled: !saveName.trim() || saveDup }}>
        <div style={{ marginBottom: 8 }}>
          <Input placeholder="输入策略名称" value={saveName} onChange={e => handleSaveNameChange(e.target.value)}
            onPressEnter={handleSaveConfirm} autoFocus style={{ fontSize: 13 }} />
        </div>
        {saveDup && (
          <div style={{ color: '#ff4d4f', fontSize: 12, display: 'flex', alignItems: 'center', gap: 4 }}>
            <ExclamationCircleOutlined /> 名称已存在，请换个名字
          </div>
        )}
      </Modal>
    </>
  );
}
