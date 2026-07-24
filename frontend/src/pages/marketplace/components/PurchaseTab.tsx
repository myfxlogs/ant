import { useState } from 'react';
import { Table, Tag, Typography, Button, Space, Empty, Drawer, Modal, Input, message } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { EyeOutlined, ThunderboltOutlined, RollbackOutlined, RocketOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'

;
import { formatDateTime } from '@/utils/date';
import { useMarketplaceCtx } from '../MarketplaceContext';
import type { PurchasedItem } from '../hooks/useMarketplace';
import ProtectedBacktestPanel from './ProtectedBacktestPanel';
import DeployScheduleModal from '@/pages/strategy/components/DeployScheduleModal';
import { marketplaceClient } from '@/client/connect';

const { Text } = Typography;

export default function PurchaseTab() {
  const { t } = useTranslation();
  const m = useMarketplaceCtx();
  const [backtestDrawerOpen, setBacktestDrawerOpen] = useState(false);
  const [backtestStrategyId, setBacktestStrategyId] = useState('');
  const [refundTarget, setRefundTarget] = useState<PurchasedItem | null>(null);
  const [refundReason, setRefundReason] = useState('');
  const [refundLoading, setRefundLoading] = useState(false);
  const [deployOpen, setDeployOpen] = useState(false);
  const [deployTarget, setDeployTarget] = useState<{ id: string; name: string } | null>(null);

  if (!m.purchasesLoading && m.purchases.length === 0) {
    return <Empty description={t('marketplace.purchases.empty')} />;
  }

  const columns: ColumnsType<PurchasedItem> = [
    {
      title: t('marketplace.purchases.strategy'),
      dataIndex: 'strategyId', key: 'strategy',
      render: (id: string, row: PurchasedItem) => {
        const s = m.strategies.find(st => st.strategyId === row.strategyId);
        return <Text>{s?.title || s?.strategyName || String(id).slice(0, 12)}</Text>;
      },
    },
    {
      title: t('marketplace.purchases.date'),
      key: 'date',
      render: (_: unknown, row: PurchasedItem) => <Text>{formatDateTime(String(row.createdAt || row.purchasedAt || ''))}</Text>,
    },
    {
      title: t('marketplace.purchases.status'),
      key: 'status',
      render: (_: unknown, row: PurchasedItem) => (
        row.active
          ? <Tag color="green">{t('common.active')}</Tag>
          : <Tag>{t('common.inactive')}</Tag>
      ),
    },
    {
      title: t('marketplace.purchases.actions'),
      key: 'actions',
      render: (_: unknown, row: PurchasedItem) => (
        <Space>
          <Button size="small" icon={<EyeOutlined />} onClick={() => {
            const s = m.strategies.find(s => s.strategyId === row.strategyId);
            if (s) m.openDetail(s);
          }}>
            {t('strategy.backtestHistory.actions.view')}
          </Button>
          <Button size="small" type="primary" icon={<ThunderboltOutlined />} onClick={() => {
            setBacktestStrategyId(row.strategyId);
            setBacktestDrawerOpen(true);
          }}>
            {t('marketplace.purchases.runBacktest')}
          </Button>
          <Button size="small" icon={<RocketOutlined />} onClick={() => {
            const s = m.strategies.find(s => s.strategyId === row.strategyId);
            setDeployTarget({ id: row.strategyId, name: s?.title || s?.strategyName || row.strategyId.slice(0, 12) });
            setDeployOpen(true);
          }}>
            {t('strategy.templates.actions.deploy', { defaultValue: 'Deploy' })}
          </Button>
          {row.active && (
            <Button size="small" danger icon={<RollbackOutlined />} onClick={() => {
              setRefundTarget(row);
              setRefundReason('');
            }}>
              {t('marketplace.purchases.refund')}
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <>
      <Table
        rowKey="subscriptionId"
        columns={columns}
        dataSource={m.purchases}
        loading={m.purchasesLoading}
        pagination={{ pageSize: 10 }}
        size="small"
      />

      <Drawer
        title={t('marketplace.backtest.title')}
        open={backtestDrawerOpen}
        onClose={() => setBacktestDrawerOpen(false)}
        width={640}
        destroyOnClose
      >
        {backtestStrategyId && <ProtectedBacktestPanel strategyId={backtestStrategyId} />}
      </Drawer>

      <Modal
        title={t('marketplace.purchases.refundTitle')}
        open={!!refundTarget}
        onCancel={() => setRefundTarget(null)}
        confirmLoading={refundLoading}
        onOk={async () => {
          if (!refundTarget) return;
          if (!refundReason.trim()) {
            message.warning(t('marketplace.purchases.refundReasonRequired'));
            return;
          }
          setRefundLoading(true);
          try {
            await marketplaceClient.requestRefund({
              subscriptionId: refundTarget.subscriptionId,
              reason: refundReason,
            });
            message.success(t('marketplace.purchases.refundSubmitted'));
            setRefundTarget(null);
          } catch {
            message.error(t('marketplace.purchases.refundFailed'));
          } finally {
            setRefundLoading(false);
          }
        }}
      >
        <Input.TextArea
          rows={4}
          placeholder={t('marketplace.purchases.refundReasonPlaceholder')}
          value={refundReason}
          onChange={e => setRefundReason(e.target.value)}
          maxLength={500}
          showCount
        />
      </Modal>

      {deployTarget && (
        <DeployScheduleModal
          open={deployOpen}
          templateId={deployTarget.id}
          templateName={deployTarget.name}
          onClose={() => setDeployOpen(false)}
        />
      )}
    </>
  );
}
