import React, { useState, useCallback, useEffect } from 'react';
import { Modal, Button, Form, Row, Col, Space, Tag, Typography } from 'antd';
import { CopyOutlined, CodeOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CODE_MODAL_ACTIONS_COPY_KEY, CODE_MODAL_TITLE_KEY, EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY, EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY, EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { SAVE_BLOCKED_NOT_VALIDATED_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';
import type { FormInstance } from 'antd';
import type { StrategyTemplate } from '@/client/strategy';
import { CodeExplainPanel } from '@/components/strategy/CodeAssist';
import { aiPrimaryClient } from '@/client/connect';
import MetadataHeader from './components/editor/MetadataHeader';
import CodeEditorPanel from './components/editor/CodeEditorPanel';
import AIPanel from './components/editor/AIPanel';

const { Text } = Typography;

export type StrategyTemplateEditModalProps = {
  open: boolean;
  editingTemplate: StrategyTemplate | null;
  form: FormInstance;
  codeValidating: boolean;
  validationResult?: { valid?: boolean; errors?: string[]; warnings?: string[] } | null;
  lastValidatedCode?: string;
  onCancel: () => void;
  onValidate: (code: string) => void;
  onClearValidation: () => void;
  onSubmit: (values: Record<string, unknown>) => void;
};

export const StrategyTemplateEditModal: React.FC<StrategyTemplateEditModalProps> = ({
  open, editingTemplate, form, codeValidating, lastValidatedCode = '', validationResult,
  onCancel, onValidate, onClearValidation, onSubmit,
}) => {
  const { t } = useTranslation();
  const watchedCode = Form.useWatch<string | undefined>('code', form) ?? '';
  const code = watchedCode || (form.getFieldValue('code') as string) || '';
  const [aiTab, setAiTab] = useState<string>('revise');
  const [fixInstruction, setFixInstruction] = useState('');
  const [aiModel, setAiModel] = useState('');

  useEffect(() => {
    if (open) { aiPrimaryClient.getAIPrimary({}).then(r => setAiModel(r.model || '')).catch(() => {}); }
  }, [open]);

  const applyAICode = useCallback((newCode: string) => {
    form.setFieldsValue({ code: newCode });
    setAiTab('validate');
  }, [form]);
  const handleFixWithAI = useCallback(() => {
    if (!validationResult) return;
    const parts: string[] = [];
    if (validationResult.errors?.length) parts.push('Fix these validation errors:\n' + validationResult.errors.map((e: string) => `- ${e}`).join('\n'));
    if (validationResult.warnings?.length) parts.push('Also address these warnings:\n' + validationResult.warnings.map((w: string) => `- ${w}`).join('\n'));
    setFixInstruction(parts.join('\n\n'));
    onClearValidation(); // clear old results before sending to AI
    setAiTab('revise');
  }, [validationResult, onClearValidation]);

  return (
    <Modal
      title={
        <Space>
          {editingTemplate ? t(EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY) : t(EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY)}
          {code !== lastValidatedCode && code.trim() && <Tag color="warning" style={{ marginLeft: 8 }}>{t(SAVE_BLOCKED_NOT_VALIDATED_KEY, { defaultValue: 'Not validated' })}</Tag>}
        </Space>
      }
      open={open} onCancel={onCancel}
      footer={[
        <Button key="cancel" onClick={onCancel} disabled={codeValidating}>{t('common.cancel')}</Button>,
        <Button key="validate" onClick={() => { setAiTab('validate'); onValidate(code); }} loading={codeValidating} icon={<ThunderboltOutlined />}>{t(EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY)}</Button>,
        <Button key="save" type="primary" onClick={() => form.submit()} loading={codeValidating} disabled={!code.trim() || code !== lastValidatedCode}>{t('common.save')}</Button>,
      ]}
      width={1280} styles={{ body: { padding: '16px 20px' } }}
    >
      <Form form={form} layout="vertical" onFinish={onSubmit} initialValues={{ isPublic: false }} onValuesChange={undefined}>
        <MetadataHeader aiModel={aiModel} />
        <Row gutter={16}>
          <Col span={15}><CodeEditorPanel form={form} code={code} /></Col>
          <Col span={9}>
            <AIPanel activeTab={aiTab} onTabChange={setAiTab} code={code} codeValidating={codeValidating}
              validationResult={validationResult} fixInstruction={fixInstruction}
              onApplyCode={applyAICode} onFixWithAI={handleFixWithAI} />
          </Col>
        </Row>
      </Form>
    </Modal>
  );
};

export type StrategyTemplateCodeViewModalProps = { open: boolean; code: string; onClose: () => void; onCopy: (code: string) => void; };

export const StrategyTemplateCodeViewModal: React.FC<StrategyTemplateCodeViewModalProps> = ({ open, code, onClose, onCopy }) => {
  const { t } = useTranslation();
  return (
    <Modal title={t(CODE_MODAL_TITLE_KEY)} open={open} onCancel={onClose}
      footer={[<Button key="copy" icon={<CopyOutlined />} onClick={() => onCopy(code)}>{t(CODE_MODAL_ACTIONS_COPY_KEY)}</Button>, <Button key="close" onClick={onClose}>{t('common.close')}</Button>]}
      width={860}>
      <pre style={{ background: '#f5f5f5', padding: 16, borderRadius: 8, maxHeight: 360, overflow: 'auto', fontFamily: 'monospace', fontSize: 13 }}>{code}</pre>
      <CodeExplainPanel code={code} />
    </Modal>
  );
};
