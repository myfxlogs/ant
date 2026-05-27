import { useRef, useCallback } from 'react';

/** Throttle a callback to at most one invocation per `delayMs` milliseconds. */
export function useThrottle<T extends (...args: any[]) => void>(
  fn: T,
  delayMs: number,
): T {
  const lastCall = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  return useCallback(
    (...args: any[]) => {
      const now = Date.now();
      const elapsed = now - lastCall.current;

      if (elapsed >= delayMs) {
        lastCall.current = now;
        fn(...args);
      } else {
        if (timer.current !== null) {
          clearTimeout(timer.current);
        }
        timer.current = setTimeout(() => {
          lastCall.current = Date.now();
          timer.current = null;
          fn(...args);
        }, delayMs - elapsed);
      }
    },
    [fn, delayMs],
  ) as T;
}
