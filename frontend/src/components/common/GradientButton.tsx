import { Button } from 'antd';
import type { ButtonProps } from 'antd';
import type { ReactNode } from 'react';

export const PRIMARY_GRADIENT = 'linear-gradient(135deg, #D4AF37 0%, #B8960B 100%)';

interface GradientButtonProps extends ButtonProps {
  children?: ReactNode;
}

export default function GradientButton({ style, type, children, ...rest }: GradientButtonProps) {
  return (
    <Button
      type={type || 'primary'}
      {...rest}
      style={{
        borderRadius: 10,
        background: PRIMARY_GRADIENT,
        ...style,
      }}
    >
      {children}
    </Button>
  );
}
