-- 199_system_config_admin_visible.up.sql
-- Add admin_visible column to system_config so the backend controls
-- which configs are exposed in the admin UI, replacing the fragile
-- frontend whitelist filter.

ALTER TABLE system_config
    ADD COLUMN IF NOT EXISTS admin_visible BOOLEAN NOT NULL DEFAULT TRUE;

-- Internal/system-level configs that should not be editable from the admin config UI.
-- These are either managed by other admin pages or are internal runtime state.
UPDATE system_config SET admin_visible = FALSE
WHERE key IN (
    'max_positions_per_account',
    'default_leverage',
    'min_lot_size',
    'econ.translation.zhipu_api_key',
    'econ.translation.zhipu_model'
);
