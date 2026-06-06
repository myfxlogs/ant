import { Button, InputNumber, Radio, Switch, Tooltip, Tag } from 'antd';
import { PRESETS } from '../../hooks/useBacktestParams';

const paramLabel: React.CSSProperties = { fontSize: 9, color: '#8c8c8c', fontWeight: 600, marginBottom: 4 };
const fieldLabel: React.CSSProperties = { fontSize: 9, color: '#8c8c8c', fontWeight: 600 };
const narrow: React.CSSProperties = { width: 80 };

interface Props {
  commission: number;
  slippage: number;
  tradeDirection: string;
  strictMode: boolean;
  onCommissionChange: (v: number | null) => void;
  onSlippageChange: (v: number | null) => void;
  onTradeDirectionChange: (v: string) => void;
  onStrictModeChange: (v: boolean) => void;
  onApplyPreset: (key: 'live_aligned' | 'exploration') => void;
}

export default function BacktestConfigSection({
  commission, slippage, tradeDirection, strictMode,
  onCommissionChange, onSlippageChange,
  onTradeDirectionChange, onStrictModeChange, onApplyPreset,
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
      <div>
        <div style={paramLabel}>Trade Direction</div>
        <Radio.Group value={tradeDirection} onChange={e => onTradeDirectionChange(e.target.value)}
          size="small" buttonStyle="solid">
          <Radio.Button value="long">↑ Long</Radio.Button>
          <Radio.Button value="short">↓ Short</Radio.Button>
          <Radio.Button value="both">Both</Radio.Button>
        </Radio.Group>
        <div style={{ marginTop: 12 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <Switch size="small" checked={strictMode} onChange={onStrictModeChange} />
            <span style={{ fontSize: 9, color: '#8c8c8c', fontWeight: 600 }}>
              Strict Mode
              <Tooltip title={strictMode
                ? 'ON: signals confirmed at bar close, executed next bar open'
                : 'OFF: same-bar close execution with 1m sub-resolution'}>
                <Tag color={strictMode ? 'blue' : 'orange'} style={{ fontSize: 8, marginLeft: 4, lineHeight: '14px' }}>
                  {strictMode ? 'ON' : 'OFF'}
                </Tag>
              </Tooltip>
            </span>
          </div>
          <div style={{ fontSize: 8, color: '#8c8c8c', marginTop: 2, lineHeight: '12px' }}>
            {strictMode ? 'Next-bar-open. Standard, conservative.' : 'Same-bar-close + MTF 1m. Higher precision.'}
          </div>
        </div>
      </div>
    </>
  );
}
