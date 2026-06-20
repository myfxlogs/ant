import { useState, useCallback } from 'react';
import { Button } from 'antd';
import { CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined, ThunderboltOutlined, SafetyOutlined, SaveOutlined } from '@ant-design/icons';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import type { BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';

type StepKey = 'check' | 'backtest' | 'save';
type StepStatus = 'idle' | 'running' | 'done' | 'failed';

interface Props {
  codeRef: React.MutableRefObject<string>;
  busy: boolean;
  hasSymbol: boolean;
  accountId?: string;
  symbol?: string;
  timeframe?: string;
  addMsg: (role: 'ai', extra: { text: string }) => void;
  setMetrics: (m: BacktestMetricsMsg | null) => void;
  fetchTemplates: () => void;
}

const iconStyle = { fontSize: 11 };

export default function WorkflowBar({ codeRef, busy, accountId, hasSymbol, symbol, timeframe, addMsg, setMetrics, fetchTemplates }: Props) {
  const [status, setStatus] = useState<Record<StepKey, StepStatus>>({ check: 'idle', backtest: 'idle', save: 'idle' });

  const setStep = (k: StepKey, s: StepStatus) => setStatus(prev => ({ ...prev, [k]: s }));

  const getCode = () => codeRef.current;

  const runCheck = useCallback(async () => {
    const code = getCode(); if (!code) return;
    setStep('check', 'running'); addMsg('ai', { text: '🔍 合规检查中...' });
    try {
      const r = await pythonStrategyApi.validate(code);
      if (r.valid && r.errors.length === 0) {
        setStep('check', 'done'); setStep('backtest', 'idle');
        addMsg('ai', { text: '✅ 合规检查通过' });
      } else {
        setStep('check', 'failed');
        const msgs = [...r.errors, ...r.warnings].map(e => `• ${e}`).join('\n');
        addMsg('ai', { text: `❌ 合规检查未通过:\n${msgs}` });
      }
    } catch (e: any) { setStep('check', 'failed'); addMsg('ai', { text: `❌ 检查异常: ${e?.message || e}` }); }
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

  const runSave = useCallback(async () => {
    const code = getCode(); if (!code) return;
    setStep('save', 'running');
    try {
      const { strategyTemplateApi } = await import('@/client/strategy-schedules');
      const name = prompt('策略名称:'); if (!name) { setStep('save', 'idle'); return; }
      await strategyTemplateApi.create({ name, code });
      setStep('save', 'done'); fetchTemplates();
      addMsg('ai', { text: `✅ 已保存策略: ${name}` });
    } catch { setStep('save', 'failed'); addMsg('ai', { text: '❌ 保存失败' }); }
  }, [addMsg, fetchTemplates]);

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
    { key: 'check' as StepKey, label: '合规检查', icon: <SafetyOutlined />, canRun: !disabled && status.check !== 'done', action: runCheck },
    { key: 'backtest' as StepKey, label: '运行回测', icon: <ThunderboltOutlined />, canRun: canBacktest && status.backtest !== 'done', action: runBacktest },
    { key: 'save' as StepKey, label: '保存策略', icon: <SaveOutlined />, canRun: canSave && status.save !== 'done', action: runSave },
  ];

  return (
    <div style={{ display: 'flex', gap: 8, marginTop: 6, flexWrap: 'wrap' }}>
      {stepConfig.map(s => (
        <div key={s.key} style={{ display: 'flex', alignItems: 'center', gap: 4,
          padding: '3px 8px', borderRadius: 6, fontSize: 11,
          background: status[s.key] === 'done' ? '#f6ffed' : status[s.key] === 'failed' ? '#fff2f0' : '#fafafa',
          border: `1px solid ${status[s.key] === 'done' ? '#b7eb8f' : status[s.key] === 'failed' ? '#ffccc7' : '#e8e8e8'}`,
        }}>
          <StatusIcon s={status[s.key]} />
          <span style={{ color: '#262626', fontWeight: 500 }}>{s.label}</span>
          {s.canRun && <Button size="small" type="link" icon={s.icon} onClick={s.action} style={{ fontSize: 10, padding: '0 4px', height: 20 }} />}
        </div>
      ))}
    </div>
  );
}
