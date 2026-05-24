-- name: ListFactorDefinitions :many
SELECT id, user_id, name, expression, canonicals, periods, lookback, is_active
FROM factor_definitions
WHERE user_id = $1 AND is_active = true
ORDER BY name;

-- name: GetFactorDefinition :one
SELECT id, user_id, name, expression, canonicals, periods, lookback, is_active
FROM factor_definitions
WHERE id = $1 LIMIT 1;
