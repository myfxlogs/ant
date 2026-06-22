import React, { useState, useCallback } from 'react';
import { Modal, Button, Form, Input, Switch, Row, Col, Typography, Tabs, Space, Tag, Segmented, message, Spin } from 'antd';
import { CopyOutlined, CodeOutlined, BulbOutlined, ThunderboltOutlined, ImportOutlined, RobotOutlined, CheckCircleOutlined, CloseCircleOutlined, WarningOutlined, InfoCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CODE_MODAL_ACTIONS_COPY_KEY, CODE_MODAL_TITLE_KEY, EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY, EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY, EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY, EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY, VISIBILITY_PRIVATE_KEY, VISIBILITY_PUBLIC_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { SAVE_BLOCKED_NOT_VALIDATED_KEY, TAB_A_I_KEY, TAB_EXPLAIN_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';

;
import type { FormInstance } from 'antd';
import type { StrategyTemplate } from '@/client/strategy';
import { AICodeReviseChat, CodeExplainPanel } from '@/components/strategy/CodeAssist';
import { codeAssistClient } from '@/client/connect';

const { TextArea } = Input;
const { Text } = Typography;

export type StrategyTemplateEditModalProps = {
	open: boolean;
	editingTemplate: StrategyTemplate | null;
	form: FormInstance;
	codeValidating: boolean;
		validationResult?: { valid?: boolean; errors?: string[]; warnings?: string[] } | null;
	// Code that last passed validation. Save is disabled until the current
	// code in the form matches this value, forcing the user to (re-)run
	// validation after every edit. Required-param values are NOT collected
	// here — they are now collected at backtest/schedule submit time.
		lastValidatedCode?: string;
	onCancel: () => void;
	onValidate: (code: string) => void;
	onSubmit: (values: Record<string, unknown>) => void;
};

export const StrategyTemplateEditModal: React.FC<StrategyTemplateEditModalProps> = ({
	open,
	editingTemplate,
	form,
	codeValidating,
	lastValidatedCode = '',
		validationResult,
	onCancel,
	onValidate,
	onSubmit,
}) => {
	const { t } = useTranslation();
	const watchedCode = Form.useWatch<string | undefined>('code', form) ?? '';
	const code = watchedCode || (form.getFieldValue('code') as string) || '';

	const [aiTab, setAiTab] = useState<string>('revise');
	const [fixInstruction, setFixInstruction] = useState('');
	const applyAICode = useCallback((newCode: string) => {
		form.setFieldsValue({ code: newCode });
		setAiTab('explain');
	}, [form]);

	const handleFixWithAI = useCallback(() => {
		if (!validationResult) return;
		const parts: string[] = [];
		if (validationResult.errors?.length) {
			parts.push('Fix these validation errors:\n' + validationResult.errors.map((e: string) => `- ${e}`).join('\n'));
		}
		if (validationResult.warnings?.length) {
			parts.push('Also address these warnings:\n' + validationResult.warnings.map((w: string) => `- ${w}`).join('\n'));
		}
		setFixInstruction(parts.join('\n\n'));
		setAiTab('revise');
	}, [validationResult]);

	// ── Import EA state ──
	const [mode, setMode] = useState<string>('write');
	const [eaCode, setEaCode] = useState('');
	const [eaTranslating, setEaTranslating] = useState(false);
	const [eaResult, setEaResult] = useState('');

	const handleImportEA = useCallback(async () => {
		if (!eaCode.trim() || eaCode.trim().length < 20) {
			message.warning(t('strategy.importEA.codeTooShort', { defaultValue: 'Please paste complete EA/indicator source code.' }));
			return;
		}
		setEaTranslating(true);
		setEaResult('');
		try {
			const resp = await codeAssistClient.transformCode({
				sourceCode: eaCode,
				sourceLang: 'auto',
				targetLang: 'python',
			});
			setEaResult(resp.targetCode || '');
		} catch {
			message.error(t('strategy.importEA.translateFailed', { defaultValue: 'Translation failed. Please try again.' }));
		} finally {
			setEaTranslating(false);
		}
	}, [eaCode, t]);

	const applyEaResult = useCallback(() => {
		if (eaResult) {
			form.setFieldsValue({ code: eaResult });
			setMode('write');
			message.success(t('strategy.importEA.applied', { defaultValue: 'EA code translated and applied to editor.' }));
		}
	}, [eaResult, form, t]);

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
				<Button key="validate" onClick={() => { setAiTab('validate'); onValidate(code); }} loading={codeValidating} icon={<ThunderboltOutlined />}>
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

				{/* ── Step 2+3: Code editor / Import EA + AI assistant ── */}
				<Row gutter={16}>
					{/* Left: Code editor / Import EA — the main workspace */}
					<Col span={15}>
						<div style={{
							border: '1px solid var(--color-border)',
							borderRadius: 10, overflow: 'hidden',
							display: 'flex', flexDirection: 'column',
						}}>
							<div style={{
								padding: '4px 14px', background: 'var(--color-bg-tertiary)',
								borderBottom: '1px solid var(--color-border)',
								display: 'flex', alignItems: 'center', gap: 6,
							}}>
								<Segmented size="small"
								value={mode} onChange={v => setMode(v as string)}
								options={[
									{ value: 'write', icon: <CodeOutlined />, label: t('strategy.importEA.writeTab', { defaultValue: 'Strategy Code' }) },
									{ value: 'import', icon: <ImportOutlined />, label: t('strategy.importEA.importTab', { defaultValue: 'Import EA' }) },
								]}
							/>
								{mode === 'write' && code.trim() && (
									<Tag style={{ marginLeft: 'auto' }}>{code.split('\n').length} lines</Tag>
								)}
							</div>

							{mode === 'write' ? (
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
							) : (
								<div style={{ display: 'flex', flexDirection: 'column', height: 420 }}>
									<TextArea
										value={eaCode}
										onChange={e => setEaCode(e.target.value)}
										rows={10}
										placeholder={t('strategy.importEA.pastePlaceholder', { defaultValue: 'Paste MQL4/MQL5 EA or indicator source code here...' })}
										style={{
											fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace',
											fontSize: 13, lineHeight: 1.6,
											border: 'none', borderRadius: 0, resize: 'none',
											flex: 'none',
										}}
									/>
									<div style={{
										padding: '6px 14px', borderTop: '1px solid var(--color-border)',
										borderBottom: '1px solid var(--color-border)',
										display: 'flex', gap: 8, alignItems: 'center',
									}}>
										<Button type="primary" size="small" icon={<RobotOutlined />}
											onClick={handleImportEA} loading={eaTranslating}>
											{t('strategy.importEA.translate', { defaultValue: 'Translate to Python' })}
										</Button>
										{eaResult && (
											<Button size="small" onClick={applyEaResult}>
												{t('strategy.importEA.apply', { defaultValue: 'Apply to Editor' })}
											</Button>
										)}
									</div>
									{eaResult ? (
										<pre style={{
											flex: 1, overflow: 'auto', margin: 0, padding: '10px 14px',
											fontFamily: '"Fira Code", "Cascadia Code", "JetBrains Mono", monospace',
											fontSize: 12, lineHeight: 1.5, background: '#f9fafb',
											color: 'var(--color-text)', whiteSpace: 'pre-wrap',
										}}>
											{eaResult}
										</pre>
									) : (
										<div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
											{eaTranslating ? (
												<Spin tip={t('strategy.importEA.translating', { defaultValue: 'AI translating...' })} />
											) : (
												<Text type="secondary">
													{t('strategy.importEA.hint', { defaultValue: 'Paste MQL4/MQL5 code and click "Translate to Python"' })}
												</Text>
											)}
										</div>
									)}
								</div>
							)}
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
													<AICodeReviseChat code={code} onApply={applyAICode} initialInstruction={fixInstruction} />
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
									{
										key: 'validate',
										label: <span><CheckCircleOutlined /> {t('strategy.validate.tab', )}</span>,
										children: (
											<div style={{ height: 416, overflow: 'auto', padding: '12px' }}>
												{codeValidating ? (
													<div style={{ textAlign: 'center', padding: 60 }}>
														<Spin size="large" />
														<Text type="secondary" style={{ display: 'block', marginTop: 12 }}>
															{t('strategy.validate.running', { defaultValue: 'Running validation...' })}
														</Text>
													</div>
												) : validationResult ? (
													<div>
														<div style={{
															padding: '10px 14px', borderRadius: 8, marginBottom: 12,
															background: validationResult.valid ? '#f6ffed' : '#fff2f0',
															border: `1px solid ${validationResult.valid ? '#b7eb8f' : '#ffccc7'}`,
															display: 'flex', alignItems: 'center', gap: 8,
														}}>
															{validationResult.valid
																? <CheckCircleOutlined style={{ color: '#52c41a', fontSize: 18 }} />
																: <CloseCircleOutlined style={{ color: '#ff4d4f', fontSize: 18 }} />
															}
															<Text strong style={{ color: validationResult.valid ? '#52c41a' : '#ff4d4f' }}>
																{validationResult.valid
																	? t('strategy.validate.passed', )
																	: t('strategy.validate.failed', )}
															</Text>
														</div>
														{validationResult.errors?.length > 0 && (
															<div style={{ marginBottom: 10 }}>
																<Text strong style={{ fontSize: 12, color: '#ff4d4f' }}>❌ {t('strategy.validate.errors', { defaultValue: 'Errors' })}</Text>
																{validationResult.errors.map((e: string, i: number) => (
																	<div key={i} style={{ fontSize: 12, padding: '4px 8px', marginTop: 2, background: '#fff2f0', borderRadius: 4 }}>
																		<CloseCircleOutlined style={{ color: '#ff4d4f', marginRight: 4 }} />{e}
																	</div>
																))}
															</div>
														)}
														{validationResult.warnings?.length > 0 && (
															<div style={{ marginBottom: 10 }}>
																<Text strong style={{ fontSize: 12, color: '#fa8c16' }}>⚠️ {t('strategy.validate.warnings', { defaultValue: 'Warnings' })}</Text>
																{validationResult.warnings.map((w: string, i: number) => (
																	<div key={i} style={{ fontSize: 12, padding: '4px 8px', marginTop: 2, background: '#fff7e6', borderRadius: 4 }}>
																		<WarningOutlined style={{ color: '#fa8c16', marginRight: 4 }} />{w}
																	</div>
																))}
															</div>
														)}
									{!validationResult.valid && (validationResult.errors?.length > 0 || validationResult.warnings?.length > 0) && (
										<div style={{ textAlign: "center", marginBottom: 12 }}>
											<Button type="primary" size="small" icon={<BulbOutlined />}
												onClick={handleFixWithAI}>
												{t("strategy.validate.fixWithAI", { defaultValue: "Send errors to AI Revise" })}
											</Button>
										</div>
									)}
														{(validationResult as any).parameters?.length > 0 && (
															<div style={{ marginBottom: 10 }}>
																<Text strong style={{ fontSize: 12, color: '#1677ff' }}>📋 {(validationResult as any).parameters.length} {t('strategy.validate.parameters', { defaultValue: 'parameters' })}</Text>
																{(validationResult as any).parameters.map((p: any, i: number) => (
																	<Tag key={i} style={{ marginTop: 2 }}>{p.key}{p.defaultValue ? ` = ${p.defaultValue}` : ''}</Tag>
																))}
															</div>
														)}
														{(validationResult as any).qualityHints?.length > 0 && (
															<div style={{ marginBottom: 10 }}>
																<Text strong style={{ fontSize: 12 }}>💡 {t('strategy.validate.hints', { defaultValue: 'Suggestions' })}</Text>
																{(validationResult as any).qualityHints.map((h: any, i: number) => (
																	<div key={i} style={{ fontSize: 11, padding: '3px 6px', marginTop: 2, background: '#f0f5ff', borderRadius: 4 }}>
																		<InfoCircleOutlined style={{ color: '#1677ff', marginRight: 4 }} />
																		{h.message || h.description || JSON.stringify(h)}
																	</div>
																))}
															</div>
														)}
														{!validationResult.errors?.length && !validationResult.warnings?.length && validationResult.valid && (
															<Text type="secondary" style={{ textAlign: 'center', display: 'block', padding: 12 }}>
																{t('strategy.validate.allClear', { defaultValue: 'All checks passed — no issues found.' })}
															</Text>
														)}
													</div>
												) : (
													<div style={{ textAlign: 'center', padding: 40 }}>
														<ThunderboltOutlined style={{ fontSize: 28, color: '#d9d9d9' }} />
														<Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
															{t('strategy.validate.hint', { defaultValue: 'Click "Validate Code" to check syntax, imports, and strategy structure.' })}
														</Text>
													</div>
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
