import { useState, useEffect, useCallback } from 'react';
import { Form } from 'antd';
import dayjs from 'dayjs';
import { useTranslation } from 'react-i18next';
import type { RequiredParamSpec } from '@/client/codeAssist';
import type { StrategyTemplate } from '@/client/strategy';
import { type QuickRangeKey, quickRangeLabel } from '../StrategyTemplatePage.utils';

export function useBacktestModal() {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);
  const [template, setTemplate] = useState<StrategyTemplate | null>(null);
  const [requiredParams, setRequiredParams] = useState<RequiredParamSpec[]>([]);
  const [paramValues, setParamValues] = useState<Record<string, unknown>>({});
  const [quickRange, setQuickRange] = useState<QuickRangeKey>('CUSTOM');
  const watchedRange = Form.useWatch('range', form) as [dayjs.Dayjs, dayjs.Dayjs] | undefined;

  useEffect(() => {
    if (!open) return;
    form.setFieldsValue({
      title: `${dayjs().format('YYYY-MM-DD HH:mm')} ${quickRangeLabel(t, quickRange)}`,
    });
  }, [open, quickRange, form, t]);

  const openModal = useCallback((tpl: StrategyTemplate, params: RequiredParamSpec[]) => {
    setTemplate(tpl);
    setRequiredParams(params);
    setParamValues({});
    setOpen(true);
  }, []);

  const closeModal = useCallback(() => {
    setOpen(false);
    form.resetFields();
  }, [form]);

  const applyQuickRange = useCallback((key: QuickRangeKey) => {
    setQuickRange(key);
    const now = dayjs();
    let start: dayjs.Dayjs;
    switch (key) {
      case '1D': start = now.subtract(1, 'day'); break;
      case '1W': start = now.subtract(1, 'week'); break;
      case '1M': start = now.subtract(1, 'month'); break;
      case '3M': start = now.subtract(3, 'month'); break;
      default: return;
    }
    form.setFieldsValue({ range: [start, now] });
  }, [form]);

  return {
    open, setOpen,
    form, submitting, setSubmitting,
    template, requiredParams, paramValues, setParamValues,
    quickRange, setQuickRange, watchedRange,
    openModal, closeModal, applyQuickRange,
  };
}
