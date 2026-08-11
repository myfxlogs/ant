import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Table, Typography, Button, Space, message, Statistic, Row, Col, Modal, Upload, Alert } from 'antd';
import { ReloadOutlined, DownloadOutlined, UploadOutlined, ThunderboltOutlined, KeyOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { depositApi } from '@/client/deposit';
import { formatAmount } from '@/utils/amount';
import { downloadBlob } from '@/utils/download';
import { buildSweepColumns, buildBundleColumns } from './SweepManagementHelpers';

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
      downloadBlob(new Blob([bundle as BlobPart]), `unsigned-bundle-${addrId.slice(0, 8)}.bin`);
    },
    onSuccess: () => message.success(t('admin.sweep.exportSuccess')),
    onError: (err: Error) => message.error(err.message),
  });

  const exportBatchMutation = useMutation({
    mutationFn: async (addrIds: string[]) => {
      const bundle = await depositApi.exportBatchUnsignedSweepBundle(addrIds);
      downloadBlob(new Blob([bundle as BlobPart]), `unsigned-batch-${Date.now()}.bin`);
    },
    onSuccess: () => {
      message.success(t('admin.sweep.batchExportSuccess'));
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
      message.success(t('admin.sweep.importSuccess') + ` batch=${data.batchId}`);
      setImportOpen(false);
      queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const undelegateMutation = useMutation({
    mutationFn: async (addrIds: string[]) => depositApi.buildUndelegateOnlyBundle(addrIds),
    onSuccess: (bundle) => {
      downloadBlob(new Blob([bundle as BlobPart]), `undelegate-${Date.now()}.bin`);
      message.success(t('admin.sweep.undelegateSuccess'));
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
        ? t('admin.sweep.xpubFpVerified')
        : t('admin.sweep.xpubFpNotSet');
      message.success(t('admin.sweep.xpubImported') + ` (${fpStatus})`);
      queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const columns = buildSweepColumns(t, selectedIds, setSelectedIds, exportMutation);
  const bundleColumns = buildBundleColumns(t);

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        <ThunderboltOutlined style={{ marginRight: 8, color: '#D4AF37' }} />
        {t('admin.sweep.title')}
      </Title>

      <Row gutter={16} style={{ marginBottom: 24 }}>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.totalUnswept')}
              value={formatAmount(dashboard?.totalUnswept || '0')}
              suffix="USDT"
              valueStyle={{ color: '#D4AF37' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.threshold')}
              value={formatAmount(dashboard?.threshold || '0')}
              suffix="USDT"
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small">
            <Statistic
              title={t('admin.sweep.pendingBundles')}
              value={pendingBundles?.length || 0}
            />
          </Card>
        </Col>
      </Row>

      <Card
        size="small"
        style={{ marginBottom: 24 }}
        title={<span><KeyOutlined style={{ marginRight: 8, color: '#D4AF37' }} />{t('admin.sweep.xpubTitle')}</span>}
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
              {t('admin.sweep.uploadXpub')}
            </Button>
          </Upload>
        }
      >
        <Alert
          type="info"
          message={t('admin.sweep.xpubHint')}
          showIcon
        />
      </Card>

      <Card
        size="small"
        style={{ marginBottom: 24 }}
        title={t('admin.sweep.dashboard')}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => queryClient.invalidateQueries({ queryKey: ['admin', 'sweep'] })}>
              {t('common.refresh')}
            </Button>
            <Button
              type="primary"
              icon={<DownloadOutlined />}
              disabled={selectedIds.length === 0}
              loading={exportBatchMutation.isPending}
              onClick={() => exportBatchMutation.mutate(selectedIds)}
            >
              {t('admin.sweep.batchExport')} ({selectedIds.length})
            </Button>
            <Button
              icon={<ThunderboltOutlined />}
              disabled={selectedIds.length === 0}
              loading={undelegateMutation.isPending}
              onClick={() => undelegateMutation.mutate(selectedIds)}
            >
              {t('admin.sweep.undelegate')}
            </Button>
            <Button icon={<UploadOutlined />} onClick={() => setImportOpen(true)}>
              {t('admin.sweep.import')}
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
            total: Number(dashboard?.total || 0),
            onChange: setPage,
            size: 'small',
          }}
        />
      </Card>

      <Card
        size="small"
        title={t('admin.sweep.pendingSignBundles')}
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
        title={t('admin.sweep.importTitle')}
        open={importOpen}
        onCancel={() => setImportOpen(false)}
        footer={null}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Alert
            type="info"
            message={t('admin.sweep.importHint')}
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
            <p style={{ fontSize: 14 }}>{t('admin.sweep.uploadHint')}</p>
          </Upload.Dragger>
        </Space>
      </Modal>
    </div>
  );
}
