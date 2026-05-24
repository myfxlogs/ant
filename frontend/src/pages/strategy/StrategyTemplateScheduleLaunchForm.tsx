import React, { useEffect, useMemo, useState } from 'react';
import {
	Alert,
	Button,
	Divider,
	Form,
	Input,
	InputNumber,
	Modal,
	Select,
	Space,
	Switch,
	Tag,
	Tooltip,
	message,
} from 'antd';
import { LockOutlined, SafetyCertificateOutlined, ExclamationCircleOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { accountApi } from '@/client/account';
import type { RequiredParamSpec } from '@/client/codeAssist';
import { RequiredParamsForm } from '@/components/strategy/CodeAssist';
import { isTradingAccountEnabled } from '@/utils/accountStatus';
import { buildParametersFromForm } from './StrategyScheduleParams';

// 这个表单是「上线到调度」Modal 的核心。它产出一份可以直接喂给
// strategyApi.createSchedule 的参数对象，并暴露风控参数（存在 parameters map
// 里以 __risk.* 为前缀的键，供后端 strategy_schedule_runner 使用）。

export type ScheduleLaunchFormValues = {
	scheduleName?: string;
	accountId: string;
	symbol: string;
	timeframe: string;
	scheduleType: 'interval' | 'kline_close' | 'hf_quote';
	intervalMs: number; // for interval
	hfCooldownMs: number; // for hf_quote
	// Risk params (optional):
	defaultVolume?: number;
	maxPositions?: number;
	stopLossPriceOffset?: number;
	takeProfitPriceOffset?: number;
	maxDrawdownPct?: number; // 0-1 (e.g. 0.2 = 20%)
	// Workflow:
	enableAfterCreate: boolean;
	strategyParams?: Record<string, unknown>;
};

export type SubmitParams = {
	form: ScheduleLaunchFormValues;
	// buildParameters 把 risk 字段编码进 parameters<string,string> map，便于传给
	// strategyApi.createSchedule。
	buildParameters: () => Record<string, string>;
};

type AccountOption = {
	id: string;
	label: string;
	mtType: string;
	isInvestor: boolean;
	isDisabled: boolean;
};

type SymbolOption = { value: string; label: string };

type Props = {
	open: boolean;
	accounts: any[]; // 来自 StrategyTemplatePage 的 accounts state
	symbols: SymbolOption[];
	symbolsLoading: boolean;
	defaults?: Partial<ScheduleLaunchFormValues>;
	requiredParams?: RequiredParamSpec[];
	paramValues?: Record<string, unknown>;
	onParamValuesChange?: (v: Record<string, unknown>) => void;
	onAccountChange?: (accountId: string) => void | Promise<void>;
	onSubmit: (params: SubmitParams) => void;
	submitting: boolean;
	disabled: boolean;
};

// 把 RPC 返回的 Account 列表转为选项。优先显示真实 MT login，alias 作为附注。
// 之所以不拼 alias 在 login 之前，是因为 alias 可能被用户误填为服务器名，
// 反而让人分不清哪个是真正的交易账号。
function normalizeAccounts(accounts: any[] | undefined): AccountOption[] {
	if (!Array.isArray(accounts)) return [];
	return accounts
		.filter(isTradingAccountEnabled)
		.map((a) => {
			const id = String(a?.id || '');
			const login = String(a?.login || '').trim();
			const alias = String(a?.alias || '').trim();
			const mtType = String(a?.mtType || a?.mt_type || '').toUpperCase();
			const isInvestor = Boolean(a?.isInvestor ?? a?.is_investor);
			const isDisabled = Boolean(a?.isDisabled ?? a?.is_disabled);
			// 主要标签：login (MT_TYPE)，若有 alias 附加一个短后缀。
			const base = `${login || id} (${mtType})`;
			const label = alias && alias !== login ? `${base} · ${alias}` : base;
			return { id, label, mtType, isInvestor, isDisabled };
		})
		.filter((o) => o.id);
}

export const StrategyTemplateScheduleLaunchForm: React.FC<Props> = ({
	open,
	accounts,
	symbols,
	symbolsLoading,
	defaults,
	requiredParams = [],
	paramValues = {},
	onParamValuesChange,
	onAccountChange,
	onSubmit,
	submitting,
	disabled,
}) => {
	const { t } = useTranslation();
	const [form] = Form.useForm<ScheduleLaunchFormValues>();

	// Trade-permission live status for the currently selected account.
	const accountOptions = useMemo(() => normalizeAccounts(accounts), [accounts]);
	const [selectedAccountId, setSelectedAccountId] = useState<string>(
		String(defaults?.accountId || accountOptions[0]?.id || ''),
	);
	const [tradePermission, setTradePermission] = useState<{
		loading: boolean;
		hasTradePermission: boolean;
		isInvestor: boolean;
		verified: boolean;
		message: string;
	}>({ loading: false, hasTradePermission: false, isInvestor: false, verified: false, message: '' });
	const [passwordModalOpen, setPasswordModalOpen] = useState<boolean>(false);

	const selectedAccount = useMemo(
		() => accountOptions.find((a) => a.id === selectedAccountId) || null,
		[accountOptions, selectedAccountId],
	);

	// 当 Modal 被打开或账户列表变化时，重置表单为 defaults。
	useEffect(() => {
		if (!open) return;
		const initial: Partial<ScheduleLaunchFormValues> = {
			scheduleName: defaults?.scheduleName || '',
			accountId: String(defaults?.accountId || accountOptions[0]?.id || ''),
			symbol: defaults?.symbol || '',
			timeframe: defaults?.timeframe || 'H1',
			scheduleType: defaults?.scheduleType || 'kline_close',
			intervalMs: defaults?.intervalMs || 300_000,
			hfCooldownMs: defaults?.hfCooldownMs || 1_000,
			defaultVolume: defaults?.defaultVolume,
			maxPositions: defaults?.maxPositions,
			stopLossPriceOffset: defaults?.stopLossPriceOffset,
			takeProfitPriceOffset: defaults?.takeProfitPriceOffset,
			maxDrawdownPct: defaults?.maxDrawdownPct,
			enableAfterCreate: defaults?.enableAfterCreate ?? true,
		};
		form.setFieldsValue(initial as any);
		const initialAccountId = String(initial.accountId || '');
		setSelectedAccountId(initialAccountId);
		if (initialAccountId) {
			// 初次打开也触发一次父组件的 symbols 加载，保证品种下拉在第一次点开就有数据。
			void onAccountChange?.(initialAccountId);
		}
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [open, accountOptions.length]);

	// 选到某个账户后，自动发一次 VerifyTradePermission。
	useEffect(() => {
		if (!open || !selectedAccountId) {
			setTradePermission({ loading: false, hasTradePermission: false, isInvestor: false, verified: false, message: '' });
			return;
		}
		// 先用本地 is_investor 字段给出乐观提示，再发远端探测。
		const local = selectedAccount;
		setTradePermission((p) => ({
			...p,
			isInvestor: Boolean(local?.isInvestor),
			hasTradePermission: !(local?.isInvestor),
			verified: false,
			message: '',
		}));
		let cancelled = false;
		void (async () => {
			setTradePermission((p) => ({ ...p, loading: true }));
			try {
				const r = await accountApi.verifyTradePermission(selectedAccountId);
				if (cancelled) return;
				setTradePermission({
					loading: false,
					hasTradePermission: r.hasTradePermission,
					isInvestor: r.isInvestor,
					verified: r.verified,
					message: r.message,
				});
			} catch (e: any) {
				if (cancelled) return;
				setTradePermission({
					loading: false,
					hasTradePermission: false,
					isInvestor: Boolean(local?.isInvestor),
					verified: false,
					message: String(e?.message || e || ''),
				});
			}
		})();
		return () => {
			cancelled = true;
		};
		// eslint-disable-next-line react-hooks/exhaustive-deps
	}, [open, selectedAccountId]);

	const watchedScheduleType = Form.useWatch('scheduleType', form);

	const buildParameters = (): Record<string, string> => {
		const v = form.getFieldsValue(true) as any;
		const out = buildParametersFromForm(v);
		for (const [key, raw] of Object.entries(paramValues || {})) {
			if (!key || raw === undefined || raw === null || raw === '') continue;
			out[key] = String(raw);
		}
		return out;
	};

	const handleFinish = async () => {
		try {
			const v = (await form.validateFields()) as ScheduleLaunchFormValues;
			// 若账户确认是投资者模式 → 阻止提交并提示填密码。
			if (tradePermission.verified && tradePermission.isInvestor) {
				message.error(t('strategy.templates.scheduleLaunch.errorInvestorAccount', '所选账户是投资者只读模式，请先填写交易密码'));
				return;
			}
			const missingParams = (requiredParams || [])
				.filter((p) => p.required)
				.filter((p) => {
					const value = paramValues[p.key];
					return value === undefined || value === null || value === '';
				});
			if (missingParams.length > 0) {
				message.error(
					t('strategy.codeAssist.fillRequiredParams', {
						defaultValue: 'Please fill the required parameters: {{keys}}',
						keys: missingParams.map((m) => m.key).join(', '),
					}),
				);
				return;
			}
			onSubmit({ form: v, buildParameters });
		} catch {
			// validate failed; AntD already highlights the offending field.
		}
	};

	const investorBanner = tradePermission.isInvestor ? (
		<Alert
			type="error"
			showIcon
			icon={<ExclamationCircleOutlined />}
			className="mb-3"
			message={t(
				'strategy.templates.scheduleLaunch.investorWarningTitle',
				'此账户无交易权限（投资者只读模式）',
			)}
			description={
				<div>
					{t(
						'strategy.templates.scheduleLaunch.investorWarningBody',
						'请填写该账户的交易密码以启用自动下单。',
					)}
					<div className="mt-2">
						<Button
							size="small"
							icon={<LockOutlined />}
							onClick={() => setPasswordModalOpen(true)}
						>
							{t('strategy.templates.scheduleLaunch.actions.updateTradingPassword', '填写交易密码')}
						</Button>
					</div>
				</div>
			}
		/>
	) : tradePermission.verified && tradePermission.hasTradePermission ? (
		<Alert
			type="success"
			showIcon
			icon={<SafetyCertificateOutlined />}
			className="mb-3"
			message={t('strategy.templates.scheduleLaunch.tradePermissionOk', '账户已验证有交易权限')}
		/>
	) : tradePermission.loading ? (
		<Alert
			type="info"
			showIcon
			className="mb-3"
			message={t('strategy.templates.scheduleLaunch.verifyingPermission', '正在验证交易权限…')}
		/>
	) : null;

	const noAccountBanner = accountOptions.length === 0 ? (
		<Alert
			type="warning"
			showIcon
			icon={<ExclamationCircleOutlined />}
			className="mb-3"
			message={t(
				'strategy.templates.scheduleLaunch.noAccountTitle',
				'还没有可用的交易账号',
			)}
			description={
				<div>
					{t(
						'strategy.templates.scheduleLaunch.noAccountBody',
						'请先在"账户管理"中添加并绑定 MT4/MT5 账号，账号联机成功后才能上线调度。',
					)}
					<div className="mt-2">
						<Button
							size="small"
							type="primary"
							onClick={() => {
								window.open('/accounts/bind', '_blank');
							}}
						>
							{t('strategy.templates.scheduleLaunch.actions.addAccount', '去添加交易账号')}
						</Button>
					</div>
				</div>
			}
		/>
	) : null;

	return (
		<>
			{noAccountBanner}
			{investorBanner}
			<Form
				form={form}
				layout="vertical"
				disabled={disabled || accountOptions.length === 0}
				onFinish={() => void handleFinish()}
			>
				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.scheduleName', '调度名称')}
					name="scheduleName"
					rules={[
						{ required: false },
						{ max: 100, message: t('strategy.templates.scheduleLaunch.form.scheduleNameMax', '名称长度需在 100 字以内') },
					]}
				>
					<Input placeholder={t('strategy.templates.scheduleLaunch.form.scheduleNamePlaceholder', '可选，用于在调度列表中区分')} />
				</Form.Item>
				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.account', '交易账户')}
					name="accountId"
					rules={[{ required: true, message: t('common.required', '必填') }]}
				>
					<Select
						showSearch
						placeholder={t('strategy.templates.scheduleLaunch.form.accountPlaceholder', '选择账户')}
						onChange={(v) => {
							const id = String(v || '');
							setSelectedAccountId(id);
							// 切换账户后重置 symbol 并通知父组件按该账户重拉品种列表。
							form.setFieldsValue({ symbol: '' });
							void onAccountChange?.(id);
						}}
						// 由于 label 现在是 ReactNode，用 optionLabelProp + filterOption 做搜索。
						optionLabelProp="labelText"
						filterOption={(input, option) => {
							const labelText = String((option as any)?.labelText || '').toLowerCase();
							return labelText.includes(input.toLowerCase());
						}}
						options={accountOptions.map((a) => ({
							value: a.id,
							labelText: a.label + (a.isInvestor ? ' [投资者只读]' : ''),
							label: (
								<Space>
									<span>{a.label}</span>
									{a.isInvestor && (
										<Tag color="red">
											{t('strategy.templates.scheduleLaunch.form.investorTag', '投资者只读')}
										</Tag>
									)}
								</Space>
							),
						}))}
					/>
				</Form.Item>

				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.symbol', '交易品种')}
					name="symbol"
					rules={[{ required: true, message: t('common.required', '必填') }]}
				>
					<Select
						showSearch
						allowClear
						loading={symbolsLoading}
						placeholder={
							symbols.length === 0 && !symbolsLoading
								? t('strategy.templates.scheduleLaunch.form.symbolPlaceholderEmpty', '请先选择账户')
								: t('strategy.templates.scheduleLaunch.form.symbolPlaceholder', '搜索品种，如 EURUSD')
						}
						options={symbols}
						optionFilterProp="label"
						notFoundContent={symbolsLoading ? t('common.loading', '加载中...') : null}
					/>
				</Form.Item>

				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.timeframe', '周期')}
					name="timeframe"
					rules={[{ required: true, message: t('common.required', '必填') }]}
				>
					<Select
						options={['M1', 'M5', 'M15', 'M30', 'H1', 'H4', 'D1'].map((tf) => ({
							value: tf,
							label: tf,
						}))}
					/>
				</Form.Item>

				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.scheduleType', '调度类型')}
					name="scheduleType"
					rules={[{ required: true }]}
				>
					<Select
						options={[
							{ value: 'interval', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.interval', '固定间隔') },
							{ value: 'kline_close', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.klineClose', 'K线收盘触发') },
							{ value: 'hf_quote', label: t('strategy.templates.scheduleLaunch.form.scheduleTypes.hfQuote', '逐笔报价（高频）') },
						]}
					/>
				</Form.Item>

				{watchedScheduleType === 'interval' ? (
					<Form.Item
						label={
							<Tooltip title={t('strategy.templates.scheduleLaunch.form.intervalMsTip', '策略重新评估的周期，单位 ms。默认 5 分钟 = 300000')}>
								<span>{t('strategy.templates.scheduleLaunch.form.intervalMs', '间隔（ms）')}</span>
							</Tooltip>
						}
						name="intervalMs"
						rules={[{ required: true, type: 'number', min: 1000, message: '>= 1000' }]}
					>
						<InputNumber style={{ width: '100%' }} min={1000} step={1000} />
					</Form.Item>
				) : null}

				{watchedScheduleType === 'hf_quote' ? (
					<Form.Item
						label={
							<Tooltip title={t('strategy.templates.scheduleLaunch.form.hfCooldownMsTip', '逐笔报价模式下连续两次 evaluate 的最短间隔，避免算力浪费。')}>
								<span>{t('strategy.templates.scheduleLaunch.form.hfCooldownMs', '冷却时间（ms）')}</span>
							</Tooltip>
						}
						name="hfCooldownMs"
						rules={[{ required: true, type: 'number', min: 100, message: '>= 100' }]}
					>
						<InputNumber style={{ width: '100%' }} min={100} step={100} />
					</Form.Item>
				) : null}

				<Divider orientation="left" plain>
					{t('strategy.templates.scheduleLaunch.form.riskSection', '风控参数（可选）')}
				</Divider>

				<Form.Item
					label={
						<Tooltip title={t('strategy.templates.scheduleLaunch.form.defaultVolumeTip', '策略信号里 volume=0 时默认下单量。手数单位。')}>
							<span>{t('strategy.templates.scheduleLaunch.form.defaultVolume', '默认手数')}</span>
						</Tooltip>
					}
					name="defaultVolume"
				>
					<InputNumber style={{ width: '100%' }} min={0} step={0.01} placeholder="0.01" />
				</Form.Item>

				<Form.Item
					label={
						<Tooltip title={t('strategy.templates.scheduleLaunch.form.maxPositionsTip', '同一品种上允许同时持有的最多持仓数；达到后本次信号跳过。')}>
							<span>{t('strategy.templates.scheduleLaunch.form.maxPositions', '最大持仓数')}</span>
						</Tooltip>
					}
					name="maxPositions"
				>
					<InputNumber style={{ width: '100%' }} min={1} step={1} placeholder="不限" />
				</Form.Item>

				<Space style={{ width: '100%' }} size="large">
					<Form.Item
						label={
							<Tooltip title={t('strategy.templates.scheduleLaunch.form.stopLossOffsetTip', '策略信号没给 SL 时使用；单位是价格（不是点）。')}>
								<span>{t('strategy.templates.scheduleLaunch.form.stopLossOffset', '止损距离（价格）')}</span>
							</Tooltip>
						}
						name="stopLossPriceOffset"
						style={{ flex: 1 }}
					>
						<InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0020" />
					</Form.Item>
					<Form.Item
						label={
							<Tooltip title={t('strategy.templates.scheduleLaunch.form.takeProfitOffsetTip', '同上，止盈。')}>
								<span>{t('strategy.templates.scheduleLaunch.form.takeProfitOffset', '止盈距离（价格）')}</span>
							</Tooltip>
						}
						name="takeProfitPriceOffset"
						style={{ flex: 1 }}
					>
						<InputNumber style={{ width: '100%' }} min={0} step={0.0001} placeholder="0.0040" />
					</Form.Item>
				</Space>

				<Form.Item
					label={
						<Tooltip title={t('strategy.templates.scheduleLaunch.form.maxDrawdownPctTip', '自峰值权益的最大回撤比例，0.2 = 20%；触发后调度自动停用。')}>
							<span>{t('strategy.templates.scheduleLaunch.form.maxDrawdownPct', '最大回撤比例（0~1）')}</span>
						</Tooltip>
					}
					name="maxDrawdownPct"
					rules={[{ type: 'number', min: 0, max: 1 }]}
				>
					<InputNumber style={{ width: '100%' }} min={0} max={1} step={0.01} placeholder="0.2" />
				</Form.Item>

				{requiredParams.length > 0 && onParamValuesChange ? (
					<>
						<Divider orientation="left" plain>
							{t('strategy.templates.scheduleLaunch.form.strategyParamsSection', '策略参数')}
						</Divider>
						<RequiredParamsForm
							parameters={requiredParams}
							values={paramValues}
							onChange={onParamValuesChange}
						/>
					</>
				) : null}

				<Divider />

				<Form.Item
					label={t('strategy.templates.scheduleLaunch.form.enableAfterCreate', '创建后立即启用')}
					name="enableAfterCreate"
					valuePropName="checked"
				>
					<Switch />
				</Form.Item>

				<Form.Item>
					<Button
						type="primary"
						htmlType="submit"
						loading={submitting}
						block
						disabled={disabled || (tradePermission.verified && tradePermission.isInvestor)}
					>
						{t('strategy.templates.scheduleLaunch.actions.create', '创建调度')}
					</Button>
				</Form.Item>
			</Form>

			<TradePasswordModal
				open={passwordModalOpen}
				accountId={selectedAccountId}
				onCancel={() => setPasswordModalOpen(false)}
				onSuccess={(res) => {
					setPasswordModalOpen(false);
					setTradePermission({
						loading: false,
						verified: true,
						hasTradePermission: res.hasTradePermission,
						isInvestor: res.isInvestor,
						message: res.message,
					});
				}}
			/>
		</>
	);
};

// ------------------------------------------------------------------
// 小子组件：让用户填新密码，提交后后端做一次 Connect 测试。
// ------------------------------------------------------------------

type TradePasswordModalProps = {
	open: boolean;
	accountId: string;
	onCancel: () => void;
	onSuccess: (res: { hasTradePermission: boolean; isInvestor: boolean; message: string }) => void;
};

const TradePasswordModal: React.FC<TradePasswordModalProps> = ({ open, accountId, onCancel, onSuccess }) => {
	const { t } = useTranslation();
	const [password, setPassword] = useState('');
	const [submitting, setSubmitting] = useState(false);

	useEffect(() => {
		if (!open) {
			setPassword('');
			setSubmitting(false);
		}
	}, [open]);

	const handleSubmit = async () => {
		if (!password || !accountId) return;
		setSubmitting(true);
		try {
			const res = await accountApi.updateTradingPassword(accountId, password);
			if (!res.success) {
				message.error(
					res.message ||
						t('strategy.templates.scheduleLaunch.updatePasswordFailed', '密码验证失败，请检查是否正确'),
				);
				return;
			}
			if (res.isInvestor) {
				message.warning(
					t(
						'strategy.templates.scheduleLaunch.updatePasswordStillInvestor',
						'登录成功，但该账户仍是投资者只读模式，无法下单。',
					),
				);
			} else {
				message.success(
					t('strategy.templates.scheduleLaunch.updatePasswordOk', '交易密码已更新，账户具备交易权限'),
				);
			}
			onSuccess(res);
		} catch (e: any) {
			message.error(String(e?.message || e));
		} finally {
			setSubmitting(false);
		}
	};

	return (
		<Modal
			title={t('strategy.templates.scheduleLaunch.updatePasswordTitle', '填写交易密码')}
			open={open}
			onCancel={onCancel}
			onOk={() => void handleSubmit()}
			confirmLoading={submitting}
			okText={t('common.confirm', '确认')}
			cancelText={t('common.cancel', '取消')}
			destroyOnClose
		>
			<div className="text-sm text-gray-600 mb-3">
				{t(
					'strategy.templates.scheduleLaunch.updatePasswordHint',
					'后端会用新密码做一次 Connect 测试，成功后覆盖当前存储的密码。MT5 账户会同时识别出是否为投资者模式。',
				)}
			</div>
			<Input.Password
				autoFocus
				placeholder={t('strategy.templates.scheduleLaunch.newPasswordPlaceholder', '新的交易密码')}
				value={password}
				onChange={(e) => setPassword(e.target.value)}
				onPressEnter={() => void handleSubmit()}
			/>
		</Modal>
	);
};
