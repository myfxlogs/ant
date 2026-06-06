import { useState } from 'react';
import {
  Button,
  Card,
  Form,
  Input,
  Modal,
  Select,
  Table,
  Tabs,
  Typography,
} from 'antd';
import { adminApi } from '@/client/admin';
import { showError, showSuccess } from '@/utils/message';
import { StatusResult } from '@/components/common/StatusResult';
import { useRpcQuery } from '@/hooks/useRpcQuery';
import { useTranslation } from 'react-i18next';
import { SafetyOutlined } from '@ant-design/icons';
import type { SanctionedCountry, UserKYCItem } from '@/gen/ant/v1/admin_jurisdiction_pb';
import { getJurisdictionColumns } from './JurisdictionGate.columns';

const { Title } = Typography;

export default function JurisdictionGate() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('sanctions');
  const [kycFilter, setKycFilter] = useState('');
  const [kycPage, setKycPage] = useState(1);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [kycModalOpen, setKycModalOpen] = useState(false);
  const [selectedUser, setSelectedUser] = useState<UserKYCItem | null>(null);
  const [form] = Form.useForm();
  const [kycForm] = Form.useForm();

  const {
    data: countries = [],
    isLoading: countriesLoading,
    error: countriesError,
    refetch: refetchCountries,
  } = useRpcQuery(['admin', 'sanctioned-countries'], () => adminApi.listSanctionedCountries());

  const {
    data: kycData,
    isLoading: kycLoading,
    error: kycError,
    refetch: refetchKYC,
  } = useRpcQuery(['admin', 'kyc-users', kycFilter, kycPage], () =>
    adminApi.listUsersByKYCStatus({ kycStatus: kycFilter, page: kycPage, pageSize: 20 }),
  );

  const kycUsers = kycData?.users ?? [];
  const kycTotal = kycData?.total ?? 0;

  const handleAddCountry = async (values: { countryCode: string; label: string }) => {
    try {
      await adminApi.addSanctionedCountry(values.countryCode.toUpperCase(), values.label);
      showSuccess(t('admin.jurisdiction.messages.countryAdded'));
      setAddModalOpen(false);
      form.resetFields();
      refetchCountries();
    } catch {
      showError(t('admin.jurisdiction.messages.countryAddFailed'));
    }
  };

  const handleRemoveCountry = async (code: string) => {
    try {
      await adminApi.removeSanctionedCountry(code);
      showSuccess(t('admin.jurisdiction.messages.countryRemoved'));
      refetchCountries();
    } catch {
      showError(t('admin.jurisdiction.messages.countryRemoveFailed'));
    }
  };

  const handleSetKYC = async (values: { kycStatus: string }) => {
    if (!selectedUser) return;
    try {
      await adminApi.setKYCStatus(selectedUser.userId, values.kycStatus);
      showSuccess(t('admin.jurisdiction.messages.kycUpdated'));
      setKycModalOpen(false);
      refetchKYC();
    } catch {
      showError(t('admin.jurisdiction.messages.kycUpdateFailed'));
    }
  };

  const handleOverride = async (user: UserKYCItem) => {
    try {
      await adminApi.setSanctionedOverride(user.userId, !user.sanctionedOverride);
      showSuccess(t('admin.jurisdiction.messages.overrideUpdated'));
      refetchKYC();
    } catch {
      showError(t('admin.jurisdiction.messages.overrideUpdateFailed'));
    }
  };

  const { countryColumns, kycColumns } = getJurisdictionColumns({
    t,
    onRemoveCountry: handleRemoveCountry,
    onSetKYC: (row) => { setSelectedUser(row); kycForm.setFieldsValue({ kycStatus: row.kycStatus }); setKycModalOpen(true); },
    onOverride: handleOverride,
  });

  return (
    <div className="space-y-4">
      <div>
        <Title level={3}>
          <SafetyOutlined size={24} stroke={1.5} className="inline mr-2" />
          {t('admin.jurisdiction.title')}
        </Title>
      </div>

      <Tabs activeKey={activeTab} onChange={setActiveTab} items={[
        {
          key: 'sanctions',
          label: t('admin.jurisdiction.sanctionedCountriesTab'),
          children: (
            <Card
              title={t('admin.jurisdiction.sanctionedCountries')}
              extra={
                <Button type="primary" onClick={() => { form.resetFields(); setAddModalOpen(true); }}>
                  {t('admin.jurisdiction.addCountry')}
                </Button>
              }
            >
              <StatusResult
                loading={countriesLoading}
                error={countriesError instanceof Error ? countriesError.message : null}
                empty={!countriesLoading && !countriesError && countries.length === 0}
                emptyText={t('admin.jurisdiction.emptySanctions')}
                onRetry={refetchCountries}
              >
                <Table rowKey="countryCode" dataSource={countries as SanctionedCountry[]} columns={countryColumns} size="small" pagination={false} />
              </StatusResult>
            </Card>
          ),
        },
        {
          key: 'kyc',
          label: t('admin.jurisdiction.kycStatusTab'),
          children: (
            <Card
              title={t('admin.jurisdiction.userKYCStatus')}
              extra={
                <Select
                  allowClear
                  placeholder={t('admin.jurisdiction.filterByKYCStatus')}
                  style={{ width: 160 }}
                  value={kycFilter || undefined}
                  onChange={(v) => { setKycFilter(v ?? ''); setKycPage(1); }}
                  options={[
                    { value: 'unverified', label: t('admin.jurisdiction.unverified') },
                    { value: 'pending', label: t('admin.jurisdiction.pending') },
                    { value: 'verified', label: t('admin.jurisdiction.verified') },
                    { value: 'rejected', label: t('admin.jurisdiction.rejected') },
                  ]}
                />
              }
            >
              <StatusResult
                loading={kycLoading}
                error={kycError instanceof Error ? kycError.message : null}
                empty={!kycLoading && !kycError && kycUsers.length === 0}
                emptyText={t('admin.jurisdiction.emptyKYC')}
                onRetry={refetchKYC}
              >
                <Table rowKey="userId" dataSource={kycUsers as UserKYCItem[]} columns={kycColumns} size="small"
                  pagination={{ current: kycPage, pageSize: 20, total: kycTotal, onChange: setKycPage, showSizeChanger: false }} />
              </StatusResult>
            </Card>
          ),
        },
      ]} />

      <Modal title={t('admin.jurisdiction.addSanctionedCountry')} open={addModalOpen}
        onCancel={() => setAddModalOpen(false)} onOk={() => form.submit()}>
        <Form form={form} layout="vertical" onFinish={handleAddCountry}>
          <Form.Item name="countryCode" label={t('admin.jurisdiction.countryCode')} rules={[{ required: true, min: 2, max: 2 }]}>
            <Input placeholder="IR" maxLength={2} style={{ textTransform: 'uppercase' }} />
          </Form.Item>
          <Form.Item name="label" label={t('admin.jurisdiction.countryLabel')} rules={[{ required: true }]}>
            <Input placeholder="Iran" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={t('admin.jurisdiction.setKYCStatus')} open={kycModalOpen}
        onCancel={() => setKycModalOpen(false)} onOk={() => kycForm.submit()}>
        <Form form={kycForm} layout="vertical" onFinish={handleSetKYC}>
          <Form.Item name="kycStatus" label={t('admin.jurisdiction.kycStatus')} rules={[{ required: true }]}>
            <Select options={[
              { value: 'unverified', label: t('admin.jurisdiction.unverified') },
              { value: 'pending', label: t('admin.jurisdiction.pending') },
              { value: 'verified', label: t('admin.jurisdiction.verified') },
              { value: 'rejected', label: t('admin.jurisdiction.rejected') },
            ]} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
