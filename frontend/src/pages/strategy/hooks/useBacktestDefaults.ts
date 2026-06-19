
import { DEFAULTS_LOADED_KEY, DEFAULTS_RESET_KEY, DEFAULTS_SAVED_KEY, SETTINGS_LOAD_KEY, SETTINGS_RESET_KEY, SETTINGS_SAVE_KEY } from '@/gen/ant/v1/i18n/strategy_backtest_params_keys';
import { useMemo } from 'react'
;
import { message } from 'antd';
import type { MenuProps } from 'antd';
import type { TFunction } from 'i18next';

const DEFAULTS_KEY = 'ant_backtest_defaults';

export const FACTORY_DEFAULTS = {
  commission: 0.001, slippage: 0.0, leverage: 1,
  tradeDirection: 'both', strictMode: true,
};

interface DefaultsValues {
  commission: number; slippage: number; leverage: number;
  tradeDirection: string; strictMode: boolean;
}

function loadDefaults(): DefaultsValues | null {
  try { const raw = localStorage.getItem(DEFAULTS_KEY); return raw ? JSON.parse(raw) : null; }
  catch { return null; }
}

function saveDefaults(vals: Record<string, unknown>) {
  try { localStorage.setItem(DEFAULTS_KEY, JSON.stringify(vals)); } catch { /* quota exceeded */ }
}

function removeDefaults() {
  try { localStorage.removeItem(DEFAULTS_KEY); } catch { /* ignore */ }
}

export function useBacktestDefaults(
  t: TFunction,
  current: DefaultsValues,
  onApplyDefaults?: (defaults: DefaultsValues) => void,
) {
  const saved = useMemo(() => loadDefaults(), []);

  const settingsItems: MenuProps['items'] = useMemo(() => [
    {
      key: 'save', label: t(SETTINGS_SAVE_KEY),
      onClick: () => {
        saveDefaults(current);
        message.success(t(DEFAULTS_SAVED_KEY));
      },
    },
    ...(saved ? [{
      key: 'load', label: t(SETTINGS_LOAD_KEY),
      onClick: () => {
        onApplyDefaults?.(saved);
        message.success(t(DEFAULTS_LOADED_KEY));
      },
    }] : []),
    {
      key: 'reset', label: t(SETTINGS_RESET_KEY),
      onClick: () => {
        removeDefaults();
        onApplyDefaults?.(FACTORY_DEFAULTS);
        message.success(t(DEFAULTS_RESET_KEY));
      },
    },
  ], [t, current, saved, onApplyDefaults]);

  return { saved, settingsItems };
}
