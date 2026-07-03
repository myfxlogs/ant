import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Input, Select, Space, Tag, Switch, Typography, Popconfirm, message } from 'antd';
import { PlusOutlined, DeleteOutlined, SettingOutlined, SafetyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { getManagedSettings, setManagedSetting, deleteManagedSetting } from '@/client/adminAgentSettings';
import type { ManagedSettingEntry } from '@/gen/ant/v1/admin_settings_pb';

const { TextArea } = Input;
const { Title, Text } = Typography;

const SETTING_LABELS: Record<string, string> = {
  allowed_models: '模型白名单 (逗号分隔)',
  enforce_allowed_models: '强制模型白名单',
  max_cost_ceiling_usd: '单策略成本上限 (USD)',
  max_iterations_per_strategy: '单策略最大迭代次数',
  disable_live_trading: '全局禁如实盘交易',
  required_risk_gates: '必需风控门 (逗号分隔)',
  audit_retention_days: '审计日志保留天数',
  allow_managed_rules_only: '仅允许 Managed 层权限规则',
};

const BOOL_KEYS = new Set(['enforce_allowed_models', 'disable_live_trading', 'allow_managed_rules_only']);

export default function AdminSettingsPage() {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<ManagedSettingEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [editKey, setEditKey] = useState('');
  const [editValue, setEditValue] = useState('');
  const [isNew, setIsNew] = useState(false);

  const fetchSettings = useCallback(async () => {
    setLoading(true);
    try {
      const list = await getManagedSettings();
      setSettings(list);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchSettings(); }, [fetchSettings]);

  const handleEdit = (entry: ManagedSettingEntry) => {
    setEditKey(entry.key);
    setEditValue(entry.value);
    setIsNew(false);
    setModalOpen(true);
  };

  const handleAdd = () => {
    setEditKey('');
    setEditValue('');
    setIsNew(true);
    setModalOpen(true);
  };

  const handleSave = async () => {
    if (!editKey.trim()) return;
    try {
      const ok = await setManagedSetting(editKey.trim(), editValue.trim());
      if (ok) {
        message.success('保存成功');
        setModalOpen(false);
        fetchSettings();
      } else {
        message.error('保存失败');
      }
    } catch {
      message.error('保存失败');
    }
  };

  const handleDelete = async (key: string) => {
    try {
      await deleteManagedSetting(key);
      message.success('已删除');
      fetchSettings();
    } catch {
      message.error('删除失败');
    }
  };

  const handleToggleBool = async (key: string, currentVal: string) => {
    const newVal = currentVal === 'true' ? 'false' : 'true';
    try {
      await setManagedSetting(key, newVal);
      fetchSettings();
    } catch {
      message.error('操作失败');
    }
  };

  const columns = [
    {
      title: '设置项',
      dataIndex: 'key',
      key: 'key',
      width: 240,
      render: (key: string) => (
        <span className="font-medium">{SETTING_LABELS[key] || key}</span>
      ),
    },
    {
      title: '值',
      dataIndex: 'value',
      key: 'value',
      render: (value: string, record: ManagedSettingEntry) => {
        if (BOOL_KEYS.has(record.key)) {
          return (
            <Switch
              checked={value === 'true'}
              onChange={() => handleToggleBool(record.key, value)}
              checkedChildren="开启"
              unCheckedChildren="关闭"
            />
          );
        }
        return <Text>{value}</Text>;
      },
    },
    {
      title: '操作',
      key: 'action',
      width: 160,
      render: (_: unknown, record: ManagedSettingEntry) => (
        <Space>
          <Button size="small" icon={<SettingOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.key)}>
            <Button size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Title level={4} style={{ margin: 0 }}>
          <SafetyOutlined /> Agent 管理设置
        </Title>
        <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
          新增设置
        </Button>
      </div>

      <Card>
        <Table
          columns={columns}
          dataSource={settings}
          rowKey="key"
          loading={loading}
          pagination={false}
          size="middle"
        />
      </Card>

      <Card title="权限规则 (permission.rule.N)" size="small">
        <Text type="secondary">
          权限规则格式: <code>ALLOW|DENY action(resource:selector)</code>
          <br />
          示例: <code>DENY live_trading(*:leverage&gt;100)</code>
          <br />
          添加规则: 新增设置项，key 填 <code>permission.rule.1</code>, <code>permission.rule.2</code> 等
        </Text>
      </Card>

      <Modal
        title={isNew ? '新增 Managed 设置' : `编辑: ${SETTING_LABELS[editKey] || editKey}`}
        open={modalOpen}
        onOk={handleSave}
        onCancel={() => setModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <Space direction="vertical" className="w-full" size="middle">
          <div>
            <Text type="secondary">设置项 Key</Text>
            <Input
              value={editKey}
              onChange={(e) => setEditKey(e.target.value)}
              disabled={!isNew}
              placeholder="例如: allowed_models, disable_live_trading, permission.rule.1"
            />
          </div>
          <div>
            <Text type="secondary">值</Text>
            {BOOL_KEYS.has(editKey) ? (
              <Select
                value={editValue}
                onChange={setEditValue}
                className="w-full"
                options={[
                  { value: 'true', label: 'true' },
                  { value: 'false', label: 'false' },
                ]}
              />
            ) : (
              <TextArea
                value={editValue}
                onChange={(e) => setEditValue(e.target.value)}
                rows={3}
                placeholder="例如: claude-sonnet-5,deepseek-v4"
              />
            )}
          </div>
        </Space>
      </Modal>
    </div>
  );
}
