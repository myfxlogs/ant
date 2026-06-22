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

import EditScheduleModal from '../EditScheduleModal';
import TriggerModal from '../TriggerModal';
import ScheduleHealthModal from '../ScheduleHealthModal';

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

      {/* Schedule modals rendered outside Tabs so they work from any tab */}
      <EditScheduleModal
        editing={lib.scheduleProps.editing}
        open={lib.scheduleProps.openEdit}
        loading={lib.scheduleProps.loading}
        form={lib.scheduleProps.form}
        templates={lib.scheduleProps.templates}
        accounts={lib.scheduleProps.accounts}
        symbols={lib.scheduleProps.symbols}
        symbolsLoading={lib.scheduleProps.symbolsLoading}
        accountIdWatch={lib.scheduleProps.accountIdWatch}
        onCancel={() => { lib.scheduleProps.setOpenEdit(false); lib.scheduleProps.setEditing(null); lib.scheduleProps.form.resetFields(); }}
        onOk={lib.scheduleProps.submitEdit}
      />
      <TriggerModal
        open={lib.scheduleProps.openTrigger}
        triggering={lib.scheduleProps.triggering}
        result={lib.scheduleProps.triggerResult}
        context={lib.scheduleProps.triggerContext}
        onCancel={() => { lib.scheduleProps.setOpenTrigger(false); lib.scheduleProps.setTriggerContext(null); lib.scheduleProps.setTriggerResult(null); }}
        onOrderSend={lib.scheduleProps.doOrderSend}
      />
      <ScheduleHealthModal
        open={lib.scheduleProps.healthOpen}
        loading={lib.scheduleProps.healthLoading}
        target={lib.scheduleProps.healthTarget}
        summary={lib.scheduleProps.healthSummary}
        onClose={() => { lib.scheduleProps.setHealthOpen(false); lib.scheduleProps.setHealthTarget(null); lib.scheduleProps.setHealthSummary(null); }}
      />
    </div>
  );
}
