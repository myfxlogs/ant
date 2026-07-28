import { useEffect, useState, useMemo, useCallback } from 'react';
import { Select } from 'antd';
import type { SelectProps } from 'antd';
import { StarFilled } from '@ant-design/icons';
import { marketApi, isMTSessionError, type SymbolInfo } from '@/client/market';
import { useTranslation } from 'react-i18next';

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
  onDropdownVisibleChange?: (open: boolean) => void;
  accountId: string;
  placeholder?: string;
  style?: React.CSSProperties;
}

export default function SymbolPicker({ value, onChange, onDropdownVisibleChange, accountId, placeholder, style }: SymbolPickerProps) {
  const { t } = useTranslation();
  const [symbols, setSymbols] = useState<SymbolInfo[]>([]);
  const [watchlist, setWatchlist] = useState<string[]>(loadWatchlist);
  const [loading, setLoading] = useState(false);
  const [mtError, setMtError] = useState(false);

  useEffect(() => {
    if (!accountId) { setSymbols([]); setMtError(false); return; }
    let cancelled = false;
    setLoading(true);
    setMtError(false);
    marketApi.getSymbols(accountId)
      .then((list) => {
        if (cancelled) return;
        setSymbols(list);
        setLoading(false);
      })
      .catch((e) => {
        if (cancelled) return;
        setLoading(false);
        if (isMTSessionError(e)) setMtError(true);
      });

    return () => { cancelled = true; };
  }, [accountId]);

  const _toggleWatchlist = useCallback((sym: string) => {
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
        label: t('market.watchlist', { defaultValue: 'Watchlist' }),
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
        label: t('market.allSymbols', { defaultValue: 'All Symbols' }),
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
  }, [symbols, watchlist, t]);

  return (
    <Select
      showSearch
      value={value || undefined}
      onChange={(v) => onChange?.(v)}
      onDropdownVisibleChange={onDropdownVisibleChange}
      loading={loading}
      placeholder={placeholder || t('market.selectSymbol', { defaultValue: 'Select symbol' })}
      style={{ minWidth: 120, ...style }}
      filterOption={(input, option) => {
        if (!option?.value) return false;
        return String(option.value).toLowerCase().includes(input.toLowerCase());
      }}
      options={options}
      notFoundContent={loading ? t('market.loadingSymbols', { defaultValue: 'Loading...' }) : mtError ? t('market.mtSessionLost', { defaultValue: '⚠ MT session lost — reconnecting…' }) : t('market.noSymbolsFound', { defaultValue: 'No symbols found' })}
    />
  );
}
