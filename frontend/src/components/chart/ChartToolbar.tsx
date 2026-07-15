import { Radio, Tooltip, Button, Space, Tag } from 'antd';
import { BarChartOutlined, AreaChartOutlined, StockOutlined, SettingOutlined, CloseOutlined } from '@ant-design/icons';
import IndicatorPicker from './IndicatorPicker';
import type { IndicatorDef } from '@/stores/chartIndicatorsStore';
import { useTranslation } from 'react-i18next';
import { CHART_TOOLS_AREA_KEY, CHART_TOOLS_CANDLE_KEY, CHART_TOOLS_ERROR_KEY, CHART_TOOLS_LIVE_KEY, CHART_TOOLS_OHLC_KEY, CHART_TOOLS_STATIC_KEY, CHART_TOOLS_STREAM_ACTIVE_KEY, CHART_TOOLS_STREAM_UNAVAILABLE_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';

export const TIMEFRAMES = [
  { label: '1m', value: '1m' }, { label: '5m', value: '5m' },
  { label: '15m', value: '15m' }, { label: '30m', value: '30m' },
  { label: '1h', value: '1h' }, { label: '4h', value: '4h' },
  { label: '1d', value: '1d' }, { label: '1w', value: '1w' },
];

export type ChartType = 'candle_solid' | 'ohlc' | 'area';

export const CHART_TYPES: { key: ChartType; icon: React.ReactNode; label: string }[] = [
  { key: 'candle_solid', icon: <StockOutlined />, label: 'Candle' },
  { key: 'ohlc', icon: <BarChartOutlined />, label: 'OHLC' },
  { key: 'area', icon: <AreaChartOutlined />, label: 'Area' },
];

interface ChartToolbarProps {
  timeframe: string;
  chartType: ChartType;
  streamActive: boolean;
  error: string | null;
  activeIndicators: { instanceId: string; defId: string; params: Record<string, unknown> }[];
  getDef: (defId: string) => IndicatorDef | undefined;
  onTimeframeChange?: (tf: string) => void;
  applyChartType: (type: ChartType) => void;
  onSettingsClick: (instanceId: string) => void;
  onRemoveIndicator: (instanceId: string) => void;
}

export default function ChartToolbar({
  timeframe, chartType, streamActive, error,
  activeIndicators, getDef,
  onTimeframeChange, applyChartType, onSettingsClick, onRemoveIndicator,
}: ChartToolbarProps) {
  const { t } = useTranslation();
  const chartTypeLabels: Record<ChartType, string> = {
    candle_solid: t(CHART_TOOLS_CANDLE_KEY, 'Candle'),
    ohlc: t(CHART_TOOLS_OHLC_KEY, 'OHLC'),
    area: t(CHART_TOOLS_AREA_KEY, 'Area'),
  };
  return (
    <>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 4, flexShrink: 0 }}>
        <div style={{ overflowX: 'auto', whiteSpace: 'nowrap', scrollbarWidth: 'none', msOverflowStyle: 'none', WebkitOverflowScrolling: 'touch' }}>
          <Radio.Group value={timeframe} onChange={e => onTimeframeChange?.(e.target.value)} size="small" optionType="button" buttonStyle="solid">
            {TIMEFRAMES.map(tf => <Radio.Button key={tf.value} value={tf.value}>{tf.label}</Radio.Button>)}
          </Radio.Group>
        </div>
        <Space size={4}>
          {CHART_TYPES.map(ct => (
            <Tooltip key={ct.key} title={chartTypeLabels[ct.key]}>
              <Button size="small" type={chartType === ct.key ? 'primary' : 'default'} icon={ct.icon} onClick={() => applyChartType(ct.key)} />
            </Tooltip>
          ))}
          <Tooltip title={streamActive ? t(CHART_TOOLS_STREAM_ACTIVE_KEY, 'Live bar stream active') : error || t(CHART_TOOLS_STREAM_UNAVAILABLE_KEY, 'Stream unavailable')}>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 11, marginLeft: 4 }}>
              <span style={{ width: 7, height: 7, borderRadius: '50%', background: streamActive ? '#22c55e' : '#ef5350',
                boxShadow: streamActive ? '0 0 4px #22c55e' : '0 0 4px #ef5350' }} />
              <span style={{ color: streamActive ? '#22c55e' : '#ef5350', fontWeight: 600 }}>
                {streamActive ? t(CHART_TOOLS_LIVE_KEY, 'LIVE') : error ? t(CHART_TOOLS_ERROR_KEY, 'ERROR') : t(CHART_TOOLS_STATIC_KEY, 'STATIC')}
              </span>
            </span>
          </Tooltip>
          <IndicatorPicker style={{ marginLeft: 4 }} />
        </Space>
      </div>
      {activeIndicators.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 6, alignItems: 'center', flexShrink: 0 }}>
          {activeIndicators.map((ind) => {
            const def = getDef(ind.defId);
            const paramsStr = def?.params?.length
              ? '(' + def.params.map(p => ind.params[p.key] ?? p.default).join(',') + ')' : '';
            return (
              <Tag key={ind.instanceId} color="processing" style={{ margin: 0, cursor: 'pointer' }}>
                <Space size={2}>
                  <span style={{ fontSize: 11 }} onClick={() => onSettingsClick(ind.instanceId)}>
                    {def?.name || ind.defId}{paramsStr}
                  </span>
                  <Button type="text" size="small" icon={<SettingOutlined />}
                    onClick={(e) => { e.stopPropagation(); onSettingsClick(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 12, height: 12, lineHeight: 1, color: 'inherit' }} />
                  <Button type="text" size="small" danger icon={<CloseOutlined />}
                    onClick={(e) => { e.stopPropagation(); onRemoveIndicator(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 12, height: 12, lineHeight: 1 }} /></Space></Tag>);
          })}
        </div>
      )}
    </>
  );
}
