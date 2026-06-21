import { useEffect, useRef, useCallback, useState } from 'react';
import { Spin } from 'antd';
import { useTranslation } from 'react-i18next';
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
import { useServerIndicators } from '@/hooks/useServerIndicators';
import './BidAskIndicator';
import './BacktestTradeOverlay';
import './serverIndicators'; // registers custom klinecharts indicators

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
  const { t } = useTranslation();
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<Chart | null>(null);
  const [chartType, setChartType] = useState<ChartType>('candle_solid');
  const volumeCollapsedRef = useRef(false);

  const { bars, loading, error, streamActive, loadingMore, loadedAll } = useChartData(symbol, timeframe, accountId, chartRef);
  const { active: activeIndicators, getDef, addIndicator, removeIndicator } = useChartIndicatorsStore();
  const [editingIndId, setEditingIndId] = useState<string | null>(null);
  // Track klinecharts paneIds keyed by store instanceId
  const kcIndRef = useRef<Map<string, string>>(new Map());

  // Server-side indicator computation: subscribes to SubscribeIndicators RPC,
  // populates shared store consumed by custom ANT_* klinecharts indicators.
  const [serverStreaming, setServerStreaming] = useState(false);
  useServerIndicators({
    symbol, timeframe, activeIndicators, chartRef,
    onStreamStatus: setServerStreaming,
  });

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = init(containerRef.current, { styles: DARK_THEME, locale: 'en-US' });
    chartRef.current = chart;
    // Auto-load Volume sub-pane
    try { chart.createIndicator('VOL', false, { id: 'volume_pane' }); } catch { /* best-effort */ }
    // B/A price lines on main chart — isStack=true + id:'candle_pane' forces candle pane placement
    try { chart.createIndicator('BIDASK', true, { id: 'candle_pane' }); } catch { /* best-effort */ }
    onChartReady?.(chart);
    const ro = new ResizeObserver(([entry]) => {
      const w = entry?.contentRect?.width || 400;
      chart.resize();
      // Auto-collapse Volume sub-pane on narrow screens to give K-line more space.
      try {
        if (w < 400 && !volumeCollapsedRef.current) {
          chart.removeIndicator('volume_pane');
          volumeCollapsedRef.current = true;
        } else if (w >= 400 && volumeCollapsedRef.current) {
          chart.createIndicator('VOL', false, { id: 'volume_pane' });
          volumeCollapsedRef.current = false;
        }
      } catch { /* best-effort */ }
    });
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
      // Use DeepPartial to avoid destroying nested candle sub-objects
      const type = chartType === 'ohlc' ? 'candle_stroke' as const :
        chartType === 'area' ? 'area' as const : 'candle_solid' as const;
      chart.setStyles({ candle: { type } } as any);
    } catch { /* chart styles best-effort */ }
  }, [chartType]);

  const applyChartType = useCallback((type: ChartType) => { setChartType(type); }, []);

  // ── Sync indicators from store to klinecharts chart ──
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    const currentIds = new Set(activeIndicators.map((a) => a.instanceId));

    // Remove indicators that are no longer in store
    for (const [instanceId, paneId] of kcIndRef.current.entries()) {
      if (!currentIds.has(instanceId)) {
        try { chart.removeIndicator(paneId); } catch { /* best-effort */ }
        kcIndRef.current.delete(instanceId);
      }
    }

    // Add or update indicators
    for (const ind of activeIndicators) {
      const km = KLINECHARTS_MAP[ind.defId];
      if (!km) continue;

      const def = getDef(ind.defId);
      const isOverlay = def?.kind === 'overlay';
      const calcParams = km.buildParams(ind.params);
      const existingPaneId = kcIndRef.current.get(ind.instanceId);

      if (existingPaneId) {
        // Update existing — override params
        try {
          chart.overrideIndicator(
            { name: km.name, calcParams, visible: ind.visible } as any,
            existingPaneId,
          );
        } catch { /* best-effort */ }
      } else {
        // Create new — use string name + explicit pane id (original pattern)
        try {
          const paneId = chart.createIndicator(
            km.name,
            isOverlay,
            { id: `ind_${ind.instanceId}` },
          );
          if (paneId) {
            kcIndRef.current.set(ind.instanceId, paneId);
            // Set calcParams after creation if non-default
            if (calcParams.length > 0) {
              try {
                chart.overrideIndicator({ name: km.name, calcParams } as any, paneId);
              } catch { /* best-effort */ }
            }
          }
        } catch { /* best-effort */ }
      }
    }
  }, [activeIndicators, getDef]);

  // ── Backtest trade markers overlay ──
  const btOverlayIdRef = useRef<string | null>(null);
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;

    if (!trades || trades.length === 0) {
      // Remove overlay when no trades.
      if (btOverlayIdRef.current) {
        try { chart.removeOverlay(btOverlayIdRef.current); } catch { /* best-effort */ }
        btOverlayIdRef.current = null;
      }
      return;
    }

    if (btOverlayIdRef.current) {
      // Update existing overlay data in-place.
      try {
        chart.overrideOverlay({
          name: 'backtest_trades',
          id: btOverlayIdRef.current,
          extendData: trades,
        } as any);
      } catch { /* best-effort */ }
    } else {
      // Create new overlay.
      try {
        const id = chart.createOverlay({
          name: 'backtest_trades',
          extendData: trades,
        });
        if (id) btOverlayIdRef.current = String(id);
      } catch { /* best-effort */ }
    }
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
        try { const r = chart.getVisibleRange?.() as { from?: number } | null; if (r?.from != null) handleLoadMore(r.from); } catch { /* best-effort */ }
      }, 300);
    };
    container.addEventListener('wheel', onInteraction, { passive: true });
    container.addEventListener('pointerdown', onInteraction, { passive: true });
    return () => { container.removeEventListener('wheel', onInteraction); container.removeEventListener('pointerdown', onInteraction); if (timer != null) window.clearTimeout(timer); };
  }, [handleLoadMore]);

  return (
    <div ref={wrapperRef} style={{ display: 'flex', flexDirection: 'column', flex: 1, minHeight: 0 }}>
      <ChartToolbar
        timeframe={timeframe} chartType={chartType}
        streamActive={streamActive || serverStreaming} error={error}
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
        {!symbol && !loading && !error && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#6b7280', zIndex: 10 }}>{t('common.selectSymbolToViewChart')}</div>}
        <div ref={containerRef} style={{ position: 'absolute', inset: 0 }} />
      </div>
    </div>
  );
}
