import { useState, useCallback, useEffect, useRef } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next'
import { COPY_FAILED_KEY, COPY_SUCCESS_KEY, SAVE_SUCCESS_KEY, VALIDATE_BEFORE_SAVE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

;
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { codeAssistApi, type ValidateExtendedResult } from '@/client/codeAssist';

export function useStrategyCode(opts?: { onValidateResult?: (result: ValidateExtendedResult) => void }) {
  const onValidateResult = opts?.onValidateResult;
  const { t } = useTranslation();
  const [code, setCode] = useState('');
  const [lastValidatedCode, setLastValidatedCode] = useState('');
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidateExtendedResult | null>(null);
  // Track previous code to detect external changes (AI apply, template load, etc.)
  const prevCodeRef = useRef(code);

  // Reset validation state when code changes externally (AI apply / template load).
  useEffect(() => {
    if (code !== prevCodeRef.current) {
      prevCodeRef.current = code;
      setValidating(false);
      setValidationResult(null);
      setLastValidatedCode('');
    }
  }, [code]);

  const handleValidate = useCallback(async () => {
    if (!code.trim()) return;
    setValidating(true);
    try {
      const result = await codeAssistApi.validateExtended(code);
      setValidationResult(result);
      if (result.valid) setLastValidatedCode(code);
      // Backend zero-trust: push sweep dimensions + strategy directives
      // from Python backend to the tuning panel.
      if (onValidateResult) onValidateResult(result);
    } catch (e: unknown) { message.error((e as Error)?.message || 'Validation failed'); }
    finally { setValidating(false); }
  }, [code, onValidateResult]);

  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(false);
  const [loadedTemplate, setLoadedTemplate] = useState<StrategyTemplate | null>(null);

  const loadTemplates = useCallback(async () => {
    setTemplatesLoading(true);
    try { const list = await strategyApi.listTemplates(); setTemplates(list || []); }
    catch (e: unknown) { message.error((e as Error)?.message || 'Failed to load templates'); }
    finally { setTemplatesLoading(false); }
  }, []);

  const handleLoadTemplate = useCallback(async (id: string): Promise<StrategyTemplate | null> => {
    try {
      const tpl = await strategyApi.getTemplate(id);
      if (tpl?.code) setCode(tpl.code);
      if (tpl?.name) setLoadedTemplate(tpl);
      setLastValidatedCode(''); setValidationResult(null);
      return tpl;
    } catch (e: unknown) { message.error((e as Error)?.message || 'Failed to load template'); return null; }
  }, []);

  const [saveModalOpen, setSaveModalOpen] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [saveForm] = Form.useForm();
  const [lastSavedId, setLastSavedId] = useState<string | null>(null);
  const canSave = code.length > 0 && lastValidatedCode.length > 0 && code === lastValidatedCode;

  const handleSave = useCallback(async () => {
    if (!canSave) { message.warning(t(VALIDATE_BEFORE_SAVE_KEY)); return; }
    if (loadedTemplate) {
      setSaveLoading(true);
      try {
        await strategyApi.updateTemplate({ id: loadedTemplate.id, code });
        message.success(t(SAVE_SUCCESS_KEY)); loadTemplates();
      } catch (e: unknown) { message.error((e as Error)?.message || 'Save failed'); }
      finally { setSaveLoading(false); }
    } else { setSaveModalOpen(true); }
  }, [code, canSave, loadedTemplate, t, loadTemplates]);

  const handleSaveAs = useCallback(() => { saveForm.resetFields(); setSaveModalOpen(true); }, [saveForm]);
  const handleSaveModalOk = useCallback(async () => {
    try {
      const values = await saveForm.validateFields(); setSaveLoading(true);
      const tpl = await strategyApi.createTemplate({ name: values.name, description: values.description || '', code });
      if (tpl?.id) setLastSavedId(tpl.id);
      message.success(t(SAVE_SUCCESS_KEY)); setSaveModalOpen(false); loadTemplates();
    } catch (e: unknown) {
      // Ant Design validateFields rejects with errorFields, not message.
      const err = e as { message?: string; errorFields?: unknown[] };
      if (err?.errorFields?.length) return; // form validation failure — Ant Design shows inline errors
      if (err?.message) message.error(err.message);
    }
    finally { setSaveLoading(false); }
  }, [code, saveForm, t, loadTemplates]);

  const handleCopy = useCallback(() => {
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => message.success(t(COPY_SUCCESS_KEY))).catch(() => message.error(t(COPY_FAILED_KEY)));
  }, [code, t]);

  return { code, setCode, validating, validationResult, setValidationResult,
    lastValidatedCode, setLastValidatedCode, handleValidate,
    templates, templatesLoading, loadedTemplate, loadTemplates, handleLoadTemplate,
    saveModalOpen, setSaveModalOpen, saveLoading, saveForm, canSave,
    handleSave, handleSaveAs, handleSaveModalOk, handleCopy, lastSavedId };
}
