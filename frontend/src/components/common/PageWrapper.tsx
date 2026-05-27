import { Suspense, type ReactNode } from 'react';
import { Spin } from 'antd';
import { ErrorBoundary } from './ErrorBoundary';

interface PageWrapperProps {
  children: ReactNode;
  fallback?: ReactNode;
  errorFallback?: ReactNode;
}

const defaultFallback = (
  <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 64 }}>
    <Spin size="large" />
  </div>
);

export function PageWrapper({ children, fallback, errorFallback }: PageWrapperProps) {
  return (
    <ErrorBoundary fallback={errorFallback}>
      <Suspense fallback={fallback ?? defaultFallback}>
        {children}
      </Suspense>
    </ErrorBoundary>
  );
}
