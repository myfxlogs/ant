import { Row, Col, Button, InputNumber } from 'antd';
import { PRESETS } from '../../hooks/useBacktestParams';

const sectionLabel: React.CSSProperties = { fontSize: 9, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 4 };
const fieldLabel: React.CSSProperties = { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 2 };
const narrow: React.CSSProperties = { width: '100%' };

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
  const slippagePct = (slippage * 100).toFixed(4).replace(/\.?0+$/, '');

  return (
    <>
      <div style={sectionLabel}>Cost Parameters</div>
      <Row gutter={8}>
        <Col span={12}>
          <div style={fieldLabel}>Commission %</div>
          <InputNumber size="small" style={narrow} min={0} max={10} step={0.01} precision={4}
            value={commission} onChange={onCommissionChange}
            formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
        </Col>
        <Col span={12}>
          <div style={fieldLabel}>
            Slippage
            <span style={{ fontSize: 8, color: '#bfbfbf', fontWeight: 400, marginLeft: 4, textTransform: 'none' }}>
              {slippagePct}%
            </span>
          </div>
          <InputNumber size="small" style={narrow} min={0} max={10} step={0.0001} precision={4}
            value={slippage} onChange={onSlippageChange} />
        </Col>
      </Row>
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
