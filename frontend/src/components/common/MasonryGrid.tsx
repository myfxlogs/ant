import { type ReactNode } from 'react';

interface MasonryGridProps {
  children: ReactNode;
  className?: string;
}

export default function MasonryGrid({ children, className = '' }: MasonryGridProps) {
  return <div className={`masonry-grid ${className}`}>{children}</div>;
}

interface MasonryItemProps {
  children: ReactNode;
  className?: string;
}

export function MasonryItem({ children, className = '' }: MasonryItemProps) {
  return <div className={`masonry-item ${className}`}>{children}</div>;
}
