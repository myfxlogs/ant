import { useState, useCallback, useEffect } from 'react';
import { Card, Table, Button, Tag, Typography, Space, InputNumber, Switch, message, Descriptions } from 'antd';
import { SettingOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { marketplaceClient } from '@/client/connect';
import { create } from '@bufbuild/protobuf';
import { EmptySchema } from '@bufbuild/protobuf/wkt';
import {
  UpdateFeeTierRequestSchema,
  GetProviderFeeTierRequestSchema,
} from '@/gen/ant/v1/marketplace_service_pb';
import type { FeeTierInfo } from '@/gen/ant/v1/marketplace_service_pb';
import { useAuthStore } from '@/stores/authStore';

const { Text } = Typography;

export default function FeeTierPanel() {
  const { t } = useTranslation();
  const { user } = useAuthStore();
  const [tiers, setTiers] = useState<FeeTierInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [editing, setEditing] = useState<Record<number, { feeRate: string; minSales: number; enabled: boolean }>>({});
  const [saving, setSaving] = useState<number | null>(null);
  const [myTier, setMyTier] = useState<{ tierName: string; feeRate: string; minSales: number; currentSales: number; nextTierMinSales: number } | null>(null);

  const fetchTiers = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await marketplaceClient.listFeeTiers(create(EmptySchema, {}));
      setTiers(resp.tiers || []);
    } catch {
      setTiers([]);
    }
    setLoading(false);
  }, []);

  const fetchMyTier = useCallback(async () => {
    if (!user?.id) return;
    try {
      const resp = await marketplaceClient.getProviderFeeTier(create(GetProviderFeeTierRequestSchema, { publisherId: user.id }));
      setMyTier({
        tierName: resp.tierName,
        feeRate: resp.feeRate,
        minSales: resp.minSalesCount,
        currentSales: resp.currentSales,
        nextTierMinSales: resp.nextTierMinSales,
      });
    } catch {
      setMyTier(null);
    }
  }, [user?.id]);

  useEffect(() => { fetchTiers(); fetchMyTier(); }, [fetchTiers, fetchMyTier]);

  const handleSave = useCallback(async (tierId: number) => {
    const edit = editing[tierId];
    if (!edit) return;
    setSaving(tierId);
    try {
      await marketplaceClient.updateFeeTier(create(UpdateFeeTierRequestSchema, {
        id: tierId,
        feeRate: edit.feeRate,
        minSalesCount: edit.minSales,
        enabled: edit.enabled,
      }));
      message.success(t('marketplace.feeTier.saved'));
      setEditing(prev => { const next = { ...prev }; delete next[tierId]; return next; });
      fetchTiers();
      fetchMyTier();
    } catch (e: any) {
      message.error(e?.message || t('marketplace.feeTier.saveFailed'));
    } finally {
      setSaving(null);
    }
  }, [editing, t, fetchTiers, fetchMyTier]);

  return (
    <div>
      {/* ── My Current Tier ── */}
      {myTier && (
        <Card size="small" style={{ marginBottom: 16 }}>
          <Descriptions title={t('marketplace.feeTier.myTier')} size="small" column={4}>
            <Descriptions.Item label={t('marketplace.feeTier.tierName')}>{myTier.tierName}</Descriptions.Item>
            <Descriptions.Item label={t('marketplace.feeTier.feeRate')}><Tag color="blue">{(Number(myTier.feeRate) * 100).toFixed(1)}%</Tag></Descriptions.Item>
            <Descriptions.Item label={t('marketplace.feeTier.currentSales')}>{myTier.currentSales}</Descriptions.Item>
            <Descriptions.Item label={t('marketplace.feeTier.minSales')}>{myTier.minSales}</Descriptions.Item>
          </Descriptions>
        </Card>
      )}

      {/* ── Tier Configuration Table ── */}
      <Card size="small" title={<span><SettingOutlined style={{ marginRight: 8 }} />{t('marketplace.feeTier.config')}</span>}>
        <Table<FeeTierInfo>
          rowKey="id"
          dataSource={tiers}
          loading={loading}
          pagination={false}
          size="small"
          columns={[
            { title: t('marketplace.feeTier.tierName'), dataIndex: 'tierName', key: 'name', width: 120 },
            {
              title: t('marketplace.feeTier.minSales'), dataIndex: 'minSalesCount', key: 'minSales', width: 120,
              render: (v: number, row: FeeTierInfo) => {
                const edit = editing[row.id];
                if (edit) return <InputNumber value={edit.minSales} onChange={n => setEditing(p => ({ ...p, [row.id]: { ...p[row.id], minSales: n || 0 } }))} size="small" />;
                return v;
              },
            },
            {
              title: t('marketplace.feeTier.feeRate'), dataIndex: 'feeRate', key: 'rate', width: 140,
              render: (v: string, row: FeeTierInfo) => {
                const edit = editing[row.id];
                if (edit) return <InputNumber value={edit.feeRate} onChange={n => setEditing(p => ({ ...p, [row.id]: { ...p[row.id], feeRate: String(n || '0') } }))} size="small" style={{ width: 100 }} />;
                return <Tag color={Number(v) <= 0.1 ? 'green' : 'gold'}>{(Number(v) * 100).toFixed(1)}%</Tag>;
              },
            },
            {
              title: t('marketplace.feeTier.enabled'), dataIndex: 'enabled', key: 'enabled', width: 80,
              render: (v: boolean, row: FeeTierInfo) => {
                const edit = editing[row.id];
                return <Switch checked={edit?.enabled ?? v} onChange={c => setEditing(p => ({ ...p, [row.id]: { feeRate: p[row.id]?.feeRate ?? row.feeRate, minSales: p[row.id]?.minSales ?? row.minSalesCount, enabled: c } }))} size="small" />;
              },
            },
            {
              title: t('marketplace.feeTier.actions'), key: 'actions', width: 160,
              render: (_: unknown, row: FeeTierInfo) => {
                const edit = editing[row.id];
                return (
                  <Space>
                    {edit ? (
                      <>
                        <Button size="small" type="primary" loading={saving === row.id} onClick={() => handleSave(row.id)}>
                          {t('common.save')}
                        </Button>
                        <Button size="small" onClick={() => setEditing(p => { const next = { ...p }; delete next[row.id]; return next; })}>
                          {t('common.cancel')}
                        </Button>
                      </>
                    ) : (
                      <Button size="small" onClick={() => setEditing(p => ({ ...p, [row.id]: { feeRate: row.feeRate, minSales: row.minSalesCount, enabled: row.enabled } }))}>
                        {t('common.edit')}
                      </Button>
                    )}
                  </Space>
                );
              },
            },
          ]}
        />
      </Card>
    </div>
  );
}
