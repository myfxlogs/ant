import { useState, useCallback } from 'react';
import { Table, Button, Tag, Space, Input, Select, Modal, InputNumber, message, Tooltip } from 'antd';
import { StarOutlined, StarFilled, SearchOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import type { AdminStrategyItem } from '@/gen/ant/v1/marketplace_service_pb';

const { Search } = Input;

export default function MarketplaceManagement() {
  const { t } = useTranslation();
  const [strategies, setStrategies] = useState<AdminStrategyItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [statusFilter, setStatusFilter] = useState('');
  const [search, setSearch] = useState('');
  const [featureModalOpen, setFeatureModalOpen] = useState(false);
  const [featureTarget, setFeatureTarget] = useState<AdminStrategyItem | null>(null);
  const [featurePriority, setFeaturePriority] = useState(0);
  const [actionLoading, setActionLoading] = useState(false);

  const fetchStrategies = useCallback(async (p: number, s: string, status: string) => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.adminListStrategies({
        status: status || undefined,
        keyword: s || undefined,
        limit: 20,
        offset: (p - 1) * 20,
      });
      setStrategies(resp.strategies || []);
      setTotal(resp.total || 0);
    } catch {
      message.error(t('admin.marketplace.loadFailed', { defaultValue: 'Failed to load strategies' }));
    } finally {
      setLoading(false);
    }
  }, [t]);

  const handleFeature = async () => {
    if (!featureTarget) return;
    setActionLoading(true);
    try {
      await marketplaceClient.adminFeatureStrategy({
        strategyId: featureTarget.strategy?.strategyId || '',
        featured: true,
        priority: featurePriority,
      });
      message.success(t('admin.marketplace.featureSuccess', { defaultValue: 'Strategy featured' }));
      setFeatureModalOpen(false);
      fetchStrategies(page, search, statusFilter);
    } catch {
      message.error(t('admin.marketplace.featureFailed', { defaultValue: 'Failed to feature strategy' }));
    } finally {
      setActionLoading(false);
    }
  };

  const handleUnfeature = async (strategyId: string) => {
    try {
      await marketplaceClient.adminFeatureStrategy({
        strategyId,
        featured: false,
        priority: 0,
      });
      message.success(t('admin.marketplace.unfeatureSuccess', { defaultValue: 'Removed featured' }));
      fetchStrategies(page, search, statusFilter);
    } catch {
      message.error(t('admin.marketplace.unfeatureFailed', { defaultValue: 'Failed to unfeature' }));
    }
  };

  const columns = [
    {
      title: t('admin.marketplace.colTitle', { defaultValue: 'Title' }),
      dataIndex: ['strategy', 'title'],
      key: 'title',
      ellipsis: true,
    },
    {
      title: t('admin.marketplace.colPublisher', { defaultValue: 'Publisher' }),
      dataIndex: ['strategy', 'strategyName'],
      key: 'publisher',
      ellipsis: true,
    },
    {
      title: t('admin.marketplace.colStatus', { defaultValue: 'Status' }),
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'published' ? 'green' : status === 'hidden' ? 'orange' : 'default'}>
          {status}
        </Tag>
      ),
    },
    {
      title: t('admin.marketplace.colPrice', { defaultValue: 'Price' }),
      key: 'price',
      render: (_: unknown, r: AdminStrategyItem) => {
        const model = r.strategy?.priceModel || '';
        const amount = r.strategy?.priceAmount || '0';
        if (model === 'free') return <Tag color="green">FREE</Tag>;
        return <span>{amount}</span>;
      },
    },
    {
      title: t('admin.marketplace.colSales', { defaultValue: 'Sales' }),
      dataIndex: 'totalSales',
      key: 'sales',
      sorter: (a: AdminStrategyItem, b: AdminStrategyItem) => a.totalSales - b.totalSales,
    },
    {
      title: t('admin.marketplace.colRevenue', { defaultValue: 'Revenue' }),
      dataIndex: 'totalRevenue',
      key: 'revenue',
      render: (v: string) => v || '0',
    },
    {
      title: t('admin.marketplace.colFeatured', { defaultValue: 'Featured' }),
      key: 'featured',
      render: (_: unknown, r: AdminStrategyItem) =>
        r.isFeatured ? (
          <Tag color="gold" icon={<StarFilled />}>
            P{r.featuredPriority}
          </Tag>
        ) : (
          <span style={{ color: '#ccc' }}>—</span>
        ),
    },
    {
      title: t('admin.marketplace.colActions', { defaultValue: 'Actions' }),
      key: 'actions',
      render: (_: unknown, r: AdminStrategyItem) => (
        <Space>
          {!r.isFeatured ? (
            <Tooltip title={t('admin.marketplace.feature', { defaultValue: 'Feature' })}>
              <Button
                size="small"
                icon={<StarOutlined />}
                onClick={() => {
                  setFeatureTarget(r);
                  setFeaturePriority(r.featuredPriority || 0);
                  setFeatureModalOpen(true);
                }}
              />
            </Tooltip>
          ) : (
            <Tooltip title={t('admin.marketplace.unfeature', { defaultValue: 'Remove featured' })}>
              <Button
                size="small"
                icon={<StarFilled />}
                onClick={() => handleUnfeature(r.strategy?.strategyId || '')}
              />
            </Tooltip>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Select
          placeholder={t('admin.marketplace.filterStatus', { defaultValue: 'All statuses' })}
          allowClear
          style={{ width: 150 }}
          value={statusFilter || undefined}
          onChange={(v) => {
            setStatusFilter(v || '');
            setPage(1);
            fetchStrategies(1, search, v || '');
          }}
          options={[
            { label: 'Published', value: 'published' },
            { label: 'Hidden', value: 'hidden' },
          ]}
        />
        <Search
          placeholder={t('admin.marketplace.searchPlaceholder', { defaultValue: 'Search by title...' })}
          allowClear
          style={{ width: 300 }}
          onSearch={(v) => {
            setSearch(v);
            setPage(1);
            fetchStrategies(1, v, statusFilter);
          }}
        />
      </Space>

      <Table
        dataSource={strategies}
        columns={columns}
        rowKey={(r) => r.strategy?.strategyId || ''}
        loading={loading}
        pagination={{
          current: page,
          total,
          pageSize: 20,
          onChange: (p) => {
            setPage(p);
            fetchStrategies(p, search, statusFilter);
          },
        }}
        size="small"
      />

      <Modal
        title={t('admin.marketplace.featureTitle', { defaultValue: 'Feature Strategy' })}
        open={featureModalOpen}
        onCancel={() => setFeatureModalOpen(false)}
        onOk={handleFeature}
        confirmLoading={actionLoading}
      >
        <p style={{ marginBottom: 8 }}>
          {t('admin.marketplace.featureDesc', { defaultValue: 'Set priority for featured placement. Higher = more prominent.' })}
        </p>
        <InputNumber
          min={0}
          max={100}
          value={featurePriority}
          onChange={(v) => setFeaturePriority(v || 0)}
          style={{ width: '100%' }}
        />
      </Modal>
    </div>
  );
}
