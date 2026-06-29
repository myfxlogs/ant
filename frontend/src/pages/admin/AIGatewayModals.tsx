import { Modal, Form, Input, InputNumber, Space, Table, Button, Switch, Popconfirm, Spin, Typography } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, ThunderboltOutlined, ReloadOutlined } from '@ant-design/icons';
import type { FormInstance } from 'antd';
import type { AIProviderInfo, AIModelConfigInfo } from '@/client/aiGateway';

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
  return (
    <>
      <Modal
        title={editingProvider ? '编辑厂商' : '添加厂商'}
        open={providerModalOpen}
        onOk={onSaveProvider}
        onCancel={onCloseProvider}
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
            <Input placeholder={editingProvider ? '留空则不修改' : 'sk-...'} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={editingModel ? '编辑模型' : '添加模型'}
        open={modelModalOpen}
        onOk={onSaveModel}
        onCancel={onCloseModel}
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
  const models = provider.models || [];
  return (
    <div style={{ padding: '8px 16px' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <Text strong><ThunderboltOutlined /> 模型列表 ({models.length})</Text>
        <Space>
          {provider.modelsLoading && <Spin size="small" />}
          <Button size="small" onClick={() => onLoadModels(provider)} icon={<ReloadOutlined />} />
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={() => onAddModel(provider)}>
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
                onChange={v => onToggleModel(provider, record, v)} />
            ),
          },
          { title: '操作', key: 'actions', width: 120,
            render: (_: unknown, record: AIModelConfigInfo) => (
              <Space>
                <Button size="small" icon={<EditOutlined />} onClick={() => onEditModel(provider, record)} />
                <Popconfirm title="确认删除此模型？" onConfirm={() => onDeleteModel(provider, record.id)}>
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
}
