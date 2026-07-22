import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Button, Space, message, Statistic, Row, Col, Modal, Upload, Alert, Checkbox } from 'antd';
import { ReloadOutlined, DownloadOutlined, UploadOutlined, ThunderboltOutlined, KeyOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { depositApi } from '@/client/deposit';
import { formatAmount } from '@/utils/amount';
import { downloadBlob } from '@/utils/download';
import type { SweepDashboardEntry, PendingSignBundleEntry } from '@/client/deposit';

const { Title } = Typography;

export default function SweepManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [importOpen, setImportOpen] = useState(false);

  const { data: dashboard, isLoading } = useQuery({
    queryKey: ['admin', 'sweep', 'dashboard', page],
    queryFn: () => depositApi.getSweepDashboard(page, 20),
  });

  const { data: pendingBundles } = useQuery({
    queryKey: ['admin', 'sweep', 'pendingBundles'],
    queryFn: () => depositApi.listPendingSignBundles(),
  });

  const exportMutation = useMutation({
    mutationFn: async (addrId: string) => {
      const bundle = await depositApi.exportUnsignedSweepBundle(addrId);
      downloadBlob(new Blob([bundle]), `unsigned-bundle-${addrId.slice(0, 8)}.bin`);
    },
    onSuccess: () => message.success(t('admin.sweep.exportSuccess', { defaultValue: 'Unsigned bundle exported. Transfer to cold signing machine via USB.' })),
    onError: (err: Error) => message.error(err.message),
  });

  const exportBatchMutation = useMutation({
    mutationFn: async (addrIds: string[]) => {
      const bundle = await depositApi.exportBatchUnsignedSweepBundle(addrIds);
      downloadBlob(new Blob([bundle]), `unsigned-batch-${Date.now()}.bin`);
    },
    onSuccess: () => {
      message.success(t('admin.sweep.batchExportSuccess', { defaultValue: 'Batch unsigned bundle exported.' }));
      setSelectedIds([]);
    },
    onError: (err: Error) => message.error(err.message),
  });

  const importMutation = useMutation({
    mutationFn: async (file: File) => {
      const buf = new Uint8Array(await file.arrayBuffer());
      return depositApi.importSignedSweepBundle(buf);
    },
    onSuccess: (data) => {
      message.success(t('admin.sweep.importSuccess', { defaultValue: 'Signed bundle imported and broadcast.' }) + ` batch=${data.batchId}`);
      setImportOpen(false);
      queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const undelegateMutation = useMutation({
    mutationFn: async (addrIds: string[]) => depositApi.buildUndelegateOnlyBundle(addrIds),
    onSuccess: (bundle) => {
      downloadBlob(new Blob([bundle]), `undelegate-${Date.now()}.bin`);
      message.success(t('admin.sweep.undelegateSuccess', { defaultValue: 'Undelegate-only bundle exported.' }));
    },
    onError: (err: Error) => message.error(err.message),
  });

  const importXpubMutation = useMutation({
    mutationFn: async (file: File) => {
      const buf = new Uint8Array(await file.arrayBuffer());
      return depositApi.importXpub(buf);
    },
    onSuccess: (data) => {
      const fpStatus = data.fingerprintVerified
        ? t('admin.sweep.xpubFpVerified', { defaultValue: 'fingerprint verified' })
        : t('admin.sweep.xpubFpNotSet', { defaultValue: 'fingerprint not set in env (verification skipped)' });
      message.success(t('admin.sweep.xpubImported', { defaultValue: 'XPUB imported and hot-reloaded' }) + ` (${fpStatus})`);
      queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const columns = [
    {
      title: '',
      key: 'select',
      width: 40,
      render: (_: any, record: SweepDashboardEntry) => (
        <Checkbox
          checked={selectedIds.includes(record.depositAddressId)}
          onChange={(e) => {
            if (e.target.checked) setSelectedIds([...selectedIds, record.depositAddressId]);
            else setSelectedIds(selectedIds.filter(id => id !== record.depositAddressId));
          }}
        />
      ),
    },
    {
      title: t('admin.sweep.address', { defaultValue: 'Address' }),
      dataIndex: 'address',
      key: 'address',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
    },
    {
      title: t('admin.sweep.unswept', { defaultValue: 'Unswept USDT' }),
      dataIndex: 'unsweptAmount',
      key: 'unsweptAmount',
      width: 140,
      sorter: (a: SweepDashboardEntry, b: SweepDashboardEntry) => parseFloat(a.unsweptAmount) - parseFloat(b.unsweptAmount),
      render: (v: string) => <span style={{ fontWeight: 600, color: parseFloat(v) > 0 ? '#D4AF37' : 'var(--color-text)' }}>{formatAmount(v)}</span>,
    },
    {
      title: t('admin.sweep.aboveThreshold', { defaultValue: 'Above Threshold' }),
      dataIndex: 'aboveThreshold',
      key: 'aboveThreshold',
      width: 120,
      render: (v: boolean) => v ? <Tag color="red">YES</Tag> : <Tag>NO</Tag>,
    },
    {
      title: t('admin.sweep.sweepStatus', { defaultValue: 'Sweep Status' }),
      dataIndex: 'sweepStatus',
      key: 'sweepStatus',
      width: 120,
      render: (v: string) => {
        const colors: Record<string, string> = { PENDING: 'orange', SWEEPING: 'blue', DONE: 'green', MANUAL_REVIEW: 'red' };
        return <Tag color={colors[v] || 'default'}>{v || 'none'}</Tag>;
      },
    },
    {
      title: t('admin.sweep.derivationIndex', { defaultValue: 'Index' }),
      dataIndex: 'derivationIndex',
      key: 'derivationIndex',
      width: 80,
    },
    {
      title: '',
      key: 'action',
      width: 100,
      render: (_: any, record: SweepDashboardEntry) => (
        <Button
          size="small"
          icon={<DownloadOutlined />}
          loading={exportMutation.isPending && exportMutation.variables === record.depositAddressId}
          onClick={() => exportMutation.mutate(record.depositAddressId)}
        >
          {t('admin.sweep.export', { defaultValue: 'Export' })}
        </Button>
      ),
    },
  ];

  const bundleColumns = [
    {
      title: t('admin.sweep.bundleId', { defaultValue: 'Batch ID' }),
      dataIndex: 'batchId',
      key: 'batchId',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 16)}...</span>,
    },
    {
      title: t('admin.sweep.addressId', { defaultValue: 'Address ID' }),
      dataIndex: 'depositAddressId',
      key: 'depositAddressId',
      ellipsis: true,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 16)}...</span> : <Tag>BATCH</Tag>,
    },
    {
      title: t('admin.sweep.builtAt', { defaultValue: 'Built At' }),
      dataIndex: 'builtAtMs',
      key: 'builtAtMs',
      width: 180,
      render: (v: string) => v ? new Date(Number(v)).toLocaleString() : '-',
    },
    {
      title: t('admin.sweep.bundleStatus', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
  ];

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        <ThunderboltOutlined style={{ marginRight: 8, color: '#D4AF37' }} />
        {t('admin.sweep.title', { defaultValue: 'Sweep Management' })}
      </Title>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.totalUnswept', { defaultValue: 'Total Unswept' })}
              value={formatAmount(dashboard?.totalUnswept || '0')}
              suffix="USDT"
              valueStyle={{ color: '#D4AF37' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.threshold', { defaultValue: 'Sweep Threshold' })}
              value={formatAmount(dashboard?.threshold || '0')}
              suffix="USDT"
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.pendingBundles', { defaultValue: 'Pending Sign Bundles' })}
              value={pendingBundles?.length || 0}
            />
          </Card>
        </Col>
      </Row>

      <Card
        size="small"
        style={{ marginBottom: 24 }}
        title={<span><KeyOutlined style={{ marginRight: 8, color: '#D4AF37' }} />{t('admin.sweep.xpubTitle', { defaultValue: 'HD Wallet XPUB' })}</span>}
        extra={
          <Upload
            accept=".bin"
            maxCount={1}
            showUploadList={false}
            beforeUpload={(file) => {
              importXpubMutation.mutate(file);
              return false;
            }}
          >
            <Button icon={<UploadOutlined />} loading={importXpubMutation.isPending}>
              {t('admin.sweep.uploadXpub', { defaultValue: 'Upload xpub-export.bin' })}
            </Button>
          </Upload>
        }
      >
        <Alert
          type="info"
          message={t('admin.sweep.xpubHint', { defaultValue: 'Upload the xpub-export.bin file generated by hdgen on the air-gapped machine. The xpub will be stored in system_config and hot-reloaded — no server restart required.' })}
          showIcon
        />
      </Card>

      <Card
        size="small"
        style={{ marginBottom: 24 }}
        title={t('admin.sweep.dashboard', { defaultValue: 'Sweep Dashboard' })}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] })}>
              {t('common.refresh', { defaultValue: 'Refresh' })}
            </Button>
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              disabled={selectedIds.length === 0}
              loading={exportBatchMutation.isPending}
              onClick={() => exportBatchMutation.mutate(selectedIds)}
            >
              {t('admin.sweep.batchExport', { defaultValue: 'Batch Export' })} ({selectedIds.length})
            </Button>
            <Button
              icon={<ThunderboltOutlined />}
              disabled={selectedIds.length === 0}
              loading={undelegateMutation.isPending}
              onClick={() => undelegateMutation.mutate(selectedIds)}
            >
              {t('admin.sweep.undelegate', { defaultValue: 'Undelegate Only' })}
            </Button>
            <Button icon={<UploadOutlined />} onClick={() => setImportOpen(true)}>
              {t('admin.sweep.import', { defaultValue: 'Import Signed' })}
            </Button>
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={dashboard?.addresses || []}
          rowKey="depositAddressId"
          loading={isLoading}
          size="small"
          pagination={{
            current: page,
            pageSize: 20,
            total: dashboard?.total || 0,
            onChange: setPage,
            size: 'small',
          }}
        />
      </Card>

      <Card
        size="small"
        title={t('admin.sweep.pendingSignBundles', { defaultValue: 'Pending Sign Bundles' })}
      >
        <Table
          columns={bundleColumns}
          dataSource={pendingBundles || []}
          rowKey="batchId"
          size="small"
          pagination={false}
        />
      </Card>

      <Modal
        title={t('admin.sweep.importTitle', { defaultValue: 'Import Signed Bundle' })}
        open={importOpen}
        onCancel={() => setImportOpen(false)}
        footer={null}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Alert
            type="info"
            message={t('admin.sweep.importHint', { defaultValue: 'Upload the signed bundle from the cold signing machine. It will be automatically broadcast.' })}
            showIcon
          />
          <Upload.Dragger
            accept=".bin"
            maxCount={1}
            beforeUpload={(file) => {
              importMutation.mutate(file);
              return false;
            }}
          >
            <p style={{ fontSize: 14 }}>{t('admin.sweep.uploadHint', { defaultValue: 'Click or drag signed bundle .bin file here' })}</p>
          </Upload.Dragger>
        </Space>
      </Modal>
    </div>
  );
}
