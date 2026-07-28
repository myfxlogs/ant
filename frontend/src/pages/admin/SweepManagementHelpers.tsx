import React from 'react';
import { Tag, Button, Checkbox } from 'antd';
import { DownloadOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';
import { formatAmount } from '@/utils/amount';
import type { SweepDashboardEntry } from '@/client/deposit';

export function buildSweepColumns(
  t: TFunction,
  selectedIds: string[],
  setSelectedIds: (ids: string[]) => void,
  exportMutation: { isPending: boolean; variables?: string; mutate: (id: string) => void },
) {
  return [
    {
      title: '',
      key: 'select',
      width: 40,
      render: (_: unknown, record: SweepDashboardEntry) => (
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
      title: t('admin.sweep.address'),
      dataIndex: 'address',
      key: 'address',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
    },
    {
      title: t('admin.sweep.unswept'),
      dataIndex: 'unsweptAmount',
      key: 'unsweptAmount',
      width: 140,
      sorter: (a: SweepDashboardEntry, b: SweepDashboardEntry) => parseFloat(a.unsweptAmount) - parseFloat(b.unsweptAmount),
      render: (v: string) => <span style={{ fontWeight: 600, color: parseFloat(v) > 0 ? '#D4AF37' : 'var(--color-text)' }}>{formatAmount(v)}</span>,
    },
    {
      title: t('admin.sweep.aboveThreshold'),
      dataIndex: 'aboveThreshold',
      key: 'aboveThreshold',
      width: 120,
      render: (v: boolean) => v ? <Tag color="red">YES</Tag> : <Tag>NO</Tag>,
    },
    {
      title: t('admin.sweep.sweepStatus'),
      dataIndex: 'sweepStatus',
      key: 'sweepStatus',
      width: 120,
      render: (v: string) => {
        const colors: Record<string, string> = { PENDING: 'orange', SWEEPING: 'blue', DONE: 'green', MANUAL_REVIEW: 'red' };
        return <Tag color={colors[v] || 'default'}>{v || 'none'}</Tag>;
      },
    },
    {
      title: t('admin.sweep.derivationIndex'),
      dataIndex: 'derivationIndex',
      key: 'derivationIndex',
      width: 80,
    },
    {
      title: '',
      key: 'action',
      width: 100,
      render: (_: unknown, record: SweepDashboardEntry) => (
        <Button
          size="small"
          icon={<DownloadOutlined />}
          loading={exportMutation.isPending && exportMutation.variables === record.depositAddressId}
          onClick={() => exportMutation.mutate(record.depositAddressId)}
        >
          {t('admin.sweep.export')}
        </Button>
      ),
    },
  ];
}

export function buildBundleColumns(t: TFunction) {
  return [
    {
      title: t('admin.sweep.bundleId'),
      dataIndex: 'batchId',
      key: 'batchId',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 16)}...</span>,
    },
    {
      title: t('admin.sweep.addressId'),
      dataIndex: 'depositAddressId',
      key: 'depositAddressId',
      ellipsis: true,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v?.slice(0, 16)}...</span> : <Tag>BATCH</Tag>,
    },
    {
      title: t('admin.sweep.builtAt'),
      dataIndex: 'builtAtMs',
      key: 'builtAtMs',
      width: 180,
      render: (v: string) => v ? new Date(Number(v)).toLocaleString() : '-',
    },
    {
      title: t('admin.sweep.bundleStatus'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (v: string) => <Tag color="blue">{v}</Tag>,
    },
  ];
}
