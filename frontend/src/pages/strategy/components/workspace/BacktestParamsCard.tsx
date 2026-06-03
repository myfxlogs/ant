import { Button, Row, Col, InputNumber, Radio, Switch, DatePicker, Tooltip } from 'antd';
import { PlayCircleOutlined, SettingOutlined, CaretUpOutlined, CaretDownOutlined } from '@ant-design/icons';
import dayjs from 'dayjs';

interface Props {
  initialCapital: number; onInitialCapitalChange: (v: number | null) => void;
  leverage: number; onLeverageChange: (v: number | null) => void;
  commission: number; onCommissionChange: (v: number | null) => void;
  slippage: number; onSlippageChange: (v: number | null) => void;
  startDate: string; onStartDateChange: (v: string) => void;
  endDate: string; onEndDateChange: (v: string) => void;
  tradeDirection: string; onTradeDirectionChange: (d: string) => void;
  highPrecision: boolean; onHighPrecisionChange: (v: boolean) => void;
  canRun: boolean; running: boolean; onRunBacktest: () => void;
  datePresets: Array<{ key: string; label: string; months: number }>;
  datePresetKey: string; onApplyDatePreset: (p: { key: string; months: number }) => void;
  expanded: boolean; onExpandedChange: (v: boolean) => void;
}

const fieldLabel: React.CSSProperties = { fontSize: 9, fontWeight: 600, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 2 };
const paramLabel: React.CSSProperties = { fontSize: 9, fontWeight: 700, color: '#8c8c8c', textTransform: 'uppercase', marginBottom: 4 };
const narrow = { width: '100%' };

export default function BacktestParamsCard(props: Props) {
  const {
    initialCapital, onInitialCapitalChange, leverage, onLeverageChange,
    commission, onCommissionChange, slippage, onSlippageChange,
    startDate, onStartDateChange, endDate, onEndDateChange,
    tradeDirection, onTradeDirectionChange, highPrecision, onHighPrecisionChange,
    canRun, running, onRunBacktest, datePresets = [], datePresetKey, onApplyDatePreset,
    expanded, onExpandedChange,
  } = props;

  return (
    <div style={{
      borderBottom: '1px solid #e8e8e8', background: '#fafbfc',
      borderTop: '1px solid #e8e8e8',
    }}>
      {/* Header */}
      <div onClick={() => onExpandedChange(!expanded)} style={{
        display: 'flex', alignItems: 'center', justifyContent: 'space-between',
        padding: '9px 14px', cursor: 'pointer', userSelect: 'none',
        background: 'linear-gradient(180deg, #f8fafc 0%, #f1f5f9 100%)',
      }} onKeyUp={e => e.key === 'Enter' && onExpandedChange(!expanded)} role="button" tabIndex={0}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <PlayCircleOutlined style={{ color: '#1890ff' }} />
          <span style={{ fontSize: 12, fontWeight: 700, color: '#262626' }}>Backtest Parameters</span>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }} onClick={e => e.stopPropagation()}>
          <Tooltip title="Settings">
            <Button size="small" type="text" icon={<SettingOutlined />} style={{ borderRadius: 6 }} />
          </Tooltip>
          <Button type="primary" size="small" loading={running} disabled={!canRun}
            onClick={onRunBacktest}
            style={{ borderRadius: 6, fontWeight: 600, boxShadow: '0 2px 8px rgba(24,144,255,0.25)' }}>
            ▶ Run Backtest
          </Button>
          <span style={{ fontSize: 9, color: '#8c8c8c', cursor: 'pointer' }}>
            {expanded ? <CaretUpOutlined /> : <CaretDownOutlined />}
          </span>
        </div>
      </div>

      {expanded && (
        <div style={{ padding: '12px 14px', display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: 14 }}>
          {/* Date Range */}
          <div>
            <div style={paramLabel}>Date Range</div>
            <div style={{ display: 'flex', gap: 4, marginBottom: 6 }}>
              {datePresets.map(p => (
                <Button key={p.key} size="small"
                  type={datePresetKey === p.key ? 'primary' : 'default'}
                  onClick={() => onApplyDatePreset(p)}
                >{p.label}</Button>
              ))}
            </div>
            <Row gutter={8}>
              <Col span={12}>
                <DatePicker size="small" style={narrow} value={startDate ? dayjs(startDate) : null}
                  onChange={(d) => d && onStartDateChange(d.format('YYYY-MM-DD'))} placeholder="Start" />
              </Col>
              <Col span={12}>
                <DatePicker size="small" style={narrow} value={endDate ? dayjs(endDate) : null}
                  onChange={(d) => d && onEndDateChange(d.format('YYYY-MM-DD'))} placeholder="End" />
              </Col>
            </Row>
          </div>

          {/* Capital & Leverage */}
          <div>
            <div style={paramLabel}>Capital & Leverage</div>
            <Row gutter={8}>
              <Col span={12}>
                <div style={fieldLabel}>Initial Capital</div>
                <InputNumber size="small" style={narrow} min={100} step={1000}
                  value={initialCapital} onChange={onInitialCapitalChange} />
              </Col>
              <Col span={12}>
                <div style={fieldLabel}>Leverage</div>
                <InputNumber size="small" style={narrow} min={1} max={125} step={1}
                  value={leverage} onChange={onLeverageChange} />
              </Col>
            </Row>
            <Row gutter={8} style={{ marginTop: 6 }}>
              <Col span={12}>
                <div style={fieldLabel}>Commission %</div>
                <InputNumber size="small" style={narrow} min={0} max={10} step={0.01}
                  value={commission} onChange={onCommissionChange} />
              </Col>
              <Col span={12}>
                <div style={fieldLabel}>Slippage %</div>
                <InputNumber size="small" style={narrow} min={0} max={10} step={0.01}
                  value={slippage} onChange={onSlippageChange} />
              </Col>
            </Row>
          </div>

          {/* Trade Direction */}
          <div>
            <div style={paramLabel}>Trade Direction</div>
            <Radio.Group value={tradeDirection} onChange={e => onTradeDirectionChange(e.target.value)}
              size="small" buttonStyle="solid">
              <Radio.Button value="long">↑ Long</Radio.Button>
              <Radio.Button value="short">↓ Short</Radio.Button>
              <Radio.Button value="both">Both</Radio.Button>
            </Radio.Group>
            <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8 }}>
              <Switch size="small" checked={highPrecision} onChange={onHighPrecisionChange} />
              <span style={{ fontSize: 9, color: '#8c8c8c', fontWeight: 600 }}>High-Precision M1</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
