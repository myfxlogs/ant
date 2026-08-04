import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Form, Space, Tag, message, Typography, Switch } from 'antd';
import { PlusOutlined, EditOutlined, ApiOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { aiGatewayApi, type AIModelConfigInfo } from '@/client/aiGateway';
import { AIGatewayModals, ProviderExpandedRow, type ProviderState } from './AIGatewayModals';

const { Title, Text } = Typography;

export default function AIGatewayManagement() {
  const { t } = useTranslation();
  const [providers, setProviders] = useState<ProviderState[]>([]);
  const [loading, setLoading] = useState(true);
  const [editingProvider, setEditingProvider] = useState<ProviderState | null>(null);
  const [providerModalOpen, setProviderModalOpen] = useState(false);
  const [modelModalOpen, setModelModalOpen] = useState(false);
  const [editingModel, setEditingModel] = useState<AIModelConfigInfo | null>(null);
  const [currentProvider, setCurrentProvider] = useState<ProviderState | null>(null);
  const [saving, setSaving] = useState(false);
  const [providerForm] = Form.useForm();
  const [modelForm] = Form.useForm();

  const loadProviders = useCallback(async () => {
    setLoading(true);
    try {
      const list = await aiGatewayApi.listProviders();
      setProviders(list.map(p => ({ ...p, models: undefined, modelsLoading: false })));
    } catch (_e) {
      message.error(t('admin.aiGateway.errors.loadProviders', { defaultValue: 'Failed to load providers' }));
    } finally {
      setLoading(false);
    }
  }, [t]);

  useEffect(() => { loadProviders(); }, [loadProviders]);

  const handleAddProvider = () => {
    setEditingProvider(null);
    providerForm.resetFields();
    setProviderModalOpen(true);
  };

  const handleEditProvider = (p: ProviderState) => {
    setEditingProvider(p);
    providerForm.setFieldsValue({ name: p.name, baseUrl: p.baseUrl });
    setProviderModalOpen(true);
  };

  const handleSaveProvider = async () => {
    try {
      const v = await providerForm.validateFields();
      setSaving(true);
      if (editingProvider) {
        const payload: Record<string, unknown> = { id: editingProvider.id };
        if (v.name !== editingProvider.name) payload.name = v.name;
        if (v.baseUrl !== editingProvider.baseUrl) payload.baseUrl = v.baseUrl;
        if (v.apiKey && v.apiKey.trim()) payload.apiKey = v.apiKey.trim();
        await aiGatewayApi.updateProvider(payload);
        message.success(t('common.saveSuccess', { defaultValue: 'Saved successfully' }));
      } else {
        await aiGatewayApi.createProvider({
          providerId: v.providerId,
          name: v.name,
          baseUrl: v.baseUrl,
          apiKey: v.apiKey.trim(),
        });
        message.success(t('common.saveSuccess', { defaultValue: 'Saved successfully' }));
      }
      setProviderModalOpen(false);
      await loadProviders();
    } catch {
      // validation error
    } finally {
      setSaving(false);
    }
  };

  const handleToggleProvider = async (p: ProviderState, enabled: boolean) => {
    try {
      await aiGatewayApi.updateProvider({ id: p.id, enabled });
      setProviders(prev => prev.map(x => x.id === p.id ? { ...x, enabled } : x));
    } catch {
      message.error(t('admin.aiGateway.errors.toggleFailed', { defaultValue: 'Toggle failed' }));
    }
  };

  const loadModels = async (provider: ProviderState) => {
    setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: true } : p));
    try {
      const models = await aiGatewayApi.listModels(provider.id);
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, models, modelsLoading: false } : p));
    } catch {
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: false } : p));
      message.error(t('admin.aiGateway.errors.loadModels', { defaultValue: 'Failed to load models' }));
    }
  };

  const handleAddModel = (provider: ProviderState) => {
    setCurrentProvider(provider);
    setEditingModel(null);
    modelForm.resetFields();
    setModelModalOpen(true);
  };

  const handleEditModel = (provider: ProviderState, model: AIModelConfigInfo) => {
    setCurrentProvider(provider);
    setEditingModel(model);
    modelForm.setFieldsValue(model);
    setModelModalOpen(true);
  };

  const handleSaveModel = async () => {
    try {
      const v = await modelForm.validateFields();
      if (!currentProvider) return;
      setSaving(true);
      await aiGatewayApi.upsertModel({
        id: editingModel?.id, providerId: currentProvider.id,
        modelName: v.modelName, displayName: v.displayName,
        pricePer1mInput: String(v.pricePer1mInput), pricePer1mOutput: String(v.pricePer1mOutput),
      });
      message.success(t('common.saveSuccess', { defaultValue: 'Saved successfully' }));
      setModelModalOpen(false);
      loadModels(currentProvider);
    } catch {
      // validation or API error
    } finally {
      setSaving(false);
    }
  };

  const handleDeleteModel = async (provider: ProviderState, modelId: string) => {
    try {
      await aiGatewayApi.deleteModel(modelId);
      message.success(t('common.deleted', { defaultValue: 'Deleted' }));
      loadModels(provider);
    } catch {
      message.error(t('common.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  };

  const handleToggleModel = async (provider: ProviderState, model: AIModelConfigInfo, enabled: boolean) => {
    try {
      await aiGatewayApi.upsertModel({
        id: model.id, providerId: provider.id, modelName: model.modelName,
        pricePer1mInput: model.pricePer1mInput, pricePer1mOutput: model.pricePer1mOutput, enabled,
      });
      setProviders(prev => prev.map(p => {
        if (p.id !== provider.id) return p;
        return { ...p, models: (p.models || []).map(m => m.id === model.id ? { ...m, enabled } : m) };
      }));
    } catch {
      message.error(t('admin.aiGateway.errors.toggleFailed', { defaultValue: 'Toggle failed' }));
    }
  };

  return (
    <div style={{ maxWidth: 1200 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <ApiOutlined style={{ marginRight: 8, color: '#722ed1' }} />
            {t('admin.aiGateway.title', { defaultValue: 'AI Gateway Management' })}
          </Title>
          <Text type="secondary" style={{ fontSize: 13 }}>
            {t('admin.aiGateway.description', { defaultValue: 'Manage AI providers, models, and pricing. Users select from available models, billed by token from wallet.' })}
          </Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadProviders} loading={loading}>{t('common.refresh', { defaultValue: 'Refresh' })}</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddProvider}>{t('admin.aiGateway.addProvider', { defaultValue: 'Add Provider' })}</Button>
        </Space>
      </div>

      <Table
        dataSource={providers}
        columns={[
          { title: t('admin.aiGateway.provider', { defaultValue: 'Provider' }), key: 'name', width: 200,
            render: (_: unknown, r: ProviderState) => <Space>
              <Tag color="blue">{r.providerId}</Tag>
              <Text strong>{r.name}</Text>
            </Space>,
          },
          { title: t('admin.aiGateway.columns.baseUrl', { defaultValue: 'Base URL' }), dataIndex: 'baseUrl', key: 'baseUrl', ellipsis: true, width: 280,
            render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text> },
          { title: t('admin.aiGateway.columns.apiKey', { defaultValue: 'API Key' }), dataIndex: 'hasApiKey', key: 'hasApiKey', width: 120,
            render: (v: boolean) => v ? <Tag color="green">{t('admin.aiGateway.configured', { defaultValue: 'Configured' })}</Tag> : <Tag color="default">{t('admin.aiGateway.notConfigured', { defaultValue: 'Not configured' })}</Tag>,
          },
          { title: t('admin.aiGateway.models', { defaultValue: 'Models' }), key: 'models', width: 80,
            render: (_: unknown, r: ProviderState) => <Tag>{r.models?.length ?? '?'} {t('common.unit', { defaultValue: 'units' })}</Tag>,
          },
          { title: t('common.enabled', { defaultValue: 'Enabled' }), dataIndex: 'enabled', key: 'enabled', width: 70,
            render: (v: boolean, r: ProviderState) =>
              <Switch size="small" checked={v} onChange={enabled => handleToggleProvider(r, enabled)} />,
          },
          { title: t('common.action', { defaultValue: 'Action' }), key: 'actions', width: 120,
            render: (_: unknown, r: ProviderState) => (
              <Button size="small" icon={<EditOutlined />} onClick={() => handleEditProvider(r)} />
            ),
          },
        ]}
        rowKey="id"
        loading={loading}
        expandable={{
          expandedRowRender: (provider: ProviderState) => (
            <ProviderExpandedRow
              provider={provider}
              onLoadModels={loadModels}
              onAddModel={handleAddModel}
              onEditModel={handleEditModel}
              onDeleteModel={handleDeleteModel}
              onToggleModel={handleToggleModel}
            />
          ),
          expandRowByClick: true,
          onExpand: (expanded: boolean, record: ProviderState) => {
            if (expanded && !record.models) loadModels(record);
          },
        }}
        pagination={false}
        size="middle"
      />

      <AIGatewayModals
        providerModalOpen={providerModalOpen}
        editingProvider={editingProvider}
        providerForm={providerForm}
        saving={saving}
        onSaveProvider={handleSaveProvider}
        onCloseProvider={() => setProviderModalOpen(false)}
        modelModalOpen={modelModalOpen}
        editingModel={editingModel}
        modelForm={modelForm}
        onSaveModel={handleSaveModel}
        onCloseModel={() => setModelModalOpen(false)}
      />
    </div>
  );
}
