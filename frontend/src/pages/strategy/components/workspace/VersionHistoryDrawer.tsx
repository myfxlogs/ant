import { useEffect, useState } from 'react';
import { Drawer, Table, Button, Space, Tag, Popconfirm, message, Typography, Empty, Spin } from 'antd';
import { RollbackOutlined, DiffOutlined, EyeOutlined, ReloadOutlined } from '@ant-design/icons';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import { strategyVersionApi } from '@/client/strategy';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text, _Paragraph } = Typography;

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
  }, [open, strategyId]);

  useEffect(() => {
    if (open && strategyId) fetchVersions();
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

  const columns = [
    {
      title: t('strategy.version.colVersion', { defaultValue: 'Version' }),
      dataIndex: 'versionNumber',
      key: 'versionNumber',
      width: 80,
      render: (n: number) => <Tag color="blue">v{n}</Tag>,
    },
    {
      title: t('strategy.version.colSummary', { defaultValue: 'Change Summary' }),
      dataIndex: 'changeSummary',
      key: 'changeSummary',
      ellipsis: true,
      render: (s: string) => s || <Text type="secondary">—</Text>,
    },
    {
      title: t('strategy.version.colLang', { defaultValue: 'Lang' }),
      dataIndex: 'sourceLang',
      key: 'sourceLang',
      width: 70,
      render: (s: string) => <Tag>{s || 'mql'}</Tag>,
    },
    {
      title: t('strategy.version.colHash', { defaultValue: 'Hash' }),
      dataIndex: 'codeHash',
      key: 'codeHash',
      width: 100,
      render: (h: string) => <Text code style={{ fontSize: 10 }}>{h?.slice(0, 8)}</Text>,
    },
    {
      title: t('strategy.version.colDate', { defaultValue: 'Date' }),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 150,
      render: (ts: unknown) => {
        if (!ts) return <Text type="secondary">—</Text>;
        const d = ts instanceof Date ? ts : new Date(ts);
        return dayjs(d).format('YYYY-MM-DD HH:mm');
      },
    },
    {
      title: t('strategy.version.colActions', { defaultValue: 'Actions' }),
      key: 'actions',
      width: 120,
      render: (_: unknown, record: StrategyVersionInfo) => (
        <Space size={4}>
          <Button size="small" icon={<EyeOutlined />} loading={viewLoading} onClick={() => handleView(record.versionNumber)} />
          <Popconfirm
            title={t('strategy.version.rollbackConfirm', { defaultValue: 'Rollback to v{{n}}?', n: record.versionNumber })}
            onConfirm={() => handleRollback(record.versionNumber)}
            okText={t('common.yes', { defaultValue: 'Yes' })}
            cancelText={t('common.no', { defaultValue: 'No' })}
          >
            <Button size="small" icon={<RollbackOutlined />} loading={rollingBack} danger />
          </Popconfirm>
        </Space>
      ),
    },
  ];

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

function DiffView({ fromCode, toCode }: { fromCode: string; toCode: string }) {
  const fromLines = fromCode.split('\n');
  const toLines = toCode.split('\n');
  const maxLen = Math.max(fromLines.length, toLines.length);
  const rows: React.ReactNode[] = [];
  for (let i = 0; i < maxLen; i++) {
    const fromLine = fromLines[i] ?? '';
    const toLine = toLines[i] ?? '';
    const changed = fromLine !== toLine;
    rows.push(
      <div key={i} style={{ display: 'flex', fontFamily: 'monospace', fontSize: 12, lineHeight: '20px' }}>
        <span style={{ width: 40, textAlign: 'right', paddingRight: 8, color: '#999', flexShrink: 0 }}>{i + 1}</span>
        <span style={{ width: '50%', padding: '0 8px', background: changed && fromLine ? '#fff1f0' : 'transparent', whiteSpace: 'pre', overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {fromLine || ' '}
        </span>
        <span style={{ width: '50%', padding: '0 8px', background: changed && toLine ? '#f6ffed' : 'transparent', whiteSpace: 'pre', overflow: 'hidden', textOverflow: 'ellipsis' }}>
          {toLine || ' '}
        </span>
      </div>
    );
  }
  return (
    <div style={{ overflow: 'auto', height: '100%' }}>
      <div style={{ display: 'flex', padding: '4px 8px', background: '#fafafa', borderBottom: '1px solid #f0f0f0', fontSize: 11, color: '#999' }}>
        <span style={{ width: 40, textAlign: 'right', paddingRight: 8 }}>#</span>
        <span style={{ width: '50%', padding: '0 8px' }}>From</span>
        <span style={{ width: '50%', padding: '0 8px' }}>To</span>
      </div>
      {rows}
    </div>
  );
}
