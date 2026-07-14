import { Modal, Form, Input, InputNumber, Space, Table, Button, Switch, Popconfirm, Spin, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd';
import type { AIProviderInfo, AIModelConfigInfo } from '@/client/aiGateway';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

export interface ProviderState extends AIProviderInfo {
  models?: AIModelConfigInfo[];
  modelsLoading?: boolean;
}

interface ModalsProps {
  providerModalOpen: boolean;
  editingProvider: ProviderState | null;
  providerForm: FormInstance;
  saving: boolean;
  onSaveProvider: () => void;
  onCloseProvider: () => void;

  modelModalOpen: boolean;
  editingModel: AIModelConfigInfo | null;
  modelForm: FormInstance;
  onSaveModel: () => void;
  onCloseModel: () => void;
}

export function AIGatewayModals({
  providerModalOpen, editingProvider, providerForm, saving, onSaveProvider, onCloseProvider,
  modelModalOpen, editingModel, modelForm, onSaveModel, onCloseModel,
}: ModalsProps) {
  const { t } = useTranslation();
  return (
    <>
      <Modal
        title={editingProvider ? t('admin.aiGateway.editProvider', { defaultValue: 'Edit Provider' }) : t('admin.aiGateway.addProvider', { defaultValue: 'Add Provider' })}
        open={providerModalOpen}
        onOk={onSaveProvider}
        onCancel={onCloseProvider}
        confirmLoading={saving}
        width={560}
      >
        <Form form={providerForm} layout="vertical">
          {!editingProvider && (
            <Form.Item name="providerId" label={t('admin.aiGateway.providerId', { defaultValue: 'Provider ID' })} rules={[{ required: true, message: t('admin.aiGateway.providerIdRequired', { defaultValue: 'Please enter provider ID' }) }]}>
              <Input placeholder="deepseek / openai / qwen ..." />
            </Form.Item>
          )}
          <Form.Item name="name" label={t('admin.aiGateway.displayName', { defaultValue: 'Display Name' })} rules={[{ required: true, message: t('admin.aiGateway.displayNameRequired', { defaultValue: 'Please enter display name' }) }]}>
            <Input placeholder="DeepSeek" />
          </Form.Item>
          <Form.Item name="baseUrl" label={t('admin.aiGateway.baseUrl', { defaultValue: 'Base URL' })} rules={[{ required: true, message: t('admin.aiGateway.baseUrlRequired', { defaultValue: 'Please enter Base URL' }) }]}>
            <Input placeholder="https://api.deepseek.com/v1" />
          </Form.Item>
          <Form.Item name="apiKey" label={t('admin.aiGateway.apiKeyLabel', { defaultValue: 'API Key' })} extra={editingProvider ? t('admin.aiGateway.apiKeyEditHint', { defaultValue: 'Leave empty to keep existing key' }) : t('admin.aiGateway.apiKeyHint', { defaultValue: 'API key, encrypted at rest' })}>
            <Input placeholder={editingProvider ? t('admin.aiGateway.apiKeyEditPlaceholder', { defaultValue: 'Leave empty to keep' }) : 'sk-...'} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingModel ? t('admin.aiGateway.editModel', { defaultValue: 'Edit Model' }) : t('admin.aiGateway.addModel', { defaultValue: 'Add Model' })}
        open={modelModalOpen}
        onOk={onSaveModel}
        onCancel={onCloseModel}
        confirmLoading={saving}
        width={500}
      >
        <Form form={modelForm} layout="vertical">
          <Form.Item name="modelName" label={t('admin.aiGateway.modelName', { defaultValue: 'Model Name' })} rules={[{ required: true, message: t('admin.aiGateway.modelNameRequired', { defaultValue: 'Please enter model name' }) }]}>
            <Input placeholder="deepseek-chat" />
          </Form.Item>
          <Form.Item name="displayName" label={t('admin.aiGateway.displayName', { defaultValue: 'Display Name' })}>
            <Input placeholder="DeepSeek Chat" />
          </Form.Item>
          <Space style={{ width: '100%' }}>
            <Form.Item name="pricePer1mInput" label={t('admin.aiGateway.priceInput', { defaultValue: 'Input Price ($/1M)' })} rules={[{ required: true }]}>
              <InputNumber min={0} step={0.00001} style={{ width: 210 }} placeholder="0.00014" />
            </Form.Item>
            <Form.Item name="pricePer1mOutput" label={t('admin.aiGateway.priceOutput', { defaultValue: 'Output Price ($/1M)' })} rules={[{ required: true }]}>
              <InputNumber min={0} step={0.00001} style={{ width: 210 }} placeholder="0.00028" />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </>
  );
}

interface ExpandedRowProps {
  provider: ProviderState;
  onLoadModels: (p: ProviderState) => void;
  onAddModel: (p: ProviderState) => void;
  onEditModel: (p: ProviderState, m: AIModelConfigInfo) => void;
  onDeleteModel: (p: ProviderState, modelId: string) => void;
  onToggleModel: (p: ProviderState, m: AIModelConfigInfo, enabled: boolean) => void;
}

export function ProviderExpandedRow({
  provider, onLoadModels, onAddModel, onEditModel, onDeleteModel, onToggleModel,
}: ExpandedRowProps) {
  const { t } = useTranslation();
  const models = provider.models || [];
  return (
    <div style={{ padding: '8px 16px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong><ThunderboltOutlined /> {t('admin.aiGateway.modelList', { count: models.length, defaultValue: `Models (${models.length})` })}</Text>
        <Space>
          {provider.modelsLoading && <Spin size="small" />}
          <Button size="small" onClick={() => onLoadModels(provider)} icon={<ReloadOutlined />} />
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => onAddModel(provider)}>
            {t('admin.aiGateway.addModel', { defaultValue: 'Add Model' })}
          </Button>
        </Space>
      </div>
      <Table
        dataSource={models}
        columns={[
          { title: t('admin.aiGateway.modelName', { defaultValue: 'Model Name' }), dataIndex: 'modelName', key: 'modelName', width: 180 },
          { title: t('admin.aiGateway.displayName', { defaultValue: 'Display Name' }), dataIndex: 'displayName', key: 'displayName', width: 160 },
          { title: t('admin.aiGateway.priceInput', { defaultValue: 'Input Price ($/1M)' }), dataIndex: 'pricePer1mInput', key: 'pi', width: 140,
            render: (v: string) => <Text>${parseFloat(v).toFixed(6)}</Text> },
          { title: t('admin.aiGateway.priceOutput', { defaultValue: 'Output Price ($/1M)' }), dataIndex: 'pricePer1mOutput', key: 'po', width: 140,
            render: (v: string) => <Text>${parseFloat(v).toFixed(6)}</Text> },
          { title: t('common.enabled', { defaultValue: 'Enabled' }), dataIndex: 'enabled', key: 'enabled', width: 70,
            render: (_: boolean, record: AIModelConfigInfo) => (
              <Switch size="small" checked={record.enabled}
                onChange={v => onToggleModel(provider, record, v)} />
            ),
          },
          { title: t('common.action', { defaultValue: 'Action' }), key: 'actions', width: 120,
            render: (_: unknown, record: AIModelConfigInfo) => (
              <Space>
                <Button size="small" icon={<EditOutlined />} onClick={() => onEditModel(provider, record)} />
                <Popconfirm title={t('admin.aiGateway.confirmDeleteModel', { defaultValue: 'Delete this model?' })} onConfirm={() => onDeleteModel(provider, record.id)}>
                  <Button size="small" danger icon={<DeleteOutlined />} />
                </Popconfirm>
              </Space>
            ),
          },
        ]}
        rowKey="id"
        size="small"
        pagination={false}
        locale={{ emptyText: t('admin.aiGateway.noModels', { defaultValue: 'No models' }) }}
      />
    </div>
  );
}
