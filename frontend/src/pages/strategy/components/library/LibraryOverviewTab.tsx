import { Button, Descriptions, Space, Tag, Typography, Popconfirm, message } from 'antd';
import { EditOutlined, DeleteOutlined, CodeOutlined, ExportOutlined, BankOutlined, GlobalOutlined, LockOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { ACTIONS_COPY_KEY, ACTIONS_VIEW_CODE_KEY, DELETE_CONFIRM_KEY, MESSAGES_CODE_COPIED_KEY, MESSAGES_COPY_FAILED_KEY, TABLE_CREATED_AT_KEY, TABLE_DESCRIPTION_KEY, TABLE_NAME_KEY, TABLE_USE_COUNT_KEY, TABLE_VISIBILITY_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { CODE_PREVIEW_KEY, CREATE_SCHEDULE_KEY, NO_SCHEDULES_KEY, OPEN_IN_WORKSPACE_KEY, PRIVATE_KEY, SAVE_AS_MINE_KEY, SCHEDULE_COUNT_KEY, SHARED_KEY, PUBLISH_KEY, SYSTEM_KEY, UNPUBLISH_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';
;
import { useNavigate } from 'react-router-dom';
import { formatDateTime } from '@/utils/date';
import { copyToClipboard } from '@/utils/clipboard';
import { useLibraryCtx, useTemplatesCtx, useSchedulesCtx } from '../../LibraryContext';
import { isSystemTemplate, isPublicTemplate } from '../../hooks/libraryTypes';

const { Text } = Typography;

export default function LibraryOverviewTab() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const tplCtx = useTemplatesCtx();
  const schCtx = useSchedulesCtx();
  const lib = useLibraryCtx();
  const tpl = tplCtx.selected;
  if (!tpl) return null;

  const id = String(tpl.id || '');
  const system = isSystemTemplate(tpl);
  const public_ = isPublicTemplate(tpl);
  const code = String(tpl.code || '');
  const count = tplCtx.scheduleCountByTemplate(id);

  const handleCopyCode = async () => {
    const ok = await copyToClipboard(code);
    if (ok) message.success(t(MESSAGES_CODE_COPIED_KEY));
    else message.error(t(MESSAGES_COPY_FAILED_KEY));
  };

  return (
    <div style={{ padding: '16px 0' }}>
      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t(TABLE_NAME_KEY)}><Text strong>{tpl.name}</Text></Descriptions.Item>
        <Descriptions.Item label={t(TABLE_DESCRIPTION_KEY)}>{tpl.description || '-'}</Descriptions.Item>
        <Descriptions.Item label={t(TABLE_VISIBILITY_KEY)}>
          {system ? <Tag color="gold" icon={<BankOutlined />}>{t(SYSTEM_KEY)}</Tag>
            : public_ ? <Tag color="blue" icon={<GlobalOutlined />}>{t(SHARED_KEY)}</Tag>
            : <Tag color="default" icon={<LockOutlined />}>{t(PRIVATE_KEY)}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={t(SCHEDULE_COUNT_KEY)}>
          {count > 0 ? <Text style={{ color: '#1677ff' }}>{t('strategy.library.scheduleRunningCount', '{{count}} 个运行中', { count })}</Text>
            : <Text type="secondary">{t(NO_SCHEDULES_KEY)}</Text>}
        </Descriptions.Item>
        <Descriptions.Item label={t(TABLE_USE_COUNT_KEY)}>{String(tpl.useCount || 0)}</Descriptions.Item>
        <Descriptions.Item label={t(TABLE_CREATED_AT_KEY)}>{formatDateTime(String(tpl.createdAt || ''))}</Descriptions.Item>
      </Descriptions>

      <Space wrap>
        {!system && (
          <>
            <Button icon={<EditOutlined />} onClick={() => tplCtx.openEdit(tpl)}>{t('common.edit')}</Button>
            {public_ ? (
              <Button onClick={() => tplCtx.handleUnpublish(id)} loading={tplCtx.publishing}>{t(UNPUBLISH_KEY)}</Button>
            ) : (
              <Button type="primary" icon={<GlobalOutlined />} onClick={() => tplCtx.handlePublish(id)} loading={tplCtx.publishing}>{t(PUBLISH_KEY)}</Button>
            )}
            <Popconfirm title={t(DELETE_CONFIRM_KEY)} onConfirm={() => tplCtx.handleDelete(id)}>
              <Button danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
            </Popconfirm>
          </>
        )}
        {code && <Button icon={<CodeOutlined />} onClick={() => { lib.setViewingCode(code); lib.setCodeViewOpen(true); }}>{t(ACTIONS_VIEW_CODE_KEY)}</Button>}
        {system ? (
          <Button type="primary" onClick={() => tplCtx.handleSaveAsMine(tpl)}>{t(SAVE_AS_MINE_KEY)}</Button>
        ) : (
          <Button onClick={schCtx.openCreate}>{t(CREATE_SCHEDULE_KEY)}</Button>
        )}
        <Button icon={<ExportOutlined />} onClick={() => navigate(`/strategy/workspace?templateId=${id}`)}>{t(OPEN_IN_WORKSPACE_KEY)}</Button>
      </Space>

      {code && (
        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <Text strong>{t(CODE_PREVIEW_KEY)}</Text>
            <Button size="small" onClick={handleCopyCode}>{t(ACTIONS_COPY_KEY)}</Button>
          </div>
          <pre style={{ background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 6, fontSize: 12, maxHeight: 240, overflow: 'auto', margin: 0 }}>
            {code.split('\n').slice(0, 20).join('\n')}
            {code.split('\n').length > 20 && '\n...'}
          </pre>
        </div>
      )}
    </div>
  );
}
