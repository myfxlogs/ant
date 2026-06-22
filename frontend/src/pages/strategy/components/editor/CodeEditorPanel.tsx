import { useState, useCallback } from 'react';
import { Button, Form, Tag, Segmented, Typography, Spin, message } from 'antd';
import { CodeOutlined, ImportOutlined, RobotOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { codeAssistClient } from '@/client/connect';
import type { FormInstance } from 'antd';
import { Input } from 'antd';

const { TextArea } = Input;
const { Text } = Typography;

interface Props {
  form: FormInstance;
  code: string;
}

export default function CodeEditorPanel({ form, code }: Props) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<string>('write');
  const [eaCode, setEaCode] = useState('');
  const [eaTranslating, setEaTranslating] = useState(false);
  const [eaResult, setEaResult] = useState('');

  const handleImportEA = useCallback(() => {
    if (!eaCode.trim() || eaCode.trim().length < 20) {
      message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
      return;
    }
    setEaTranslating(true); setEaResult('');
    // IIFE isolates the promise so React doesn't trace rejections to the event handler.
    (async () => {
      try {
        const resp = await codeAssistClient.transformCode({ sourceCode: eaCode, sourceLang: 'auto', targetLang: 'python' });
        setEaResult(resp.targetCode || '');
      } catch (err: any) {
        const msg = String(err?.rawMessage || err?.message || '');
        if (msg.includes('insufficient') || msg.includes('balance') || msg.includes('InsufficientBalance')) {
          message.error(t('strategy.importEA.insufficientBalance', { defaultValue: 'AI balance insufficient. Please top up in AI Gateway settings.' }));
        } else {
          message.error(msg || t('strategy.importEA.translateFailed', { defaultValue: 'Translation failed. Please try again.' }));
        }
      }
      finally { setEaTranslating(false); }
    })().catch(() => {});
  }, [eaCode, t]);

  const applyEaResult = useCallback(() => {
    if (eaResult) { form.setFieldsValue({ code: eaResult }); setMode('write'); }
  }, [eaResult, form]);

  return (
    <div style={{ border: '1px solid var(--color-border)', borderRadius: 10, overflow: 'hidden', display: 'flex', flexDirection: 'column' }}>
      <div style={{ padding: '4px 14px', background: 'var(--color-bg-tertiary)', borderBottom: '1px solid var(--color-border)', display: 'flex', alignItems: 'center', gap: 6 }}>
        <Segmented size="small" value={mode} onChange={v => setMode(v as string)}
          options={[
            { value: 'write', icon: <CodeOutlined />, label: t('strategy.importEA.writeTab', { defaultValue: 'Strategy Code' }) },
            { value: 'import', icon: <ImportOutlined />, label: t('strategy.importEA.importTab', { defaultValue: 'Import EA' }) },
          ]} />
        {mode === 'write' && code.trim() && <Tag style={{ marginLeft: 'auto' }}>{code.split('\n').length} lines</Tag>}
      </div>

      {mode === 'write' ? (
        <Form.Item name="code" rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY) }]} style={{ marginBottom: 0 }}>
          <TextArea rows={18} placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY)}
            style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', height: 420 }} />
        </Form.Item>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', height: 420 }}>
          <TextArea value={eaCode} onChange={e => setEaCode(e.target.value)} rows={10}
            placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA code...' })}
            style={{ fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace', fontSize: 13, lineHeight: 1.6, border: 'none', borderRadius: 0, resize: 'none', flex: 'none' }} />
          <div style={{ padding: '6px 14px', borderTop: '1px solid var(--color-border)', borderBottom: '1px solid var(--color-border)', display: 'flex', gap: 8, alignItems: 'center' }}>
            <Button type="primary" size="small" icon={<RobotOutlined />} onClick={handleImportEA} loading={eaTranslating}>
              {t('strategy.importEA.translate', { defaultValue: 'Translate to Python' })}</Button>
            {eaResult && <Button size="small" onClick={applyEaResult}>{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}</Button>}
          </div>
          {eaResult ? (
            <pre style={{ flex: 1, overflow: 'auto', margin: 0, padding: '10px 14px', fontFamily: '"Fira Code", monospace', fontSize: 12, lineHeight: 1.5, background: '#f9fafb', color: 'var(--color-text)', whiteSpace: 'pre-wrap' }}>{eaResult}</pre>
          ) : (
            <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
              {eaTranslating ? <Spin tip={t('strategy.importEA.translating', { defaultValue: 'AI translating...' })} />
                : <Text type="secondary">{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click Translate' })}</Text>}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
