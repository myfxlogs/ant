import { memo } from 'react';
import { formatPrice } from '@/utils/price';

interface PositionPriceProps {
  symbol: string;
  defaultPrice?: number;
  className?: string;
}

export const PositionPrice = memo<PositionPriceProps>(({ 
  symbol,
  defaultPrice,
  className = '' 
}) => {
  // 直接使用后端返回的 currentPrice（defaultPrice）
  // ProfitUpdateEvent 流会实时更新这个价格
  const price = defaultPrice && defaultPrice > 0 ? defaultPrice : undefined;
  
  return (
    <span className={className}>
      {formatPrice(price, symbol)}
    </span>
  );
});
