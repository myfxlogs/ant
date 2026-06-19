import { Table, Button, Popconfirm, Tag } from 'antd';
import { CloseOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next'
import { CLOSE_POSITION_CONFIRM_KEY, CLOSE_POSITION_KEY, NO_POSITIONS_KEY, OPEN_TIME_KEY, PRICE_KEY, SIDE_KEY, STOP_LOSS_KEY, SYMBOL_KEY, TAKE_PROFIT_KEY, VOLUME_KEY } from '@/gen/ant/v1/i18n/trading_keys';

;
import { useMemo, useCallback } from 'react';
import { useTradingStore } from '@/stores/tradingStore';
import { useTrading } from '@/hooks/useTrading';
import type { Position } from '@/types/trading';

export default function PositionsTable() {
  const { t } = useTranslation();
  const positions = useTradingStore((s) => s.positions);
  const currentAccountId = useTradingStore((s) => s.currentAccountId);
  const loading = useTradingStore((s) => s.loading);
  const removePosition = useTradingStore((s) => s.removePosition);
  const { closeOrder } = useTrading();

  const handleClose = useCallback(async (ticket: number) => {
    if (!currentAccountId) return;
    try {
      await closeOrder({ accountId: currentAccountId, ticket: BigInt(ticket) });
      removePosition(currentAccountId, ticket);
    } catch {
      // error shown by hook
    }
  }, [currentAccountId, closeOrder, removePosition]);

  const columns = useMemo(() => {
    const fmt = (v: number | undefined) =>
      v != null ? v.toFixed(5) : '-';
    const fmtPnL = (v: number | undefined) =>
      v != null ? v.toFixed(2) : '-';
    const fmtTime = (v: number | undefined) => {
      if (!v || v <= 0) return '-';
      return new Date(v * 1000).toLocaleString();
    };

    return [
      {
        title: '#',
        dataIndex: 'ticket',
        key: 'ticket',
        width: 80,
        render: (v: number) => String(v),
      },
      {
        title: t(SYMBOL_KEY, 'Symbol'),
        dataIndex: 'symbol',
        key: 'symbol',
        width: 100,
      },
      {
        title: t(SIDE_KEY, 'Side'),
        dataIndex: 'type',
        key: 'type',
        width: 80,
        render: (v: number) => {
          const s = String(v ?? '');
          const color = s.includes('buy') ? 'green' : 'red';
          return <Tag color={color}>{s.toUpperCase()}</Tag>;
        },
      },
      {
        title: t(VOLUME_KEY, 'Volume'),
        dataIndex: 'volume',
        key: 'volume',
        width: 80,
        render: (v: number) => (v ?? 0).toFixed(2),
      },
      {
        title: t(PRICE_KEY, 'Open Price'),
        dataIndex: 'openPrice',
        key: 'openPrice',
        width: 100,
        render: (v: number) => fmt(v),
      },
      {
        title: t(PRICE_KEY, 'Current'),
        dataIndex: 'currentPrice',
        key: 'currentPrice',
        width: 100,
        render: (v: number) => fmt(v),
      },
      {
        title: t(STOP_LOSS_KEY, 'SL'),
        dataIndex: 'sl',
        key: 'sl',
        width: 80,
        render: (v: number) => (v && v !== 0 ? fmt(v) : '-'),
      },
      {
        title: t(TAKE_PROFIT_KEY, 'TP'),
        dataIndex: 'tp',
        key: 'tp',
        width: 80,
        render: (v: number) => (v && v !== 0 ? fmt(v) : '-'),
      },
      {
        title: t(OPEN_TIME_KEY, 'Open Time'),
        dataIndex: 'openTime',
        key: 'openTime',
        width: 160,
        render: (v: number | undefined) => fmtTime(v),
      },
      {
        title: 'P&L',
        dataIndex: 'profit',
        key: 'profit',
        width: 100,
        render: (v: number) => {
          const color = (v ?? 0) >= 0 ? '#52c41a' : '#ff4d4f';
          return <span style={{ color, fontWeight: 600 }}>{fmtPnL(v)}</span>;
        },
      },
      {
        title: '',
        key: 'actions',
        width: 80,
        render: (_: unknown, record: Position) => (
          <Popconfirm
            title={t(CLOSE_POSITION_CONFIRM_KEY, 'Close this position?')}
            onConfirm={() => handleClose(record.ticket)}
            okText={t('common.ok')}
            cancelText={t('common.cancel')}
          >
            <Button size="small" danger icon={<CloseOutlined />}>
              {t(CLOSE_POSITION_KEY, 'Close')}
            </Button>
          </Popconfirm>
        ),
      },
    ];
  }, [t, handleClose]);

  return (
    <Table
      dataSource={positions as unknown as Record<string, unknown>[]}
      columns={columns}
      rowKey={(r) => String((r as unknown as Position).ticket)}
      loading={loading}
      size="small"
      pagination={false}
      locale={{ emptyText: t(NO_POSITIONS_KEY, 'No open positions') }}
      scroll={{ x: 1040 }}
      tableLayout="fixed"
      style={{ marginTop: 16 }}
    />
  );
}
