import { Button, Modal, Tag } from 'antd';
import { CaretRightOutlined, DeleteOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next'
import { DISABLED_CONFIRM_DELETE_CONTENT_KEY, DISABLED_CONFIRM_DELETE_TITLE_KEY, DISABLED_MOBILE_BALANCE_LABEL_KEY, DISABLED_MOBILE_EQUITY_LABEL_KEY, DISABLED_TABLE_ACCOUNT_KEY, DISABLED_TABLE_ACTIONS_KEY, DISABLED_TABLE_BALANCE_KEY, DISABLED_TABLE_BROKER_KEY, DISABLED_TABLE_EQUITY_KEY, DISABLED_TABLE_TYPE_KEY, DISABLED_TITLE_KEY } from '@/gen/ant/v1/i18n/accounts_keys';

;

type Props = {
  accounts: Account[];
  onEnable: (id: string) => void;
  onDelete: (id: string) => void;
};

export default function DisabledAccountsSection({ accounts, onEnable, onDelete }: Props) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  if (!accounts || accounts.length === 0) return null;

  return (
    <div className="mt-8">
      <h3 className="text-lg font-semibold mb-4" style={{ color: 'var(--color-text-muted)' }}>
        {t(DISABLED_TRADING_TITLE_KEY)}
      </h3>
      <div
        className="hidden md:block rounded-xl overflow-hidden"
        style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)' }}
      >
        <table className="w-full">
          <thead>
            <tr style={{ background: 'var(--color-bg-secondary)' }}>
              <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_TRADING_ACCOUNT_KEY)}
              </th>
              <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_TRADING_TYPE_KEY)}
              </th>
              <th className="text-left p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_BROKER_KEY)}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_TRADING_BALANCE_KEY)}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_TRADING_EQUITY_KEY)}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: 'var(--color-text-muted)' }}>
                {t(DISABLED_TABLE_ACTIONS_KEY)}
              </th>
            </tr>
          </thead>
          <tbody>
            {accounts.map((account) => (
              <tr
                key={account.id}
                className="border-b hover:bg-gray-50"
                style={{ borderColor: 'var(--color-border)', opacity: 0.7, cursor: 'pointer' }}
                onClick={() => navigate(`/accounts/${account.id}`)}
              >
                <td className="p-3 font-medium" style={{ color: 'var(--color-text)' }}>
                  {account.login}
                </td>
                <td className="p-3">
                  <Tag color={account.mtType === 'MT4' ? 'blue' : 'purple'} style={{ borderRadius: '4px' }}>
                    {account.mtType}
                  </Tag>
                </td>
                <td className="p-3" style={{ color: 'var(--color-text-muted)' }}>
                  {account.brokerCompany || '-'}
                </td>
                <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>
                  {(account.balance || 0).toFixed(2)} {account.currency || 'USD'}
                </td>
                <td className="text-right p-3" style={{ color: 'var(--color-text)' }}>
                  {(account.equity || 0).toFixed(2)} {account.currency || 'USD'}
                </td>
                <td className="text-right p-3">
                  <div className="flex justify-end gap-2">
                    <Button
                      size="small"
                      icon={<CaretRightOutlined style={{ fontSize: 14 }} />}
                      onClick={() => onEnable(account.id)}
                      style={{ borderRadius: '6px' }}
                    >
                      {t('common.enable')}
                    </Button>
                    <Button
                      size="small"
                      danger
                      icon={<DeleteOutlined style={{ fontSize: 14 }} />}
                      onClick={() => {
                        Modal.confirm({
                          title: t(DISABLED_CONFIRM_DELETE_TRADING_TITLE_KEY),
                          content: t(DISABLED_CONFIRM_DELETE_CONTENT_KEY),
                          okText: t('common.confirm'),
                          cancelText: t('common.cancel'),
                          onOk: () => onDelete(account.id),
                        });
                      }}
                      style={{ borderRadius: '6px' }}
                    >
                      {t('common.delete')}
                    </Button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <div className="md:hidden space-y-3">
        {accounts.map((account) => (
          <div
            key={account.id}
            className="rounded-xl p-4"
            style={{ background: 'var(--color-bg-card)', boxShadow: '0 2px 8px var(--color-shadow)', opacity: 0.7, cursor: 'pointer' }}
            onClick={() => navigate(`/accounts/${account.id}`)}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="font-medium" style={{ color: 'var(--color-text)' }}>
                  {account.login}
                </span>
                <Tag color={account.mtType === 'MT4' ? 'blue' : 'purple'} style={{ borderRadius: '4px' }}>
                  {account.mtType}
                </Tag>
              </div>
              <Tag color="red">{t('common.disabled')}</Tag>
            </div>
            <div className="text-sm mb-3" style={{ color: 'var(--color-text-muted)' }}>
              {account.brokerCompany || '-'}
            </div>
            <div className="flex justify-between mb-3 text-sm">
              <div>
                <span style={{ color: 'var(--color-text-muted)' }}>{t(DISABLED_MOBILE_BALANCE_LABEL_KEY)}</span>
                <span style={{ color: 'var(--color-text)' }}>{(account.balance || 0).toFixed(2)}</span>
              </div>
              <div>
                <span style={{ color: 'var(--color-text-muted)' }}>{t(DISABLED_MOBILE_EQUITY_LABEL_KEY)}</span>
                <span style={{ color: 'var(--color-text)' }}>{(account.equity || 0).toFixed(2)}</span>
              </div>
              <div style={{ color: 'var(--color-text-muted)' }}>{account.currency || 'USD'}</div>
            </div>
            <div className="flex gap-2">
              <Button
                size="small"
                icon={<CaretRightOutlined style={{ fontSize: 14 }} />}
                onClick={() => onEnable(account.id)}
                style={{ borderRadius: '6px', flex: 1 }}
              >
                {t('common.enable')}
              </Button>
              <Button
                size="small"
                danger
                icon={<DeleteOutlined style={{ fontSize: 14 }} />}
                onClick={() => {
                  Modal.confirm({
                    title: t(DISABLED_CONFIRM_DELETE_TRADING_TITLE_KEY),
                    content: t(DISABLED_CONFIRM_DELETE_CONTENT_KEY),
                    okText: t('common.confirm'),
                    cancelText: t('common.cancel'),
                    onOk: () => onDelete(account.id),
                  });
                }}
                style={{ borderRadius: '6px', flex: 1 }}
              >
                {t('common.delete')}
              </Button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
