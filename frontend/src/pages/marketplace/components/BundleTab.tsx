import { useState, useCallback, useEffect } from 'react';
import { Card, Table, Button, Modal, Form, Input, InputNumber, Select, Tag, Typography, Space, Popconfirm, message, Empty, Row, Col, Statistic } from 'antd';
import { PlusOutlined, DeleteOutlined, ShoppingOutlined, GiftOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import {
  CreateBundleRequestSchema,
  ListBundlesRequestSchema,
  PurchaseBundleRequestSchema,
  DeleteBundleRequestSchema,
} from '@/gen/ant/v1/marketplace_service_pb';
import type { BundleInfo } from '@/gen/ant/v1/marketplace_service_pb';
import { useAuthStore } from '@/stores/authStore';

const { Text } = Typography;

export default function BundleTab() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const [bundles, setBundles] = useState<BundleInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [purchasing, setPurchasing] = useState<string | null>(null);
  const [form] = Form.useForm();
  const [strategyIdsRaw, setStrategyIdsRaw] = useState('');

  const fetchBundles = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.listBundles(create(ListBundlesRequestSchema, { limit: 50, offset: 0 }));
      setBundles(resp.bundles || []);
      setTotal(resp.total || 0);
    } catch {
      setBundles([]);
      setTotal(0);
    }
    setLoading(false);
  }, []);

  useEffect(() => { fetchBundles(); }, [fetchBundles]);

  const handleCreate = useCallback(async () => {
    try {
      const vals = await form.validateFields();
      const ids = strategyIdsRaw.split(/[\n,]/).map(s => s.trim()).filter(Boolean);
      if (ids.length < 2) {
        message.error(t('marketplace.bundle.needTwoStrategies', '至少选择 2 个策略'));
        return;
      }
      setCreating(true);
      const resp = await marketplaceClient.createBundle(create(CreateBundleRequestSchema, {
        title: vals.title,
        description: vals.description || '',
        priceModel: vals.priceModel,
        priceAmount: String(vals.priceAmount || '0'),
        strategyIds: ids,
        platformFeeRate: vals.platformFeeRate || '0.10',
      }));
      message.success(t('marketplace.bundle.created', '捆绑包创建成功'));
      setCreateOpen(false);
      form.resetFields();
      setStrategyIdsRaw('');
      fetchBundles();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.bundle.createFailed', '创建失败'));
    } finally {
      setCreating(false);
    }
  }, [form, strategyIdsRaw, t, fetchBundles]);

  const handlePurchase = useCallback(async (bundleId: string) => {
    const idempotencyKey = `bundle_${bundleId}_${Date.now()}`;
    setPurchasing(bundleId);
    try {
      const resp = await marketplaceClient.purchaseBundle(create(PurchaseBundleRequestSchema, { bundleId, idempotencyKey }));
      message.success(t('marketplace.bundle.purchased', '购买成功！已订阅包内所有策略。'));
      fetchBundles();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.bundle.purchaseFailed', '购买失败'));
    } finally {
      setPurchasing(null);
    }
  }, [t, fetchBundles]);

  const handleDelete = useCallback(async (bundleId: string) => {
    try {
      await marketplaceClient.deleteBundle(create(DeleteBundleRequestSchema, { bundleId }));
      message.success(t('marketplace.bundle.deleted', '已删除'));
      fetchBundles();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.bundle.deleteFailed', '删除失败'));
    }
  }, [t, fetchBundles]);

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Text strong style={{ fontSize: 15 }}>{t('marketplace.bundle.title', '策略捆绑包')}</Text>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setCreateOpen(true)}>
          {t('marketplace.bundle.create', '创建捆绑包')}
        </Button>
      </div>

      {bundles.length === 0 && !loading ? (
        <Empty description={t('marketplace.bundle.empty', '暂无捆绑包')} />
      ) : (
        <Table<BundleInfo>
          rowKey="id"
          dataSource={bundles}
          loading={loading}
          pagination={{ pageSize: 10, total }}
          size="small"
          columns={[
            {
              title: t('marketplace.bundle.name', '名称'), dataIndex: 'title', key: 'title',
              render: (v: string, row: BundleInfo) => (
                <div>
                  <Text strong>{v}</Text>
                  <br />
                  <Text type="secondary" style={{ fontSize: 12 }}>{row.description}</Text>
                </div>
              ),
            },
            {
              title: t('marketplace.bundle.strategies', '包含策略'), key: 'items',
              render: (_: unknown, row: BundleInfo) => (
                <Space wrap>
                  {(row.items || []).map((item, i) => (
                    <Tag key={i}>{item.title || item.strategyId}</Tag>
                  ))}
                </Space>
              ),
            },
            {
              title: t('marketplace.detail.price', '价格'), key: 'price',
              render: (_: unknown, row: BundleInfo) => (
                <Tag color={row.priceModel === 'once' ? 'gold' : 'blue'}>
                  {row.priceModel === 'once'
                    ? `¥${row.priceAmount}`
                    : `¥${row.priceAmount}/${t('marketplace.bundle.month', '月')}`}
                </Tag>
              ),
              width: 120,
            },
            {
              title: t('marketplace.bundle.purchases', '购买数'), dataIndex: 'totalPurchases', key: 'purchases', width: 80,
            },
            {
              title: t('marketplace.bundle.actions', '操作'), key: 'actions', width: 160,
              render: (_: unknown, row: BundleInfo) => (
                <Space>
                  <Button
                    type="primary" size="small" icon={<ShoppingOutlined />}
                    loading={purchasing === row.id}
                    onClick={() => handlePurchase(row.id)}
                  >
                    {t('marketplace.bundle.buy', '购买')}
                  </Button>
                  {row.publisherId === user?.id && (
                    <Popconfirm
                      title={t('marketplace.bundle.confirmDelete', '确认删除此捆绑包？')}
                      onConfirm={() => handleDelete(row.id)}
                    >
                      <Button danger size="small" icon={<DeleteOutlined />} />
                    </Popconfirm>
                  )}
                </Space>
              ),
            },
          ]}
        />
      )}

      <Modal
        title={t('marketplace.bundle.createTitle', '创建策略捆绑包')}
        open={createOpen}
        onOk={handleCreate}
        onCancel={() => setCreateOpen(false)}
        confirmLoading={creating}
        okText={t('marketplace.bundle.create', '创建')}
        width={600}
      >
        <Form form={form} layout="vertical" initialValues={{ priceModel: 'once', platformFeeRate: '0.10' }}>
          <Form.Item name="title" label={t('marketplace.bundle.nameLabel', '捆绑包名称')} rules={[{ required: true }]}>
            <Input placeholder={t('marketplace.bundle.namePlaceholder', '如：趋势跟踪三件套')} />
          </Form.Item>
          <Form.Item name="description" label={t('marketplace.bundle.descLabel', '描述')}>
            <Input.TextArea rows={2} placeholder={t('marketplace.bundle.descPlaceholder', '描述捆绑包的优势...')} />
          </Form.Item>
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item name="priceModel" label={t('marketplace.bundle.priceModel', '定价模式')}>
                <Select options={[
                  { value: 'once', label: t('marketplace.bundle.once', '一次性购买') },
                  { value: 'subscription', label: t('marketplace.bundle.subscription', '按月订阅') },
                ]} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="priceAmount" label={t('marketplace.bundle.priceAmount', '价格 (¥)')} rules={[{ required: true }]}>
                <InputNumber min={0} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item label={t('marketplace.bundle.strategyIds', '策略 ID（每行一个）')} required>
            <Input.TextArea
              rows={4}
              value={strategyIdsRaw}
              onChange={e => setStrategyIdsRaw(e.target.value)}
              placeholder={'uuid1\nuuid2\nuuid3'}
            />
          </Form.Item>
          <Form.Item name="platformFeeRate" label={t('marketplace.bundle.platformFee', '平台费率')}>
            <Input placeholder="0.10" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
