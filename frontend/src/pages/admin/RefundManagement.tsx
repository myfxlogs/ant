import { useState, useCallback } from 'react';
import { Table, Tag, Button, Space, Select, Modal, Input, message, Tooltip } from 'antd';
import { CheckOutlined, CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { RefundRequestItem } from '@/gen/ant/v1/marketplace_service_pb';

export default function RefundManagement() {
  const { t } = useTranslation();
  const [requests, setRequests] = useState<RefundRequestItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [reviewModalOpen, setReviewModalOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<RefundRequestItem | null>(null);
  const [approveAction, setApproveAction] = useState(true);
  const [reviewNote, setReviewNote] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  const fetchRequests = useCallback(async (p: number, status: string) => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.adminListRefundRequests({
        status: status || undefined,
        limit: 20,
        offset: (p - 1) * 20,
      });
      setRequests(resp.requests || []);
      setTotal(resp.total || 0);
    } catch {
      message.error(t('admin.refund.loadFailed', { defaultValue: 'Failed to load refund requests' }));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleProcess = async () => {
    if (!reviewTarget) return;
    setActionLoading(true);
    try {
      const resp = await marketplaceClient.adminProcessRefund({
        refundId: reviewTarget.refundId,
        approve: approveAction,
        reviewNote,
      });
      if (resp.success) {
        message.success(approveAction
          ? t('admin.refund.approved', { defaultValue: 'Refund approved and executed' })
          : t('admin.refund.rejected', { defaultValue: 'Refund request rejected' }));
        setReviewModalOpen(false);
        fetchRequests(page, statusFilter);
      } else {
        message.error(resp.error || 'Failed to process refund');
      }
    } catch {
      message.error(t('admin.refund.processFailed', { defaultValue: 'Failed to process refund' }));
    } finally {
      setActionLoading(false);
    }
  };

  const columns = [
    {
      title: t('admin.refund.colUser', { defaultValue: 'User' }),
      dataIndex: 'userName',
      key: 'userName',
      ellipsis: true,
    },
    {
      title: t('admin.refund.colStrategy', { defaultValue: 'Strategy' }),
      dataIndex: 'strategyTitle',
      key: 'strategyTitle',
      ellipsis: true,
    },
    {
      title: t('admin.refund.colAmount', { defaultValue: 'Amount' }),
      dataIndex: 'amount',
      key: 'amount',
    },
    {
      title: t('admin.refund.colReason', { defaultValue: 'Reason' }),
      dataIndex: 'reason',
      key: 'reason',
      ellipsis: true,
    },
    {
      title: t('admin.refund.colStatus', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
        const color = status === 'pending' ? 'orange' : status === 'approved' ? 'green' : 'red';
        return <Tag color={color}>{status}</Tag>;
      },
    },
    {
      title: t('admin.refund.colDate', { defaultValue: 'Date' }),
      dataIndex: 'createdAt',
      key: 'createdAt',
    },
    {
      title: t('admin.refund.colActions', { defaultValue: 'Actions' }),
      key: 'actions',
      render: (_: unknown, r: RefundRequestItem) =>
        r.status === 'pending' ? (
          <Space>
            <Tooltip title={t('admin.refund.approve', { defaultValue: 'Approve & Execute' })}>
              <Button
                size="small"
                type="primary"
                icon={<CheckOutlined />}
                onClick={() => {
                  setReviewTarget(r);
                  setApproveAction(true);
                  setReviewNote('');
                  setReviewModalOpen(true);
                }}
              />
            </Tooltip>
            <Tooltip title={t('admin.refund.reject', { defaultValue: 'Reject' })}>
              <Button
                size="small"
                danger
                icon={<CloseOutlined />}
                onClick={() => {
                  setReviewTarget(r);
                  setApproveAction(false);
                  setReviewNote('');
                  setReviewModalOpen(true);
                }}
              />
            </Tooltip>
          </Space>
        ) : (
          <span style={{ color: '#ccc' }}>{r.reviewNote || '—'}</span>
        ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder={t('admin.refund.filterStatus', { defaultValue: 'All statuses' })}
          allowClear
          style={{ width: 150 }}
          value={statusFilter || undefined}
          onChange={(v) => {
            setStatusFilter(v || '');
            setPage(1);
            fetchRequests(1, v || '');
          }}
          options={[
            { label: 'Pending', value: 'pending' },
            { label: 'Approved', value: 'approved' },
            { label: 'Rejected', value: 'rejected' },
          ]}
        />
      </Space>

      <Table
        dataSource={requests}
        columns={columns}
        rowKey="refundId"
        loading={loading}
        pagination={{
          current: page,
          total,
          pageSize: 20,
          onChange: (p) => {
            setPage(p);
            fetchRequests(p, statusFilter);
          },
        }}
        size="small"
      />

      <Modal
        title={approveAction
          ? t('admin.refund.approveTitle', { defaultValue: 'Approve Refund' })
          : t('admin.refund.rejectTitle', { defaultValue: 'Reject Refund' })}
        open={reviewModalOpen}
        onCancel={() => setReviewModalOpen(false)}
        confirmLoading={actionLoading}
        onOk={handleProcess}
        okType={approveAction ? 'primary' : 'danger'}
        okText={approveAction ? 'Approve' : 'Reject'}
      >
        <p style={{ marginBottom: 8 }}>
          <strong>{reviewTarget?.userName}</strong> — {reviewTarget?.strategyTitle} ({reviewTarget?.amount})
        </p>
        <p style={{ marginBottom: 8, color: '#666' }}>{reviewTarget?.reason}</p>
        <Input.TextArea
          rows={3}
          placeholder={t('admin.refund.reviewNotePlaceholder', { defaultValue: 'Review note (optional for reject, recommended for approve)...' })}
          value={reviewNote}
          onChange={e => setReviewNote(e.target.value)}
          maxLength={500}
          showCount
        />
      </Modal>
    </div>
  );
}
