import { type ReactNode, type CSSProperties } from 'react';

interface GlassCardProps {
  children: ReactNode;
  hover?: boolean;
  className?: string;
  style?: CSSProperties;
  onClick?: () => void;
}

export default function GlassCard({ children, hover = true, className = '', style, onClick }: GlassCardProps) {
  return (
    <div
      className={`glass-card ${hover ? 'glass-card-hover' : ''} ${className}`}
      style={{ cursor: onClick ? 'pointer' : undefined, ...style }}
      onClick={onClick}
    >
      {children}
    </div>
  );
}
