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
      message.warning(t('strategy.templates.messages.backtestRunningCannotPublish'));
      return;
    }
    const draftId = String(scheduleFlow.templateDraftId || '').trim();
    if (!draftId) {
      message.error(t('strategy.templates.messages.missingDraftIdCannotPublish'));
      return;
    }
    setScheduleFlow((p) => ({ ...p, publishing: true }));
    try {
      const draft: any = await strategyApi.getTemplate(draftId);
      const draftCode = String(draft?.code || '').trim();
      if (!draftCode) {
        message.error(t('strategy.templates.messages.strategyCodeEmptyCannotPublish'));
        return;
      }
      const resp: any = await strategyApi.publishTemplateDraft(draftId);
      const tid = String(resp?.id || resp?.template?.id || resp?.templateId || '').trim();
      if (!tid) {
        message.warning(t('strategy.templates.messages.publishedButNoTemplateId'));
        return;
      }
      setScheduleFlow((p) => ({ ...p, templateId: tid }));
      setRuns((prev) =>
        (prev || []).map((it) =>
          String(it?.id || '') === String(scoreRunId || '') ? { ...it, templateId: tid } : it,
        ),
      );
      message.success(t('strategy.templates.messages.templatePublished'));
    } catch (e: unknown) {
      if (isErrTemplateNotDraft(e)) {
        try {
          const tpl: any = await strategyApi.getTemplate(draftId);
          const status = String(tpl?.status || '').trim().toLowerCase();
          if (status !== 'published') {
            const baseName =
              String(tpl?.name || t('strategy.templates.defaultDraftName')).trim() ||
              t('strategy.templates.defaultDraftName');
            const newDraft: any = await strategyApi.createTemplateDraft({ name: baseName });
            const newId = String(newDraft?.id || '').trim();
            if (!newId) {
              message.warning(t('strategy.templates.messages.cannotPublishAndCreateDraftFailed'));
              return;
            }
            const codeToPublish = String(tpl?.code || '').trim();
            if (!codeToPublish) {
              message.error(t('strategy.templates.messages.strategyCodeEmptyCannotPublish'));
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
              message.warning(t('strategy.templates.messages.republishedButNoTemplateId'));
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
            message.success(t('strategy.templates.messages.templateRepublished'));
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
          message.info(t('strategy.templates.messages.templateAlreadyPublished'));
          return;
        } catch (_e2) {
          message.warning(t('strategy.templates.messages.templateNotDraftUnknownPublishStatus'));
          return;
        }
      }
      const errMsg =
        String(
          (e as any)?.rawMessage ||
            ((e as any)?.code !== undefined ? `code=${String((e as any).code)} ` : '') + ((e as any)?.message || '') ||
            e,
        ) || t('strategy.templates.messages.publishFailed');
      message.error(errMsg);
    } finally {
      setScheduleFlow((p) => ({ ...p, publishing: false }));
    }
  };

  return { handlePublishTemplate };
}
