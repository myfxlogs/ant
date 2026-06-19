import { useEffect, useMemo, useState } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next'
import { SCHEDULE_LAUNCH_ERROR_INVESTOR_ACCOUNT_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

;
import { accountApi } from '@/client/account';
import type { RequiredParamSpec } from '@/client/codeAssist';
import { DEFAULT_TIMEFRAME } from '@/constants/timeframes';
import { isTradingAccountEnabled } from '@/utils/accountStatus';
import { buildParametersFromForm } from './StrategyScheduleParams';
import type { ScheduleLaunchFormValues } from './StrategyTemplateScheduleLaunchForm';

interface AccountOption { id: string; label: string; mtType: string; isInvestor: boolean; isDisabled: boolean; }

function normalizeAccounts(accounts: any[] | undefined): AccountOption[] {
  if (!Array.isArray(accounts)) return [];
  return accounts.filter(isTradingAccountEnabled).map((a) => {
    const id = String(a?.id || ''); const login = String(a?.login || '').trim();
    const alias = String(a?.alias || '').trim(); const mtType = String(a?.mtType || a?.mt_type || '').toUpperCase();
    const isInvestor = Boolean(a?.isInvestor ?? a?.is_investor);
    const base = `${login || id} (${mtType})`;
    const label = alias && alias !== login ? `${base} · ${alias}` : base;
    return { id, label, mtType, isInvestor, isDisabled: Boolean(a?.isDisabled ?? a?.is_disabled) };
  }).filter((o) => o.id);
}

interface UseScheduleLaunchFormParams {
  open: boolean;
  accounts: any[];
  defaults?: Partial<ScheduleLaunchFormValues>;
  requiredParams?: RequiredParamSpec[];
  paramValues?: Record<string, unknown>;
  onParamValuesChange?: (v: Record<string, unknown>) => void;
  onAccountChange?: (accountId: string) => void | Promise<void>;
}

export function useScheduleLaunchForm({
  open, accounts, defaults, requiredParams = [], paramValues = {},
  onParamValuesChange, onAccountChange,
}: UseScheduleLaunchFormParams) {
  const { t } = useTranslation();
  const [form] = Form.useForm<ScheduleLaunchFormValues>();
  const accountOptions = useMemo(() => normalizeAccounts(accounts), [accounts]);
  const [selectedAccountId, setSelectedAccountId] = useState<string>(String(defaults?.accountId || accountOptions[0]?.id || ''));
  const [tradePermission, setTradePermission] = useState<{ loading: boolean; hasTradePermission: boolean; isInvestor: boolean; verified: boolean; message: string }>({ loading: false, hasTradePermission: false, isInvestor: false, verified: false, message: '' });
  const [passwordModalOpen, setPasswordModalOpen] = useState<boolean>(false);

  const selectedAccount = useMemo(() => accountOptions.find((a) => a.id === selectedAccountId) || null, [accountOptions, selectedAccountId]);

  useEffect(() => {
    if (!open) return;
    const initial: Partial<ScheduleLaunchFormValues> = {
      scheduleName: defaults?.scheduleName || '', accountId: String(defaults?.accountId || accountOptions[0]?.id || ''),
      symbol: defaults?.symbol || '', timeframe: defaults?.timeframe || DEFAULT_TIMEFRAME,
      scheduleType: defaults?.scheduleType || 'kline_close',
      intervalMs: defaults?.intervalMs || 300_000, hfCooldownMs: defaults?.hfCooldownMs || 1_000,
      defaultVolume: defaults?.defaultVolume, maxPositions: defaults?.maxPositions,
      stopLossPriceOffset: defaults?.stopLossPriceOffset, takeProfitPriceOffset: defaults?.takeProfitPriceOffset,
      maxDrawdownPct: defaults?.maxDrawdownPct, enableAfterCreate: defaults?.enableAfterCreate ?? true,
    };
    form.setFieldsValue(initial as Record<string, unknown>);
    const initialAccountId = String(initial.accountId || '');
    setSelectedAccountId(initialAccountId);
    if (initialAccountId) void onAccountChange?.(initialAccountId);
  }, [open, accountOptions.length]);

  useEffect(() => {
    if (!open || !selectedAccountId) {
      setTradePermission({ loading: false, hasTradePermission: false, isInvestor: false, verified: false, message: '' });
      return;
    }
    const local = selectedAccount;
    setTradePermission((p) => ({ ...p, isInvestor: Boolean(local?.isInvestor), hasTradePermission: !(local?.isInvestor), verified: false, message: '' }));
    let cancelled = false;
    void (async () => {
      setTradePermission((p) => ({ ...p, loading: true }));
      try {
        const r = await accountApi.verifyTradePermission(selectedAccountId);
        if (cancelled) return;
        setTradePermission({ loading: false, hasTradePermission: r.hasTradePermission, isInvestor: r.isInvestor, verified: r.verified, message: r.message });
      } catch (e: unknown) {
        if (cancelled) return;
        setTradePermission({ loading: false, hasTradePermission: false, isInvestor: Boolean(local?.isInvestor), verified: false, message: String((e as any)?.message || e || '') });
      }
    })();
    return () => { cancelled = true; };
  }, [open, selectedAccountId]);

  const watchedScheduleType = Form.useWatch('scheduleType', form);

  const buildParameters = (): Record<string, string> => {
    const v = form.getFieldsValue(true);
    const out = buildParametersFromForm(v);
    for (const [key, raw] of Object.entries(paramValues || {})) {
      if (!key || raw === undefined || raw === null || raw === '') continue;
      out[key] = String(raw);
    }
    return out;
  };

  const handleFinish = async (onSubmit: any) => {
    try {
      const v = (await form.validateFields()) as ScheduleLaunchFormValues;
      if (tradePermission.verified && tradePermission.isInvestor) {
        message.error(t(SCHEDULE_LAUNCH_ERROR_INVESTOR_ACCOUNT_KEY, '所选账户是投资者只读模式，请先填写交易密码'));
        return;
      }
      const missingParams = (requiredParams || []).filter((p) => p.required).filter((p) => {
        const value = paramValues[p.key];
        return value === undefined || value === null || value === '';
      });
      if (missingParams.length > 0) {
        message.error(t('strategy.codeAssist.fillRequiredParams', { defaultValue: 'Please fill the required parameters: {{keys}}', keys: missingParams.map((m) => m.key).join(', ') }));
        return;
      }
      onSubmit({ form: v, buildParameters });
    } catch { /* validate failed */ }
  };

  return {
    form, accountOptions, selectedAccount, selectedAccountId, setSelectedAccountId,
    tradePermission, setTradePermission, passwordModalOpen, setPasswordModalOpen,
    watchedScheduleType, buildParameters, handleFinish,
  };
}

export { normalizeAccounts };
