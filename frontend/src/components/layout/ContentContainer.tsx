import type { ReactNode } from 'react';

interface ContentContainerProps {
  children: ReactNode;
}

export default function ContentContainer({ children }: ContentContainerProps) {
  return (
    <div
      style={{
        maxWidth: 1360,
        margin: '0 auto',
        width: '100%',
      }}
    >
      {children}
    </div>
  );
}
