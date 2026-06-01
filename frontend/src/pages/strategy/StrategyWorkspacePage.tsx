import { useState, useEffect, useCallback, Suspense, lazy } from 'react';
import { Tabs, Button, Collapse, message, Spin, Form } from 'antd';
import {
  MenuFoldOutlined, MenuUnfoldOutlined, RobotOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useAccount } from '@/hooks/useAccount';
import { strategyApi, type StrategyTemplate } from '@/client/strategy';
import { codeAssistApi, type ValidateExtendedResult } from '@/client/codeAssist';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import WorkspaceCodePanel from './components/workspace/WorkspaceCodePanel';
import WorkspaceChartTab from './components/workspace/WorkspaceChartTab';
import WorkspaceBacktestPanel from './components/workspace/WorkspaceBacktestPanel';
import WorkspaceTemplateManager from './components/workspace/WorkspaceTemplateManager';
import { AICodeReviseChat, CodeExplainPanel } from '@/components/strategy/CodeAssist';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

type BacktestStatus = 'idle' | 'running' | 'completed' | 'error';

export default function StrategyWorkspacePage() {
  const { t } = useTranslation();

  // Code
  const [code, setCode] = useState('');
  const [lastValidatedCode, setLastValidatedCode] = useState('');

  // Account / Symbol / Timeframe
  const { accounts: allAccounts, fetchAccounts } = useAccount();
  const activeAccounts = allAccounts.filter((a) => !a.isDisabled);
  const [accountId, setAccountId] = useState('');
  const [symbol, setSymbol] = useState('');
  const [timeframe, setTimeframe] = useState('1h');

  // Validation
  const [validating, setValidating] = useState(false);
  const [validationResult, setValidationResult] = useState<ValidateExtendedResult | null>(null);

  // Template
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [templatesLoading, setTemplatesLoading] = useState(false);
  const [loadedTemplate, setLoadedTemplate] = useState<StrategyTemplate | null>(null);
  const [saveModalOpen, setSaveModalOpen] = useState(false);
  const [saveLoading, setSaveLoading] = useState(false);
  const [saveForm] = Form.useForm();

  // Backtest
  const [backtestSubmitting, setBacktestSubmitting] = useState(false);
  const [backtestStatus, setBacktestStatus] = useState<BacktestStatus>('idle');
  const [backtestMetrics, setBacktestMetrics] = useState<any>(null);
  const [backtestError, setBacktestError] = useState('');

  // UI
  const [leftPanelCollapsed, setLeftPanelCollapsed] = useState(false);
  const [activeRightTab, setActiveRightTab] = useState('chart');

  // Load accounts & templates on mount
  useEffect(() => {
    fetchAccounts().then((list) => {
      const enabled = (list || []).filter((a) => !a.isDisabled);
      if (enabled.length > 0 && !accountId) setAccountId(enabled[0].id);
    });
    loadTemplates();
  }, []); // eslint-disable-line react-hooks/exhaustive-deps

  const loadTemplates = useCallback(async () => {
    setTemplatesLoading(true);
    try {
      const list = await strategyApi.listTemplates();
      setTemplates(list || []);
    } catch {
      // silent
    } finally {
      setTemplatesLoading(false);
    }
  }, []);

  const handleLoadTemplate = useCallback(async (id: string) => {
    try {
      const tpl = await strategyApi.getTemplate(id);
      if (tpl?.code) setCode(tpl.code);
      if (tpl?.name) {
        setLoadedTemplate(tpl as StrategyTemplate);
      }
      setLastValidatedCode('');
      setValidationResult(null);
    } catch (e: any) {
      message.error(e?.message || 'Failed to load template');
    }
  }, []);

  const handleValidate = useCallback(async () => {
    if (!code.trim()) return;
    setValidating(true);
    try {
      const result = await codeAssistApi.validateExtended(code);
      setValidationResult(result);
      if (result.valid) setLastValidatedCode(code);
    } catch (e: any) {
      message.error(e?.message || 'Validation failed');
    } finally {
      setValidating(false);
    }
  }, [code]);

  const handleRunBacktest = useCallback(async () => {
    if (!code || !symbol) return;
    setBacktestSubmitting(true);
    try {
      const result = await pythonStrategyApi.startBacktestRun({
        code,
        accountId,
        symbol,
        timeframe,
        initialCapital: 10000,
      });
      const runId = result.runId;
      if (!runId) throw new Error('No run ID returned');
      setBacktestStatus('running');
      setActiveRightTab('backtest');

      // Watch for updates
      const stopWatching = await pythonStrategyApi.watchBacktestRun(runId, (update: any) => {
        if (update.status === 'SUCCEEDED' || update.status === 'FAILED' || update.status === 'CANCELED') {
          setBacktestStatus(update.status === 'SUCCEEDED' ? 'completed' : 'error');
          setBacktestMetrics(update.metrics || null);
          setBacktestError(update.error || '');
          stopWatching();
        } else {
          setBacktestMetrics(update.metrics || null);
        }
      });
    } catch (e: any) {
      message.error(e?.message || 'Backtest failed');
      setBacktestStatus('error');
      setBacktestError(e?.message || 'Unknown error');
    } finally {
      setBacktestSubmitting(false);
    }
  }, [code, symbol, accountId, timeframe]);

  const handleSave = useCallback(async () => {
    if (!code || !lastValidatedCode || code !== lastValidatedCode) {
      message.warning(t('strategy.workspace.validateBeforeSave', 'Please validate code before saving'));
      return;
    }
    if (loadedTemplate) {
      setSaveLoading(true);
      try {
        await strategyApi.updateTemplate({
          id: loadedTemplate.id,
          name: loadedTemplate.name,
          description: loadedTemplate.description || '',
          code,
        });
        message.success(t('strategy.workspace.saveSuccess', 'Saved'));
        loadTemplates();
      } catch (e: any) {
        message.error(e?.message || 'Save failed');
      } finally {
        setSaveLoading(false);
      }
    } else {
      setSaveModalOpen(true);
    }
  }, [code, lastValidatedCode, loadedTemplate, t, loadTemplates]);

  const handleSaveAs = useCallback(() => {
    saveForm.resetFields();
    setSaveModalOpen(true);
  }, [saveForm]);

  const handleSaveModalOk = useCallback(async () => {
    try {
      const values = await saveForm.validateFields();
      setSaveLoading(true);
      await strategyApi.createTemplate({
        name: values.name,
        description: values.description || '',
        code,
      });
      message.success(t('strategy.workspace.saveSuccess', 'Saved'));
      setSaveModalOpen(false);
      loadTemplates();
    } catch (e: any) {
      if (e?.message) message.error(e.message);
    } finally {
      setSaveLoading(false);
    }
  }, [code, saveForm, t, loadTemplates]);

  const handleCopy = useCallback(() => {
    if (!code) return;
    navigator.clipboard.writeText(code).then(() => {
      message.success(t('strategy.workspace.copySuccess', 'Copied'));
    }).catch(() => {
      message.error(t('strategy.workspace.copyFailed', 'Copy failed'));
    });
  }, [code, t]);

  const canSave = code.length > 0 && lastValidatedCode.length > 0 && code === lastValidatedCode;

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: 'calc(100vh - 112px)' }}>
      {/* Header */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        padding: '8px 0 12px 0',
      }}>
        <h2 style={{ margin: 0 }}>{t('strategy.workspace.title', 'Strategy Workspace')}</h2>
        <Button
          type="text"
          icon={leftPanelCollapsed ? <MenuUnfoldOutlined /> : <MenuFoldOutlined />}
          onClick={() => setLeftPanelCollapsed(!leftPanelCollapsed)}
        />
      </div>

      {/* Main split */}
      <div style={{ display: 'flex', flex: 1, gap: 16, overflow: 'hidden' }}>
        {/* Left panel */}
        {!leftPanelCollapsed && (
          <div style={{
            width: 480, minWidth: 480, overflowY: 'auto',
            borderRight: '1px solid rgba(0,0,0,0.08)',
            paddingRight: 12,
          }}>
            <WorkspaceCodePanel
              code={code}
              onCodeChange={setCode}
              accounts={activeAccounts}
              accountId={accountId}
              onAccountIdChange={setAccountId}
              symbol={symbol}
              onSymbolChange={setSymbol}
              timeframe={timeframe}
              onTimeframeChange={setTimeframe}
              validating={validating}
              onValidate={handleValidate}
              validationResult={validationResult}
              onRunBacktest={handleRunBacktest}
              backtestSubmitting={backtestSubmitting}
              canSave={canSave}
              onSave={handleSave}
              onCopy={handleCopy}
            />

            <Collapse
              ghost
              size="small"
              style={{ marginTop: 16 }}
              items={[
                {
                  key: 'ai',
                  label: <span><RobotOutlined /> {t('strategy.workspace.aiAssist', 'AI Assistant')}</span>,
                  children: (
<AICodeReviseChat code={code} onApply={setCode} />
                  ),
                },
                {
                  key: 'template',
                  label: t('strategy.workspace.template.title', 'Template'),
                  children: (
                    <WorkspaceTemplateManager
                      templates={templates}
                      loading={templatesLoading}
                      loadedTemplate={loadedTemplate}
                      onLoad={handleLoadTemplate}
                      onSaveAs={handleSaveAs}
                    />
                  ),
                },
              ]}
            />
          </div>
        )}

        {/* Right panel */}
        <div style={{ flex: 1, overflowY: 'auto' }}>
          <Tabs
            activeKey={activeRightTab}
            onChange={setActiveRightTab}
            items={[
              {
                key: 'chart',
                label: t('strategy.workspace.chart', 'K-line'),
                children: (
                  <WorkspaceChartTab
                    symbol={symbol}
                    timeframe={timeframe}
                    onTimeframeChange={setTimeframe}
                  />
                ),
              },
              {
                key: 'backtest',
                label: t('strategy.workspace.backtest', 'Backtest'),
                children: (
                  <WorkspaceBacktestPanel
                    status={backtestStatus}
                    metrics={backtestMetrics}
                    errorMessage={backtestError}
                  />
                ),
              },
              {
                key: 'ai',
                label: t('strategy.workspace.ai', 'AI'),
                children: (
<CodeExplainPanel code={code} />
                ),
              },
            ]}
          />
        </div>
      </div>

      {/* Save modal */}
      <Suspense fallback={null}>
        <SaveTemplateModal
          open={saveModalOpen}
          confirmLoading={saveLoading}
          form={saveForm}
          onCancel={() => setSaveModalOpen(false)}
          onOk={handleSaveModalOk}
        />
      </Suspense>
    </div>
  );
}
