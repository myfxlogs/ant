import { useMemo } from 'react';
import { Button, InputNumber, Row, Col, Segmented, DatePicker, Radio, Switch, Select } from 'antd';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import {
  BOTH_KEY, CAPITAL_KEY, COMMISSION_KEY, CURRENT_DRAFT_KEY, DATE_RANGE_KEY, DIRECTION_KEY,
  END_DATE_KEY, EXECUTION_KEY, LEVERAGE_KEY, LONG_KEY, LOT_SIZE_KEY,
  MORE_KEY, SHORT_KEY, SLIPPAGE_KEY, START_DATE_KEY, STRATEGY_KEY, STRATEGY_PARAMS_KEY, STRICT_MODE_KEY,
} from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { VALIDATE_TO_SEE_PARAMS_KEY } from '@/gen/ant/v1/i18n/strategy_code_assist_keys';
import { COMMON_EDIT_KEY } from '@/gen/ant/v1/i18n/base_keys';
import StrategyParamsModal from './StrategyParamsModal';
import { paramLabel } from '@/utils/paramLabel';
import { DATE_PRESETS, PRESETS } from '@/pages/strategy/hooks/backtestParamHelpers';
import type { StrategyTemplate } from '@/client/strategy';
import type { useBacktestRunner, BacktestRunnerInputs } from './useBacktestRunner';

const S = {
  sectionLabel: { fontSize: 12, fontWeight: 700, color: '#595959', marginBottom: 6 },
  fieldLabel: { fontSize: 12, fontWeight: 500, color: '#8c8c8c', marginBottom: 4 },
  narrow: { width: '100%' },
};

interface Props {
  runner: ReturnType<typeof useBacktestRunner>;
  inputs: BacktestRunnerInputs;
  templates: {
    list: StrategyTemplate[];
    loading: boolean;
    selectedId: string;
    onSelect: (id: string | null) => void;
  };
}

export default function BacktestParamsTab({ runner, inputs, templates }: Props) {
  const { t, i18n } = useTranslation();
  const loc = i18n.language;

  const tplList = templates?.list || [];
  const selectedTpl = tplList.find((tpl: StrategyTemplate) => tpl.id === templates?.selectedId);
  const i18nData = useMemo(() => {
    return selectedTpl?.i18n ?? null;
  }, [selectedTpl?.i18n]);

  const slippagePct = (runner.slippage * 100).toFixed(4).replace(/\.?0+$/, '');

  return (
    <div>
      {/* Strategy + Date + Strict + Presets — single row */}
      <div style={{ marginBottom: 8, display: 'flex', alignItems: 'flex-end', gap: 8, flexWrap: 'wrap' }}>
        <div>
          <div style={S.fieldLabel}>{t(STRATEGY_KEY)}</div>
          <Select size="small" style={{ width: 180 }}
            loading={templates.loading}
            value={templates.selectedId || '__draft__'}
            options={[
              { value: '__draft__', label: t(CURRENT_DRAFT_KEY) },
              ...(tplList.length > 0 ? [{ value: '__sep__', label: '──────────────', disabled: true }] : []),
              ...tplList.map((tpl: StrategyTemplate) => ({ value: tpl.id, label: tpl.name })),
            ]}
            onChange={(val) => {
              if (val === '__draft__') templates.onSelect(null);
              else if (val !== '__sep__') templates.onSelect(val);
            }}
          />
        </div>
        <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
          <div style={S.fieldLabel}>{t(DATE_RANGE_KEY)}</div>
          <div style={{ display: 'flex', gap: 4 }}>
            <Segmented size="small" value={runner.datePreset}
              onChange={(v) => {
                const p = DATE_PRESETS.find(d => d.key === v);
                if (p) runner.applyDatePreset(p);
              }}
              options={DATE_PRESETS.map(p => ({ value: p.key, label: p.label }))}
            />
            <DatePicker size="small" style={{ width: 110 }}
              value={runner.startDate ? dayjs(runner.startDate) : null}
              onChange={(d) => d && runner.setStartDate(d.format('YYYY-MM-DD'))}
              placeholder={t(START_DATE_KEY)} />
            <DatePicker size="small" style={{ width: 110 }}
              value={runner.endDate ? dayjs(runner.endDate) : null}
              onChange={(d) => d && runner.setEndDate(d.format('YYYY-MM-DD'))}
              placeholder={t(END_DATE_KEY)} />
          </div>
        </div>
        <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <span style={S.fieldLabel}>{t(STRICT_MODE_KEY)}</span>
            <Switch size="small" checked={runner.strictMode} onChange={runner.setStrictMode} />
          </div>
        </div>
        <div style={{ borderLeft: '1px solid #e8e8e8', paddingLeft: 8 }}>
          {(Object.keys(PRESETS) as Array<keyof typeof PRESETS>).map((key) => (
            <Button key={key} size="small" onClick={() => runner.applyPreset(key)}
              style={{ fontSize: 12, padding: '0 8px', height: 26 }}>
              {t(`strategy.backtestParams.presets.${key === 'live_aligned' ? 'liveAligned' : 'exploration'}`)}
            </Button>
          ))}
        </div>
      </div>

      {/* Standard params — compact single row */}
      <div style={S.sectionLabel}>{t(EXECUTION_KEY)}</div>
      <Row gutter={6} style={{ marginBottom: 6 }}>
        <Col span={4}>
          <div style={S.fieldLabel}>{t(CAPITAL_KEY)}</div>
          <InputNumber size="small" style={S.narrow} min={100} step={1000}
            value={runner.initialCapital} onChange={(v) => runner.setInitialCapital(v ?? 10000)}
            formatter={v => `$ ${v}`.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}
            parser={v => v!.replace(/\$\s?|(,*)/g, '') as unknown as number} />
        </Col>
        <Col span={3}>
          <div style={S.fieldLabel}>{t(LEVERAGE_KEY)}</div>
          <InputNumber size="small" style={S.narrow} min={1} step={1}
            value={runner.leverage} onChange={(v) => runner.setLeverage(v ?? 1)}
            formatter={v => `${v}x`} parser={v => v!.replace('x', '') as unknown as number} />
        </Col>
        <Col span={3}>
          <div style={S.fieldLabel}>{t(LOT_SIZE_KEY)}</div>
          <InputNumber size="small" style={S.narrow} min={0.01} max={100} step={0.01}
            value={runner.lotSize} onChange={(v) => runner.setLotSize(v ?? 0.01)} />
        </Col>
        <Col span={5}>
          <div style={S.fieldLabel}>{t(COMMISSION_KEY)}
            <span style={{ fontSize: 12, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>%</span>
          </div>
          <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.01} precision={4}
            value={runner.commission} onChange={(v) => runner.setCommission(v ?? 0.001)}
            formatter={v => `${v}%`} parser={v => v!.replace('%', '') as unknown as number} />
        </Col>
        <Col span={5}>
          <div style={S.fieldLabel}>{t(SLIPPAGE_KEY)}
            <span style={{ fontSize: 12, color: '#bfbfbf', fontWeight: 400, marginLeft: 3 }}>{slippagePct}%</span>
          </div>
          <InputNumber size="small" style={S.narrow} min={0} max={10} step={0.0001} precision={4}
            value={runner.slippage} onChange={(v) => runner.setSlippage(v ?? 0)} />
        </Col>
        <Col span={4}>
          <div style={S.fieldLabel}>{t(DIRECTION_KEY)}</div>
          <Radio.Group value={runner.tradeDirection} onChange={e => runner.setTradeDirection(e.target.value)}
            size="small" buttonStyle="solid" style={{ display: 'flex' }}>
            <Radio.Button value="long" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(LONG_KEY)}</Radio.Button>
            <Radio.Button value="short" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(SHORT_KEY)}</Radio.Button>
            <Radio.Button value="both" style={{ flex: 1, textAlign: 'center', fontSize: 12 }}>{t(BOTH_KEY)}</Radio.Button>
          </Radio.Group>
        </Col>
      </Row>
      {/* Strategy params — title row + tag preview (TradingView Settings style) */}
      <div style={{ marginTop: 10, borderTop: '1px solid #f0f0f0', paddingTop: 8 }}>
        {runner.extractedParams.length > 0 ? (
          <>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
              <span style={S.sectionLabel}>{t(STRATEGY_PARAMS_KEY)} ({runner.extractedParams.length})</span>
              <Button size="small" onClick={() => runner.setStrategyParamsModalOpen(true)}>
                {t(COMMON_EDIT_KEY)}
              </Button>
            </div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '2px 12px' }}>
              {runner.extractedParams.slice(0, 8).map((p) => {
                const v = runner.strategyParamValues[p.name] ?? p.default;
                const label = paramLabel(p.name, loc, i18nData) || p.label || p.name;
                return (
                  <span key={p.name} style={{ fontSize: 11, color: '#595959', whiteSpace: 'nowrap' }}>
                    <span style={{ color: '#8c8c8c' }}>{label}</span>
                    <span style={{ fontWeight: 500 }}>={v}</span>
                  </span>
                );
              })}
              {runner.extractedParams.length > 8 && (
                <span style={{ fontSize: 11, color: '#bfbfbf' }}>
                  +{runner.extractedParams.length - 8} {t(MORE_KEY)}
                </span>
              )}
            </div>
          </>
        ) : (
          <span style={{ fontSize: 11, color: '#bfbfbf' }}>
            {t(VALIDATE_TO_SEE_PARAMS_KEY)}
          </span>
        )}
      </div>
      <StrategyParamsModal
        open={runner.strategyParamsModalOpen}
        params={runner.extractedParams}
        values={runner.strategyParamValues}
        i18nData={i18nData}
        onClose={() => runner.setStrategyParamsModalOpen(false)}
        onChange={runner.setParam}
      />
    </div>
  );
}
