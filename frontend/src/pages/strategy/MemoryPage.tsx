import { useState, useEffect, useCallback } from 'react';
import { Card, Table, Button, Modal, Input, Space, Tag, Typography, Popconfirm, message } from 'antd';
import { PlusOutlined, DeleteOutlined, ExperimentOutlined, BookOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import {
  listMemory,
  saveUserTemplate,
  deleteUserTemplate,
  deleteAgentExperience,
} from '@/client/agentMemory';
import type { UserTemplateEntry, ExperienceEntry } from '@/gen/ant/v1/agent_gateway_pb';

export default function MemoryPage() {
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
        message.success(t('strategy.memory.saveSuccess', 'Saved'));
        setModalOpen(false);
        setTplName('');
        setTplContent('');
        fetchMemory();
      } else {
        message.error(t('strategy.memory.saveFailed', 'Save failed'));
      }
    } catch {
      message.error(t('strategy.memory.saveFailed', 'Save failed'));
    }
  };

  const handleDeleteTemplate = async (id: string) => {
    try {
      await deleteUserTemplate(id);
      fetchMemory();
    } catch { /* ignore */ }
  };

  const handleDeleteExperience = async (id: string) => {
    try {
      await deleteAgentExperience(id);
      fetchMemory();
    } catch { /* ignore */ }
  };

  const tplColumns = [
    {
      title: t('strategy.memory.name', 'Name'),
      dataIndex: 'name',
      key: 'name',
      width: 200,
    },
    {
      title: t('strategy.memory.content', 'Content'),
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: '',
      key: 'action',
      width: 80,
      render: (_: unknown, record: UserTemplateEntry) => (
        <Popconfirm
          title={t('strategy.memory.confirmDelete', 'Delete?')}
          onConfirm={() => handleDeleteTemplate(record.id)}
        >
          <Button size="small" type="text" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  const expColumns = [
    {
      title: t('strategy.memory.category', 'Category'),
      dataIndex: 'category',
      key: 'category',
      width: 160,
      render: (cat: string) => <Tag>{cat}</Tag>,
    },
    {
      title: t('strategy.memory.content', 'Content'),
      dataIndex: 'content',
      key: 'content',
      ellipsis: true,
    },
    {
      title: '',
      key: 'action',
      width: 80,
      render: (_: unknown, record: ExperienceEntry) => (
        <Popconfirm
          title={t('strategy.memory.confirmDelete', 'Delete?')}
          onConfirm={() => handleDeleteExperience(record.id)}
        >
          <Button size="small" type="text" icon={<DeleteOutlined />} danger />
        </Popconfirm>
      ),
    },
  ];

  return (
    <div style={{ padding: 24, maxWidth: 900 }}>
      <Typography.Title level={4}>
        {t('strategy.memory.title', 'Agent Memory')}
      </Typography.Title>

      <Card
        size="small"
        title={
          <Space>
            <BookOutlined />
            <span>{t('strategy.memory.templates', 'User Strategy Templates')}</span>
          </Space>
        }
        extra={
          <Button size="small" icon={<PlusOutlined />} onClick={() => setModalOpen(true)}>
            {t('strategy.memory.add', 'Add')}
          </Button>
        }
        style={{ marginBottom: 16 }}
      >
        <Table
          dataSource={templates}
          columns={tplColumns}
          rowKey="id"
          size="small"
          loading={loading}
          pagination={false}
          locale={{ emptyText: t('strategy.memory.empty', 'No data') }}
        />
      </Card>

      <Card
        size="small"
        title={
          <Space>
            <ExperimentOutlined />
            <span>{t('strategy.memory.experiences', 'Agent Experiences')}</span>
          </Space>
        }
        style={{ marginBottom: 16 }}
      >
        <Table
          dataSource={experiences}
          columns={expColumns}
          rowKey="id"
          size="small"
          loading={loading}
          pagination={false}
          locale={{ emptyText: t('strategy.memory.empty', 'No data') }}
        />
      </Card>

      <Modal
        title={t('strategy.memory.addTemplate', 'Add Strategy Template')}
        open={modalOpen}
        onOk={handleSaveTemplate}
        onCancel={() => setModalOpen(false)}
        okText={t('strategy.memory.save', 'Save')}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input
            placeholder={t('strategy.memory.namePlaceholder', 'Template name')}
            value={tplName}
            onChange={(e) => setTplName(e.target.value)}
          />
          <Input.TextArea
            rows={4}
            placeholder={t('strategy.memory.contentPlaceholder', 'Strategy description, preferences, rules...')}
            value={tplContent}
            onChange={(e) => setTplContent(e.target.value)}
          />
        </Space>
      </Modal>
    </div>
  );
}
