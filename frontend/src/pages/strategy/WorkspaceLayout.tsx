import { lazy, Suspense } from 'react';

const SaveTemplateModal = lazy(() => import('@/components/strategy/SaveTemplateModal'));

interface SaveTemplateWrapperProps {
  ws: any;
}

export function SaveTemplateWrapper({ ws }: SaveTemplateWrapperProps) {
  return (
    <Suspense fallback={null}>
      <SaveTemplateModal open={ws.code.saveModalOpen} confirmLoading={ws.code.saveLoading} form={ws.code.saveForm}
        onCancel={() => ws.code.setSaveModalOpen(false)} onOk={ws.code.handleSaveModalOk} />
    </Suspense>
  );
}
