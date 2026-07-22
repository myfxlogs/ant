import { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Button, Table, Tag, Modal, Form, Input, message, Space, Popconfirm } from 'antd';
import { PlusOutlined, DeleteOutlined, SafetyOutlined } from '@ant-design/icons';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { webauthnApi } from '@/client/webauthn';

export function WhitelistManagement() {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [addOpen, setAddOpen] = useState(false);
  const [form] = Form.useForm();

  const { data: whitelist, isLoading } = useQuery({
    queryKey: ['webauthn', 'whitelist'],
    queryFn: () => webauthnApi.listWhitelistAddresses(),
  });

  const addMutation = useMutation({
    mutationFn: (values: { address: string; label: string }) =>
      webauthnApi.addWhitelistAddress(values.address, values.label),
    onSuccess: () => {
      message.success(t('wallet.whitelist.added', { defaultValue: 'Whitelist address added (pending confirmation)' }));
      setAddOpen(false);
      form.resetFields();
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'whitelist'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const removeMutation = useMutation({
    mutationFn: (id: string) => webauthnApi.removeWhitelistAddress(id),
    onSuccess: () => {
      message.success(t('wallet.whitelist.removed', { defaultValue: 'Whitelist address removed' }));
      queryClient.invalidateQueries({ queryKey: ['webauthn', 'whitelist'] });
    },
    onError: (err: Error) => message.error(err.message),
  });

  const statusColors: Record<string, string> = {
    PENDING_CONFIRMATION: 'orange',
    ACTIVE: 'green',
    REMOVED: 'default',
  };

  const columns = [
    {
      title: t('wallet.whitelist.label', { defaultValue: 'Label' }),
      dataIndex: 'label',
      key: 'label',
      width: 120,
    },
    {
      title: t('wallet.whitelist.address', { defaultValue: 'Address' }),
      dataIndex: 'address',
      key: 'address',
      ellipsis: true,
      render: (v: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{v}</span>,
    },
    {
      title: t('wallet.whitelist.status', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      width: 160,
      render: (v: string) => <Tag color={statusColors[v] || 'default'}>{v}</Tag>,
    },
    {
      title: t('wallet.whitelist.confirmedAt', { defaultValue: 'Confirmed' }),
      dataIndex: 'confirmedAtTsMs',
      key: 'confirmedAtTsMs',
      width: 180,
      render: (v: string) => v ? new Date(Number(v)).toLocaleString() : '-',
    },
    {
      title: '',
      key: 'action',
      width: 80,
      render: (_: any, record: any) =>
        record.status !== 'REMOVED' && (
          <Popconfirm
            title={t('wallet.whitelist.confirmRemove', { defaultValue: 'Remove this address?' })}
            onConfirm={() => removeMutation.mutate(record.id)}
          >
            <Button type="text" danger icon={<DeleteOutlined />} size="small" />
          </Popconfirm>
        ),
    },
  ];

  return (
    <Card
      size="small"
      style={{ marginBottom: 24 }}
      title={<span><SafetyOutlined style={{ marginRight: 8, color: '#D4AF37' }} />{t('wallet.whitelist.title', { defaultValue: 'Withdrawal Whitelist' })}</span>}
      extra={<Button type="primary" size="small" icon={<PlusOutlined />} onClick={() => setAddOpen(true)}>{t('wallet.whitelist.add', { defaultValue: 'Add Address' })}</Button>}
    >
      <Table
        columns={columns}
        dataSource={whitelist || []}
        rowKey="id"
        loading={isLoading}
        size="small"
        pagination={false}
      />
      <Modal
        title={t('wallet.whitelist.add', { defaultValue: 'Add Whitelist Address' })}
        open={addOpen}
        onCancel={() => setAddOpen(false)}
        onOk={() => form.validateFields().then(v => addMutation.mutate(v))}
        confirmLoading={addMutation.isPending}
        okText={t('common.add', { defaultValue: 'Add' })}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="address"
            label={t('wallet.whitelist.addressLabel', { defaultValue: 'TRC20 Address' })}
            rules={[{ required: true, message: t('wallet.whitelist.addressRequired', { defaultValue: 'Please enter address' }) }]}
          >
            <Input placeholder="T..." style={{ fontFamily: 'monospace' }} />
          </Form.Item>
          <Form.Item name="label" label={t('wallet.whitelist.labelLabel', { defaultValue: 'Label (optional)' })}>
            <Input placeholder={t('wallet.whitelist.labelPlaceholder', { defaultValue: 'e.g. My Binance wallet' })} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
