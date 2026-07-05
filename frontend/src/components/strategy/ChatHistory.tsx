import { useEffect, useRef, useState, memo, useCallback } from 'react';
import { Alert, Statistic, Row, Col, Progress, Button } from 'antd';
import { LoadingOutlined, CopyOutlined, CodeOutlined, RightOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { Light as SyntaxHighlighter } from 'react-syntax-highlighter';
import python from 'react-syntax-highlighter/dist/esm/languages/hljs/python';
import { atomOneDark } from 'react-syntax-highlighter/dist/esm/styles/hljs';
import type { StrategyPlan } from '@/gen/ant/v1/agent_gateway_pb';
import type { StrategyProfile } from '@/gen/ant/v1/agent_profile_pb';
import type { BacktestAnalysis } from '@/gen/ant/v1/agent_analysis_pb';
import PlanCard from './PlanCard';

SyntaxHighlighter.registerLanguage('python', python);

interface CodeSegment { type: 'text' | 'code'; content: string; lang?: string }

const CollapsibleBlock = memo(function CollapsibleBlock({
  icon, title, subtitle, children, defaultOpen = false,
}: {
  icon: React.ReactNode;
  title: string;
  subtitle?: string;
  children?: React.ReactNode;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const toggle = useCallback(() => setOpen((o) => !o), []);
  return (
    <div style={{ marginBottom: 6 }}>
      <div
        onClick={toggle}
        style={{
          display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer',
          padding: '4px 10px', borderRadius: 6,
          background: 'var(--ant-color-fill-tertiary)',
          fontSize: 11, color: 'var(--ant-color-text-secondary)',
          userSelect: 'none', transition: 'background 0.15s',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--ant-color-fill-quaternary)')}
        onMouseLeave={(e) => (e.currentTarget.style.background = 'var(--ant-color-fill-tertiary)')}
      >
        <RightOutlined style={{ fontSize: 9, transform: open ? 'rotate(90deg)' : 'none', transition: 'transform 0.15s' }} />
        {icon}
        <span style={{ fontWeight: 500 }}>{title}</span>
        {subtitle && <span style={{ color: 'var(--ant-color-text-tertiary)', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}> — {subtitle}</span>}
      </div>
      {open && children && (
        <div style={{ marginTop: 4, padding: '8px 12px', fontSize: 12, lineHeight: '18px' }}>
          {children}
        </div>
      )}
    </div>
  );
});

function parseCodeBlocks(text: string): CodeSegment[] {
  const segments: CodeSegment[] = [];
  const re = /```(\w+)?\n([\s\S]*?)```/g;
  let lastIdx = 0;
  let match: RegExpExecArray | null;
  while ((match = re.exec(text)) !== null) {
    if (match.index > lastIdx) {
      segments.push({ type: 'text', content: text.slice(lastIdx, match.index).trim() });
    }
    segments.push({ type: 'code', lang: match[1] || 'python', content: match[2] });
    lastIdx = re.lastIndex;
  }
  if (lastIdx < text.length) {
    const tail = text.slice(lastIdx).trim();
    if (tail) segments.push({ type: 'text', content: tail });
  }
  return segments;
}

const CodeBlock = memo(({ code, lang, onApply }: { code: string; lang: string; onApply?: (code: string) => void }) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const copy = () => { navigator.clipboard.writeText(code); setCopied(true); setTimeout(() => setCopied(false), 2000); };

  return (
    <div style={{ marginTop: 8, borderRadius: 6, overflow: 'hidden', border: '1px solid var(--ant-color-border)' }}>
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '4px 10px', background: '#282c34',
      }}>
        <span style={{ fontSize: 11, color: '#abb2bf' }}><CodeOutlined /> {lang}</span>
        <div style={{ display: 'flex', gap: 4 }}>
          <Button size="small" type="text" icon={<CopyOutlined />}
            onClick={copy} style={{ color: '#abb2bf', fontSize: 11 }}>{copied ? '✓' : ''}</Button>
          {onApply && (
            <Button size="small" type="primary" onClick={() => onApply(code)}
              style={{ fontSize: 11 }}>{t('strategy.gen.execApplyCode', 'Apply')}</Button>
          )}
        </div>
      </div>
      <SyntaxHighlighter
        language={lang} style={atomOneDark} showLineNumbers wrapLines
        customStyle={{ margin: 0, fontSize: 12, maxHeight: 300, overflow: 'auto' }}
        lineNumberStyle={{ fontSize: 10, minWidth: '2em', color: '#636d83' }}
      >{code}</SyntaxHighlighter>
    </div>
  );
});

const StreamContent = memo(({ text, onApply }: { text: string; onApply?: (code: string) => void }) => {
  const segments = parseCodeBlocks(text);
  if (segments.length === 1 && segments[0].type === 'text') {
    return <div style={{ fontSize: 13, lineHeight: '20px', whiteSpace: 'pre-wrap' }}>{segments[0].content}</div>;
  }
  return (
    <div>
      {segments.map((seg, i) => seg.type === 'code'
        ? <CodeBlock key={i} code={seg.content} lang={seg.lang || 'python'} onApply={onApply} />
        : <div key={i} style={{ fontSize: 13, lineHeight: '20px', whiteSpace: 'pre-wrap', marginBottom: 4 }}>{seg.content}</div>
      )}
    </div>
  );
});

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
}

interface Props {
  turns: ChatTurn[];
  onPlanConfirm?: () => void;
  onPlanRefine?: (feedback: string) => void;
  planRefining?: boolean;
  activePlanId?: string;
  onApplyCode?: (code: string) => void;
}

const phaseLabels: Record<string, string> = {
  planning: 'Planning...',
  chatting: 'Thinking...',
  generating: 'Generating...',
  compiling: 'Compiling...',
  backtesting: 'Backtesting...',
  analyzing: 'Analyzing...',
};

function isNoMarketData(s: string) {
  return /insufficient market data|0 bars|need.*≥.*2/i.test(s);
}

export default function ChatHistory({ turns, onPlanConfirm, onPlanRefine, planRefining, activePlanId, onApplyCode }: Props) {
  const { t } = useTranslation();
  const endRef = useRef<HTMLDivElement>(null);

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
            {/* AI avatar */}
            <div style={{
              width: 24, height: 24, borderRadius: 6, flexShrink: 0,
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              background: 'linear-gradient(135deg, #6366f1, #8b5cf6)', color: '#fff', fontSize: 13,
              marginTop: 2,
            }}>
              ⚡
            </div>

            {/* AI content — full width, no bubble */}
            <div style={{ flex: 1, minWidth: 0 }}>
              {/* Phase indicator — like Windsurf tool call */}
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

              {/* Done indicator */}
              {turn.phase === 'done' && (
                <div style={{ marginBottom: 6, fontSize: 11, color: 'var(--ant-color-text-tertiary)' }}>
                  {(turn.compileError || turn.backtestError) && !turn.metrics ? (
                    <span style={{ color: 'var(--ant-color-error)' }}>✕ {t('strategy.gen.failed', 'Failed')}</span>
                  ) : (
                    <span style={{ color: 'var(--ant-color-success)' }}>✓ {t('strategy.gen.done', 'Done')}</span>
                  )}
                  {turn.timestamp && ` · ${turn.timestamp}`}
                </div>
              )}

              {/* No market data */}
              {noData && (
                <Alert
                  type="warning" showIcon
                  message={t('strategy.gen.noMarketData', 'No market data available')}
                  description={t('strategy.gen.noMarketDataHint', 'Please select a trading account and a symbol so the chart can load market data, then try again.')}
                  style={{ marginBottom: 8, fontSize: 12 }}
                />
              )}

              {/* Plan card — collapsible */}
              {turn.plan && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📋</span>}
                  title={t('strategy.gen.plan', 'Strategy Plan')}
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

              {/* Errors */}
              {turn.compileError && !noData && (
                <Alert type="error" showIcon style={{ marginBottom: 8, fontSize: 12 }}
                  message={t('strategy.gen.compileError', 'Compile Error')}
                  description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{turn.compileError}</pre>}
                />
              )}
              {turn.backtestError && !noData && (
                <Alert type="error" showIcon style={{ marginBottom: 8, fontSize: 12 }}
                  message={t('strategy.gen.backtestError', 'Backtest Error')}
                  description={<pre style={{ fontSize: 11, whiteSpace: 'pre-wrap', margin: 0 }}>{turn.backtestError}</pre>}
                />
              )}

              {/* Stream text — natural flow with code blocks (hidden when plan exists to avoid duplicate) */}
              {turn.streamText && !turn.plan && (
                <div style={{ marginBottom: 8 }}>
                  <StreamContent text={turn.streamText} onApply={onApplyCode} />
                </div>
              )}

              {/* Metrics */}
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

              {/* Coverage */}
              {turn.coverageScore && turn.coverageScore > 0 && (
                <Progress
                  percent={turn.coverageScore * 100}
                  size="small"
                  format={(p) => `${t('strategy.gen.coverage', 'Coverage')}: ${(p || 0).toFixed(0)}%`}
                  style={{ marginBottom: 8, maxWidth: 200 }}
                />
              )}

              {/* Profile — collapsible */}
              {turn.profile && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📊</span>}
                  title={turn.profile.strategyType || t('strategy.gen.profile', 'Strategy Profile')}
                  subtitle={turn.profile.description}
                >
                  {turn.profile.entryLogic && <div><strong>Entry:</strong> {turn.profile.entryLogic}</div>}
                  {turn.profile.exitLogic && <div><strong>Exit:</strong> {turn.profile.exitLogic}</div>}
                  {turn.profile.riskManagement && <div><strong>Risk:</strong> {turn.profile.riskManagement}</div>}
                  {turn.profile.indicatorsUsed && turn.profile.indicatorsUsed.length > 0 && (
                    <div><strong>Indicators:</strong> {turn.profile.indicatorsUsed.join(', ')}</div>
                  )}
                </CollapsibleBlock>
              )}

              {/* Analysis — collapsible */}
              {turn.analysis?.summary && (
                <CollapsibleBlock
                  icon={<span style={{ fontSize: 12 }}>📝</span>}
                  title={t('strategy.gen.analysis', 'Backtest Analysis')}
                  subtitle={turn.analysis.summary}
                />
              )}

              {/* Generic error */}
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
