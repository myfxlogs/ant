import React, { useEffect, useState } from 'react';
import { Alert, Button, Form, Input, InputNumber, Space, Spin, Switch, Tag } from 'antd';
import { BulbOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { APPLY_ALL_SUGGESTIONS_KEY, DEFAULT_LABEL_KEY, EXPLAIN_KEY, OPTIONAL_PARAMS_DESC_KEY, OPTIONAL_PARAMS_TITLE_KEY, REQUIRED_KEY, REQUIRED_PARAMS_DESC_KEY, REQUIRED_PARAMS_TITLE_KEY, SUGGESTED_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';

;

import { codeAssistApi, type RequiredParamSpec } from '@/client/codeAssist';

const { TextArea: _TextArea } = Input;

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
					value={(typeof v === 'number' ? v : null)}
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
				value={v === undefined || v === null ? undefined : String(v)}
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
					message={t(REQUIRED_PARAMS_TITLE_KEY, { defaultValue: 'Required parameters' })}
					description={t(REQUIRED_PARAMS_DESC_KEY, {
						defaultValue: 'The strategy reads these parameters but no default was provided. Fill them in before saving.',
					})}
				/>
			)}
			{required.length > 0 && hasAnySuggestion && (
				<div style={{ marginBottom: 8 }}>
					<Button size="small" icon={<ThunderboltOutlined />} onClick={applyAllSuggestions}>
						{t(APPLY_ALL_SUGGESTIONS_KEY, { defaultValue: 'Apply suggested defaults' })}
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
								<Tag color="red">{t(REQUIRED_KEY, { defaultValue: 'required' })}</Tag>
								{p.type ? <Tag>{p.type}</Tag> : null}
								{p.suggested !== undefined && p.suggested !== null ? (
									<Tag color="blue">
										{t(SUGGESTED_KEY, { defaultValue: 'suggested' })}: {String(p.suggested)}
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
							message={t(OPTIONAL_PARAMS_TITLE_KEY, { defaultValue: 'Optional parameters' })}
							description={t(OPTIONAL_PARAMS_DESC_KEY, {
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
												{t(DEFAULT_LABEL_KEY, { defaultValue: 'default' })}: {String(p.default)}
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
		} catch (e: unknown) {
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
					{t(EXPLAIN_KEY, { defaultValue: 'Explain code' })}
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

// AICodeReviseChat extracted to its own file for size compliance.
export { AICodeReviseChat, type AICodeReviseChatProps } from "./AICodeReviseChat";
