-- name: ListPlatformStrategies :many
SELECT id, name, description, expression, category FROM platform_strategies WHERE is_official = true ORDER BY name;

-- name: GetPlatformFactor :one
SELECT id, name, expression, canonicals, periods, lookback FROM platform_factors WHERE name = $1 LIMIT 1;

-- name: ListUserSubscriptions :many
SELECT id, target_user_id, target_strategy_id, kind FROM user_subscriptions WHERE subscriber_user_id = $1 AND active = true;
