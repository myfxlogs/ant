import { Button, Dropdown, Modal, Spin, Tag } from 'antd';
import type { MenuProps } from 'antd';
import {
  IconChartLine,
  IconDotsVertical,
  IconEdit,
  IconInfoCircle,
  IconList,
  IconPlayerPause,
  IconPlayerPlay,
  IconTrash,
} from '@tabler/icons-react';
import { useMemo } from 'react';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next';

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
  if ((account as any).isDisabled) {
    return { icon: '⚪', color: '#8A9AA5', text: t('accounts.card.status.disabled') };
  }
  switch ((account as any).status) {
    case 'connected':
      return { icon: '🟢', color: '#00A651', text: t('accounts.card.status.connected') };
    case 'connecting':
      return { icon: '🟡', color: '#FF9800', text: t('accounts.card.status.connecting') };
    case 'disconnected':
      return { icon: '🔴', color: '#E53935', text: t('accounts.card.status.disconnected') };
    case 'error':
      return { icon: '🔴', color: '#E53935', text: t('accounts.card.status.error') };
    default:
      return { icon: '⚪', color: '#8A9AA5', text: t('common.unknown') };
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
  const balance = realtimeInfo?.balance ?? (account as any).balance ?? 0;
  const equity = realtimeInfo?.equity ?? (account as any).equity ?? 0;

  const balanceDisplay = useMemo(() => {
    const isNegative = balance < 0;
    const color = isNegative ? '#E53935' : '#141D22';
    return { text: `${isNegative ? '-' : ''}${Math.abs(balance).toFixed(2)} ${(account as any).currency || 'USD'}`, color };
  }, [balance, account]);

  const equityDisplay = useMemo(() => {
    const isNegative = equity < 0;
    const color = isNegative ? '#E53935' : '#141D22';
    return { text: `${isNegative ? '-' : ''}${Math.abs(equity).toFixed(2)} ${(account as any).currency || 'USD'}`, color };
  }, [equity, account]);

  const handleStatusClick = () => {
    if (!(account as any).isDisabled && (account as any).status !== 'connected') {
      onConnect(account.id);
    }
  };

  const menuItems: MenuProps['items'] = [
    {
      key: 'toggle',
      label: (account as any).isDisabled ? t('common.enable') : t('common.disable'),
      icon: (account as any).isDisabled ? (
        enablingAccount === account.id ? (
          <Spin size="small" />
        ) : (
          <IconPlayerPlay size={14} stroke={1.5} />
        )
      ) : (
        <IconPlayerPause size={14} stroke={1.5} />
      ),
      onClick: () => {
        if ((account as any).isDisabled) {
          onEnable(account.id);
        } else {
          onDisable(account.id);
        }
      },
    },
    {
      key: 'edit',
      label: t('common.edit'),
      icon: <IconEdit size={14} stroke={1.5} />,
      onClick: () => onEdit(account),
    },
    {
      type: 'divider',
    },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: <IconTrash size={14} stroke={1.5} />,
      danger: true,
      onClick: () => {
        Modal.confirm({
          title: t('accounts.card.deleteConfirm.title'),
          content: t('accounts.card.deleteConfirm.content'),
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
        background: '#FFFFFF',
        boxShadow: '0 4px 24px rgba(0, 0, 0, 0.08)',
        opacity: (account as any).isDisabled ? 0.6 : 1,
      }}
    >
      <div className="p-5">
        <div className="flex items-center justify-between mb-4">
          <div className="flex items-center gap-3">
            <span className="text-xl">{status.icon}</span>
            <div>
              <div className="font-semibold text-lg" style={{ color: '#141D22' }}>
                {(account as any).login}
              </div>
              <Tag
                color={(account as any).mtType === 'MT4' ? 'blue' : 'purple'}
                style={{ borderRadius: '4px', marginLeft: '8px' }}
              >
                {(account as any).mtType}
              </Tag>
            </div>
          </div>
          <Tag
            style={{
              background: `${status.color}20`,
              color: status.color,
              border: 'none',
              borderRadius: '6px',
              cursor: !(account as any).isDisabled && (account as any).status !== 'connected' ? 'pointer' : 'default',
            }}
            onClick={handleStatusClick}
          >
            {status.text}
          </Tag>
          <Dropdown menu={{ items: menuItems }} trigger={['click']}>
            <Button
              type="text"
              size="small"
              icon={<IconDotsVertical size={16} stroke={1.5} />}
              style={{ color: '#8A9AA5' }}
            />
          </Dropdown>
        </div>

        <div className="space-y-2 mb-4">
          <div className="flex justify-between">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.card.fields.balance')}</span>
            <span className="font-medium" style={{ color: balanceDisplay.color }}>
              {balanceDisplay.text}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.card.fields.equity')}</span>
            <span className="font-medium" style={{ color: equityDisplay.color }}>
              {equityDisplay.text}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.card.fields.broker')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>
              {(account as any).brokerCompany}
            </span>
          </div>
          <div className="flex justify-between">
            <span style={{ color: '#8A9AA5' }}>{t('accounts.card.fields.server')}</span>
            <span className="font-medium" style={{ color: '#141D22' }}>
              {(account as any).brokerServer}
            </span>
          </div>
        </div>

        <div className="flex gap-2 pt-3" style={{ borderTop: '1px solid rgba(0, 0, 0, 0.06)' }}>
          <Button
            size="small"
            icon={<IconChartLine size={14} stroke={1.5} />}
            onClick={() => onNavigateToTrading(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t('accounts.card.actions.positions')}
          </Button>
          <Button
            size="small"
            icon={<IconList size={14} stroke={1.5} />}
            onClick={() => onNavigateToDetail(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t('accounts.card.actions.orders')}
          </Button>
          <Button
            size="small"
            icon={<IconInfoCircle size={14} stroke={1.5} />}
            onClick={() => onNavigateToDetail(account.id)}
            style={{ borderRadius: '6px' }}
          >
            {t('accounts.card.actions.details')}
          </Button>
        </div>
      </div>
    </div>
  );
}
