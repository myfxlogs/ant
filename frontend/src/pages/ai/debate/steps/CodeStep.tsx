import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button, Form, Modal, Space, Typography, message as antMessage } from 'antd';
import { useTranslation } from 'react-i18next';
import { strategyApi } from '@/client/strategy';
import { validatePythonSandbox, violationsToFeedback, type Violation } from '../flow/codeValidator';
import { CodeDisplay } from './CodeDisplay';
import { RejectCodeModal, SaveTemplateModal } from './CodeStepModals';

const { Text } = Typography;

export function CodeStep(props: {
  code: { text: string; python: string; loading: boolean; elapsedSeconds: number };
  onBack: () => void;
  onReject: (feedback: string) => Promise<void>;
  sending: boolean;
  onRetryCodeGen?: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { code, onBack, onReject, sending, onRetryCodeGen } = props;

  const [rejectOpen, setRejectOpen] = useState(false);
  const [rejectText, setRejectText] = useState('');
  const [saveOpen, setSaveOpen] = useState(false);
  const [saveForm] = Form.useForm<{ name: string; description: string }>();
  const [saving, setSaving] = useState(false);

  const codeToUse = code.python || code.text;
  const hasCode = !!codeToUse && !code.loading;
  const canAct = hasCode && !sending;

  const violations: Violation[] = useMemo(
    () => (hasCode ? validatePythonSandbox(codeToUse) : []),
    [hasCode, codeToUse],
  );

  const isValid = hasCode && violations.length === 0;
  const canSave = canAct && isValid;

  function violationLabel(v: Violation): string {
    const i18nKey = `ai.debate.v2.validation.codes.${v.code}`;
    const translated = t(i18nKey, { defaultValue: '' });
    const base = translated || v.message;
    return v.hit ? `${base} (${v.hit})` : base;
  }

  function handleAutoRewrite() {
    if (violations.length === 0) return;
    Modal.confirm({
      title: t('ai.debate.v2.validation.rewriteConfirmTitle', { defaultValue: 'Regenerate the code?' }),
      content: t('ai.debate.v2.validation.rewriteConfirmContent', {
        defaultValue: 'The code did not pass the sandbox validator. We will send the violations back to the code agent and regenerate. Continue?',
      }),
      okText: t('ai.debate.v2.validation.rewriteOk', { defaultValue: 'Regenerate' }),
      cancelText: t('ai.debate.v2.validation.rewriteCancel', { defaultValue: 'Cancel' }),
      onOk: async () => { await onReject(violationsToFeedback(violations)); },
    });
  }

  async function handleReject() {
    const fb = rejectText.trim();
    if (!fb) {
      antMessage.warning(t('ai.debate.v2.rejectFeedbackRequired', {
        defaultValue: 'Please describe what to improve so the model can rewrite the code.',
      }));
      return;
    }
    setRejectOpen(false);
    setRejectText('');
    await onReject(fb);
  }

  async function handleSave() {
    let values: { name: string; description: string };
    try { values = await saveForm.validateFields(); } catch { return; }
    setSaving(true);
    try {
      const resp = await strategyApi.createTemplate({
        name: values.name.trim(), description: (values.description || '').trim(),
        code: codeToUse, parameters: [], isPublic: false, tags: ['ai-debate'],
      });
      antMessage.success(t('ai.debate.v2.saveSuccess', { defaultValue: 'Saved as a private template' }));
      setSaveOpen(false);
      saveForm.resetFields();
      Modal.confirm({
        title: t('ai.debate.v2.saveGotoConfirmTitle', { defaultValue: 'Go to Strategy Templates?' }),
        content: t('ai.debate.v2.saveGotoConfirmContent', {
          defaultValue: 'The template has been saved. Want to open it in Strategy Templates to run a backtest?',
        }),
        okText: t('ai.debate.v2.saveGotoOk', { defaultValue: 'Open templates' }),
        cancelText: t('ai.debate.v2.saveGotoCancel', { defaultValue: 'Stay here' }),
        onOk: () => {
          const id = resp?.id || '';
          navigate(id ? `/strategy/templates?group=user&templateId=${encodeURIComponent(String(id))}` : '/strategy/templates?group=user');
        },
      });
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '';
      antMessage.error(msg || t('ai.debate.v2.saveFailed', { defaultValue: 'Failed to save template' }));
    } finally { setSaving(false); }
  }

  return (
    <div>
      <div style={{ marginBottom: 8 }}>
        <Text strong>{t('ai.debate.v2.codeTitle', { defaultValue: 'Code proposal' })}</Text>
        <Text type="secondary" style={{ marginLeft: 8, fontSize: 12 }}>
          {t('ai.debate.v2.codeHint', { defaultValue: 'Generated from the agreed summaries of all previous steps.' })}
        </Text>
      </div>

      <CodeDisplay
        codeText={code.text}
        codePython={code.python}
        loading={code.loading}
        elapsedSeconds={code.elapsedSeconds}
        hasCode={hasCode}
        isValid={isValid}
        violations={violations}
        sending={sending}
        onRetryCodeGen={onRetryCodeGen}
        onAutoRewrite={handleAutoRewrite}
        violationLabel={violationLabel}
      />

      <div style={{ marginTop: 12, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <Button onClick={onBack}>{t('ai.debate.v2.back', { defaultValue: 'Back' })}</Button>
        <Space>
          <Button danger disabled={!canAct} onClick={() => setRejectOpen(true)}>
            {t('ai.debate.v2.rejectCode', { defaultValue: 'Reject & rewrite' })}
          </Button>
          <Button
            type="primary" disabled={!canSave}
            title={!canSave && hasCode ? t('ai.debate.v2.validation.saveBlocked', {
              defaultValue: 'Save is disabled until the code passes sandbox validation.',
            }) : undefined}
            onClick={() => setSaveOpen(true)}
          >
            {t('ai.debate.v2.saveTemplate', { defaultValue: 'Save as template' })}
          </Button>
        </Space>
      </div>

      <RejectCodeModal open={rejectOpen} sending={sending} rejectText={rejectText}
        onRejectTextChange={setRejectText} onConfirm={handleReject} onCancel={() => setRejectOpen(false)} />
      <SaveTemplateModal open={saveOpen} saving={saving} form={saveForm}
        onConfirm={handleSave} onCancel={() => setSaveOpen(false)} />
    </div>
  );
}
