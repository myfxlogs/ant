import { useState, useMemo } from 'react';
import { Table, Tag, Select, Space, Statistic, Button, Upload, message, Alert } from 'antd';
import { ReloadOutlined, UploadOutlined } from '@ant-design/icons';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useTranslation } from 'react-i18next';
import { depositApi } from '@/client/deposit';

export default function DepositAddressesTab() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState<string>('');
  const [importing, setImporting] = useState(false);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['admin', 'deposit-addresses', page, statusFilter],
    queryFn: () => depositApi.listDepositAddresses({ page, pageSize: 20, status: statusFilter }),
  });

  const handleImport = async (file: File) => {
    setImporting(true);
    try {
      const buf = new Uint8Array(await file.arrayBuffer());
      const result = await depositApi.importDepositAddresses(buf);
      message.success(t('admin.depositAddresses.importSuccess', {
        imported: result.imported,
        skipped: result.skipped,
        defaultValue: `Imported ${result.imported} addresses${result.skipped > 0 ? `, skipped ${result.skipped} duplicates` : ''}`,
      }));
      queryClient.invalidateQueries({ queryKey: ['admin', 'deposit-addresses'] });
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : String(err) || t('admin.depositAddresses.importFailed', { defaultValue: 'Import failed' }));
    } finally {
      setImporting(false);
    }
    return false;
  };

  const columns = useMemo(() => [
    {
      title: t('admin.depositAddresses.address', { defaultValue: 'Address' }),
      dataIndex: 'address',
      key: 'address',
      width: 200,
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
    },
    {
      title: t('admin.depositAddresses.user', { defaultValue: 'User ID' }),
      dataIndex: 'userId',
      key: 'userId',
      width: 200,
      ellipsis: true,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 8)}...</span> : '-',
    },
    {
      title: t('admin.depositAddresses.index', { defaultValue: 'Index' }),
      dataIndex: 'derivationIndex',
      key: 'derivationIndex',
      width: 80,
    },
    {
      title: t('admin.depositAddresses.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => {
        const colors: Record<string, string> = { AVAILABLE: 'blue', ASSIGNED: 'green', RETIRED: 'default' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('admin.depositAddresses.received', { defaultValue: 'Received USDT' }),
      dataIndex: 'hasReceivedUsdt',
      key: 'hasReceivedUsdt',
      width: 120,
      render: (v: boolean) => v ? <Tag color="gold">Yes</Tag> : <Tag>No</Tag>,
    },
    {
      title: t('admin.depositAddresses.network', { defaultValue: 'Network' }),
      dataIndex: 'network',
      key: 'network',
      width: 100,
      render: (v: string) => <Tag>{v || 'TRC20'}</Tag>,
    },
    {
      title: t('admin.depositAddresses.assignedAt', { defaultValue: 'Assigned At' }),
      dataIndex: 'assignedAt',
      key: 'assignedAt',
      width: 180,
      render: (v: unknown) => v ? new Date(v.seconds * 1000).toLocaleString() : '-',
    },
  ], [t]);

  return (
    <div className="space-y-4">
      <Alert
        type="info"
        showIcon
        message={t('admin.depositAddresses.importHint', { defaultValue: 'Use hdgen tool on an offline machine to generate deposit_addresses.bin, then upload it here.' })}
        style={{ marginBottom: 16 }}
      />

      <Space style={{ marginBottom: 16 }}>
        <Select
          value={statusFilter}
          onChange={(v) => { setStatusFilter(v); setPage(1); }}
          style={{ width: 160 }}
          options={[
            { label: t('admin.depositAddresses.all', { defaultValue: 'All Status' }), value: '' },
            { label: 'Available', value: 'AVAILABLE' },
            { label: 'Assigned', value: 'ASSIGNED' },
            { label: 'Retired', value: 'RETIRED' },
          ]}
        />
        <Upload
          accept=".bin"
          showUploadList={false}
          beforeUpload={(file) => { handleImport(file); return false; }}
          disabled={importing}
        >
          <Button icon={<UploadOutlined />} loading={importing}>
            {t('admin.depositAddresses.import', { defaultValue: 'Import Addresses' })}
          </Button>
        </Upload>
        <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
          {t('common.refresh', { defaultValue: 'Refresh' })}
        </Button>
      </Space>

      <Space size="large" style={{ marginBottom: 16 }}>
        <Statistic
          title={t('admin.depositAddresses.availablePool', { defaultValue: 'Available in Pool' })}
          value={Number(data?.availableCount ?? 0)}
          valueStyle={{ color: '#1677ff' }}
        />
        <Statistic
          title={t('admin.depositAddresses.total', { defaultValue: 'Total Addresses' })}
          value={Number(data?.total ?? 0)}
        />
      </Space>

      <Table
        columns={columns}
        dataSource={data?.addresses || []}
        rowKey="id"
        loading={isLoading}
        pagination={{
          current: page,
          pageSize: 20,
          total: data?.total || 0,
          onChange: setPage,
          showTotal: (total) => t('admin.depositAddresses.totalItems', { total, defaultValue: `${total} addresses` }),
          size: 'small',
        }}
        size="small"
      />
    </div>
  );
}
