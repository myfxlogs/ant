// useServerIndicators — subscribes to server-side indicator computation via ConnectRPC streaming.
//
// When active, replaces klinecharts built-in indicator computation with server-computed values.
// The hook manages the stream lifecycle (connect, parse, reconnect, cleanup) and populates
// the shared serverIndicatorStore consumed by custom ANT_* klinecharts indicators.
//
// Usage:
//   useServerIndicators(symbol, timeframe, activeIndicators, chartRef, setStreamStatus)

import { useEffect, useRef, useCallback } from 'react';
import type { Chart } from 'klinecharts';
import type { ActiveIndicator } from '@/stores/chartIndicatorsStore';
import { streamClient } from '@/client/connect';
import type { IndicatorUpdateEvent } from '@/gen/ant/v1/stream_event_indicator_pb';
import { isLikelyStreamTransportFailure } from '@/utils/streamErrors';
import {
  setServerIndicatorData,
  clearServerIndicatorData,
  type ServerIndicatorData,
} from '@/components/chart/serverIndicators';

const TRANSPORT_FAILURE_CAP = 8;

interface UseServerIndicatorsOptions {
  symbol: string;
  timeframe: string;
  activeIndicators: ActiveIndicator[];
  chartRef: React.MutableRefObject<Chart | null>;
  onStreamStatus?: (streaming: boolean) => void;
}

export function useServerIndicators({
  symbol,
  timeframe,
  activeIndicators,
  chartRef,
  onStreamStatus,
}: UseServerIndicatorsOptions) {
  const abortedRef = useRef(false);
  const streamingRef = useRef(false);

  // Reapply chart data to trigger indicator recalc with fresh server values.
  const refreshChart = useCallback(() => {
    const chart = chartRef.current;
    if (!chart) return;
    try {
      const data = chart.getDataList?.();
      if (data?.length) chart.applyNewData([...data]);
    } catch { /* best-effort */ }
  }, [chartRef]);

  useEffect(() => {
    const ids = activeIndicators.map((a) => a.defId);
    if (ids.length === 0 || !symbol || !timeframe) {
      // Clear server data when no indicators active.
      clearServerIndicatorData();
      if (streamingRef.current) {
        onStreamStatus?.(false);
        streamingRef.current = false;
      }
      return;
    }

    abortedRef.current = false;
    let transportFailStreak = 0;
    const k = `${symbol}/${timeframe}`; // key for cleanup cache

    const runStream = async (retryCount = 0) => {
      if (abortedRef.current) return;

      try {
        const params: Record<string, string> = {};
        for (const ind of activeIndicators) {
          for (const [key, val] of Object.entries(ind.params)) {
            params[`${ind.defId}.${key}`] = String(val);
          }
        }

        const stream = streamClient.subscribeIndicators({
          symbol,
          timeframe,
          indicatorIds: ids,
          params,
        });

        if (!streamingRef.current) {
          streamingRef.current = true;
          onStreamStatus?.(true);
        }

        let received = 0;
        for await (const event of stream) {
          if (abortedRef.current) break;
          transportFailStreak = 0;

          const e = event as IndicatorUpdateEvent;
          received++;

          const data: ServerIndicatorData = {
            values: e.values || [],
            series: {},
            pane: e.pane || 'sub',
          };

          if (e.series) {
            for (const [name, s] of Object.entries(e.series)) {
              if (s) data.series[name] = s.values || [];
            }
          }

          setServerIndicatorData(e.indicatorId, data);
        }

        // All indicators received — refresh chart.
        if (received > 0) refreshChart();

        if (streamingRef.current) {
          streamingRef.current = false;
          onStreamStatus?.(false);
        }
      } catch (error) {
        if (abortedRef.current) return;
        if ((error as Error).name === 'AbortError') return;
        if (String(error).includes('canceled') || String(error).includes('aborted')) return;

        if (isLikelyStreamTransportFailure(error)) {
          transportFailStreak++;
          if (transportFailStreak >= TRANSPORT_FAILURE_CAP) {
            console.warn('[useServerIndicators] transport failure cap reached');
            return;
          }
        }

        const delay = Math.min(1000 * Math.pow(2, retryCount), 15000);
        setTimeout(() => runStream(retryCount + 1), delay);
      }
    };

    runStream();

    return () => {
      abortedRef.current = true;
      // Clear data for indicators being removed.
      if (activeIndicators.length === 0) {
        clearServerIndicatorData();
      } else {
        // Only clear indicators that are active (others get cleaned on unmount).
        for (const id of ids) {
          clearServerIndicatorData(id);
        }
      }
      if (streamingRef.current) {
        streamingRef.current = false;
        onStreamStatus?.(false);
      }
    };
  }, [symbol, timeframe, activeIndicators.map((a) => `${a.defId}:${JSON.stringify(a.params)}`).join(','), refreshChart, onStreamStatus]);

  return { refreshChart };
}
