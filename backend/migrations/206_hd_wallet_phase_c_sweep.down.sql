-- 206_hd_wallet_phase_c_sweep.down.sql

DROP TABLE IF EXISTS sweep_bundles;

DROP INDEX IF EXISTS idx_sweep_logs_batch_id;
ALTER TABLE sweep_logs DROP COLUMN IF EXISTS batch_id;
ALTER TABLE sweep_logs DROP COLUMN IF EXISTS leg_type;
ALTER TABLE sweep_logs DROP COLUMN IF EXISTS leg_seq;
