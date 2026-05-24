-- name: GetCanonical :one
SELECT canonical FROM broker_symbols WHERE broker = $1 AND symbol_raw = $2 LIMIT 1;

-- name: ListBrokerSymbols :many
SELECT broker, symbol_raw, canonical, digits, point_value, lot_size, lot_step, lot_min, lot_max, trade_mode
FROM broker_symbols
WHERE broker = $1
ORDER BY symbol_raw;
