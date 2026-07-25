import { useState } from 'react';
import { Form, Tag, Segmented, Input } from 'antd';
import { CodeOutlined, ImportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import ImportEAPanel from './ImportEAPanel';
import type { FormInstance } from 'antd';

const { TextArea } = Input;

interface Props {
  form: FormInstance;
  code: string;
  onStrategyIdChange?: (id: string | undefined) => void;
}

export default function CodeEditorPanel({ form, code, onStrategyIdChange }: Props) {
  const { t } = useTranslation();
  const [mode, setMode] = useState<string>('write');

  const handleApplyCode = (newCode: string) => {
    form.setFieldsValue({ code: newCode });
    setMode('write');
  };

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
        <ImportEAPanel onApplyCode={handleApplyCode} onStrategyIdChange={onStrategyIdChange} />
      )}
    </div>
  );
}
