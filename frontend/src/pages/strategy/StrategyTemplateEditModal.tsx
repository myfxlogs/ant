import React from 'react';
import { Modal, Button, Collapse, Form, Input, Switch } from 'antd';
import { CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CODE_MODAL_ACTIONS_COPY_KEY, CODE_MODAL_TITLE_KEY, EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY, EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY, EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY, EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY, EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY, EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY, EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY, VISIBILITY_PRIVATE_KEY, VISIBILITY_PUBLIC_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
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
			label: t(TAB_A_I_KEY, { defaultValue: 'AI revise' }),
			children: <AICodeReviseChat code={code} onApply={applyAICode} />,
		},
		{
			key: 'explain',
			label: t(TAB_EXPLAIN_KEY, { defaultValue: 'Explain code' }),
			children: <CodeExplainPanel code={code} />,
		},
	];

	return (
		<Modal
			title={editingTemplate ? t(EDIT_TEMPLATE_MODAL_TITLE_EDIT_KEY) : t(EDIT_TEMPLATE_MODAL_TITLE_CREATE_KEY)}
			open={open}
			onCancel={onCancel}
			footer={[
				<Button key="cancel" onClick={onCancel} disabled={codeValidating}>
					{t('common.cancel')}
				</Button>,
				<Button key="validate" onClick={onValidate} loading={codeValidating}>
					{t(EDIT_TEMPLATE_MODAL_ACTIONS_VALIDATE_CODE_KEY)}
				</Button>,
				<Button
					key="save"
					type="primary"
					onClick={() => form.submit()}
					loading={codeValidating}
					disabled={!code.trim() || code !== lastValidatedCode}
					title={code && code !== lastValidatedCode
						? t(SAVE_BLOCKED_NOT_VALIDATED_KEY, {
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
				onValuesChange={undefined}
			>
				<Form.Item
					name="name"
					label={t(EDIT_TEMPLATE_MODAL_FIELDS_NAME_KEY)}
					rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_NAME_REQUIRED_KEY) }]}
				>
					<Input placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_NAME_KEY)} />
				</Form.Item>
				<Form.Item name="description" label={t(EDIT_TEMPLATE_MODAL_FIELDS_DESCRIPTION_KEY)}>
					<TextArea rows={2} placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_DESCRIPTION_KEY)} />
				</Form.Item>
				<Form.Item
					name="code"
					label={t(EDIT_TEMPLATE_MODAL_FIELDS_CODE_KEY)}
					rules={[{ required: true, message: t(EDIT_TEMPLATE_MODAL_VALIDATION_CODE_REQUIRED_KEY) }]}
				>
					<TextArea
						rows={12}
						placeholder={t(EDIT_TEMPLATE_MODAL_PLACEHOLDERS_CODE_SAMPLE_KEY)}
						style={{ fontFamily: 'monospace' }}
					/>
				</Form.Item>
				<Form.Item name="isPublic" label={t(EDIT_TEMPLATE_MODAL_FIELDS_PUBLIC_SHARE_KEY)} valuePropName="checked">
					<Switch
						checkedChildren={t(VISIBILITY_PUBLIC_KEY)}
						unCheckedChildren={t(VISIBILITY_PRIVATE_KEY)}
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
