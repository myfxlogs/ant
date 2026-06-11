import { Button, Descriptions, Space, Tag, Typography, Popconfirm, message } from 'antd';
import { EditOutlined, DeleteOutlined, CodeOutlined, SendOutlined, ExportOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import type { StrategyTemplate } from '@/client/strategy';
import { formatDateTime } from '@/utils/date';
import { copyToClipboard } from '@/utils/clipboard';

const { Text, Paragraph } = Typography;

interface Props {
  template: StrategyTemplate;
  scheduleCount: number;
  publishing: boolean;
  onEdit: (tpl: StrategyTemplate) => void;
  onDelete: (id: string) => void;
  onPublish: (id: string) => void;
  onUnpublish: (id: string) => void;
  onViewCode: (code: string) => void;
  onRunBacktest: (tpl: StrategyTemplate) => void;
  onOpenCreateSchedule: () => void;
}

export default function LibraryOverviewTab({
  template, scheduleCount, publishing,
  onEdit, onDelete, onPublish, onUnpublish, onViewCode, onRunBacktest, onOpenCreateSchedule,
}: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const id = String(template.id || '');
  const tags = Array.isArray((template as any).tags) ? (template as any).tags : [];
  const isSystem = Boolean((template as any).isSystem) || tags.includes('preset') || id.startsWith('default-');
  const isPublic = Boolean((template as any).isPublic);
  const code = String((template as any).code || '');

  const handleCopyCode = async () => {
    const ok = await copyToClipboard(code);
    if (ok) message.success(t('strategy.templates.messages.codeCopied'));
    else message.error(t('strategy.templates.messages.copyFailed'));
  };

  return (
    <div style={{ padding: '16px 0' }}>
      <Descriptions column={1} size="small" bordered style={{ marginBottom: 16 }}>
        <Descriptions.Item label={t('strategy.templates.table.name')}>
          <Text strong>{template.name}</Text>
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.description')}>
          {template.description || '-'}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.visibility')}>
          {isPublic
            ? <Tag color="blue">{t('strategy.templates.visibility.public')}</Tag>
            : <Tag>{t('strategy.templates.visibility.private')}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.library.publishStatus', '发布状态')}>
          {isSystem
            ? <Tag color="gold">{t('strategy.templates.badges.preset', '预设')}</Tag>
            : isPublic
              ? <Tag color="green">{t('strategy.library.published', '已发布')}</Tag>
              : <Tag>{t('strategy.library.draft', '草稿')}</Tag>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.library.scheduleCount', '调度')}>
          {scheduleCount > 0
            ? <Text style={{ color: '#1677ff' }}>{t('strategy.library.scheduleRunningCount', { count: scheduleCount }, '{{count}} 个运行中')}</Text>
            : <Text type="secondary">{t('strategy.library.noSchedules', '无')}</Text>}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.useCount')}>
          {String((template as any).useCount || 0)}
        </Descriptions.Item>
        <Descriptions.Item label={t('strategy.templates.table.createdAt')}>
          {formatDateTime(String(template.createdAt || ''))}
        </Descriptions.Item>
      </Descriptions>

      {/* Action buttons */}
      <Space wrap>
        {!isSystem && (
          <>
            <Button icon={<EditOutlined />} onClick={() => onEdit(template)}>
              {t('common.edit')}
            </Button>
            {isPublic ? (
              <Button onClick={() => onUnpublish(id)} loading={publishing}>
                {t('strategy.library.unpublish', '下架')}
              </Button>
            ) : (
              <Button type="primary" icon={<SendOutlined />} onClick={() => onPublish(id)} loading={publishing}>
                {t('strategy.library.publish', '发布到市场')}
              </Button>
            )}
            <Popconfirm
              title={t('strategy.templates.deleteConfirm', '确定删除?')}
              onConfirm={() => onDelete(id)}
              okText={String(t('common.yes'))} cancelText={String(t('common.no'))}
            >
              <Button danger icon={<DeleteOutlined />}>{t('common.delete')}</Button>
            </Popconfirm>
          </>
        )}
        {code && (
          <Button icon={<CodeOutlined />} onClick={() => onViewCode(code)}>
            {t('strategy.templates.actions.viewCode', '查看代码')}
          </Button>
        )}
        <Button onClick={() => onRunBacktest(template)}>
          {t('strategy.templates.actions.backtest', '回测')}
        </Button>
        <Button onClick={onOpenCreateSchedule}>
          {t('strategy.library.createSchedule', '创建调度')}
        </Button>
        <Button icon={<ExportOutlined />}
          onClick={() => navigate(`/strategy/workspace?templateId=${id}`)}>
          {t('strategy.library.openInWorkspace', '在Workspace中打开')}
        </Button>
      </Space>

      {/* Code preview (first 20 lines) */}
      {code && (
        <div style={{ marginTop: 16 }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8 }}>
            <Text strong>{t('strategy.library.codePreview', '代码预览')}</Text>
            <Button size="small" onClick={handleCopyCode}>{t('strategy.templates.actions.copyCode', '复制')}</Button>
          </div>
          <pre style={{
            background: '#1e1e1e', color: '#d4d4d4', padding: 12, borderRadius: 6,
            fontSize: 12, maxHeight: 240, overflow: 'auto', margin: 0,
          }}>
            {code.split('\n').slice(0, 20).join('\n')}
            {code.split('\n').length > 20 && '\n...'}
          </pre>
        </div>
      )}
    </div>
  );
}
