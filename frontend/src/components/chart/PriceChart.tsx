import { useEffect, useRef, useCallback, useState } from 'react';
import { Radio, Spin, Tooltip, Button, Space, Tag } from 'antd';
import { BarChartOutlined, AreaChartOutlined, StockOutlined, SettingOutlined, CloseOutlined } from '@ant-design/icons';
import { init, dispose } from 'klinecharts';
import type { Chart } from 'klinecharts';
import DARK_THEME from './chartTheme';
import DrawingToolbar from './DrawingToolbar';
import IndicatorPicker from './IndicatorPicker';
import { marketApi } from '@/client/market';
import { useChartData, toChartBar } from './useChartData';
import { useChartIndicatorsStore, KLINECHARTS_MAP } from '@/stores/chartIndicatorsStore';
import IndicatorSettingsModal from './IndicatorSettingsModal';
import './BidAskIndicator';
import './BacktestTradeOverlay';

const TIMEFRAMES = [
  { label: '1m', value: '1m' }, { label: '5m', value: '5m' },
  { label: '15m', value: '15m' }, { label: '30m', value: '30m' },
  { label: '1h', value: '1h' }, { label: '4h', value: '4h' },
  { label: '1d', value: '1d' }, { label: '1w', value: '1w' },
];

type ChartType = 'candle_solid' | 'ohlc' | 'area';

const CHART_TYPES: { key: ChartType; icon: React.ReactNode; label: string }[] = [
  { key: 'candle_solid', icon: <StockOutlined />, label: 'Candle' },
  { key: 'ohlc', icon: <BarChartOutlined />, label: 'OHLC' },
  { key: 'area', icon: <AreaChartOutlined />, label: 'Area' },
];

interface Props {
  symbol: string;
  timeframe?: string;
  onTimeframeChange?: (tf: string) => void;
  accountId?: string;
  onChartReady?: (chart: Chart | null) => void;
  trades?: Array<{
    side: string; openPrice: number; closePrice?: number;
    openTime: number; closeTime?: number; pnl?: number;
  }>;
}

export default function PriceChart({ symbol, timeframe = '1h', onTimeframeChange, accountId, onChartReady, trades }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<Chart | null>(null);
  const [tooNarrow, setTooNarrow] = useState(false);
  const [chartType, setChartType] = useState<ChartType>('candle_solid');

  const { bars, loading, error, streamActive, loadingMore, loadedAll } = useChartData(symbol, timeframe, accountId, chartRef);
  const activeIndicators = useChartIndicatorsStore((s) => s.active);
  const getDef = useChartIndicatorsStore((s) => s.getDef);
  const removeIndicator = useChartIndicatorsStore((s) => s.removeIndicator);
  const [editingIndId, setEditingIndId] = useState<string | null>(null);
  const createdRef = useRef<Map<string, { paneId: string; name: string; paramsKey: string }>>(new Map());

  // Sync Zustand active indicators → klinecharts chart.
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    const prev = createdRef.current;
    const next = new Map<string, { paneId: string; name: string; paramsKey: string }>();

    for (const ind of activeIndicators) {
      const km = KLINECHARTS_MAP[ind.defId];
      if (!km) continue;

      const paramsKey = JSON.stringify(ind.params);
      const existing = prev.get(ind.instanceId);

      if (existing && existing.name === km.name && existing.paramsKey === paramsKey) {
        next.set(ind.instanceId, existing);
        prev.delete(ind.instanceId);
        continue;
      }

      // Remove old instance if params changed.
      if (existing) {
        try { chart.removeIndicator(existing.paneId, existing.name); } catch { /* */ }
      }

      const def = getDef(ind.defId);
      const isStack = def?.kind === 'overlay';
      const calcParams = km.buildParams(ind.params);
      try {
        const paneId = chart.createIndicator(km.name, isStack, { id: `ind_${ind.instanceId}` }) as unknown as string;
        if (paneId) {
          if (calcParams.length > 0) {
            try { (chart as any).setIndicatorCalcParams?.(paneId, km.name, calcParams); } catch { /* */ }
          }
          next.set(ind.instanceId, { paneId, name: km.name, paramsKey });
        }
      } catch { /* indicator not supported */ }
    }

    for (const [id, info] of prev) {
      try { chart.removeIndicator(info.paneId, info.name); } catch { /* */ }
    }
    createdRef.current = next;
  }, [activeIndicators]);

  // Responsive
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)');
    setTooNarrow(mq.matches);
    const h = (e: MediaQueryListEvent) => setTooNarrow(e.matches);
    mq.addEventListener('change', h);
    return () => mq.removeEventListener('change', h);
  }, []);

  // applyNewData when bars change
  useEffect(() => {
    if (!chartRef.current) return;
    chartRef.current.applyNewData(bars.map(toChartBar));
  }, [bars]);

  // Create chart on mount — CSS flex gives the container its height
  useEffect(() => {
    if (!containerRef.current) return;
    const el = containerRef.current;
    el.style.width = '100%';
    el.style.height = '100%';
    const chart = init(el, { styles: DARK_THEME });
    if (!chart) { console.error('klinecharts init returned null'); return; }
    chartRef.current = chart;
    try { chart.createIndicator('VOL', false, { id: 'volume_pane',
      styles: { bars: { upColor: 'rgba(38,166,154,0.6)', downColor: 'rgba(239,83,80,0.6)' }, lines: [] } });
    } catch { /* ignore */ }
    try { chart.createIndicator('BIDASK', true, { id: 'candle_pane' }); } catch { /* */ }
    onChartReady?.(chart);
    // ResizeObserver: keep chart canvas in sync with container size
    const ro = new ResizeObserver(() => chartRef.current?.resize());
    ro.observe(el);
    return () => { ro.disconnect(); onChartReady?.(null); dispose(el); chartRef.current = null; };
  }, []);

  // Chart type
  useEffect(() => {
    if (!chartRef.current) return;
    chartRef.current.setStyles({ candle: { ...DARK_THEME.candle, type: chartType } });
  }, [chartType]);

  const tradeOverlayRef = useRef<string | null>(null);

  // Trade markers overlay — show buy/sell arrows on chart.
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    // Remove previous overlay.
    if (tradeOverlayRef.current) {
      try { chart.removeOverlay(tradeOverlayRef.current); } catch { /* */ }
      tradeOverlayRef.current = null;
    }

    if (!trades?.length) return;

    // Map trade times to bar indices in the chart's current data.
    const chartData = (chart as any).getDataList?.() || [];
    const timeToIdx = new Map<number, number>();
    chartData.forEach((d: any, i: number) => {
      if (d?.timestamp) timeToIdx.set(Math.floor(d.timestamp / 1000), i);
    });

    const tradePoints = trades
      .filter((t) => timeToIdx.has(t.openTime))
      .map((t) => ({
        ...t,
        entryIndex: timeToIdx.get(t.openTime) ?? -1,
        exitIndex: t.closeTime ? timeToIdx.get(t.closeTime) ?? -1 : -1,
      }))
      .filter((t) => t.entryIndex >= 0);

    if (!tradePoints.length) return;

    try {
      const id = chart.createOverlay({
        name: 'backtest_trades',
        points: tradePoints.map((t) => ({ dataIndex: t.entryIndex })),
        extendData: tradePoints,
        lock: true,
      } as any) as unknown as string;
      if (id) tradeOverlayRef.current = id;
    } catch { /* */ }
  }, [trades, bars]);

  const applyChartType = useCallback((type: ChartType) => setChartType(type), []);

  // Load more on scroll-left — directly append to chart
  const handleLoadMore = useCallback((timestamp: number | null) => {
    const chart = chartRef.current;
    const oldest = barsRef.current[0];
    if (!chart || !timestamp || !oldest || loadingMore.current || loadedAll.current) return;
    if (timestamp >= oldest.time) return;
    loadingMore.current = true;
    marketApi.getKlines({ symbol: marketApi.resolveSymbol(symbol), timeframe, count: 300, before: oldest.time, accountId })
      .then((older) => {
        if (older.length === 0) { loadedAll.current = true; return; }
        chart.applyNewData(older.map(toChartBar), true);
      }).catch(() => { /* silent */ }).finally(() => { loadingMore.current = false; });
  }, [symbol, timeframe, accountId, loadingMore, loadedAll]);

  // Listen for scroll/pan to trigger load-more
  useEffect(() => {
    const chart = chartRef.current, container = containerRef.current;
    if (!chart || !container) return;
    let timer: number | null = null;
    const onInteraction = () => {
      if (timer != null) window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        try { const r = (chart as any).getVisibleRange?.() as { from?: number } | null; if (r?.from != null) handleLoadMore(r.from); } catch { /* */ }
      }, 300);
    };
    container.addEventListener('wheel', onInteraction, { passive: true });
    container.addEventListener('pointerdown', onInteraction, { passive: true });
    return () => { container.removeEventListener('wheel', onInteraction); container.removeEventListener('pointerdown', onInteraction); if (timer != null) window.clearTimeout(timer); };
  }, [handleLoadMore]);

  if (tooNarrow) {
    return <div ref={wrapperRef} style={{ padding: 24, textAlign: 'center', color: '#6b7280', border: '1px solid rgba(0,0,0,0.08)', borderRadius: 8, background: 'rgba(0,0,0,0.02)' }}>Chart hidden on narrow screens — switch to a wider viewport to see price data.</div>;
  }

  return (
    <div ref={wrapperRef} style={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      {/* Toolbar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 4, flexShrink: 0 }}>
        <Radio.Group value={timeframe} onChange={e => onTimeframeChange?.(e.target.value)} size="small" optionType="button" buttonStyle="solid">
          {TIMEFRAMES.map(tf => <Radio.Button key={tf.value} value={tf.value}>{tf.label}</Radio.Button>)}
        </Radio.Group>
        <Space size={4}>
          {CHART_TYPES.map(ct => (
            <Tooltip key={ct.key} title={ct.label}>
              <Button size="small" type={chartType === ct.key ? 'primary' : 'default'} icon={ct.icon} onClick={() => applyChartType(ct.key)} />
            </Tooltip>
          ))}
          <Tooltip title={streamActive ? 'Live bar stream active' : error || 'Stream unavailable — historical data only'}>
            <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4, fontSize: 11, marginLeft: 4 }}>
              <span style={{ width: 7, height: 7, borderRadius: '50%', background: streamActive ? '#22c55e' : '#ef5350',
                boxShadow: streamActive ? '0 0 4px #22c55e' : '0 0 4px #ef5350' }} />
              <span style={{ color: streamActive ? '#22c55e' : '#ef5350', fontWeight: 600 }}>
                {streamActive ? 'LIVE' : error ? 'ERROR' : 'STATIC'}
              </span>
            </span>
          </Tooltip>
          <IndicatorPicker style={{ marginLeft: 4 }} />
        </Space>
      </div>

      {/* Active indicator chips */}
      {activeIndicators.length > 0 && (
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 6, alignItems: 'center', flexShrink: 0 }}>
          {activeIndicators.map((ind) => {
            const def = getDef(ind.defId);
            const paramsStr = def?.params?.length
              ? '(' + def.params.map(p => ind.params[p.key] ?? p.default).join(',') + ')' : '';
            return (
              <Tag key={ind.instanceId} color="processing" style={{ margin: 0, cursor: 'pointer' }}>
                <Space size={2}>
                  <span style={{ fontSize: 11 }} onClick={() => setEditingIndId(ind.instanceId)}>
                    {def?.name || ind.defId}{paramsStr}
                  </span>
                  <Button type="text" size="small" icon={<SettingOutlined />}
                    onClick={(e) => { e.stopPropagation(); setEditingIndId(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 12, height: 12, lineHeight: 1, color: 'inherit' }} />
                  <Button type="text" size="small" danger icon={<CloseOutlined />}
                    onClick={(e) => { e.stopPropagation(); removeIndicator(ind.instanceId); }}
                    style={{ padding: 0, minWidth: 12, height: 12, lineHeight: 1 }} /></Space></Tag>);
          })}
        </div>)}

      {/* Indicator settings modal */}
      {editingIndId && (() => {
        const ind = activeIndicators.find(a => a.instanceId === editingIndId);
        const def = ind ? getDef(ind.defId) : undefined;
        return ind && def
          ? <IndicatorSettingsModal visible={true} indicator={ind} def={def} onClose={() => setEditingIndId(null)} />
          : null;
      })()}

      {/* Chart */}
      <div style={{ flex: '1 1 0', minHeight: 0, position: 'relative', background: '#131722', borderRadius: 4, overflow: 'hidden' }}>
        <DrawingToolbar chart={chartRef.current} />
        {loading && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10, background: 'rgba(0,0,0,0.3)' }}><Spin /></div>}
        {error && !loading && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ef5350', zIndex: 10 }}>{error}</div>}
        {!symbol && !loading && !error && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#6b7280', zIndex: 10 }}>Select a symbol to view chart</div>}
        <div ref={containerRef} style={{ width: '100%', height: '100%' }} />
      </div>
    </div>
  );
}
