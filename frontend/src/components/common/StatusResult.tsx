import { Spin, Empty, Result, Button, type ResultProps } from 'antd';
import type { ReactNode } from 'react';

export interface StatusResultProps {
  loading?: boolean;
  error?: Error | string | null;
  empty?: boolean;
  emptyText?: string;
  onRetry?: () => void;
  children?: ReactNode;
}

export function StatusResult({ loading, error, empty, emptyText, onRetry, children }: StatusResultProps) {
  if (loading) {
    return (
      <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', padding: 64 }}>
        <Spin size="large" />
      </div>
    );
  }

  if (error) {
    const msg = typeof error === 'string' ? error : error.message;
    const extra: ResultProps['extra'] = onRetry
      ? <Button type="primary" onClick={onRetry}>Retry</Button>
      : undefined;
    return <Result status="error" title="Error" subTitle={msg} extra={extra} />;
  }

  if (empty) {
    return <Empty description={emptyText || 'No data'} />;
  }

  return <>{children}</>;
}
