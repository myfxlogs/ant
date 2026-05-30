import { Alert, Button, Empty, Spin, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import { CodeExplainPanel } from '@/components/strategy/CodeAssist';
import { formatElapsed } from '../helpers';
import type { Violation } from '../flow/codeValidator';

const { Text, Paragraph } = Typography;

export function CodeDisplay(props: {
  codeText: string;
  codePython: string;
  loading: boolean;
  elapsedSeconds: number;
  hasCode: boolean;
  isValid: boolean;
  violations: Violation[];
  sending: boolean;
  onRetryCodeGen?: () => Promise<void>;
  onAutoRewrite: () => void;
  violationLabel: (v: Violation) => string;
}) {
  const { t } = useTranslation();
  const {
    codeText, codePython, loading, elapsedSeconds, hasCode, isValid,
    violations, sending, onRetryCodeGen, onAutoRewrite, violationLabel,
  } = props;

  const codeToUse = codePython || codeText;
  const codeSpinTip =
    loading && codeToUse
      ? t('ai.debate.v2.codeRegenerating', { defaultValue: 'Rewriting code from your feedback…' })
      : t('ai.debate.v2.codeGenerating', { defaultValue: 'Generating code…' });

  return (
    <>
      {loading ? (
        <>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message={t('ai.debate.v2.modelWaitBanner', {
              defaultValue: 'Model is working… Elapsed {{time}} (duration depends on model and network)',
              time: formatElapsed(elapsedSeconds),
            })}
          />
          {codeToUse ? (
            <pre style={{ background: '#0f172a', color: '#e2e8f0', padding: 16, borderRadius: 8, maxHeight: 360, overflow: 'auto', fontSize: 13, lineHeight: 1.5, marginBottom: 16, whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
              <code>{codeToUse}</code>
            </pre>
          ) : null}
          <div style={{ textAlign: 'center', padding: codeToUse ? 8 : 32 }}>
            <Spin size="large" tip={codeSpinTip} />
            <div style={{ marginTop: 12 }}>
              <Text type="secondary" style={{ fontSize: 12 }}>
                {t('ai.debate.v2.modelWaitBubble', { defaultValue: 'Elapsed {{time}}', time: formatElapsed(elapsedSeconds) })}
              </Text>
            </div>
          </div>
        </>
      ) : codePython ? (
        <pre style={{ background: '#0f172a', color: '#e2e8f0', padding: 16, borderRadius: 8, maxHeight: 520, overflow: 'auto', fontSize: 13, lineHeight: 1.5 }}>
          <code>{codePython}</code>
        </pre>
      ) : codeText ? (
        <Paragraph>
          <pre style={{ whiteSpace: 'pre-wrap' }}>{codeText}</pre>
        </Paragraph>
      ) : (
        <div>
          {onRetryCodeGen ? (
            <Alert
              type="warning"
              showIcon
              style={{ marginBottom: 12 }}
              message={t('ai.debate.v2.codeMissingHint', {
                defaultValue: 'If the gateway timed out (e.g. HTTP 524), the session may already be on this step without code. Use the button below to go back one step and trigger code generation again.',
              })}
              action={
                <Button type="primary" loading={sending} onClick={() => void onRetryCodeGen()}>
                  {t('ai.debate.v2.retryCodeGen', { defaultValue: 'Try generating code again' })}
                </Button>
              }
            />
          ) : null}
          <Empty description={t('ai.debate.v2.codeEmpty', { defaultValue: 'Code not generated yet.' })} />
        </div>
      )}

      {hasCode && isValid && (
        <div style={{ marginTop: 12 }}>
          <CodeExplainPanel code={codeToUse} />
        </div>
      )}

      {hasCode && (
        <div style={{ marginTop: 12 }}>
          {isValid ? (
            <Alert
              type="success"
              showIcon
              message={t('ai.debate.v2.validation.passTitle', { defaultValue: 'Code validation passed. You can save it as a template.' })}
              description={t('ai.debate.v2.validation.passDesc', { defaultValue: '' }) || undefined}
            />
          ) : (
            <Alert
              type="error"
              showIcon
              message={t('ai.debate.v2.validation.failTitle', { defaultValue: 'Sandbox validation failed' })}
              description={(
                <div>
                  <div style={{ marginBottom: 6 }}>
                    {t('ai.debate.v2.validation.failDesc', { defaultValue: 'The following issues block saving. You can ask the code agent to regenerate.' })}
                  </div>
                  <ul style={{ margin: 0, paddingLeft: 20 }}>
                    {violations.map((v, i) => (
                      <li key={`${v.code}-${i}`}>{violationLabel(v)}</li>
                    ))}
                  </ul>
                  <div style={{ marginTop: 8 }}>
                    <Button type="primary" danger size="small" loading={sending} onClick={onAutoRewrite}>
                      {t('ai.debate.v2.validation.rewriteBtn', { defaultValue: 'Send violations back & regenerate' })}
                    </Button>
                  </div>
                </div>
              )}
            />
          )}
        </div>
      )}
    </>
  );
}
