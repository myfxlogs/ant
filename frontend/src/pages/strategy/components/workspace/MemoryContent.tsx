import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Input, Space, Tag, Popconfirm, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ExperimentOutlined, BookOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  MEMORY_SAVE_SUCCESS_KEY, MEMORY_SAVE_FAILED_KEY, MEMORY_NAME_KEY,
  MEMORY_CONTENT_KEY, MEMORY_CONFIRM_DELETE_KEY, MEMORY_CATEGORY_KEY,
  MEMORY_TEMPLATES_KEY, MEMORY_ADD_KEY, MEMORY_EMPTY_KEY,
  MEMORY_EXPERIENCES_KEY, MEMORY_ADD_TEMPLATE_KEY, MEMORY_SAVE_KEY,
  MEMORY_NAME_PLACEHOLDER_KEY, MEMORY_CONTENT_PLACEHOLDER_KEY,
} from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import {
  listMemory,
  saveUserTemplate,
  deleteUserTemplate,
  deleteAgentExperience,
} from '@/client/agentMemory';
import type { UserTemplateEntry, ExperienceEntry } from '@/gen/ant/v1/agent_gateway_pb';

export default function MemoryContent() {
  const { t } = useTranslation();
  const [templates, setTemplates] = useState<UserTemplateEntry[]>([]);
  const [experiences, setExperiences] = useState<ExperienceEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [modalOpen, setModalOpen] = useState(false);
  const [tplName, setTplName] = useState('');
  const [tplContent, setTplContent] = useState('');

  const fetchMemory = useCallback(async () => {
    setLoading(true);
    try {
      const mem = await listMemory();
      setTemplates(mem.templates || []);
      setExperiences(mem.experiences || []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchMemory(); }, [fetchMemory]);

  const handleSaveTemplate = async () => {
    if (!tplName.trim() || !tplContent.trim()) return;
    try {
      const ok = await saveUserTemplate(tplName.trim(), tplContent.trim(), '{}');
      if (ok) {
        message.success(t(MEMORY_SAVE_SUCCESS_KEY));
        setModalOpen(false);
        setTplName('');
        setTplContent('');
        fetchMemory();
      } else {
        message.error(t(MEMORY_SAVE_FAILED_KEY));
      }
    } catch {
      message.error(t(MEMORY_SAVE_FAILED_KEY));
    }
  };

  const handleDeleteTemplate = async (id: string) => {
    try { await deleteUserTemplate(id); fetchMemory(); } catch { /* ignore */ }
  };

  const handleDeleteExperience = async (id: string) => {
    try { await deleteAgentExperience(id); fetchMemory(); } catch { /* ignore */ }
  };

  const tplColumns = [
    { title: t(MEMORY_NAME_KEY), dataIndex: 'name', key: 'name', width: 200 },
    { title: t(MEMORY_CONTENT_KEY), dataIndex: 'content', key: 'content', ellipsis: true },
    {
      title: '', key: 'action', width: 80,
      render: (_: unknown, record: UserTemplateEntry) => (
        <Popconfirm title={t(MEMORY_CONFIRM_DELETE_KEY)} onConfirm={() => handleDeleteTemplate(record.id)}>
          <Button size="small" type="text" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  const expColumns = [
    { title: t(MEMORY_CATEGORY_KEY), dataIndex: 'category', key: 'category', width: 160, render: (cat: string) => <Tag>{cat}</Tag> },
    { title: t(MEMORY_CONTENT_KEY), dataIndex: 'content', key: 'content', ellipsis: true },
    {
      title: '', key: 'action', width: 80,
      render: (_: unknown, record: ExperienceEntry) => (
        <Popconfirm title={t(MEMORY_CONFIRM_DELETE_KEY)} onConfirm={() => handleDeleteExperience(record.id)}>
          <Button size="small" type="text" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card
        size="small"
        title={<Space><BookOutlined /><span>{t(MEMORY_TEMPLATES_KEY)}</span></Space>}
        extra={<Button size="small" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>{t(MEMORY_ADD_KEY)}</Button>}
      >
        <Table dataSource={templates} columns={tplColumns} rowKey="id" size="small" loading={loading} pagination={false} locale={{ emptyText: t(MEMORY_EMPTY_KEY) }} />
      </Card>

      <Card
        size="small"
        title={<Space><ExperimentOutlined /><span>{t(MEMORY_EXPERIENCES_KEY)}</span></Space>}
      >
        <Table dataSource={experiences} columns={expColumns} rowKey="id" size="small" loading={loading} pagination={false} locale={{ emptyText: t(MEMORY_EMPTY_KEY) }} />
      </Card>

      <Modal title={t(MEMORY_ADD_TEMPLATE_KEY)} open={modalOpen} onOk={handleSaveTemplate} onCancel={() => setModalOpen(false)} okText={t(MEMORY_SAVE_KEY)}>
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input placeholder={t(MEMORY_NAME_PLACEHOLDER_KEY)} value={tplName} onChange={(e) => setTplName(e.target.value)} />
          <Input.TextArea rows={4} placeholder={t(MEMORY_CONTENT_PLACEHOLDER_KEY)} value={tplContent} onChange={(e) => setTplContent(e.target.value)} />
        </Space>
      </Modal>
    </div>
  );
}
