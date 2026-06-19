import { Input, Segmented, List, Tag, Typography, Button, Popconfirm } from 'antd';
import { SearchOutlined, PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined, LoadingOutlined, BankOutlined, GlobalOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { DELETE_CONFIRM_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { useLibraryCtx } from '../../LibraryContext';
import { isSystemTemplate, isPublicTemplate } from '../../hooks/libraryTypes';
import type { StrategyTemplate } from '@/client/strategy';

const { Text } = Typography;

export default function LibraryLeftPanel() {
  const { t } = useTranslation();
  const lib = useLibraryCtx();

  return (
    <div style={{
      width: 340, minWidth: 340, flexShrink: 0, borderRight: '1px solid #f0f0f0',
      display: 'flex', flexDirection: 'column', height: '100%', background: '#fafbfc',
    }}>
      <div style={{ padding: '12px 14px', borderBottom: '1px solid #f0f0f0' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong style={{ fontSize: 13 }}>{t(MY_STRATEGIES_KEY)}</Text>
          <Button type="primary" size="small" icon={<PlusOutlined />} onClick={lib.openCreate}>
            {t(CREATE_KEY)}
          </Button>
        </div>
        <Segmented block size="small" value={lib.filter}
          onChange={v => lib.setFilter(v as any)}
          options={[
            { value: 'user', label: t(FILTER_MINE_KEY) },
            { value: 'system', label: t(FILTER_SYSTEM_KEY) },
          ]}
          style={{ marginBottom: 8 }}
        />
        <Input size="small" prefix={<SearchOutlined />} placeholder={t(SEARCH_PLACEHOLDER_KEY)}
          value={lib.search} onChange={e => lib.setSearch(e.target.value)} allowClear />
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {lib.templatesError ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#ff4d4f' }}>{lib.templatesError}</div>
        ) : (
          <List
            loading={lib.templatesLoading ? { spinning: true, indicator: <LoadingOutlined style={{ fontSize: 20 }} /> } : false}
            dataSource={lib.templates}
            locale={{ emptyText: t(EMPTY_KEY) }}
            renderItem={(tpl: StrategyTemplate) => {
              const id = String(tpl.id || '');
              const system = isSystemTemplate(tpl);
              const public_ = isPublicTemplate(tpl);
              const isSelected = lib.selectedId === id;
              const count = lib.scheduleCountByTemplate(id);

              return (
                <div key={id} onClick={() => lib.selectTemplate(id)}
                  role="button" tabIndex={0}
                  onKeyUp={e => e.key === 'Enter' && lib.selectTemplate(id)}
                  style={{
                    padding: '10px 14px', cursor: 'pointer',
                    background: isSelected ? '#e6f4ff' : 'transparent',
                    borderBottom: '1px solid #f5f5f5', transition: 'background 0.15s',
                  }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <Text strong style={{ fontSize: 13, maxWidth: 200 }} ellipsis>{String(tpl.name || '')}</Text>
                    <span>
                      {system && <Tag color="gold" icon={<BankOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(SYSTEM_KEY)}</Tag>}
                      {!system && public_ && <Tag color="blue" icon={<GlobalOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(SHARED_KEY)}</Tag>}
                      {!system && !public_ && <Tag color="default" icon={<LockOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(PRIVATE_KEY)}</Tag>}
                    </span>
                  </div>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <Text type="secondary" style={{ fontSize: 12 }}>
                      {count > 0 ? t('strategy.library.scheduleCount', '{{count}} 个运行中', { count }) : t(NO_SCHEDULES_KEY)}
                    </Text>
                    {!system && (
                      <span onClick={e => e.stopPropagation()} style={{ display: 'flex', gap: 4 }}>
                        <Button type="text" size="small" icon={<EditOutlined />} onClick={() => lib.openEdit(tpl)} />
                        {public_ ? (
                          <Button type="text" size="small" onClick={() => lib.handleUnpublish(id)} loading={lib.publishing}>
                            <Text style={{ fontSize: 11 }}>{t(UNPUBLISH_SHORT_KEY)}</Text>
                          </Button>
                        ) : (
                          <Button type="text" size="small" onClick={() => lib.handlePublish(id)} loading={lib.publishing}>
                            <Text style={{ fontSize: 11 }}>{t(SHARE_KEY)}</Text>
                          </Button>
                        )}
                        <Popconfirm title={t(DELETE_CONFIRM_KEY)} onConfirm={() => lib.handleDelete(id)}>
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
