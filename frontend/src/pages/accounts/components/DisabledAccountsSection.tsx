import { Button, Modal, Tag } from 'antd';
import { IconPlayerPlay, IconTrash } from '@tabler/icons-react';
import type { Account } from '@/types/account';
import { useTranslation } from 'react-i18next';

type Props = {
  accounts: Account[];
  onEnable: (id: string) => void;
  onDelete: (id: string) => void;
};

export default function DisabledAccountsSection({ accounts, onEnable, onDelete }: Props) {
  const { t } = useTranslation();
  if (!accounts || accounts.length === 0) return null;

  return (
    <div className="mt-8">
      <h3 className="text-lg font-semibold mb-4" style={{ color: '#8A9AA5' }}>
        {t('accounts.disabled.title')}
      </h3>
      <div
        className="hidden md:block rounded-xl overflow-hidden"
        style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)' }}
      >
        <table className="w-full">
          <thead>
            <tr style={{ background: '#F5F7F9' }}>
              <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.account')}
              </th>
              <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.type')}
              </th>
              <th className="text-left p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.broker')}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.balance')}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.equity')}
              </th>
              <th className="text-right p-3 text-sm font-medium" style={{ color: '#8A9AA5' }}>
                {t('accounts.disabled.table.actions')}
              </th>
            </tr>
          </thead>
          <tbody>
            {accounts.map((account) => (
              <tr
                key={account.id}
                className="border-b hover:bg-gray-50"
                style={{ borderColor: 'rgba(0, 0, 0, 0.06)', opacity: 0.7 }}
              >
                <td className="p-3 font-medium" style={{ color: '#141D22' }}>
                  {(account as any).login}
                </td>
                <td className="p-3">
                  <Tag color={(account as any).mtType === 'MT4' ? 'blue' : 'purple'} style={{ borderRadius: '4px' }}>
                    {(account as any).mtType}
                  </Tag>
                </td>
                <td className="p-3" style={{ color: '#8A9AA5' }}>
                  {(account as any).brokerCompany || '-'}
                </td>
                <td className="text-right p-3" style={{ color: '#141D22' }}>
                  {(((account as any).balance || 0) as number).toFixed(2)} {(account as any).currency || 'USD'}
                </td>
                <td className="text-right p-3" style={{ color: '#141D22' }}>
                  {(((account as any).equity || 0) as number).toFixed(2)} {(account as any).currency || 'USD'}
                </td>
                <td className="text-right p-3">
                  <div className="flex justify-end gap-2">
                    <Button
                      size="small"
                      icon={<IconPlayerPlay size={14} stroke={1.5} />}
                      onClick={() => onEnable(account.id)}
                      style={{ borderRadius: '6px' }}
                    >
                      {t('common.enable')}
                    </Button>
                    <Button
                      size="small"
                      danger
                      icon={<IconTrash size={14} stroke={1.5} />}
                      onClick={() => {
                        Modal.confirm({
                          title: t('accounts.disabled.confirmDelete.title'),
                          content: t('accounts.disabled.confirmDelete.content'),
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
            style={{ background: '#FFFFFF', boxShadow: '0 2px 8px rgba(0, 0, 0, 0.06)', opacity: 0.7 }}
          >
            <div className="flex items-center justify-between mb-3">
              <div className="flex items-center gap-2">
                <span className="font-medium" style={{ color: '#141D22' }}>
                  {(account as any).login}
                </span>
                <Tag color={(account as any).mtType === 'MT4' ? 'blue' : 'purple'} style={{ borderRadius: '4px' }}>
                  {(account as any).mtType}
                </Tag>
              </div>
              <Tag color="red">{t('common.disabled')}</Tag>
            </div>
            <div className="text-sm mb-3" style={{ color: '#8A9AA5' }}>
              {(account as any).brokerCompany || '-'}
            </div>
            <div className="flex justify-between mb-3 text-sm">
              <div>
                <span style={{ color: '#8A9AA5' }}>{t('accounts.disabled.mobile.balanceLabel')}</span>
                <span style={{ color: '#141D22' }}>{(((account as any).balance || 0) as number).toFixed(2)}</span>
              </div>
              <div>
                <span style={{ color: '#8A9AA5' }}>{t('accounts.disabled.mobile.equityLabel')}</span>
                <span style={{ color: '#141D22' }}>{(((account as any).equity || 0) as number).toFixed(2)}</span>
              </div>
              <div style={{ color: '#8A9AA5' }}>{(account as any).currency || 'USD'}</div>
            </div>
            <div className="flex gap-2">
              <Button
                size="small"
                icon={<IconPlayerPlay size={14} stroke={1.5} />}
                onClick={() => onEnable(account.id)}
                style={{ borderRadius: '6px', flex: 1 }}
              >
                {t('common.enable')}
              </Button>
              <Button
                size="small"
                danger
                icon={<IconTrash size={14} stroke={1.5} />}
                onClick={() => {
                  Modal.confirm({
                    title: t('accounts.disabled.confirmDelete.title'),
                    content: t('accounts.disabled.confirmDelete.content'),
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
