import { Result, Button, type ResultProps } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';
import SkeletonCard from './SkeletonCard';
import EmptyState from './EmptyState';

export interface StatusResultProps {
  loading?: boolean;
  error?: Error | string | null;
  empty?: boolean;
  emptyText?: string;
  emptyDescription?: string;
  emptyActionText?: string;
  emptyAction?: () => void;
  onRetry?: () => void;
  children?: ReactNode;
}

export function StatusResult({ loading, error, empty, emptyText, emptyDescription, emptyActionText, emptyAction, onRetry, children }: StatusResultProps) {
  const { t } = useTranslation();
  if (loading) {
    return (
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(280px, 1fr))', gap: 'var(--space-md)' }}>
        {[1, 2, 3].map(i => <SkeletonCard key={i} lines={3} hasIcon />)}
      </div>
    );
  }

  if (error) {
    const msg = typeof error === 'string' ? error : error.message;
    const extra: ResultProps['extra'] = onRetry
      ? <Button type="primary" onClick={onRetry}>{t('common.retry')}</Button>
      : undefined;
    return <Result status="error" title={t('common.error')} subTitle={msg} extra={extra} />;
  }

  if (empty) {
    return (
      <EmptyState
        title={emptyText || t('common.noData')}
        description={emptyDescription}
        actionText={emptyActionText}
        onAction={emptyAction}
      />
    );
  }

  return <>{children}</>;
}
