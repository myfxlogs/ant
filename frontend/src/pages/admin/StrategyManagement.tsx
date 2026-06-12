import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Tag, Space, Modal, Input, Popconfirm, message, Tabs, Form, Select } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, WarningOutlined, StopOutlined, UndoOutlined, FileProtectOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { adminStrategyApi } from '@/client/admin';
import type { SystemStrategy } from '@/gen/ant/v1/admin_strategy_pb';
import type { StrategySummary } from '@/gen/ant/v1/admin_strategy_pb';

const { TextArea } = Input;

export default function StrategyManagement() {
  const { t } = useTranslation();
  const [tab, setTab] = useState('preset');

  // ── Preset state ──
  const [presets, setPresets] = useState<SystemStrategy[]>([]);
  const [presetsLoading, setPresetsLoading] = useState(false);
  const [editModalOpen, setEditModalOpen] = useState(false);
  const [editingPreset, setEditingPreset] = useState<SystemStrategy | null>(null);
  const [presetSaving, setPresetSaving] = useState(false);
  const [form] = Form.useForm();

  // ── All strategies state ──
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
  const [actionLoading, setActionLoading] = useState<string>(''); // id of row being acted on

  const pageSize = 15;

  // ── Preset CRUD ──
  const fetchPresets = useCallback(async () => {
    setPresetsLoading(true);
    try {
      const resp = await adminStrategyApi.listSystemStrategies();
      setPresets(resp.strategies || []);
    } catch { message.error(t('admin.strategy.messages.loadPresetFailed')); }
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
        message.success(t('admin.strategy.messages.presetUpdated'));
      } else {
        await adminStrategyApi.createSystemStrategy({ name: values.name, description: values.description, code: values.code, tags });
        message.success(t('admin.strategy.messages.presetCreated'));
      }
      setEditModalOpen(false);
      fetchPresets();
    } catch { message.error(t('admin.strategy.messages.saveFailed')); }
    finally { setPresetSaving(false); }
  };

  const handleDeletePreset = async (id: string) => {
    try {
      await adminStrategyApi.deleteSystemStrategy(id);
      message.success(t('admin.strategy.messages.presetDeleted'));
      fetchPresets();
    } catch { message.error(t('admin.strategy.messages.deleteFailed')); }
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
    } catch { message.error(t('admin.strategy.messages.loadStrategiesFailed')); }
    finally { setAllLoading(false); }
  }, [t]);

  useEffect(() => {
    if (tab === 'all') fetchAll(allPage, allSearch, flagFilter);
  }, [tab, allPage, allSearch, flagFilter, fetchAll]);

  // ── Compliance actions ──
  const handleViewCode = async (id: string) => {
    try {
      const d = await adminStrategyApi.getStrategyDetail(id);
      setViewingCode(d.code || '');
      setCodeViewOpen(true);
    } catch { message.error(t('admin.strategy.messages.loadStrategiesFailed')); }
  };

  const runAction = async (action: () => Promise<void>, successMsg: string, failMsg: string, id: string) => {
    setActionLoading(id);
    try {
      await action();
      message.success(successMsg);
      fetchAll(allPage, allSearch, flagFilter);
    } catch { message.error(failMsg); }
    finally { setActionLoading(''); }
  };

  const handleFlag = async () => {
    if (!flagReason.trim()) return;
    await runAction(
      () => adminStrategyApi.flagStrategy(flagTarget, flagReason),
      t('admin.strategy.messages.flagSuccess'),
      t('admin.strategy.messages.flagFailed'),
      flagTarget,
    );
    setFlagModalOpen(false);
    setFlagReason('');
  };

  const handleUnflag = (id: string) => {
    runAction(
      () => adminStrategyApi.unflagStrategy(id),
      t('admin.strategy.messages.unflagSuccess'),
      t('admin.strategy.messages.unflagFailed'),
      id,
    );
  };

  const handleUnpublish = (id: string) => {
    runAction(
      () => adminStrategyApi.unpublishStrategy(id),
      t('admin.strategy.messages.unpublishSuccess'),
      t('admin.strategy.messages.unpublishFailed'),
      id,
    );
  };

  const handleDisable = (id: string) => {
    runAction(
      () => adminStrategyApi.disableStrategy(id),
      t('admin.strategy.messages.disableSuccess'),
      t('admin.strategy.messages.disableFailed'),
      id,
    );
  };

  const handleEnable = (id: string) => {
    runAction(
      () => adminStrategyApi.enableStrategy(id),
      t('admin.strategy.messages.enableSuccess'),
      t('admin.strategy.messages.enableFailed'),
      id,
    );
  };

  const handleArchive = (id: string) => {
    runAction(
      () => adminStrategyApi.archiveStrategy(id),
      t('admin.strategy.messages.archiveSuccess'),
      t('admin.strategy.messages.archiveFailed'),
      id,
    );
  };

  const flagColor = (f: string): string | undefined => {
    if (f === 'flagged') return 'orange';
    if (f === 'disabled') return 'red';
    if (f === 'archived') return 'default';
    return undefined;
  };

  // ── Columns ──
  const presetColumns = [
    { title: t('admin.strategy.columns.name'), dataIndex: 'name', key: 'name', width: 200 },
    { title: t('admin.strategy.columns.description'), dataIndex: 'description', key: 'description', ellipsis: true },
    { title: t('admin.strategy.columns.tags'), dataIndex: 'tags', key: 'tags', render: (tags: string[]) => tags?.map(tg => <Tag key={tg}>{tg}</Tag>) },
    { title: t('admin.strategy.columns.uses'), dataIndex: 'useCount', key: 'useCount', width: 80 },
    {
      title: t('admin.strategy.columns.actions'), key: 'actions', width: 120,
      render: (_: unknown, r: SystemStrategy) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEditPreset(r)} />
          <Popconfirm title={t('admin.strategy.preset.deleteConfirm')} onConfirm={() => handleDeletePreset(r.id)}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const allColumns = [
    { title: t('admin.strategy.columns.name'), dataIndex: 'name', key: 'name', width: 180 },
    { title: t('admin.strategy.columns.owner'), key: 'owner', width: 150, render: (_: unknown, r: StrategySummary) => r.userEmail || r.userId || (r.isSystem ? t('admin.strategy.columns.system') : '—') },
    { title: t('admin.strategy.columns.type'), key: 'type', width: 80, render: (_: unknown, r: StrategySummary) => r.isSystem ? <Tag color="gold">{t('admin.strategy.columns.preset')}</Tag> : <Tag>{t('admin.strategy.columns.user')}</Tag> },
    { title: t('admin.strategy.columns.status'), dataIndex: 'status', key: 'status', width: 90 },
    { title: t('admin.strategy.columns.public'), key: 'public', width: 80, render: (_: unknown, r: StrategySummary) => r.isPublic ? <Tag color="blue">{t('admin.strategy.columns.yes')}</Tag> : <Tag>{t('admin.strategy.columns.no')}</Tag> },
    {
      title: t('admin.strategy.columns.flag'), dataIndex: 'flag', key: 'flag', width: 120,
      render: (f: string, r: StrategySummary) => f ? <Tag color={flagColor(f)}>{f}{r.flagReason ? `: ${r.flagReason}` : ''}</Tag> : <span style={{ color: 'var(--color-text-secondary)' }}>—</span>,
    },
    { title: t('admin.strategy.columns.schedules'), dataIndex: 'scheduleCount', key: 'scheduleCount', width: 90 },
    { title: t('admin.strategy.columns.uses'), dataIndex: 'useCount', key: 'useCount', width: 70 },
    {
      title: t('admin.strategy.columns.actions'), key: 'actions', width: 240, fixed: 'right' as const,
      render: (_: unknown, r: StrategySummary) => (
        <Space size="small">
          <Button size="small" icon={<EyeOutlined />} onClick={() => handleViewCode(r.id)} loading={actionLoading === r.id}>
            {t('admin.strategy.actions.code')}
          </Button>
          {r.flag !== 'archived' && (
            <>
              {r.flag === 'flagged' ? (
                <Button size="small" icon={<CheckCircleOutlined />}
                  loading={actionLoading === r.id}
                  onClick={() => handleUnflag(r.id)}>
                  {t('admin.strategy.actions.unflag')}
                </Button>
              ) : (
                <Button size="small" icon={<WarningOutlined />}
                  loading={actionLoading === r.id}
                  onClick={() => { setFlagTarget(r.id); setFlagModalOpen(true); }}>
                  {t('admin.strategy.actions.flag')}
                </Button>
              )}
              {r.flag !== 'disabled' ? (
                <>
                  {r.isPublic && !r.isSystem && (
                    <Button size="small" loading={actionLoading === r.id}
                      onClick={() => handleUnpublish(r.id)}>
                      {t('admin.strategy.actions.unpublish')}
                    </Button>
                  )}
                  {!r.isSystem && (
                    <Popconfirm title={t('admin.strategy.actions.disableConfirm')} onConfirm={() => handleDisable(r.id)}>
                      <Button size="small" danger icon={<StopOutlined />} loading={actionLoading === r.id} />
                    </Popconfirm>
                  )}
                </>
              ) : (
                <>
                  <Button size="small" icon={<UndoOutlined />} loading={actionLoading === r.id}
                    onClick={() => handleEnable(r.id)}>
                    {t('admin.strategy.actions.enable')}
                  </Button>
                  <Popconfirm title={t('admin.strategy.actions.archiveConfirm')} onConfirm={() => handleArchive(r.id)}>
                    <Button size="small" icon={<FileProtectOutlined />} loading={actionLoading === r.id} />
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
          label: t('admin.strategy.tabs.preset'),
          children: (
            <div>
              <div style={{ marginBottom: 12 }}>
                <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePreset}>
                  {t('admin.strategy.preset.add')}
                </Button>
              </div>
              <Table rowKey="id" columns={presetColumns} dataSource={presets} loading={presetsLoading} size="small" pagination={false} />
            </div>
          ),
        },
        {
          key: 'all',
          label: t('admin.strategy.tabs.allStrategies'),
          children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Input.Search placeholder={t('admin.strategy.all.searchPlaceholder')} allowClear style={{ width: 240 }}
                  value={allSearch} onChange={e => { setAllSearch(e.target.value); setAllPage(1); }} />
                <Select allowClear placeholder={t('admin.strategy.all.flagFilter')} style={{ width: 140 }} value={flagFilter || undefined}
                  onChange={v => { setFlagFilter(v || ''); setAllPage(1); }}
                  options={[
                    { value: '', label: t('admin.strategy.all.allActive') },
                    { value: 'flagged', label: t('admin.strategy.all.flagged') },
                    { value: 'disabled', label: t('admin.strategy.all.disabled') },
                    { value: 'archived', label: t('admin.strategy.all.archived') },
                  ]} />
              </Space>
              <Table rowKey="id" columns={allColumns} dataSource={allStrategies} loading={allLoading} size="small"
                pagination={{ current: allPage, pageSize, total: allTotal, onChange: (p) => setAllPage(p), showSizeChanger: false, showTotal: (cnt) => t('admin.strategy.all.total', { count: cnt }) }}
                scroll={{ x: 1200 }} />
            </div>
          ),
        },
      ]} />

      {/* Preset edit/create modal */}
      <Modal title={editingPreset ? t('admin.strategy.preset.edit') : t('admin.strategy.preset.create')} open={editModalOpen}
        onCancel={() => setEditModalOpen(false)} onOk={handleSavePreset} confirmLoading={presetSaving} width={700}>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label={t('admin.strategy.columns.name')} rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="description" label={t('admin.strategy.columns.description')}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label={t('admin.strategy.columns.code')} rules={[{ required: true }]}>
            <TextArea rows={12} style={{ fontFamily: 'monospace', fontSize: 13 }} />
          </Form.Item>
          <Form.Item name="tags" label={t('admin.strategy.columns.tags')}>
            <Input placeholder={t('admin.strategy.columns.tagsPlaceholder')} />
          </Form.Item>
        </Form>
      </Modal>

      {/* Code view modal */}
      <Modal title={t('admin.strategy.actions.code')} open={codeViewOpen} onCancel={() => setCodeViewOpen(false)} footer={null} width={700}>
        <pre style={{ background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 6, fontSize: 12, maxHeight: 500, overflow: 'auto' }}>{viewingCode}</pre>
      </Modal>

      {/* Flag reason modal */}
      <Modal title={t('admin.strategy.actions.flag')} open={flagModalOpen}
        onCancel={() => { setFlagModalOpen(false); setFlagReason(''); }}
        onOk={handleFlag}
        confirmLoading={actionLoading === flagTarget}>
        <TextArea rows={3} placeholder={t('admin.strategy.columns.flag') + '...'} value={flagReason} onChange={e => setFlagReason(e.target.value)} />
      </Modal>
    </div>
  );
}
