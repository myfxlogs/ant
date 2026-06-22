import { Tabs, Typography, Empty } from 'antd';
import { useTranslation } from 'react-i18next'
import { BACKTEST_HISTORY_KEY, OVERVIEW_KEY, SCHEDULES_KEY, SELECT_HINT_KEY } from '@/gen/ant/v1/i18n/strategy_library_keys';

;
import { useLibraryCtx, useTemplatesCtx, useSchedulesCtx } from '../../LibraryContext';
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
  const tplCtx = useTemplatesCtx();
  const schCtx = useSchedulesCtx();
  const lib = useLibraryCtx();

  if (!tplCtx.selected) {
    return (
      <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#8c8c8c', flexDirection: 'column', gap: 8 }}>
        <Empty description={t(SELECT_HINT_KEY)} />
      </div>
    );
  }

  const sys = isSystemTemplate(tplCtx.selected);

  const tabItems = [
    {
      key: 'overview' as LibraryTab,
      label: t(OVERVIEW_KEY),
      children: <LibraryOverviewTab />,
    },
    {
      key: 'schedules' as LibraryTab,
      label: t(SCHEDULES_KEY),
      children: <LibraryScheduleTab {...schCtx} />,
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
          <Title level={5} style={{ margin: 0 }}>{String(tplCtx.selected.name || '')}</Title>
          <Text type="secondary" style={{ fontSize: 12 }}>{String(tplCtx.selected.id || '')}</Text>
        </div>
      </div>
      <Tabs activeKey={lib.activeTab} onChange={key => lib.setActiveTab(key as LibraryTab)}
        items={tabItems.map(item => ({ key: item.key, label: item.label, children: item.children }))} />

      {/* Schedule modals rendered outside Tabs so they work from any tab */}
      <EditScheduleModal
        editing={schCtx.editing}
        open={schCtx.openEdit}
        loading={schCtx.loading}
        form={schCtx.form}
        templates={schCtx.templates}
        accounts={schCtx.accounts}
        symbols={schCtx.symbols}
        symbolsLoading={schCtx.symbolsLoading}
        accountIdWatch={schCtx.accountIdWatch}
        onCancel={() => { schCtx.setOpenEdit(false); schCtx.setEditing(null); schCtx.form.resetFields(); }}
        onOk={schCtx.submitEdit}
      />
      <TriggerModal
        open={schCtx.openTrigger}
        triggering={schCtx.triggering}
        result={schCtx.triggerResult}
        context={schCtx.triggerContext}
        onCancel={() => { schCtx.setOpenTrigger(false); schCtx.setTriggerContext(null); schCtx.setTriggerResult(null); }}
        onOrderSend={schCtx.doOrderSend}
      />
      <ScheduleHealthModal
        open={schCtx.healthOpen}
        loading={schCtx.healthLoading}
        target={schCtx.healthTarget}
        summary={schCtx.healthSummary}
        onClose={() => { schCtx.setHealthOpen(false); schCtx.setHealthTarget(null); schCtx.setHealthSummary(null); }}
      />
    </div>
  );
}
