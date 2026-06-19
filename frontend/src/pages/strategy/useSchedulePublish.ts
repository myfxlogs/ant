
import { DEFAULT_DRAFT_NAME_KEY, MESSAGES_BACKTEST_RUNNING_CANNOT_PUBLISH_KEY, MESSAGES_CANNOT_PUBLISH_AND_CREATE_DRAFT_FAILED_KEY, MESSAGES_MISSING_DRAFT_ID_CANNOT_PUBLISH_KEY, MESSAGES_PUBLISHED_BUT_NO_TEMPLATE_ID_KEY, MESSAGES_PUBLISH_FAILED_KEY, MESSAGES_REPUBLISHED_BUT_NO_TEMPLATE_ID_KEY, MESSAGES_STRATEGY_CODE_EMPTY_CANNOT_PUBLISH_KEY, MESSAGES_TEMPLATE_ALREADY_PUBLISHED_KEY, MESSAGES_TEMPLATE_NOT_DRAFT_UNKNOWN_PUBLISH_STATUS_KEY, MESSAGES_TEMPLATE_PUBLISHED_KEY, MESSAGES_TEMPLATE_REPUBLISHED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';

import { message } from 'antd';
import { strategyApi } from '@/client/strategy';
import { codeAssistApi, type RequiredParamSpec } from '@/client/codeAssist';
import { isTerminalRun, isErrTemplateNotDraft } from './StrategyTemplatePage.utils';
import type { ScheduleFlowState } from './StrategyTemplateScheduleLaunchModal';

export function formatPercent(v: unknown): string {
  if (v === undefined || v === null || v === '') return '-';
  const n = typeof v === 'number' ? v : Number(v);
  if (!Number.isFinite(n)) return String(v);
  return `${(n * 100).toFixed(2)}%`;
}

export function formatFloat(v: unknown, digits = 3): string {
  if (v === undefined || v === null || v === '') return '-';
  const n = typeof v === 'number' ? v : Number(v);
  if (!Number.isFinite(n)) return String(v);
  return n.toFixed(digits);
}

export function formatInt(v: unknown): string {
  if (v === undefined || v === null || v === '') return '-';
  const n = typeof v === 'number' ? v : Number(v);
  if (!Number.isFinite(n)) return String(v);
  return String(Math.round(n));
}

export function useSchedulePublish({
  scheduleFlow,
  setScheduleFlow,
  setRuns,
  scoreRunId,
  t,
}: {
  scheduleFlow: ScheduleFlowState;
  setScheduleFlow: React.Dispatch<React.SetStateAction<ScheduleFlowState>>;
  setRuns: React.Dispatch<React.SetStateAction<any[]>>;
  scoreRunId: string;
  t: (key: string, opts?: any) => string;
}) {
  const handlePublishTemplate = async () => {
    if (!isTerminalRun((scheduleFlow as any).run)) {
      message.warning(t(MESSAGES_BACKTEST_RUNNING_CANNOT_PUBLISH_KEY));
      return;
    }
    const draftId = String(scheduleFlow.templateDraftId || '').trim();
    if (!draftId) {
      message.error(t(MESSAGES_MISSING_DRAFT_ID_CANNOT_PUBLISH_KEY));
      return;
    }
    setScheduleFlow((p) => ({ ...p, publishing: true }));
    try {
      const draft: any = await strategyApi.getTemplate(draftId);
      const draftCode = String(draft?.code || '').trim();
      if (!draftCode) {
        message.error(t(MESSAGES_STRATEGY_CODE_EMPTY_CANNOT_PUBLISH_KEY));
        return;
      }
      const resp: any = await strategyApi.publishTemplateDraft(draftId);
      const tid = String(resp?.id || resp?.template?.id || resp?.templateId || '').trim();
      if (!tid) {
        message.warning(t(MESSAGES_PUBLISHED_BUT_NO_TEMPLATE_ID_KEY));
        return;
      }
      setScheduleFlow((p) => ({ ...p, templateId: tid }));
      setRuns((prev) =>
        (prev || []).map((it) =>
          String(it?.id || '') === String(scoreRunId || '') ? { ...it, templateId: tid } : it,
        ),
      );
      message.success(t(MESSAGES_TEMPLATE_PUBLISHED_KEY));
    } catch (e: unknown) {
      if (isErrTemplateNotDraft(e)) {
        try {
          const tpl: any = await strategyApi.getTemplate(draftId);
          const status = String(tpl?.status || '').trim().toLowerCase();
          if (status !== 'published') {
            const baseName =
              String(tpl?.name || t(DEFAULT_DRAFT_NAME_KEY)).trim() ||
              t(DEFAULT_DRAFT_NAME_KEY);
            const newDraft: any = await strategyApi.createTemplateDraft({ name: baseName });
            const newId = String(newDraft?.id || '').trim();
            if (!newId) {
              message.warning(t(MESSAGES_CANNOT_PUBLISH_AND_CREATE_DRAFT_FAILED_KEY));
              return;
            }
            const codeToPublish = String(tpl?.code || '').trim();
            if (!codeToPublish) {
              message.error(t(MESSAGES_STRATEGY_CODE_EMPTY_CANNOT_PUBLISH_KEY));
              return;
            }
            await strategyApi.updateTemplateDraft({
              id: newId,
              name: String(tpl?.name || '').trim() || baseName,
              description: String(tpl?.description || '').trim(),
              code: codeToPublish,
              parameters: Array.isArray(tpl?.parameters) ? tpl.parameters : [],
              tags: Array.isArray(tpl?.tags) ? tpl.tags : [],
            });
            const pub: any = await strategyApi.publishTemplateDraft(newId);
            const tid = String(pub?.id || pub?.template?.id || pub?.templateId || '').trim();
            if (!tid) {
              message.warning(t(MESSAGES_REPUBLISHED_BUT_NO_TEMPLATE_ID_KEY));
              return;
            }
            setScheduleFlow((p) => ({ ...p, templateId: tid }));
            setRuns((prev) =>
              (prev || []).map((it) =>
                String(it?.id || '') === String(scoreRunId || '')
                  ? { ...it, templateId: tid }
                  : it,
              ),
            );
            message.success(t(MESSAGES_TEMPLATE_REPUBLISHED_KEY));
            return;
          }
          setScheduleFlow((p) => ({ ...p, templateId: draftId }));
          setRuns((prev) =>
            (prev || []).map((it) =>
              String(it?.id || '') === String(scoreRunId || '')
                ? { ...it, templateId: draftId }
                : it,
            ),
          );
          message.info(t(MESSAGES_TEMPLATE_ALREADY_PUBLISHED_KEY));
          return;
        } catch (_e2) {
          message.warning(t(MESSAGES_TEMPLATE_NOT_DRAFT_UNKNOWN_PUBLISH_STATUS_KEY));
          return;
        }
      }
      const errMsg =
        String(
          (e as any)?.rawMessage ||
            ((e as any)?.code !== undefined ? `code=${String((e as any).code)} ` : '') + ((e as any)?.message || '') ||
            e,
        ) || t(MESSAGES_PUBLISH_FAILED_KEY);
      message.error(errMsg);
    } finally {
      setScheduleFlow((p) => ({ ...p, publishing: false }));
    }
  };

  return { handlePublishTemplate };
}
