import { useEffect, useRef, useState } from 'react';
import type { StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import { isNoMarketData } from './chatUtils';
import ChatAITurn from './ChatAITurn';

export type Phase = 'idle' | 'planning' | 'chatting' | 'generating' | 'compiling' | 'backtesting' | 'analyzing' | 'done';

export interface ChatTurn {
  id: string;
  role: 'user' | 'ai';
  message: string;
  timestamp?: string;
  metrics?: { label: string; value: string; positive?: boolean }[];
  plan?: StrategyPlan;
  phase?: Phase;
  streamText?: string;
  compileError?: string;
  backtestError?: string;
  error?: string;
  coverageScore?: number;
  attempts?: number;
  profile?: StrategyProfile;
  analysis?: BacktestAnalysis;
  hasCode?: boolean;
  generatedCode?: string;
  reasoning?: string;
}

interface Props {
  turns: ChatTurn[];
  onPlanConfirm?: () => void;
  onPlanRefine?: (feedback: string) => void;
  planRefining?: boolean;
  activePlanId?: string;
  onApplyCode?: (code: string) => void;
}

export default function ChatHistory({ turns, onPlanConfirm, onPlanRefine, planRefining, activePlanId, onApplyCode }: Props) {
  const endRef = useRef<HTMLDivElement>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);

  const handleCopy = (id: string, code: string) => {
    navigator.clipboard.writeText(code);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [turns.length]);

  if (turns.length === 0) return null;

  return (
    <div style={{ padding: '8px 12px' }}>
      {turns.map((turn) => {
        if (turn.role === 'user') {
          return (
            <div key={turn.id} style={{ margin: '16px 0', display: 'flex', justifyContent: 'flex-end' }}>
              <div style={{
                maxWidth: '80%',
                background: 'var(--ant-color-fill-quaternary)',
                borderRadius: '10px 10px 2px 10px',
                padding: '8px 14px',
                fontSize: 13,
                lineHeight: '20px',
                color: 'var(--ant-color-text)',
                whiteSpace: 'pre-wrap',
              }}>
                {turn.message}
              </div>
            </div>
          );
        }

        const isBusy = !!turn.phase && turn.phase !== 'idle' && turn.phase !== 'done';
        const noData = (turn.error && isNoMarketData(turn.error)) || (turn.backtestError && isNoMarketData(turn.backtestError));

        return (
          <ChatAITurn
            key={turn.id}
            turn={turn}
            isBusy={isBusy}
            noData={!!noData}
            copiedId={copiedId}
            onCopy={handleCopy}
            onPlanConfirm={onPlanConfirm}
            onPlanRefine={onPlanRefine}
            planRefining={planRefining}
            activePlanId={activePlanId}
            onApplyCode={onApplyCode}
          />
        );
      })}
      <div ref={endRef} />
    </div>
  );
}
