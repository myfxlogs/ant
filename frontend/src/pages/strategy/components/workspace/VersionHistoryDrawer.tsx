import { useEffect, useState } from 'react';
import { Drawer, Table, Button, Space, Empty, Spin, message } from 'antd';
import { DiffOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import { strategyVersionApi } from '@/client/strategy';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';
import { buildVersionColumns, DiffView } from './VersionHistoryDrawerHelpers';

interface Props {
  open: boolean;
  strategyId: string | undefined;
  onClose: () => void;
  onRollback?: (sourceCode: string) => void;
}

export default function VersionHistoryDrawer({ open, strategyId, onClose, onRollback }: Props) {
  const { t } = useTranslation();
  const [versions, setVersions] = useState<StrategyVersionInfo[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(20);
  const [selectedVersions, setSelectedVersions] = useState<number[]>([]);
  const [diffModalOpen, setDiffModalOpen] = useState(false);
  const [diffData, setDiffData] = useState<{ fromCode: string; toCode: string; fromVer: number; toVer: number } | null>(null);
  const [diffLoading, setDiffLoading] = useState(false);
  const [viewVersion, setViewVersion] = useState<{ code: string; ver: number } | null>(null);
  const [viewLoading, setViewLoading] = useState(false);
  const [rollingBack, setRollingBack] = useState(false);

  const fetchVersions = async () => {
    if (!strategyId) return;
    setLoading(true);
    try {
      const r = await strategyVersionApi.list(strategyId, pageSize, (page - 1) * pageSize);
      setVersions(r.versions);
      setTotal(r.total);
    } catch (e) {
      message.error(String((e as Error)?.message || t('strategy.version.loadFailed', { defaultValue: 'Failed to load versions' })));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open && strategyId) {
      setPage(1);
      setSelectedVersions([]);
      fetchVersions();
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchVersions not memoized, mount/prop-change only  | REF: rd.md#part-0.2-hooks-deps
  }, [open, strategyId]);

  useEffect(() => {
    if (open && strategyId) fetchVersions();
  // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchVersions not memoized, page-change only  | REF: rd.md#part-0.2-hooks-deps
  }, [page]);

  const handleRollback = async (versionNumber: number) => {
    if (!strategyId) return;
    setRollingBack(true);
    try {
      const r = await strategyVersionApi.rollback(strategyId, versionNumber);
      message.success(t('strategy.version.rollbackSuccess', { defaultValue: 'Rolled back to version {{n}}', n: versionNumber }));
      if (onRollback && r.restoredSourceCode) {
        onRollback(r.restoredSourceCode);
      }
      fetchVersions();
    } catch (e) {
      message.error(String((e as Error)?.message || t('strategy.version.rollbackFailed', { defaultValue: 'Rollback failed' })));
    } finally {
      setRollingBack(false);
    }
  };

  const handleView = async (versionNumber: number) => {
    if (!strategyId) return;
    setViewLoading(true);
    try {
      const r = await strategyVersionApi.get(strategyId, versionNumber);
      setViewVersion({ code: r.sourceCode, ver: versionNumber });
    } catch (e) {
      message.error(String((e as Error)?.message || t('strategy.version.loadVersionFailed', { defaultValue: 'Failed to load version' })));
    } finally {
      setViewLoading(false);
    }
  };

  const handleDiff = async () => {
    if (!strategyId || selectedVersions.length !== 2) return;
    const [a, b] = selectedVersions.sort((x, y) => x - y);
    setDiffLoading(true);
    setDiffModalOpen(true);
    try {
      const r = await strategyVersionApi.diff(strategyId, a, b);
      setDiffData({ fromCode: r.fromSourceCode, toCode: r.toSourceCode, fromVer: a, toVer: b });
    } catch (e) {
      message.error(String((e as Error)?.message || t('strategy.version.loadDiffFailed', { defaultValue: 'Failed to load diff' })));
    } finally {
      setDiffLoading(false);
    }
  };

  const columns = buildVersionColumns(t, handleView, handleRollback, viewLoading, rollingBack);

  const rowSelection = {
    selectedRowKeys: selectedVersions,
    onChange: (keys: React.Key[]) => {
      setSelectedVersions(keys.map(k => Number(k)).slice(-2));
    },
    getCheckboxProps: () => ({ disabled: false }),
  };

  return (
    <>
      <Drawer
        title={t('strategy.version.title', { defaultValue: 'Version History' })}
        open={open}
        onClose={onClose}
        width={720}
        styles={{ body: { padding: 0 } }}
        extra={
          <Space>
            <Button
              size="small"
              icon={<DiffOutlined />}
              disabled={selectedVersions.length !== 2}
              onClick={handleDiff}
            >
              {t('strategy.version.diff', { defaultValue: 'Diff' })}
            </Button>
            <Button size="small" icon={<ReloadOutlined />} onClick={fetchVersions} loading={loading} />
          </Space>
        }
      >
        <Table
          dataSource={versions}
          columns={columns}
          rowKey="versionNumber"
          rowSelection={rowSelection}
          loading={loading}
          size="small"
          pagination={{
            current: page,
            pageSize,
            total,
            onChange: (p) => setPage(p),
            showSizeChanger: false,
            size: 'small',
          }}
          locale={{ emptyText: <Empty description={t('strategy.version.empty', { defaultValue: 'No version history yet' })} /> }}
        />
      </Drawer>

      {/* Diff Modal */}
      <Drawer
        title={diffData ? `Diff: v${diffData.fromVer} → v${diffData.toVer}` : 'Diff'}
        open={diffModalOpen}
        onClose={() => { setDiffModalOpen(false); setDiffData(null); }}
        width={900}
        styles={{ body: { padding: 0 } }}
      >
        {diffLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : diffData ? (
          <DiffView fromCode={diffData.fromCode} toCode={diffData.toCode} />
        ) : null}
      </Drawer>

      {/* View Version Modal */}
      <Drawer
        title={viewVersion ? `Version ${viewVersion.ver}` : 'Version'}
        open={!!viewVersion}
        onClose={() => setViewVersion(null)}
        width={800}
        styles={{ body: { padding: 0 } }}
      >
        {viewLoading ? (
          <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
        ) : viewVersion ? (
          <pre style={{ padding: 12, fontSize: 12, overflow: 'auto', height: '100%', background: '#fafafa', margin: 0 }}>
            {viewVersion.code}
          </pre>
        ) : null}
      </Drawer>
    </>
  );
}
