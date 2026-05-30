import { useEffect, useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, message, Space, Switch, Tag, Select, Alert } from 'antd';
import { EditOutlined } from '@ant-design/icons';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';
import { useTranslation } from 'react-i18next';
import { StatusResult } from '@/components/common/StatusResult';

export default function SystemConfigPage() {
  const { t } = useTranslation();
  const [configs, setConfigs] = useState<AdminConfigType[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [currentConfig, setCurrentConfig] = useState<AdminConfigType | null>(null);
  const [form] = Form.useForm();

	const isAIProviderCatalog = currentConfig?.key === 'ai.provider_catalog';
	const isEconAIConfig = currentConfig?.key === 'econ.translation.ai_config';
  const isStrategyHealthConfig = currentConfig?.key === 'strategy.schedule.health_grading_config';

  const strategyHealthConfigTemplate = {
    green_success_rate: 90,
    green_max_failed_runs: 1,
    yellow_success_rate: 60,
    min_sample_size: 1,
  };

  const fetchConfigs = async () => {
    setLoading(true);
    setError(null);
    try {
      const result = await adminApi.listConfigs();
      setConfigs(
        (result || []).filter((c) =>
          c?.key === 'max_accounts_per_user' ||
          c?.key === 'ai.provider_catalog' ||
          c?.key === 'econ.translation.ai_config' ||
          c?.key === 'strategy.schedule.health_grading_config'
        ),
      );
    } catch (err) {
      const msg = getErrorMessage(err, t('admin.config.messages.loadFailed'));
      setError(msg);
      message.error(msg);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchConfigs();
  }, []);

  const handleEdit = (config: AdminConfigType) => {
    setCurrentConfig(config);
    if (config.key === 'econ.translation.ai_config') {
      const raw = (config.value || '').toString().trim();
      let initial: any = {
        provider: 'zhipu',
        api_key: '',
        model: 'glm-4-flash',
        base_url: '',
        enabled: true,
      };
      if (raw) {
        try {
          const parsed = JSON.parse(raw);
          if (parsed && typeof parsed === 'object') {
            initial = { ...initial, ...parsed };
          }
        } catch {
          // ignore parse error, use defaults
        }
      }
      form.setFieldsValue({
        provider: initial.provider,
        api_key: initial.api_key,
        model: initial.model,
        base_url: initial.base_url,
        enabled: initial.enabled,
        description: config.description,
      });
    } else {
      form.setFieldsValue({
        value: config.value,
        description: config.description,
      });
    }
    setEditModalVisible(true);
  };

  const handleSave = async (values: Record<string, unknown>) => {
    if (!currentConfig) return;
    try {
			if (isAIProviderCatalog) {
				const raw = (values.value || '').trim();
				if (!raw) {
					message.error(t('admin.config.validation.jsonEmpty'));
					return;
				}
				try {
					JSON.parse(raw);
				} catch {
					message.error(t('admin.config.validation.jsonInvalid'));
					return;
				}
			} else if (isStrategyHealthConfig) {
				const raw = (values.value || '').trim();
				if (!raw) {
					message.error(t('admin.config.validation.jsonEmpty'));
					return;
				}
				let parsed: unknown;
				try {
					parsed = JSON.parse(raw);
				} catch {
					message.error(t('admin.config.validation.jsonInvalid'));
					return;
				}
				const greenSuccessRate = Number(parsed?.green_success_rate);
				const yellowSuccessRate = Number(parsed?.yellow_success_rate);
				const greenMaxFailedRuns = Number(parsed?.green_max_failed_runs);
				const minSampleSize = Number(parsed?.min_sample_size);
				if (!Number.isFinite(greenSuccessRate) || greenSuccessRate < 0 || greenSuccessRate > 100) {
					message.error(t('admin.config.validation.greenSuccessRateRange'));
					return;
				}
				if (!Number.isFinite(yellowSuccessRate) || yellowSuccessRate < 0 || yellowSuccessRate > 100) {
					message.error(t('admin.config.validation.yellowSuccessRateRange'));
					return;
				}
				if (yellowSuccessRate > greenSuccessRate) {
					message.error(t('admin.config.validation.yellowNotGreaterThanGreen'));
					return;
				}
				if (!Number.isFinite(greenMaxFailedRuns) || greenMaxFailedRuns < 0) {
					message.error(t('admin.config.validation.greenMaxFailedRunsNonNegative'));
					return;
				}
				if (!Number.isFinite(minSampleSize) || minSampleSize < 0) {
					message.error(t('admin.config.validation.minSampleSizeNonNegative'));
					return;
				}
			} else if (isEconAIConfig) {
				const provider = (values.provider || 'zhipu').toString().trim();
				const apiKey = (values.api_key || '').toString().trim();
				const model = (values.model || '').toString().trim();
				const baseURL = (values.base_url || '').toString().trim();
				const enabled = values.enabled !== false;
				if (!apiKey) {
					message.error(t('admin.config.validation.apiKeyRequired'));
					return;
				}
				if (!model) {
					message.error(t('admin.config.validation.modelRequired'));
					return;
				}
				const cfg = {
					provider,
					api_key: apiKey,
					model,
					base_url: baseURL,
					enabled,
				};
				await adminApi.setConfig(currentConfig.key, {
					value: JSON.stringify(cfg),
					description: values.description || currentConfig.description || '',
				});
			} else {
				await adminApi.setConfig(currentConfig.key, values);
			}
      message.success(t('admin.config.messages.updated'));
      setEditModalVisible(false);
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.config.messages.updateFailed')));
    }
  };

  const handleFormatJson = () => {
		if (!currentConfig || (!isAIProviderCatalog && !isStrategyHealthConfig)) return;
		const raw = (form.getFieldValue('value') || '').toString().trim();
		if (!raw) return;
		try {
			const obj = JSON.parse(raw);
			form.setFieldsValue({ value: JSON.stringify(obj, null, 2) });
		} catch {
			message.error(t('admin.config.validation.jsonInvalid'));
		}
	};

  const handleUseStrategyHealthTemplate = () => {
    if (!isStrategyHealthConfig) return;
    form.setFieldsValue({
      value: JSON.stringify(strategyHealthConfigTemplate, null, 2),
    });
  };

  const handleToggleEnabled = async (key: string, enabled: boolean) => {
    try {
      await adminApi.toggleConfigEnabled(key, enabled);
      message.success(enabled ? t('admin.config.messages.enabled') : t('admin.config.messages.disabled'));
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, t('admin.config.messages.operationFailed')));
    }
  };

  const getKeyLabel = (key: string): string => {
    const labelMap: Record<string, string> = {
      'max_accounts_per_user': t('admin.config.maxAccountsPerUser'),
      'ai.provider_catalog': t('admin.config.aiProviderCatalog'),
      'econ.translation.ai_config': t('admin.config.econAIConfig'),
      'strategy.schedule.health_grading_config': t('admin.config.strategyHealthConfig'),
    };
    return labelMap[key] || key;
  };

  const columns = [
    {
      title: t('admin.config.configItem'),
      dataIndex: 'key',
      key: 'key',
      width: 200,
      render: (text: string) => (
				<span className="font-medium">{getKeyLabel(text)}</span>
			),
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
      <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>{t('admin.config.title')}</h1>

      <Card>
        <StatusResult error={error} onRetry={fetchConfigs}>
        <Table
          scroll={{ x: "max-content" }}
          columns={columns}
          dataSource={configs}
          rowKey="key"
          loading={loading}
          pagination={false}
        />
        </StatusResult>
      </Card>

      <Modal
        title={t('admin.config.editConfig', { key: currentConfig?.key || '' })}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleSave} layout="vertical">
				{(isAIProviderCatalog || isStrategyHealthConfig) && (
					<Form.Item name="value" label={t('admin.config.value')} rules={[{ required: true }]}>
						<Input.TextArea
							placeholder={t('admin.config.placeholders.json')}
							rows={10}
							style={{ fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace' }}
						/>
					</Form.Item>
				)}
        {isStrategyHealthConfig && (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            message={t('admin.config.thresholdInfo')}
            description={t('admin.config.thresholdDesc')}
          />
        )}
				{isEconAIConfig && (
					<>
						<Form.Item name="provider" label={t('admin.config.provider')} rules={[{ required: true }]}>
							<Select
								options={[
									{ label: t('admin.config.providerOptions.zhipu'), value: 'zhipu' },
									{ label: t('admin.config.providerOptions.deepseek'), value: 'deepseek' },
									{ label: t('admin.config.providerOptions.custom'), value: 'custom' },
								]}
							/>
						</Form.Item>
						<Form.Item name="api_key" label="API Key" rules={[{ required: true }]}>
							<Input.Password placeholder={t('admin.config.placeholders.apiKey')} />
						</Form.Item>
						<Form.Item name="model" label={t('admin.config.modelName')} rules={[{ required: true }]}>
							<Input placeholder={t('admin.config.placeholders.model')} />
						</Form.Item>
						<Form.Item name="base_url" label={t('admin.config.baseUrlLabel')}>
							<Input placeholder={t('admin.config.placeholders.baseUrl')} />
						</Form.Item>
						<Form.Item name="enabled" label={t('admin.config.enableToggle')} valuePropName="checked">
							<Switch />
						</Form.Item>
					</>
				)}
				{!isAIProviderCatalog && !isEconAIConfig && !isStrategyHealthConfig && (
					<Form.Item name="value" label={t('admin.config.value')} rules={[{ required: true }]}>
						<Input placeholder={t('admin.config.placeholders.configValue')} />
					</Form.Item>
				)}
          <Form.Item name="description" label={t('admin.config.description')}>
            <Input.TextArea placeholder={t('admin.config.placeholders.description')} rows={3} />
          </Form.Item>
          <Form.Item>
            <Space>
						{isAIProviderCatalog && (
							<Button onClick={handleFormatJson}>{t('admin.config.formatJson')}</Button>
						)}
            {isStrategyHealthConfig && (
              <>
                <Button onClick={handleUseStrategyHealthTemplate}>{t('admin.config.fillTemplate')}</Button>
                <Button onClick={handleFormatJson}>{t('admin.config.formatJson')}</Button>
              </>
            )}
              <Button type="primary" htmlType="submit">{t('common.save')}</Button>
              <Button onClick={() => setEditModalVisible(false)}>{t('common.cancel')}</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
