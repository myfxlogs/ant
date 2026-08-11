import { Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import type { MarketplaceQualityPreview } from '@/gen/ant/v1/backtest_run_query_pb';
import type { GateEvaluationUpdate, GateResult } from '@/gen/ant/v1/ai_gate_pb';

export function GatePreview({ gateUpdate, gateResults, qualityPreview }: {
  gateUpdate?: GateEvaluationUpdate | null;
  gateResults?: GateResult[];
  qualityPreview?: MarketplaceQualityPreview | null;
}) {
  const { t } = useTranslation();
  return (
    <div style={{ marginBottom: 12, padding: '8px 12px', border: '1px solid #e6fffb', borderRadius: 8, background: '#f6ffed' }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: '#52c41a', marginBottom: 6 }}>
        {t('strategy.backtest.autoGate', { defaultValue: 'Auto Gate Evaluation' })}
      </div>
      {gateResults && gateResults.length > 0 && (
        <div style={{ marginBottom: 6, display: 'flex', flexWrap: 'wrap', gap: 4 }}>
          {gateResults.map((g, i) => (
            <Tag key={i} color={g.skipped ? 'default' : g.passed ? 'success' : 'error'} style={{ fontSize: 10 }}>
              {g.gate}: {g.skipped ? 'SKIP' : g.passed ? 'PASS' : 'FAIL'}
            </Tag>
          ))}
        </div>
      )}
      {gateUpdate?.completed && (
        <div style={{ marginBottom: 4, fontSize: 12 }}>
          <Tag color={gateUpdate.completed.passed ? 'success' : 'error'}>
            {gateUpdate.completed.passed ? 'PASS' : 'FAIL'}
          </Tag>
          <span style={{ color: '#595959' }}>{gateUpdate.completed.summary}</span>
        </div>
      )}
      {qualityPreview && (
        <div style={{ fontSize: 12 }}>
          <Tag color={qualityPreview.publishable ? 'success' : 'warning'}>
            {qualityPreview.publishable
              ? t('strategy.backtest.publishable', { defaultValue: 'Publishable' })
              : t('strategy.backtest.notPublishable', { defaultValue: 'Not Publishable' })}
          </Tag>
          {qualityPreview.violations && qualityPreview.violations.length > 0 && (
            <div style={{ marginTop: 4 }}>
              {qualityPreview.violations.map((v, i) => (
                <div key={i} style={{ color: '#8c8c8c', fontSize: 11 }}>{v.metric}: {v.actual} / {v.threshold}</div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
