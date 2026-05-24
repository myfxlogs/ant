import { useEffect, useMemo, useRef, useState } from 'react';
import { pythonStrategyApi } from '@/client/pythonStrategy';

export type WatchBacktestState = {
	run: any | null;
	metrics: any | null;
	equityCurve: number[];
	loading: boolean;
	error: string | null;
	isTerminal: boolean;
};

function isTerminalRun(run: any): boolean {
	return Boolean(run?.isTerminal || run?.is_terminal);
}

export function useWatchBacktestRun(runId?: string | null): WatchBacktestState {
	const [run, setRun] = useState<any | null>(null);
	const [metrics, setMetrics] = useState<any | null>(null);
	const [equityCurve, setEquityCurve] = useState<number[]>([]);
	const [error, setError] = useState<string | null>(null);
	const stoppedRef = useRef(false);
	const pollTimerRef = useRef<number | null>(null);

	useEffect(() => {
		stoppedRef.current = false;
		if (!runId) {
			if (pollTimerRef.current) {
				window.clearInterval(pollTimerRef.current);
				pollTimerRef.current = null;
			}
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
		const stopPolling = () => {
			if (pollTimerRef.current) {
				window.clearInterval(pollTimerRef.current);
				pollTimerRef.current = null;
			}
		};
		const startPolling = () => {
			if (pollTimerRef.current) return;
			pollTimerRef.current = window.setInterval(async () => {
				try {
					const snapshot: any = await pythonStrategyApi.getBacktestRun(runId);
					if (stoppedRef.current) return;
					setRun(snapshot?.run ?? null);
					setMetrics(snapshot?.metrics ?? null);
					setEquityCurve(snapshot?.equityCurve ?? []);
					if (isTerminalRun(snapshot?.run)) {
						stoppedRef.current = true;
						unsubscribe?.();
						unsubscribe = null;
						stopPolling();
					}
				} catch {
					// ignore
				}
			}, 2000);
		};

		(async () => {
			try {
				// First fetch current snapshot (fast first paint + stream fallback).
				const snapshot: any = await pythonStrategyApi.getBacktestRun(runId);
				if (stoppedRef.current) return;
				setRun(snapshot?.run ?? null);
				setMetrics(snapshot?.metrics ?? null);
				setEquityCurve(snapshot?.equityCurve ?? []);

				unsubscribe = pythonStrategyApi.watchBacktestRun(
					runId,
					(u: any) => {
						if (stoppedRef.current) return;
						setRun(u?.run ?? null);
						setMetrics(u?.metrics ?? null);
						setEquityCurve(u?.equityCurve ?? []);
						if (isTerminalRun(u?.run)) {
							stoppedRef.current = true;
							unsubscribe?.();
							unsubscribe = null;
							stopPolling();
						}
					},
					(e: any) => {
						if (stoppedRef.current) return;
						setError(String(e));
						startPolling();
					},
				);
			} catch (e) {
				if (stoppedRef.current) return;
				setError(String(e));
				startPolling();
			}
		})();

		return () => {
			stoppedRef.current = true;
			unsubscribe?.();
			stopPolling();
		};
	}, [runId]);

	const isTerminal = useMemo(() => isTerminalRun(run), [run]);

	const loading = useMemo(() => {
		if (!runId) return false;
		if (error) return false;
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
