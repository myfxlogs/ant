import { Spin, Empty, Result, Button, type ResultProps } from 'antd';
import type { ReactNode } from 'react';
import { useTranslation } from 'react-i18next';

export interface StatusResultProps {
  loading?: boolean;
  error?: Error | string | null;
  empty?: boolean;
  emptyText?: string;
  onRetry?: () => void;
  children?: ReactNode;
}

export function StatusResult({ loading, error, empty, emptyText, onRetry, children }: StatusResultProps) {
  const { t } = useTranslation();
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
      ? <Button type="primary" onClick={onRetry}>{t('common.retry')}</Button>
      : undefined;
    return <Result status="error" title={t('common.error')} subTitle={msg} extra={extra} />;
  }

  if (empty) {
    return <Empty description={emptyText || t('common.noData')} />;
  }

  return <>{children}</>;
}
