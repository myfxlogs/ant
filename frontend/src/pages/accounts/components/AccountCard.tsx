import { Button, Dropdown, Modal, Spin, Tag } from 'antd';
import type { MenuProps } from 'antd';
import {
  LineChartOutlined,
  MoreOutlined,
  EditOutlined,
  InfoCircleOutlined,
  UnorderedListOutlined,
  PauseCircleOutlined,
  CaretRightOutlined,
  DeleteOutlined,
} from '@ant-design/icons';
import { useMemo } from 'react';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next'
import { CARD_ACTIONS_DETAILS_KEY, CARD_ACTIONS_ORDERS_KEY, CARD_ACTIONS_POSITIONS_KEY, CARD_DELETE_CONFIRM_CONTENT_KEY, CARD_DELETE_CONFIRM_TITLE_KEY, CARD_FIELDS_BALANCE_KEY, CARD_FIELDS_BROKER_KEY, CARD_FIELDS_EQUITY_KEY, CARD_FIELDS_SERVER_KEY, CARD_STATUS_CONNECTED_KEY, CARD_STATUS_CONNECTING_KEY, CARD_STATUS_DISABLED_KEY, CARD_STATUS_DISCONNECTED_KEY, CARD_STATUS_ERROR_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;

type Props = {
  account: Account;
  enablingAccount: string | null;
  realtimeInfo?: { balance: number; equity: number; profit: number };
  onDisable: (id: string) => void;
  onEnable: (id: string) => void;
  onDelete: (id: string) => void;
  onEdit: (account: Account) => void;
  onConnect: (id: string) => void;
  onNavigateToTrading: (accountId: string) => void;
  onNavigateToDetail: (accountId: string) => void;
};

const getStatusIndicator = (account: Account, t: (key: string) => string) => {
  const status = (account.status || account.accountStatus || '').toLowerCase();
  if (status === 'disconnected' || status === 'frozen' || account.isDisabled === true) {
    return { icon: '⚪', color: 'var(--color-text-muted)', text: t(CARD_STATUS_DISABLED_KEY) };
  }
  switch (status) {
    case 'connected':
      return { icon: '🟢', color: '#00A651', text: t(CARD_STATUS_CONNECTED_KEY) };
    case 'connecting':
      return { icon: '🟡', color: '#FF9800', text: t(CARD_STATUS_CONNECTING_KEY) };
    case 'disconnected':
      return { icon: '🔴', color: '#E53935', text: t(CARD_STATUS_DISCONNECTED_KEY) };
    case 'error':
      return { icon: '🔴', color: '#E53935', text: t(CARD_STATUS_ERROR_KEY) };
    case 'circuit_open':
      return { icon: '🔴', color: '#E53935', text: t('accounts.status.circuit_open', 'Circuit Open') };
    case 'circuit_half_open':
      return { icon: '🟡', color: '#FF9800', text: t('accounts.status.circuit_half_open', 'Circuit Testing') };
    default:
      return { icon: '⚪', color: 'var(--color-text-muted)', text: t('common.unknown') };
  }
};

export default function AccountCard({
  account,
  enablingAccount,
  realtimeInfo,
  onDisable,
  onEnable,
  onDelete,
  onEdit,
  onConnect,
  onNavigateToTrading,
  onNavigateToDetail,
}: Props) {
  const { t } = useTranslation();
  const status = getStatusIndicator(account, t);
  const balance = realtimeInfo?.balance ?? account.balance ?? 0;
  const equity = realtimeInfo?.equity ?? account.equity ?? 0;

  const balanceDisplay = useMemo(() => {
    const isNegative = balance < 0;
    const color = isNegative ? '#E53935' : 'var(--color-text)';
    return { text: `${isNegative ? '-' : ''}${Math.abs(balance).toFixed(2)} ${account.currency || 'USD'}`, color };
  }, [balance, account]);

  const equityDisplay = useMemo(() => {
    const isNegative = equity < 0;
    const color = isNegative ? '#E53935' : 'var(--color-text)';
    return { text: `${isNegative ? '-' : ''}${Math.abs(equity).toFixed(2)} ${account.currency || 'USD'}`, color };
  }, [equity, account]);

  const handleStatusClick = () => {
    if (!account.isDisabled && account.status !== 'connected') {
      onConnect(account.id);
    }
  };

  const menuItems: MenuProps['items'] = [
    {
      key: 'toggle',
      label: account.isDisabled ? t('common.enable') : t('common.disable'),
      icon: account.isDisabled ? (
        enablingAccount === account.id ? (
          <Spin size="small" />
        ) : (
          <CaretRightOutlined style={{ fontSize: 14 }} />
        )
      ) : (
        <PauseCircleOutlined style={{ fontSize: 14 }} />
      ),
      onClick: () => {
        if (account.isDisabled) {
          onEnable(account.id);
        } else {
          onDisable(account.id);
        }
      },
    },
    {
      key: 'edit',
      label: t('common.edit'),
      icon: <EditOutlined style={{ fontSize: 14 }} />,
      onClick: () => onEdit(account),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: <DeleteOutlined style={{ fontSize: 14 }} />,
      danger: true,
      onClick: () => {
        Modal.confirm({
          title: t(CARD_DELETE_CONFIRM_TITLE_KEY),
          content: t(CARD_DELETE_CONFIRM_CONTENT_KEY),
          okText: t('common.confirm'),
          cancelText: t('common.cancel'),
          onOk: () => onDelete(account.id),
        });
      },
    },
  ];

  return (
    <div
      key={account.id}
      className="rounded-2xl overflow-hidden transition-all"
      style={{
        background: 'var(--color-bg-card)',
        boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
        opacity: account.isDisabled ? 0.6 : 1,
      }}
    >
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="text-xl">{status.icon}</span>
            <div>
              <div className="font-semibold text-lg" style={{ color: 'var(--color-text)' }}>
                {account.login}
              </div>
              <Tag
                color={account.mtType === 'MT4' ? 'blue' : 'purple'}
                style={{ borderRadius: '4px', marginLeft: '8px' }}
              >
                {account.mtType}
              </Tag>
            </div>
          </div>
          <Tag
            style={{
              background: `${status.color}20`,
              color: status.color,
              border: 'none',
              borderRadius: '6px',
              cursor: !account.isDisabled && account.status !== 'connected' ? 'pointer' : 'default',
            }}
            onClick={handleStatusClick}
          >
            {status.text}
          </Tag>
          <Dropdown menu={{ items: menuItems }} trigger={['click']}>
            <Button
              type="text"
              size="small"
              icon={<MoreOutlined style={{ fontSize: 16 }} />}
              style={{ color: 'var(--color-text-muted)' }}
            />
          </Dropdown>
        </div>

        <div className="space-y-2 mb-4">
          <div className="flex justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>{t(CARD_FIELDS_BALANCE_KEY)}</span>
            <span className="font-medium" style={{ color: balanceDisplay.color }}>
              {balanceDisplay.text}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>{t(CARD_FIELDS_EQUITY_KEY)}</span>
            <span className="font-medium" style={{ color: equityDisplay.color }}>
              {equityDisplay.text}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>{t(CARD_FIELDS_BROKER_KEY)}</span>
            <span className="font-medium" style={{ color: 'var(--color-text)' }}>
              {account.brokerCompany}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: 'var(--color-text-muted)' }}>{t(CARD_FIELDS_SERVER_KEY)}</span>
            <span className="font-medium" style={{ color: 'var(--color-text)' }}>
              {account.brokerServer}
            </span>
          </div>
        </div>

        <div className="flex gap-2 pt-3" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.06)' }}>
          <Button
            size="small"
            icon={<LineChartOutlined style={{ fontSize: 14 }} />}
            onClick={() => onNavigateToTrading(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t(CARD_ACTIONS_POSITIONS_KEY)}
          </Button>
          <Button
            size="small"
            icon={<UnorderedListOutlined style={{ fontSize: 14 }} />}
            onClick={() => onNavigateToDetail(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t(CARD_ACTIONS_ORDERS_KEY)}
          </Button>
          <Button
            size="small"
            icon={<InfoCircleOutlined style={{ fontSize: 14 }} />}
            onClick={() => onNavigateToDetail(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t(CARD_ACTIONS_DETAILS_KEY)}
          </Button>
        </div>
      </div>
    </div>
  );
}
