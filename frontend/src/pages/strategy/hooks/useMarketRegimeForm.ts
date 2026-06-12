import { useState, useEffect, useMemo, useCallback } from 'react';
import { Form } from 'antd';
import { useAccount } from '@/hooks/useAccount';
import { marketApi } from '@/client/market';
import { marketRegimeApi, type MarketRegime } from '@/client/marketRegime';

export function useMarketRegimeForm() {
  const [form] = Form.useForm();
  const [result, setResult] = useState<MarketRegime | null>(null);
  const [loading, setLoading] = useState(false);

  // Account selection — reuse workspace pattern (useAccount → activeAccounts)
  const { accounts, fetchAccounts } = useAccount();
  const activeAccounts = useMemo(
    () => (accounts || []).filter((a) => !a.isDisabled),
    [accounts],
  );

  useEffect(() => { fetchAccounts(); }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const watchedAccountId = Form.useWatch('accountId', form) as string | undefined;

  // Clear symbol + symbol cache when account changes (workspace pattern)
  const handleAccountChange = useCallback(() => {
    form.setFieldValue('symbol', '');
    marketApi.clearSymbolCache();
  }, [form]);

  const detect = useCallback(async () => {
    const values = form.getFieldsValue() as {
      accountId: string; symbol: string; timeframe: string; count: number;
    };
    setLoading(true);
    try {
      const row = await marketRegimeApi.detect(values);
      setResult(row);
      return row;
    } finally {
      setLoading(false);
    }
  }, [form]);

  return {
    form,
    activeAccounts,
    watchedAccountId,
    result,
    loading,
    handleAccountChange,
    detect,
  };
}
