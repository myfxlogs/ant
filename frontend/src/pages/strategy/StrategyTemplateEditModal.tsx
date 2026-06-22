import React, { useState, useCallback } from 'react';
import { Modal, Button, Form, Input, Switch, Row, Col, Typography, Tabs, Space, Tag } from 'antd';
import { CopyOutlined, CodeOutlined, BulbOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CODE_MODAL_ACTIONS_COPY_KEY, CODE_MODAL_TITLE_KEY, EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY, EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY, EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY, EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY, VISIBILITY_PRIVATE_KEY, VISIBILITY_PUBLIC_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { SAVE_BLOCKED_NOT_VALIDATED_KEY, TAB_A_I_KEY, TAB_EXPLAIN_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';

;
import type { FormInstance } from 'antd';
import type { StrategyTemplate } from '@/client/strategy';
import { AICodeReviseChat, CodeExplainPanel } from '@/components/strategy/CodeAssist';

const { TextArea } = Input;
const { Text } = Typography;

export type StrategyTemplateEditModalProps = {
	open: boolean;
	editingTemplate: StrategyTemplate | null;
	form: FormInstance;
	codeValidating: boolean;
	// Code that last passed validation. Save is disabled until the current
	// code in the form matches this value, forcing the user to (re-)run
	// validation after every edit. Required-param values are NOT collected
	// here — they are now collected at backtest/schedule submit time.
	lastValidatedCode?: string;
	onCancel: () => void;
	onValidate: () => void;
	onSubmit: (values: Record<string, unknown>) => void;
};

export const StrategyTemplateEditModal: React.FC<StrategyTemplateEditModalProps> = ({
	open,
	editingTemplate,
	form,
	codeValidating,
	lastValidatedCode = '',
	onCancel,
	onValidate,
	onSubmit,
}) => {
	const { t } = useTranslation();
	const watchedCode = Form.useWatch<string | undefined>('code', form) ?? '';
	const code = watchedCode || (form.getFieldValue('code') as string) || '';

	const [aiTab, setAiTab] = useState<string>('revise');
	const applyAICode = useCallback((newCode: string) => {
		form.setFieldsValue({ code: newCode });
		// After applying revised code, switch to Explain tab so user can review.
		setAiTab('explain');
	}, [form]);

	return (
		<Modal
			title={
				<Space>
					{editingTemplate ? t(EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY) : t(EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY)}
					{code !== lastValidatedCode && code.trim() && (
						<Tag color="warning" style={{ marginLeft: 8 }}>
							{t(SAVE_BLOCKED_NOT_VALIDATED_KEY, { defaultValue: 'Not validated' })}
						</Tag>
					)}
				</Space>
			}
			open={open}
			onCancel={onCancel}
			footer={[
				<Button key="cancel" onClick={onCancel} disabled={codeValidating}>
					{t('common.cancel')}
				</Button>,
				<Button key="validate" onClick={onValidate} loading={codeValidating} icon={<ThunderboltOutlined />}>
					{t(EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY)}
				</Button>,
				<Button
					key="save"
					type="primary"
					onClick={() => form.submit()}
					loading={codeValidating}
					disabled={!code.trim() || code !== lastValidatedCode}
				>
					{t('common.save')}
				</Button>,
			]}
			width={1280}
			styles={{ body: { padding: '16px 20px' } }}
		>
			<Form
				form={form}
				layout="vertical"
				onFinish={onSubmit}
				initialValues={{ isPublic: false }}
				onValuesChange={undefined}
			>
				{/* ── Step 1: Metadata header ── */}
				<div style={{
					background: 'var(--color-bg-secondary)', borderRadius: 10,
					padding: '12px 16px', marginBottom: 14,
				}}>
					<Row gutter={16} align="bottom">
						<Col span={7}>
							<Form.Item
								name="name"
								label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY)}</Text>}
								rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY) }]}
								style={{ marginBottom: 0 }}
							>
								<Input placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY)} />
							</Form.Item>
						</Col>
						<Col span={11}>
							<Form.Item name="description" label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY)}</Text>} style={{ marginBottom: 0 }}>
								<TextArea rows={1} placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY)} autoSize={{ minRows: 1, maxRows: 2 }} />
							</Form.Item>
						</Col>
						<Col span={6}>
							<Form.Item name="isPublic" label={<Text strong>{t(EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY)}</Text>} valuePropName="checked" style={{ marginBottom: 0 }}>
								<Switch checkedChildren={t(VISIBILITY_PUBLIC_KEY)} unCheckedChildren={t(VISIBILITY_PRIVATE_KEY)} />
							</Form.Item>
						</Col>
					</Row>
				</div>

				{/* ── Step 2+3: Code editor + AI assistant (workflow panels) ── */}
				<Row gutter={16}>
					{/* Left: Code editor — the main workspace */}
					<Col span={15}>
						<div style={{
							border: '1px solid var(--color-border)',
							borderRadius: 10, overflow: 'hidden',
						}}>
							<div style={{
								padding: '6px 14px', background: 'var(--color-bg-tertiary)',
								borderBottom: '1px solid var(--color-border)',
								display: 'flex', alignItems: 'center', gap: 6,
							}}>
								<CodeOutlined />
								<Text strong style={{ fontSize: 13 }}>{t(EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY)}</Text>
								{code.trim() && (
									<Tag style={{ marginLeft: 'auto' }}>{code.split('\n').length} lines</Tag>
								)}
							</div>
							<Form.Item
								name="code"
								rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY) }]}
								style={{ marginBottom: 0 }}
							>
								<TextArea
									rows={18}
									placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY)}
									style={{
										fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace',
										fontSize: 13, lineHeight: 1.6,
										border: 'none', borderRadius: 0, resize: 'none',
										height: 420,
									}}
								/>
							</Form.Item>
						</div>
					</Col>

					{/* Right: AI assistant — Revise & Explain in tabs */}
					<Col span={9}>
						<div style={{
							border: '1px solid var(--color-border)',
							borderRadius: 10, overflow: 'hidden',
							height: 466, display: 'flex', flexDirection: 'column',
						}}>
							<Tabs
								activeKey={aiTab}
								onChange={setAiTab}
								size="small"
								style={{ flex: 'none', padding: '0 12px' }}
								tabBarStyle={{ marginBottom: 0 }}
								items={[
									{
										key: 'revise',
										label: <span><BulbOutlined /> {t(TAB_A_I_KEY, { defaultValue: 'AI Revise' })}</span>,
										children: (
											<div style={{ height: 416, overflow: 'auto', padding: '8px 4px' }}>
												{!code.trim() ? (
													<Text type="secondary" style={{ display: 'block', textAlign: 'center', padding: 40 }}>
														{t('strategy.ai.reviseHint', { defaultValue: 'Write code first, then ask AI to improve it.' })}
													</Text>
												) : (
													<AICodeReviseChat code={code} onApply={applyAICode} />
												)}
											</div>
										),
									},
									{
										key: 'explain',
										label: <span><CodeOutlined /> {t(TAB_EXPLAIN_KEY, { defaultValue: 'Explain' })}</span>,
										children: (
											<div style={{ height: 416, overflow: 'auto', padding: '8px 4px' }}>
												{!code.trim() ? (
													<Text type="secondary" style={{ display: 'block', textAlign: 'center', padding: 40 }}>
														{t('strategy.ai.explainHint', { defaultValue: 'Write code to see AI explanation.' })}
													</Text>
												) : (
													<CodeExplainPanel code={code} />
												)}
											</div>
										),
									},
								]}
							/>
						</div>
					</Col>
				</Row>
			</Form>
		</Modal>
	);
};

export type StrategyTemplateCodeViewModalProps = {
	open: boolean;
	code: string;
	onClose: () => void;
	onCopy: (code: string) => void;
};

export const StrategyTemplateCodeViewModal: React.FC<StrategyTemplateCodeViewModalProps> = ({ open, code, onClose, onCopy }) => {
	const { t } = useTranslation();
	return (
		<Modal
			title={t(CODE_MODAL_TITLE_KEY)}
			open={open}
			onCancel={onClose}
			footer={[
				<Button key="copy" icon={<CopyOutlined />} onClick={() => onCopy(code)}>
					{t(CODE_MODAL_ACTIONS_COPY_KEY)}
				</Button>,
				<Button key="close" onClick={onClose}>
					{t('common.close')}
				</Button>,
			]}
			width={860}
		>
			<pre style={{
				background: '#f5f5f5',
				padding: 16,
				borderRadius: 8,
				maxHeight: 360,
				overflow: 'auto',
				fontFamily: 'monospace',
				fontSize: 13,
			}}>
				{code}
			</pre>
			<CodeExplainPanel code={code} />
		</Modal>
	);
};
