import { Form, InputNumber, Switch, Row, Col } from 'antd';
import { useTranslation } from 'react-i18next';
import { STRATEGY_PARAMS_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { paramLabel } from '@/utils/paramLabel';

interface ExtractedParam {
  name: string;
  type: string;
  default: string;
  label: string;
}

interface Props {
  extractedParams: ExtractedParam[];
  strategyParamValues: Record<string, string>;
  onChange: (name: string, value: string) => void;
  language: string;
}

export default function StrategyParamsSection({
  extractedParams, strategyParamValues, onChange, language,
}: Props) {
  const { t } = useTranslation();
  if (extractedParams.length === 0) return null;
  return (
    <div style={{ borderTop: '1px solid var(--ant-color-border)', marginTop: 12, paddingTop: 12 }}>
      <div style={{ fontSize: 12, fontWeight: 700, color: '#595959', marginBottom: 8 }}>
        {t(STRATEGY_PARAMS_KEY)} ({extractedParams.length})
      </div>
      <Row gutter={[12, 8]}>
        {extractedParams.map((p) => {
          const label = paramLabel(p.name, language, null) || p.label || p.name;
          const value = strategyParamValues[p.name] ?? p.default;
          if (p.type === 'bool') {
            return (
              <Col span={8} key={p.name}>
                <Form.Item label={label} style={{ marginBottom: 0 }}>
                  <Switch size="small" checked={value === 'True' || value === 'true'}
                    onChange={(v) => onChange(p.name, v ? 'True' : 'False')} />
                </Form.Item>
              </Col>
            );
          }
          const step = p.type === 'float' ? 0.01 : 1;
          return (
            <Col span={8} key={p.name}>
              <Form.Item label={label} style={{ marginBottom: 0 }}>
                <InputNumber size="small" style={{ width: '100%' }} step={step}
                  value={Number(value)} onChange={(v) => onChange(p.name, String(v ?? p.default))} />
              </Form.Item>
            </Col>
          );
        })}
      </Row>
    </div>
  );
}
