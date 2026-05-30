-- name: GetAccount :one
SELECT * FROM mt_accounts WHERE id = $1 AND user_id = $2;

-- name: ListAccounts :many
SELECT * FROM mt_accounts WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetAccountCredentials :one
SELECT login, password, mt_type, broker_host FROM mt_accounts WHERE id = $1 AND user_id = $2;

-- name: UserOwnsAccount :one
SELECT EXISTS(SELECT 1 FROM mt_accounts WHERE id = $1 AND user_id = $2) AS owns;

-- name: UpdateAccountMetrics :exec
UPDATE mt_accounts SET balance = $3, equity = $4, credit = $5, margin = $6, free_margin = $7, margin_level = $8, account_status = 'connected', updated_at = CURRENT_TIMESTAMP WHERE id = $1 AND user_id = $2;

-- name: GetAccountSnapshots :many
SELECT id, balance, equity, credit, margin, free_margin, margin_level, account_status FROM mt_accounts WHERE user_id = $1;
