import { Tabs, Typography, Empty } from 'antd';
import { useTranslation } from 'react-i18next'
import { BACKTEST_HISTORY_KEY, OVERVIEW_KEY, SCHEDULES_KEY, SELECT_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';

;
import { useLibraryCtx } from '../../LibraryContext';
import { isSystemTemplate } from '../../hooks/libraryTypes';
import LibraryOverviewTab from './LibraryOverviewTab';
import LibraryScheduleTab from './LibraryScheduleTab';
import LibraryBacktestHistoryTab from './LibraryBacktestHistoryTab';
import type { LibraryTab } from '../../hooks/useStrategyLibrary';

const { Title, Text } = Typography;

export default function LibraryRightPanel() {
  const { t } = useTranslation();
  const lib = useLibraryCtx();

  if (!lib.selected) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8c8c8c', flexDirection: 'column', gap: 8 }}>
        <Empty description={t(SELECT_HINT_KEY)} />
      </div>
    );
  }

  const sys = isSystemTemplate(lib.selected);

  const tabItems = [
    {
      key: 'overview' as LibraryTab,
      label: t(OVERVIEW_KEY),
      children: <LibraryOverviewTab />,
    },
    {
      key: 'schedules' as LibraryTab,
      label: t(SCHEDULES_KEY),
      children: <LibraryScheduleTab {...lib.scheduleProps} />,
    },
    {
      key: 'backtest' as LibraryTab,
      label: t(BACKTEST_HISTORY_KEY),
      children: <LibraryBacktestHistoryTab />,
    },
  ];

  return (
    <div style={{ flex: 1, overflowY: 'auto', padding: '0 20px', minWidth: 0 }}>
      <div style={{ padding: '16px 0 8px', borderBottom: '1px solid #f0f0f0', marginBottom: 0 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Title level={5} style={{ margin: 0 }}>{String(lib.selected.name || '')}</Title>
          <Text type="secondary" style={{ fontSize: 12 }}>{String(lib.selected.id || '')}</Text>
        </div>
      </div>
      <Tabs activeKey={lib.activeTab} onChange={key => lib.setActiveTab(key as LibraryTab)}
        items={tabItems.map(item => ({ key: item.key, label: item.label, children: item.children }))} />
    </div>
  );
}
