import { WATCHLIST_KEY } from '@/gen/ant/v1/i18n/strategy_workspace_keys';
import { useState, useCallback } from 'react';

const WATCHLIST_KEY = 'ant_watchlist_symbols';

function loadWatchlist(): string[] {
  try {
    const raw = localStorage.getItem(WATCHLIST_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch { return []; }
}

function saveWatchlist(list: string[]) {
  localStorage.setItem(WATCHLIST_KEY, JSON.stringify(list));
}

export function useWatchlist() {
  const [watchlist, setWatchlist] = useState<string[]>(loadWatchlist);

  const isInWatchlist = (symbol: string) => watchlist.includes(symbol);

  const toggleWatchlist = useCallback((symbol: string) => {
    if (!symbol) return;
    setWatchlist((prev) => {
      const next = prev.includes(symbol) ? prev.filter((s) => s !== symbol) : [...prev, symbol];
      saveWatchlist(next);
      return next;
    });
  }, []);

  return { watchlist, isInWatchlist, toggleWatchlist };
}
