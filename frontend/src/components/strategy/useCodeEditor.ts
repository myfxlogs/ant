import { useState, useEffect, useCallback } from 'react';
import { Form, message } from 'antd';
import { useTranslation } from 'react-i18next';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { accountApi } from '@/client/account';
import { strategyTemplateApi } from '@/client/strategy';
import { marketApi } from '@/client/market';
import { useAIStore } from '@/stores/aiStore';
import { copyToClipboard } from '@/utils/clipboard';
import { getErrorMessage } from '@/utils/error';
import type { Account, CodeEditorProps, PreviewResult, StrategyTemplate } from './CodeEditor.types';

export function useCodeEditor({ code: controlledCode, onCodeChange, initialCode }: CodeEditorProps) {
  const { t } = useTranslation();
  const [codeInternal, setCodeInternal] = useState<string>(initialCode || '');
  const [accounts, setAccounts] = useState<Account[]>([]);
  const [selectedAccount, setSelectedAccount] = useState<string>('');
  const [symbol, setSymbol] = useState<string>('');
  const [symbols, setSymbols] = useState<{ value: string; label: string }[]>([]);
  const [symbolsLoading, setSymbolsLoading] = useState(false);
  const [timeframe, setTimeframe] = useState<string>('H1');
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [validating, setValidating] = useState(false);
  const [previewResult, setPreviewResult] = useState<PreviewResult | null>(null);
  const [validationResult, setValidationResult] = useState<{ valid: boolean; errors: string[]; warnings: string[] } | null>(null);

  const [saveTemplateOpen, setSaveTemplateOpen] = useState(false);
  const [saveTemplateLoading, setSaveTemplateLoading] = useState(false);
  const [saveTemplateForm] = Form.useForm();

  const sendAIMessage = useAIStore((s) => s.sendMessage);

  const code = controlledCode !== undefined ? controlledCode : codeInternal;

  const setCode = useCallback((next: string) => {
    if (controlledCode !== undefined) {
      onCodeChange?.(next);
      return;
    }
    setCodeInternal(next);
    onCodeChange?.(next);
  }, [controlledCode, onCodeChange]);

  const loadAccounts = useCallback(async () => {
    try {
      const data = await accountApi.list() as Account[];
      const accountList = (data || []).map((a: Account) => ({
        id: a.id,
        login: a.login,
        mtType: a.mtType,
        isDisabled: !!a.isDisabled,
      }));
      setAccounts(accountList);
    } catch (error) {
      message.error(getErrorMessage(error, '加载账户列表失败'));
    }
  }, []);

  const loadSymbols = async (accountId: string) => {
    if (!accountId) { setSymbols([]); setSymbol(''); return; }
    setSymbolsLoading(true);
    try {
      const list = await marketApi.getSymbols(accountId);
      const seen = new Set<string>();
      const opts = (list || [])
        .map((s: { symbol?: unknown }) => String(s?.symbol || '').trim())
        .filter((v) => v)
        .filter((v) => { if (seen.has(v)) return false; seen.add(v); return true; })
        .map((v) => ({ value: v, label: v }));
      setSymbols(opts);
      if (opts.length > 0) {
        const exists = !!opts.find((o) => o.value === symbol);
        if (!symbol || !exists) { setSymbol(opts[0].value); }
      }
    } catch (error) {
      setSymbols([]); setSymbol('');
      message.error(getErrorMessage(error, '加载品种失败'));
    } finally { setSymbolsLoading(false); }
  };

  const loadTemplates = useCallback(async () => {
    try {
      const data = await pythonStrategyApi.getTemplates();
      if (data && Array.isArray(data)) {
        setTemplates(data);
        if (data.length > 0 && controlledCode === undefined && !initialCode) {
          setCodeInternal(data[0].code);
        }
      }
    } catch (error) {
      message.error(getErrorMessage(error, '加载策略模板失败'));
    }
  }, [controlledCode, initialCode]);

  const handleTemplateSelect = useCallback(
    (template: StrategyTemplate) => { setCode(template.code); setValidationResult(null); setPreviewResult(null); },
    [setCode],
  );

  useEffect(() => { loadTemplates(); loadAccounts(); }, [loadTemplates, loadAccounts]);

  useEffect(() => {
    if (templates.length > 0 && controlledCode === undefined && !initialCode) {
      handleTemplateSelect(templates[0]);
    }
  }, [templates, controlledCode, initialCode, handleTemplateSelect]);

  const handleValidate = async () => {
    if (!code.trim()) { message.warning(t('strategy.codeEditor.messages.enterCode')); return; }
    setValidating(true); setValidationResult(null);
    try {
      const data = await pythonStrategyApi.validate(code);
      setValidationResult(data);
      if (data?.valid) message.success(t('strategy.codeEditor.messages.validateOk'));
      else message.error(t('strategy.codeEditor.messages.validateFailed'));
    } catch { message.error(t('strategy.codeEditor.messages.validateError')); }
    finally { setValidating(false); }
  };

  const handlePreview = async () => {
    if (!code.trim()) { message.warning(t('strategy.codeEditor.messages.enterCode')); return; }
    if (!selectedAccount) { message.warning(t('strategy.codeEditor.messages.selectAccount')); return; }
    setLoading(true); setPreviewResult(null);
    try {
      const data = await pythonStrategyApi.execute({ code, accountId: selectedAccount, symbol, timeframe });
      setPreviewResult(data);
      if (data?.success) message.success(t('strategy.codeEditor.messages.previewOk'));
      else message.error(data?.error || t('strategy.codeEditor.messages.execFailed'));
    } catch { message.error(t('strategy.codeEditor.messages.previewFailed')); }
    finally { setLoading(false); }
  };

  const openSaveTemplate = () => {
    if (!code.trim()) { message.warning(t('strategy.codeEditor.messages.enterCode')); return; }
    setSaveTemplateOpen(true);
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await saveTemplateForm.validateFields();
      setSaveTemplateLoading(true);
      await strategyTemplateApi.create({
        name: values.name, description: values.description || '', code,
        parameters: [], isPublic: false, tags: [],
      });
      message.success(t('strategy.codeEditor.messages.savedAsTemplate'));
      setSaveTemplateOpen(false);
    } catch { } finally { setSaveTemplateLoading(false); }
  };

  const copyCode = async () => {
    const ok = await copyToClipboard(code);
    if (ok) message.success(t('strategy.codeEditor.messages.copied'));
    else message.error(t('strategy.codeEditor.messages.copyFailed'));
  };

  const sendToAIWithContext = (title: string, details: string) => {
    const payload = [
      t('strategy.codeEditor.aiPrompt.intro'), '',
      t('strategy.codeEditor.aiPrompt.problem', { title }), '',
      t('strategy.codeEditor.aiPrompt.currentCodeTitle'),
      t('strategy.codeEditor.aiPrompt.pythonFenceStart'),
      code || '',
      t('strategy.codeEditor.aiPrompt.fenceEnd'), '',
      t('strategy.codeEditor.aiPrompt.outputTitle'),
      details, '',
      t('strategy.codeEditor.aiPrompt.outro'),
    ].join('\n');
    sendAIMessage(payload, selectedAccount || undefined);
  };

  return {
    code, setCode, accounts, selectedAccount, setSelectedAccount,
    symbol, setSymbol, symbols, symbolsLoading, timeframe, setTimeframe,
    loading, validating, previewResult, validationResult,
    saveTemplateOpen, saveTemplateLoading, saveTemplateForm,
    loadSymbols, handleValidate, handlePreview, openSaveTemplate,
    handleSaveTemplate, copyCode, sendToAIWithContext,
    setSaveTemplateOpen,
  };
}
