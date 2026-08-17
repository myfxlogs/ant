/**
 * StreamWatchdog: detects stale SSE connections (half-open TCP, zombie fetch
 * readers that neither yield nor reject) and forces reconnection.
 *
 * The watchdog monitors `lastEventAt` — updated via `touch()` on every received
 * event (including heartbeats/pings). If no event arrives within
 * `staleThresholdMs`, `onStale()` fires once.
 *
 * Global hooks (`online`, `visibilitychange`) trigger immediate staleness
 * checks on all active watchdogs, so recovery from sleep/wake or network
 * switch is seconds-fast instead of waiting for the next check interval.
 */

// --- Global recheck registry -------------------------------------------------

/** All active watchdogs, for global event-driven recheck. */
const activeWatchdogs = new Set<StreamWatchdogInternal>();

let globalHooksInstalled = false;

function ensureGlobalHooks(): void {
  if (globalHooksInstalled || typeof window === 'undefined') return;
  globalHooksInstalled = true;

  // Network restored → immediate check (seconds-fast recovery).
  window.addEventListener('online', () => {
    for (const w of activeWatchdogs) w.checkNow();
  });

  // Tab back to foreground → check (background timers are throttled/frozen).
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      for (const w of activeWatchdogs) w.checkNow();
    }
  });
}

// --- Watchdog implementation -------------------------------------------------

interface StreamWatchdogInternal {
  checkNow(): void;
}

export interface StreamWatchdog {
  touch(): void;
  start(): void;
  stop(): void;
  checkNow(): void;
}

export function createStreamWatchdog(opts: {
  staleThresholdMs: number;
  onStale: () => void;
  checkIntervalMs?: number;
}): StreamWatchdog {
  const checkIntervalMs = opts.checkIntervalMs ?? 5000;
  let lastEventAt = 0;
  let running = false;
  let staleTriggered = false;
  let intervalId: ReturnType<typeof setInterval> | null = null;

  const check = () => {
    if (!running || staleTriggered) return;
    if (Date.now() - lastEventAt >= opts.staleThresholdMs) {
      staleTriggered = true;
      console.warn('[stream] stale detected, forcing reconnect');
      opts.onStale();
    }
  };

  const watchdog: StreamWatchdog = {
    touch(): void {
      lastEventAt = Date.now();
      staleTriggered = false;
    },

    start(): void {
      if (running) return;
      running = true;
      staleTriggered = false;
      // Reset lastEventAt so a freshly-built connection gets a grace period
      // equal to the threshold (don't kill before first event arrives).
      lastEventAt = Date.now();
      intervalId = setInterval(check, checkIntervalMs);
      activeWatchdogs.add(watchdog);
      ensureGlobalHooks();
    },

    stop(): void {
      running = false;
      staleTriggered = false;
      if (intervalId !== null) {
        clearInterval(intervalId);
        intervalId = null;
      }
      activeWatchdogs.delete(watchdog);
    },

    checkNow(): void {
      check();
    },
  };

  return watchdog;
}
