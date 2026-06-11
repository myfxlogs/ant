import { useCallback, useEffect, useState } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import type { StrategyTemplate } from '@/client/strategy';
import { strategyTemplateApi, type CreateTemplateRequest } from '@/client/strategy-schedules';
import { codeAssistApi } from '@/client/codeAssist';
import { DEFAULT_TEMPLATES } from '../StrategyLibrary.defaults';
import { isSystemTemplate, isPublicTemplate } from './libraryTypes';

export type TemplateFilter = 'all' | 'user' | 'system';

export function useLibraryTemplates() {
  const { t, i18n } = useTranslation();
  const [templates, setTemplates] = useState<StrategyTemplate[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<TemplateFilter>('all');
  const [search, setSearch] = useState('');
  const [selectedId, setSelectedId] = useState<string>('');
  const [editOpen, setEditOpen] = useState(false);
  const [editing, setEditing] = useState<StrategyTemplate | null>(null);
  const [codeValidating, setCodeValidating] = useState(false);
  const [lastValidatedCode, setLastValidatedCode] = useState<string>('');
  const [publishing, setPublishing] = useState(false);
  const queryClient = useQueryClient();

  const fetchTemplates = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const resp = await strategyTemplateApi.list();
      setTemplates(resp || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : t('strategy.templates.messages.fetchTemplateListFailed'));
    } finally { setLoading(false); }
  }, [t]);

  useEffect(() => { fetchTemplates(); }, [fetchTemplates]);
  useEffect(() => {
    const onLang = () => { void fetchTemplates(); };
    i18n.on('languageChanged', onLang);
    return () => { i18n.off('languageChanged', onLang); };
  }, [i18n, fetchTemplates]);

  const allTemplates: StrategyTemplate[] = templates.length > 0
    ? templates
    : (DEFAULT_TEMPLATES as any[]).map(d => ({
        ...d,
        name: d.nameKey ? t(d.nameKey) : d.name,
        description: d.descriptionKey ? t(d.descriptionKey) : d.description,
      })) as StrategyTemplate[];

  const filtered = allTemplates.filter(tpl => {
    const system = isSystemTemplate(tpl);
    if (filter === 'system' && !system) return false;
    if (filter === 'user' && system) return false;
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

  const handleSave = useCallback(async (values: Record<string, unknown>) => {
    try {
      setCodeValidating(true);
      const code = String(values.code || '');
      const ext = await codeAssistApi.validateExtended(code);
      if (!ext.valid) { message.error(ext.errors?.[0] || ext.warnings?.[0] || t('strategy.templates.messages.codeValidationNotPassed')); return; }
      const data: CreateTemplateRequest = { name: String(values.name || ''), description: String(values.description || ''), code, parameters: [], isPublic: Boolean(values.isPublic) || false, tags: [] };
      if (editing) { await strategyTemplateApi.update({ id: editing.id, ...data }); message.success(t('strategy.templates.messages.templateUpdated')); }
      else { await strategyTemplateApi.create(data); message.success(t('strategy.templates.messages.templateCreated')); setFilter('user'); }
      setEditOpen(false); fetchTemplates();
    } catch { message.error(t('common.saveFailed')); }
    finally { setCodeValidating(false); }
  }, [editing, fetchTemplates, t]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await strategyTemplateApi.delete(id);
      message.success(t('strategy.templates.messages.templateDeleted'));
      if (String(selectedId) === id) setSelectedId('');
      fetchTemplates();
    } catch (err: unknown) {
      const e = err as { code?: string; rawMessage?: string; message?: string };
      if (String(e?.code || '').toLowerCase().includes('permission')
        || String(e?.rawMessage || e?.message || '').toLowerCase().includes('system template')) {
        message.error(t('strategy.templates.messages.systemTemplateReadOnly'));
        return;
      }
      message.error(t('common.deleteFailed'));
    }
  }, [selectedId, fetchTemplates, t]);

  const handlePublish = useCallback(async (id: string) => {
    setPublishing(true);
    try {
      await strategyTemplateApi.update({ id, isPublic: true } as any);
      message.success(t('strategy.library.publishSuccess'));
      fetchTemplates();
      queryClient.invalidateQueries({ queryKey: ['marketplace'] });
    } catch { message.error(t('common.saveFailed')); }
    finally { setPublishing(false); }
  }, [fetchTemplates, t, queryClient]);

  const handleUnpublish = useCallback(async (id: string) => {
    setPublishing(true);
    try {
      await strategyTemplateApi.update({ id, isPublic: false } as any);
      message.success(t('strategy.library.unpublishSuccess'));
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
    codeValidating, lastValidatedCode, setLastValidatedCode,
    publishing, fetchTemplates, openCreate, openEdit, handleSave, handleDelete,
    handlePublish, handleUnpublish,
  };
}
