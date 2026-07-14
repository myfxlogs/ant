import { useTranslation } from 'react-i18next';
import { Card, Table, Tag, Typography, Button, Modal, Input, Select, Space, message } from 'antd';
import { CheckOutlined, CloseOutlined, ReloadOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { depositApi } from '@/client/deposit';
import { queryKeys } from '@/queries/queryKeys';
import { formatAmount } from '@/utils/amount';
import { useMemo, useState } from 'react';

const { Title } = Typography;

export default function DepositManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [reviewModal, setReviewModal] = useState<{ depositId: string; action: 'approve' | 'reject' } | null>(null);
  const [reviewNote, setReviewNote] = useState('');

  const { data, isLoading, refetch } = useQuery({
    queryKey: [...queryKeys.deposit.all, page, statusFilter],
    queryFn: () => depositApi.listDeposits({ page, pageSize: 20, status: statusFilter }),
  });

  const approveMutation = useMutation({
    mutationFn: (depositId: string) => depositApi.approveDeposit(depositId, reviewNote),
    onSuccess: () => {
      message.success(t('admin.deposit.approved', { defaultValue: 'Deposit approved and wallet credited.' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.deposit.all });
      setReviewModal(null);
      setReviewNote('');
    },
    onError: (err: Error) => message.error(err.message || t('admin.deposit.approveFailed', { defaultValue: 'Failed to approve deposit.' })),
  });

  const rejectMutation = useMutation({
    mutationFn: (depositId: string) => depositApi.rejectDeposit(depositId, reviewNote),
    onSuccess: () => {
      message.success(t('admin.deposit.rejected', { defaultValue: 'Deposit rejected.' }));
      queryClient.invalidateQueries({ queryKey: queryKeys.deposit.all });
      setReviewModal(null);
      setReviewNote('');
    },
    onError: (err: Error) => message.error(err.message || t('admin.deposit.rejectFailed', { defaultValue: 'Failed to reject deposit.' })),
  });

  const columns = useMemo(() => [
    {
      title: t('admin.deposit.table.user', { defaultValue: 'User' }),
      dataIndex: 'userEmail',
      key: 'userEmail',
      width: 200,
      ellipsis: true,
    },
    {
      title: t('admin.deposit.table.amount', { defaultValue: 'USDT Amount' }),
      dataIndex: 'amount',
      key: 'amount',
      width: 120,
    },
    {
      title: t('admin.deposit.table.amountUsd', { defaultValue: 'USD Credit' }),
      dataIndex: 'amountUsd',
      key: 'amountUsd',
      width: 120,
      render: (v: string) => <span style={{ color: '#00A651', fontWeight: 500 }}>+{formatAmount(v)}</span>,
    },
    {
      title: t('admin.deposit.table.txHash', { defaultValue: 'Tx Hash' }),
      dataIndex: 'txHash',
      key: 'txHash',
      width: 180,
      ellipsis: true,
      render: (v: string) => v ? <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v.slice(0, 20)}...</span> : '-',
    },
    {
      title: t('admin.deposit.table.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (v: string) => {
        const colors: Record<string, string> = { PENDING: 'orange', APPROVED: 'green', REJECTED: 'red' };
        return <Tag color={colors[v] || 'default'}>{v}</Tag>;
      },
    },
    {
      title: t('admin.deposit.table.reviewNote', { defaultValue: 'Review Note' }),
      dataIndex: 'reviewNote',
      key: 'reviewNote',
      ellipsis: true,
      width: 200,
    },
    {
      title: t('admin.deposit.table.time', { defaultValue: 'Time' }),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 180,
      render: (v: any) => v ? new Date(v.seconds * 1000).toLocaleString() : '-',
    },
    {
      title: t('admin.deposit.table.action', { defaultValue: 'Action' }),
      key: 'action',
      width: 160,
      render: (_: any, record: any) => {
        if (record.status !== 'PENDING') return null;
        return (
          <Space>
            <Button
              size="small"
              type="primary"
              icon={<CheckOutlined />}
              onClick={() => { setReviewModal({ depositId: record.id, action: 'approve' }); setReviewNote(''); }}
            >
              {t('admin.deposit.approve', { defaultValue: 'Approve' })}
            </Button>
            <Button
              size="small"
              danger
              icon={<CloseOutlined />}
              onClick={() => { setReviewModal({ depositId: record.id, action: 'reject' }); setReviewNote(''); }}
            >
              {t('admin.deposit.reject', { defaultValue: 'Reject' })}
            </Button>
          </Space>
        );
      },
    },
  ], [t]);

  return (
    <div>
      <Title level={4} style={{ margin: '0 0 16px 0', fontFamily: 'Poppins, sans-serif' }}>
        {t('admin.deposit.title', { defaultValue: 'Deposit Management' })}
      </Title>

      <Card size="small">
        <Space style={{ marginBottom: 16 }}>
          <Select
            value={statusFilter}
            onChange={(v) => { setStatusFilter(v); setPage(1); }}
            style={{ width: 150 }}
            options={[
              { value: '', label: t('admin.deposit.allStatuses', { defaultValue: 'All Statuses' }) },
              { value: 'PENDING', label: 'Pending' },
              { value: 'APPROVED', label: 'Approved' },
              { value: 'REJECTED', label: 'Rejected' },
            ]}
          />
          <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
            {t('common.refresh', { defaultValue: 'Refresh' })}
          </Button>
        </Space>

        <Table
          columns={columns}
          dataSource={data?.deposits || []}
          rowKey="id"
          loading={isLoading}
          pagination={{
            current: page,
            pageSize: 20,
            total: data?.total || 0,
            onChange: setPage,
            size: 'small',
          }}
          size="small"
        />
      </Card>

      {/* Review Modal */}
      <Modal
        title={reviewModal?.action === 'approve'
          ? t('admin.deposit.approveTitle', { defaultValue: 'Approve Deposit' })
          : t('admin.deposit.rejectTitle', { defaultValue: 'Reject Deposit' })}
        open={!!reviewModal}
        onCancel={() => setReviewModal(null)}
        onOk={() => {
          if (reviewModal?.action === 'approve') {
            approveMutation.mutate(reviewModal.depositId);
          } else if (reviewModal?.action === 'reject') {
            rejectMutation.mutate(reviewModal.depositId);
          }
        }}
        confirmLoading={approveMutation.isPending || rejectMutation.isPending}
        okText={reviewModal?.action === 'approve'
          ? t('admin.deposit.approve', { defaultValue: 'Approve' })
          : t('admin.deposit.reject', { defaultValue: 'Reject' })}
        okButtonProps={{ danger: reviewModal?.action === 'reject' }}
      >
        <div style={{ marginBottom: 12 }}>
          <label style={{ display: 'block', marginBottom: 4, fontWeight: 500 }}>
            {t('admin.deposit.reviewNoteLabel', { defaultValue: 'Review Note (optional)' })}
          </label>
          <Input.TextArea
            value={reviewNote}
            onChange={(e) => setReviewNote(e.target.value)}
            rows={3}
            placeholder={t('admin.deposit.reviewNotePlaceholder', { defaultValue: 'Add a note for this review...' })}
          />
        </div>
        {reviewModal?.action === 'approve' && (
          <Tag color="green">
            {t('admin.deposit.approveWarning', { defaultValue: 'Approving will credit the user wallet immediately.' })}
          </Tag>
        )}
      </Modal>
    </div>
  );
}
