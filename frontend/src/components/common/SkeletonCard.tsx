interface SkeletonCardProps {
  lines?: number;
  hasIcon?: boolean;
  hasChart?: boolean;
}

export default function SkeletonCard({ lines = 3, hasIcon = true, hasChart = false }: SkeletonCardProps) {
  return (
    <div className="glass-card" style={{ padding: 'var(--space-md)' }}>
      {hasIcon && (
        <div className="flex items-center gap-3 mb-4">
          <div className="skeleton-block" style={{ width: 40, height: 40, borderRadius: 'var(--radius-md)' }} />
          <div className="flex-1">
            <div className="skeleton-block" style={{ width: '60%', height: 16, marginBottom: 6 }} />
            <div className="skeleton-block" style={{ width: '40%', height: 12 }} />
          </div>
        </div>
      )}
      {hasChart && (
        <div className="skeleton-block mb-4" style={{ width: '100%', height: 120, borderRadius: 'var(--radius-md)' }} />
      )}
      {Array.from({ length: lines }).map((_, i) => (
        <div
          key={i}
          className="skeleton-block"
          style={{ width: `${90 - i * 10}%`, height: 14, marginBottom: i < lines - 1 ? 10 : 0 }}
        />
      ))}
    </div>
  );
}
