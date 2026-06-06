import { Button, Popconfirm, Space, Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { SanctionedCountry, UserKYCItem } from '@/gen/ant/v1/admin_jurisdiction_pb';

export function getJurisdictionColumns({
  t, onRemoveCountry, onSetKYC, onOverride,
}: {
  t: (key: string, opts?: any) => string;
  onRemoveCountry: (code: string) => void;
  onSetKYC: (row: UserKYCItem) => void;
  onOverride: (row: UserKYCItem) => void;
}) {
  const countryColumns: ColumnsType<SanctionedCountry> = [
    { title: t('admin.jurisdiction.countryCode'), dataIndex: 'countryCode', width: 120 },
    { title: t('admin.jurisdiction.countryLabel'), dataIndex: 'label' },
    { title: t('admin.jurisdiction.addedBy'), dataIndex: 'addedBy', width: 200, ellipsis: true },
    {
      title: t('admin.jurisdiction.actions'),
      width: 100,
      render: (_, row) => (
        <Button size="small" danger onClick={() => onRemoveCountry(row.countryCode)}>
          {t('common.remove')}
        </Button>
      ),
    },
  ];

  const kycColumns: ColumnsType<UserKYCItem> = [
    { title: t('admin.jurisdiction.userEmail'), dataIndex: 'email', width: 200, ellipsis: true },
    {
      title: t('admin.jurisdiction.kycStatus'),
      dataIndex: 'kycStatus',
      width: 100,
      render: (v: string) => (
        <Tag color={v === 'verified' ? 'green' : v === 'rejected' ? 'red' : 'orange'}>{v}</Tag>
      ),
    },
    { title: t('admin.jurisdiction.country'), dataIndex: 'countryCode', width: 80 },
    {
      title: t('admin.jurisdiction.sanctioned'),
      dataIndex: 'isSanctioned',
      width: 100,
      render: (v: boolean) => v ? <Tag color="red">{t('common.yes')}</Tag> : <Tag>{t('common.no')}</Tag>,
    },
    {
      title: t('admin.jurisdiction.disclaimer'),
      dataIndex: 'disclaimerAccepted',
      width: 100,
      render: (v: boolean) => v ? <Tag color="green">{t('common.yes')}</Tag> : <Tag color="orange">{t('common.no')}</Tag>,
    },
    {
      title: t('admin.jurisdiction.questionnaire'),
      dataIndex: 'questionnaireCompleted',
      width: 120,
      render: (v: boolean) => v ? <Tag color="green">{t('common.yes')}</Tag> : <Tag color="orange">{t('common.no')}</Tag>,
    },
    {
      title: t('admin.jurisdiction.override'),
      dataIndex: 'sanctionedOverride',
      width: 100,
      render: (v: boolean) => v ? <Tag color="blue">{t('common.yes')}</Tag> : <Tag>{t('common.no')}</Tag>,
    },
    {
      title: t('admin.jurisdiction.actions'),
      width: 200,
      render: (_, row) => (
        <Space size="small">
          <Button size="small" onClick={() => onSetKYC(row)}>
            {t('admin.jurisdiction.setKYC')}
          </Button>
          <Popconfirm
            title={row.sanctionedOverride ? t('admin.jurisdiction.confirmRevokeOverride') : t('admin.jurisdiction.confirmGrantOverride')}
            description={t('admin.jurisdiction.overrideWarning')}
            onConfirm={() => onOverride(row)}
            okText={t('common.confirm')}
            cancelText={t('common.cancel')}
            okButtonProps={{ danger: true }}
          >
            <Button size="small">
              {row.sanctionedOverride ? t('admin.jurisdiction.revokeOverride') : t('admin.jurisdiction.grantOverride')}
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return { countryColumns, kycColumns };
}
