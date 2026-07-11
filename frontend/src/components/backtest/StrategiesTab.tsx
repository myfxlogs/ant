import { lazy, Suspense } from 'react';
import { Button, Table, Tag, Space, Tooltip, Input, Segmented, Popconfirm, Typography } from 'antd';
import { PlayCircleOutlined, EditOutlined, HistoryOutlined, DeleteOutlined, PlusOutlined, BankOutlined, GlobalOutlined, LockOutlined, SearchOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';
import { useLibraryTemplates } from '@/pages/strategy/hooks/useLibraryTemplates';
import { isSystemTemplate, isPublicTemplate } from '@/pages/strategy/hooks/libraryTypes';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_UPDATED_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { TEMPLATE_SAVE_AS_KEY, TEMPLATE_LOAD_KEY, TEMPLATES_KEY, UNTITLED_DRAFT_KEY, NAME_KEY, RUN_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { HISTORY_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import {
  CREATE_KEY as LIBRARY_CREATE_KEY, FILTER_MINE_KEY, FILTER_SYSTEM_KEY,
  SEARCH_PLACEHOLDER_KEY as LIBRARY_SEARCH_PLACEHOLDER_KEY, SYSTEM_KEY, SHARED_KEY,
  PRIVATE_KEY, UNPUBLISH_SHORT_KEY, SHARE_KEY,
} from '@/gen/ant/v1/i18n/strategy_library_keys';
import { EMPTY_KEY as ASSET_EMPTY_KEY } from '@/gen/ant/v1/i18n/strategy_asset_keys';
import { DELETE_CONFIRM_KEY as TEMPLATES_DELETE_CONFIRM_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

const { Text } = Typography;
const PublishToMarketModal = lazy(() => import('@/pages/strategy/components/PublishToMarketModal'));

interface Props {
  templates: StrategyTemplate[];
  loading: boolean;
  selectedId: string;
  hasUnsavedDraft: boolean;
  draftName: string;
  onSelect: (id: string | null) => void;
  onRunBacktest: () => void;
  onOpenHistory: (templateId?: string) => void;
  onSaveAs: () => void;
}

export default function StrategiesTab({
  templates, loading, selectedId, hasUnsavedDraft, draftName,
  onSelect, onRunBacktest, onOpenHistory, onSaveAs,
}: Props) {
  const { t } = useTranslation();
  const tpl = useLibraryTemplates();

  const dataSource = [
    ...(hasUnsavedDraft ? [{
      key: '__draft__',
      id: '__draft__',
      name: draftName || t(UNTITLED_DRAFT_KEY),
      isDraft: true,
      status: 'modified',
      updatedAt: undefined,
      isSystem: false,
      isPublic: false,
    }] : []),
    ...tpl.templates.map(item => ({
      key: item.id,
      id: item.id,
      name: item.name,
      isDraft: false,
      status: item.status || 'saved',
      updatedAt: item.updatedAt,
      isSystem: isSystemTemplate(item),
      isPublic: isPublicTemplate(item),
      raw: item,
    })),
  ];

  const columns = [
    {
      title: t(NAME_KEY),
      dataIndex: 'name',
      key: 'name',
      render: (name: string, record: any) => (
        <Space>
          <span style={{ fontWeight: 600 }}>{name}</span>
          {record.isDraft
            ? <Tag color="orange">{t(COMMON_UNSAVED_KEY)}</Tag>
            : record.isSystem
              ? <Tag color="gold" icon={<BankOutlined />} style={{ fontSize: 10 }}>{t(SYSTEM_KEY)}</Tag>
              : record.isPublic
                ? <Tag color="blue" icon={<GlobalOutlined />} style={{ fontSize: 10 }}>{t(SHARED_KEY)}</Tag>
                : <Tag color="default" icon={<LockOutlined />} style={{ fontSize: 10 }}>{t(PRIVATE_KEY)}</Tag>
          }
        </Space>
      ),
    },
    {
      title: t(COMMON_UPDATED_KEY),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 140,
      render: (v: any) => v
        ? new Date(Number(v.seconds) * 1000).toLocaleString()
        : '—',
    },
    {
      title: '',
      key: 'actions',
      width: 200,
      render: (_: any, record: any) => (
        <Space size={0}>
          {!record.isDraft && (
            <Tooltip title={t(TEMPLATE_LOAD_KEY)}>
              <Button size="small" type="text" icon={<EditOutlined />}
                onClick={(e) => { e.stopPropagation(); onSelect(record.id); }} />
            </Tooltip>
          )}
          {!record.isDraft && (
            <Tooltip title={t(RUN_BACKTEST_KEY)}>
              <Button size="small" type="text" icon={<PlayCircleOutlined />}
                onClick={(e) => { e.stopPropagation(); onSelect(record.id); onRunBacktest(); }}
                style={{ color: '#3fb950' }} />
            </Tooltip>
          )}
          {!record.isDraft && (
            <Tooltip title={t(HISTORY_KEY)}>
              <Button size="small" type="text" icon={<HistoryOutlined />}
                onClick={(e) => { e.stopPropagation(); onOpenHistory(record.id); }} />
            </Tooltip>
          )}
          {!record.isDraft && !record.isSystem && (
            record.isPublic ? (
              <Button size="small" type="text" onClick={(e) => { e.stopPropagation(); tpl.handleUnpublish(record.id); }} loading={tpl.publishing}>
                <Text style={{ fontSize: 11 }}>{t(UNPUBLISH_SHORT_KEY)}</Text>
              </Button>
            ) : (
              <Button size="small" type="text" onClick={(e) => { e.stopPropagation(); tpl.handlePublish(record.id); }} loading={tpl.publishing}>
                <Text style={{ fontSize: 11 }}>{t(SHARE_KEY)}</Text>
              </Button>
            )
          )}
          {!record.isDraft && !record.isSystem && (
            <Popconfirm title={t(TEMPLATES_DELETE_CONFIRM_KEY)} onConfirm={() => tpl.handleDelete(record.id)}>
              <Button size="small" type="text" danger icon={<DeleteOutlined />}
                onClick={(e) => e.stopPropagation()} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
        <span style={{ fontSize: 12, fontWeight: 700, color: '#595959' }}>
          {t(TEMPLATES_KEY)} ({dataSource.length})
        </span>
        <Space size={4}>
          <Button size="small" onClick={onSaveAs}>
            {t(TEMPLATE_SAVE_AS_KEY)}
          </Button>
          <Button size="small" type="primary" icon={<PlusOutlined />} onClick={tpl.openCreate}>
            {t(LIBRARY_CREATE_KEY)}
          </Button>
        </Space>
      </div>
      <Segmented block size="small" value={tpl.filter}
        onChange={v => tpl.setFilter(v as any)}
        options={[
          { value: 'user', label: t(FILTER_MINE_KEY) },
          { value: 'system', label: t(FILTER_SYSTEM_KEY) },
        ]}
        style={{ marginBottom: 8 }}
      />
      <Input size="small" placeholder={t(LIBRARY_SEARCH_PLACEHOLDER_KEY)} prefix={<SearchOutlined />}
        value={tpl.search} onChange={e => tpl.setSearch(e.target.value)} allowClear style={{ marginBottom: 8 }} />
      <Table
        size="small"
        loading={tpl.loading || loading}
        dataSource={dataSource}
        columns={columns}
        pagination={false}
        rowKey="key"
        locale={{ emptyText: t(ASSET_EMPTY_KEY) }}
        rowClassName={(record: any) => record.id === selectedId ? 'ant-table-row-selected' : ''}
        onRow={(record: any) => ({
          onClick: () => !record.isDraft && onSelect(record.id),
          style: { cursor: record.isDraft ? 'default' : 'pointer' },
        })}
      />
      <Suspense fallback={null}>
        <PublishToMarketModal
          open={tpl.publishModalOpen}
          template={tpl.publishingTemplate}
          onClose={tpl.closePublishModal}
          onPublished={() => { tpl.fetchTemplates(); }}
        />
      </Suspense>
    </div>
  );
}
