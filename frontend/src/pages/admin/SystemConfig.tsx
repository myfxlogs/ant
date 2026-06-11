import { Card, Table, Button, Tag, Switch } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { formatDateTime } from '@/utils/date';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';
import { useSystemConfig } from './useSystemConfig';
import SystemConfigEditModal from './components/SystemConfigEditModal';
import type { SystemConfig as AdminConfigType } from '@/client/admin';

export default function SystemConfigPage() {
  const { t } = useTranslation();
  const {
    configs, loading, error, editModalVisible, currentConfig, form,
    isAIProviderCatalog, isEconAIConfig, isStrategyHealthConfig,
    fetchConfigs, handleEdit, handleSave, handleFormatJson,
    handleUseStrategyHealthTemplate, handleToggleEnabled, getKeyLabel,
    setEditModalVisible,
  } = useSystemConfig();

  const columns = [
    {
      title: t('admin.config.configItem'),
      dataIndex: 'key',
      key: 'key',
      width: 200,
      render: (text: string) => <span className="font-medium">{getKeyLabel(text)}</span>,
    },
    {
      title: t('admin.config.value'),
      dataIndex: 'value',
      key: 'value',
      width: 150,
      ellipsis: true,
      render: (text: string, record: AdminConfigType) => {
        if (
          record.key === 'ai.provider_catalog' ||
          record.key === 'econ.translation.ai_config' ||
          record.key === 'strategy.schedule.health_grading_config'
        ) {
          return <Tag color="processing">JSON</Tag>;
        }
        return text;
      },
    },
    {
      title: t('admin.config.description'),
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: t('admin.config.status'),
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? t('common.enabled') : t('common.disabled')}
        </Tag>
      ),
    },
    {
      title: t('admin.config.toggle'),
      key: 'toggle',
      width: 80,
      render: (_: unknown, record: AdminConfigType) => (
        <Switch
          checked={record.enabled}
          onChange={(checked) => handleToggleEnabled(record.key, checked)}
          checkedChildren={t('admin.config.on')}
          unCheckedChildren={t('admin.config.off')}
        />
      ),
    },
    {
      title: t('admin.config.updatedAt'),
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (_text: unknown, record: AdminConfigType) => formatDateTime(record.updated_at),
    },
    {
      title: t('common.edit'),
      key: 'action',
      width: 100,
      render: (_: unknown, record: AdminConfigType) => (
        <Button
          type="link"
          size="small"
          icon={<EditOutlined size={14} />}
          onClick={() => handleEdit(record)}
        >
          {t('common.edit')}
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold" style={{ color: 'var(--color-text)' }}>{t('admin.config.title')}</h1>
      <Card>
        <StatusResult error={error} onRetry={fetchConfigs}>
          <Table
            scroll={{ x: 'max-content' }}
            columns={columns}
            dataSource={configs}
            rowKey="key"
            loading={loading}
            pagination={false}
          />
        </StatusResult>
      </Card>
      <SystemConfigEditModal
        visible={editModalVisible}
        currentConfig={currentConfig}
        form={form}
        isAIProviderCatalog={isAIProviderCatalog}
        isEconAIConfig={isEconAIConfig}
        isStrategyHealthConfig={isStrategyHealthConfig}
        onSave={handleSave}
        onCancel={() => setEditModalVisible(false)}
        onFormatJson={handleFormatJson}
        onUseTemplate={handleUseStrategyHealthTemplate}
      />
    </div>
  );
}
