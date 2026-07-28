import { Alert } from 'antd';
import { LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ChatTurn } from './ChatHistory';
import { CHAT_BOX_THINKING_KEY } from '@/gen/ant/v1/i18n/ai_core_keys';
import { FAILED_KEY, DONE_KEY, NO_MARKET_DATA_KEY, NO_MARKET_DATA_HINT_KEY, PLAN_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import PlanCard from './PlanCard';
import CollapsibleBlock from './CollapsibleBlock';
import { StreamContent, phaseLabels } from './chatUtils';
import { ErrorAlerts, GeneratedCodeCard, MetricsAndProfile } from './ChatAITurnSections';

interface Props {
  turn: ChatTurn;
  isBusy: boolean;
  noData: boolean;
  copiedId: string | null;
  onCopy: (id: string, code: string) => void;
  onPlanConfirm?: () => void;
  onPlanRefine?: (feedback: string) => void;
  planRefining?: boolean;
  activePlanId?: string;
  onApplyCode?: (code: string) => void;
}

export default function ChatAITurn({ turn, isBusy, noData, copiedId, onCopy, onPlanConfirm, onPlanRefine, planRefining, activePlanId, onApplyCode }: Props) {
  const { t } = useTranslation();

  return (
    <div style={{ margin: '16px 0', display: 'flex', gap: 10 }}>
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
          <Alert type="warning" showIcon
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

        <ErrorAlerts turn={turn} noData={noData} />

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

        <GeneratedCodeCard turn={turn} copiedId={copiedId} onCopy={onCopy} onApplyCode={onApplyCode} />

        <MetricsAndProfile turn={turn} />
      </div>
    </div>
  );
}
