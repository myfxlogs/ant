import { Form, Select, Space } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  SIMULATION_MODE_KEY, SIGNAL_TIMING_KEY, FILL_RULE_KEY,
  NEXT_BAR_OPEN_KEY, SAME_BAR_CLOSE_KEY, EXEC_MARKET_KEY, EXEC_LIMIT_KEY,
  MT_LIVE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';

interface Props {
  simulationMode: 'KLINE_RANGE' | 'OHLC_PATH';
  signalTiming: 'next_bar_open' | 'same_bar_close';
  fillRule: 'bar_close' | 'market' | 'limit';
  onSimulationModeChange: (v: 'KLINE_RANGE' | 'OHLC_PATH') => void;
  onSignalTimingChange: (v: 'next_bar_open' | 'same_bar_close') => void;
  onFillRuleChange: (v: 'bar_close' | 'market' | 'limit') => void;
}

export default function ExecutionAssumptionsSelectors({
  simulationMode, signalTiming, fillRule,
  onSimulationModeChange, onSignalTimingChange, onFillRuleChange,
}: Props) {
  const { t } = useTranslation();
  return (
    <Space style={{ width: '100%' }} sizes={['33.3%', '33.3%', '33.3%']}>
      <Form.Item label={t(SIMULATION_MODE_KEY)} style={{ width: '33.3%' }}>
        <Select
          value={simulationMode}
          onChange={onSimulationModeChange}
          style={{ width: '100%' }}
          options={[
            { value: 'KLINE_RANGE', label: t(MT_LIVE_KEY) },
            { value: 'OHLC_PATH', label: t('strategy.backtestParams.mtOhlcPath') },
          ]}
        />
      </Form.Item>
      <Form.Item label={t(SIGNAL_TIMING_KEY)} style={{ width: '33.3%' }}>
        <Select
          value={signalTiming}
          onChange={onSignalTimingChange}
          style={{ width: '100%' }}
          options={[
            { value: 'next_bar_open', label: t(NEXT_BAR_OPEN_KEY) },
            { value: 'same_bar_close', label: t(SAME_BAR_CLOSE_KEY) },
          ]}
        />
      </Form.Item>
      <Form.Item label={t(FILL_RULE_KEY)} style={{ width: '33.3%' }}>
        <Select
          value={fillRule}
          onChange={onFillRuleChange}
          style={{ width: '100%' }}
          options={[
            { value: 'bar_close', label: t(SAME_BAR_CLOSE_KEY) },
            { value: 'market', label: t(EXEC_MARKET_KEY) },
            { value: 'limit', label: t(EXEC_LIMIT_KEY) },
          ]}
        />
      </Form.Item>
    </Space>
  );
}
