import React from 'react';
import { Modal, Button, Collapse, Form, Input, Switch } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd';
import type { StrategyTemplate } from '@/client/strategy';
import { AICodeReviseChat, CodeExplainPanel } from '@/components/strategy/CodeAssist';

const { TextArea } = Input;

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
	onSubmit: (values: any) => void;
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
	// AI 面板需要看到「当前 code 字段实时值」。Form.useWatch 直接订阅 form，
	// 用户键入或 applyAICode 写入都会同步到下方 panels —— 无需手动维护
	// useState + useEffect 镜像（同时绕开 react-hooks/set-state-in-effect）。
	const watchedCode = Form.useWatch<string | undefined>('code', form) ?? '';
	const code = watchedCode || (form.getFieldValue('code') as string) || '';

	const applyAICode = (newCode: string) => {
		form.setFieldsValue({ code: newCode });
	};

	const collapseItems = [
		{
			key: 'ai',
			label: t('strategy.codeAssist.tabAI', { defaultValue: 'AI revise' }),
			children: <AICodeReviseChat code={code} onApply={applyAICode} />,
		},
		{
			key: 'explain',
			label: t('strategy.codeAssist.tabExplain', { defaultValue: 'Explain code' }),
			children: <CodeExplainPanel code={code} />,
		},
	];

	return (
		<Modal
			title={editingTemplate ? t('strategy.templates.editTemplateModal.title.edit') : t('strategy.templates.editTemplateModal.title.create')}
			open={open}
			onCancel={onCancel}
			footer={[
				<Button key="cancel" onClick={onCancel} disabled={codeValidating}>
					{t('common.cancel')}
				</Button>,
				<Button key="validate" onClick={onValidate} loading={codeValidating}>
					{t('strategy.templates.editTemplateModal.actions.validateCode')}
				</Button>,
				<Button
					key="save"
					type="primary"
					onClick={() => form.submit()}
					loading={codeValidating}
					disabled={!code.trim() || code !== lastValidatedCode}
					title={code && code !== lastValidatedCode
						? t('strategy.codeAssist.saveBlockedNotValidated', {
							defaultValue: 'Please run "Validate code" first. Save is disabled until validation passes.',
						})
						: undefined}
				>
					{t('common.save')}
				</Button>,
			]}
			width={900}
		>
			<Form
				form={form}
				layout="vertical"
				onFinish={onSubmit}
				initialValues={{ isPublic: false }}
				onValuesChange={(_, all) => setCode(all?.code || '')}
			>
				<Form.Item
					name="name"
					label={t('strategy.templates.editTemplateModal.fields.name')}
					rules={[{ required: true, message: t('strategy.templates.editTemplateModal.validation.nameRequired') }]}
				>
					<Input placeholder={t('strategy.templates.editTemplateModal.placeholders.name')} />
				</Form.Item>
				<Form.Item name="description" label={t('strategy.templates.editTemplateModal.fields.description')}>
					<TextArea rows={2} placeholder={t('strategy.templates.editTemplateModal.placeholders.description')} />
				</Form.Item>
				<Form.Item
					name="code"
					label={t('strategy.templates.editTemplateModal.fields.code')}
					rules={[{ required: true, message: t('strategy.templates.editTemplateModal.validation.codeRequired') }]}
				>
					<TextArea
						rows={12}
						placeholder={t('strategy.templates.editTemplateModal.placeholders.codeSample')}
						style={{ fontFamily: 'monospace' }}
					/>
				</Form.Item>
				<Form.Item name="isPublic" label={t('strategy.templates.editTemplateModal.fields.publicShare')} valuePropName="checked">
					<Switch
						checkedChildren={t('strategy.templates.visibility.public')}
						unCheckedChildren={t('strategy.templates.visibility.private')}
					/>
				</Form.Item>
			</Form>
			<Collapse items={collapseItems} style={{ marginTop: 12 }} />
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
			title={t('strategy.templates.codeModal.title')}
			open={open}
			onCancel={onClose}
			footer={[
				<Button key="copy" icon={<CopyOutlined />} onClick={() => onCopy(code)}>
					{t('strategy.templates.codeModal.actions.copy')}
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
