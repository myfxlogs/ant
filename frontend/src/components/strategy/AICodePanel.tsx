import { useState, useCallback } from 'react';
import PlanPanel from './PlanPanel';
import ExecutionPanel from './ExecutionPanel';

interface Props {
  symbol?: string;
  timeframe?: string;
  sessionId?: string;
  onApply: (code: string, previousCode?: string) => void;
}

type Stage = 'plan' | 'execute';

export default function AICodePanel({ symbol, timeframe, sessionId, onApply }: Props) {
  const [stage, setStage] = useState<Stage>('plan');
  const [plan, setPlan] = useState('');
  const [prevCode, setPrevCode] = useState<string | undefined>();

  const handlePlanConfirmed = useCallback((p: string) => {
    setPlan(p);
    setStage('execute');
  }, []);

  const handleApply = useCallback((code: string, previousCode?: string) => {
    setPrevCode(previousCode || code);
    onApply(code, previousCode);
  }, [onApply]);

  const handleReset = useCallback(() => {
    setStage('plan');
    setPlan('');
  }, []);

  if (stage === 'execute' && plan) {
    return (
      <ExecutionPanel
        plan={plan}
        symbol={symbol}
        timeframe={timeframe}
        sessionId={sessionId}
        previousCode={prevCode}
        onApply={handleApply}
        onReset={handleReset}
      />
    );
  }

  return (
    <PlanPanel
      symbol={symbol}
      timeframe={timeframe}
      sessionId={sessionId}
      onPlanConfirmed={handlePlanConfirmed}
    />
  );
}
