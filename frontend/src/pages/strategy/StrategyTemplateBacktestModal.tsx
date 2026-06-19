import React from 'react';
import { Modal, Button, Form, Input, InputNumber, Select, Space, Row, Col, DatePicker, Typography, Tag } from 'antd';
import { useTranslation } from 'react-i18next'
import { ACTIONS_BACKTEST_KEY, BACKTEST_ACCOUNT_DISABLED_SUFFIX_KEY, BACKTEST_FIELDS_ACCOUNT_KEY, BACKTEST_FIELDS_EXTRA_SYMBOLS_KEY, BACKTEST_FIELDS_INITIAL_CAPITAL_KEY, BACKTEST_FIELDS_RANGE_KEY, BACKTEST_FIELDS_SYMBOL_KEY, BACKTEST_FIELDS_TIMEFRAME_KEY, BACKTEST_FIELDS_TITLE_KEY, BACKTEST_MODAL_TITLE_WITH_NAME_KEY, BACKTEST_PARAMETERS_TITLE_KEY, BACKTEST_PLACEHOLDERS_ACCOUNT_KEY, BACKTEST_PLACEHOLDERS_EXTRA_SYMBOLS_KEY, BACKTEST_PLACEHOLDERS_RANGE_KEY, BACKTEST_PLACEHOLDERS_SYMBOL_KEY, BACKTEST_VALIDATION_ACCOUNT_REQUIRED_KEY, BACKTEST_VALIDATION_INITIAL_CAPITAL_REQUIRED_KEY, BACKTEST_VALIDATION_RANGE_REQUIRED_KEY, BACKTEST_VALIDATION_SYMBOL_REQUIRED_KEY, BACKTEST_VALIDATION_TIMEFRAME_REQUIRED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import type { FormInstance } from 'antd';
import type dayjs from 'dayjs';
import type { StrategyTemplate } from '@/client/strategy';
import { quickRangeLabel, type QuickRangeKey } from './StrategyTemplatePage.utils';
import { RequiredParamsForm } from '@/components/strategy/CodeAssist';
import type { RequiredParamSpec } from '@/client/codeAssist';
import { TIMEFRAMES, DEFAULT_TIMEFRAME } from '@/constants/timeframes';

const { RangePicker } = DatePicker;

export type StrategyTemplateBacktestModalProps = {
	open: boolean;
	template: StrategyTemplate | null;
	form: FormInstance;
	submitting: boolean;
	accounts: any[];
	symbols: { value: string; label: string }[];
	symbolsLoading: boolean;
	quickRange: QuickRangeKey;
	watchedRange: [dayjs.Dayjs, dayjs.Dayjs] | undefined;
	// Required-parameter section, populated when the user opens the modal
	// (the parent extracts ``params['xxx']`` keys via the strategy-service
	// static analyser). Submission is blocked unless all required keys are
	// filled — symbol/timeframe/account live in the form fields below.
	requiredParams?: RequiredParamSpec[];
	paramValues?: Record<string, unknown>;
	onParamValuesChange?: (v: Record<string, unknown>) => void;
	onCancel: () => void;
	onSubmit: () => void;
	onApplyQuickRange: (key: QuickRangeKey) => void;
	onSetQuickRange: (key: QuickRangeKey) => void;
	onAccountChange: (accountId: string) => Promise<void>;
};

export const StrategyTemplateBacktestModal: React.FC<StrategyTemplateBacktestModalProps> = ({
	open,
	template,
	form,
	submitting,
	accounts,
	symbols,
	symbolsLoading,
	quickRange,
	watchedRange,
	requiredParams = [],
	paramValues = {},
	onParamValuesChange,
	onCancel,
	onSubmit,
	onApplyQuickRange,
	onSetQuickRange,
	onAccountChange,
}) => {
	const { t } = useTranslation();
	return (
		<Modal
			title={template ? t(BACKTEST_MODAL_TITLE_WITH_NAME_KEY, { name: template.name }) : t(ACTIONS_BACKTEST_KEY)}
			open={open}
			onCancel={onCancel}
			onOk={onSubmit}
			confirmLoading={submitting}
			width={720}
		>
			<Form form={form} size="small" layout="vertical" initialValues={{ timeframe: DEFAULT_TIMEFRAME, initialCapital: 10000 }}>
				{template && Array.isArray(template?.parameters) && template.parameters.length > 0 && (
					<div
						style={{
							marginBottom: 12,
							padding: 8,
							borderRadius: 4,
							background: '#fafafa',
							border: '1px solid #f0f0f0',
						}}
					>
						<Typography.Text strong>
							{t(BACKTEST_PARAMETERS_TITLE_KEY, '策略参数')}
						</Typography.Text>
						<div style={{ marginTop: 6 }}>
							<Space size={[6, 6]} wrap>
								{template.parameters.map((p: { name?: string; label?: string; type?: string; default?: unknown }) => (
									<Tag key={String(p?.name || '')} color="blue" style={{ marginInlineEnd: 0 }}>
										<span style={{ fontWeight: 500 }}>{String(p?.label || p?.name || '')}</span>
										<span style={{ opacity: 0.65 }}> ({String(p?.name || '')})</span>
										{p?.default !== undefined && p?.default !== '' && (
											<span style={{ opacity: 0.65 }}> = {String(p.default)}</span>
										)}
									</Tag>
								))}
							</Space>
						</div>
					</div>
				)}
				<Form.Item name="title" label={t(BACKTEST_FIELDS_TITLE_KEY)}>
					<Input readOnly />
				</Form.Item>
				<Row gutter={8}>
					<Col flex="260px">
						<Form.Item
							name="accountId"
							label={t(BACKTEST_FIELDS_ACCOUNT_KEY)}
							rules={[{ required: true, message: t(BACKTEST_VALIDATION_ACCOUNT_REQUIRED_KEY) }]}
						>
							<Select
								size="small"
								placeholder={t(BACKTEST_PLACEHOLDERS_ACCOUNT_KEY)}
								onChange={async (v) => {
									form.setFieldsValue({ symbol: '' });
									await onAccountChange(String(v));
								}}
								options={(accounts || []).map((a: { id?: string; login?: string; brokerCompany?: string }) => ({
									value: String(a.id),
									label: `${a.login ?? a.id} (${a.mtType ?? ''})${a.isDisabled ? t(BACKTEST_ACCOUNT_DISABLED_SUFFIX_KEY) : ''}`,
									disabled: !!a.isDisabled,
								}))}
							/>
						</Form.Item>
					</Col>
					<Col flex="260px">
						<Form.Item
							name="symbol"
							label={t(BACKTEST_FIELDS_SYMBOL_KEY)}
							rules={[{ required: true, message: t(BACKTEST_VALIDATION_SYMBOL_REQUIRED_KEY) }]}
						>
							<Select
								size="small"
								showSearch
								allowClear
								loading={symbolsLoading}
								placeholder={t(BACKTEST_PLACEHOLDERS_SYMBOL_KEY)}
								options={symbols}
								optionFilterProp="label"
							/>
						</Form.Item>
					</Col>
				</Row>
				<Form.Item
					name="extraSymbols"
					label={t(BACKTEST_FIELDS_EXTRA_SYMBOLS_KEY, '辅助标的（可多选）')}
					tooltip={t(
						'strategy.templates.backtest.tooltips.extraSymbols',
						'除主标的外，额外拉取的 K 线（同账户、同周期）。策略通过 context["closes_by_symbol"] 访问。',
					)}
				>
					<Select
						size="small"
						mode="multiple"
						allowClear
						loading={symbolsLoading}
						placeholder={t(BACKTEST_PLACEHOLDERS_EXTRA_SYMBOLS_KEY, '可选，配对/轮动策略常用')}
						options={symbols}
						optionFilterProp="label"
						maxTagCount="responsive"
					/>
				</Form.Item>
				<Row gutter={8}>
					<Col flex="160px">
						<Form.Item
							name="timeframe"
							label={t(BACKTEST_FIELDS_TIMEFRAME_KEY)}
							rules={[{ required: true, message: t(BACKTEST_VALIDATION_TIMEFRAME_REQUIRED_KEY) }]}
						>
							<Select
								size="small"
								options={TIMEFRAMES.map(tf => ({ value: tf, label: tf }))}
							/>
						</Form.Item>
					</Col>
					<Col flex="220px">
						<Form.Item
							name="initialCapital"
							label={t(BACKTEST_FIELDS_INITIAL_CAPITAL_KEY)}
							rules={[{ required: true, message: t(BACKTEST_VALIDATION_INITIAL_CAPITAL_REQUIRED_KEY) }]}
						>
							<InputNumber style={{ width: '100%' }} min={1} step={100} size="small" />
						</Form.Item>
					</Col>
				</Row>
				<Form.Item label={t(BACKTEST_FIELDS_RANGE_KEY)}>
					<div style={{ marginBottom: 4 }}>
						<Space size="small" wrap>
							<Button type={quickRange === '1D' ? 'primary' : 'default'} size="small" onClick={() => onApplyQuickRange('1D')}>
								{quickRangeLabel(t, '1D')}
							</Button>
							<Button type={quickRange === '3D' ? 'primary' : 'default'} size="small" onClick={() => onApplyQuickRange('3D')}>
								{quickRangeLabel(t, '3D')}
							</Button>
							<Button type={quickRange === '1W' ? 'primary' : 'default'} size="small" onClick={() => onApplyQuickRange('1W')}>
								{quickRangeLabel(t, '1W')}
							</Button>
							<Button type={quickRange === '1Y' ? 'primary' : 'default'} size="small" onClick={() => onApplyQuickRange('1Y')}>
								{quickRangeLabel(t, '1Y')}
							</Button>
							<Button type={quickRange === 'CUSTOM' ? 'primary' : 'default'} size="small" onClick={() => onApplyQuickRange('CUSTOM')}>
								{quickRangeLabel(t, 'CUSTOM')}
							</Button>
						</Space>
					</div>

					<Input
						size="small"
						readOnly
						style={{ maxWidth: 420 }}
						value={
							watchedRange?.[0] && watchedRange?.[1]
								? `${watchedRange[0].format('YYYY-MM-DD HH:mm')} → ${watchedRange[1].format('YYYY-MM-DD HH:mm')}`
								: ''
						}
						placeholder={t(BACKTEST_PLACEHOLDERS_RANGE_KEY)}
					/>

					<Form.Item name="range" rules={[{ required: true, message: t(BACKTEST_VALIDATION_RANGE_REQUIRED_KEY) }]}>
						<div style={{ marginTop: 4, display: quickRange === 'CUSTOM' ? 'block' : 'none', maxWidth: 420 }}>
							<RangePicker style={{ width: '100%' }} showTime onChange={() => onSetQuickRange('CUSTOM')} size="small" />
						</div>
					</Form.Item>
				</Form.Item>
				{requiredParams.length > 0 && onParamValuesChange ? (
					<RequiredParamsForm
						parameters={requiredParams}
						values={paramValues}
						onChange={onParamValuesChange}
					/>
				) : null}
			</Form>
		</Modal>
	);
};
