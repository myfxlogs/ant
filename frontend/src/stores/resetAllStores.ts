import { useTradingStore } from '@/stores/tradingStore';
import { useNotificationStore } from '@/stores/notificationStore';
import { useWorkspaceStore } from '@/stores/workspaceStore';
import { useChartIndicatorsStore } from '@/stores/chartIndicatorsStore';

/**
 * Reset all user-scoped stores to their initial state.
 * Called on logout and on user.id change (defensive guard in App.tsx).
 *
 * This prevents data leakage when switching users:
 * user A logs out → user B logs in → B should never see A's data.
 */
export function resetAllStores(): void {
  useTradingStore.getState().reset();
  useNotificationStore.getState().reset();
  useWorkspaceStore.getState().reset();
  useChartIndicatorsStore.getState().reset();
}
