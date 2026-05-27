import { useEffect, useState, useMemo, useCallback } from 'react';
import { Select } from 'antd';
import type { SelectProps } from 'antd';
import { StarFilled } from '@ant-design/icons';
import { marketApi, type SymbolInfo } from '@/client/market';

const WATCHLIST_KEY = 'ant_watchlist_symbols';

function loadWatchlist(): string[] {
  try {
    const raw = localStorage.getItem(WATCHLIST_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

interface SymbolPickerProps {
  value?: string;
  onChange?: (symbol: string) => void;
  accountId: string;
  placeholder?: string;
  style?: React.CSSProperties;
}

export default function SymbolPicker({ value, onChange, accountId, placeholder = 'Select symbol', style }: SymbolPickerProps) {
  const [symbols, setSymbols] = useState<SymbolInfo[]>([]);
  const [watchlist, setWatchlist] = useState<string[]>(loadWatchlist);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!accountId) return;
    let cancelled = false;
    setLoading(true);
    marketApi.getSymbols(accountId)
      .then((list) => {
        if (cancelled) return;
        setSymbols(list);
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setLoading(false);
      });

    return () => { cancelled = true; };
  }, [accountId]);

  const toggleWatchlist = useCallback((sym: string) => {
    setWatchlist((prev) => {
      const next = prev.includes(sym)
        ? prev.filter((s) => s !== sym)
        : [...prev, sym];
      localStorage.setItem(WATCHLIST_KEY, JSON.stringify(next));
      return next;
    });
  }, []);

  const options: SelectProps['options'] = useMemo(() => {
    const watchlistSymbols = symbols.filter((s) => watchlist.includes(s.symbol));
    const otherSymbols = symbols.filter((s) => !watchlist.includes(s.symbol));

    const groups: SelectProps['options'] = [];

    if (watchlistSymbols.length > 0) {
      groups.push({
        label: 'Watchlist',
        options: watchlistSymbols.map((s) => ({
          value: s.symbol,
          label: (
            <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span>
                <StarFilled style={{ color: '#D4AF37', marginRight: 6, fontSize: 12 }} />
                <span style={{ fontWeight: 600 }}>{s.symbol}</span>
              </span>
              {s.description && (
                <span style={{ color: '#6b7280', fontSize: 12 }}>{s.description}</span>
              )}
            </span>
          ),
        })),
      });
    }

    if (otherSymbols.length > 0) {
      groups.push({
        label: 'All Symbols',
        options: otherSymbols.map((s) => ({
          value: s.symbol,
          label: (
            <span style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
              <span style={{ fontWeight: 500 }}>{s.symbol}</span>
              {s.description && (
                <span style={{ color: '#6b7280', fontSize: 12 }}>{s.description}</span>
              )}
            </span>
          ),
        })),
      });
    }

    return groups;
  }, [symbols, watchlist]);

  return (
    <Select
      showSearch
      value={value || undefined}
      onChange={(v) => onChange?.(v)}
      loading={loading}
      placeholder={placeholder}
      style={{ minWidth: 120, ...style }}
      filterOption={(input, option) => {
        if (!option?.value) return false;
        return String(option.value).toLowerCase().includes(input.toLowerCase());
      }}
      options={options}
      notFoundContent={loading ? 'Loading...' : 'No symbols found'}
    />
  );
}
