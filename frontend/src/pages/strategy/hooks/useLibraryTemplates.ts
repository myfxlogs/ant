import { useCallback, useEffect, useState } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next'
import { MESSAGES_CODE_VALIDATION_NOT_PASSED_KEY, MESSAGES_FETCH_TEMPLATE_LIST_FAILED_KEY, MESSAGES_SYSTEM_TEMPLATE_READ_ONLY_KEY, MESSAGES_TEMPLATE_CREATED_KEY, MESSAGES_TEMPLATE_DELETED_KEY, MESSAGES_TEMPLATE_UPDATED_KEY } from '@/gen/ant/v1/i18n/strategy_templates_keys';
import { MY_COPY_KEY, SAVE_AS_MINE_SUCCESS_KEY, UNPUBLISH_SUCCESS_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';

import { useQueryClient } from '@tanstack/react-query';
import type { StrategyTemplate } from '@/client/strategy';
import { strategyTemplateApi, type CreateTemplateRequest } from '@/client/strategy-schedules';
import { codeAssistApi, type ValidateExtendedResult } from '@/client/codeAssist';
import type { ParameterEntry } from '@/gen/ant/v1/parameter_entry_pb';
import { buildParamI18n } from '@/utils/paramLabel';
import { useAuthStore } from '@/stores/authStore';
import { isSystemTemplate } from './libraryTypes';

export type TemplateFilter = 'user' | 'system';

export function useLibraryTemplates() {
  const { t, i18n } = useTranslation();
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<TemplateFilter>('user');
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string>('');
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<StrategyTemplate | null>(null);
  const [codeValidating, setCodeValidating] = useState(false);
  const [lastValidatedCode, setLastValidatedCode] = useState<string>('');
  const [validationResult, setValidationResult] = useState<ValidateExtendedResult | null>(null);
  const [publishing, setPublishing] = useState(false);
  const queryClient = useQueryClient();
  const currentUserId = useAuthStore(s => s.user?.id);

  const fetchTemplates = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const resp = await strategyTemplateApi.list();
      setTemplates(resp || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t(MESSAGES_FETCH_TEMPLATE_LIST_FAILED_KEY));
    } finally { setLoading(false); }
  }, [t]);

  useEffect(() => { fetchTemplates(); }, [fetchTemplates]);
  useEffect(() => {
    const onLang = () => { void fetchTemplates(); };
    i18n.on('languageChanged', onLang);
    return () => { i18n.off('languageChanged', onLang); };
  }, [i18n, fetchTemplates]);

  const allTemplates: StrategyTemplate[] = templates;

  const filtered = allTemplates.filter(tpl => {
    const system = isSystemTemplate(tpl);
    if (filter === 'system' && !system) return false;
    if (filter === 'user') {
      if (system) return false;
      // Exclude other users' templates — discovery is through Marketplace, not Library
      if (currentUserId && String(tpl.userId || '') !== currentUserId) return false;
    }
    if (search) {
      const q = search.toLowerCase();
      if (!String(tpl.name || '').toLowerCase().includes(q)
        && !String(tpl.description || '').toLowerCase().includes(q)) return false;
    }
    return true;
  });

  const selected = filtered.find(t => String(t.id) === selectedId) || null;

  const openCreate = useCallback(() => {
    setEditing(null); setLastValidatedCode(''); setEditOpen(true);
  }, []);

  const openEdit = useCallback((tpl: StrategyTemplate) => {
    setEditing(tpl); setLastValidatedCode(''); setEditOpen(true);
  }, []);

  const handleSaveAsMine = useCallback(async (tpl: StrategyTemplate) => {
    try {
      const data: CreateTemplateRequest = {
        name: `${String(tpl.name || '')} (${t(MY_COPY_KEY)})`,
        description: String(tpl.description || ''),
        code: String(tpl.code || ''),
        parameters: [],
        isPublic: false,
        tags: Array.isArray(tpl.tags) ? [...tpl.tags] : [],
      };
      const created = await strategyTemplateApi.create(data);
      message.success(t(SAVE_AS_MINE_SUCCESS_KEY));
      fetchTemplates();
      setSelectedId(String(created.id || ''));
      setFilter('user');
    } catch { message.error(t('common.saveFailed')); }
  }, [fetchTemplates, t]);

  const handleValidate = useCallback(async (code: string) => {
    if (!code.trim()) return;
    setCodeValidating(true);
    setValidationResult(null);
    try {
      const result = await codeAssistApi.validateExtended(code);
      setValidationResult(result);
      if (result.valid) {
        setLastValidatedCode(code);
        message.success(t('strategy.validate.passed', { defaultValue: 'Validation passed — Save is now unlocked.' }));
      } else {
        message.error(result.errors?.[0] || result.warnings?.[0] || t(MESSAGES_CODE_VALIDATION_NOT_PASSED_KEY));
      }
    } catch (e: unknown) { message.error((e as Error)?.message || t('strategy.validate.failed', { defaultValue: 'Validation failed' })); }
    finally { setCodeValidating(false); }
  }, [t]);

  const handleSave = useCallback(async (values: Record<string, unknown>) => {
    try {
      setCodeValidating(true);
      const code = String(values.code || '');
      const paramEntries = await ensureValidated(code, lastValidatedCode, validationResult, setLastValidatedCode, t);
      if (paramEntries === null) return;

      const i18n = paramEntries.length > 0 ? await buildParamI18n(paramEntries) : null;
      const data: CreateTemplateRequest = {
        name: String(values.name || ''), description: String(values.description || ''), code,
        parameters: paramEntries.map(e => ({ key: e.name, type: (e.type || 'string') as string, defaultValue: e.default })),
        isPublic: Boolean(values.isPublic) || false, tags: [], i18n: i18n || undefined,
      };
      if (editing) { await strategyTemplateApi.update({ id: editing.id, ...data }); message.success(t(MESSAGES_TEMPLATE_UPDATED_KEY)); }
      else { await strategyTemplateApi.create(data); message.success(t(MESSAGES_TEMPLATE_CREATED_KEY)); setFilter('user'); }
      setEditOpen(false); fetchTemplates();
    } catch { message.error(t('common.saveFailed')); }
    finally { setCodeValidating(false); }
  }, [editing, fetchTemplates, t, lastValidatedCode, validationResult]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await strategyTemplateApi.delete(id);
      message.success(t(MESSAGES_TEMPLATE_DELETED_KEY));
      if (String(selectedId) === id) setSelectedId('');
      fetchTemplates();
    } catch (err: unknown) {
      const e = err as { code?: string; rawMessage?: string; message?: string };
      if (String(e?.code || '').toLowerCase().includes('permission')
        || String(e?.rawMessage || e?.message || '').toLowerCase().includes('system template')) {
        message.error(t(MESSAGES_SYSTEM_TEMPLATE_READ_ONLY_KEY));
        return;
      }
      message.error(t('common.deleteFailed'));
    }
  }, [selectedId, fetchTemplates, t]);

  const [publishModalOpen, setPublishModalOpen] = useState(false);
  const [publishingTemplate, setPublishingTemplate] = useState<StrategyTemplate | null>(null);

  const openPublishModal = useCallback((tpl: StrategyTemplate) => {
    setPublishingTemplate(tpl);
    setPublishModalOpen(true);
  }, []);

  const closePublishModal = useCallback(() => {
    setPublishingTemplate(null);
    setPublishModalOpen(false);
  }, []);

  const handlePublish = useCallback((id: string) => {
    const tpl = allTemplates.find(t => String(t.id) === id);
    if (tpl) openPublishModal(tpl);
  }, [allTemplates, openPublishModal]);

  const handleUnpublish = useCallback(async (id: string) => {
    setPublishing(true);
    try {
      await strategyTemplateApi.update({ id, isPublic: false });
      message.success(t(UNPUBLISH_SUCCESS_KEY));
      fetchTemplates();
      queryClient.invalidateQueries({ queryKey: ['marketplace'] });
    } catch { message.error(t('common.saveFailed')); }
    finally { setPublishing(false); }
  }, [fetchTemplates, t, queryClient]);

  return {
    templates: filtered, allTemplates,
    loading, error, filter, setFilter, search, setSearch,
    selectedId, setSelectedId, selected,
    editOpen, setEditOpen, editing, setEditing,
    codeValidating, lastValidatedCode, setLastValidatedCode, validationResult, setValidationResult,
    publishing, fetchTemplates, openCreate, openEdit, handleSaveAsMine, handleValidate, handleSave, handleDelete,
    handlePublish, handleUnpublish,
    // Marketplace publish modal
    publishModalOpen, setPublishModalOpen, publishingTemplate,
    openPublishModal, closePublishModal,
  };
}

async function ensureValidated(
  code: string,
  lastValidatedCode: string | undefined,
  validationResult: { parameterEntries?: ParameterEntry[] } | null,
  setLastValidatedCode: (s: string) => void,
  t: (k: string) => string,
): Promise<ParameterEntry[] | null> {
  const paramEntries = validationResult?.parameterEntries ?? [];
  if (code === lastValidatedCode) return paramEntries;
  const ext = await codeAssistApi.validateExtended(code);
  if (!ext.valid) {
    message.error(ext.errors?.[0] || ext.warnings?.[0] || t(MESSAGES_CODE_VALIDATION_NOT_PASSED_KEY));
    return null;
  }
  setLastValidatedCode(code);
  return ext.parameterEntries ?? paramEntries;
}
