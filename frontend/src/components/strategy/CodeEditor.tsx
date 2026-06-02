import React from 'react';
import { Card, Button, Select, Input, Alert, Space } from 'antd';
import { PlayCircleOutlined, CheckCircleOutlined, CopyOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import SaveTemplateModal from './SaveTemplateModal';
import { useCodeEditor } from './useCodeEditor';
import type { CodeEditorProps } from './CodeEditor.types';

const { TextArea } = Input;

const CodeEditor: React.FC<CodeEditorProps> = (props) => {
  const { t } = useTranslation();
  const ctx = useCodeEditor(props);

  return (
    <div className="p-6">
      <Card title={t('strategy.codeEditor.title')} className="mb-4">
        <div className="mb-4 flex gap-4">
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.account')}</label>
            <Select
              style={{ width: '100%' }}
              value={ctx.selectedAccount}
              onChange={(v) => { ctx.setSelectedAccount(v); ctx.loadSymbols(v); }}
              placeholder={t('strategy.codeEditor.placeholders.selectAccount')}
            >
              {ctx.accounts.map((account) => (
                <Select.Option key={account.id} value={account.id} disabled={!!account.isDisabled}>
                  {account.login} ({account.mtType}){account.isDisabled ? t('strategy.codeEditor.labels.disabledSuffix') : ''}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.symbol')}</label>
            <Select
              showSearch allowClear
              loading={ctx.symbolsLoading}
              style={{ width: '100%' }}
              value={ctx.symbol}
              onChange={(v) => ctx.setSymbol(v || '')}
              placeholder={
                !ctx.selectedAccount ? t('strategy.codeEditor.placeholders.selectAccountFirst')
                  : ctx.symbolsLoading ? t('strategy.codeEditor.placeholders.loadingSymbols')
                  : t('strategy.codeEditor.placeholders.selectSymbol')
              }
              options={ctx.symbols}
              disabled={!ctx.selectedAccount || ctx.symbolsLoading}
              optionFilterProp="label"
              filterOption={(input, option) => {
                const key = String(option?.value || '').toLowerCase();
                const label = String(option?.label || '').toLowerCase();
                const q = input.toLowerCase();
                return key.includes(q) || label.includes(q);
              }}
              notFoundContent={
                !ctx.selectedAccount ? null
                  : ctx.symbolsLoading ? t('strategy.codeEditor.placeholders.loadingSymbols')
                  : t('strategy.codeEditor.placeholders.noSymbols')
              }
            />
          </div>
          <div className="flex-1">
            <label className="block mb-1 text-sm">{t('strategy.codeEditor.labels.timeframe')}</label>
            <Select style={{ width: '100%' }} value={ctx.timeframe} onChange={ctx.setTimeframe}>
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
            <Button size="small" icon={<CopyOutlined />} onClick={ctx.copyCode}>
              {t('strategy.codeEditor.actions.copy')}
            </Button>
          </div>
          <TextArea
            rows={15}
            value={ctx.code}
            onChange={(e) => ctx.setCode(e.target.value)}
            placeholder={t('strategy.codeEditor.placeholders.code')}
            style={{ fontFamily: 'monospace' }}
          />
        </div>

        <Space>
          <Button icon={<CheckCircleOutlined />} onClick={ctx.handleValidate} loading={ctx.validating}>
            {t('strategy.codeEditor.actions.validate')}
          </Button>
          <Button type="primary" icon={<PlayCircleOutlined />} onClick={ctx.handlePreview} loading={ctx.loading}>
            {t('strategy.codeEditor.actions.preview')}
          </Button>
          <Button onClick={ctx.openSaveTemplate}>
            {t('strategy.codeEditor.actions.saveAsTemplate')}
          </Button>
        </Space>

        <div className="mt-2 text-xs text-gray-500">
          {t('strategy.codeEditor.hints.previewInfo')}
        </div>
      </Card>

      {ctx.validationResult && (
        <Card title={t('strategy.codeEditor.cards.validationResult')} className="mb-4">
          <div className="mb-3">
            <Button size="small" onClick={() => {
              const details = JSON.stringify({
                valid: !!ctx.validationResult?.valid,
                errors: ctx.validationResult?.errors || [],
                warnings: ctx.validationResult?.warnings || [],
              }, null, 2);
              ctx.sendToAIWithContext(t('strategy.codeEditor.actions.sendToAIFixTitleValidate'), details);
            }}>
              {t('strategy.codeEditor.actions.sendToAI')}
            </Button>
          </div>
          {ctx.validationResult.valid ? (
            <Alert title={t('strategy.codeEditor.messages.validateOk')} type="success" />
          ) : (
            <div>
              {ctx.validationResult.errors.map((err, i) => (
                <Alert key={i} title={err} type="error" className="mb-2" />
              ))}
              {ctx.validationResult.warnings.map((warn, i) => (
                <Alert key={i} title={warn} type="warning" className="mb-2" />
              ))}
            </div>
          )}
        </Card>
      )}

      {ctx.previewResult && (
        <Card title={t('strategy.codeEditor.cards.previewResult')} className="mb-4">
          <div className="mb-3">
            <Button size="small" onClick={() => {
              const details = JSON.stringify(ctx.previewResult || {}, null, 2);
              ctx.sendToAIWithContext(t('strategy.codeEditor.actions.sendToAIFixTitlePreview'), details);
            }}>
              {t('strategy.codeEditor.actions.sendToAI')}
            </Button>
          </div>
          {ctx.previewResult.success ? (
            <Alert title={t('strategy.codeEditor.messages.previewSuccess')} type="success" />
          ) : (
            <Alert title={ctx.previewResult.error || t('strategy.codeEditor.messages.previewFailed')} type="error" />
          )}
        </Card>
      )}
      <SaveTemplateModal
        open={ctx.saveTemplateOpen}
        confirmLoading={ctx.saveTemplateLoading}
        form={ctx.saveTemplateForm}
        afterOpenChange={(open) => {
          if (!open) return;
          ctx.saveTemplateForm.setFieldsValue({ name: '', description: '' });
        }}
        onCancel={() => ctx.setSaveTemplateOpen(false)}
        onOk={ctx.handleSaveTemplate}
      />
    </div>
  );
};

export default CodeEditor;
