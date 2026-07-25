import { useEffect, useRef, useState } from 'react';
import { Alert, Button, Statistic, Row, Col, Progress } from 'antd';
import { LoadingOutlined, CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import { CHAT_BOX_THINKING_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';
import { BACKTEST_ERROR_KEY, FAILED_KEY, DONE_KEY, NO_MARKET_DATA_KEY, NO_MARKET_DATA_HINT_KEY, PLAN_KEY, COMPILE_ERROR_KEY, FINAL_CODE_KEY, COPY_KEY, APPLY_TO_EDITOR_KEY, COVERAGE_KEY, PROFILE_KEY, ANALYSIS_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import PlanCard from './PlanCard';
import CollapsibleBlock from './CollapsibleBlock';
import { StreamContent, phaseLabels, isNoMarketData } from './chatUtils';

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
  const { t } = useTranslation();
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
        const isUser = turn.role === 'user';
        const isBusy = turn.phase && turn.phase !== 'idle' && turn.phase !== 'done';
        const noData = (turn.error && isNoMarketData(turn.error)) || (turn.backtestError && isNoMarketData(turn.backtestError));

        if (isUser) {
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

        return (
          <div key={turn.id} style={{ margin: '16px 0', display: 'flex', gap: 10 }}>
            <div style={{
              width: 24, height: 24, borderRadius: 6, flexShrink: 0,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'linear-gradient(135deg, #6366f1, #8b5cf6)', color: '#fff', fontSize: 13,
              marginTop: 2,
            }}>
              ⚡
            </div>

            <div style={{ flex: 1, minWidth: 0 }}>
              {isBusy && turn.phase && (
                <div style={{
                  display: 'inline-flex', alignItems: 'center', gap: 6,
                  padding: '3px 10px', borderRadius: 6, marginBottom: 6,
                  background: 'var(--ant-color-fill-tertiary)', fontSize: 11,
                  color: 'var(--ant-color-text-secondary)',
                }}>
                  <LoadingOutlined style={{ fontSize: 11 }} />
                  {t(`strategy.gen.${turn.phase}`, phaseLabels[turn.phase] || turn.phase)}
                  {turn.attempts && turn.attempts > 1 && ` · ${turn.attempts}/3`}
                  {turn.coverageScore && turn.coverageScore > 0 && ` · ${(turn.coverageScore * 100).toFixed(0)}%`}
                </div>
              )}

              {turn.phase === 'done' && (
                <div style={{ marginBottom: 6, fontSize: 11, color: 'var(--ant-color-text-tertiary)' }}>
                  {(turn.error || turn.compileError || turn.backtestError) && !turn.metrics ? (
                    <span style={{ color: 'var(--ant-color-error)' }}>✕ {t(FAILED_KEY)}</span>
                  ) : (
                    <span style={{ color: 'var(--ant-color-success)' }}>✓ {t(DONE_KEY)}</span>
                  )}
                  {turn.timestamp && ` · ${turn.timestamp}`}
                </div>
              )}

              {noData && (
                <Alert
                  type="warning" showIcon
                  message={t(NO_MARKET_DATA_KEY)}
                  description={t(NO_MARKET_DATA_HINT_KEY)}
                  style={{ marginBottom: 8, fontSize: 12 }}
                />
              )}

              {turn.plan && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📋</span>}
                  title={t(PLAN_KEY)}
                  subtitle={turn.plan.type || turn.plan.entry}
                  defaultOpen={activePlanId === turn.id}
                >
                  <PlanCard
                    plan={turn.plan}
                    onConfirm={onPlanConfirm || (() => {})}
                    onRefine={onPlanRefine || (() => {})}
                    refining={activePlanId === turn.id && planRefining}
                  />
                </CollapsibleBlock>
              )}

              {turn.compileError && !noData && (
                <Alert type="error" showIcon style={{ marginBottom: 8, fontSize: 12 }}
                  message={t(COMPILE_ERROR_KEY)}
                  description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{turn.compileError}</pre>}
                />
              )}
              {turn.backtestError && !noData && (
                <Alert type="error" showIcon style={{ marginBottom: 8, fontSize: 12 }}
                  message={t(BACKTEST_ERROR_KEY)}
                  description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{turn.backtestError}</pre>}
                />
              )}

              {turn.reasoning && turn.reasoning.length > 0 && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>💭</span>}
                  title={t(CHAT_BOX_THINKING_KEY, 'Thinking Process')}
                  defaultOpen={false}
                >
                  <div style={{
                    fontSize: 12, lineHeight: '18px', whiteSpace: 'pre-wrap',
                    color: 'var(--ant-color-text-tertiary)',
                    maxHeight: 300, overflowY: 'auto',
                    padding: '4px 0',
                  }}>
                    {turn.reasoning}
                  </div>
                </CollapsibleBlock>
              )}

              {turn.streamText && !turn.plan && (
                <div style={{ marginBottom: 8 }}>
                  <StreamContent text={turn.streamText} />
                </div>
              )}

              {/* I1: generatedCode is the single deliverable — rendered as THE Apply card */}
              {turn.generatedCode && (
                <div style={{ marginBottom: 8, background: 'var(--ant-color-bg-container)', borderRadius: 8, padding: 12, border: '1px solid var(--ant-color-border)' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, gap: 8 }}>
                    <strong>{t(FINAL_CODE_KEY)}</strong>
                    <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
                      <Button size="small" icon={<CopyOutlined />}
                        onClick={() => handleCopy(turn.id, turn.generatedCode!)}>
                        {copiedId === turn.id ? '✓' : t(COPY_KEY)}
                      </Button>
                      <Button size="small" type="primary"
                        disabled={turn.phase !== 'done' || !!(turn.compileError || turn.error)}
                        onClick={() => onApplyCode?.(turn.generatedCode!)}>
                        {t(APPLY_TO_EDITOR_KEY)}
                      </Button>
                    </div>
                  </div>
                  <pre style={{ fontSize: 12, maxHeight: 200, overflow: 'auto', background: '#f5f5f5', padding: 8, borderRadius: 4, margin: 0 }}>{turn.generatedCode}</pre>
                </div>
              )}

              {turn.metrics && turn.metrics.length > 0 && (
                <Row gutter={8} style={{ marginBottom: 8 }}>
                  {turn.metrics.map((m, i) => (
                    <Col key={i} span={6}>
                      <Statistic
                        title={m.label}
                        value={m.value}
                        valueStyle={{
                          fontSize: 14,
                          color: m.positive === true ? '#3fb950' : m.positive === false ? '#f85149' : undefined,
                        }}
                      />
                    </Col>
                  ))}
                </Row>
              )}

              {turn.coverageScore && turn.coverageScore > 0 && (
                <Progress
                  percent={turn.coverageScore * 100}
                  size="small"
                  format={(p) => `${t(COVERAGE_KEY)}: ${(p || 0).toFixed(0)}%`}
                  style={{ marginBottom: 8, maxWidth: 200 }}
                />
              )}

              {turn.profile && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📊</span>}
                  title={turn.profile.strategyType || t(PROFILE_KEY)}
                  subtitle={turn.profile.description}
                >
                  {turn.profile.entryLogic && <div><strong>{t('strategy.chat.entry', { defaultValue: 'Entry:' })}</strong> {turn.profile.entryLogic}</div>}
                  {turn.profile.exitLogic && <div><strong>{t('strategy.chat.exit', { defaultValue: 'Exit:' })}</strong> {turn.profile.exitLogic}</div>}
                  {turn.profile.riskManagement && <div><strong>{t('strategy.chat.risk', { defaultValue: 'Risk:' })}</strong> {turn.profile.riskManagement}</div>}
                  {turn.profile.indicatorsUsed && turn.profile.indicatorsUsed.length > 0 && (
                    <div><strong>{t('strategy.chat.indicators', { defaultValue: 'Indicators:' })}</strong> {turn.profile.indicatorsUsed.join(', ')}</div>
                  )}
                </CollapsibleBlock>
              )}

              {turn.analysis?.summary && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📝</span>}
                  title={t(ANALYSIS_KEY)}
                  subtitle={turn.analysis.summary}
                />
              )}

              {turn.error && !noData && (
                <Alert type="warning" showIcon style={{ marginBottom: 8, fontSize: 12 }}
                  message={turn.error}
                />
              )}
            </div>
          </div>
        );
      })}
      <div ref={endRef} />
    </div>
  );
}
