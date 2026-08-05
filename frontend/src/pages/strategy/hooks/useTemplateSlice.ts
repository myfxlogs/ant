import { useState, useEffect, useCallback } from 'react';
import { useSearchParams, useParams, useNavigate } from 'react-router-dom';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useAuthStore } from '@/stores/authStore';

interface LoadedTemplate {
  userId?: string;
  isSystem?: boolean;
  parameters?: Array<{ name?: string; type?: string; default?: string; label?: string }>;
  code?: string;
}

interface TemplateSliceDeps {
  handleLoadTemplate: (id: string) => Promise<LoadedTemplate | null>;
  validateCode: (code: string) => void;
  updateExtractedParams: (params: unknown[] | null) => void;
}

export interface TemplateSlice {
  selectedId: string;
  onSelect: (templateId: string | null) => void;
}

export function useTemplateSlice(deps: TemplateSliceDeps): TemplateSlice {
  const [searchParams] = useSearchParams();
  const { id: routeId } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const currentUserId = useAuthStore(s => s.user?.id);
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');

  const handleSelectTemplate = useCallback(async (templateId: string | null) => {
    if (!templateId) {
      setSelectedTemplateId('');
      deps.updateExtractedParams(null);
      return;
    }
    setSelectedTemplateId(templateId);
    const tpl = await deps.handleLoadTemplate(templateId);
    if (!tpl) { setSelectedTemplateId(''); return; }
    const isOwner = tpl.userId && tpl.userId === currentUserId;
    const isSystem = !!tpl.isSystem;
    if (!isOwner && !isSystem) {
      navigate('/strategy');
      return;
    }
    setCenterTab('code');
    if (tpl.parameters?.length) {
      const params = tpl.parameters.map((p: unknown) => ({
        name: p.name || '', type: p.type || 'string',
        default: p.default || '', label: p.label || p.name || '',
      }));
      deps.updateExtractedParams(params);
    } else if (tpl.code?.trim()) {
      deps.validateCode(tpl.code);
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- deps properties individually listed  | REF: rd.md#part-0.2-hooks-deps
  }, [deps.handleLoadTemplate, deps.validateCode, deps.updateExtractedParams, setCenterTab, navigate, currentUserId]);

  // Load template from URL on mount — supports both ?templateId=X and /:id/edit route param.
  useEffect(() => {
    const tid = routeId || searchParams.get('templateId');
    if (tid) handleSelectTemplate(tid);
  }, [searchParams, routeId, handleSelectTemplate]);

  return { selectedId: selectedTemplateId, onSelect: handleSelectTemplate };
}
