import { Input, Segmented, List, Tag, Typography, Button, Popconfirm, Badge } from 'antd';
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined, LoadingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';
import type { TemplateFilter } from '../../hooks/useLibraryTemplates';

const { Text } = Typography;

interface Props {
  templates: StrategyTemplate[];
  loading: boolean;
  error: string | null;
  filter: TemplateFilter;
  onFilterChange: (f: TemplateFilter) => void;
  search: string;
  onSearchChange: (s: string) => void;
  selectedId: string;
  onSelect: (id: string) => void;
  onEdit: (tpl: StrategyTemplate) => void;
  onDelete: (id: string) => void;
  onPublish: (id: string) => void;
  onUnpublish: (id: string) => void;
  publishing: boolean;
  scheduleCountByTemplate: (templateId: string) => number;
  onCreate: () => void;
  onOpenInWorkspace?: (tpl: StrategyTemplate) => void;
}

export default function LibraryLeftPanel({
  templates, loading, error, filter, onFilterChange, search, onSearchChange,
  selectedId, onSelect, onEdit, onDelete, onPublish, onUnpublish, publishing,
  scheduleCountByTemplate, onCreate, onOpenInWorkspace,
}: Props) {
  const { t } = useTranslation();

  return (
    <div style={{
      width: 340, minWidth: 340, flexShrink: 0, borderRight: '1px solid #f0f0f0',
      display: 'flex', flexDirection: 'column', height: '100%', background: '#fafbfc',
    }}>
      {/* Header */}
      <div style={{ padding: '12px 14px', borderBottom: '1px solid #f0f0f0' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong style={{ fontSize: 13 }}>{t('strategy.library.myStrategies', '我的策略')}</Text>
          <Button type="primary" size="small" icon={<PlusOutlined />} onClick={onCreate}>
            {t('strategy.library.create', '新建')}
          </Button>
        </div>
        <Segmented
          block size="small"
          value={filter}
          onChange={(v) => onFilterChange(v as TemplateFilter)}
          options={[
            { value: 'all', label: t('strategy.library.filterAll', '全部') },
            { value: 'user', label: t('strategy.library.filterMine', '我的') },
            { value: 'system', label: t('strategy.library.filterSystem', '系统') },
          ]}
          style={{ marginBottom: 8 }}
        />
        <Input
          size="small" prefix={<SearchOutlined />} placeholder={t('strategy.library.searchPlaceholder', '搜索策略...')}
          value={search} onChange={e => onSearchChange(e.target.value)} allowClear
        />
      </div>

      {/* List */}
      <div style={{ flex: 1, overflowY: 'auto' }}>
        {error ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#ff4d4f' }}>{error}</div>
        ) : (
          <List
            loading={loading ? { spinning: true, indicator: <LoadingOutlined style={{ fontSize: 20 }} /> } : false}
            dataSource={templates}
            locale={{ emptyText: t('strategy.library.empty', '暂无策略') }}
            renderItem={(tpl: StrategyTemplate) => {
              const id = String(tpl.id || '');
              const tags = Array.isArray((tpl as any).tags) ? (tpl as any).tags : [];
              const isSystem = Boolean((tpl as any).isSystem) || tags.includes('preset') || id.startsWith('default-');
              const isPublic = Boolean((tpl as any).isPublic);
              const isSelected = selectedId === id;
              const scheduleCount = scheduleCountByTemplate(id);

              return (
                <div
                  key={id}
                  onClick={() => onSelect(id)}
                  role="button" tabIndex={0}
                  onKeyUp={e => e.key === 'Enter' && onSelect(id)}
                  style={{
                    padding: '10px 14px', cursor: 'pointer',
                    background: isSelected ? '#e6f4ff' : 'transparent',
                    borderBottom: '1px solid #f5f5f5',
                    transition: 'background 0.15s',
                  }}
                  onMouseEnter={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = '#f5f5f5'; }}
                  onMouseLeave={e => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = 'transparent'; }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <Text strong style={{ fontSize: 13, maxWidth: 200 }} ellipsis>
                      {String(tpl.name || '')}
                    </Text>
                    <span>
                      {isSystem && <Tag color="gold" style={{ margin: 0, fontSize: 10 }}>{t('strategy.templates.badges.preset', '预设')}</Tag>}
                      {!isSystem && isPublic && <Tag color="blue" style={{ margin: 0, fontSize: 10 }}>{t('strategy.library.published', '已发布')}</Tag>}
                      {!isSystem && !isPublic && <Tag style={{ margin: 0, fontSize: 10 }}>{t('strategy.library.draft', '草稿')}</Tag>}
                    </span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {scheduleCount > 0
                        ? t('strategy.library.scheduleCount', { count: scheduleCount }, '{{count}} 个运行中')
                        : t('strategy.library.noSchedules', '未运行')}
                    </Text>
                    {!isSystem && (
                      <span onClick={e => e.stopPropagation()} style={{ display: 'flex', gap: 4 }}>
                        <Button type="text" size="small" icon={<EditOutlined />}
                          onClick={() => onEdit(tpl)}
                          title={String(t('common.edit'))} />
                        {isPublic ? (
                          <Button type="text" size="small"
                            onClick={() => onUnpublish(id)}
                            loading={publishing}
                            title={String(t('strategy.library.unpublish', '下架'))}>
                            <Text style={{ fontSize: 11 }}>{t('strategy.library.unpublishShort', '下架')}</Text>
                          </Button>
                        ) : (
                          <Button type="text" size="small"
                            onClick={() => onPublish(id)}
                            loading={publishing}
                            title={String(t('strategy.library.publish', '发布'))}>
                            <SendOutlined />
                          </Button>
                        )}
                        <Popconfirm
                          title={t('strategy.templates.deleteConfirm', '确定删除?')}
                          onConfirm={() => onDelete(id)}
                          okText={String(t('common.yes'))} cancelText={String(t('common.no'))}
                        >
                          <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                        </Popconfirm>
                      </span>
                    )}
                  </div>
                </div>
              );
            }}
          />
        )}
      </div>
    </div>
  );
}
