-- name: GetCanonical :one
SELECT canonical FROM broker_symbols WHERE broker_id = $1 AND symbol_raw = $2 LIMIT 1;

-- name: ListBrokerSymbols :many
SELECT broker_id, symbol_raw, canonical, digits, point, tick_size, contract_size, min_lot, max_lot, lot_step, trade_mode
FROM broker_symbols
WHERE broker_id = $1
ORDER BY symbol_raw;
