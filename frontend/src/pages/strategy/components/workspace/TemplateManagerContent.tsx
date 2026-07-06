import { lazy, Suspense } from 'react';
import { Input, Segmented, List, Tag, Typography, Button, Popconfirm, Form } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, LoadingOutlined, BankOutlined, GlobalOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useLibraryTemplates } from '../../hooks/useLibraryTemplates';
import { isSystemTemplate, isPublicTemplate } from '../../hooks/libraryTypes';
import PublishToMarketModal from '../PublishToMarketModal';
import type { StrategyTemplate } from '@/client/strategy';

const { Text } = Typography;
const StrategyTemplateEditModal = lazy(() => import('../../StrategyTemplateEditModal').then(m => ({ default: m.StrategyTemplateEditModal })));

export default function TemplateManagerContent() {
  const { t } = useTranslation();
  const tpl = useLibraryTemplates();
  const [editForm] = Form.useForm();

  const openEdit = (template: StrategyTemplate) => {
    tpl.openEdit(template);
    editForm.setFieldsValue({ name: template.name, description: template.description, code: (template as any).code, isPublic: (template as any).isPublic });
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <div style={{ padding: '8px 0', borderBottom: '1px solid #f0f0f0', flexShrink: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
          <Text strong style={{ fontSize: 13 }}>{t('strategy.library.myStrategies', 'My Strategies')}</Text>
          <Button type="primary" size="small" icon={<PlusOutlined />} onClick={tpl.openCreate}>
            {t('strategy.library.create', 'Create')}
          </Button>
        </div>
        <Segmented block size="small" value={tpl.filter}
          onChange={v => tpl.setFilter(v as any)}
          options={[
            { value: 'user', label: t('strategy.library.filterMine', 'Mine') },
            { value: 'system', label: t('strategy.library.filterSystem', 'System') },
          ]}
          style={{ marginBottom: 8 }}
        />
        <Input size="small" placeholder={t('strategy.library.searchPlaceholder', 'Search...')}
          value={tpl.search} onChange={e => tpl.setSearch(e.target.value)} allowClear />
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {tpl.error ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#ff4d4f' }}>{tpl.error}</div>
        ) : (
          <List
            loading={tpl.loading ? { spinning: true, indicator: <LoadingOutlined style={{ fontSize: 20 }} /> } : false}
            dataSource={tpl.templates}
            locale={{ emptyText: t('strategy.asset.empty', 'No data') }}
            renderItem={(item: StrategyTemplate) => {
              const id = String(item.id || '');
              const system = isSystemTemplate(item);
              const public_ = isPublicTemplate(item);
              return (
                <div key={id}
                  style={{ padding: '10px 14px', borderBottom: '1px solid #f5f5f5' }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 4 }}>
                    <Text strong style={{ fontSize: 13, maxWidth: 200 }} ellipsis>{String(item.name || '')}</Text>
                    <span>
                      {system && <Tag color="gold" icon={<BankOutlined />} style={{ margin: 0, fontSize: 10 }}>{t('strategy.library.system', 'System')}</Tag>}
                      {!system && public_ && <Tag color="blue" icon={<GlobalOutlined />} style={{ margin: 0, fontSize: 10 }}>{t('strategy.library.shared', 'Shared')}</Tag>}
                      {!system && !public_ && <Tag color="default" icon={<LockOutlined />} style={{ margin: 0, fontSize: 10 }}>{t('strategy.library.private', 'Private')}</Tag>}
                    </span>
                  </div>
                  {!system && (
                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 4 }}>
                      <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(item)} />
                      {public_ ? (
                        <Button type="text" size="small" onClick={() => tpl.handleUnpublish(id)} loading={tpl.publishing}>
                          <Text style={{ fontSize: 11 }}>{t('strategy.library.unpublishShort', 'Unpublish')}</Text>
                        </Button>
                      ) : (
                        <Button type="text" size="small" onClick={() => tpl.handlePublish(id)} loading={tpl.publishing}>
                          <Text style={{ fontSize: 11 }}>{t('strategy.library.share', 'Share')}</Text>
                        </Button>
                      )}
                      <Popconfirm title={t('strategy.templates.deleteConfirm', 'Delete?')} onConfirm={() => tpl.handleDelete(id)}>
                        <Button type="text" size="small" danger icon={<DeleteOutlined />} />
                      </Popconfirm>
                    </div>
                  )}
                </div>
              );
            }}
          />
        )}
      </div>

      <Suspense fallback={null}>
        {tpl.editOpen && (
          <StrategyTemplateEditModal open={tpl.editOpen} editingTemplate={tpl.editing} form={editForm}
            codeValidating={tpl.codeValidating} lastValidatedCode={tpl.lastValidatedCode}
            validationResult={tpl.validationResult}
            onCancel={() => { tpl.setEditOpen(false); editForm.resetFields(); }}
            onValidate={tpl.handleValidate}
            onClearValidation={() => { tpl.setValidationResult(null); tpl.setLastValidatedCode(''); }}
            onSubmit={async (values: Record<string, unknown>) => { await tpl.handleSave(values); editForm.resetFields(); }} />
        )}
      </Suspense>

      <PublishToMarketModal
        open={tpl.publishModalOpen}
        template={tpl.publishingTemplate}
        onClose={tpl.closePublishModal}
        onPublished={() => { tpl.fetchTemplates(); }}
      />
    </div>
  );
}
