import React, { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Space, Spin, Switch, Tag, message } from 'antd';
import { BulbOutlined, RobotOutlined, SendOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';

import { codeAssistApi, type CodeChatMessage, type RequiredParamSpec } from '@/client/codeAssist';

const { TextArea } = Input;

// --- 1. Required parameters form -----------------------------------------

// Substring -> i18n key. First match wins. Lowercased keys are matched
// against the param key. Descriptions live in the strategy.codeAssist.paramDescriptions namespace.
const PARAM_DESCRIPTION_RULES: Array<{ contains: string[]; i18nKey: string }> = [
	{ contains: ['risk_level', 'risklevel'], i18nKey: 'riskLevel' },
	{ contains: ['take_profit', 'tp_pct', 'tp_ratio'], i18nKey: 'takeProfit' },
	{ contains: ['stop_loss', 'sl_pct', 'sl_ratio'], i18nKey: 'stopLoss' },
	{ contains: ['max_loss'], i18nKey: 'maxLoss' },
	{ contains: ['confidence'], i18nKey: 'confidence' },
	{ contains: ['threshold'], i18nKey: 'threshold' },
	{ contains: ['lot', 'volume', 'size'], i18nKey: 'lotSize' },
	{ contains: ['fast'], i18nKey: 'fastPeriod' },
	{ contains: ['slow'], i18nKey: 'slowPeriod' },
	{ contains: ['signal'], i18nKey: 'signalPeriod' },
	{ contains: ['rsi'], i18nKey: 'rsiPeriod' },
	{ contains: ['ema'], i18nKey: 'emaPeriod' },
	{ contains: ['sma', 'ma_'], i18nKey: 'smaPeriod' },
	{ contains: ['period', 'length', 'window'], i18nKey: 'genericPeriod' },
	{ contains: ['pct', 'percent', 'ratio'], i18nKey: 'genericPercent' },
];

const useParamDescription = () => {
	const { t } = useTranslation();
	return (key: string): string => {
		const k = key.toLowerCase();
		for (const rule of PARAM_DESCRIPTION_RULES) {
			if (rule.contains.some((needle) => k.includes(needle))) {
				return t(`strategy.codeAssist.paramDescriptions.${rule.i18nKey}`, { defaultValue: '' });
			}
		}
		return '';
	};
};

export interface RequiredParamsFormProps {
	parameters: RequiredParamSpec[];
	values: Record<string, unknown>;
	onChange: (values: Record<string, unknown>) => void;
}

export const RequiredParamsForm: React.FC<RequiredParamsFormProps> = ({ parameters, values, onChange }) => {
	const { t } = useTranslation();
	const describe = useParamDescription();
	const required = parameters.filter((p) => p.required);
	const optional = parameters.filter((p) => !p.required);
	if (parameters.length === 0) return null;

	const placeholderFor = (p: RequiredParamSpec): string => {
		if (p.suggested !== undefined && p.suggested !== null) return String(p.suggested);
		if (p.default !== undefined && p.default !== null) return String(p.default);
		return '';
	};

	const renderInput = (p: RequiredParamSpec) => {
		const v = values[p.key];
		const set = (nv: unknown) => onChange({ ...values, [p.key]: nv });
		if (p.type === 'int' || p.type === 'float') {
			return (
				<InputNumber
					style={{ width: '100%' }}
					value={v as any}
					onChange={(nv) => set(nv)}
					placeholder={placeholderFor(p)}
				/>
			);
		}
		if (p.type === 'bool') {
			return <Switch checked={Boolean(v ?? p.suggested ?? p.default)} onChange={(nv) => set(nv)} />;
		}
		return (
			<Input
				value={v as any}
				onChange={(e) => set(e.target.value)}
				placeholder={placeholderFor(p)}
			/>
		);
	};

	const applyAllSuggestions = () => {
		const next: Record<string, unknown> = { ...values };
		for (const p of required) {
			if (p.suggested !== undefined && p.suggested !== null && (next[p.key] === undefined || next[p.key] === '' || next[p.key] === null)) {
				next[p.key] = p.suggested;
			}
		}
		onChange(next);
	};

	const hasAnySuggestion = required.some((p) => p.suggested !== undefined && p.suggested !== null);

	return (
		<div style={{ marginTop: 8 }}>
			{required.length > 0 && (
				<Alert
					type="warning"
					showIcon
					style={{ marginBottom: 8 }}
					message={t('strategy.codeAssist.requiredParamsTitle', { defaultValue: 'Required parameters' })}
					description={t('strategy.codeAssist.requiredParamsDesc', {
						defaultValue: 'The strategy reads these parameters but no default was provided. Fill them in before saving.',
					})}
				/>
			)}
			{required.length > 0 && hasAnySuggestion && (
				<div style={{ marginBottom: 8 }}>
					<Button size="small" icon={<ThunderboltOutlined />} onClick={applyAllSuggestions}>
						{t('strategy.codeAssist.applyAllSuggestions', { defaultValue: 'Apply suggested defaults' })}
					</Button>
				</div>
			)}
			<Form layout="vertical" style={{ marginTop: 4 }}>
				{required.map((p) => (
					<Form.Item
						key={p.key}
						label={
							<Space>
								<span style={{ fontFamily: 'monospace' }}>{p.key}</span>
								<Tag color="red">{t('strategy.codeAssist.required', { defaultValue: 'required' })}</Tag>
								{p.type ? <Tag>{p.type}</Tag> : null}
								{p.suggested !== undefined && p.suggested !== null ? (
									<Tag color="blue">
										{t('strategy.codeAssist.suggested', { defaultValue: 'suggested' })}: {String(p.suggested)}
									</Tag>
								) : null}
							</Space>
						}
						required
						extra={describe(p.key) || undefined}
					>
						{renderInput(p)}
					</Form.Item>
				))}
				{optional.length > 0 && (
					<>
						<Alert
							type="info"
							showIcon
							style={{ marginTop: 8, marginBottom: 8 }}
							message={t('strategy.codeAssist.optionalParamsTitle', { defaultValue: 'Optional parameters' })}
							description={t('strategy.codeAssist.optionalParamsDesc', {
								defaultValue:
									'These parameters already have defaults from the code. Leave a field blank to use the default, or override it for this run only — the saved strategy is not modified.',
							})}
						/>
						{optional.map((p) => (
							<Form.Item
								key={p.key}
								label={
									<Space>
										<span style={{ fontFamily: 'monospace' }}>{p.key}</span>
										{p.type ? <Tag>{p.type}</Tag> : null}
										{p.default !== undefined && p.default !== null ? (
											<Tag color="default">
												{t('strategy.codeAssist.defaultLabel', { defaultValue: 'default' })}: {String(p.default)}
											</Tag>
										) : null}
									</Space>
								}
								extra={describe(p.key) || undefined}
							>
								{renderInput(p)}
							</Form.Item>
						))}
					</>
				)}
			</Form>
		</div>
	);
};

// --- 2. Code explain panel -----------------------------------------------

export interface CodeExplainPanelProps {
	code: string;
	autoOnMount?: boolean;
}

export const CodeExplainPanel: React.FC<CodeExplainPanelProps> = ({ code, autoOnMount }) => {
	const { t, i18n } = useTranslation();
	const [loading, setLoading] = useState(false);
	const [text, setText] = useState('');
	const [error, setError] = useState('');

	const explain = async () => {
		if (!code.trim()) return;
		setLoading(true);
		setError('');
		try {
			const out = await codeAssistApi.explain({ code, locale: i18n.language });
			setText(out || '');
		} catch (e: any) {
			setError(String(e?.message || e || 'failed'));
		} finally {
			setLoading(false);
		}
	};

	useEffect(() => {
		// Reset when code changes; only auto-explain if requested.
		setText('');
		setError('');
		if (autoOnMount && code.trim()) {
			void explain();
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [code]);

	return (
		<div style={{ marginTop: 8 }}>
			<Space style={{ marginBottom: 8 }}>
				<Button icon={<BulbOutlined />} onClick={() => void explain()} loading={loading} disabled={!code.trim()}>
					{t('strategy.codeAssist.explain', { defaultValue: 'Explain code' })}
				</Button>
			</Space>
			{loading && !text ? <Spin /> : null}
			{error ? <Alert type="error" showIcon message={error} /> : null}
			{text ? (
				<div
					style={{
						background: '#fafafa',
						border: '1px solid #f0f0f0',
						padding: 12,
						borderRadius: 6,
						whiteSpace: 'pre-wrap',
						fontSize: 13,
						lineHeight: 1.6,
					}}
				>
					{text}
				</div>
			) : null}
		</div>
	);
};

// --- 3. AI revise chat (modal-side panel) --------------------------------

export interface AICodeReviseChatProps {
	code: string;
	onApply: (newCode: string) => void;
}

export const AICodeReviseChat: React.FC<AICodeReviseChatProps> = ({ code, onApply }) => {
	const { t, i18n } = useTranslation();
	const [history, setHistory] = useState<CodeChatMessage[]>([]);
	const [draft, setDraft] = useState('');
	const [loading, setLoading] = useState(false);

	const send = async () => {
		const instr = draft.trim();
		if (!instr) {
			message.warning(t('strategy.codeAssist.enterInstruction', {
				defaultValue: 'Please describe what you want to change.',
			}));
			return;
		}
		if (!code.trim()) {
			message.warning(t('strategy.codeAssist.codeEmpty', {
				defaultValue: 'There is no code to revise yet.',
			}));
			return;
		}
		setLoading(true);
		try {
			const out = await codeAssistApi.revise({
				code,
				instruction: instr,
				history,
				locale: i18n.language,
			});
			const newHistory = [
				...history,
				{ role: 'user' as const, content: instr },
				{ role: 'assistant' as const, content: out.text },
			];
			setHistory(newHistory);
			setDraft('');
			if (out.python) {
				onApply(out.python);
				message.success(t('strategy.codeAssist.codeUpdated', {
					defaultValue: 'Code updated. Please re-run validation before saving.',
				}));
			} else {
				message.warning(t('strategy.codeAssist.noPython', {
					defaultValue: 'AI did not return a Python block. Try rephrasing.',
				}));
			}
		} catch (e: any) {
			message.error(String(e?.message || e || 'failed'));
		} finally {
			setLoading(false);
		}
	};

	const messagesView = useMemo(
		() =>
			history.map((m, i) => (
				<div
					key={i}
					style={{
						margin: '6px 0',
						padding: '6px 10px',
						borderRadius: 6,
						background: m.role === 'user' ? '#e6f4ff' : '#f6ffed',
						fontSize: 12,
						whiteSpace: 'pre-wrap',
					}}
				>
					<b style={{ color: m.role === 'user' ? '#1677ff' : '#389e0d' }}>
						{m.role === 'user' ? t('common.you', { defaultValue: 'You' }) : 'AI'}
					</b>
					<div>{m.content}</div>
				</div>
			)),
		[history, t],
	);

	return (
		<div style={{ border: '1px solid #f0f0f0', borderRadius: 6, padding: 8, background: '#fff' }}>
			<Space style={{ marginBottom: 6 }}>
				<RobotOutlined />
				<span>{t('strategy.codeAssist.aiReviseTitle', { defaultValue: 'AI assistant — revise code' })}</span>
			</Space>
			<div style={{ maxHeight: 200, overflow: 'auto', marginBottom: 6 }}>{messagesView}</div>
			<TextArea
				rows={2}
				value={draft}
				onChange={(e) => setDraft(e.target.value)}
				placeholder={t('strategy.codeAssist.reviseInputPlaceholder', {
					defaultValue: 'e.g. Replace SMA(20) with EMA(50) and add a 1% stop-loss.',
				})}
			/>
			<div style={{ marginTop: 6, textAlign: 'right' }}>
				<Button type="primary" icon={<SendOutlined />} loading={loading} onClick={() => void send()}>
					{t('strategy.codeAssist.reviseSend', { defaultValue: 'Send to AI' })}
				</Button>
			</div>
		</div>
	);
};
