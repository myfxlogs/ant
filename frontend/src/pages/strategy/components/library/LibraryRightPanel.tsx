import { Tabs, Typography, Empty } from 'antd';
import { useTranslation } from 'react-i18next';
import type { StrategyTemplate } from '@/client/strategy';
import type { LibraryTab } from '../../hooks/useStrategyLibrary';
import LibraryOverviewTab from './LibraryOverviewTab';
import LibraryScheduleTab from './LibraryScheduleTab';
import LibraryBacktestHistoryTab from './LibraryBacktestHistoryTab';

const { Title, Text } = Typography;

interface Props {
  selectedTemplate: StrategyTemplate | null;
  activeTab: LibraryTab;
  onTabChange: (tab: LibraryTab) => void;
  // Shared
  scheduleCount: number;
  publishing: boolean;
  // Overview
  onEdit: (tpl: StrategyTemplate) => void;
  onDelete: (id: string) => void;
  onPublish: (id: string) => void;
  onUnpublish: (id: string) => void;
  onViewCode: (code: string) => void;
  onRunBacktest: (tpl: StrategyTemplate) => void;
  onOpenCreateSchedule: () => void;
  // Schedule tab (all props passed through from hook)
  scheduleProps: any;
  // Backtest history tab
  backtestProps: any;
}

export default function LibraryRightPanel({
  selectedTemplate, activeTab, onTabChange,
  scheduleCount, publishing,
  onEdit, onDelete, onPublish, onUnpublish, onViewCode, onRunBacktest, onOpenCreateSchedule,
  scheduleProps, backtestProps,
}: Props) {
  const { t } = useTranslation();

  if (!selectedTemplate) {
    return (
      <div style={{
        flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
        color: '#8c8c8c', flexDirection: 'column', gap: 8,
      }}>
        <Empty description={t('strategy.library.selectHint', '选择左侧策略查看详情')} />
      </div>
    );
  }

  const isSystem = Boolean((selectedTemplate as any).isSystem)
    || (Array.isArray((selectedTemplate as any).tags) ? (selectedTemplate as any).tags : []).includes('preset')
    || String(selectedTemplate.id || '').startsWith('default-');

  const tabItems = [
    {
      key: 'overview' as LibraryTab,
      label: t('strategy.library.overview', '概览'),
      children: (
        <LibraryOverviewTab
          template={selectedTemplate}
          scheduleCount={scheduleCount}
          publishing={publishing}
          onEdit={onEdit}
          onDelete={onDelete}
          onPublish={onPublish}
          onUnpublish={onUnpublish}
          onViewCode={onViewCode}
          onRunBacktest={onRunBacktest}
          onOpenCreateSchedule={onOpenCreateSchedule}
        />
      ),
    },
    {
      key: 'schedules' as LibraryTab,
      label: t('strategy.library.schedules', '调度'),
      children: <LibraryScheduleTab {...scheduleProps} />,
    },
    {
      key: 'backtest' as LibraryTab,
      label: t('strategy.library.backtestHistory', '回测历史'),
      children: <LibraryBacktestHistoryTab {...backtestProps} />,
    },
  ];

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '0 20px', minWidth: 0 }}>
      {/* Header */}
      <div style={{ padding: '16px 0 8px', borderBottom: '1px solid #f0f0f0', marginBottom: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Title level={5} style={{ margin: 0 }}>
            {String(selectedTemplate.name || '')}
          </Title>
          <Text type="secondary" style={{ fontSize: 12 }}>
            {String(selectedTemplate.id || '')}
          </Text>
        </div>
      </div>
      <Tabs
        activeKey={activeTab}
        onChange={(key) => onTabChange(key as LibraryTab)}
        items={tabItems.map(item => ({ key: item.key, label: item.label, children: item.children }))}
      />
    </div>
  );
}
