import { useState, useEffect, useCallback } from 'react';
import { Table, Button, Tag, Space, Modal, Input, Popconfirm, message, Tabs, Form, Select } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, EyeOutlined, WarningOutlined, StopOutlined, UndoOutlined, FileProtectOutlined, CheckCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { STRATEGY_ACTIONS_ARCHIVE_CONFIRM_KEY, STRATEGY_ACTIONS_CODE_KEY, STRATEGY_ACTIONS_DISABLE_CONFIRM_KEY, STRATEGY_ACTIONS_ENABLE_KEY, STRATEGY_ACTIONS_FLAG_KEY, STRATEGY_ACTIONS_PUBLISH_KEY, STRATEGY_ACTIONS_UNFLAG_KEY, STRATEGY_ACTIONS_UNPUBLISH_KEY, STRATEGY_ALL_ALL_ACTIVE_KEY, STRATEGY_ALL_ARCHIVED_KEY, STRATEGY_ALL_DISABLED_KEY, STRATEGY_ALL_FLAGGED_KEY, STRATEGY_ALL_FLAG_FILTER_KEY, STRATEGY_ALL_SEARCH_PLACEHOLDER_KEY, STRATEGY_ALL_TOTAL_KEY, STRATEGY_COLUMNS_ACTIONS_KEY, STRATEGY_COLUMNS_CODE_KEY, STRATEGY_COLUMNS_DESCRIPTION_KEY, STRATEGY_COLUMNS_FLAG_KEY, STRATEGY_COLUMNS_NAME_KEY, STRATEGY_COLUMNS_NO_KEY, STRATEGY_COLUMNS_OWNER_KEY, STRATEGY_COLUMNS_PRESET_KEY, STRATEGY_COLUMNS_PUBLIC_KEY, STRATEGY_COLUMNS_SCHEDULES_KEY, STRATEGY_COLUMNS_STATUS_KEY, STRATEGY_COLUMNS_SYSTEM_KEY, STRATEGY_COLUMNS_TAGS_KEY, STRATEGY_COLUMNS_TAGS_PLACEHOLDER_KEY, STRATEGY_COLUMNS_TYPE_KEY, STRATEGY_COLUMNS_USER_KEY, STRATEGY_COLUMNS_USES_KEY, STRATEGY_COLUMNS_YES_KEY, STRATEGY_MESSAGES_ARCHIVE_FAILED_KEY, STRATEGY_MESSAGES_ARCHIVE_SUCCESS_KEY, STRATEGY_MESSAGES_DELETE_FAILED_KEY, STRATEGY_MESSAGES_DISABLE_FAILED_KEY, STRATEGY_MESSAGES_DISABLE_SUCCESS_KEY, STRATEGY_MESSAGES_ENABLE_FAILED_KEY, STRATEGY_MESSAGES_ENABLE_SUCCESS_KEY, STRATEGY_MESSAGES_FLAG_FAILED_KEY, STRATEGY_MESSAGES_FLAG_SUCCESS_KEY, STRATEGY_MESSAGES_LOAD_PRESET_FAILED_KEY, STRATEGY_MESSAGES_LOAD_STRATEGIES_FAILED_KEY, STRATEGY_MESSAGES_PRESET_CREATED_KEY, STRATEGY_MESSAGES_PRESET_DELETED_KEY, STRATEGY_MESSAGES_PRESET_UPDATED_KEY, STRATEGY_MESSAGES_PUBLISH_FAILED_KEY, STRATEGY_MESSAGES_PUBLISH_SUCCESS_KEY, STRATEGY_MESSAGES_SAVE_FAILED_KEY, STRATEGY_MESSAGES_UNFLAG_FAILED_KEY, STRATEGY_MESSAGES_UNFLAG_SUCCESS_KEY, STRATEGY_MESSAGES_UNPUBLISH_FAILED_KEY, STRATEGY_MESSAGES_UNPUBLISH_SUCCESS_KEY, STRATEGY_PRESET_ADD_KEY, STRATEGY_PRESET_CREATE_KEY, STRATEGY_PRESET_DELETE_CONFIRM_KEY, STRATEGY_PRESET_EDIT_KEY, STRATEGY_TABS_ALL_STRATEGIES_KEY, STRATEGY_TABS_PRESET_KEY } from '@/gen/ant/v1/i18n/admin_keys';

;
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

  // ── All strategies oversight ──
  const fetchAll = useCallback(async (page: number, search: string, flag: string) => {
    setAllLoading(true);
    try {
      const resp = await adminStrategyApi.listAllStrategies({
        page, pageSize, search: search || undefined, flag: flag || undefined,
      });
      setAllStrategies(resp.strategies || []);
      setAllTotal(resp.total || 0);
    } catch { message.error(t(STRATEGY_MESSAGES_LOAD_STRATEGIES_FAILED_KEY)); }
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
    } catch { message.error(t(STRATEGY_MESSAGES_LOAD_STRATEGIES_FAILED_KEY)); }
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
      t(STRATEGY_MESSAGES_FLAG_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_FLAG_FAILED_KEY),
      flagTarget,
    );
    setFlagModalOpen(false);
    setFlagReason('');
  };

  const handleUnflag = (id: string) => {
    runAction(
      () => adminStrategyApi.unflagStrategy(id),
      t(STRATEGY_MESSAGES_UNFLAG_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_UNFLAG_FAILED_KEY),
      id,
    );
  };

  const handleUnpublish = (id: string) => {
    runAction(
      () => adminStrategyApi.unpublishStrategy(id),
      t(STRATEGY_MESSAGES_UNPUBLISH_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_UNPUBLISH_FAILED_KEY),
      id,
    );
  };

  const handlePublish = (id: string) => {
    runAction(
      () => adminStrategyApi.publishStrategy(id),
      t(STRATEGY_MESSAGES_PUBLISH_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_PUBLISH_FAILED_KEY),
      id,
    );
  };

  const handleDisable = (id: string) => {
    runAction(
      () => adminStrategyApi.disableStrategy(id),
      t(STRATEGY_MESSAGES_DISABLE_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_DISABLE_FAILED_KEY),
      id,
    );
  };

  const handleEnable = (id: string) => {
    runAction(
      () => adminStrategyApi.enableStrategy(id),
      t(STRATEGY_MESSAGES_ENABLE_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_ENABLE_FAILED_KEY),
      id,
    );
  };

  const handleArchive = (id: string) => {
    runAction(
      () => adminStrategyApi.archiveStrategy(id),
      t(STRATEGY_MESSAGES_ARCHIVE_SUCCESS_KEY),
      t(STRATEGY_MESSAGES_ARCHIVE_FAILED_KEY),
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

  const allColumns = [
    { title: t(STRATEGY_COLUMNS_NAME_KEY), dataIndex: 'name', key: 'name', width: 180 },
    { title: t(STRATEGY_COLUMNS_OWNER_KEY), key: 'owner', width: 150, render: (_: unknown, r: StrategySummary) => r.userEmail || r.userId || (r.isSystem ? t(STRATEGY_COLUMNS_SYSTEM_KEY) : '—') },
    { title: t(STRATEGY_COLUMNS_TRADING_TYPE_KEY), key: 'type', width: 80, render: (_: unknown, r: StrategySummary) => r.isSystem ? <Tag color="gold">{t(STRATEGY_COLUMNS_PRESET_KEY)}</Tag> : <Tag>{t(STRATEGY_COLUMNS_USER_KEY)}</Tag> },
    { title: t(STRATEGY_COLUMNS_STATUS_KEY), dataIndex: 'status', key: 'status', width: 90 },
    { title: t(STRATEGY_COLUMNS_PUBLIC_KEY), key: 'public', width: 80, render: (_: unknown, r: StrategySummary) => r.isPublic ? <Tag color="blue">{t(STRATEGY_COLUMNS_YES_KEY)}</Tag> : <Tag>{t(STRATEGY_COLUMNS_NO_KEY)}</Tag> },
    {
      title: t(STRATEGY_COLUMNS_FLAG_KEY), dataIndex: 'flag', key: 'flag', width: 120,
      render: (f: string, r: StrategySummary) => f ? <Tag color={flagColor(f)}>{f}{r.flagReason ? `: ${r.flagReason}` : ''}</Tag> : <span style={{ color: 'var(--color-text-secondary)' }}>—</span>,
    },
    { title: t(STRATEGY_COLUMNS_SCHEDULES_KEY), dataIndex: 'scheduleCount', key: 'scheduleCount', width: 90 },
    { title: t(STRATEGY_COLUMNS_USES_KEY), dataIndex: 'useCount', key: 'useCount', width: 70 },
    {
      title: t(STRATEGY_COLUMNS_ACTIONS_KEY), key: 'actions', width: 240, fixed: 'right' as const,
      render: (_: unknown, r: StrategySummary) => (
        <Space size="small">
          <Button size="small" icon={<EyeOutlined />} onClick={() => handleViewCode(r.id)} loading={actionLoading === r.id}>
            {t(STRATEGY_ACTIONS_CODE_KEY)}
          </Button>
          {r.flag !== 'archived' && (
            <>
              {r.flag === 'flagged' ? (
                <Button size="small" icon={<CheckCircleOutlined />}
                  loading={actionLoading === r.id}
                  onClick={() => handleUnflag(r.id)}>
                  {t(STRATEGY_ACTIONS_UNFLAG_KEY)}
                </Button>
              ) : (
                <Button size="small" icon={<WarningOutlined />}
                  loading={actionLoading === r.id}
                  onClick={() => { setFlagTarget(r.id); setFlagModalOpen(true); }}>
                  {t(STRATEGY_ACTIONS_FLAG_KEY)}
                </Button>
              )}
              {r.flag !== 'disabled' ? (
                <>
                  {!r.isSystem && r.isPublic ? (
                    <Button size="small" loading={actionLoading === r.id}
                      onClick={() => handleUnpublish(r.id)}>
                      {t(STRATEGY_ACTIONS_UNPUBLISH_KEY)}
                    </Button>
                  ) : !r.isSystem && !r.isPublic ? (
                    <Button size="small" loading={actionLoading === r.id}
                      onClick={() => handlePublish(r.id)}>
                      {t(STRATEGY_ACTIONS_PUBLISH_KEY)}
                    </Button>
                  ) : null}
                  {!r.isSystem && (
                    <Popconfirm title={t(STRATEGY_ACTIONS_DISABLE_CONFIRM_KEY)} onConfirm={() => handleDisable(r.id)}>
                      <Button size="small" danger icon={<StopOutlined />} loading={actionLoading === r.id} />
                    </Popconfirm>
                  )}
                </>
              ) : (
                <>
                  <Button size="small" icon={<UndoOutlined />} loading={actionLoading === r.id}
                    onClick={() => handleEnable(r.id)}>
                    {t(STRATEGY_ACTIONS_ENABLE_KEY)}
                  </Button>
                  <Popconfirm title={t(STRATEGY_ACTIONS_ARCHIVE_CONFIRM_KEY)} onConfirm={() => handleArchive(r.id)}>
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
          children: (
            <div>
              <Space style={{ marginBottom: 12 }}>
                <Input.Search placeholder={t(STRATEGY_ALL_SEARCH_PLACEHOLDER_KEY)} allowClear style={{ width: 240 }}
                  value={allSearch} onChange={e => { setAllSearch(e.target.value); setAllPage(1); }} />
                <Select allowClear placeholder={t(STRATEGY_ALL_FLAG_FILTER_KEY)} style={{ width: 140 }} value={flagFilter || undefined}
                  onChange={v => { setFlagFilter(v || ''); setAllPage(1); }}
                  options={[
                    { value: '', label: t(STRATEGY_ALL_ALL_ACTIVE_KEY) },
                    { value: 'flagged', label: t(STRATEGY_ALL_FLAGGED_KEY) },
                    { value: 'disabled', label: t(STRATEGY_ALL_DISABLED_KEY) },
                    { value: 'archived', label: t(STRATEGY_ALL_ARCHIVED_KEY) },
                  ]} />
              </Space>
              <Table rowKey="id" columns={allColumns} dataSource={allStrategies} loading={allLoading} size="small"
                pagination={{ current: allPage, pageSize, total: allTotal, onChange: (p) => setAllPage(p), showSizeChanger: false, showTotal: (cnt) => t(STRATEGY_ALL_TOTAL_KEY, { count: cnt }) }}
                scroll={{ x: 1200 }} />
            </div>
          ),
        },
      ]} />

      {/* Preset edit/create modal */}
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

      {/* Code view modal */}
      <Modal title={t(STRATEGY_ACTIONS_CODE_KEY)} open={codeViewOpen} onCancel={() => setCodeViewOpen(false)} footer={null} width={700}>
        <pre style={{ background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 6, fontSize: 12, maxHeight: 500, overflow: 'auto' }}>{viewingCode}</pre>
      </Modal>

      {/* Flag reason modal */}
      <Modal title={t(STRATEGY_ACTIONS_FLAG_KEY)} open={flagModalOpen}
        onCancel={() => { setFlagModalOpen(false); setFlagReason(''); }}
        onOk={handleFlag}
        confirmLoading={actionLoading === flagTarget}>
        <TextArea rows={3} placeholder={t(STRATEGY_COLUMNS_FLAG_KEY) + '...'} value={flagReason} onChange={e => setFlagReason(e.target.value)} />
      </Modal>
    </div>
  );
}
