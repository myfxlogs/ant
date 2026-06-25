import { Modal, Row, Col, InputNumber, Switch } from 'antd';
import { useTranslation } from 'react-i18next';
import { STRATEGY_PARAMS_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import type { ExtractedParam } from './useBacktestRunner';

const S = {
  fieldLabel: { fontSize: 12, fontWeight: 500, color: '#8c8c8c', marginBottom: 4 },
  narrow: { width: '100%' },
};

interface Props {
  open: boolean;
  params: ExtractedParam[];
  values: Record<string, string>;
  onClose: () => void;
  onChange: (name: string, value: string) => void;
}

export default function StrategyParamsModal({ open, params, values, onClose, onChange }: Props) {
  const { t } = useTranslation();
  if (params.length === 0) return null;

  return (
    <Modal
      title={t(STRATEGY_PARAMS_KEY)}
      open={open}
      onCancel={onClose}
      onOk={onClose}
      width={640}
      destroyOnClose
    >
      <div style={{ maxHeight: '60vh', overflowY: 'auto', paddingRight: 4 }}>
        <Row gutter={[12, 8]}>
          {params.map((p) => {
            const value = values[p.name] ?? p.default;
            if (p.type === 'bool') {
              return (
                <Col span={8} key={p.name}>
                  <div style={S.fieldLabel}>{p.label || p.name}</div>
                  <Switch size="small" checked={value === 'True' || value === 'true'}
                    onChange={(v) => onChange(p.name, v ? 'True' : 'False')} />
                </Col>
              );
            }
            const step = p.type === 'float' ? 0.01 : 1;
            return (
              <Col span={8} key={p.name}>
                <div style={S.fieldLabel}>{p.label || p.name}</div>
                <InputNumber size="small" style={S.narrow} step={step}
                  value={Number(value)} onChange={(v) => onChange(p.name, String(v ?? p.default))} />
              </Col>
            );
          })}
        </Row>
      </div>
    </Modal>
  );
}
