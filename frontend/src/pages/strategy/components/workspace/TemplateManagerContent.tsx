import { lazy, Suspense } from 'react';
import { Input, Segmented, List, Tag, Typography, Button, Popconfirm, Form } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, LoadingOutlined, BankOutlined, GlobalOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useLibraryTemplates } from '../../hooks/useLibraryTemplates';
import { isSystemTemplate, isPublicTemplate } from '../../hooks/libraryTypes';
import PublishToMarketModal from '../PublishToMarketModal';
import type { StrategyTemplate } from '@/client/strategy';
import {
  MY_STRATEGIES_KEY, CREATE_KEY as LIBRARY_CREATE_KEY, FILTER_MINE_KEY, FILTER_SYSTEM_KEY,
  SEARCH_PLACEHOLDER_KEY as LIBRARY_SEARCH_PLACEHOLDER_KEY, SYSTEM_KEY, SHARED_KEY,
  PRIVATE_KEY, UNPUBLISH_SHORT_KEY, SHARE_KEY,
} from '@/gen/ant/v1/i18n/strategy_library_keys';
import { EMPTY_KEY as ASSET_EMPTY_KEY } from '@/gen/ant/v1/i18n/strategy_asset_keys';
import { DELETE_CONFIRM_KEY as TEMPLATES_DELETE_CONFIRM_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

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
          <Text strong style={{ fontSize: 13 }}>{t(MY_STRATEGIES_KEY)}</Text>
          <Button type="primary" size="small" icon={<PlusOutlined />} onClick={tpl.openCreate}>
            {t(LIBRARY_CREATE_KEY)}
          </Button>
        </div>
        <Segmented block size="small" value={tpl.filter}
          onChange={v => tpl.setFilter(v as any)}
          options={[
            { value: 'user', label: t(FILTER_MINE_KEY) },
            { value: 'system', label: t(FILTER_SYSTEM_KEY) },
          ]}
          style={{ marginBottom: 8 }}
        />
        <Input size="small" placeholder={t(LIBRARY_SEARCH_PLACEHOLDER_KEY)}
          value={tpl.search} onChange={e => tpl.setSearch(e.target.value)} allowClear />
      </div>

      <div style={{ flex: 1, overflowY: 'auto' }}>
        {tpl.error ? (
          <div style={{ padding: 24, textAlign: 'center', color: '#ff4d4f' }}>{tpl.error}</div>
        ) : (
          <List
            loading={tpl.loading ? { spinning: true, indicator: <LoadingOutlined style={{ fontSize: 20 }} /> } : false}
            dataSource={tpl.templates}
            locale={{ emptyText: t(ASSET_EMPTY_KEY) }}
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
                      {system && <Tag color="gold" icon={<BankOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(SYSTEM_KEY)}</Tag>}
                      {!system && public_ && <Tag color="blue" icon={<GlobalOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(SHARED_KEY)}</Tag>}
                      {!system && !public_ && <Tag color="default" icon={<LockOutlined />} style={{ margin: 0, fontSize: 10 }}>{t(PRIVATE_KEY)}</Tag>}
                    </span>
                  </div>
                  {!system && (
                    <div style={{ display: 'flex', justifyContent: 'flex-end', gap: 4 }}>
                      <Button type="text" size="small" icon={<EditOutlined />} onClick={() => openEdit(item)} />
                      {public_ ? (
                        <Button type="text" size="small" onClick={() => tpl.handleUnpublish(id)} loading={tpl.publishing}>
                          <Text style={{ fontSize: 11 }}>{t(UNPUBLISH_SHORT_KEY)}</Text>
                        </Button>
                      ) : (
                        <Button type="text" size="small" onClick={() => tpl.handlePublish(id)} loading={tpl.publishing}>
                          <Text style={{ fontSize: 11 }}>{t(SHARE_KEY)}</Text>
                        </Button>
                      )}
                      <Popconfirm title={t(TEMPLATES_DELETE_CONFIRM_KEY)} onConfirm={() => tpl.handleDelete(id)}>
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
