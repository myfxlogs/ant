/** Ask the LLM to generate a short conversation title from the first user message. */
export async function generateTitle(firstMsg: string): Promise<string> {
  try {
    const prompt = `你是一个标题生成器。根据用户的第一条消息，生成一个简短的对话标题（3-8个字）。只返回标题本身，不要加引号、句号或任何额外文字。\n\n用户消息: "${firstMsg}"\n\n标题:`;
    const { aiApi } = await import('@/client/ai');
    const result = await aiApi.chat({ message: prompt });
    const title = (result.message || '').replace(/^["'「『]|["'」』]$/g, '').trim();
    return title.slice(0, 30) || firstMsg.slice(0, 20);
  } catch {
    return firstMsg.slice(0, 20);
  }
}

export type TabKey = 'chat' | 'history' | 'strategies';

export type Conversation = { id: string; title: string; created_at: string };

export type Template = { id: string; name: string; code: string };

import { conversate, type ConversateCallbacks } from '@/client/strategyPlan';
import { strategyRuntimeApi } from '@/client/strategyRuntime';
import { isSucceededRun } from '@/pages/strategy/StrategyTemplatePage.utils';
import type { ToolResult, BacktestMetricsMsg } from '@/gen/ant/v1/strategy_execution_pb';
import { BacktestMetricsMsgSchema } from '@/gen/ant/v1/strategy_execution_pb';
import { create } from '@bufbuild/protobuf';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';
import {
  BACKTEST_LABEL_KEY, COMPLIANCE_CHECK_KEY, CONTINUE_MESSAGE_KEY, PASSED_KEY,
} from '@/gen/ant/v1/i18n/strategy_ai_chat_keys';
import type { ChatMsg } from './ChatMessageItem';

interface RunConversateArgs {
  msg: string;
  curCode: string;
  isFirst: boolean;
  sessionId?: string;
  symbol?: string;
  timeframe?: string;
  planRef: React.MutableRefObject<string>;
  codeRef: React.MutableRefObject<string>;
  prevCodeRef: React.MutableRefObject<string>;
  metricsRef: React.MutableRefObject<BacktestMetricsMsg | null>;
  firstUserMsgRef: React.MutableRefObject<string>;
  bumpCodeGen: () => void;
  addMsg: (role: 'user' | 'ai', extra: Partial<ChatMsg>) => void;
  setMetrics: React.Dispatch<React.SetStateAction<BacktestMetricsMsg | null>>;
  setBusy: React.Dispatch<React.SetStateAction<boolean>>;
  tryAutoName: (convId: string, firstMsg: string) => Promise<void>;
  t: (key: string) => string;
}

export function runConversate({
  msg, curCode, isFirst, sessionId, symbol, timeframe,
  planRef, codeRef, prevCodeRef, metricsRef, firstUserMsgRef,
  bumpCodeGen, addMsg, setMetrics, setBusy, tryAutoName, t,
}: RunConversateArgs): () => void {
  const curPlan = planRef.current;
  const curMetrics = metricsRef.current;
  return conversate(
    { message: msg, conversationId: sessionId, symbol, timeframe, plan: curPlan, currentCode: curCode, backtestMetricsJson: curMetrics ? JSON.stringify(curMetrics) : '' },
    {
      onDelta: () => {},
      onPlan: (p) => { planRef.current = p; },
      onCode: (c) => { codeRef.current = c; prevCodeRef.current = curCode; bumpCodeGen(); },
      onPreviousCode: (c) => { prevCodeRef.current = c; },
      onToolCall: () => {},
      onToolResult: (tr: ToolResult) => {
        const icon = tr.success ? '✅' : '❌';
        const label = tr.name === 'compliance_check' ? t(COMPLIANCE_CHECK_KEY) : tr.name === 'backtest' ? t(BACKTEST_LABEL_KEY) : tr.name;
        const detail = tr.error || (tr.success ? t(PASSED_KEY) : '');
        addMsg('ai', { text: `${icon} ${label}: ${detail}` });
        if (tr.name === 'backtest' && tr.outputJson) try {
          const out = JSON.parse(tr.outputJson);
          if (out.run_id) strategyRuntimeApi.watchBacktestRun(out.run_id, (u: BacktestRunUpdate) => {
            if (u.run && isSucceededRun(u.run) && u.metrics) setMetrics(create(BacktestMetricsMsgSchema, { totalReturn: u.metrics.totalReturn, sharpeRatio: u.metrics.sharpeRatio, maxDrawdown: u.metrics.maxDrawdown, winRate: u.metrics.winRate, totalTrades: u.metrics.totalTrades, profitFactor: u.metrics.profitFactor }));
          });
        } catch {}
      },
      onError: (e) => { addMsg('ai', { text: '❌ ' + e }); setBusy(false); },
      onDone: () => {
        const p = planRef.current, c = codeRef.current, pc = prevCodeRef.current;
        const chatMsg: Partial<ChatMsg> = { role: 'ai' };
        let hasContent = false;
        if (p) { chatMsg.plan = p; hasContent = true; }
        if (c) { chatMsg.code = c; chatMsg.prevCode = pc; hasContent = true; }
        if (!hasContent) chatMsg.text = t(CONTINUE_MESSAGE_KEY);
        addMsg('ai', chatMsg);
        setBusy(false);
        if (isFirst && sessionId) {
          const firstMsg = firstUserMsgRef.current || msg;
          tryAutoName(sessionId, firstMsg);
        }
      },
    } satisfies ConversateCallbacks,
  );
}
