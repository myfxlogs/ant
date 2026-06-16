import { useState, useEffect, useCallback } from 'react';
import {
  Table, Button, Modal, Form, Input, InputNumber,
  Switch, Space, Tag, Popconfirm, message, Typography, Spin
} from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ApiOutlined, ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons';
import { aiGatewayApi, type AIProviderInfo, type AIModelConfigInfo } from '@/client/aiGateway';

const { Title, Text } = Typography;

interface ProviderState extends AIProviderInfo {
  models?: AIModelConfigInfo[];
  modelsLoading?: boolean;
}

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
      console.error('listProviders failed', e);
      message.error('加载厂商列表失败');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadProviders(); }, [loadProviders]);

  // ── Provider CRUD ──
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
    } catch (e) {
      console.error('saveProvider failed', e);
    } finally {
      setSaving(false);
    }
  };

  const handleToggleProvider = async (p: ProviderState, enabled: boolean) => {
    try {
      await aiGatewayApi.updateProvider({ id: p.id, enabled });
      setProviders(prev => prev.map(x => x.id === p.id ? { ...x, enabled } : x));
    } catch (e) {
      console.error('toggleProvider failed', e);
      message.error('切换状态失败');
    }
  };

  // ── Model CRUD ──
  const loadModels = async (provider: ProviderState) => {
    setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: true } : p));
    try {
      const models = await aiGatewayApi.listModels(provider.id);
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, models, modelsLoading: false } : p));
    } catch {
      setProviders(prev => prev.map(p => p.id === provider.id ? { ...p, modelsLoading: false } : p));
      console.error('listModels failed', e); message.error('加载模型列表失败');
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
        id: editingModel?.id,
        providerId: currentProvider.id,
        modelName: v.modelName,
        displayName: v.displayName,
        pricePer1mInput: String(v.pricePer1mInput),
        pricePer1mOutput: String(v.pricePer1mOutput),
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
      console.error('deleteModel failed', e); message.error('删除失败');
    }
  };

  const handleToggleModel = async (provider: ProviderState, model: AIModelConfigInfo, enabled: boolean) => {
    try {
      await aiGatewayApi.upsertModel({
        id: model.id, providerId: provider.id, modelName: model.modelName,
        pricePer1mInput: model.pricePer1mInput, pricePer1mOutput: model.pricePer1mOutput,
        enabled,
      });
      setProviders(prev => prev.map(p => {
        if (p.id !== provider.id) return p;
        return { ...p, models: (p.models || []).map(m => m.id === model.id ? { ...m, enabled } : m) };
      }));
    } catch {
      message.error('切换失败');
    }
  };

  const expandedRowRender = (provider: ProviderState) => {
    const models = provider.models || [];
    return (
      <div style={{ padding: '8px 16px' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong><ThunderboltOutlined /> 模型列表 ({models.length})</Text>
          <Space>
            {provider.modelsLoading && <Spin size="small" />}
            <Button size="small" onClick={() => loadModels(provider)} icon={<ReloadOutlined />} />
            <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => handleAddModel(provider)}>
              添加模型
            </Button>
          </Space>
        </div>
        <Table
          dataSource={models}
          columns={[
            { title: '模型名', dataIndex: 'modelName', key: 'modelName', width: 180 },
            { title: '显示名', dataIndex: 'displayName', key: 'displayName', width: 160 },
            { title: '输入价格 ($/1M)', dataIndex: 'pricePer1mInput', key: 'pi', width: 140,
              render: (v: string) => <Text>${parseFloat(v).toFixed(6)}</Text> },
            { title: '输出价格 ($/1M)', dataIndex: 'pricePer1mOutput', key: 'po', width: 140,
              render: (v: string) => <Text>${parseFloat(v).toFixed(6)}</Text> },
            { title: '启用', dataIndex: 'enabled', key: 'enabled', width: 70,
              render: (_: boolean, record: AIModelConfigInfo) => (
                <Switch size="small" checked={record.enabled}
                  onChange={v => handleToggleModel(provider, record, v)} />
              ),
            },
            { title: '操作', key: 'actions', width: 120,
              render: (_: unknown, record: AIModelConfigInfo) => (
                <Space>
                  <Button size="small" icon={<EditOutlined />} onClick={() => handleEditModel(provider, record)} />
                  <Popconfirm title="确认删除此模型？" onConfirm={() => handleDeleteModel(provider, record.id)}>
                    <Button size="small" danger icon={<DeleteOutlined />} />
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
          rowKey="id"
          size="small"
          pagination={false}
          locale={{ emptyText: '暂无模型' }}
        />
      </div>
    );
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
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddProvider}>
            添加厂商
          </Button>
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
          expandedRowRender,
          expandRowByClick: true,
          onExpand: (expanded: boolean, record: ProviderState) => {
            if (expanded && !record.models) loadModels(record);
          },
        }}
        pagination={false}
        size="middle"
      />

      {/* Provider Edit Modal */}
      <Modal
        title={editingProvider ? '编辑厂商' : '添加厂商'}
        open={providerModalOpen}
        onOk={handleSaveProvider}
        onCancel={() => setProviderModalOpen(false)}
        confirmLoading={saving}
        width={560}
      >
        <Form form={providerForm} layout="vertical">
          {!editingProvider && (
            <Form.Item name="providerId" label="厂商 ID" rules={[{ required: true, message: '请输入厂商 ID' }]}>
              <Input placeholder="deepseek / openai / qwen ..." />
            </Form.Item>
          )}
          <Form.Item name="name" label="显示名称" rules={[{ required: true, message: '请输入显示名称' }]}>
            <Input placeholder="DeepSeek" />
          </Form.Item>
          <Form.Item name="baseUrl" label="Base URL" rules={[{ required: true, message: '请输入 Base URL' }]}>
            <Input placeholder="https://api.deepseek.com/v1" />
          </Form.Item>
          <Form.Item name="apiKey" label="API Key" extra={editingProvider ? '留空则不修改已有 Key' : 'API 密钥，加密存储'}>
            <Input
              placeholder={editingProvider ? '留空则不修改' : 'sk-...'}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Model Edit Modal */}
      <Modal
        title={editingModel ? '编辑模型' : '添加模型'}
        open={modelModalOpen}
        onOk={handleSaveModel}
        onCancel={() => setModelModalOpen(false)}
        confirmLoading={saving}
        width={500}
      >
        <Form form={modelForm} layout="vertical">
          <Form.Item name="modelName" label="模型名" rules={[{ required: true, message: '请输入模型名' }]}>
            <Input placeholder="deepseek-chat" />
          </Form.Item>
          <Form.Item name="displayName" label="显示名">
            <Input placeholder="DeepSeek Chat" />
          </Form.Item>
          <Space style={{ width: '100%' }}>
            <Form.Item name="pricePer1mInput" label="输入价格 ($/1M)" rules={[{ required: true }]}>
              <InputNumber min={0} step={0.00001} style={{ width: 210 }} placeholder="0.00014" />
            </Form.Item>
            <Form.Item name="pricePer1mOutput" label="输出价格 ($/1M)" rules={[{ required: true }]}>
              <InputNumber min={0} step={0.00001} style={{ width: 210 }} placeholder="0.00028" />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  );
}
