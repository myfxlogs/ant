import { Suspense, type ReactNode } from 'react';
import { Spin } from 'antd';
import { ErrorBoundary } from './ErrorBoundary';

export function PageWrapper({ children }: { children: ReactNode }) {
  return (
    <ErrorBoundary>
      <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 64 }}><Spin size="large" /></div>}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}
