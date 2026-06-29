import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Modal, Form, Input, Space, Tag, message, Typography, Switch } from 'antd';
import { PlusOutlined, EditOutlined, ApiOutlined, ReloadOutlined } from '@ant-design/icons';
import { aiGatewayApi, type AIProviderInfo, type AIModelConfigInfo } from '@/client/aiGateway';
import { AIGatewayModals, ProviderExpandedRow, type ProviderState } from './AIGatewayModals';

const { Title, Text } = Typography;

export default function AIGatewayManagement() {
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
    } catch (e) {
      message.error('加载厂商列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

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
        const payload: Record<string, any> = { id: editingProvider.id };
        if (v.name !== editingProvider.name) payload.name = v.name;
        if (v.baseUrl !== editingProvider.baseUrl) payload.baseUrl = v.baseUrl;
        if (v.apiKey && v.apiKey.trim()) payload.apiKey = v.apiKey.trim();
        await aiGatewayApi.updateProvider(payload);
        message.success('保存成功');
      } else {
        message.info('添加厂商功能待后端接口支持');
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
      message.error('切换状态失败');
    }
  };

  const loadModels = async (provider: ProviderState) => {
    setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: true } : p));
    try {
      const models = await aiGatewayApi.listModels(provider.id);
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, models, modelsLoading: false } : p));
    } catch {
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: false } : p));
      message.error('加载模型列表失败');
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
      message.success('保存成功');
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
      message.success('已删除');
      loadModels(provider);
    } catch {
      message.error('删除失败');
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
      message.error('切换失败');
    }
  };

  return (
    <div style={{ maxWidth: 1200 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div>
          <Title level={4} style={{ margin: 0 }}>
            <ApiOutlined style={{ marginRight: 8, color: '#722ed1' }} />
            AI 网关管理
          </Title>
          <Text type="secondary" style={{ fontSize: 13 }}>
            管理平台中转的 AI 厂商、模型和定价。用户从可用模型列表中选择，按 Token 计费从钱包扣款。
          </Text>
        </div>
        <Space>
          <Button icon={<ReloadOutlined />} onClick={loadProviders} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddProvider}>添加厂商</Button>
        </Space>
      </div>

      <Table
        dataSource={providers}
        columns={[
          { title: '厂商', key: 'name', width: 200,
            render: (_: unknown, r: ProviderState) => <Space>
              <Tag color="blue">{r.providerId}</Tag>
              <Text strong>{r.name}</Text>
            </Space>,
          },
          { title: 'Base URL', dataIndex: 'baseUrl', key: 'baseUrl', ellipsis: true, width: 280,
            render: (v: string) => <Text code style={{ fontSize: 11 }}>{v}</Text> },
          { title: 'API Key', dataIndex: 'hasApiKey', key: 'hasApiKey', width: 120,
            render: (v: boolean) => v ? <Tag color="green">已配置</Tag> : <Tag color="default">未配置</Tag>,
          },
          { title: '模型', key: 'models', width: 80,
            render: (_: unknown, r: ProviderState) => <Tag>{r.models?.length ?? '?'} 个</Tag>,
          },
          { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70,
            render: (v: boolean, r: ProviderState) =>
              <Switch size="small" checked={v} onChange={enabled => handleToggleProvider(r, enabled)} />,
          },
          { title: '操作', key: 'actions', width: 120,
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
