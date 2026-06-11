import { useCallback, useEffect, useState } from 'react';
import { message } from 'antd';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';
import { strategyTemplateApi, type CreateTemplateRequest } from '@/client/strategy-schedules';
import { codeAssistApi, type RequiredParamSpec } from '@/client/codeAssist';
import { DEFAULT_TEMPLATES } from '../StrategyTemplatePage.defaults';

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

  const fetchTemplates = useCallback(async () => {
    setLoading(true); setError(null);
    try {
      const resp = await strategyTemplateApi.list();
      setTemplates(resp || []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : t('strategy.templates.messages.fetchTemplateListFailed');
      setError(msg);
    } finally { setLoading(false); }
  }, [t]);

  useEffect(() => { fetchTemplates(); }, [fetchTemplates]);
  useEffect(() => {
    const onLang = () => { void fetchTemplates(); };
    i18n.on('languageChanged', onLang);
    return () => { i18n.off('languageChanged', onLang); };
  }, [i18n, fetchTemplates]);

  // ── Derived: merge with defaults if backend returns empty ──
  const allTemplates = templates.length > 0
    ? templates
    : (DEFAULT_TEMPLATES as any[]).map(d => ({ ...d, name: d.nameKey ? t(d.nameKey) : d.name, description: d.descriptionKey ? t(d.descriptionKey) : d.description }));

  // ── Filtered list ──
  const filtered = allTemplates.filter(tpl => {
    const tags = Array.isArray((tpl as any).tags) ? (tpl as any).tags : [];
    const isSystem = Boolean((tpl as any).isSystem) || tags.includes('preset') || String(tpl.id || '').startsWith('default-');
    if (filter === 'system' && !isSystem) return false;
    if (filter === 'user' && isSystem) return false;
    if (search) {
      const q = search.toLowerCase();
      if (!String(tpl.name || '').toLowerCase().includes(q) && !String(tpl.description || '').toLowerCase().includes(q)) return false;
    }
    return true;
  });

  // ── Selected template ──
  const selected = filtered.find(t => String(t.id) === selectedId) || null;

  // ── CRUD actions ──
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
      const data: CreateTemplateRequest = {
        name: String(values.name || ''), description: String(values.description || ''),
        code, parameters: [], isPublic: Boolean(values.isPublic) || false, tags: [],
      };
      if (editing) {
        await strategyTemplateApi.update({ id: editing.id, ...data });
        message.success(t('strategy.templates.messages.templateUpdated'));
      } else {
        await strategyTemplateApi.create(data);
        message.success(t('strategy.templates.messages.templateCreated'));
        setFilter('user');
      }
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
      const code = String(e?.code || '').toLowerCase();
      const msg = String(e?.rawMessage || e?.message || '');
      if (code.includes('permission') || msg.toLowerCase().includes('system template')) {
        message.error(t('strategy.templates.messages.systemTemplateReadOnly', '系统模板不可删除或修改'));
        return;
      }
      message.error(t('common.deleteFailed'));
    }
  }, [selectedId, fetchTemplates, t]);

  const handlePublish = useCallback(async (id: string) => {
    setPublishing(true);
    try {
      await strategyTemplateApi.update({ id, isPublic: true } as any);
      message.success(t('strategy.library.publishSuccess', '已发布'));
      fetchTemplates();
    } catch { message.error(t('common.saveFailed')); }
    finally { setPublishing(false); }
  }, [fetchTemplates, t]);

  const handleUnpublish = useCallback(async (id: string) => {
    setPublishing(true);
    try {
      await strategyTemplateApi.update({ id, isPublic: false } as any);
      message.success(t('strategy.library.unpublishSuccess', '已下架'));
      fetchTemplates();
    } catch { message.error(t('common.saveFailed')); }
    finally { setPublishing(false); }
  }, [fetchTemplates, t]);

  return {
    templates: filtered, allTemplates,
    loading, error,
    filter, setFilter, search, setSearch,
    selectedId, setSelectedId, selected,
    editOpen, setEditOpen, editing, setEditing,
    codeValidating, lastValidatedCode, setLastValidatedCode,
    publishing,
    fetchTemplates, openCreate, openEdit, handleSave, handleDelete,
    handlePublish, handleUnpublish,
  };
}
