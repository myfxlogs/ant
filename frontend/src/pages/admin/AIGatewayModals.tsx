import { Modal, Form, Input, InputNumber, Space, Table, Button, Switch, Popconfirm, Spin, Typography, Select, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, ReloadOutlined, SearchOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd';
import type { AIProviderInfo, AIModelConfigInfo } from '@/client/aiGateway';
import { aiGatewayApi } from '@/client/aiGateway';
import { useTranslation } from 'react-i18next';
import { useState } from 'react';

const { Text } = Typography;

// Known pricing reference for common models (USD per 1M tokens).
// Used to auto-fill pricing when a model is selected from Discover.
const MODEL_PRICING_REF: Record<string, { displayName?: string; input: number; output: number }> = {
  // DeepSeek
  'deepseek-chat': { displayName: 'DeepSeek Chat', input: 0.27, output: 1.10 },
  'deepseek-reasoner': { displayName: 'DeepSeek Reasoner', input: 0.55, output: 2.19 },
  'deepseek-coder': { displayName: 'DeepSeek Coder', input: 0.14, output: 0.28 },
  // Zhipu GLM
  'glm-5.2': { displayName: 'GLM-5.2', input: 1.11, output: 3.89 },
  'glm-5.1': { displayName: 'GLM-5.1', input: 0.83, output: 3.33 },
  'glm-5-turbo': { displayName: 'GLM-5 Turbo', input: 0.69, output: 3.06 },
  'glm-5': { displayName: 'GLM-5', input: 0.56, output: 2.50 },
  'glm-4.7': { displayName: 'GLM-4.7', input: 0.28, output: 1.11 },
  'glm-4.7-flashx': { displayName: 'GLM-4.7 FlashX', input: 0.069, output: 0.42 },
  'glm-4.7-flash': { displayName: 'GLM-4.7 Flash (Free)', input: 0, output: 0 },
  'glm-4.5-air': { displayName: 'GLM-4.5 Air', input: 0.11, output: 0.28 },
  'glm-4-plus': { displayName: 'GLM-4 Plus', input: 0.69, output: 0.69 },
  'glm-4-air': { displayName: 'GLM-4 Air', input: 0.069, output: 0.069 },
  'glm-4-flashx-250414': { displayName: 'GLM-4 FlashX', input: 0.014, output: 0.014 },
  'glm-4-long': { displayName: 'GLM-4 Long', input: 0.14, output: 0.14 },
  // OpenAI
  'gpt-4o': { displayName: 'GPT-4o', input: 2.50, output: 10.00 },
  'gpt-4o-mini': { displayName: 'GPT-4o Mini', input: 0.15, output: 0.60 },
  'gpt-4-turbo': { displayName: 'GPT-4 Turbo', input: 10.00, output: 30.00 },
  'o1': { displayName: 'o1', input: 15.00, output: 60.00 },
  'o1-mini': { displayName: 'o1 Mini', input: 3.00, output: 12.00 },
};

function lookupModelPricing(modelName: string) {
  const key = modelName.toLowerCase().trim();
  // exact match first
  if (MODEL_PRICING_REF[key]) return MODEL_PRICING_REF[key];
  // prefix match (e.g. "glm-4.7-20250101" → "glm-4.7")
  for (const ref of Object.keys(MODEL_PRICING_REF)) {
    if (key.startsWith(ref)) return MODEL_PRICING_REF[ref];
  }
  return undefined;
}

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
  currentProvider: ProviderState | null;
  modelForm: FormInstance;
  onSaveModel: () => void;
  onCloseModel: () => void;
}

export function AIGatewayModals({
  providerModalOpen, editingProvider, providerForm, saving, onSaveProvider, onCloseProvider,
  modelModalOpen, editingModel, currentProvider, modelForm, onSaveModel, onCloseModel,
}: ModalsProps) {
  const { t } = useTranslation();
  const [discoveredModels, setDiscoveredModels] = useState<string[]>([]);
  const [discovering, setDiscovering] = useState(false);

  const handleDiscoverModels = async () => {
    if (!currentProvider) return;
    setDiscovering(true);
    try {
      const models = await aiGatewayApi.discoverGatewayModels(currentProvider.id);
      setDiscoveredModels(models);
      if (models.length === 0) {
        message.info(t('admin.aiGateway.noModelsDiscovered', { defaultValue: 'No models discovered. Check API key and base URL.' }));
      }
    } catch {
      message.error(t('admin.aiGateway.discoverFailed', { defaultValue: 'Failed to discover models' }));
    } finally {
      setDiscovering(false);
    }
  };

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
              <Input placeholder={t('admin.aiGateway.providerIdPlaceholder')} />
            </Form.Item>
          )}
          <Form.Item name="name" label={t('admin.aiGateway.displayName', { defaultValue: 'Display Name' })} rules={[{ required: true, message: t('admin.aiGateway.displayNameRequired', { defaultValue: 'Please enter display name' }) }]}>
            <Input placeholder={t('admin.aiGateway.displayNamePlaceholder')} />
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
          <Form.Item label={t('admin.aiGateway.modelName', { defaultValue: 'Model Name' })} required>
            <Space.Compact style={{ width: '100%' }}>
              <Form.Item name="modelName" noStyle rules={[{ required: true, message: t('admin.aiGateway.modelNameRequired', { defaultValue: 'Please enter or select model name' }) }]}>
                <Select
                  showSearch
                  allowClear
                  placeholder="deepseek-v4-flash"
                  style={{ width: '100%' }}
                  options={discoveredModels.map(m => ({ label: m, value: m }))}
                  notFoundContent={discovering ? <Spin size="small" /> : undefined}
                  filterOption={(input, option) =>
                    (option?.label as string)?.toLowerCase().includes(input.toLowerCase())
                  }
                  onChange={(value: string) => {
                    if (!value) return;
                    const ref = lookupModelPricing(value);
                    if (ref) {
                      if (ref.displayName) modelForm.setFieldValue('displayName', ref.displayName);
                      modelForm.setFieldValue('pricePer1mInput', ref.input);
                      modelForm.setFieldValue('pricePer1mOutput', ref.output);
                    }
                  }}
                />
              </Form.Item>
              <Button
                icon={<SearchOutlined />}
                onClick={handleDiscoverModels}
                loading={discovering}
                style={{ marginLeft: 8 }}
              >
                {t('admin.aiGateway.discover', { defaultValue: 'Discover' })}
              </Button>
            </Space.Compact>
          </Form.Item>
          <Form.Item name="displayName" label={t('admin.aiGateway.displayName', { defaultValue: 'Display Name' })}>
            <Input placeholder={t('admin.aiGateway.displayNamePlaceholder', { defaultValue: 'DeepSeek Chat' })} />
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
