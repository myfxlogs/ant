import { useState, useMemo, useRef, lazy, Suspense } from 'react';
import { Input, Card, Row, Col, Statistic, Tag, Spin, Empty, AutoComplete } from 'antd';
import { SearchOutlined, StarFilled, StarOutlined, RiseOutlined, FallOutlined, MinusOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { useWatchlist } from './useWatchlist';
import { useSymbolStats } from './useSymbolStats';

const PriceChart = lazy(() => import('@/components/chart/PriceChart'));

const COMMON_SYMBOLS = [
  'EURUSD', 'GBPUSD', 'USDJPY', 'AUDUSD', 'NZDUSD', 'USDCAD', 'USDCHF',
  'EURGBP', 'EURJPY', 'GBPJPY', 'AUDJPY', 'NZDJPY', 'CADJPY', 'CHFJPY',
  'XAUUSD', 'XAGUSD', 'BTCUSD', 'ETHUSD', 'US30', 'US100', 'GER40',
];

export default function Market() {
  const { t } = useTranslation();
  const [symbol, setSymbol] = useState<string>('');
  const [inputValue, setInputValue] = useState<string>('');
  const [timeframe, setTimeframe] = useState('1h');
  const { watchlist, isInWatchlist, toggleWatchlist } = useWatchlist();
  const stats = useSymbolStats(symbol);

  const handleToggleWatchlist = () => toggleWatchlist(symbol);

  const allKnownSymbols = useMemo(() => {
    const set = new Set<string>();
    COMMON_SYMBOLS.forEach((s) => set.add(s.toUpperCase()));
    watchlist.forEach((s) => set.add(s.toUpperCase()));
    return [...set].sort();
  }, [watchlist]);

  const autoOptions = useMemo(() => {
    const q = (inputValue || '').toUpperCase();
    return allKnownSymbols
      .filter((s) => !q || s.includes(q))
      .map((s) => ({ value: s }));
  }, [allKnownSymbols, inputValue]);

  const handleSelect = (value: string) => { setSymbol(value.toUpperCase()); setInputValue(value.toUpperCase()); };

  const bidNum = parseFloat(stats.bid);
  const askNum = parseFloat(stats.ask);
  const prevBidRef = useRef(bidNum);
  const prevAskRef = useRef(askNum);
  const bidIcon = !bidNum ? <MinusOutlined /> : bidNum > (prevBidRef.current || bidNum) ? <RiseOutlined style={{ color: '#22c55e' }} /> : <FallOutlined style={{ color: '#ef5350' }} />;
  const askIcon = !askNum ? <MinusOutlined /> : askNum > (prevAskRef.current || askNum) ? <RiseOutlined style={{ color: '#22c55e' }} /> : <FallOutlined style={{ color: '#ef5350' }} />;

  return (
    <div style={{ padding: '16px 24px', maxWidth: 1400, margin: '0 auto' }}>
      <div style={{ display: 'flex', gap: 16, marginBottom: 16, alignItems: 'flex-start' }}>
        <Card size="small" style={{ flex: '0 0 320px' }}>
          <div style={{ display: 'flex', gap: 8, marginBottom: 12 }}>
            <AutoComplete
              value={inputValue}
              onChange={setInputValue}
              onSelect={handleSelect}
              options={autoOptions}
              style={{ flex: 1 }}
              placeholder={t('market.searchSymbol')}
            >
              <Input prefix={<SearchOutlined />} />
            </AutoComplete>
            <span onClick={handleToggleWatchlist} style={{ cursor: 'pointer', fontSize: 18, lineHeight: '32px' }}>
              {isInWatchlist(symbol) ? <StarFilled style={{ color: '#fadb14' }} /> : <StarOutlined />}
            </span>
          </div>
          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginBottom: 12 }}>
            {watchlist.map((s) => (
              <Tag key={s} color={s === symbol ? 'blue' : 'default'} style={{ cursor: 'pointer' }}
                onClick={() => { setSymbol(s); setInputValue(s); }}>
                {s}
              </Tag>
            ))}
            {watchlist.length === 0 && <span style={{ color: '#8c8c8c', fontSize: 12 }}>{t('market.emptyWatchlist')}</span>}
          </div>
          <Row gutter={12}>
            <Col span={8}><Statistic title={t('market.bid')} value={stats.bid} loading={stats.loading} prefix={bidIcon} valueStyle={{ fontSize: 18 }} /></Col>
            <Col span={8}><Statistic title={t('market.ask')} value={stats.ask} loading={stats.loading} prefix={askIcon} valueStyle={{ fontSize: 18 }} /></Col>
            <Col span={8}><Statistic title={t('market.spread')} value={stats.spread} loading={stats.loading} valueStyle={{ fontSize: 18 }} /></Col>
          </Row>
          <div style={{ marginTop: 12 }}>
            <span style={{ fontSize: 11, color: '#8c8c8c' }}>{t('market.common')}</span>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: 4, marginTop: 4 }}>
              {COMMON_SYMBOLS.map((s) => (
                <Tag key={s} color={s === symbol ? 'blue' : 'default'} style={{ cursor: 'pointer', fontSize: 10 }}
                  onClick={() => { setSymbol(s); setInputValue(s); }}>
                  {s}
                </Tag>
              ))}
            </div>
          </div>
        </Card>
        <div style={{ flex: '1 1 0', minHeight: 500 }}>
          {symbol ? (
            <Suspense fallback={<div style={{ height: 500, display: 'flex', alignItems: 'center', justifyContent: 'center' }}><Spin /></div>}>
              <PriceChart symbol={symbol} timeframe={timeframe} onTimeframeChange={setTimeframe} />
            </Suspense>
          ) : (
            <Empty description={t('market.selectSymbol')} style={{ marginTop: 120 }} />
          )}
        </div>
      </div>
    </div>
  );
}
