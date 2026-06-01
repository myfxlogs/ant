import { useEffect, useRef, useCallback, useState } from 'react';
import { Radio, Spin, Tooltip } from 'antd';
import { InfoCircleOutlined } from '@ant-design/icons';
import { init, dispose, type KLineData } from 'klinecharts';
import type { Chart } from 'klinecharts';
import { marketApi, type KlineData as ApiKlineData } from '@/client/market';

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
  accountId?: string;
}

function toKLineData(bar: ApiKlineData): KLineData {
  return {
    timestamp: bar.time * 1000, // seconds → milliseconds
    open: bar.open,
    high: bar.high,
    low: bar.low,
    close: bar.close,
    volume: bar.volume,
  };
}

export default function PriceChart({ symbol, timeframe = '1h', onTimeframeChange, height = 500, accountId }: PriceChartProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<Chart | null>(null);
  const [bars, setBars] = useState<ApiKlineData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [tooNarrow, setTooNarrow] = useState(false);
  const loadingMore = useRef(false);
  const loadedAll = useRef(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // responsive — hide chart below 1280px viewport width
  useEffect(() => {
    const mq = window.matchMedia('(max-width: 1279px)');
    setTooNarrow(mq.matches);
    const handler = (e: MediaQueryListEvent) => setTooNarrow(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  // Fetch klines on symbol/timeframe change + poll
  useEffect(() => {
    if (!symbol) return;
    let cancelled = false;
    setLoading(true);
    setError(null);
    loadedAll.current = false;
    loadingMore.current = false;

    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }

    const canonical = marketApi.resolveSymbol(symbol);
    const doFetch = (count: number) =>
      marketApi.getKlines({ symbol: canonical, timeframe, count, accountId });

    doFetch(INITIAL_BARS)
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

    // Poll every 5s for live data
    pollRef.current = setInterval(() => {
      if (cancelled) return;
      doFetch(5)
        .then((latest) => {
          if (cancelled || latest.length === 0) return;
          setBars((prev) => {
            const index = new Map<number, ApiKlineData>();
            for (const b of prev) index.set(b.time, b);
            for (const b of latest) index.set(b.time, b);
            return [...index.values()].sort((a, b) => a.time - b.time);
          });
        })
        .catch(() => { /* silent */ });
    }, 5000);

    return () => {
      cancelled = true;
      if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null; }
    };
  }, [symbol, timeframe]);

  // Create chart on mount
  useEffect(() => {
    if (!containerRef.current) return;

    // Ensure container has explicit dimensions before init
    const el = containerRef.current;
    el.style.width = '100%';
    el.style.height = `${height}px`;

    const chart = init(el, {
      styles: {
        backgroundColor: '#131722',
        grid: {
          show: true,
          horizontal: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [] },
          vertical: { show: true, color: 'rgba(255,255,255,0.06)', style: 'dashed' as const, size: 1, dashedValue: [] },
        },
        candle: {
          bar: {
            upColor: '#26a69a',
            downColor: '#ef5350',
            noChangeColor: '#888888',
            upBorderColor: '#26a69a',
            downBorderColor: '#ef5350',
            noChangeBorderColor: '#888888',
            upWickColor: '#26a69a',
            downWickColor: '#ef5350',
            noChangeWickColor: '#888888',
          },
        },
        xAxis: {
          axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
          tickText: { color: '#d1d5db' },
        },
        yAxis: {
          axisLine: { show: true, color: 'rgba(255,255,255,0.1)' },
          tickText: { color: '#d1d5db' },
        },
        crosshair: {
          show: true,
          horizontal: { show: true, line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [] } },
          vertical: { show: true, line: { show: true, color: 'rgba(255,255,255,0.3)', style: 'dashed' as const, size: 1, dashedValue: [] } },
        },
      },
    });

    if (!chart) {
      console.error('klinecharts init returned null');
      return;
    }
    chartRef.current = chart;

    return () => {
      dispose(el);
      chartRef.current = null;
    };
  }, []);

  // Update chart data when bars change
  useEffect(() => {
    if (!chartRef.current || bars.length === 0) return;
    chartRef.current.applyNewData(bars.map(toKLineData));
  }, [bars]);

  // Load more bars when user scrolls left
  const handleLoadMore = useCallback((timestamp: number | null) => {
    if (!timestamp || bars.length === 0) return;
    if (loadingMore.current || loadedAll.current) return;
    // klinecharts passes the timestamp of the leftmost visible bar
    const firstBarTime = bars[0].time;
    if (timestamp >= firstBarTime) return;
    loadingMore.current = true;
    marketApi.getKlines({ symbol: canonical, timeframe, count: INITIAL_BARS, before: firstBarTime, accountId })
      .then((older) => {
        if (older.length === 0) { loadedAll.current = true; return; }
        setBars((prev) => [...older, ...prev]);
      })
      .catch(() => { /* silent */ })
      .finally(() => { loadingMore.current = false; });
  }, [bars, symbol, timeframe]);

  // Wire load-more listener
  useEffect(() => {
    const chart = chartRef.current;
    if (!chart) return;
    // klinecharts fires onVisibleRangeChange — but v9 doesn't expose this directly.
    // Instead, listen for mouse/touch interactions and check visible range.
    // For simplicity, use a MutationObserver-free approach: check on wheel/pointer.
    const container = containerRef.current;
    if (!container) return;

    let timer: ReturnType<typeof setTimeout> | null = null;
    const onInteraction = () => {
      if (timer) clearTimeout(timer);
      timer = setTimeout(() => {
        try {
          // Access visible data range through chart internals
          const range = (chart as any).getVisibleRange?.() as { from?: number; to?: number } | null;
          if (range?.from != null) {
            handleLoadMore(range.from);
          }
        } catch { /* ignore */ }
      }, 300);
    };

    container.addEventListener('wheel', onInteraction, { passive: true });
    container.addEventListener('pointerdown', onInteraction, { passive: true });

    return () => {
      container.removeEventListener('wheel', onInteraction);
      container.removeEventListener('pointerdown', onInteraction);
      if (timer) clearTimeout(timer);
    };
  }, [handleLoadMore]);

  // Update chart height
  useEffect(() => {
    const chart = chartRef.current;
    if (chart && containerRef.current) {
      containerRef.current.style.height = `${height}px`;
      chart.resize();
    }
  }, [height]);

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
      {/* Timeframe switcher */}
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
        <div ref={containerRef} style={{ width: '100%', height }} />
      </div>
    </div>
  );
}
