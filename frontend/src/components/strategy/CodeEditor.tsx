import React, { useState, useEffect, useCallback } from 'react';
import { Card, Button, Select, Input, message, Alert, Space, Form } from 'antd';
import { PlayCircleOutlined, CheckCircleOutlined, CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import { accountApi } from '@/client/account';
import { strategyTemplateApi } from '@/client/strategy';
import { marketApi } from '@/client/market';
import { useAIStore } from '@/stores/aiStore';
import { copyToClipboard } from '@/utils/clipboard';
import { getErrorMessage } from '@/utils/error';
import SaveTemplateModal from './SaveTemplateModal';
import type {
  Account,
  CodeEditorProps,
  PreviewResult,
  StrategyTemplate,
} from './CodeEditor.types';

const { TextArea } = Input;

const CodeEditor: React.FC<CodeEditorProps> = ({ code: controlledCode, onCodeChange, initialCode }) => {
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
      const data = await accountApi.list() as any[];
      const accountList = (data || []).map((a: any) => ({
        id: a.id,
        login: a.login,
		mtType: a.mtType || a.mtType,
		isDisabled: !!a.isDisabled,
      }));
      setAccounts(accountList);
    } catch (error) {
      message.error(getErrorMessage(error, '加载账户列表失败'));
    }
  }, []);

  const loadSymbols = async (accountId: string) => {
    if (!accountId) {
      setSymbols([]);
      setSymbol('');
      return;
    }

    setSymbolsLoading(true);
    try {
      const list = await marketApi.getSymbols(accountId);
      const seen = new Set<string>();
      const opts = (list || [])
        .map((s) => String((s as any)?.symbol || '').trim())
        .filter((v) => v)
        .filter((v) => {
          if (seen.has(v)) return false;
          seen.add(v);
          return true;
        })
        .map((v) => ({
          value: v,
          label: v,
        }));
      setSymbols(opts);
      if (opts.length > 0) {
        const exists = !!opts.find((o) => o.value === symbol);
        if (!symbol || !exists) {
          setSymbol(opts[0].value);
        }
      }
    } catch (error) {
      setSymbols([]);
      setSymbol('');
      message.error(getErrorMessage(error, '加载品种失败'));
    } finally {
      setSymbolsLoading(false);
    }
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
    (template: StrategyTemplate) => {
      setCode(template.code);
      setValidationResult(null);
      setPreviewResult(null);
    },
    [setCode]
  );

  useEffect(() => {
    loadTemplates();
    loadAccounts();
  }, [loadTemplates, loadAccounts]);

  useEffect(() => {
    if (templates.length > 0) {
      if (controlledCode === undefined && !initialCode) {
        handleTemplateSelect(templates[0]);
      }
    }
  }, [templates, controlledCode, initialCode, handleTemplateSelect]);

  const handleValidate = async () => {
    if (!code.trim()) {
      message.warning(t('strategy.codeEditor.messages.enterCode'));
      return;
    }

    setValidating(true);
    setValidationResult(null);

    try {
      const data = await pythonStrategyApi.validate(code);
      setValidationResult(data);
      if (data?.valid) {
        message.success(t('strategy.codeEditor.messages.validateOk'));
      } else {
        message.error(t('strategy.codeEditor.messages.validateFailed'));
      }
    } catch (_error) {
      message.error(t('strategy.codeEditor.messages.validateError'));
    } finally {
      setValidating(false);
    }
  };

  const handlePreview = async () => {
    if (!code.trim()) {
      message.warning(t('strategy.codeEditor.messages.enterCode'));
      return;
    }
    if (!selectedAccount) {
      message.warning(t('strategy.codeEditor.messages.selectAccount'));
      return;
    }

    setLoading(true);
    setPreviewResult(null);

    try {
      const data = await pythonStrategyApi.execute({
        code,
        accountId: selectedAccount,
        symbol,
        timeframe,
      });
      setPreviewResult(data);
      if (data?.success) {
        message.success(t('strategy.codeEditor.messages.previewOk'));
      } else {
        message.error(data?.error || t('strategy.codeEditor.messages.execFailed'));
      }
    } catch (_error) {
      message.error(t('strategy.codeEditor.messages.previewFailed'));
    } finally {
      setLoading(false);
    }
  };

  const openSaveTemplate = () => {
    if (!code.trim()) {
      message.warning(t('strategy.codeEditor.messages.enterCode'));
      return;
    }
    setSaveTemplateOpen(true);
  };

  const handleSaveTemplate = async () => {
    try {
      const values = await saveTemplateForm.validateFields();
      setSaveTemplateLoading(true);

      await strategyTemplateApi.create({
        name: values.name,
        description: values.description || '',
        code,
        parameters: [],
        isPublic: false,
        tags: [],
      });

      message.success(t('strategy.codeEditor.messages.savedAsTemplate'));
      setSaveTemplateOpen(false);
    } catch (_e) {
    } finally {
      setSaveTemplateLoading(false);
    }
  };

  const copyCode = async () => {
    const ok = await copyToClipboard(code);
    if (ok) {
      message.success(t('strategy.codeEditor.messages.copied'));
      return;
    }
    message.error(t('strategy.codeEditor.messages.copyFailed'));
  };

  const sendToAIWithContext = (title: string, details: string) => {
    const payload = [
      t('strategy.codeEditor.aiPrompt.intro'),
      '',
      t('strategy.codeEditor.aiPrompt.problem', { title }),
      '',
      t('strategy.codeEditor.aiPrompt.currentCodeTitle'),
      t('strategy.codeEditor.aiPrompt.pythonFenceStart'),
      code || '',
      t('strategy.codeEditor.aiPrompt.fenceEnd'),
      '',
      t('strategy.codeEditor.aiPrompt.outputTitle'),
      details,
      '',
      t('strategy.codeEditor.aiPrompt.outro'),
    ].join('\n');

    sendAIMessage(payload, selectedAccount || undefined);
  };

  return (
    <div className="p-6">
      <Card title={t('strategy.codeEditor.title')} className="mb-4">
        <div className="mb-4 flex gap-4">
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.account')}</label>
            <Select
              style={{ width: '100%' }}
              value={selectedAccount}
              onChange={(v) => {
                setSelectedAccount(v);
                loadSymbols(v);
              }}
              placeholder={t('strategy.codeEditor.placeholders.selectAccount')}
            >
              {accounts.map((account) => (
                <Select.Option key={account.id} value={account.id} disabled={!!account.isDisabled}>
                  {account.login} ({account.mtType}){account.isDisabled ? t('strategy.codeEditor.labels.disabledSuffix') : ''}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.symbol')}</label>
            <Select
              showSearch
              allowClear
              loading={symbolsLoading}
              style={{ width: '100%' }}
              value={symbol}
              onChange={(v) => setSymbol(v || '')}
              placeholder={
                !selectedAccount
                  ? t('strategy.codeEditor.placeholders.selectAccountFirst')
                  : symbolsLoading
                    ? t('strategy.codeEditor.placeholders.loadingSymbols')
                    : t('strategy.codeEditor.placeholders.selectSymbol')
              }
              options={symbols}
              disabled={!selectedAccount || symbolsLoading}
              optionFilterProp="label"
              filterOption={(input, option) => {
                const key = String((option as any)?.value || '').toLowerCase();
                const label = String((option as any)?.label || '').toLowerCase();
                const q = input.toLowerCase();
                return key.includes(q) || label.includes(q);
              }}
              notFoundContent={
                !selectedAccount
                  ? null
                  : symbolsLoading
                    ? t('strategy.codeEditor.placeholders.loadingSymbols')
                    : t('strategy.codeEditor.placeholders.noSymbols')
              }
            />
          </div>
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.timeframe')}</label>
            <Select
              style={{ width: '100%' }}
              value={timeframe}
              onChange={setTimeframe}
            >
              <Select.Option value="M1">M1</Select.Option>
              <Select.Option value="M5">M5</Select.Option>
              <Select.Option value="M15">M15</Select.Option>
              <Select.Option value="M30">M30</Select.Option>
              <Select.Option value="H1">H1</Select.Option>
              <Select.Option value="H4">H4</Select.Option>
              <Select.Option value="D1">D1</Select.Option>
            </Select>
          </div>
        </div>

        <div className="mb-4">
          <div className="flex justify-between items-center mb-2">
            <span className="text-sm">{t('strategy.codeEditor.labels.code')}</span>
            <Button size="small" icon={<CopyOutlined />} onClick={copyCode}>
              {t('strategy.codeEditor.actions.copy')}
            </Button>
          </div>
          <TextArea
            rows={15}
            value={code}
            onChange={(e) => setCode(e.target.value)}
            placeholder={t('strategy.codeEditor.placeholders.code')}
            style={{ fontFamily: 'monospace' }}
          />
        </div>

        <Space>
          <Button icon={<CheckCircleOutlined />} onClick={handleValidate} loading={validating}>
            {t('strategy.codeEditor.actions.validate')}
          </Button>
          <Button type="primary" icon={<PlayCircleOutlined />} onClick={handlePreview} loading={loading}>
            {t('strategy.codeEditor.actions.preview')}
          </Button>
          <Button onClick={openSaveTemplate}>
            {t('strategy.codeEditor.actions.saveAsTemplate')}
          </Button>
        </Space>

		<div className="mt-2 text-xs text-gray-500">
			{t('strategy.codeEditor.hints.previewInfo')}
		</div>
      </Card>

      {validationResult && (
        <Card title={t('strategy.codeEditor.cards.validationResult')} className="mb-4">
          <div className="mb-3">
            <Button
              size="small"
              onClick={() => {
                const details = JSON.stringify(
                  {
                    valid: !!validationResult?.valid,
                    errors: validationResult?.errors || [],
                    warnings: validationResult?.warnings || [],
                  },
                  null,
                  2,
                );
                sendToAIWithContext(t('strategy.codeEditor.actions.sendToAIFixTitleValidate'), details);
              }}
            >
              {t('strategy.codeEditor.actions.sendToAI')}
            </Button>
          </div>
          {validationResult.valid ? (
            <Alert title={t('strategy.codeEditor.messages.validateOk')} type="success" />
          ) : (
            <div>
              {validationResult.errors.map((err, i) => (
                <Alert key={i} title={err} type="error" className="mb-2" />
              ))}
              {validationResult.warnings.map((warn, i) => (
                <Alert key={i} title={warn} type="warning" className="mb-2" />
              ))}
            </div>
          )}
        </Card>
      )}

      {previewResult && (
        <Card title={t('strategy.codeEditor.cards.previewResult')} className="mb-4">
          <div className="mb-3">
            <Button
              size="small"
              onClick={() => {
                const details = JSON.stringify(previewResult || {}, null, 2);
                sendToAIWithContext(t('strategy.codeEditor.actions.sendToAIFixTitlePreview'), details);
              }}
            >
              {t('strategy.codeEditor.actions.sendToAI')}
            </Button>
          </div>
          {previewResult.success ? (
            <Alert title={t('strategy.codeEditor.messages.previewSuccess')} type="success" />
          ) : (
            <Alert title={previewResult.error || t('strategy.codeEditor.messages.previewFailed')} type="error" />
          )}
        </Card>
      )}
      <SaveTemplateModal
        open={saveTemplateOpen}
        confirmLoading={saveTemplateLoading}
        form={saveTemplateForm}
        afterOpenChange={(open) => {
          if (!open) return;
          saveTemplateForm.setFieldsValue({
            name: '',
            description: '',
          });
        }}
        onCancel={() => setSaveTemplateOpen(false)}
        onOk={handleSaveTemplate}
      />
    </div>
  );
};

export default CodeEditor;
