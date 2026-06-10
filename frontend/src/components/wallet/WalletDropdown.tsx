import { useState } from 'react';
import { Dropdown, Tag, Typography, Button, Descriptions, Divider } from 'antd';
import { WalletOutlined } from '@ant-design/icons';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { walletApi } from '@/client/wallet';
import { queryKeys } from '@/queries/queryKeys';
import { formatAmount } from '@/utils/amount';

const { Text } = Typography;

export default function WalletDropdown() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [open, setOpen] = useState(false);

  const { data: wallet } = useQuery({
    queryKey: queryKeys.wallet.all,
    queryFn: () => walletApi.getWallet(),
    staleTime: 30_000,
    retry: false,
  });

  const hasWallet = !!wallet;
  const balance = wallet?.balance ?? '0';
  const currency = wallet?.currency || 'USD';
  const accountNumber = wallet?.accountNumber;
  const frozen = wallet?.frozenBalance ?? '0';

  const formattedBalance = formatAmount(balance);
  const formattedFrozen = formatAmount(frozen);

  const dropdownContent = hasWallet ? (
    <div style={{ width: 280, background: 'var(--color-bg-card)', borderRadius: 8, overflow: 'hidden' }}>
      <div style={{ textAlign: 'center', padding: '20px 16px 12px' }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {t('wallet.balance')}
        </Text>
        <div
          style={{
            fontSize: 24,
            fontWeight: 700,
            color: 'var(--color-text)',
            fontFamily: 'Poppins, sans-serif',
            margin: '4px 0',
          }}
        >
          {formattedBalance} <Text style={{ fontSize: 14, color: 'var(--color-text-muted)' }}>{currency}</Text>
        </div>
        {accountNumber && (
          <Tag color="blue" style={{ marginTop: 4 }}>
            {t('wallet.accountNumber')}: {accountNumber}
          </Tag>
        )}
      </div>

      <Divider style={{ margin: 0 }} />

      <div style={{ padding: '8px 16px' }}>
        <Descriptions column={1} size="small" colon={false}>
          <Descriptions.Item
            label={<Text type="secondary">{t('wallet.frozen')}</Text>}
          >
            <Text style={{ color: 'var(--color-text-muted)' }}>
              {formattedFrozen} {currency}
            </Text>
          </Descriptions.Item>
        </Descriptions>
      </div>

      <Divider style={{ margin: 0 }} />

      <div style={{ display: 'flex', gap: 8, padding: '12px 16px' }}>
        <Button
          type="default"
          style={{ flex: 1 }}
          onClick={() => {
            setOpen(false);
            navigate('/wallet/deposit');
          }}
        >
          {t('wallet.deposit')}
        </Button>
        <Button
          type="default"
          style={{ flex: 1 }}
          onClick={() => {
            setOpen(false);
            navigate('/wallet/withdraw');
          }}
        >
          {t('wallet.withdraw')}
        </Button>
      </div>

      <div style={{ padding: '0 16px 12px' }}>
        <Button
          type="primary"
          block
          onClick={() => {
            setOpen(false);
            navigate('/wallet');
          }}
        >
          {t('wallet.history')}
        </Button>
      </div>
    </div>
  ) : (
    <div style={{ width: 280, padding: '32px 16px', textAlign: 'center', background: 'var(--color-bg-card)', borderRadius: 8 }}>
      <Text type="secondary">{t('common.loading')}</Text>
    </div>
  );

  return (
    <Dropdown
      open={open}
      onOpenChange={setOpen}
      dropdownRender={() => dropdownContent}
      trigger={['click']}
      placement="bottomRight"
    >
      <div className="flex items-center gap-1.5 p-2 rounded-lg cursor-pointer transition-colors"
           style={{ background: 'var(--color-bg-secondary)' }}>
        <WalletOutlined
          style={{ fontSize: 18, color: '#D4AF37' }}
        />
        <span
          className="hidden sm:inline text-sm"
          style={{ color: hasWallet ? 'var(--color-text)' : 'var(--color-text-muted)', fontWeight: 500 }}
        >
          {hasWallet ? formattedBalance : '—'}
        </span>
        {hasWallet && (
          <span
            className="hidden sm:inline text-xs"
            style={{ color: 'var(--color-text-muted)' }}
          >
            {currency}
          </span>
        )}
      </div>
    </Dropdown>
  );
}
