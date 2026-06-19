import {
  LineChartOutlined, RiseOutlined, FallOutlined, PercentageOutlined,
  WalletOutlined, DollarOutlined, WarningOutlined,
} from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { DETAIL_CARDS_BALANCE_KEY, DETAIL_CARDS_CREDIT_KEY, DETAIL_CARDS_EQUITY_KEY, DETAIL_CARDS_FLOATING_PROFIT_KEY, DETAIL_CARDS_MARGIN_FREE_KEY, DETAIL_CARDS_MARGIN_LEVEL_KEY, DETAIL_CARDS_MARGIN_USED_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;
import { SmallInfoCard } from './AccountDetail.shared';

type Props = {
  isStreamLoading: boolean;
  disabled: boolean;
  formatCurrency: (value: number) => string;
  balance: number;
  equity: number;
  profit: number;
  profitPercent: number;
  margin: number;
  freeMargin: number;
  marginLevel: number;
  credit: number;
};

export default function AccountMetricsCards({
  isStreamLoading, disabled, formatCurrency,
  balance, equity, profit, profitPercent,
  margin, freeMargin, marginLevel, credit,
}: Props) {
  const { t } = useTranslation();

  return (
    <>
      {/* ── Primary metrics: Equity · P&L · Margin Level ── */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-4">
        {/* Equity */}
        <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
          <div className="flex items-center gap-2 mb-1">
            <LineChartOutlined style={{ color: 'var(--color-text-muted)', fontSize: 14 }} />
            <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t(DETAIL_CARDS_TRADING_EQUITY_KEY)}</span>
          </div>
          {isStreamLoading
            ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
            : <div className="text-xl font-bold" style={{ color: 'var(--color-text)' }}>{formatCurrency(equity)}</div>
          }
          <div style={{ color: 'var(--color-text-muted)', fontSize: 11, marginTop: 2 }}>
            {t(DETAIL_CARDS_TRADING_BALANCE_KEY)}: {formatCurrency(balance)}
          </div>
        </div>

        {/* Floating P&L */}
        <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
          <div className="flex items-center gap-2 mb-1">
            {profit >= 0 ? <RiseOutlined style={{ color: 'var(--color-success)', fontSize: 14 }} /> : <FallOutlined style={{ color: 'var(--color-danger)', fontSize: 14 }} />}
            <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t(DETAIL_CARDS_FLOATING_TRADING_PROFIT_KEY)}</span>
          </div>
          {isStreamLoading
            ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
            : <>
              <div className="text-xl font-bold" style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)' }}>
                {profit >= 0 ? '+' : ''}{formatCurrency(profit)}
              </div>
              <div style={{ color: profit >= 0 ? 'var(--color-success)' : 'var(--color-danger)', fontSize: 12, marginTop: 2 }}>
                {disabled ? '--' : `${profitPercent >= 0 ? '+' : ''}${profitPercent.toFixed(2)}%`}
              </div>
            </>
          }
        </div>

        {/* Margin Level */}
        <div className="rounded-xl p-4" style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}>
          <div className="flex items-center gap-2 mb-1">
            <PercentageOutlined style={{ color: 'var(--color-text-muted)', fontSize: 14 }} />
            <span style={{ color: 'var(--color-text-muted)', fontSize: 13 }}>{t(DETAIL_CARDS_TRADING_MARGIN_LEVEL_KEY)}</span>
          </div>
          {isStreamLoading
            ? <div className="text-xl" style={{ color: 'var(--color-text-muted)' }}>{t('common.loading')}</div>
            : <>
              <div className="text-xl font-bold" style={{
                color: margin > 0 && (marginLevel || 0) < 100 ? 'var(--color-danger)' : 'var(--color-text)',
              }}>
                {margin > 0 ? `${(marginLevel || 0).toFixed(2)}%` : '--'}
              </div>
              <div style={{ color: 'var(--color-text-muted)', fontSize: 11, marginTop: 2 }}>
                {t(DETAIL_CARDS_MARGIN_USED_KEY)}: {formatCurrency(margin)}
              </div>
            </>
          }
        </div>
      </div>

      {/* ── Secondary metrics: Balance · Margin · Free Margin · Credit ── */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-3 mb-4">
        <SmallInfoCard
          icon={<WalletOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
          label={t(DETAIL_CARDS_TRADING_BALANCE_KEY)}
          value={formatCurrency(balance)}
          loading={isStreamLoading}
        />
        <SmallInfoCard
          icon={<DollarOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
          label={t(DETAIL_CARDS_MARGIN_USED_KEY)}
          value={formatCurrency(margin)}
          loading={isStreamLoading}
        />
        <SmallInfoCard
          icon={<DollarOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
          label={t(DETAIL_CARDS_MARGIN_FREE_KEY)}
          value={formatCurrency(freeMargin)}
          loading={isStreamLoading}
        />
        <SmallInfoCard
          icon={<WarningOutlined style={{ color: 'var(--color-text-muted)', fontSize: 13 }} />}
          label={t(DETAIL_CARDS_CREDIT_KEY)}
          value={formatCurrency(credit)}
          loading={isStreamLoading}
        />
      </div>
    </>
  );
}
