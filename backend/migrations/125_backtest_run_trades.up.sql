-- 125_backtest_run_trades.up.sql
-- Per-trade records for backtest runs, keyed by (run_id, ticket).
-- Python backtest runner writes these; Go ListBacktestRunTrades handler reads them.

CREATE TABLE IF NOT EXISTS backtest_run_trades (
    run_id     UUID NOT NULL REFERENCES backtest_runs(id) ON DELETE CASCADE,
    ticket     BIGINT NOT NULL,
    side       VARCHAR(12) NOT NULL,
    volume     DOUBLE PRECISION NOT NULL,
    open_ts    BIGINT NOT NULL,
    open_price DOUBLE PRECISION NOT NULL,
    close_ts   BIGINT NOT NULL DEFAULT 0,
    close_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    pnl        DOUBLE PRECISION NOT NULL DEFAULT 0,
    commission DOUBLE PRECISION NOT NULL DEFAULT 0,
    reason     VARCHAR(256) NOT NULL DEFAULT '',
    PRIMARY KEY (run_id, ticket)
);
