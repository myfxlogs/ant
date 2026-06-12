import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Tag, Space, Modal, Input, Popconfirm, message, Tabs, Typography, Form, Select } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, WarningOutlined, StopOutlined, UndoOutlined, FileProtectOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { adminStrategyApi } from '@/client/admin';

const { TextArea } = Input;
const { Text } = Typography;

type SystemStrategy = {
  id: string;
  name: string;
  description: string;
  code: string;
  isActive: boolean;
  useCount: number;
  tags: string[];
  createdAt?: string;
};

type StrategySummary = {
  id: string;
  name: string;
  userId: string;
  userEmail: string;
  status: string;
  isSystem: boolean;
  isPublic: boolean;
  flag: string;
  flagReason: string;
  flaggedBy: string;
  scheduleCount: number;
  useCount: number;
  tags: string[];
  createdAt?: string;
};

export default function StrategyManagement() {
  const { t } = useTranslation();
  const [tab, setTab] = useState('preset');

  // Preset state
  const [presets, setPresets] = useState<SystemStrategy[]>([]);
  const [presetsLoading, setPresetsLoading] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingPreset, setEditingPreset] = useState<SystemStrategy | null>(null);
  const [form] = Form.useForm();

  // All strategies state
  const [allStrategies, setAllStrategies] = useState<StrategySummary[]>([]);
  const [allLoading, setAllLoading] = useState(false);
  const [allTotal, setAllTotal] = useState(0);
  const [allPage, setAllPage] = useState(1);
  const [allSearch, setAllSearch] = useState('');
  const [flagFilter, setFlagFilter] = useState('');
  const [codeViewOpen, setCodeViewOpen] = useState(false);
  const [viewingCode, setViewingCode] = useState('');
  const [flagModalOpen, setFlagModalOpen] = useState(false);
  const [flagTarget, setFlagTarget] = useState<string>('');
  const [flagReason, setFlagReason] = useState('');

  const pageSize = 15;

  // ── Preset CRUD ──
  const fetchPresets = useCallback(async () => {
    setPresetsLoading(true);
    try {
      const resp = await adminStrategyApi.listSystemStrategies();
      setPresets(resp.strategies || []);
    } catch { message.error('Failed to load preset strategies'); }
    finally { setPresetsLoading(false); }
  }, []);

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
    const tags = values.tags ? String(values.tags).split(',').map((t: string) => t.trim()).filter(Boolean) : [];
    try {
      if (editingPreset) {
        await adminStrategyApi.updateSystemStrategy({ id: editingPreset.id, name: values.name, description: values.description, code: values.code, tags });
        message.success('Preset updated');
      } else {
        await adminStrategyApi.createSystemStrategy({ name: values.name, description: values.description, code: values.code, tags });
        message.success('Preset created');
      }
      setEditModalOpen(false);
      fetchPresets();
    } catch { message.error('Save failed'); }
  };

  const handleDeletePreset = async (id: string) => {
    try {
      await adminStrategyApi.deleteSystemStrategy(id);
      message.success('Preset deleted');
      fetchPresets();
    } catch { message.error('Delete failed'); }
  };

  // ── All strategies oversight ──
  const fetchAll = useCallback(async (page: number, search: string, flag: string) => {
    setAllLoading(true);
    try {
      const resp = await adminStrategyApi.listAllStrategies({
        page, pageSize, search: search || undefined, flag: flag || undefined,
      });
      setAllStrategies(resp.strategies || []);
      setAllTotal(resp.total || 0);
    } catch { message.error('Failed to load strategies'); }
    finally { setAllLoading(false); }
  }, []);

  useEffect(() => {
    if (tab === 'all') fetchAll(allPage, allSearch, flagFilter);
  }, [tab, allPage, allSearch, flagFilter, fetchAll]);

  // ── Compliance actions ──
  const handleViewCode = async (id: string) => {
    try {
      const d = await adminStrategyApi.getStrategyDetail(id);
      setViewingCode(d.code || '');
      setCodeViewOpen(true);
    } catch { message.error('Failed to load strategy code'); }
  };

  const handleFlag = async () => {
    if (!flagReason.trim()) return;
    try {
      await adminStrategyApi.flagStrategy(flagTarget, flagReason);
      message.success('Strategy flagged');
      setFlagModalOpen(false);
      setFlagReason('');
      fetchAll(allPage, allSearch, flagFilter);
    } catch { message.error('Flag failed'); }
  };

  const handleUnpublish = async (id: string) => {
    try { await adminStrategyApi.unpublishStrategy(id); message.success('Unpublished'); fetchAll(allPage, allSearch, flagFilter); }
    catch { message.error('Unpublish failed'); }
  };

  const handleDisable = async (id: string) => {
    try { await adminStrategyApi.disableStrategy(id); message.success('Disabled — all schedules stopped'); fetchAll(allPage, allSearch, flagFilter); }
    catch { message.error('Disable failed'); }
  };

  const handleEnable = async (id: string) => {
    try { await adminStrategyApi.enableStrategy(id); message.success('Enabled'); fetchAll(allPage, allSearch, flagFilter); }
    catch { message.error('Enable failed'); }
  };

  const handleArchive = async (id: string) => {
    try { await adminStrategyApi.archiveStrategy(id); message.success('Archived'); fetchAll(allPage, allSearch, flagFilter); }
    catch { message.error('Archive failed'); }
  };

  const flagColor = (f: string) => {
    if (f === 'flagged') return 'orange';
    if (f === 'disabled') return 'red';
    if (f === 'archived') return 'default';
    return undefined;
  };

  // ── Columns ──
  const presetColumns = [
    { title: 'Name', dataIndex: 'name', key: 'name', width: 200 },
    { title: 'Description', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: 'Tags', dataIndex: 'tags', key: 'tags', render: (tags: string[]) => tags?.map(t => <Tag key={t}>{t}</Tag>) },
    { title: 'Uses', dataIndex: 'useCount', key: 'useCount', width: 80 },
    {
      title: 'Actions', key: 'actions', width: 120,
      render: (_: unknown, r: SystemStrategy) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditPreset(r)} />
          <Popconfirm title="Delete this preset?" onConfirm={() => handleDeletePreset(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const allColumns = [
    { title: 'Name', dataIndex: 'name', key: 'name', width: 180 },
    { title: 'Owner', key: 'owner', width: 150, render: (_: unknown, r: StrategySummary) => r.userEmail || r.userId || (r.isSystem ? '— System —' : '—') },
    { title: 'Type', key: 'type', width: 80, render: (_: unknown, r: StrategySummary) => r.isSystem ? <Tag color="gold">Preset</Tag> : <Tag>User</Tag> },
    { title: 'Status', dataIndex: 'status', key: 'status', width: 90 },
    { title: 'Public', key: 'public', width: 80, render: (_: unknown, r: StrategySummary) => r.isPublic ? <Tag color="blue">Yes</Tag> : <Tag>No</Tag> },
    {
      title: 'Flag', dataIndex: 'flag', key: 'flag', width: 100,
      render: (f: string, r: StrategySummary) => f ? <Tag color={flagColor(f)}>{f}{r.flagReason ? `: ${r.flagReason}` : ''}</Tag> : <Text type="secondary">—</Text>,
    },
    { title: 'Schedules', dataIndex: 'scheduleCount', key: 'scheduleCount', width: 90 },
    { title: 'Uses', dataIndex: 'useCount', key: 'useCount', width: 70 },
    {
      title: 'Actions', key: 'actions', width: 200, fixed: 'right' as const,
      render: (_: unknown, r: StrategySummary) => (
        <Space size="small">
          <Button size="small" icon={<EyeOutlined />} onClick={() => handleViewCode(r.id)}>Code</Button>
          {!r.isSystem && r.flag !== 'archived' && (
            <>
              {r.flag !== 'disabled' ? (
                <>
                  <Button size="small" icon={<WarningOutlined />} onClick={() => { setFlagTarget(r.id); setFlagModalOpen(true); }}>Flag</Button>
                  {r.isPublic && <Button size="small" onClick={() => handleUnpublish(r.id)}>Unpublish</Button>}
                  <Popconfirm title="Stop all schedules?" onConfirm={() => handleDisable(r.id)}>
                    <Button size="small" danger icon={<StopOutlined />} />
                  </Popconfirm>
                </>
              ) : (
                <>
                  <Button size="small" icon={<UndoOutlined />} onClick={() => handleEnable(r.id)}>Enable</Button>
                  <Popconfirm title="Archive this strategy?" onConfirm={() => handleArchive(r.id)}>
                    <Button size="small" icon={<FileProtectOutlined />} />
                  </Popconfirm>
                </>
              )}
            </>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: 16 }}>
      <Tabs activeKey={tab} onChange={setTab} items={[
        {
          key: 'preset',
          label: 'Preset Strategies',
          children: (
            <div>
              <div style={{ marginBottom: 12 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePreset}>Add Preset</Button>
              </div>
              <Table rowKey="id" columns={presetColumns} dataSource={presets} loading={presetsLoading} size="small" pagination={false} />
            </div>
          ),
        },
        {
          key: 'all',
          label: 'All Strategies',
          children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Input.Search placeholder="Search by name..." allowClear style={{ width: 240 }}
                  value={allSearch} onChange={e => { setAllSearch(e.target.value); setAllPage(1); }} />
                <Select allowClear placeholder="Flag filter" style={{ width: 140 }} value={flagFilter || undefined}
                  onChange={v => { setFlagFilter(v || ''); setAllPage(1); }}
                  options={[
                    { value: '', label: 'All Active' },
                    { value: 'flagged', label: 'Flagged' },
                    { value: 'disabled', label: 'Disabled' },
                    { value: 'archived', label: 'Archived' },
                  ]} />
              </Space>
              <Table rowKey="id" columns={allColumns} dataSource={allStrategies} loading={allLoading} size="small"
                pagination={{ current: allPage, pageSize, total: allTotal, onChange: (p) => setAllPage(p), showSizeChanger: false, showTotal: (t) => `${t} total` }}
                scroll={{ x: 1100 }} />
            </div>
          ),
        },
      ]} />

      {/* Preset edit/create modal */}
      <Modal title={editingPreset ? 'Edit Preset' : 'Create Preset'} open={editModalOpen}
        onCancel={() => setEditModalOpen(false)} onOk={handleSavePreset} width={700}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="Name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input />
          </Form.Item>
          <Form.Item name="code" label="Code" rules={[{ required: true }]}>
            <TextArea rows={12} style={{ fontFamily: 'monospace', fontSize: 13 }} />
          </Form.Item>
          <Form.Item name="tags" label="Tags (comma-separated)">
            <Input placeholder="trend-following, ma" />
          </Form.Item>
        </Form>
      </Modal>

      {/* Code view modal */}
      <Modal title="Strategy Code" open={codeViewOpen} onCancel={() => setCodeViewOpen(false)} footer={null} width={700}>
        <pre style={{ background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 6, fontSize: 12, maxHeight: 500, overflow: 'auto' }}>{viewingCode}</pre>
      </Modal>

      {/* Flag reason modal */}
      <Modal title="Flag Strategy" open={flagModalOpen}
        onCancel={() => { setFlagModalOpen(false); setFlagReason(''); }}
        onOk={handleFlag}>
        <TextArea rows={3} placeholder="Reason for flagging..." value={flagReason} onChange={e => setFlagReason(e.target.value)} />
      </Modal>
    </div>
  );
}
