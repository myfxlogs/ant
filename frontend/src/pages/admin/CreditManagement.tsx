import { useState, useCallback, useEffect } from 'react';
import { Card, Table, Tag, Button, Modal, Input, Form, message, Statistic } from 'antd';
import { creditApi } from '@/client/credit';
import type { CreditTransaction } from '@/gen/ant/v1/credit_pb';

interface CreditError {
  message?: string;
}

export default function CreditManagement() {
  const [transactions, setTransactions] = useState<CreditTransaction[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [refundModalOpen, setRefundModalOpen] = useState(false);
  const [addForm] = Form.useForm();
  const [refundForm] = Form.useForm();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await creditApi.listAllTransactions(page, 20);
      setTransactions(resp.transactions);
      setTotal(Number(resp.total));
    } catch {
      // fail silently
    } finally {
      setLoading(false);
    }
  }, [page]);

  useEffect(() => { load(); }, [load]);

  const handleAdd = async () => {
    const values = await addForm.validateFields();
    try {
      const newBal = await creditApi.addCredits(values.userId, values.amount, values.description || 'Admin top-up');
      message.success(`Credits added. New balance: ${newBal}`);
      setAddModalOpen(false);
      addForm.resetFields();
      load();
    } catch (e: unknown) {
      const err = e as CreditError;
      message.error(err?.message || 'Failed to add credits');
    }
  };

  const handleRefund = async () => {
    const values = await refundForm.validateFields();
    try {
      const newBal = await creditApi.refundCredits(values.userId, values.amount, values.description || 'Admin refund');
      message.success(`Credits refunded. New balance: ${newBal}`);
      setRefundModalOpen(false);
      refundForm.resetFields();
      load();
    } catch (e: unknown) {
      const err = e as CreditError;
      message.error(err?.message || 'Refund failed — check user has sufficient balance');
    }
  };

  const columns = [
    { title: 'User ID', dataIndex: 'userId', ellipsis: true },
    {
      title: 'Type',
      dataIndex: 'txType',
      render: (type: string) => <Tag>{type}</Tag>,
    },
    { title: 'Amount', dataIndex: 'amount' },
    { title: 'Balance After', dataIndex: 'balanceAfter' },
    { title: 'Description', dataIndex: 'description', ellipsis: true },
    {
      title: 'Time',
      dataIndex: 'createdAtTsMs',
      render: (ms: bigint) => new Date(Number(ms)).toLocaleString(),
    },
  ];

  return (
    <div className="p-6 space-y-4">
      <div className="flex gap-4">
        <Card size="small">
          <Statistic title="Total Transactions" value={total} />
        </Card>
        <div className="flex gap-2 items-end">
          <Button type="primary" onClick={() => setAddModalOpen(true)}>
            Add Credits
          </Button>
          <Button onClick={() => setRefundModalOpen(true)}>
            Refund Credits
          </Button>
        </div>
      </div>

      <Card title="Credit Transactions" size="small">
        <Table
          columns={columns}
          dataSource={transactions}
          rowKey={(r) => r.id}
          loading={loading}
          size="small"
          pagination={{
            current: page,
            total,
            pageSize: 20,
            onChange: setPage,
            showSizeChanger: false,
          }}
        />
      </Card>

      <Card title="Refund Policy" size="small">
        <ul className="text-sm text-gray-600 list-disc pl-6">
          <li>Unused credited credits can be refunded within 7 days (original payment method).</li>
          <li>Already consumed credits are non-refundable.</li>
          <li>Free/granted credits are non-refundable.</li>
        </ul>
      </Card>

      <Modal
        title="Add Credits"
        open={addModalOpen}
        onOk={handleAdd}
        onCancel={() => setAddModalOpen(false)}
      >
        <Form form={addForm} layout="vertical">
          <Form.Item name="userId" label="User ID" rules={[{ required: true }]}>
            <Input placeholder="User UUID" />
          </Form.Item>
          <Form.Item name="amount" label="Amount (credits)" rules={[{ required: true }]}>
            <Input type="number" placeholder="e.g. 500" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Admin top-up" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title="Refund Credits"
        open={refundModalOpen}
        onOk={handleRefund}
        onCancel={() => setRefundModalOpen(false)}
      >
        <Form form={refundForm} layout="vertical">
          <Form.Item name="userId" label="User ID" rules={[{ required: true }]}>
            <Input placeholder="User UUID" />
          </Form.Item>
          <Form.Item name="amount" label="Amount (credits)" rules={[{ required: true }]}>
            <Input type="number" placeholder="e.g. 200" />
          </Form.Item>
          <Form.Item name="description" label="Description">
            <Input placeholder="Admin refund" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
