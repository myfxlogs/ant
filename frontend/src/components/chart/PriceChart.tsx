import { useEffect, useRef, useCallback, useState } from 'react';
import { Radio, Spin, Tooltip, Button, Space } from 'antd';
import { InfoCircleOutlined, BarChartOutlined, LineChartOutlined, AreaChartOutlined, StockOutlined } from '@ant-design/icons';
import { init, dispose, type KLineData } from 'klinecharts';
import type { Chart } from 'klinecharts';
import DARK_THEME from './chartTheme';
import DrawingToolbar from './DrawingToolbar';
import IndicatorPicker from './IndicatorPicker';
import ActiveIndicatorsBar from './ActiveIndicatorsBar';
import { useChartData } from './useChartData';
import './BidAskIndicator';

const TIMEFRAMES = [
  { label: '1m', value: '1m' }, { label: '5m', value: '5m' },
  { label: '15m', value: '15m' }, { label: '30m', value: '30m' },
  { label: '1h', value: '1h' }, { label: '4h', value: '4h' },
  { label: '1d', value: '1d' }, { label: '1w', value: '1w' },
];

type ChartType = 'candle_solid' | 'ohlc' | 'area' | 'line';

const CHART_TYPES: { key: ChartType; icon: React.ReactNode; label: string }[] = [
  { key: 'candle_solid', icon: <StockOutlined />, label: 'Candle' },
  { key: 'ohlc', icon: <BarChartOutlined />, label: 'OHLC' },
  { key: 'area', icon: <AreaChartOutlined />, label: 'Area' },
  { key: 'line', icon: <LineChartOutlined />, label: 'Line' },
];

function toKLineData(bar: { time: number; open: number; high: number; low: number; close: number; volume: number }): KLineData {
  return { timestamp: bar.time * 1000, open: bar.open, high: bar.high, low: bar.low, close: bar.close, volume: bar.volume };
}

interface Props {
  symbol: string;
  timeframe?: string;
  onTimeframeChange?: (tf: string) => void;
  height?: number;
  accountId?: string;
  onChartReady?: (chart: Chart | null) => void;
}

export default function PriceChart({ symbol, timeframe = '1h', onTimeframeChange, height = 500, accountId, onChartReady }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<Chart | null>(null);
  const [tooNarrow, setTooNarrow] = useState(false);
  const [chartType, setChartType] = useState<ChartType>('candle_solid');

  const { bars, loading, error, loadingMore, loadedAll, loadMore } = useChartData(symbol, timeframe, accountId);

  // Responsive
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)');
    setTooNarrow(mq.matches);
    const handler = (e: MediaQueryListEvent) => setTooNarrow(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // applyNewData when bars change.
  useEffect(() => {
    if (!chartRef.current) return;
    chartRef.current.applyNewData(bars.map(toKLineData));
  }, [bars]);

  // Create chart on mount.
  useEffect(() => {
    if (!containerRef.current) return;
    const el = containerRef.current;
    el.style.width = '100%';
    el.style.height = `${height}px`;

    const chart = init(el, { styles: DARK_THEME });
    if (!chart) { console.error('klinecharts init returned null'); return; }
    chartRef.current = chart;

    try {
      chart.createIndicator('VOL', false, {
        id: 'volume_pane',
        styles: { bars: { upColor: 'rgba(38,166,154,0.6)', downColor: 'rgba(239,83,80,0.6)' }, lines: [] },
      });
    } catch { /* ignore */ }

    try { chart.createIndicator('BIDASK', true, { id: 'candle_pane' }); } catch (e) { console.error('BIDASK createIndicator failed', e); }

    onChartReady?.(chart);
    return () => { onChartReady?.(null); dispose(el); chartRef.current = null; };
  }, []);

  // Chart type.
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    try { (chart as any).setCandleStickChartType?.(chartType); } catch { /* ignore */ }
  }, [chartType]);

  const applyChartType = useCallback((type: ChartType) => {
    setChartType(type);
    const chart = chartRef.current;
    if (!chart) return;
    try { (chart as any).setCandleStickChartType?.(type); } catch {
      chart.setStyleOptions({ candle: { ...DARK_THEME.candle, type: type === 'ohlc' ? 'ohlc' as const : 'candle_solid' as const } });
    }
  }, []);

  // Load more on scroll-left.
  const handleLoadMore = useCallback((timestamp: number | null) => {
    if (!timestamp || bars.length === 0 || loadingMore.current || loadedAll.current) return;
    if (timestamp >= bars[0].time) return;
    loadingMore.current = true;
    loadMore(bars[0].time).finally(() => { loadingMore.current = false; });
  }, [bars, loadMore, loadingMore, loadedAll]);

  // Listen for scroll/pan to trigger load-more.
  useEffect(() => {
    const chart = chartRef.current;
    const container = containerRef.current;
    if (!chart || !container) return;
    let timer: ReturnType<typeof setTimeout> | null = null;
    const onInteraction = () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        try { const r = (chart as any).getVisibleRange?.() as { from?: number } | null; if (r?.from != null) handleLoadMore(r.from); } catch { /* */ }
      }, 300);
    };
    container.addEventListener('wheel', onInteraction, { passive: true });
    container.addEventListener('pointerdown', onInteraction, { passive: true });
    return () => { container.removeEventListener('wheel', onInteraction); container.removeEventListener('pointerdown', onInteraction); if (timer) clearTimeout(timer); };
  }, [handleLoadMore]);

  // Resize.
  useEffect(() => {
    if (chartRef.current && containerRef.current) { containerRef.current.style.height = `${height}px`; chartRef.current.resize(); }
  }, [height]);

  if (tooNarrow) {
    return <div ref={wrapperRef} style={{ padding: 24, textAlign: 'center', color: '#6b7280', border: '1px solid rgba(0,0,0,0.08)', borderRadius: 8, background: 'rgba(0,0,0,0.02)' }}>Chart hidden on narrow screens — switch to a wider viewport to see price data.</div>;
  }

  return (
    <div ref={wrapperRef} style={{ position: 'relative' }}>
      {/* Toolbar */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 8, flexWrap: 'wrap', gap: 4 }}>
        <Space size="small" wrap>
          <Radio.Group value={timeframe} onChange={e => onTimeframeChange?.(e.target.value)} size="small" optionType="button" buttonStyle="solid">
            {TIMEFRAMES.map(tf => <Radio.Button key={tf.value} value={tf.value}>{tf.label}</Radio.Button>)}
          </Radio.Group>
        </Space>
        <Space size={4}>
          {CHART_TYPES.map(ct => (
            <Tooltip key={ct.key} title={ct.label}>
              <Button size="small" type={chartType === ct.key ? 'primary' : 'default'} icon={ct.icon} onClick={() => applyChartType(ct.key)} />
            </Tooltip>
          ))}
          <Tooltip title="Mid-price OHLC candles + BIDASK indicator">
            <span style={{ color: '#6b7280', fontSize: 12, cursor: 'help', userSelect: 'none', marginLeft: 4 }}><InfoCircleOutlined style={{ marginRight: 4 }} />real-time</span>
          </Tooltip>
          <IndicatorPicker style={{ marginLeft: 4 }} />
        </Space>
      </div>

      <ActiveIndicatorsBar style={{ marginBottom: 6 }} />

      {/* Chart */}
      <div style={{ position: 'relative', minHeight: height, background: '#131722', borderRadius: 4, overflow: 'hidden' }}>
        <DrawingToolbar chart={chartRef.current} />
        {loading && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10, background: 'rgba(0,0,0,0.3)' }}><Spin /></div>}
        {error && !loading && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#ef5350', zIndex: 10 }}>{error}</div>}
        {!symbol && !loading && !error && <div style={{ position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#6b7280', zIndex: 10 }}>Select a symbol to view chart</div>}
        <div ref={containerRef} style={{ width: '100%', height }} />
      </div>
    </div>
  );
}
