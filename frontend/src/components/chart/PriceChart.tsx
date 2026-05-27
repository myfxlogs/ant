import { useEffect, useRef, useCallback, useState } from 'react';
import { Radio, Spin, Tooltip } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import { createChart, type IChartApi, type ISeriesApi, type CandlestickData, type HistogramData, type Time, ColorType } from 'lightweight-charts';
import { marketApi, type KlineData } from '@/client/market';

const TIMEFRAMES = [
  { label: '1m', value: '1m' },
  { label: '5m', value: '5m' },
  { label: '15m', value: '15m' },
  { label: '30m', value: '30m' },
  { label: '1h', value: '1h' },
  { label: '4h', value: '4h' },
  { label: '1d', value: '1d' },
  { label: '1w', value: '1w' },
];

const INITIAL_BARS = 300;

interface PriceChartProps {
  symbol: string;
  timeframe?: string;
  onTimeframeChange?: (tf: string) => void;
  height?: number;
}

function toCandleData(bar: KlineData): CandlestickData {
  return {
    time: bar.time as Time,
    open: bar.open,
    high: bar.high,
    low: bar.low,
    close: bar.close,
  };
}

function toVolumeData(bar: KlineData): HistogramData {
  return {
    time: bar.time as Time,
    value: bar.volume,
    color: bar.close >= bar.open ? 'rgba(38,166,154,0.5)' : 'rgba(239,83,80,0.5)',
  };
}

export default function PriceChart({ symbol, timeframe = '1h', onTimeframeChange, height = 500 }: PriceChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<IChartApi | null>(null);
  const candleSeriesRef = useRef<ISeriesApi<'Candlestick'> | null>(null);
  const volumeSeriesRef = useRef<ISeriesApi<'Histogram'> | null>(null);
  const [bars, setBars] = useState<KlineData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tooNarrow, setTooNarrow] = useState(false);
  const loadingMore = useRef(false);
  const loadedAll = useRef(false);
  const handleVisibleRangeChangeRef = useRef<((range: { from: number } | null) => void) | null>(null);

  // B-4.5: responsive — hide chart below 1280px viewport width.
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 1279px)');
    setTooNarrow(mq.matches);
    const handler = (e: MediaQueryListEvent) => setTooNarrow(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // Fetch klines on symbol/timeframe change.
  useEffect(() => {
    if (!symbol) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    loadedAll.current = false;
    loadingMore.current = false;

    marketApi.getKlines({ symbol, timeframe, count: INITIAL_BARS })
      .then((data) => {
        if (cancelled) return;
        setBars(data);
        setLoading(false);
      })
      .catch((err) => {
        if (cancelled) return;
        setError(err.message || 'Failed to load chart data');
        setLoading(false);
      });

    return () => { cancelled = true; };
  }, [symbol, timeframe]);

  // Create chart on mount.
  useEffect(() => {
    if (!containerRef.current) return;

    const chart = createChart(containerRef.current, {
      layout: {
        background: { type: ColorType.Solid, color: 'transparent' },
        textColor: '#d1d5db',
      },
      grid: {
        vertLines: { color: 'rgba(255,255,255,0.06)' },
        horzLines: { color: 'rgba(255,255,255,0.06)' },
      },
      crosshair: { mode: 0 },
      rightPriceScale: {
        borderColor: 'rgba(255,255,255,0.1)',
      },
      timeScale: {
        borderColor: 'rgba(255,255,255,0.1)',
        timeVisible: true,
        secondsVisible: false,
      },
      width: containerRef.current.clientWidth,
      height,
    });

    const candleSeries = chart.addCandlestickSeries({
      upColor: '#26a69a',
      downColor: '#ef5350',
      borderUpColor: '#26a69a',
      borderDownColor: '#ef5350',
      wickUpColor: '#26a69a',
      wickDownColor: '#ef5350',
    });

    const volumeSeries = chart.addHistogramSeries({
      priceFormat: { type: 'volume' },
      priceScaleId: '',
    });
    volumeSeries.priceScale().applyOptions({
      scaleMargins: { top: 0.8, bottom: 0 },
    });

    chartRef.current = chart;
    candleSeriesRef.current = candleSeries;
    volumeSeriesRef.current = volumeSeries;

    const handleResize = () => {
      if (containerRef.current && chartRef.current) {
        chartRef.current.applyOptions({ width: containerRef.current.clientWidth });
      }
    };
    const observer = new ResizeObserver(handleResize);
    observer.observe(containerRef.current);

    const rangeHandler = (range: { from: number } | null) => {
      handleVisibleRangeChangeRef.current?.(range);
    };
    chart.timeScale().subscribeVisibleLogicalRangeChange(rangeHandler);

    return () => {
      chart.timeScale().unsubscribeVisibleLogicalRangeChange(rangeHandler);
      observer.disconnect();
      chart.remove();
      chartRef.current = null;
      candleSeriesRef.current = null;
      volumeSeriesRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Update chart data when bars change.
  useEffect(() => {
    if (!candleSeriesRef.current || !volumeSeriesRef.current || bars.length === 0) return;
    candleSeriesRef.current.setData(bars.map(toCandleData));
    volumeSeriesRef.current.setData(bars.map(toVolumeData));
  }, [bars]);

  // Load more bars when user scrolls/drags to the left edge.
  const handleVisibleRangeChange = useCallback((range: { from: number } | null) => {
    if (!range || bars.length === 0) return;
    if (loadingMore.current || loadedAll.current) return;
    const firstBarTime = bars[0].time;
    if (range.from >= firstBarTime - 60) return;
    loadingMore.current = true;
    marketApi.getKlines({ symbol, timeframe, count: INITIAL_BARS, before: firstBarTime })
      .then((older) => {
        if (older.length === 0) { loadedAll.current = true; return; }
        setBars((prev) => [...older, ...prev]);
      })
      .catch(() => { /* silent */ })
      .finally(() => { loadingMore.current = false; });
  }, [bars, symbol, timeframe]);
  handleVisibleRangeChangeRef.current = handleVisibleRangeChange;

  // Update chart height when prop changes.
  useEffect(() => {
    if (chartRef.current) {
      chartRef.current.applyOptions({ height });
    }
  }, [height]);

  // B-4.5: hide chart on narrow viewports (< 1280px) to prevent layout breakage.
  if (tooNarrow) {
    return (
      <div ref={wrapperRef} style={{
        padding: 24, textAlign: 'center', color: '#6b7280',
        border: '1px solid rgba(0,0,0,0.08)', borderRadius: 8,
        background: 'rgba(0,0,0,0.02)',
      }}>
        Chart hidden on narrow screens — switch to a wider viewport to see price data.
      </div>
    );
  }

  return (
    <div ref={wrapperRef} style={{ position: 'relative' }}>
      {/* Timeframe switcher + delay tooltip */}
      <div style={{
        display: 'flex', justifyContent: 'space-between', alignItems: 'center',
        marginBottom: 8,
      }}>
        <Radio.Group
          value={timeframe}
          onChange={(e) => onTimeframeChange?.(e.target.value)}
          size="small"
          optionType="button"
          buttonStyle="solid"
        >
          {TIMEFRAMES.map((tf) => (
            <Radio.Button key={tf.value} value={tf.value}>{tf.label}</Radio.Button>
          ))}
        </Radio.Group>
        <Tooltip title="OHLC data — approximately 5s delay. No real-time Bid/Ask spread.">
          <span style={{ color: '#6b7280', fontSize: 12, cursor: 'help', userSelect: 'none' }}>
            <InfoCircleOutlined style={{ marginRight: 4 }} />
            ~5s delay
          </span>
        </Tooltip>
      </div>

      {/* Chart container */}
      <div style={{ position: 'relative', minHeight: height }}>
        {loading && (
          <div style={{
            position: 'absolute', inset: 0, display: 'flex',
            alignItems: 'center', justifyContent: 'center',
            zIndex: 10, background: 'rgba(0,0,0,0.3)',
          }}>
            <Spin />
          </div>
        )}
        {error && !loading && (
          <div style={{
            position: 'absolute', inset: 0, display: 'flex',
            alignItems: 'center', justifyContent: 'center',
            color: '#ef5350', zIndex: 10,
          }}>
            {error}
          </div>
        )}
        {!symbol && !loading && !error && (
          <div style={{
            position: 'absolute', inset: 0, display: 'flex',
            alignItems: 'center', justifyContent: 'center',
            color: '#6b7280', zIndex: 10,
          }}>
            Select a symbol to view chart
          </div>
        )}
        <div ref={containerRef} style={{ width: '100%' }} />
      </div>
    </div>
  );
}
