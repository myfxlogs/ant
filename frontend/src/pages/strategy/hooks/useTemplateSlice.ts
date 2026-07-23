import { useState, useEffect, useCallback } from 'react';
import { useSearchParams, useParams } from 'react-router-dom';
import { useWorkspaceStore } from '@/stores/workspaceStore';

interface TemplateSliceDeps {
  handleLoadTemplate: (id: string) => Promise<any>;
  validateCode: (code: string) => void;
  updateExtractedParams: (params: any[] | null) => void;
}

export interface TemplateSlice {
  selectedId: string;
  onSelect: (templateId: string | null) => void;
}

export function useTemplateSlice(deps: TemplateSliceDeps): TemplateSlice {
  const [searchParams] = useSearchParams();
  const { id: routeId } = useParams<{ id: string }>();
  const setCenterTab = useWorkspaceStore(s => s.setCenterTab);
  const setRightTab = useWorkspaceStore(s => s.setRightTab);
  const [selectedTemplateId, setSelectedTemplateId] = useState<string>('');

  const handleSelectTemplate = useCallback(async (templateId: string | null) => {
    if (!templateId) {
      setSelectedTemplateId('');
      deps.updateExtractedParams(null);
      return;
    }
    setSelectedTemplateId(templateId);
    setCenterTab('code');
    const tpl = await deps.handleLoadTemplate(templateId);
    if (!tpl) { setSelectedTemplateId(''); return; }
    if (tpl.parameters?.length) {
      const params = tpl.parameters.map((p: any) => ({
        name: p.name || '', type: p.type || 'string',
        default: p.default || '', label: p.label || p.name || '',
      }));
      deps.updateExtractedParams(params);
    } else if (tpl.code?.trim()) {
      deps.validateCode(tpl.code);
    }
  }, [deps.handleLoadTemplate, deps.validateCode, deps.updateExtractedParams, setCenterTab]);

  // Load template from URL on mount — supports both ?templateId=X and /:id/edit route param.
  useEffect(() => {
    const tid = routeId || searchParams.get('templateId');
    if (tid) handleSelectTemplate(tid);
  }, [searchParams, routeId, handleSelectTemplate]);

  // Auto-open AI panel when ?ai=1 is present.
  useEffect(() => {
    if (searchParams.get('ai') === '1') {
      setRightTab('chat');
    }
  }, [searchParams, setRightTab]);

  return { selectedId: selectedTemplateId, onSelect: handleSelectTemplate };
}
