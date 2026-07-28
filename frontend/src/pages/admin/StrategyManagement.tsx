import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Tag, Space, Modal, Input, Popconfirm, message, Tabs, Form } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  STRATEGY_COLUMNS_ACTIONS_KEY, STRATEGY_COLUMNS_CODE_KEY,
  STRATEGY_COLUMNS_DESCRIPTION_KEY, STRATEGY_COLUMNS_NAME_KEY, STRATEGY_COLUMNS_TAGS_KEY,
  STRATEGY_COLUMNS_TAGS_PLACEHOLDER_KEY, STRATEGY_COLUMNS_USES_KEY,
  STRATEGY_MESSAGES_DELETE_FAILED_KEY, STRATEGY_MESSAGES_LOAD_PRESET_FAILED_KEY,
  STRATEGY_MESSAGES_PRESET_CREATED_KEY, STRATEGY_MESSAGES_PRESET_DELETED_KEY,
  STRATEGY_MESSAGES_PRESET_UPDATED_KEY, STRATEGY_MESSAGES_SAVE_FAILED_KEY,
  STRATEGY_PRESET_ADD_KEY, STRATEGY_PRESET_CREATE_KEY, STRATEGY_PRESET_DELETE_CONFIRM_KEY,
  STRATEGY_PRESET_EDIT_KEY, STRATEGY_TABS_ALL_STRATEGIES_KEY, STRATEGY_TABS_PRESET_KEY,
} from '@/gen/ant/v1/i18n/admin_keys';
import { adminStrategyApi } from '@/client/admin';
import type { SystemStrategy } from '@/gen/ant/v1/admin_strategy_pb';
import StrategyManagementAll from './StrategyManagementAll';

const { TextArea } = Input;

export default function StrategyManagement() {
  const { t } = useTranslation();
  const [tab, setTab] = useState('preset');

  const [presets, setPresets] = useState<SystemStrategy[]>([]);
  const [presetsLoading, setPresetsLoading] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingPreset, setEditingPreset] = useState<SystemStrategy | null>(null);
  const [presetSaving, setPresetSaving] = useState(false);
  const [form] = Form.useForm();

  const fetchPresets = useCallback(async () => {
    setPresetsLoading(true);
    try {
      const resp = await adminStrategyApi.listSystemStrategies();
      setPresets(resp.strategies || []);
    } catch { message.error(t(STRATEGY_MESSAGES_LOAD_PRESET_FAILED_KEY)); }
    finally { setPresetsLoading(false); }
  }, [t]);

  useEffect(() => { fetchPresets(); }, [fetchPresets]);

  const openCreatePreset = () => {
    setEditingPreset(null);
    form.resetFields();
    setEditModalOpen(true);
  };

  const openEditPreset = (s: SystemStrategy) => {
    setEditingPreset(s);
    form.setFieldsValue({ name: s.name, description: s.description, code: s.code, tags: (s.tags || []).join(', ') });
    setEditModalOpen(true);
  };

  const handleSavePreset = async () => {
    const values = await form.validateFields();
    const tags = values.tags ? String(values.tags).split(',').map((tg: string) => tg.trim()).filter(Boolean) : [];
    setPresetSaving(true);
    try {
      if (editingPreset) {
        await adminStrategyApi.updateSystemStrategy({ id: editingPreset.id, name: values.name, description: values.description, code: values.code, tags });
        message.success(t(STRATEGY_MESSAGES_PRESET_UPDATED_KEY));
      } else {
        await adminStrategyApi.createSystemStrategy({ name: values.name, description: values.description, code: values.code, tags });
        message.success(t(STRATEGY_MESSAGES_PRESET_CREATED_KEY));
      }
      setEditModalOpen(false);
      fetchPresets();
    } catch { message.error(t(STRATEGY_MESSAGES_SAVE_FAILED_KEY)); }
    finally { setPresetSaving(false); }
  };

  const handleDeletePreset = async (id: string) => {
    try {
      await adminStrategyApi.deleteSystemStrategy(id);
      message.success(t(STRATEGY_MESSAGES_PRESET_DELETED_KEY));
      fetchPresets();
    } catch { message.error(t(STRATEGY_MESSAGES_DELETE_FAILED_KEY)); }
  };

  const presetColumns = [
    { title: t(STRATEGY_COLUMNS_NAME_KEY), dataIndex: 'name', key: 'name', width: 200 },
    { title: t(STRATEGY_COLUMNS_DESCRIPTION_KEY), dataIndex: 'description', key: 'description', ellipsis: true },
    { title: t(STRATEGY_COLUMNS_TAGS_KEY), dataIndex: 'tags', key: 'tags', render: (tags: string[]) => tags?.map(tg => <Tag key={tg}>{tg}</Tag>) },
    { title: t(STRATEGY_COLUMNS_USES_KEY), dataIndex: 'useCount', key: 'useCount', width: 80 },
    {
      title: t(STRATEGY_COLUMNS_ACTIONS_KEY), key: 'actions', width: 120,
      render: (_: unknown, r: SystemStrategy) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditPreset(r)} />
          <Popconfirm title={t(STRATEGY_PRESET_DELETE_CONFIRM_KEY)} onConfirm={() => handleDeletePreset(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Tabs activeKey={tab} onChange={setTab} items={[
        {
          key: 'preset',
          label: t(STRATEGY_TABS_PRESET_KEY),
          children: (
            <div>
              <div style={{ marginBottom: 12 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePreset}>
                  {t(STRATEGY_PRESET_ADD_KEY)}
                </Button>
              </div>
              <Table rowKey="id" columns={presetColumns} dataSource={presets} loading={presetsLoading} size="small" pagination={false} />
            </div>
          ),
        },
        {
          key: 'all',
          label: t(STRATEGY_TABS_ALL_STRATEGIES_KEY),
          children: <StrategyManagementAll />,
        },
      ]} />

      <Modal title={editingPreset ? t(STRATEGY_PRESET_EDIT_KEY) : t(STRATEGY_PRESET_CREATE_KEY)} open={editModalOpen}
        onCancel={() => setEditModalOpen(false)} onOk={handleSavePreset} confirmLoading={presetSaving} width={700}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t(STRATEGY_COLUMNS_NAME_KEY)} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t(STRATEGY_COLUMNS_DESCRIPTION_KEY)}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label={t(STRATEGY_COLUMNS_CODE_KEY)} rules={[{ required: true }]}>
            <TextArea rows={12} style={{ fontFamily: 'monospace', fontSize: 13 }} />
          </Form.Item>
          <Form.Item name="tags" label={t(STRATEGY_COLUMNS_TAGS_KEY)}>
            <Input placeholder={t(STRATEGY_COLUMNS_TAGS_PLACEHOLDER_KEY)} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
