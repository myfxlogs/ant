import { Form, Input, Modal, Typography } from 'antd';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd/es/form';

const { Paragraph } = Typography;

export function RejectCodeModal(props: {
  open: boolean;
  sending: boolean;
  rejectText: string;
  onRejectTextChange: (v: string) => void;
  onConfirm: () => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const { open, sending, rejectText, onRejectTextChange, onConfirm, onCancel } = props;

  return (
    <Modal
      title={t('ai.debate.v2.rejectModalTitle', { defaultValue: 'Reject current code & rewrite' })}
      open={open}
      onOk={onConfirm}
      onCancel={onCancel}
      okText={t('ai.debate.v2.rejectModalOk', { defaultValue: 'Regenerate' })}
      cancelText={t('ai.debate.v2.rejectModalCancel', { defaultValue: 'Cancel' })}
      confirmLoading={sending}
      destroyOnClose
    >
      <Paragraph type="secondary" style={{ marginTop: 0 }}>
        {t('ai.debate.v2.rejectModalHint', {
          defaultValue: 'Describe what to fix (e.g. use RSI instead of MACD, tighten stop-loss to 0.5%, avoid trading on news days...). The model will regenerate the code based on your feedback.',
        })}
      </Paragraph>
      <Input.TextArea
        rows={5}
        value={rejectText}
        onChange={(e) => onRejectTextChange(e.target.value)}
        placeholder={t('ai.debate.v2.rejectModalPlaceholder', {
          defaultValue: 'What should be changed?',
        })}
      />
    </Modal>
  );
}

export function SaveTemplateModal(props: {
  open: boolean;
  saving: boolean;
  form: FormInstance<{ name: string; description: string }>;
  onConfirm: () => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const { open, saving, form, onConfirm, onCancel } = props;

  return (
    <Modal
      title={t('ai.debate.v2.saveModalTitle', { defaultValue: 'Save as strategy template' })}
      open={open}
      onOk={onConfirm}
      onCancel={onCancel}
      okText={t('ai.debate.v2.saveModalOk', { defaultValue: 'Save' })}
      cancelText={t('ai.debate.v2.saveModalCancel', { defaultValue: 'Cancel' })}
      confirmLoading={saving}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label={t('ai.debate.v2.saveFieldName', { defaultValue: 'Template name' })}
          rules={[{ required: true, message: t('ai.debate.v2.saveNameRequired', { defaultValue: 'Name is required' }) }]}
        >
          <Input maxLength={80} />
        </Form.Item>
        <Form.Item
          name="description"
          label={t('ai.debate.v2.saveFieldDesc', { defaultValue: 'Description (optional)' })}
        >
          <Input.TextArea rows={3} maxLength={500} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
