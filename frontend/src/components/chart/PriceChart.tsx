import { useEffect, useRef, useCallback, useState } from 'react';
import { Spin } from 'antd';
import { init, dispose } from 'klinecharts';
import type { Chart } from 'klinecharts';
import DARK_THEME from './chartTheme';
import DrawingToolbar from './DrawingToolbar';
import ChartToolbar from './ChartToolbar';
import type { ChartType } from './ChartToolbar';
import { marketApi } from '@/client/market';
import { useChartData, toChartBar } from './useChartData';
import { useChartIndicatorsStore, KLINECHARTS_MAP } from '@/stores/chartIndicatorsStore';
import IndicatorSettingsModal from './IndicatorSettingsModal';
import './BidAskIndicator';
import './BacktestTradeOverlay';

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
  const { active: activeIndicators, getDef, addIndicator, removeIndicator } = useChartIndicatorsStore();
  const [editingIndId, setEditingIndId] = useState<string | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = init(containerRef.current, { styles: DARK_THEME, locale: 'en-US' });
    chartRef.current = chart;
    onChartReady?.(chart);
    const ro = new ResizeObserver(([entry]) => setTooNarrow((entry?.contentRect?.width || 300) < 300));
    ro.observe(containerRef.current);
    return () => { ro.disconnect(); dispose(containerRef.current!); };
  }, [onChartReady]);

  useEffect(() => {
    if (!chartRef.current || bars.length === 0) return;
    chartRef.current.applyNewData(bars.map(toChartBar));
  }, [bars]);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    try {
      const styles = chart.getStyles();
      if (!styles) return;
      styles.candle = chartType === 'ohlc' ? (styles.candle ? { ...styles.candle, type: 'ohlc' } : { type: 'ohlc' }) :
        chartType === 'area' ? { ...styles.candle, type: 'area' } : { type: 'candle_solid' };
      chart.setStyles(styles);
    } catch { /* chart styles best-effort */ }
  }, [chartType]);

  const applyChartType = useCallback((type: ChartType) => { setChartType(type); }, []);

  useEffect(() => {
    const chart = chartRef.current;
    if (!chart || !trades) return;
    try { (chart as any).setOverlayData?.('backtest_trades', trades); } catch { /* overlay best-effort */ }
  }, [trades]);

  const handleLoadMore = useCallback((oldest: { time: number }) => {
    if (loadingMore.current || loadedAll.current) return;
    loadingMore.current = true;
    marketApi.getKlines({ symbol: marketApi.resolveSymbol(symbol), timeframe, count: 300, before: oldest.time, accountId })
      .then((older) => {
        if (older.length === 0) { loadedAll.current = true; return; }
        chartRef.current?.applyNewData(older.map(toChartBar), true);
      }).catch(() => { /* silent */ }).finally(() => { loadingMore.current = false; });
  }, [symbol, timeframe, accountId, loadingMore, loadedAll]);

  useEffect(() => {
    const chart = chartRef.current, container = containerRef.current;
    if (!chart || !container) return;
    let timer: number | null = null;
    const onInteraction = () => {
      if (timer != null) window.clearTimeout(timer);
      timer = window.setTimeout(() => {
        try { const r = (chart as any).getVisibleRange?.() as { from?: number } | null; if (r?.from != null) handleLoadMore(r.from); } catch { /* best-effort */ }
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
      <ChartToolbar
        timeframe={timeframe} chartType={chartType}
        streamActive={streamActive} error={error}
        activeIndicators={activeIndicators} getDef={getDef}
        onTimeframeChange={onTimeframeChange} applyChartType={applyChartType}
        onSettingsClick={setEditingIndId} onRemoveIndicator={removeIndicator}
      />
      {editingIndId && (() => {
        const ind = activeIndicators.find(a => a.instanceId === editingIndId);
        const def = ind ? getDef(ind.defId) : undefined;
        return ind && def
          ? <IndicatorSettingsModal visible={true} indicator={ind} def={def} onClose={() => setEditingIndId(null)} />
          : null;
      })()}
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
