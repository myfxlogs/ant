import React from 'react';
import { Tag, Space, Button, Popconfirm, Typography } from 'antd';
import { RollbackOutlined, EyeOutlined } from '@ant-design/icons';
import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';
import dayjs from 'dayjs';
import type { StrategyVersionInfo } from '@/gen/ant/v1/strategy_runtime_pb';

const { Text } = Typography;

export function buildVersionColumns(
  t: TFunction,
  handleView: (versionNumber: number) => void,
  handleRollback: (versionNumber: number) => void,
  viewLoading: boolean,
  rollingBack: boolean,
) {
  return [
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
}

export function DiffView({ fromCode, toCode }: { fromCode: string; toCode: string }) {
  const { t } = useTranslation();
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
        <span style={{ width: '50%', padding: '0 8px' }}>{t('strategy.version.diffFrom', { defaultValue: 'From' })}</span>
        <span style={{ width: '50%', padding: '0 8px' }}>{t('strategy.version.diffTo', { defaultValue: 'To' })}</span>
      </div>
      {rows}
    </div>
  );
}
