import { Modal, Form, Input, Button, Space, Select, Switch, Alert } from 'antd';
import { useTranslation } from 'react-i18next';
import type { SystemConfig as AdminConfigType } from '@/client/admin';

interface Props {
  visible: boolean;
  currentConfig: AdminConfigType | null;
  form: any;
  isAIProviderCatalog: boolean;
  isEconAIConfig: boolean;
  isStrategyHealthConfig: boolean;
  onSave: (values: Record<string, unknown>) => void;
  onCancel: () => void;
  onFormatJson: () => void;
  onUseTemplate: () => void;
}

export default function SystemConfigEditModal({
  visible, currentConfig, form, isAIProviderCatalog,
  isEconAIConfig, isStrategyHealthConfig,
  onSave, onCancel, onFormatJson, onUseTemplate,
}: Props) {
  const { t } = useTranslation();

  return (
    <Modal
      title={t('admin.config.editConfig', { key: currentConfig?.key || '' })}
      open={visible}
      onCancel={onCancel}
      footer={null}
    >
      <Form form={form} onFinish={onSave} layout="vertical">
        {(isAIProviderCatalog || isStrategyHealthConfig) && (
          <Form.Item name="value" label={t('admin.config.value')} rules={[{ required: true }]}>
            <Input.TextArea
              placeholder={t('admin.config.placeholders.json')}
              rows={10}
              style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
            />
          </Form.Item>
        )}
        {isStrategyHealthConfig && (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('admin.config.thresholdInfo')}
            description={t('admin.config.thresholdDesc')}
          />
        )}
        {isEconAIConfig && (
          <>
            <Form.Item name="provider" label={t('admin.config.provider')} rules={[{ required: true }]}>
              <Select
                options={[
                  { label: t('admin.config.providerOptions.zhipu'), value: 'zhipu' },
                  { label: t('admin.config.providerOptions.deepseek'), value: 'deepseek' },
                  { label: t('admin.config.providerOptions.custom'), value: 'custom' },
                ]}
              />
            </Form.Item>
            <Form.Item name="api_key" label={t('admin.config.apiKey', { defaultValue: 'API Key' })} rules={[{ required: true }]}>
              <Input.Password placeholder={t('admin.config.placeholders.apiKey')} />
            </Form.Item>
            <Form.Item name="model" label={t('admin.config.modelName')} rules={[{ required: true }]}>
              <Input placeholder={t('admin.config.placeholders.model')} />
            </Form.Item>
            <Form.Item name="base_url" label={t('admin.config.baseUrlLabel')}>
              <Input placeholder={t('admin.config.placeholders.baseUrl')} />
            </Form.Item>
            <Form.Item name="enabled" label={t('admin.config.enableToggle')} valuePropName="checked">
              <Switch />
            </Form.Item>
          </>
        )}
        {!isAIProviderCatalog && !isEconAIConfig && !isStrategyHealthConfig && (
          <Form.Item name="value" label={t('admin.config.value')} rules={[{ required: true }]}>
            <Input placeholder={t('admin.config.placeholders.configValue')} />
          </Form.Item>
        )}
        <Form.Item name="description" label={t('admin.config.description')}>
          <Input.TextArea placeholder={t('admin.config.placeholders.description')} rows={3} />
        </Form.Item>
        <Form.Item>
          <Space>
            {isAIProviderCatalog && (
              <Button onClick={onFormatJson}>{t('admin.config.formatJson')}</Button>
            )}
            {isStrategyHealthConfig && (
              <>
                <Button onClick={onUseTemplate}>{t('admin.config.fillTemplate')}</Button>
                <Button onClick={onFormatJson}>{t('admin.config.formatJson')}</Button>
              </>
            )}
            <Button type="primary" htmlType="submit">{t('common.save')}</Button>
            <Button onClick={onCancel}>{t('common.cancel')}</Button>
          </Space>
        </Form.Item>
      </Form>
    </Modal>
  );
}
