# 09 · PostgreSQL Schema Catalog

> Comprehensive table catalog derived from 88 `.up.sql` migration files.
> Dual-storage boundary: PG is the system-of-record for business entities;
> ClickHouse (`docs/spec/13-clickhouse-schema.md`) handles time-series (ticks, bars, factors, signals).
> Path: `backend/migrations/`

---

## 1. Core Identity

Tables for user accounts, authentication, and access control.

### `users`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Primary user identity and authentication record. |
| Key columns | `id` (UUID PK), `email` (unique), `password_hash`, `role` (user/super_admin/operation/customer_service/audit), `status` (active/disabled) |
| Indexes | `idx_users_email` (unique lookup), `idx_users_status`, `idx_users_role`, `idx_users_email_trgm` (trigram search, 016), `idx_users_nickname_trgm` |
| Altered by | 009 (role index) |

### `api_keys`

| Property | Value |
|---|---|
| Migration | `015_api_keys.up.sql` |
| Purpose | User-scoped API keys for programmatic access with scope-based authorization. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `key_hash` (unique), `scopes` (TEXT[]), `expires_at`, `revoked_at` |
| Indexes | `idx_api_keys_user_id`, `idx_api_keys_revoked_at`, `idx_api_keys_expires_at` |

### `permissions`

| Property | Value |
|---|---|
| Migration | `009_admin_tables.up.sql` |
| Purpose | Permission code registry (RBAC). |
| Key columns | `id` (UUID PK), `code` (unique, e.g. `trade:execute`), `name` |
| Indexes | Implicit unique on `code` |

### `role_permissions`

| Property | Value |
|---|---|
| Migration | `009_admin_tables.up.sql` |
| Purpose | Many-to-many join between roles and permissions. |
| Key columns | `role` (VARCHAR), `permission_id` (FK permissions), composite PK `(role, permission_id)` |
| Indexes | Composite PK |

### `user_jurisdiction`

| Property | Value |
|---|---|
| Migration | `118_jurisdictional_gate.up.sql` |
| Purpose | Per-user KYC status, country jurisdiction, disclaimer acceptance, and risk questionnaire tracking. |
| Key columns | `user_id` (UUID PK FK users), `kyc_status`, `country_code` (ISO 3166-1 alpha-2), `disclaimer_accepted_at`, `risk_score` |
| Indexes | `idx_user_jurisdiction_kyc` |

---

## 2. MT Accounts and Brokers

Tables for MT4/MT5 account management, broker configuration, and symbol normalization.

### `mt_accounts`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | MT4/MT5 trading account bindings -- broker connection credentials, balance snapshot, and stream status. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `mt_type` (mt4/mt5), `broker_company`, `login`, `password`, `broker_host`, `account_status`, `stream_status`, `mt_token`, `broker_id` (FK brokers, added 112) |
| Indexes | `idx_mt_accounts_user`, `idx_mt_accounts_mt_type`, `idx_mt_accounts_status`, `idx_mt_accounts_user_id`, `idx_mt_accounts_user_status`, `idx_mt_accounts_user_type`, `idx_mt_accounts_login_trgm` (trigram), `idx_mt_accounts_created_at`, `idx_mt_accounts_status_type`, `idx_mt_accounts_covering` (covering index) |
| Constraints | `uk_mt_account_login UNIQUE(login, mt_type, broker_server)` (115, was `uk_user_mttype_login` originally) |
| Altered by | 011 (account_type), 014 (indexes), 016 (indexes), 098 (v2 fields: mtapi_port, password_encrypted, mtapi_token_encrypted, canonical_subscribed_symbols), 112 (broker_id FK), 115 (unique constraint change) |
| Views | `mt_accounts_v2` (098, recreated in 112) -- maps v1 fields to v2 naming |

### `brokers`

| Property | Value |
|---|---|
| Migration | `112_brokers.up.sql` |
| Purpose | Admin-configurable broker endpoints -- maps broker code to mtapi.io gateway. |
| Key columns | `id` (UUID PK), `code`, `name`, `platform` (MT4/MT5), `mtapi_endpoint` (empty = use public gateway), `default_server` |
| Indexes | Implicit unique on `(code, platform)` |

### `broker_symbols`

| Property | Value |
|---|---|
| Migration | `088_broker_symbols.up.sql` |
| Purpose | Broker-native symbol list for normalizer resolution. Populated by symbolsync on broker connect. |
| Key columns | `broker_id` (UUID), `symbol_raw` (TEXT), composite PK `(broker_id, symbol_raw)`, `canonical` (FK canonical_symbols), `digits`, `point`, `contract_size`, `min_lot`, `max_lot`, `lot_step`, `trade_mode`, `sessions_quote` (JSONB), `sessions_trade` (JSONB), `raw_payload` (JSONB), `partial`, `tenant_id` (added 117) |
| Indexes | `idx_broker_symbols_canonical`, `idx_broker_symbols_broker` |
| Altered by | 111 (NOTIFY trigger for normalizer cache invalidation), 117 (tenant_id) |
| Note | Migration 099 also creates a `broker_symbols` table with `IF NOT EXISTS` and a different schema (`id` UUID PK, `UNIQUE(broker, symbol_raw)`). Since 088 runs first, 099 is a no-op in production. |

### `canonical_symbols`

| Property | Value |
|---|---|
| Migration | `087_canonical_symbols.up.sql` |
| Purpose | Normalized symbol dictionary -- every broker symbol maps to exactly one canonical. |
| Key columns | `canonical` (TEXT PK), `asset_class` (forex/crypto/commodity/index/stock), `base_ccy`, `quote_ccy`, `description`, `enabled` |
| Indexes | Implicit PK |

### `symbols`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Legacy static symbol reference table (pre-canonical). Superseded by `canonical_symbols` + `broker_symbols`. |
| Key columns | `id` (UUID PK), `symbol` (unique), `description`, `digits`, `contract_size`, `min_lot`, `max_lot`, `lot_step`, `spread`, `swap_long`, `swap_short` |
| Indexes | `idx_symbols_symbol` |

### `market_quotes`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Legacy real-time quote cache (PG-based). Superseded by ClickHouse `md_ticks` / `md_bars`. |
| Key columns | `id` (UUID PK), `symbol` (unique), `bid`, `ask`, `last`, `volume`, `high`, `low` |
| Indexes | `idx_quotes_symbol` |

### `strategy_symbols`

| Property | Value |
|---|---|
| Migration | `089_strategy_symbols.up.sql` |
| Purpose | Strategy-to-canonical-symbol whitelist -- which canonical symbols a strategy trades on. |
| Key columns | `strategy_id` (FK strategies), `canonical` (FK canonical_symbols), composite PK `(strategy_id, canonical)`, `enabled` |
| Indexes | `idx_strategy_symbols_canonical` |

---

## 3. Trading and Orders

Tables for position tracking, order lifecycle, trade history, and paper trading.

### `positions`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Current open positions per MT account. |
| Key columns | `id` (UUID PK), `mt_account_id` (FK mt_accounts), `platform` (MT4/MT5), `ticket` (BIGINT), `symbol`, `order_type` (SMALLINT), `volume`, `open_price`, `current_price`, `stop_loss`, `take_profit`, `open_time`, `profit`, `swap`, `commission` |
| Indexes | `idx_positions_mt_account`, `idx_positions_symbol`, `idx_positions_platform` |
| Constraints | `uk_position_ticket UNIQUE(mt_account_id, ticket)` |

### `orders`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Pending (working) orders per MT account. |
| Key columns | `id` (UUID PK), `mt_account_id` (FK mt_accounts), `platform`, `ticket` (BIGINT), `symbol`, `order_type` (SMALLINT), `volume`, `price`, `stop_limit_price`, `stop_loss`, `take_profit`, `expiration`, `expiration_type`, `state` (VARCHAR, added 091), `broker_symbol_raw` (added 091) |
| Indexes | `idx_orders_mt_account`, `idx_orders_symbol`, `idx_orders_platform`, `idx_orders_state` (added 091) |
| Constraints | `uk_order_ticket UNIQUE(mt_account_id, ticket)` |
| Altered by | 091 (state column with CHECK: NEW/VALIDATED/RISK_APPROVED/SUBMITTED/WORKING/PARTIALLY_FILLED/FILLED/CANCELLED/REJECTED/FAILED/EXPIRED, broker_symbol_raw) |

### `trade_records`

| Property | Value |
|---|---|
| Migration | `004_trade_records.up.sql` |
| Purpose | Historical (closed) trade records -- completed orders with P&L. |
| Key columns | `id` (UUID PK), `account_id` (FK mt_accounts), `ticket` (BIGINT), `symbol`, `order_type`, `volume`, `open_price`, `close_price`, `profit`, `swap`, `commission`, `open_time`, `close_time`, `platform`, `schedule_id` (FK strategy_schedules, added 030) |
| Indexes | `idx_trade_records_account`, `idx_trade_records_symbol`, `idx_trade_records_close_time`, `idx_trade_records_platform`, `idx_trade_records_account_close_time` (014), `idx_trade_records_account_symbol_close_time` (014), `idx_trade_records_schedule_id` (030) |
| Constraints | `uk_trade_record_ticket UNIQUE(account_id, ticket, close_time)` |

### `order_history`

| Property | Value |
|---|---|
| Migration | `012_enhanced_logs.up.sql` |
| Purpose | Extended order history with auto-trade attribution, superseding basic `trade_records` for enriched queries. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts), `ticket`, `order_type`, `symbol`, `volume`, `open_price`, `close_price`, `open_time`, `close_time`, `stop_loss`, `take_profit`, `profit`, `commission`, `swap`, `is_auto_trade`, `schedule_id` (FK strategy_schedules) |
| Indexes | `idx_order_history_user`, `idx_order_history_account`, `idx_order_history_ticket`, `idx_order_history_symbol`, `idx_order_history_open_time`, `idx_order_history_close_time` |

### `trade_logs`

| Property | Value |
|---|---|
| Migration | `002_trade_logs.up.sql` |
| Purpose | Action-level trade log (manual/auto trade operations). |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts), `action` (VARCHAR), `symbol`, `order_type`, `volume`, `price`, `ticket`, `profit`, `message` |
| Indexes | `idx_trade_logs_user`, `idx_trade_logs_account`, `idx_trade_logs_action`, `idx_trade_logs_created_at`, `idx_trade_logs_symbol` |

### `trading_logs`

| Property | Value |
|---|---|
| Migration | `010_auto_trading.up.sql` |
| Purpose | Extended trading log with strategy/execution attribution for auto-trading. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts), `strategy_id` (FK strategies), `execution_id` (FK strategy_executions), `log_type` (trade/signal/error/system), `action`, `symbol`, `details` (JSONB), `message` |
| Indexes | `idx_trading_logs_user`, `idx_trading_logs_account`, `idx_trading_logs_strategy`, `idx_trading_logs_type`, `idx_trading_logs_created` |

### `paper_accounts`

| Property | Value |
|---|---|
| Migration | `113_paper_trading.up.sql` |
| Purpose | Paper (simulated) trading accounts -- isolated from live MT accounts. |
| Key columns | `id` (UUID PK), `user_id`, `name`, `initial_balance` (NUMERIC 20,8), `current_balance`, `equity`, `currency`, `archived` |
| Indexes | None beyond PK |

### `paper_orders`

| Property | Value |
|---|---|
| Migration | `113_paper_trading.up.sql` |
| Purpose | Paper trading orders -- lifecycle states for simulated execution. |
| Key columns | `id` (UUID PK), `paper_account_id` (FK paper_accounts), `strategy_id`, `symbol`, `side` (VARCHAR), `volume` (NUMERIC 10,2), `fill_price`, `slippage_bps`, `state` (VARCHAR, e.g. PENDING) |
| Indexes | `idx_paper_orders_account`, `idx_paper_orders_strategy` |

---

## 4. Market Data (PG-hosted)

Business-metadata and caching tables for market data. Time-series OHLCV/tick data lives in ClickHouse (`md_ticks`, `md_bars`), not PG.

### `kline_data`

| Property | Value |
|---|---|
| Migration | `003_kline_data.up.sql` |
| Purpose | Legacy OHLCV bar storage (PG). Marked for deprecation -- M9 target to DROP after verifying zero business reads. Time-series bars now in ClickHouse `md_bars`. |
| Key columns | `id` (UUID PK), `symbol`, `timeframe`, `open_time`, `close_time`, `open_price`, `high_price`, `low_price`, `close_price`, `tick_volume`, `real_volume`, `spread` |
| Indexes | `idx_kline_symbol`, `idx_kline_timeframe`, `idx_kline_open_time`, `idx_kline_symbol_tf_time`, `idx_kline_symbol_tf_open_time_desc` (014) |
| Constraints | `uk_kline_symbol_tf_time UNIQUE(symbol, timeframe, open_time)` |
| Altered by | 013 (tick_volume, real_volume, spread -- added as DDL but columns already existed), 033 (date column) |

### `kline_cache_metadata`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Metadata for AI agent K-line caches -- tracks cache hits, TTL, fetch windows. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `cache_key` (unique), `source`, `account_id`, `symbol`, `timeframe`, `start_time`, `end_time`, `hit`, `ttl_seconds`, `fetched_at`, `expired_at` |
| Indexes | `idx_kline_cache_metadata_user_symbol` |

### `economic_events`

| Property | Value |
|---|---|
| Migration | `043_economic_events.up.sql` |
| Purpose | Economic calendar events from external sources (FRED, etc.) -- stores actual/previous/estimate values. |
| Key columns | `id` (UUID PK), `source`, `external_id`, `date`, `time`, `country`, `event`, `impact`, `actual`, `previous`, `estimate`, `currency`, `timestamp` (BIGINT, Unix) |
| Indexes | `ux_economic_events_source_external_id_date` (unique), `idx_economic_events_date` |

### `macro_indicator_observations`

| Property | Value |
|---|---|
| Migration | `044_macro_indicator_observations.up.sql` |
| Purpose | Time-series observations for macro indicators (CPI, unemployment, Fed funds, GDP). |
| Key columns | `id` (UUID PK), `indicator_code`, `obs_date` (DATE), `value` (NUMERIC) |
| Indexes | `idx_macro_indicator_obs_indicator_date` |
| Constraints | `UNIQUE(indicator_code, obs_date)` |

---

## 5. Strategies and AI

Tables for strategy definitions, AI-assisted trading, agent definitions, workflows, debates, and jobs.

### `strategies`

| Property | Value |
|---|---|
| Migration | `005_strategies.up.sql` |
| Purpose | User-defined trading strategies -- conditions, actions, risk control as JSONB. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts), `name`, `description`, `symbol`, `conditions` (JSONB), `actions` (JSONB), `risk_control` (JSONB), `status` (active/paused/stopped), `auto_execute` |
| Indexes | `idx_strategies_user_id`, `idx_strategies_account_id`, `idx_strategies_symbol`, `idx_strategies_status` |
| Altered by | 034 (FK targets fix), 090 (canonical migration backfill via strategy_symbols) |

### `strategy_templates`

| Property | Value |
|---|---|
| Migration | `012_strategy_template_refactor.up.sql` |
| Purpose | Reusable strategy code templates -- separates strategy logic (template) from execution config (schedule). |
| Key columns | `id` (UUID PK), `user_id` (FK users), `name`, `description`, `code` (TEXT, the strategy code), `parameters` (JSONB), `is_public`, `tags` (VARCHAR[]), `use_count`, `status` (added 039), `i18n` (JSONB, added 050), `is_system` (BOOLEAN, added 053) |
| Indexes | `idx_strategy_templates_user_id`, `idx_strategy_templates_is_public`, `idx_strategy_templates_tags` (GIN), `idx_strategy_templates_name`, `idx_strategy_templates_status` (039), `idx_strategy_templates_user_system` (053) |
| Altered by | 039 (status column), 040 (TIMESTAMPTZ type), 050 (i18n JSONB), 053 (is_system flag) |

### `strategy_schedules`

| Property | Value |
|---|---|
| Migration | `012_strategy_template_refactor.up.sql` (as `strategy_schedules_v2`), renamed in `027_unify_strategy_schedules.up.sql` and `028_upgrade_strategy_schedules_schema.up.sql`. Original `strategy_schedules` from 010 renamed to `strategy_schedules_legacy`. |
| Purpose | Strategy execution configurations -- binds a template to an account with schedule settings and backtest snapshots. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `template_id` (FK strategy_templates), `account_id` (FK mt_accounts), `name`, `symbol`, `timeframe`, `parameters` (JSONB), `schedule_type` (cron/interval/event), `schedule_config` (JSONB), `backtest_metrics` (JSONB), `risk_score`, `risk_level`, `risk_reasons` (JSONB), `risk_warnings` (JSONB), `last_backtest_at`, `is_active`, `last_run_at`, `next_run_at`, `run_count`, `last_error`, `manual_run_count` (added 031), `last_manual_run_at` (031), `last_manual_error` (031), `enable_count` (added 032) |
| Indexes | `idx_strategy_schedules_user_id`, `idx_strategy_schedules_template_id`, `idx_strategy_schedules_account_id`, `idx_strategy_schedules_symbol`, `idx_strategy_schedules_is_active`, `idx_strategy_schedules_risk_level`, `idx_strategy_schedules_next_run_at`, `idx_strategy_schedules_manual_run_count` (031), `idx_strategy_schedules_enable_count` (032) |
| Altered by | 027 (rename from _v2), 028 (rename + backfill from legacy), 029 (name cleanup), 031 (manual run fields), 032 (enable_count), 034 (FK targets), 038 (unique active constraint), 054 (finalize unification) |

### `strategy_executions`

| Property | Value |
|---|---|
| Migration | `010_auto_trading.up.sql` |
| Purpose | Strategy execution history -- records each run of a schedule with generated signals and executed orders. |
| Key columns | `id` (UUID PK), `strategy_id` (FK strategies), `schedule_id` (FK strategy_schedules), `account_id` (FK mt_accounts), `status` (running/completed/failed/cancelled), `signals` (JSONB), `orders` (JSONB), `error_message`, `started_at`, `completed_at` |
| Indexes | `idx_strategy_executions_strategy`, `idx_strategy_executions_account`, `idx_strategy_executions_status`, `idx_strategy_executions_started` |

### `strategy_signals`

| Property | Value |
|---|---|
| Migration | `005_strategies.up.sql` |
| Purpose | Signals generated by strategy evaluation -- buy/sell/close with price targets. |
| Key columns | `id` (UUID PK), `strategy_id` (FK strategies), `account_id`, `symbol`, `signal_type` (buy/sell/close), `volume` (NUMERIC 18,6), `price`, `stop_loss`, `take_profit`, `reason`, `status` (pending/confirmed/executed/cancelled), `executed_at`, `ticket`, `profit` |
| Indexes | `idx_strategy_signals_strategy_id`, `idx_strategy_signals_account_id`, `idx_strategy_signals_status`, `idx_strategy_signals_created_at` |

### `strategy_execution_logs`

| Property | Value |
|---|---|
| Migration | `012_enhanced_logs.up.sql` |
| Purpose | Detailed execution logs per strategy run -- signal generation, order placement, K-line snapshots. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `schedule_id` (FK strategy_schedules), `template_id` (FK strategy_templates), `account_id` (FK mt_accounts), `symbol`, `timeframe`, `status`, `signal_type`, `signal_price` (NUMERIC 20,8), `signal_volume`, `executed_order_id`, `executed_price`, `profit` (NUMERIC 20,8), `execution_time_ms`, `kline_data` (JSONB), `strategy_params` (JSONB) |
| Indexes | `idx_strategy_exec_logs_user`, `idx_strategy_exec_logs_schedule`, `idx_strategy_exec_logs_account`, `idx_strategy_exec_logs_status`, `idx_strategy_exec_logs_created_at` |

### `strategy_schedule_runtime_states`

| Property | Value |
|---|---|
| Migration | `026_strategy_schedule_runtime_states.up.sql` |
| Purpose | Runtime state blobs for hot-reloadable strategy schedules -- serialized JSON state per schedule. |
| Key columns | `schedule_id` (UUID PK, references strategy_schedules), `state` (JSONB) |
| Indexes | `idx_strategy_schedule_runtime_states_updated_at` |

### `strategy_state`

| Property | Value |
|---|---|
| Migration | `104_strategy_state.up.sql` |
| Purpose | Serialized strategy snapshots for cross-instance state transfer and hot-reload. |
| Key columns | `id` (UUID PK), `instance_id` (TEXT unique), `strategy_name`, `strategy_version` (semver), `account_id` (FK mt_accounts), `user_id`, `state_data` (JSONB), `schema_version`, `status` (active/stopped/error) |
| Indexes | `idx_strategy_state_user`, `idx_strategy_state_name_version`, `idx_strategy_state_active` (partial unique on `instance_id WHERE status = 'active'`) |

### `strategy_models`

| Property | Value |
|---|---|
| Migration | `097_strategy_models.up.sql` |
| Purpose | ONNX model registry for quantengine -- maps strategies to model artifacts. |
| Key columns | `id` (UUID PK), `strategy_id` (FK strategies), `version` (INT), `onnx_uri` (TEXT), `inputs` (JSONB), `outputs` (JSONB), `owner_user_id` (FK users), `active` |
| Indexes | `idx_strategy_models_strategy`, `idx_strategy_models_active` (partial) |
| Constraints | `UNIQUE(strategy_id, version)` |

### `ai_configs`

| Property | Value |
|---|---|
| Migration | `005_ai_configs.up.sql` |
| Purpose | Per-user AI provider configuration (v1). Superseded by `ai_config_profiles`. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `provider` (zhipu/deepseek), `api_key`, `model_name`, `max_tokens`, `temperature`, `is_active`, `base_url` (added 035) |
| Indexes | `idx_ai_configs_user`, `idx_ai_configs_provider`, `idx_ai_configs_active` |
| Constraints | `uk_user_provider UNIQUE(user_id, provider)` |

### `ai_config_profiles`

| Property | Value |
|---|---|
| Migration | `036_ai_config_profiles.up.sql` |
| Purpose | Per-user AI configuration profiles -- supports multiple profiles per user (deep/quick/default roles). |
| Key columns | `id` (UUID PK), `user_id` (FK users), `name`, `provider`, `api_key`, `model_name`, `base_url`, `enabled`, `is_current`, `role` (added 042: deep/quick/default), inference params (added 055) |
| Indexes | `idx_ai_config_profiles_user`, `idx_ai_config_profiles_user_current`, `uk_ai_config_profiles_user_current` (partial unique), `uk_ai_config_profiles_user_name` (unique), `idx_ai_config_profiles_user_role` (042) |
| Altered by | 042 (role), 055 (inference params: temperature, max_tokens, top_p) |

### `system_ai_configs`

| Property | Value |
|---|---|
| Migration | `056_system_ai_configs.up.sql` |
| Purpose | System-level AI provider configurations -- one row per provider with encrypted secrets (AES-GCM). |
| Key columns | `provider_id` (TEXT PK, e.g. openai/anthropic/deepseek), `name`, `base_url`, `organization`, `models` (TEXT[]), `default_model`, `temperature`, `timeout_seconds`, `max_tokens`, `purposes` (TEXT[]), `primary_for` (TEXT[]), `secret_ciphertext` (BYTEA), `secret_salt` (BYTEA), `secret_nonce` (BYTEA), `has_secret`, `enabled`, `user_scope` (added 059) |
| Indexes | `idx_system_ai_enabled`, `idx_system_ai_primary` (GIN) |
| Altered by | 059 (user_scope boolean), 061 (user_ai_primary rework), 062 (timeout_seconds default to 300), 063 (remove gemini row) |

### `ai_conversations`

| Property | Value |
|---|---|
| Migration | `114_ai_conversations.up.sql` (v2, replaces 021) |
| Purpose | AI strategy generation conversations -- conversation threads with state, context, and strategy linkage. |
| Key columns | `id` (UUID PK), `user_id`, `title`, `state` (active/archived), `strategy_id` (UUID), `context_json` (JSONB), `archived_at` |
| Indexes | `idx_ai_conversations_user` |
| Note | 021 v1 had `user_id FK users`, `title`, no state/context fields. 114 v2 adds state, strategy_id, context_json, archived_at. |

### `ai_messages`

| Property | Value |
|---|---|
| Migration | `114_ai_conversations.up.sql` (v2, replaces 021) |
| Purpose | Individual messages within AI conversations -- user/assistant/system roles. |
| Key columns | `id` (UUID PK), `conversation_id` (FK ai_conversations), `role` (user/assistant/system), `content` (TEXT), `metadata_json` (JSONB) |
| Indexes | `idx_ai_messages_conv` |

### `ai_agent_definitions`

| Property | Value |
|---|---|
| Migration | `048_ai_agent_definitions.up.sql` |
| Purpose | Per-user AI agent definitions -- each agent has a type (style/signals/risk/macro/sentiment/portfolio/execution/code), identity, and model profile. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `profile_id` (FK ai_config_profiles), `agent_key` (TEXT), `type` (TEXT, CHECK constraint), `name`, `identity`, `input_hint`, `enabled`, `position` (INT) |
| Indexes | `idx_ai_agent_definitions_user_profile`, `idx_ai_agent_definitions_profile_position` |
| Constraints | `uk_ai_agent_definitions_profile_key UNIQUE(profile_id, agent_key)` |
| Altered by | 057 (model_profile), 058 (model_locator), 060 (collapse locator) |

### `ai_workflow_runs`

| Property | Value |
|---|---|
| Migration | `037_ai_workflow_runs.up.sql` |
| Purpose | AI workflow execution runs -- tracks multi-step AI workflows with status and context. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `title`, `status` (running/succeeded/failed/canceled), `context_json` (TEXT) |
| Indexes | `idx_ai_workflow_runs_user_id`, `idx_ai_workflow_runs_updated_at` |

### `ai_workflow_steps`

| Property | Value |
|---|---|
| Migration | `037_ai_workflow_runs.up.sql` |
| Purpose | Individual steps within an AI workflow run -- input/output/error per step. |
| Key columns | `id` (UUID PK), `run_id` (FK ai_workflow_runs), `step_key`, `title`, `status` (running/done/error), `input` (TEXT), `output` (TEXT), `error` (TEXT), `duration_ms` |
| Indexes | `idx_ai_workflow_steps_run_id`, `idx_ai_workflow_steps_created_at` |

### `debate_sessions`

| Property | Value |
|---|---|
| Migration | `047_debate_sessions.up.sql` |
| Purpose | Multi-agent debate conversations -- tracks agent roster, status flow, and turn references. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `title`, `status` (v1: idle/clarifying/intent_confirm/debating/consensus/code_proposal/saved/archived; v2: `v2:*` prefixed), `agents` (TEXT[]), `current_intent_turn_id`, `current_consensus_turn_id`, `current_code_turn_id`, `template_id`, `param_schema` (JSONB, added 051) |
| Indexes | `idx_debate_sessions_user_id`, `idx_debate_sessions_updated_at` |
| Altered by | 049 (relax status CHECK for v2: prefix), 051 (param_schema JSONB) |

### `debate_turns`

| Property | Value |
|---|---|
| Migration | `047_debate_sessions.up.sql` |
| Purpose | Individual turns within a debate session -- agent opinions, user feedback, consensus, code proposals. |
| Key columns | `id` (UUID PK), `session_id` (FK debate_sessions), `parent_turn_id` (self-referencing FK), `type` (CHECK constraint with v1 + v2 types), `role`, `status` (pending/awaiting_user/approved/rejected/superseded), `content_text` (TEXT), `content_json` (JSONB) |
| Indexes | `idx_debate_turns_session_id`, `idx_debate_turns_created_at`, `idx_debate_turns_session_created` |
| Altered by | 049 (relax type CHECK for v2_ values) |

### `agent_tokens`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Agent API tokens for programmatic access with scoped permissions, rate limits, and paper-only mode. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `name`, `token_prefix`, `token_hash` (unique), `scopes` (TEXT[]), `account_allowlist` (TEXT[]), `symbol_allowlist` (TEXT[]), `paper_only`, `rate_limit_per_min`, `expires_at`, `status` (active/revoked) |
| Indexes | `idx_agent_tokens_user_id`, `idx_agent_tokens_status` |

### `agent_audit_logs`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Audit log for agent API calls -- records every RPC invocation with scope and risk decisions. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `agent_token_id` (FK agent_tokens), `agent_name`, `rpc_service`, `rpc_method`, `scope`, `status_code`, `idempotency_key`, `risk_decision`, `request_summary`, `response_summary`, `duration_ms` |
| Indexes | `idx_agent_audit_logs_user_id_created_at`, `idx_agent_audit_logs_agent_token_id` |

### `jobs`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Async job queue for long-running operations (backtests, experiments, data exports). |
| Key columns | `id` (UUID PK), `user_id` (FK users), `kind` (TEXT), `status` (TEXT), `progress` (DOUBLE PRECISION), `stage`, `request_summary`, `result_ref`, `result_summary`, `error_code`, `error_message`, `idempotency_key`, `started_at`, `finished_at`, `expires_at` |
| Indexes | `idx_jobs_user_id_created_at`, `idx_jobs_status`, `idx_jobs_user_kind_idempotency` (partial unique) |

### `job_events`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Event stream for jobs -- sequential progress/status events emitted during job execution. |
| Key columns | `id` (UUID PK), `job_id` (FK jobs), `seq` (BIGINT), `type`, `status`, `progress`, `stage`, `message`, `payload` (JSONB) |
| Indexes | `idx_job_events_job_id_seq` |
| Constraints | `UNIQUE(job_id, seq)` |

### `strategy_experiments`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Parameter optimization experiments -- grid/random search over strategy parameter space. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `base_template_id`, `status`, `parameter_space` (JSONB), `search_method` (grid/random), `max_candidates`, `objective`, `market_regime_ref`, `best_candidate_id`, `job_id` (FK jobs) |
| Indexes | `idx_strategy_experiments_user_id_created_at` |

### `strategy_experiment_candidates`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Individual candidate parameter sets within a strategy experiment -- scored and ranked. |
| Key columns | `id` (UUID PK), `experiment_id` (FK strategy_experiments), `parameters` (JSONB), `draft_code_ref`, `backtest_run_id`, `score` (DOUBLE PRECISION), `grade`, `score_components` (JSONB), `rank` (INT), `summary`, `recommendation` |
| Indexes | `idx_strategy_experiment_candidates_experiment_id_rank` |

### `strategy_assets` (publish registry)

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Published strategy assets -- owner-controlled versions with visibility and review status. |
| Key columns | `id` (UUID PK), `owner_user_id` (FK users), `source_template_id`, `source_version`, `name`, `description`, `visibility` (private/public), `review_status` (pending/approved/rejected), `rating_summary`, `clone_count`, `latest_version` |
| Indexes | `idx_strategy_assets_owner_user_id`, `idx_strategy_assets_visibility_review_status` |

### `strategy_asset_clones`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Records of user clones of published strategy assets -- tracks sync availability. |
| Key columns | `id` (UUID PK), `asset_id` (FK strategy_assets), `user_id` (FK users), `cloned_template_id`, `source_version`, `sync_available`, `last_sync_check_at` |
| Indexes | `idx_strategy_asset_clones_user_id`, `idx_strategy_asset_clones_asset_id` |

### `paper_strategies`

| Property | Value |
|---|---|
| Migration | `113_paper_trading.up.sql` |
| Purpose | Paper trading strategy definitions with DSL code and optional backtest metrics. |
| Key columns | `id` (UUID PK), `user_id`, `name`, `description`, `category`, `dsl_code` (TEXT), `backtest_metrics` (JSONB), `promoted_at`, `archived` |
| Indexes | `idx_paper_strategies_user` |

---

## 6. Backtest

Tables for backtest datasets, runs, and tick-driven backtests.

### `backtest_datasets`

| Property | Value |
|---|---|
| Migration | `017_backtest_datasets.up.sql` |
| Purpose | Frozen (snapshot) datasets for reproducible backtests -- captures bar ranges with cost model. |
| Key columns | `id` (UUID PK), `user_id`, `account_id`, `symbol`, `timeframe`, `from_time`, `to_time`, `count`, `frozen`, `cost_model_snapshot` (JSONB, added 019) |
| Indexes | `idx_backtest_datasets_user`, `idx_backtest_datasets_account`, `idx_backtest_datasets_symbol_tf` |
| Altered by | 019 (cost_model_snapshot) |

### `backtest_dataset_bars`

| Property | Value |
|---|---|
| Migration | `017_backtest_datasets.up.sql` |
| Purpose | Bar-level data within a frozen backtest dataset -- OHLCV rows keyed by dataset. |
| Key columns | `dataset_id` (FK backtest_datasets), `symbol`, `timeframe`, `open_time`, `close_time`, `open_price` (NUMERIC 18,8), `high_price`, `low_price`, `close_price`, `tick_volume` |
| Indexes | `idx_backtest_dataset_bars_dataset_time` |
| Constraints | `fk_backtest_dataset_bars_dataset`, `uk_backtest_dataset_bars_unique UNIQUE(dataset_id, open_time)` |

### `backtest_runs`

| Property | Value |
|---|---|
| Migration | `020_backtest_runs.up.sql` |
| Purpose | Backtest execution records -- strategy code hash, metrics, equity curve, and async lifecycle. |
| Key columns | `id` (UUID PK), `user_id`, `account_id`, `symbol`, `timeframe`, `dataset_id`, `strategy_code_hash`, `python_service_version`, `cost_model_snapshot` (JSONB), `metrics` (JSONB), `equity_curve` (JSONB), `status` (added 023), `error` (023), `started_at` (023), `finished_at` (023), `strategy_code` (023), `initial_capital` (023), `mode` (KLINE_RANGE/TICK_RANGE, added 024), `from_ts` (024), `to_ts` (024), `cancel_requested_at` (025), `lease_until` (025), `template_id` (UUID, added 041), `template_draft_id` (041), `extra_symbols` (TEXT[], added 052) |
| Indexes | `idx_backtest_runs_user`, `idx_backtest_runs_account`, `idx_backtest_runs_dataset`, `idx_backtest_runs_user_created_at` (023), `idx_backtest_runs_account_created_at` (023), `idx_backtest_runs_status` (023), `idx_backtest_runs_mode` (024), `idx_backtest_runs_range` (024), `idx_backtest_runs_cancel_requested_at` (025), `idx_backtest_runs_lease_until` (025), `idx_backtest_runs_template_id` (041), `idx_backtest_runs_template_draft_id` (041) |
| Altered by | 023 (async fields), 024 (mode/range), 025 (cancel/lease), 040 (TIMESTAMPTZ), 041 (template refs), 052 (extra_symbols) |

### `tick_datasets`

| Property | Value |
|---|---|
| Migration | `018_tick_datasets.up.sql` |
| Purpose | Tick-level datasets for reproducible tick-driven backtests. |
| Key columns | `id` (UUID PK), `user_id`, `account_id`, `symbol`, `from_time`, `to_time`, `frozen` |
| Indexes | `idx_tick_datasets_user`, `idx_tick_datasets_account`, `idx_tick_datasets_symbol` |

### `tick_dataset_ticks`

| Property | Value |
|---|---|
| Migration | `018_tick_datasets.up.sql` |
| Purpose | Individual tick records within a tick dataset -- bid/ask per timestamp. |
| Key columns | `dataset_id` (FK tick_datasets), `time` (TIMESTAMP), `bid` (NUMERIC 18,8), `ask` (NUMERIC 18,8) |
| Indexes | `idx_tick_dataset_ticks_dataset_time` |
| Constraints | `fk_tick_dataset_ticks_dataset`, `uk_tick_dataset_ticks_unique UNIQUE(dataset_id, time)` |

---

## 7. Platform (ADR-0006)

Tables for the C2C platform: shared platform resources and user-private overrides.

### `platform_strategies`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Platform-shared strategy definitions -- all users can read/subscribe. |
| Key columns | `id` (UUID PK), `name`, `description`, `expression` (TEXT), `canonicals` (TEXT[]), `periods` (TEXT[]), `category`, `is_official`, `published_by` (FK users), `tenant_id` (added 117) |
| Indexes | None beyond PK |
| Altered by | 117 (tenant_id) |

### `platform_factors`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Platform-shared factor definitions -- reusable factor expressions. |
| Key columns | `id` (UUID PK), `name` (unique), `expression` (TEXT), `description`, `canonicals` (TEXT[]), `periods` (TEXT[]), `lookback` (INT), `category` (builtin/custom) |
| Indexes | Implicit unique on `name` |

### `platform_ai_agents`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Platform-shared AI agent definitions -- default agents available to all users. |
| Key columns | `id` (UUID PK), `name` (unique), `role`, `system_prompt` (TEXT), `model`, `temperature`, `is_default` |
| Indexes | Implicit unique on `name` |

### `admins`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Platform admin user registry -- maps user IDs to admin scopes. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `scope` (default: `platform:admin`) |
| Indexes | None beyond PK |

### `user_subscriptions`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | User subscriptions to other users' strategies or copy-trade links. |
| Key columns | `id` (UUID PK), `subscriber_user_id` (FK users), `target_user_id` (FK users), `target_strategy_id`, `kind` (strategy/copy_trade), `active` |
| Indexes | Implicit unique on `(subscriber_user_id, target_strategy_id, kind)` |

### `user_strategy_publishes`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Records of user strategy publications to the platform with royalty settings. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `platform_strategy_id` (FK platform_strategies), `royalty_pct` (INT) |
| Indexes | Implicit unique on `(user_id, platform_strategy_id)` |

### `copy_trade_links`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Copy-trade configuration -- links a subscription to specific account pairs with ratio and limits. |
| Key columns | `id` (UUID PK), `subscription_id` (FK user_subscriptions), `from_account_id` (UUID), `to_account_id` (UUID), `ratio` (DOUBLE PRECISION), `max_lots`, `active` |
| Indexes | None beyond PK |

### `user_factor_overrides`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Per-user factor expression overrides -- user can customize factor parameters. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `factor_name`, `expression`, `canonicals` (TEXT[]), `periods` (TEXT[]), `lookback` (INT) |
| Indexes | Implicit unique on `(user_id, factor_name)` |

### `user_ai_agents`

| Property | Value |
|---|---|
| Migration | `110_platform_tables.up.sql` |
| Purpose | Per-user AI agent customizations -- user-specific system prompts and model settings. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `agent_name`, `system_prompt` (TEXT), `model`, `temperature` |
| Indexes | Implicit unique on `(user_id, agent_name)` |

---

## 8. Risk and Compliance

Tables for risk management, jurisdictional gate, and tenant configuration.

### `risk_configs`

| Property | Value |
|---|---|
| Migration | `010_auto_trading.up.sql` |
| Purpose | Per-user/account risk configuration -- position limits, daily loss, drawdown, trailing stop. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts, NULL = global), `max_risk_percent`, `max_daily_loss`, `max_drawdown_percent`, `max_positions`, `max_lot_size`, `daily_loss_used`, `trailing_stop_enabled`, `trailing_stop_pips` |
| Indexes | `idx_risk_configs_user`, `idx_risk_configs_account` |
| Constraints | `uk_user_account_risk UNIQUE(user_id, account_id)` |

### `user_risk_profiles`

| Property | Value |
|---|---|
| Migration | `093_user_risk_profiles.up.sql` |
| Purpose | Per-user risk profile -- strategies inherit owner's profile. Replaces Strategy.RiskControl JSONB. |
| Key columns | `user_id` (UUID PK FK users), `max_positions`, `daily_loss_limit` (NUMERIC 18,2), `max_drawdown_pct` (NUMERIC 5,2), `margin_level_min`, `session_enabled`, `slippage_max_points`, `reject_rate_max` (NUMERIC 5,2), `heartbeat_timeout_s`, `killswitch_enabled`, `symbol_whitelist` (TEXT[]), `capability_tier` (INT, 0-3, added 116), `order_types_allowed` (TEXT[], 116), `lot_per_order_max` (116), `daily_order_max` (116), `leverage_max` (116) |
| Indexes | Implicit PK |
| Altered by | 094 (data migration from strategies.risk_control), 116 (capability tier + order constraints) |

### `risk_events`

| Property | Value |
|---|---|
| Migration | `092_risk_events.up.sql` |
| Purpose | Immutable audit log for every risk rule evaluation -- compliance record. |
| Key columns | `id` (UUID PK), `order_id` (FK orders), `account_id`, `rule_name` (e.g. max_position/daily_loss/margin), `result` (PASS/BLOCK/WARN), `detail` (JSONB) |
| Indexes | `idx_risk_events_order`, `idx_risk_events_account`, `idx_risk_events_created` |

### `sanctioned_countries`

| Property | Value |
|---|---|
| Migration | `118_jurisdictional_gate.up.sql` |
| Purpose | Admin-managed list of sanctioned countries (ISO 3166-1 alpha-2). |
| Key columns | `country_code` (VARCHAR(2) PK), `label`, `added_by` (FK users) |
| Indexes | Implicit PK |

### `tenant_config`

| Property | Value |
|---|---|
| Migration | `117_tenant_id_and_config.up.sql` |
| Purpose | Multi-tenant key/value configuration store -- versioned, with PG NOTIFY for hot-reload. |
| Key columns | `id` (BIGSERIAL PK), `tenant_id`, `config_key`, `config_value`, `version` (INT), `updated_by` |
| Indexes | `idx_tenant_config_tenant`, `UNIQUE(tenant_id, config_key)` |

---

## 9. Marketplace

Tables for the strategy marketplace.

### `marketplace_strategies`

| Property | Value |
|---|---|
| Migration | `095_marketplace_strategies.up.sql` |
| Purpose | User-published strategies for community sharing -- pricing model, asset class, tags, subscriber stats. |
| Key columns | `id` (UUID PK), `strategy_id` (FK strategies), `publisher_id` (FK users), `title`, `description`, `price_model` (free/monthly/once), `price_amount`, `asset_class`, `symbols` (TEXT[]), `timeframe`, `risk_level` (low/medium/high), `tags` (TEXT[]), `total_subscribers`, `total_pnl`, `win_rate`, `status` (published/hidden/removed) |
| Indexes | `idx_marketplace_status`, `idx_marketplace_publisher`, `idx_marketplace_asset_class` |

---

## 10. Factor / Signal

Tables for factor definitions and signal tracking. Factor values and signals are stored in ClickHouse (`factor_values`, `signals`); PG stores definitions and configurations only.

### `factor_definitions`

| Property | Value |
|---|---|
| Migration | `100_factor_definitions_v2.up.sql` (replaces 096 v1) |
| Purpose | DSL-based factor definitions -- each row defines a named factor expression with applicable canonicals, periods, and lookback. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `name`, `expression` (TEXT), `canonicals` (TEXT[]), `periods` (TEXT[]), `lookback` (INT, default 100), `is_active` |
| Indexes | Implicit unique on `(user_id, name)` |
| Note | 096 v1 had `owner_user_id`, `name UNIQUE`, `symbols TEXT[]`, no canonicals/periods/lookback. 100 v2 DROPs v1 and creates with `user_id FK`, `UNIQUE(user_id, name)`, `canonicals`, `periods`, `lookback`. 5 built-in factors seeded in 096: momentum_5, volatility_20, rsi_14, ma_ratio, bb_position. |

### `strategy_signals`

See Section 5 (Strategies and AI). Strategy-generated trading signals stored in PG for relational queries; ClickHouse `signals` table stores the time-series audit trail.

---

## 11. Admin and System

Tables for operational logging, system configuration, and global settings.

### `admin_logs`

| Property | Value |
|---|---|
| Migration | `009_admin_tables.up.sql` |
| Purpose | Admin operation audit log -- records admin actions with IP, user agent, and result. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `module`, `action_type`, `target_type`, `target_id`, `ip_address`, `user_agent`, `request_method`, `request_path`, `details` (JSONB), `success` (BOOLEAN), `error_message` |
| Indexes | `idx_admin_logs_user`, `idx_admin_logs_module`, `idx_admin_logs_created` (DESC), `idx_admin_logs_action_type` |

### `system_operation_logs`

| Property | Value |
|---|---|
| Migration | `012_enhanced_logs.up.sql` |
| Purpose | System-level operation logs -- resource changes with old/new value snapshots. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `operation_type`, `module`, `resource_type`, `resource_id` (UUID), `action`, `old_value` (JSONB), `new_value` (JSONB), `ip_address`, `user_agent`, `status`, `error_message`, `duration_ms` |
| Indexes | `idx_system_op_logs_user`, `idx_system_op_logs_operation`, `idx_system_op_logs_module`, `idx_system_op_logs_resource`, `idx_system_op_logs_created_at` |

### `account_connection_logs`

| Property | Value |
|---|---|
| Migration | `012_enhanced_logs.up.sql` |
| Purpose | MT account connection event logs -- connect/disconnect with duration and error details. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id` (FK mt_accounts), `event_type`, `status`, `message`, `error_detail`, `server_host`, `server_port`, `login_id`, `connection_duration_seconds` |
| Indexes | `idx_account_conn_logs_user`, `idx_account_conn_logs_account`, `idx_account_conn_logs_event_type`, `idx_account_conn_logs_created_at` |

### `system_config`

| Property | Value |
|---|---|
| Migration | `001_init.up.sql` |
| Purpose | Simple key/value system configuration store. Seeded with defaults (max_accounts_per_user, default_leverage, etc.). |
| Key columns | `key` (VARCHAR PK), `value` (TEXT), `description`, `enabled` (BOOLEAN, added 022) |
| Indexes | Implicit PK |
| Altered by | 022 (enabled column) |

### `global_settings`

| Property | Value |
|---|---|
| Migration | `010_auto_trading.up.sql` |
| Purpose | Per-user global settings -- auto-trade toggle and notification preferences. |
| Key columns | `id` (UUID PK), `user_id` (FK users, unique), `auto_trade_enabled`, `notification_enabled`, `email_notification`, `sms_notification` |
| Indexes | `idx_global_settings_user` |

### `market_regimes`

| Property | Value |
|---|---|
| Migration | `064_agent_job_experiment_assets.up.sql` |
| Purpose | Detected market regime segments -- ML-classified market states with confidence and features. |
| Key columns | `id` (UUID PK), `user_id` (FK users), `account_id`, `symbol`, `timeframe`, `regime` (TEXT), `confidence` (DOUBLE PRECISION), `features` (JSONB), `segments` (JSONB), `strategy_families` (TEXT[]), `from_time` (TIMESTAMPTZ), `to_time` (TIMESTAMPTZ), `model_version` |
| Indexes | `idx_market_regimes_user_symbol_created_at` |

### `event_translations`

| Property | Value |
|---|---|
| Migration | `045_event_translations.up.sql` |
| Purpose | Localized translations for external economic event titles (e.g. FRED releases). |
| Key columns | `id` (UUID PK), `source`, `original_text`, `lang` (VARCHAR 16), `translated_text`, `provider` |
| Indexes | `idx_event_translations_source_lang` |
| Constraints | `UNIQUE(source, original_text, lang)` |

### `schema_migrations`

| Property | Value |
|---|---|
| Migration | `003_kline_data.up.sql` |
| Purpose | Migration version tracker -- records which migrations have been applied. |
| Key columns | `version` (VARCHAR PK), `applied_at` |
| Indexes | Implicit PK |

---

## 12. Migration Index

Maps every migration file to the tables it creates or alters. ALTER-only migrations are noted as such.

| # | Migration Name | Tables Created | Tables Altered |
|---|---|---|---|
| 001 | init | `users`, `mt_accounts`, `positions`, `orders`, `symbols`, `market_quotes`, `system_config` | -- |
| 002 | trade_logs | `trade_logs` | -- |
| 003 | kline_data | `kline_data`, `schema_migrations` | -- |
| 004 | trade_records | `trade_records` | -- |
| 005 | ai_configs | `ai_configs` | -- |
| 005 | strategies | `strategies`, `strategy_signals` | -- |
| 008 | fix_order_type_length | -- | `positions`, `orders` (type length) |
| 009 | admin_tables | `permissions`, `role_permissions`, `admin_logs` | `users` (role index) |
| 010 | auto_trading | `strategy_schedules`, `strategy_executions`, `risk_configs`, `global_settings`, `trading_logs` | -- |
| 011 | account_type | -- | `mt_accounts` (account_type) |
| 012 | enhanced_logs | `account_connection_logs`, `strategy_execution_logs`, `order_history`, `system_operation_logs` | `strategy_execution_logs` (FK to strategy_schedules_v2, strategy_templates); `order_history` (FK to strategy_schedules_v2) |
| 012 | strategy_template_refactor | `strategy_templates`, `strategy_schedules_v2` | -- |
| 013 | add_tick_volume | -- | `kline_data` (tick_volume, real_volume, spread) |
| 014 | stream_perf_indexes | -- | `mt_accounts`, `trade_records`, `kline_data` (performance indexes) |
| 015 | api_keys | `api_keys` | -- |
| 016 | account_management_indexes | -- | `mt_accounts`, `users` (trigram + covering indexes) |
| 017 | backtest_datasets | `backtest_datasets`, `backtest_dataset_bars` | -- |
| 018 | tick_datasets | `tick_datasets`, `tick_dataset_ticks` | -- |
| 019 | backtest_dataset_cost_snapshot | -- | `backtest_datasets` (cost_model_snapshot) |
| 020 | backtest_runs | `backtest_runs` | -- |
| 021 | ai_conversations (v1) | `ai_conversations`, `ai_messages` | -- |
| 022 | system_config_enabled | -- | `system_config` (enabled) |
| 023 | backtest_runs_async_fields | -- | `backtest_runs` (status, error, started_at, finished_at, strategy_code, initial_capital + indexes) |
| 024 | backtest_runs_mode_range | -- | `backtest_runs` (mode, from_ts, to_ts + indexes) |
| 025 | backtest_runs_worker_cancel | -- | `backtest_runs` (cancel_requested_at, lease_until + indexes) |
| 026 | strategy_schedule_runtime_states | `strategy_schedule_runtime_states` | -- |
| 027 | unify_strategy_schedules | -- | `strategy_schedules_v2` RENAME TO `strategy_schedules`; old `strategy_schedules` RENAME TO `strategy_schedules_legacy` |
| 028 | upgrade_strategy_schedules_schema | `strategy_schedules` (fallback CREATE if not exists) | Renames _v2 to strategy_schedules; backfills from legacy; adds indexes |
| 029 | cleanup_strategy_schedules_names | -- | `strategy_schedules` (name dedup) |
| 030 | trade_records_add_schedule_id | -- | `trade_records` (schedule_id FK + index) |
| 031 | strategy_schedules_add_manual_run_fields | -- | `strategy_schedules` (manual_run_count, last_manual_run_at, last_manual_error + index) |
| 032 | strategy_schedules_add_enable_count | -- | `strategy_schedules` (enable_count + index) |
| 033 | add_kline_date_column | -- | `kline_data` (date column) |
| 034 | fix_schedule_fk_targets | -- | `strategy_schedules` FK, `strategies` FK |
| 035 | ai_configs_add_base_url | -- | `ai_configs` (base_url) |
| 036 | ai_config_profiles | `ai_config_profiles` | Seeds profiles from `ai_configs` |
| 037 | ai_workflow_runs | `ai_workflow_runs`, `ai_workflow_steps` | -- |
| 038 | strategy_schedules_unique_active | -- | `strategy_schedules` (partial unique index on active) |
| 039 | strategy_templates_add_status | -- | `strategy_templates` (status + index) |
| 040 | timestamptz_utc | -- | `strategy_templates`, `backtest_runs` (column type to TIMESTAMPTZ) |
| 041 | backtest_runs_template_ref | -- | `backtest_runs` (template_id, template_draft_id + indexes) |
| 042 | ai_config_profiles_add_role | -- | `ai_config_profiles` (role + index) |
| 043 | economic_events | `economic_events` | -- |
| 044 | macro_indicator_observations | `macro_indicator_observations` | -- |
| 045 | event_translations | `event_translations` | -- |
| 046 | system_config_econ_translation | -- | `system_config` (economic translation settings) |
| 047 | debate_sessions | `debate_sessions`, `debate_turns` | -- |
| 048 | ai_agent_definitions | `ai_agent_definitions` | -- |
| 049 | debate_v2_status_types | -- | `debate_sessions` (relax status CHECK), `debate_turns` (relax type CHECK) |
| 050 | strategy_templates_i18n | -- | `strategy_templates` (i18n JSONB) |
| 051 | debate_sessions_param_schema | -- | `debate_sessions` (param_schema JSONB) |
| 052 | backtest_runs_extra_symbols | -- | `backtest_runs` (extra_symbols TEXT[]) |
| 053 | strategy_templates_is_system | -- | `strategy_templates` (is_system + index) |
| 054 | finalize_strategy_schedules_unification | -- | `strategy_schedules` (finalize migration, cleanup legacy) |
| 055 | ai_config_profiles_inference_params | -- | `ai_config_profiles` (inference parameters) |
| 056 | system_ai_configs | `system_ai_configs` | Seeds 7 default provider rows |
| 057 | ai_agent_model_profile | -- | `ai_agent_definitions` (model_profile) |
| 058 | ai_agent_model_locator | -- | `ai_agent_definitions` (model_locator) |
| 059 | system_ai_configs_user_scope | -- | `system_ai_configs` (user_scope) |
| 060 | collapse_ai_agent_locator | -- | `ai_agent_definitions` (collapse locator fields) |
| 061 | user_ai_primary | -- | `system_ai_configs` (primary_for rework) |
| 062 | ai_http_timeout_300 | -- | `system_ai_configs` (timeout_seconds default 300) |
| 063 | remove_gemini_system_ai | -- | `system_ai_configs` (DELETE gemini row) |
| 064 | agent_job_experiment_assets | `agent_tokens`, `agent_audit_logs`, `jobs`, `job_events`, `strategy_experiments`, `strategy_experiment_candidates`, `strategy_assets`, `strategy_asset_clones`, `market_regimes`, `kline_cache_metadata` | -- |
| 087 | canonical_symbols | `canonical_symbols` | -- |
| 088 | broker_symbols | `broker_symbols` | -- |
| 089 | strategy_symbols | `strategy_symbols` | -- |
| 090 | strategies_canonical_migration | -- | `strategy_symbols` (backfill from strategies.symbol) |
| 091 | orders_state_machine | -- | `orders` (state, broker_symbol_raw + index) |
| 092 | risk_events | `risk_events` | -- |
| 093 | user_risk_profiles | `user_risk_profiles` | -- |
| 094 | risk_control_migration | -- | `user_risk_profiles` (seed from strategies.risk_control JSONB) |
| 095 | marketplace_strategies | `marketplace_strategies` | -- |
| 096 | factor_definitions (v1) | `factor_definitions` (v1 schema) | Seeds 5 built-in factors |
| 097 | strategy_models | `strategy_models` | -- |
| 098 | mt_accounts_v2_fields | -- | `mt_accounts` (mtapi_port, mtapi_token_encrypted, password_encrypted, canonical_subscribed_symbols); creates view `mt_accounts_v2` |
| 099 | broker_symbols (duplicate) | `broker_symbols` (IF NOT EXISTS -- no-op if 088 ran first) | -- |
| 100 | factor_definitions_v2 | `factor_definitions` (v2, replaces 096) | DROPs v1 factor_definitions CASCADE |
| 104 | strategy_state | `strategy_state` | -- |
| 110 | platform_tables | `platform_strategies`, `platform_factors`, `platform_ai_agents`, `admins`, `user_subscriptions`, `user_strategy_publishes`, `copy_trade_links`, `user_factor_overrides`, `user_ai_agents` | -- |
| 111 | broker_symbols_notify | -- | `broker_symbols` (PG NOTIFY trigger) |
| 112 | brokers | `brokers` | `mt_accounts` (broker_id FK); seeds brokers from mt_accounts; recreates `mt_accounts_v2` view |
| 113 | paper_trading | `paper_accounts`, `paper_orders`, `paper_strategies` | -- |
| 114 | ai_conversations (v2) | `ai_conversations`, `ai_messages` (v2 schema, supersedes 021) | -- |
| 115 | mt_accounts_unique_login | -- | `mt_accounts` (replace uk_user_mttype_login with uk_mt_account_login) |
| 116 | user_risk_profiles_capabilities | -- | `user_risk_profiles` (capability_tier, order_types_allowed, lot_per_order_max, daily_order_max, leverage_max) |
| 117 | tenant_id_and_config | `tenant_config` | `broker_symbols` (tenant_id), `signal_configs` (if exists), `platform_strategies` (tenant_id) |
| 118 | jurisdictional_gate | `user_jurisdiction`, `sanctioned_countries` | Seeds sanctioned countries |

---

## 13. Cross-Reference: ClickHouse Schema

ClickHouse stores time-series data that PG is not suited for. See `docs/spec/13-clickhouse-schema.md` for full DDL.

| Data Category | PostgreSQL (this doc) | ClickHouse | Rationale |
|---|---|---|---|
| Users, accounts, orders | `users`, `mt_accounts`, `orders` | -- | Relational integrity, FK constraints |
| Ticks | -- | `md_ticks` | 5M+ rows/account/day, 90-day TTL |
| OHLCV Bars | `kline_data` (legacy, M9 deprecation) | `md_bars` | Long-term retention, no TTL |
| Factor Values | -- | `factor_values` | 50K rows/account/day, 2-year TTL |
| Factor Definitions | `factor_definitions` | -- | Business editable, relational queries |
| Signals (audit) | `strategy_signals` (PG business logic) | `signals` (CH time-series) | PG for relational; CH for high-frequency audit trail |
| Broker Symbols | `broker_symbols`, `canonical_symbols` | -- | Configuration, FK relationships |
| AI Configs | `ai_configs`, `ai_config_profiles`, `system_ai_configs` | -- | Relational, encrypted secrets |
| Debates, Conversations | `debate_sessions`, `ai_conversations` | -- | Relational, JSONB content |
| Backtest Data | `backtest_datasets`, `backtest_runs` | -- | Metadata; actual bar data in `backtest_dataset_bars` (PG) or re-fetched from CH |
