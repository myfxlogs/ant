import { Button, InputNumber } from 'antd';
import { PRESETS } from '../../hooks/useBacktestParams';

const paramLabel: React.CSSProperties = { fontSize: 9, color: '#8c8c8c', fontWeight: 600, marginBottom: 4 };
const fieldLabel: React.CSSProperties = { fontSize: 9, color: '#8c8c8c', fontWeight: 600 };
const narrow: React.CSSProperties = { width: 80 };

interface Props {
  commission: number;
  slippage: number;
  onCommissionChange: (v: number | null) => void;
  onSlippageChange: (v: number | null) => void;
  onApplyPreset: (key: 'live_aligned' | 'exploration') => void;
}

export default function BacktestConfigSection({
  commission, slippage,
  onCommissionChange, onSlippageChange, onApplyPreset,
}: Props) {
  return (
    <>
      <div>
        <div style={paramLabel}>Commission (% per lot)</div>
        <InputNumber size="small" style={narrow} min={0} max={10} step={0.01} precision={4}
          value={commission} onChange={onCommissionChange} />
      </div>
      <div>
        <div style={paramLabel}>Slippage (ratio)</div>
        <div style={fieldLabel}>
          Slippage (ratio)
          <span style={{ fontSize: 8, color: '#bfbfbf', marginLeft: 2 }}>
            {(slippage * 100).toFixed(4).replace(/\.?0+$/, '')}%
          </span>
        </div>
        <InputNumber size="small" style={narrow} min={0} max={10} step={0.0001} precision={4}
          value={slippage} onChange={onSlippageChange} />
      </div>
      <div style={{ marginTop: 6, display: 'flex', gap: 4 }}>
        {Object.entries(PRESETS).map(([key, p]) => (
          <Button key={key} size="small"
            onClick={() => onApplyPreset(key as 'live_aligned' | 'exploration')}
            style={{ fontSize: 9, padding: '0 8px', height: 22 }}
          >{p.label}</Button>
        ))}
      </div>
    </>
  );
}
