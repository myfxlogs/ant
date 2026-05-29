import { useState, useEffect, useCallback, useMemo, useRef, lazy, Suspense } from 'react';
import { Input, Card, Row, Col, Statistic, Tag, Spin, Empty, AutoComplete } from 'antd';
import { SearchOutlined, StarFilled, StarOutlined, RiseOutlined, FallOutlined, MinusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketClient } from '@/client/connect';
import type { GetSymbolStatsResponse, TickMsg } from '@/gen/ant/v1/market_service_pb';

const PriceChart = lazy(() => import('@/components/chart/PriceChart'));

// Common trading symbols across major asset classes.
const COMMON_SYMBOLS = [
  // Forex majors
  'EURUSD', 'GBPUSD', 'USDJPY', 'AUDUSD', 'NZDUSD', 'USDCAD', 'USDCHF',
  // Forex crosses
  'EURGBP', 'EURJPY', 'GBPJPY', 'AUDJPY', 'NZDJPY', 'CADJPY', 'CHFJPY',
  // Metals
  'XAUUSD', 'XAGUSD',
  // Crypto (USD pairs)
  'BTCUSD', 'ETHUSD',
  // Indices
  'US30', 'US100', 'GER40',
];

const WATCHLIST_KEY = 'ant_watchlist_symbols';

function loadWatchlist(): string[] {
  try {
    const raw = localStorage.getItem(WATCHLIST_KEY);
    return raw ? (JSON.parse(raw) as string[]) : [];
  } catch {
    return [];
  }
}

function saveWatchlist(list: string[]) {
  localStorage.setItem(WATCHLIST_KEY, JSON.stringify(list));
}

interface StatState {
  bid: string;
  ask: string;
  spread: string;
  loading: boolean;
}

export default function Market() {
  const { t } = useTranslation();
  const [symbol, setSymbol] = useState<string>('');
  const [inputValue, setInputValue] = useState<string>('');
  const [timeframe, setTimeframe] = useState('1h');
  const [watchlist, setWatchlist] = useState<string[]>(loadWatchlist);
  const [stats, setStats] = useState<StatState>({ bid: '-', ask: '-', spread: '-', loading: false });

  const tickAbortRef = useRef<AbortController | null>(null);

  // Fetch initial symbol stats + subscribe to real-time tick stream.
  useEffect(() => {
    if (!symbol) {
      setStats({ bid: '-', ask: '-', spread: '-', loading: false });
      return;
    }
    let cancelled = false;
    setStats((s) => ({ ...s, loading: true }));

    // Initial fetch.
    marketClient.getSymbolStats({ canonical: symbol, broker: '' })
      .then((res: GetSymbolStatsResponse) => {
        if (cancelled) return;
        setStats({
          bid: res.currentBid || '-',
          ask: res.currentAsk || '-',
          spread: res.spread || '-',
          loading: false,
        });
      })
      .catch(() => {
        if (cancelled) return;
        setStats({ bid: '-', ask: '-', spread: '-', loading: false });
      });

    // Subscribe to real-time tick stream for live bid/ask updates.
    const ac = new AbortController();
    tickAbortRef.current = ac;
    (async () => {
      try {
        const stream = marketClient.streamTicks(
          { canonicals: [symbol], broker: '' },
          { signal: ac.signal },
        );
        for await (const tick of stream) {
          if (cancelled || ac.signal.aborted) break;
          if (tick.canonical !== symbol) continue;
          setStats((s) => {
            const bid = tick.bid || s.bid;
            const ask = tick.ask || s.ask;
            const bidN = parseFloat(bid);
            const askN = parseFloat(ask);
            const spread = (bidN && askN) ? (askN - bidN).toFixed(5) : s.spread;
            return { ...s, bid, ask, spread, loading: false };
          });
        }
      } catch {
        // Stream ended or aborted — ignore.
      }
    })();

    return () => {
      cancelled = true;
      ac.abort();
    };
  }, [symbol]);

  const isInWatchlist = watchlist.includes(symbol);

  const toggleWatchlist = useCallback(() => {
    if (!symbol) return;
    setWatchlist((prev) => {
      const next = prev.includes(symbol)
        ? prev.filter((s) => s !== symbol)
        : [...prev, symbol];
      saveWatchlist(next);
      return next;
    });
  }, [symbol]);

  // Build a flat set of all known symbols for case-insensitive lookup.
  const allKnownSymbols = useMemo(() => {
    const set = new Set<string>();
    COMMON_SYMBOLS.forEach((s) => set.add(s.toUpperCase()));
    watchlist.forEach((s) => set.add(s.toUpperCase()));
    return [...set].sort();
  }, [watchlist]);

  // Autocomplete options: filtered case-insensitively by current input.
  const autoOptions = useMemo(() => {
    const search = inputValue.trim().toUpperCase();
    const matches = (s: string) => !search || s.toUpperCase().includes(search);

    const matchedWatchlist = watchlist.filter(matches);
    const matchedCommon = COMMON_SYMBOLS.filter((s) => !watchlist.includes(s) && matches(s));

    const opts: { label: string; options: { value: string; label: React.ReactNode }[] }[] = [];
    if (matchedWatchlist.length > 0) {
      opts.push({
        label: t('market.watchlist'),
        options: matchedWatchlist.map((s) => ({
          value: s,
          label: <span><StarFilled style={{ color: '#D4AF37', fontSize: 12, marginRight: 6 }} />{s}</span>,
        })),
      });
    }
    if (matchedCommon.length > 0) {
      opts.push({
        label: t('market.popularSymbols'),
        options: matchedCommon.map((s) => ({ value: s, label: s })),
      });
    }
    return opts;
  }, [watchlist, inputValue, t]);

  const bidNum = parseFloat(stats.bid);
  const askNum = parseFloat(stats.ask);
  const midPrice = (bidNum && askNum) ? ((bidNum + askNum) / 2).toFixed(5) : '-';

  return (
    <div style={{ padding: '0 0 24px 0' }}>
      <h2 style={{ marginBottom: 16 }}>{t('menu.market')}</h2>

      {/* Symbol search bar */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col xs={24} sm={8} md={6}>
          <AutoComplete
            style={{ width: '100%' }}
            options={autoOptions}
            value={inputValue}
            onChange={setInputValue}
            onSelect={(val: string) => {
              setSymbol(val);
              setInputValue(val);
            }}
            onSearch={setInputValue}
            defaultActiveFirstOption
          >
            <Input
              prefix={<SearchOutlined style={{ color: '#9ca3af' }} />}
              placeholder={t('market.searchPlaceholder')}
              onPressEnter={() => {
                const v = inputValue.trim().toUpperCase();
                if (!v) return;
                // Case-insensitive exact match from known symbols.
                const match = allKnownSymbols.find((s) => s.toUpperCase() === v);
                if (match) { setSymbol(match); setInputValue(match); }
                else { setSymbol(v); setInputValue(v); }
              }}
              suffix={
                symbol ? (
                  <span onClick={toggleWatchlist} style={{ cursor: 'pointer', color: isInWatchlist ? '#D4AF37' : '#9ca3af' }}>
                    {isInWatchlist ? <StarFilled /> : <StarOutlined />}
                  </span>
                ) : null
              }
            />
          </AutoComplete>
        </Col>
      </Row>

      {!symbol ? (
        <Empty
          description={t('market.noSymbolSelected')}
          style={{ padding: 48 }}
        >
          <div style={{ marginTop: 16 }}>
            <div style={{ fontSize: 14, color: '#8c8c8c', marginBottom: 12 }}>{t('market.popularSymbols')}</div>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8, justifyContent: 'center' }}>
              {COMMON_SYMBOLS.slice(0, 12).map((s) => (
                <Tag
                  key={s}
                  style={{ cursor: 'pointer', padding: '4px 12px', fontSize: 14 }}
                  onClick={() => { setSymbol(s); setInputValue(s); }}
                >
                  {s}
                </Tag>
              ))}
            </div>
          </div>
        </Empty>
      ) : (
        <>
          {/* Stats bar */}
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col xs={12} sm={6} md={4}>
              <Card size="small">
                <Statistic
                  title={t('market.bid')}
                  value={stats.loading ? '...' : stats.bid}
                  prefix={<RiseOutlined style={{ color: '#E53935' }} />}
                  valueStyle={{ fontSize: 16, fontFamily: 'monospace' }}
                />
              </Card>
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Card size="small">
                <Statistic
                  title={t('market.ask')}
                  value={stats.loading ? '...' : stats.ask}
                  prefix={<FallOutlined style={{ color: '#00A651' }} />}
                  valueStyle={{ fontSize: 16, fontFamily: 'monospace' }}
                />
              </Card>
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Card size="small">
                <Statistic
                  title={t('market.spread')}
                  value={stats.loading ? '...' : stats.spread}
                  prefix={<MinusOutlined />}
                  valueStyle={{ fontSize: 16, fontFamily: 'monospace' }}
                />
              </Card>
            </Col>
            <Col xs={12} sm={6} md={4}>
              <Card size="small">
                <Statistic
                  title={t('market.mid')}
                  value={stats.loading ? '...' : midPrice}
                  valueStyle={{ fontSize: 16, fontFamily: 'monospace' }}
                />
              </Card>
            </Col>
          </Row>

          {/* Chart */}
          <div style={{ marginBottom: 16 }}>
            <Suspense fallback={<div style={{ display: 'flex', justifyContent: 'center', padding: 48 }}><Spin /></div>}>
              <PriceChart
                symbol={symbol}
                timeframe={timeframe}
                onTimeframeChange={setTimeframe}
              />
            </Suspense>
          </div>

          {/* Watchlist quick-access */}
          {watchlist.length > 0 && (
            <Card size="small" title={t('market.watchlist')} style={{ marginBottom: 16 }}>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {watchlist.map((s) => (
                  <Tag
                    key={s}
                    color={s === symbol ? 'gold' : 'default'}
                    style={{ cursor: 'pointer', padding: '2px 10px' }}
                    onClick={() => { setSymbol(s); setInputValue(s); }}
                  >
                    {s === symbol ? <StarFilled style={{ marginRight: 4 }} /> : <StarOutlined style={{ marginRight: 4 }} />}
                    {s}
                  </Tag>
                ))}
              </div>
            </Card>
          )}
        </>
      )}
    </div>
  );
}
