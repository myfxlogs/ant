import { useState, useCallback } from 'react';
import { Table, Tag, Button, Space, Modal, Input, InputNumber, Select, message } from 'antd';
import { PlusOutlined, StopOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { CouponInfo } from '@/gen/ant/v1/marketplace_service_pb';

export default function CouponManagement() {
  const { t } = useTranslation();
  const [coupons, setCoupons] = useState<CouponInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [createForm, setCreateForm] = useState({
    code: '',
    discountType: 'percentage',
    discountValue: '',
    minPurchase: '0',
    maxUses: 0,
    expiresAt: '',
  });
  const [createLoading, setCreateLoading] = useState(false);

  const fetchCoupons = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.listCoupons({ enabledOnly: false });
      setCoupons(resp.coupons || []);
    } catch {
      message.error(t('admin.coupon.loadFailed', { defaultValue: 'Failed to load coupons' }));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleCreate = async () => {
    if (!createForm.code.trim() || !createForm.discountValue.trim()) {
      message.warning(t('admin.coupon.fillRequired', { defaultValue: 'Please fill required fields' }));
      return;
    }
    setCreateLoading(true);
    try {
      const resp = await marketplaceClient.createCoupon({
        code: createForm.code.trim().toUpperCase(),
        discountType: createForm.discountType,
        discountValue: createForm.discountValue,
        minPurchaseAmount: createForm.minPurchase,
        maxUses: createForm.maxUses,
        expiresAt: createForm.expiresAt || undefined,
      });
      if (resp.success) {
        message.success(t('admin.coupon.created', { defaultValue: 'Coupon created' }));
        setCreateOpen(false);
        fetchCoupons();
      } else {
        message.error(resp.error || 'Failed to create coupon');
      }
    } catch {
      message.error(t('admin.coupon.createFailed', { defaultValue: 'Failed to create coupon' }));
    } finally {
      setCreateLoading(false);
    }
  };

  const handleDisable = async (couponId: string) => {
    try {
      await marketplaceClient.disableCoupon({ couponId });
      message.success(t('admin.coupon.disabled', { defaultValue: 'Coupon disabled' }));
      fetchCoupons();
    } catch {
      message.error(t('admin.coupon.disableFailed', { defaultValue: 'Failed to disable coupon' }));
    }
  };

  const columns = [
    { title: t('admin.coupon.colCode', { defaultValue: 'Code' }), dataIndex: 'code', key: 'code' },
    {
      title: t('admin.coupon.colType', { defaultValue: 'Type' }),
      dataIndex: 'discountType', key: 'discountType',
      render: (v: string) => <Tag>{v}</Tag>,
    },
    {
      title: t('admin.coupon.colValue', { defaultValue: 'Value' }),
      dataIndex: 'discountValue', key: 'discountValue',
      render: (v: string, r: CouponInfo) => r.discountType === 'percentage' ? `${v}%` : v,
    },
    { title: t('admin.coupon.colMinPurchase', { defaultValue: 'Min Purchase' }), dataIndex: 'minPurchaseAmount', key: 'minPurchase' },
    {
      title: t('admin.coupon.colUsage', { defaultValue: 'Usage' }),
      key: 'usage',
      render: (_: unknown, r: CouponInfo) => `${r.usedCount}/${r.maxUses || '∞'}`,
    },
    {
      title: t('admin.coupon.colExpires', { defaultValue: 'Expires' }),
      dataIndex: 'expiresAt', key: 'expiresAt',
      render: (v: string) => v || 'Never',
    },
    {
      title: t('admin.coupon.colStatus', { defaultValue: 'Status' }),
      dataIndex: 'enabled', key: 'enabled',
      render: (v: boolean) => <Tag color={v ? 'green' : 'red'}>{v ? 'Active' : 'Disabled'}</Tag>,
    },
    {
      title: t('admin.coupon.colActions', { defaultValue: 'Actions' }),
      key: 'actions',
      render: (_: unknown, r: CouponInfo) =>
        r.enabled && (
          <Button size="small" danger icon={<StopOutlined />} onClick={() => handleDisable(r.id)}>
            {t('admin.coupon.disable', { defaultValue: 'Disable' })}
          </Button>
        ),
    },
  ];

  return (
    <div>
      <Button type="primary" icon={<PlusOutlined />} style={{ marginBottom: 16 }} onClick={() => setCreateOpen(true)}>
        {t('admin.coupon.create', { defaultValue: 'Create Coupon' })}
      </Button>

      <Table
        dataSource={coupons}
        columns={columns}
        rowKey="id"
        loading={loading}
        pagination={{ pageSize: 20 }}
        size="small"
      />

      <Modal
        title={t('admin.coupon.createTitle', { defaultValue: 'Create Coupon' })}
        open={createOpen}
        onCancel={() => setCreateOpen(false)}
        onOk={handleCreate}
        confirmLoading={createLoading}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input
            placeholder={t('admin.coupon.codePlaceholder', { defaultValue: 'Coupon code (e.g. SUMMER20)' })}
            value={createForm.code}
            onChange={e => setCreateForm({ ...createForm, code: e.target.value })}
          />
          <Select
            value={createForm.discountType}
            onChange={v => setCreateForm({ ...createForm, discountType: v })}
            style={{ width: '100%' }}
            options={[
              { label: 'Percentage (%)', value: 'percentage' },
              { label: 'Fixed Amount', value: 'fixed' },
            ]}
          />
          <Input
            placeholder={t('admin.coupon.valuePlaceholder', { defaultValue: 'Discount value (e.g. 20 for 20% or 50 for ¥50)' })}
            value={createForm.discountValue}
            onChange={e => setCreateForm({ ...createForm, discountValue: e.target.value })}
          />
          <Input
            placeholder={t('admin.coupon.minPurchasePlaceholder', { defaultValue: 'Minimum purchase amount (0 = none)' })}
            value={createForm.minPurchase}
            onChange={e => setCreateForm({ ...createForm, minPurchase: e.target.value })}
          />
          <InputNumber
            placeholder={t('admin.coupon.maxUsesPlaceholder', { defaultValue: 'Max uses (0 = unlimited)' })}
            value={createForm.maxUses}
            onChange={v => setCreateForm({ ...createForm, maxUses: v || 0 })}
            min={0}
            style={{ width: '100%' }}
          />
          <Input
            placeholder={t('admin.coupon.expiresPlaceholder', { defaultValue: 'Expires at (ISO 8601, empty = never)' })}
            value={createForm.expiresAt}
            onChange={e => setCreateForm({ ...createForm, expiresAt: e.target.value })}
          />
        </Space>
      </Modal>
    </div>
  );
}
