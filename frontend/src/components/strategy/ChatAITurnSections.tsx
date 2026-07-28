import { Alert, Button, Statistic, Row, Col, Progress } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { ChatTurn } from './ChatHistory';
import { BACKTEST_ERROR_KEY, COMPILE_ERROR_KEY, FINAL_CODE_KEY, COPY_KEY, APPLY_TO_EDITOR_KEY, COVERAGE_KEY, PROFILE_KEY, ANALYSIS_KEY } from '@/gen/ant/v1/i18n/strategy_gen_keys';
import CollapsibleBlock from './CollapsibleBlock';

export function ErrorAlerts({ turn, noData }: { turn: ChatTurn; noData: boolean }) {
  const { t } = useTranslation();
  return (
    <>
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
      {turn.error && !noData && (
        <Alert type="warning" showIcon style={{ marginBottom: 8, fontSize: 12 }}
          message={turn.error}
        />
      )}
    </>
  );
}

export function GeneratedCodeCard({ turn, copiedId, onCopy, onApplyCode }: {
  turn: ChatTurn; copiedId: string | null; onCopy: (id: string, code: string) => void; onApplyCode?: (code: string) => void;
}) {
  const { t } = useTranslation();
  if (!turn.generatedCode) return null;
  return (
    <div style={{ marginBottom: 8, background: 'var(--ant-color-bg-container)', borderRadius: 8, padding: 12, border: '1px solid var(--ant-color-border)' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, gap: 8 }}>
        <strong>{t(FINAL_CODE_KEY)}</strong>
        <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
          <Button size="small" icon={<CopyOutlined />}
            onClick={() => onCopy(turn.id, turn.generatedCode!)}>
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
  );
}

export function MetricsAndProfile({ turn }: { turn: ChatTurn }) {
  const { t } = useTranslation();
  return (
    <>
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
    </>
  );
}
