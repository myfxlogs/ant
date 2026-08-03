type UmamiWindow = Window & {
  umami?: {
    track: (eventName: string, props?: Record<string, string | number | boolean>) => void;
  };
};

export function trackFunnelEvent(event: string, props?: Record<string, string | number | boolean>): void {
  try {
    const w = window as UmamiWindow;
    w.umami?.track(event, props);
  } catch {
    // umami not loaded — silently ignore
  }
}

export const FunnelEvents = {
  REGISTER: 'funnel_register',
  FIRST_GENERATION: 'funnel_first_generation',
  FIRST_BACKTEST: 'funnel_first_backtest',
  FIRST_MT_BIND: 'funnel_first_mt_bind',
  FIRST_LIVE: 'funnel_first_live',
} as const;
