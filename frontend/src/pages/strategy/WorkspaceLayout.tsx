import { lazy, Suspense } from 'react';
import type { useStrategyCode } from './hooks/useStrategyCode';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

type CodeCtx = ReturnType<typeof useStrategyCode>;

interface SaveTemplateWrapperProps {
  code: CodeCtx;
}

export function SaveTemplateWrapper({ code }: SaveTemplateWrapperProps) {
  return (
    <Suspense fallback={null}>
      <SaveTemplateModal open={code.saveModalOpen} confirmLoading={code.saveLoading} form={code.saveForm}
        onCancel={() => code.setSaveModalOpen(false)} onOk={code.handleSaveModalOk} />
    </Suspense>
  );
}
