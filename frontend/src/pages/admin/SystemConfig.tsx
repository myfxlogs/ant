import { useEffect, useState } from 'react';
import { Card, Table, Button, Modal, Form, Input, message, Space, Switch, Tag, Select, Alert } from 'antd';
import { IconEdit } from '@tabler/icons-react';
import { adminApi, type SystemConfig as AdminConfigType } from '@/client/admin';
import { formatDateTime } from '@/utils/date';
import { getErrorMessage } from '@/utils/error';

export default function SystemConfigPage() {
  const [configs, setConfigs] = useState<AdminConfigType[]>([]);
  const [loading, setLoading] = useState(true);
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
    try {
      const result = await adminApi.listConfigs();
      setConfigs(
        (result || []).filter((c: any) =>
          c?.key === 'max_accounts_per_user' ||
          c?.key === 'ai.provider_catalog' ||
          c?.key === 'econ.translation.ai_config' ||
          c?.key === 'strategy.schedule.health_grading_config'
        ),
      );
    } catch (error) {
      message.error(getErrorMessage(error, '加载配置失败'));
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

  const handleSave = async (values: any) => {
    if (!currentConfig) return;
    try {
			if (isAIProviderCatalog) {
				const raw = (values.value || '').trim();
				if (!raw) {
					message.error('JSON 不能为空');
					return;
				}
				try {
					JSON.parse(raw);
				} catch {
					message.error('JSON 格式不正确');
					return;
				}
			} else if (isStrategyHealthConfig) {
				const raw = (values.value || '').trim();
				if (!raw) {
					message.error('JSON 不能为空');
					return;
				}
				let parsed: any;
				try {
					parsed = JSON.parse(raw);
				} catch {
					message.error('JSON 格式不正确');
					return;
				}
				const greenSuccessRate = Number(parsed?.green_success_rate);
				const yellowSuccessRate = Number(parsed?.yellow_success_rate);
				const greenMaxFailedRuns = Number(parsed?.green_max_failed_runs);
				const minSampleSize = Number(parsed?.min_sample_size);
				if (!Number.isFinite(greenSuccessRate) || greenSuccessRate < 0 || greenSuccessRate > 100) {
					message.error('green_success_rate 必须在 0~100 之间');
					return;
				}
				if (!Number.isFinite(yellowSuccessRate) || yellowSuccessRate < 0 || yellowSuccessRate > 100) {
					message.error('yellow_success_rate 必须在 0~100 之间');
					return;
				}
				if (yellowSuccessRate > greenSuccessRate) {
					message.error('yellow_success_rate 不能大于 green_success_rate');
					return;
				}
				if (!Number.isFinite(greenMaxFailedRuns) || greenMaxFailedRuns < 0) {
					message.error('green_max_failed_runs 必须大于等于 0');
					return;
				}
				if (!Number.isFinite(minSampleSize) || minSampleSize < 0) {
					message.error('min_sample_size 必须大于等于 0');
					return;
				}
			} else if (isEconAIConfig) {
				const provider = (values.provider || 'zhipu').toString().trim();
				const apiKey = (values.api_key || '').toString().trim();
				const model = (values.model || '').toString().trim();
				const baseURL = (values.base_url || '').toString().trim();
				const enabled = values.enabled !== false;
				if (!apiKey) {
					message.error('API Key 不能为空');
					return;
				}
				if (!model) {
					message.error('模型名称不能为空');
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
      message.success('配置已更新');
      setEditModalVisible(false);
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, '更新失败'));
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
			message.error('JSON 格式不正确');
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
      message.success(enabled ? '配置已启用' : '配置已禁用');
      fetchConfigs();
    } catch (error) {
      message.error(getErrorMessage(error, '操作失败'));
    }
  };

  const keyLabelMap: Record<string, string> = {
    'max_accounts_per_user': '每用户最大账户数',
    'ai.provider_catalog': 'AI 模型提供商目录',
    'econ.translation.ai_config': '经济日历翻译模型配置',
    'strategy.schedule.health_grading_config': '策略健康分级阈值配置',
  };

  const columns = [
    {
      title: '配置项',
      dataIndex: 'key',
      key: 'key',
      width: 200,
      render: (text: string) => (
				<span className="font-medium">{keyLabelMap[text] || text}</span>
			),
    },
    {
      title: '值',
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
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 100,
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '开关',
      key: 'toggle',
      width: 80,
      render: (_: unknown, record: AdminConfigType) => (
        <Switch
          checked={record.enabled}
          onChange={(checked) => handleToggleEnabled(record.key, checked)}
          checkedChildren="开"
          unCheckedChildren="关"
        />
      ),
    },
    {
      title: '更新时间',
      dataIndex: 'updated_at',
      key: 'updated_at',
      width: 180,
      render: (_text: any, record: AdminConfigType) => formatDateTime(record.updated_at),
    },
    {
      title: '操作',
      key: 'action',
      width: 100,
      render: (_: unknown, record: AdminConfigType) => (
        <Button
          type="link"
          size="small"
          icon={<IconEdit size={14} />}
          onClick={() => handleEdit(record)}
        >
          编辑
        </Button>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold" style={{ color: '#141D22' }}>系统配置</h1>

      <Card>
        <Table
          scroll={{ x: "max-content" }}
          columns={columns}
          dataSource={configs}
          rowKey="key"
          loading={loading}
          pagination={false}
        />
      </Card>

      <Modal
        title={`编辑配置: ${currentConfig?.key || ''}`}
        open={editModalVisible}
        onCancel={() => setEditModalVisible(false)}
        footer={null}
      >
        <Form form={form} onFinish={handleSave} layout="vertical">
				{(isAIProviderCatalog || isStrategyHealthConfig) && (
					<Form.Item name="value" label="值" rules={[{ required: true }]}> 
						<Input.TextArea
							placeholder="请输入 JSON"
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
            message="阈值字段说明"
            description="green_success_rate: 绿色成功率阈值；green_max_failed_runs: 绿色最大失败次数；yellow_success_rate: 黄色成功率阈值；min_sample_size: 最小样本数。"
          />
        )}
				{isEconAIConfig && (
					<>
						<Form.Item name="provider" label="提供商" rules={[{ required: true }]}> 
							<Select
								options={[
									{ label: '智谱 (Zhipu)', value: 'zhipu' },
									{ label: 'DeepSeek', value: 'deepseek' },
									{ label: '自定义 / OpenAI 兼容', value: 'custom' },
								]}
							/>
						</Form.Item>
						<Form.Item name="api_key" label="API Key" rules={[{ required: true }]}> 
							<Input.Password placeholder="请输入 API Key" />
						</Form.Item>
						<Form.Item name="model" label="模型名称" rules={[{ required: true }]}> 
							<Input placeholder="例如 glm-4-flash / deepseek-chat / gpt-4o-mini" />
						</Form.Item>
						<Form.Item name="base_url" label="Base URL（可选，仅自定义/OpenAI 兼容）"> 
							<Input placeholder="例如 https://api.openai.com 或自建网关" />
						</Form.Item>
						<Form.Item name="enabled" label="是否启用" valuePropName="checked"> 
							<Switch />
						</Form.Item>
					</>
				)}
				{!isAIProviderCatalog && !isEconAIConfig && !isStrategyHealthConfig && (
					<Form.Item name="value" label="值" rules={[{ required: true }]}> 
						<Input placeholder="请输入配置值" />
					</Form.Item>
				)}
          <Form.Item name="description" label="描述">
            <Input.TextArea placeholder="请输入描述" rows={3} />
          </Form.Item>
          <Form.Item>
            <Space>
						{isAIProviderCatalog && (
							<Button onClick={handleFormatJson}>格式化 JSON</Button>
						)}
            {isStrategyHealthConfig && (
              <>
                <Button onClick={handleUseStrategyHealthTemplate}>填充示例</Button>
                <Button onClick={handleFormatJson}>格式化 JSON</Button>
              </>
            )}
              <Button type="primary" htmlType="submit">保存</Button>
              <Button onClick={() => setEditModalVisible(false)}>取消</Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
