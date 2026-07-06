import { useState, useCallback, memo } from 'react';
import { RightOutlined } from '@ant-design/icons';

const CollapsibleBlock = memo(function CollapsibleBlock({
  icon, title, subtitle, children, defaultOpen = false,
}: {
  icon: React.ReactNode;
  title: string;
  subtitle?: string;
  children?: React.ReactNode;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const toggle = useCallback(() => setOpen((o) => !o), []);
  return (
    <div style={{ marginBottom: 6 }}>
      <div
        onClick={toggle}
        style={{
          display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer',
          padding: '4px 10px', borderRadius: 6,
          background: 'var(--ant-color-fill-tertiary)',
          fontSize: 11, color: 'var(--ant-color-text-secondary)',
          userSelect: 'none', transition: 'background 0.15s',
        }}
        onMouseEnter={(e) => (e.currentTarget.style.background = 'var(--ant-color-fill-quaternary)')}
        onMouseLeave={(e) => (e.currentTarget.style.background = 'var(--ant-color-fill-tertiary)')}
      >
        <RightOutlined style={{ fontSize: 10, transition: 'transform 0.2s', transform: open ? 'rotate(90deg)' : 'rotate(0deg)' }} />
        <span style={{ flex: 1 }}>{icon} <span style={{ fontWeight: 600 }}>{title}</span></span>
        {subtitle && <span style={{ color: 'var(--ant-color-text-tertiary)' }}>{subtitle}</span>}
      </div>
      {open && <div style={{ padding: '6px 10px 6px 24px' }}>{children}</div>}
    </div>
  );
});

export default CollapsibleBlock;
