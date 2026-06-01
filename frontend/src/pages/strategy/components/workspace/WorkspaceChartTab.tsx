import PriceChart from '@/components/chart/PriceChart';

interface Props {
  symbol: string;
  timeframe: string;
  onTimeframeChange: (tf: string) => void;
}

export default function WorkspaceChartTab({ symbol, timeframe, onTimeframeChange }: Props) {
  if (!symbol) {
    return (
      <div style={{
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        height: 500, color: '#6b7280', fontSize: 14,
      }}>
        Select a symbol to view chart
      </div>
    );
  }
  return (
    <PriceChart
      symbol={symbol}
      timeframe={timeframe}
      onTimeframeChange={onTimeframeChange}
      height={500}
    />
  );
}
