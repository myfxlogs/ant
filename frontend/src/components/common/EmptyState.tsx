import { type ReactNode } from 'react';
import { Button } from 'antd';
import { useTranslation } from 'react-i18next';

interface EmptyStateProps {
  icon?: ReactNode;
  title?: string;
  description?: string;
  actionText?: string;
  onAction?: () => void;
}

export default function EmptyState({ icon, title, description, actionText, onAction }: EmptyStateProps) {
  const { t } = useTranslation();
  return (
    <div className="empty-state">
      <div className="empty-state-icon">
        {icon ?? (
          <svg width="36" height="36" viewBox="0 0 24 24" fill="none" stroke="var(--color-text-muted)" strokeWidth="1.5">
            <rect x="3" y="3" width="18" height="18" rx="4" />
            <path d="M3 9h18M9 3v18" opacity="0.4" />
          </svg>
        )}
      </div>
      <div style={{ fontSize: 16, fontWeight: 600, color: 'var(--color-text)', marginBottom: 6 }}>
        {title ?? t('common.noData', { defaultValue: 'No data' })}
      </div>
      {description && (
        <div style={{ fontSize: 14, color: 'var(--color-text-muted)', maxWidth: 360, lineHeight: 1.6 }}>
          {description}
        </div>
      )}
      {actionText && onAction && (
        <Button
          type="primary"
          onClick={onAction}
          style={{ marginTop: 'var(--space-md)', borderRadius: 'var(--radius-sm)' }}
        >
          {actionText}
        </Button>
      )}
    </div>
  );
}
