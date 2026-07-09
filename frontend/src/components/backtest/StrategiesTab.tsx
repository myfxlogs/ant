import { Button, Table, Tag, Space, Tooltip } from 'antd';
import { PlayCircleOutlined, EditOutlined, HistoryOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';
import { COMMON_UNSAVED_KEY, COMMON_SAVED_KEY, COMMON_UPDATED_KEY } from '@/gen/ant/v1/i18n/base_keys';
import { TEMPLATE_SAVE_AS_KEY, TEMPLATE_LOAD_KEY, TEMPLATES_KEY, UNTITLED_DRAFT_KEY, NAME_KEY, RUN_BACKTEST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { HISTORY_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';

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

  const dataSource = [
    ...(hasUnsavedDraft ? [{
      key: '__draft__',
      id: '__draft__',
      name: draftName || t(UNTITLED_DRAFT_KEY),
      isDraft: true,
      status: 'modified',
      updatedAt: undefined,
    }] : []),
    ...templates.map(tpl => ({
      key: tpl.id,
      id: tpl.id,
      name: tpl.name,
      isDraft: false,
      status: tpl.status || 'saved',
      updatedAt: tpl.updatedAt,
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
            : <Tag color="green">{t(COMMON_SAVED_KEY)}</Tag>
          }
        </Space>
      ),
    },
    {
      title: t(COMMON_UPDATED_KEY),
      dataIndex: 'updatedAt',
      key: 'updatedAt',
      width: 160,
      render: (v: any) => v
        ? new Date(Number(v.seconds) * 1000).toLocaleString()
        : '—',
    },
    {
      title: '',
      key: 'actions',
      width: 140,
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
        <Button size="small" onClick={onSaveAs}>
          {t(TEMPLATE_SAVE_AS_KEY)}
        </Button>
      </div>
      <Table
        size="small"
        loading={loading}
        dataSource={dataSource}
        columns={columns}
        pagination={false}
        rowKey="key"
        rowClassName={(record: any) => record.id === selectedId ? 'ant-table-row-selected' : ''}
        onRow={(record: any) => ({
          onClick: () => !record.isDraft && onSelect(record.id),
          style: { cursor: record.isDraft ? 'default' : 'pointer' },
        })}
      />
    </div>
  );
}
