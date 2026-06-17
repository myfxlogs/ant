import { useEffect, useMemo, useRef, useState } from 'react';
import { pythonStrategyApi } from '@/client/pythonStrategy';
import type { BacktestRunUpdate } from '@/gen/ant/v1/backtest_run_query_pb';
import { isTerminalRun } from '@/pages/strategy/StrategyTemplatePage.utils';

export type WatchBacktestState = {
	run: any | null;
	metrics: any | null;
	equityCurve: number[];
	loading: boolean;
	error: string | null;
	isTerminal: boolean;
};

export function useWatchBacktestRun(runId?: string | null): WatchBacktestState {
	const [run, setRun] = useState<any | null>(null);
	const [metrics, setMetrics] = useState<any | null>(null);
	const [equityCurve, setEquityCurve] = useState<number[]>([]);
	const [error, setError] = useState<string | null>(null);
	const stoppedRef = useRef(false);
	useEffect(() => {
		stoppedRef.current = false;
		if (!runId) {
			queueMicrotask(() => {
				if (stoppedRef.current) return;
				setRun(null);
				setMetrics(null);
				setEquityCurve([]);
				setError(null);
			});
			return;
		}

		queueMicrotask(() => {
			if (stoppedRef.current) return;
			setError(null);
		});

		let unsubscribe: (() => void) | null = null;

		(async () => {
			try {
				// First fetch current snapshot (fast first paint).
				const snapshot: any = await pythonStrategyApi.getBacktestRun(runId);
				if (stoppedRef.current) return;
				setRun(snapshot?.run ?? null);
				setMetrics(snapshot?.metrics ?? null);
				setEquityCurve(snapshot?.equityCurve ?? []);

				// Terminal runs already have all data in the snapshot.
				if (isTerminalRun(snapshot?.run)) {
					stoppedRef.current = true;
					return;
				}

				// Push-first: use SSE stream for live updates. No fallback polling.
				unsubscribe = pythonStrategyApi.watchBacktestRun(
					runId,
					(u: BacktestRunUpdate) => {
						if (stoppedRef.current) return;
						setRun(u?.run ?? null);
						setMetrics(u?.metrics ?? null);
						setEquityCurve(u?.equityCurve ?? []);
						if (isTerminalRun(u?.run)) {
							stoppedRef.current = true;
							unsubscribe?.();
							unsubscribe = null;
						}
					},
					(_e: unknown) => {
						// Stream failed — stop. Initial snapshot already has data.
						if (!stoppedRef.current) stoppedRef.current = true;
					},
				);
			} catch (e) {
				if (stoppedRef.current) return;
				setError(String(e));
			}
		})();

		return () => {
			stoppedRef.current = true;
			unsubscribe?.();
		};
	}, [runId]);

	const isTerminal = useMemo(() => isTerminalRun(run), [run]);

	const loading = useMemo(() => {
		if (!runId) return false;
		// Only show loading when we have neither data nor error.
		if (error && run == null) return false;
		return run == null;
	}, [runId, run, error]);

	return {
		run,
		metrics,
		equityCurve,
		loading,
		error,
		isTerminal,
	};
}
