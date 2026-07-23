import { useState, useCallback, useMemo } from 'react';
import { Table, Button, Tag, Typography, Space, Popconfirm, Input, Segmented, message } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, PauseCircleOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { strategyApi } from '@/client/strategy';
import { queryKeys } from '@/queries/queryKeys';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import type { StrategyTemplate } from '@/gen/ant/v1/strategy_template_entity_pb';
import type { StrategySchedule } from '@/gen/ant/v1/strategy_schedule_entity_pb';
import Seo from '@/components/common/Seo';

const { Title } = Typography;

type FilterKey = 'all' | 'mine' | 'preset';

export default function StrategyLibraryPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);

  const [search, setSearch] = useState('');
  const [filter, setFilter] = useState<FilterKey>('all');

  const { data: templates = [], isLoading } = useQuery({
    queryKey: queryKeys.templates.list(),
    queryFn: () => strategyApi.listTemplates() as Promise<StrategyTemplate[]>,
  });

  const { data: schedules = [] } = useQuery({
    queryKey: queryKeys.schedules.list(),
    queryFn: () => strategyApi.listSchedules() as Promise<StrategySchedule[]>,
  });

  // Count running schedules per template
  const runningByTemplate = useMemo(() => {
    const map = new Map<string, number>();
    for (const s of schedules) {
      if (s.status === 'ACTIVE') {
        const tid = s.templateId;
        if (tid) map.set(tid, (map.get(tid) || 0) + 1);
      }
    }
    return map;
  }, [schedules]);

  const filtered = useMemo(() => {
    let list = templates;
    if (filter === 'mine') list = list.filter(t => !t.isSystem);
    if (filter === 'preset') list = list.filter(t => t.isSystem);
    if (search) {
      const q = search.toLowerCase();
      list = list.filter(t => (t.name || '').toLowerCase().includes(q));
    }
    return list;
  }, [templates, filter, search]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await strategyApi.deleteTemplate(id);
      message.success(t('common.deleted', { defaultValue: 'Deleted' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.templates.list() });
    } catch {
      message.error(t('common.deleteFailed', { defaultValue: 'Delete failed' }));
    }
  }, [t, queryClient]);

  const handleOpen = useCallback((tpl: StrategyTemplate) => {
    // Navigate to workspace and load this template
    navigate(`/strategy/workspace?template=${tpl.id}`);
  }, [navigate]);

  const handleCreate = useCallback(() => {
    navigate('/strategy/workspace');
  }, [navigate]);

  const runningCount = (id: string) => runningByTemplate.get(id) || 0;

  return (
    <>
      <Seo title={t('strategy.library.title')} path="/strategy/library" />
      <div style={{ padding: '24px 24px 80px', background: 'var(--color-bg-secondary)', minHeight: '100vh' }}>
        <div className="max-w-7xl mx-auto">
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
            <Title level={3} style={{ margin: 0 }}>{t('strategy.library.title')}</Title>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              {t('strategy.library.create')}
            </Button>
          </div>

          <div style={{ display: 'flex', gap: 12, marginBottom: 16 }}>
            <Input
              placeholder={t('strategy.library.searchPlaceholder')}
              allowClear
              value={search}
              onChange={e => setSearch(e.target.value)}
              style={{ width: 280 }}
            />
            <Segmented
              value={filter}
              onChange={v => setFilter(v as FilterKey)}
              options={[
                { value: 'all', label: t('strategy.library.filterAll') },
                { value: 'mine', label: t('strategy.library.filterMine') },
                { value: 'preset', label: t('strategy.library.filterSystem') },
              ]}
            />
          </div>

          <Table<StrategyTemplate>
            rowKey="id"
            dataSource={filtered}
            loading={isLoading}
            pagination={{ pageSize: 20, showSizeChanger: true, showTotal: (t) => `${t} strategies` }}
            size="middle"
            locale={{ emptyText: t('strategy.library.empty') }}
            columns={[
              {
                title: t('strategy.library.overview'),
                dataIndex: 'name',
                key: 'name',
                render: (name: string, row: StrategyTemplate) => (
                  <div>
                    <div style={{ fontWeight: 600, color: 'var(--color-text)', cursor: 'pointer' }}
                      onClick={() => handleOpen(row)}>
                      {name || 'Untitled'}
                    </div>
                    <div style={{ fontSize: 12, color: 'var(--color-text-muted)', marginTop: 2 }}>
                      {row.description?.slice(0, 80) || ''}
                    </div>
                  </div>
                ),
              },
              {
                title: t('strategy.library.publishStatus'),
                key: 'status',
                width: 120,
                render: (_: unknown, row: StrategyTemplate) => {
                  if (row.isSystem) return <Tag color="blue">{t('strategy.library.system')}</Tag>;
                  if (row.isPublic) return <Tag color="green">{t('strategy.library.shared')}</Tag>;
                  return <Tag>{t('strategy.library.private')}</Tag>;
                },
              },
              {
                title: t('strategy.library.schedules'),
                key: 'runs',
                width: 110,
                render: (_: unknown, row: StrategyTemplate) => {
                  const n = runningCount(row.id);
                  return n > 0
                    ? <Tag color="green">{t('strategy.library.scheduleRunningCount', { count: n })}</Tag>
                    : <span style={{ color: 'var(--color-text-muted)', fontSize: 12 }}>{t('strategy.library.noSchedules')}</span>;
                },
              },
              {
                title: t('strategy.library.actions'),
                key: 'actions',
                width: 160,
                render: (_: unknown, row: StrategyTemplate) => (
                  <Space size={4}>
                    <Button size="small" type="primary" icon={<EditOutlined />}
                      onClick={() => handleOpen(row)}>
                      {t('strategy.library.openInWorkspace')}
                    </Button>
                    {!row.isSystem && (
                      <Popconfirm
                        title={t('common.confirmDelete', { defaultValue: 'Confirm delete?' })}
                        onConfirm={() => handleDelete(row.id)}
                        okText={t('common.delete')}
                        cancelText={t('common.cancel')}
                      >
                        <Button size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    )}
                  </Space>
                ),
              },
            ]}
          />
        </div>
      </div>
    </>
  );
}
