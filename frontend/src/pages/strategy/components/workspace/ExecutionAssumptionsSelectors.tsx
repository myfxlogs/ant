import { Form, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import {
  REPLAY_MODEL_KEY, REPLAY_OHLC_PATH_KEY, REPLAY_KLINE_RANGE_KEY, REPLAY_OPEN_PRICE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';

export type ReplayModel = 'ohlc_path' | 'kline_range' | 'open_price';

interface Props {
  value: ReplayModel;
  onChange: (v: ReplayModel) => void;
}

export default function ExecutionAssumptionsSelectors({ value, onChange }: Props) {
  const { t } = useTranslation();
  return (
    <Form.Item label={t(REPLAY_MODEL_KEY)}>
      <Select
        value={value}
        onChange={onChange}
        style={{ width: '100%' }}
        options={[
          { value: 'ohlc_path', label: t(REPLAY_OHLC_PATH_KEY) },
          { value: 'kline_range', label: t(REPLAY_KLINE_RANGE_KEY) },
          { value: 'open_price', label: t(REPLAY_OPEN_PRICE_KEY) },
        ]}
      />
    </Form.Item>
  );
}
