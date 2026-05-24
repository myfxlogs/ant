import React from 'react';
import { Modal, Button, Form, Input, InputNumber, Select, Space, Row, Col, DatePicker, Typography, Tag } from 'antd';
import { useTranslation } from 'react-i18next';
import type { FormInstance } from 'antd';
import type dayjs from 'dayjs';
import type { StrategyTemplate } from '@/client/strategy';
import { quickRangeLabel, type QuickRangeKey } from './StrategyTemplatePage.utils';
import { RequiredParamsForm } from '@/components/strategy/CodeAssist';
import type { RequiredParamSpec } from '@/client/codeAssist';

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
			title={template ? t('strategy.templates.backtest.modalTitleWithName', { name: template.name }) : t('strategy.templates.actions.backtest')}
			open={open}
			onCancel={onCancel}
			onOk={onSubmit}
			confirmLoading={submitting}
			width={720}
		>
			<Form form={form} size="small" layout="vertical" initialValues={{ timeframe: 'H1', initialCapital: 10000 }}>
				{template && Array.isArray((template as any)?.parameters) && (template as any).parameters.length > 0 && (
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
							{t('strategy.templates.backtest.parameters.title', '策略参数')}
						</Typography.Text>
						<div style={{ marginTop: 6 }}>
							<Space size={[6, 6]} wrap>
								{(template as any).parameters.map((p: any) => (
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
				<Form.Item name="title" label={t('strategy.templates.backtest.fields.title')}>
					<Input readOnly />
				</Form.Item>
				<Row gutter={8}>
					<Col flex="260px">
						<Form.Item
							name="accountId"
							label={t('strategy.templates.backtest.fields.account')}
							rules={[{ required: true, message: t('strategy.templates.backtest.validation.accountRequired') }]}
						>
							<Select
								size="small"
								placeholder={t('strategy.templates.backtest.placeholders.account')}
								onChange={async (v) => {
									form.setFieldsValue({ symbol: '' });
									await onAccountChange(String(v));
								}}
								options={(accounts || []).map((a: any) => ({
									value: String(a.id),
									label: `${a.login ?? a.id} (${a.mtType ?? ''})${a.isDisabled ? t('strategy.templates.backtest.accountDisabledSuffix') : ''}`,
									disabled: !!a.isDisabled,
								}))}
							/>
						</Form.Item>
					</Col>
					<Col flex="260px">
						<Form.Item
							name="symbol"
							label={t('strategy.templates.backtest.fields.symbol')}
							rules={[{ required: true, message: t('strategy.templates.backtest.validation.symbolRequired') }]}
						>
							<Select
								size="small"
								showSearch
								allowClear
								loading={symbolsLoading}
								placeholder={t('strategy.templates.backtest.placeholders.symbol')}
								options={symbols}
								optionFilterProp="label"
							/>
						</Form.Item>
					</Col>
				</Row>
				<Form.Item
					name="extraSymbols"
					label={t('strategy.templates.backtest.fields.extraSymbols', '辅助标的（可多选）')}
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
						placeholder={t('strategy.templates.backtest.placeholders.extraSymbols', '可选，配对/轮动策略常用')}
						options={symbols}
						optionFilterProp="label"
						maxTagCount="responsive"
					/>
				</Form.Item>
				<Row gutter={8}>
					<Col flex="160px">
						<Form.Item
							name="timeframe"
							label={t('strategy.templates.backtest.fields.timeframe')}
							rules={[{ required: true, message: t('strategy.templates.backtest.validation.timeframeRequired') }]}
						>
							<Select
								size="small"
								options={[
									{ value: 'M1', label: 'M1' },
									{ value: 'M5', label: 'M5' },
									{ value: 'M15', label: 'M15' },
									{ value: 'M30', label: 'M30' },
									{ value: 'H1', label: 'H1' },
									{ value: 'H4', label: 'H4' },
									{ value: 'D1', label: 'D1' },
								]}
							/>
						</Form.Item>
					</Col>
					<Col flex="220px">
						<Form.Item
							name="initialCapital"
							label={t('strategy.templates.backtest.fields.initialCapital')}
							rules={[{ required: true, message: t('strategy.templates.backtest.validation.initialCapitalRequired') }]}
						>
							<InputNumber style={{ width: '100%' }} min={1} step={100} size="small" />
						</Form.Item>
					</Col>
				</Row>
				<Form.Item label={t('strategy.templates.backtest.fields.range')}>
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
						placeholder={t('strategy.templates.backtest.placeholders.range')}
					/>

					<Form.Item name="range" rules={[{ required: true, message: t('strategy.templates.backtest.validation.rangeRequired') }]}>
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
