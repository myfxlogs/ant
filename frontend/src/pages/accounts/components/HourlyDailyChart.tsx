import type React from 'react';
import { useState } from 'react';
import { Bar, CartesianGrid, Cell, ComposedChart, Line, ResponsiveContainer, Tooltip, XAxis, YAxis } from 'recharts';
import { useTranslation } from 'react-i18next';

type Props = {
  hourlyData: Record<string, unknown>[];
  dailyPnLData: Record<string, unknown>[];
  currency: string;
};

type RechartsMouseState = { activeTooltipIndex?: number | string };

function pickTooltipIndex(state: RechartsMouseState, len: number): number | null {
  const idx = state?.activeTooltipIndex;
  if (typeof idx !== 'number') return null;
  if (idx < 0 || idx >= len) return null;
  return idx;
}

export function HourlyDailyChart({ hourlyData, dailyPnLData, currency }: Props) {
  const { t } = useTranslation();
  const [timeView, setTimeView] = useState<'hourly' | 'daily'>('hourly');
  const [selectedHourlyIndex, setSelectedHourlyIndex] = useState(0);
  const [selectedDailyIndex, setSelectedDailyIndex] = useState(0);

  const safeHourly = selectedHourlyIndex < hourlyData.length ? selectedHourlyIndex : 0;
  const safeDaily = selectedDailyIndex < dailyPnLData.length ? selectedDailyIndex : 0;
  const selectedTimePoint = timeView === 'hourly' ? (hourlyData[safeHourly] || null) : (dailyPnLData[safeDaily] || null);

  const formatMoney = (value: number) => `${value >= 0 ? '+' : ''}${Number(value || 0).toFixed(2)} ${currency || 'USD'}`;
  const formatRatio = (value: number) => `${Number(value || 0).toFixed(2)}%`;
  const preventFocus = (e: React.MouseEvent) => e.preventDefault();

  const renderChart = () => {
    const data = timeView === 'hourly' ? hourlyData : dailyPnLData;
    const selectedIdx = timeView === 'hourly' ? safeHourly : safeDaily;
    const setSelected = timeView === 'hourly' ? setSelectedHourlyIndex : setSelectedDailyIndex;
    const xKey = timeView === 'hourly' ? 'hourLabel' : 'date';
    const emptyKey = timeView === 'hourly' ? 'accounts.analytics.empty.hourly' : 'accounts.analytics.empty.dailyPnL';

    if (data.length === 0) {
      return <div className="flex items-center justify-center h-[250px]" style={{ color: '#8A9AA5' }}>{t(emptyKey)}</div>;
    }

    return (
      <div className="outline-none [&_.recharts-wrapper]:!outline-none [&_.recharts-surface]:outline-none" onMouseDown={preventFocus}>
        <ResponsiveContainer width="100%" height={250}>
          <ComposedChart
            data={data}
            onMouseMove={(state) => { const idx = pickTooltipIndex(state as RechartsMouseState, data.length); if (idx != null) setSelected(idx); }}
            onClick={(state) => { const idx = pickTooltipIndex(state as RechartsMouseState, data.length); if (idx != null) setSelected(idx); }}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
            <XAxis dataKey={xKey} stroke="#8A9AA5" fontSize={10} />
            <YAxis yAxisId="left" stroke="#8A9AA5" fontSize={10} />
            <YAxis yAxisId="right" orientation="right" stroke="#8A9AA5" fontSize={10} />
            <Tooltip cursor={false} wrapperStyle={{ pointerEvents: 'none' }} contentStyle={{ background: '#FFFFFF', border: '1px solid #E5E7EB', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0, 0, 0, 0.08)' }} />
            <Bar yAxisId="left" dataKey="trades" radius={[3, 3, 0, 0]} barSize={timeView === 'hourly' ? 18 : 24} isAnimationActive={false}>
              {data.map((_: unknown, index: number) => (
                <Cell key={`cell-${index}`} fill={index === selectedIdx ? '#2B6CB0' : (timeView === 'hourly' ? '#64B5F6' : '#4DB6AC')} style={{ cursor: 'pointer' }} onClick={() => setSelected(index)} />
              ))}
            </Bar>
            <Line yAxisId="right" type="monotone" dataKey="profit" stroke="#FF9800" strokeWidth={2} dot={false} activeDot={false} style={{ pointerEvents: 'none' }} isAnimationActive={false} />
          </ComposedChart>
        </ResponsiveContainer>
      </div>
    );
  };

  return (
    <div className="rounded-2xl p-5 mb-6" style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}>
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-semibold" style={{ color: '#141D22' }}>
          {t('accounts.analytics.hourlyTitle')} / {t('accounts.analytics.dailyPnLTitle')}
        </h2>
        <div className="inline-flex rounded-lg p-1" style={{ background: '#F5F7F9' }}>
          {(['hourly', 'daily'] as const).map((tab) => (
            <button key={tab} onClick={() => setTimeView(tab)}
              className="px-3 py-1.5 rounded text-xs font-semibold transition-all"
              style={{ background: timeView === tab ? '#FFFFFF' : 'transparent', color: timeView === tab ? '#141D22' : '#8A9AA5', boxShadow: timeView === tab ? '0 1px 3px rgba(0, 0, 0, 0.08)' : 'none' }}>
              {t(`accounts.analytics.advancedTabs.${tab}`)}
            </button>
          ))}
        </div>
      </div>
      <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
        <div className="xl:col-span-2 min-w-0">{renderChart()}</div>
        <div className="rounded-xl p-3 border" style={{ borderColor: '#E5E7EB', background: '#F8FAFC' }}>
          <div className="text-sm font-semibold mb-2" style={{ color: '#1F2937' }}>
            {timeView === 'hourly' ? (selectedTimePoint?.hourLabel || '--') : `${selectedTimePoint?.date || '--'} ${selectedTimePoint?.day || ''}`}
          </div>
          <div className="text-xs space-y-1.5" style={{ color: '#475467' }}>
            <div>{t('accounts.analytics.timeDetail.lots')}: <span className="font-semibold">{Number(selectedTimePoint?.lots || 0).toFixed(2)}</span></div>
            <div>{t('accounts.analytics.timeDetail.trades')}: <span className="font-semibold">{Number(selectedTimePoint?.trades || 0)}</span></div>
            <div>{t('accounts.analytics.timeDetail.profitAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.profit || 0))}</span></div>
            <div>{t('accounts.analytics.timeDetail.balance')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.balance || 0))}</span></div>
            <div>{t('accounts.analytics.timeDetail.profitFactor')}: <span className="font-semibold">{Number(selectedTimePoint?.profitFactor || 0).toFixed(2)}</span></div>
            <div>{t('accounts.analytics.timeDetail.maxFloatingLossAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.maxFloatingLossAmount || 0))}</span></div>
            <div>{t('accounts.analytics.timeDetail.maxFloatingLossRatio')}: <span className="font-semibold">{formatRatio(Number(selectedTimePoint?.maxFloatingLossRatio || 0))}</span></div>
            <div>{t('accounts.analytics.timeDetail.maxFloatingProfitAmount')}: <span className="font-semibold">{formatMoney(Number(selectedTimePoint?.maxFloatingProfitAmount || 0))}</span></div>
            <div>{t('accounts.analytics.timeDetail.maxFloatingProfitRatio')}: <span className="font-semibold">{formatRatio(Number(selectedTimePoint?.maxFloatingProfitRatio || 0))}</span></div>
          </div>
        </div>
      </div>
    </div>
  );
}
