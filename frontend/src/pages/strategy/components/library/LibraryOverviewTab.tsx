import { Button, Descriptions, Space, Tag, Typography, Popconfirm, message } from 'antd';
import { EditOutlined, DeleteOutlined, CodeOutlined, SendOutlined, ExportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { formatDateTime } from '@/utils/date';
import { copyToClipboard } from '@/utils/clipboard';
import { useLibraryCtx } from '../../LibraryContext';
import { isSystemTemplate, isPublicTemplate } from '../../hooks/libraryTypes';

const { Text } = Typography;

export default function LibraryOverviewTab() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const lib = useLibraryCtx();
  const tpl = lib.selected;
  if (!tpl) return null;

  const id = String(tpl.id || '');
  const system = isSystemTemplate(tpl);
  const public_ = isPublicTemplate(tpl);
  const code = String((tpl as any).code || '');
  const count = lib.scheduleCountByTemplate(id);

  const handleCopyCode = async () => {
    const ok = await copyToClipboard(code);
    if (ok) message.success(t('strategy.templates.messages.codeCopied'));
    else message.error(t('strategy.templates.messages.copyFailed'));
  };

  return (
    <div style={{ padding: '16px 0' }}>
      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('strategy.templates.table.name')}><Text strong>{tpl.name}</Text></Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.description')}>{tpl.description || '-'}</Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.visibility')}>
          {public_ ? <Tag color="blue">{t('strategy.templates.visibility.public')}</Tag> : <Tag>{t('strategy.templates.visibility.private')}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.library.publishStatus')}>
          {system ? <Tag color="gold">{t('strategy.templates.badges.preset')}</Tag>
            : public_ ? <Tag color="green">{t('strategy.library.published')}</Tag>
            : <Tag>{t('strategy.library.draft')}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.library.scheduleCount')}>
          {count > 0 ? <Text style={{ color: '#1677ff' }}>{t('strategy.library.scheduleRunningCount', '{{count}} 个运行中', { count })}</Text>
            : <Text type="secondary">{t('strategy.library.noSchedules')}</Text>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.useCount')}>{String((tpl as any).useCount || 0)}</Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.createdAt')}>{formatDateTime(String(tpl.createdAt || ''))}</Descriptions.Item>
      </Descriptions>

      <Space wrap>
        {!system && (
          <>
            <Button icon={<EditOutlined />} onClick={() => lib.openEdit(tpl)}>{t('common.edit')}</Button>
            {public_ ? (
              <Button onClick={() => lib.handleUnpublish(id)} loading={lib.publishing}>{t('strategy.library.unpublish')}</Button>
            ) : (
              <Button type="primary" icon={<SendOutlined />} onClick={() => lib.handlePublish(id)} loading={lib.publishing}>{t('strategy.library.publish')}</Button>
            )}
            <Popconfirm title={t('strategy.templates.deleteConfirm')} onConfirm={() => lib.handleDelete(id)}>
              <Button danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
            </Popconfirm>
          </>
        )}
        {code && <Button icon={<CodeOutlined />} onClick={() => { lib.setViewingCode(code); lib.setCodeViewOpen(true); }}>{t('strategy.templates.actions.viewCode')}</Button>}
        <Button onClick={lib.scheduleProps.openCreate}>{t('strategy.library.createSchedule')}</Button>
        <Button icon={<ExportOutlined />} onClick={() => navigate(`/strategy/workspace?templateId=${id}`)}>{t('strategy.library.openInWorkspace')}</Button>
      </Space>

      {code && (
        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <Text strong>{t('strategy.library.codePreview')}</Text>
            <Button size="small" onClick={handleCopyCode}>{t('strategy.templates.actions.copyCode')}</Button>
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
