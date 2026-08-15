import { useState, useEffect, useCallback } from 'react';
import { strategyTemplateApi } from '@/client/strategy-schedules';
import { codeAssistApi } from '@/client/codeAssist';

export interface ExtractedParam {
  name: string;
  type: string;
  default: string;
  label: string;
}

interface UseStrategyParamsOptions {
  open: boolean;
  templateId: string | undefined;
  initialValues?: Record<string, string>;
}

interface UseStrategyParamsResult {
  extractedParams: ExtractedParam[];
  strategyParamValues: Record<string, string>;
  setStrategyParamValues: React.Dispatch<React.SetStateAction<Record<string, string>>>;
  paramsLoading: boolean;
}

export function useStrategyParams({
  open,
  templateId,
  initialValues,
}: UseStrategyParamsOptions): UseStrategyParamsResult {
  const [extractedParams, setExtractedParams] = useState<ExtractedParam[]>([]);
  const [strategyParamValues, setStrategyParamValues] = useState<Record<string, string>>({});
  const [paramsLoading, setParamsLoading] = useState(false);

  const loadStrategyParams = useCallback(async (tplId: string) => {
    if (!tplId) return;
    setParamsLoading(true);
    try {
      const tpl = await strategyTemplateApi.get(tplId);
      const code = String(tpl?.code || '');
      if (!code) { setParamsLoading(false); return; }
      const result = await codeAssistApi.validateExtended(code);
      if (result.valid && result.parameterEntries) {
        setExtractedParams(result.parameterEntries.map(e => ({
          name: e.name, type: e.type, default: e.default, label: e.label || '',
        })));
      }
    } catch { /* template not found or validation failed */ }
    setParamsLoading(false);
  }, []);

  // Load effect — only fires when modal opens or template changes.
  // E1 lesson: deps are [open, templateId] — NOT extractedParams (infinite loop).
  useEffect(() => {
    if (open && templateId) {
      if (initialValues) {
        const vals: Record<string, string> = {};
        for (const [k, v] of Object.entries(initialValues)) {
          if (!k.startsWith('__risk.') && !k.startsWith('__schedule.')) vals[k] = String(v);
        }
        setStrategyParamValues(vals);
      } else {
        setStrategyParamValues({});
      }
      void loadStrategyParams(templateId);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open, templateId, loadStrategyParams]);

  // Merge effect — when extracted params arrive, fill in defaults for keys
  // the user hasn't set yet. Uses functional update to preserve user edits.
  useEffect(() => {
    if (extractedParams.length === 0) return;
    setStrategyParamValues(prev => {
      const next = { ...prev };
      for (const p of extractedParams) {
        if (!(p.name in next)) next[p.name] = p.default;
      }
      return next;
    });
  }, [extractedParams]);

  // Cleanup on close.
  useEffect(() => {
    if (!open) {
      setExtractedParams([]);
      setStrategyParamValues({});
    }
  }, [open]);

  return {
    extractedParams,
    strategyParamValues,
    setStrategyParamValues,
    paramsLoading,
  };
}
