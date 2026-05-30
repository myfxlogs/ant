package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Service implements the C2C marketplace (strategy publish + subscribe).
// M12-B1: unified model — Publish writes to both user_strategy_publishes
// and marketplace_strategies; ListPublished JOINs both for rich metadata.
type Service struct {
	pg *pgxpool.Pool
}

// New creates a marketplace service.
func New(pg *pgxpool.Pool) *Service {
	return &Service{pg: pg}
}

// PublishParams carries the full strategy metadata for publishing.
type PublishParams struct {
	UserID      string
	StrategyID  string
	Title       string
	Description string
	PriceModel  string
	PriceAmount float64
	AssetClass  string
	Symbols     []string
	Timeframe   string
	RiskLevel   string
	Tags        []string
}

// PublishedStrategy represents a strategy listed in the marketplace
// with full metadata from marketplace_strategies (M12-B1).
type PublishedStrategy struct {
	PublishID        string
	StrategyID       string
	StrategyName     string
	PublisherUserID  string
	PublishedAt      time.Time
	Title            string
	Description      string
	PriceModel       string
	PriceAmount      *float64
	AssetClass       string
	Symbols          []string
	Timeframe        *string
	RiskLevel        string
	Tags             []string
	TotalSubscribers int
	WinRate          *float64
	TotalPnL         *float64
}

// SubscriptionItem represents a user's subscription.
type SubscriptionItem struct {
	SubscriptionID string
	TargetUserID   string
	StrategyID     string
	Kind           string
	Active         bool
	CreatedAt      time.Time
}

// Publish adds a strategy to the marketplace. Writes to both
// user_strategy_publishes (ownership tracking) and marketplace_strategies
// (rich listing metadata). Uses a transaction for atomicity.
func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	publishID := uuid.New()
	stratID := uuid.New()

	// Insert into user_strategy_publishes (ownership + publish tracking).
	_, err = tx.Exec(ctx, `
		INSERT INTO user_strategy_publishes (id, user_id, platform_strategy_id, published_at)
		VALUES ($1, $2, $3, now())
	`, publishID, params.UserID, params.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert publish: %w", err)
	}

	// Insert into marketplace_strategies (rich listing metadata).
	symbolsJSON := "[]"
	if len(params.Symbols) > 0 {
		symbolsJSON = "["
		for i, s := range params.Symbols {
			if i > 0 {
				symbolsJSON += ","
			}
			symbolsJSON += `"` + s + `"`
		}
		symbolsJSON += "]"
	}
	tagsJSON := "[]"
	if len(params.Tags) > 0 {
		tagsJSON = "["
		for i, t := range params.Tags {
			if i > 0 {
				tagsJSON += ","
			}
			tagsJSON += `"` + t + `"`
		}
		tagsJSON += "]"
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO marketplace_strategies
			(id, strategy_id, publisher_id, title, description, price_model, price_amount,
			 asset_class, symbols, timeframe, risk_level, tags, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'published', now(), now())
	`, stratID, publishID, params.UserID, params.Title, params.Description,
		params.PriceModel, params.PriceAmount, params.AssetClass,
		symbolsJSON, params.Timeframe, params.RiskLevel, tagsJSON)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert listing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: publish commit: %w", err)
	}
	return publishID.String(), nil
}

// Subscribe subscribes a user to a published strategy.
func (s *Service) Subscribe(ctx context.Context, userID, publisherUserID, strategyID, kind string) (string, error) {
	id := uuid.New().String()
	_, err := s.pg.Exec(ctx, `
		INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active)
		VALUES ($1, $2, $3, $4, $5, true)
	`, id, userID, publisherUserID, strategyID, kind)
	if err != nil {
		return "", fmt.Errorf("marketplace: subscribe: %w", err)
	}
	return id, nil
}

// Unsubscribe deactivates a subscription.
func (s *Service) Unsubscribe(ctx context.Context, userID, subscriptionID string) error {
	_, err := s.pg.Exec(ctx, `
		UPDATE user_subscriptions SET active = false
		WHERE id = $1 AND subscriber_user_id = $2
	`, subscriptionID, userID)
	if err != nil {
		return fmt.Errorf("marketplace: unsubscribe: %w", err)
	}
	return nil
}

// ListPublished returns strategies published to the marketplace with full
// metadata from marketplace_strategies (M12-B1). Optionally filters by asset_class.
func (s *Service) ListPublished(ctx context.Context, userID string, limit int, assetClass string) ([]PublishedStrategy, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
		SELECT
			usp.id,
			usp.platform_strategy_id,
			COALESCE(ms.title, ps.name, usp.platform_strategy_id::text),
			usp.user_id,
			usp.published_at,
			COALESCE(ms.title, ''),
			COALESCE(ms.description, ''),
			COALESCE(ms.price_model, ''),
			ms.price_amount,
			COALESCE(ms.asset_class, ''),
			COALESCE(ms.symbols, '[]'),
			ms.timeframe,
			COALESCE(ms.risk_level, ''),
			COALESCE(ms.tags, '[]'),
			COALESCE(ms.total_subscribers, 0),
			ms.win_rate,
			ms.total_pnl
		FROM user_strategy_publishes usp
		LEFT JOIN marketplace_strategies ms ON ms.strategy_id = usp.id
		LEFT JOIN platform_strategies ps ON ps.id::text = usp.platform_strategy_id::text
		WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if userID != "" {
		query += fmt.Sprintf(" AND usp.user_id::text = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	if assetClass != "" {
		query += fmt.Sprintf(" AND ms.asset_class = $%d", argIdx)
		args = append(args, assetClass)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY usp.published_at DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PublishedStrategy
	for rows.Next() {
		var p PublishedStrategy
		var symbolsRaw, tagsRaw string
		if err := rows.Scan(
			&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt,
			&p.Title, &p.Description, &p.PriceModel, &p.PriceAmount,
			&p.AssetClass, &symbolsRaw, &p.Timeframe, &p.RiskLevel, &tagsRaw,
			&p.TotalSubscribers, &p.WinRate, &p.TotalPnL,
		); err != nil {
			return nil, err
		}
		// Parse JSON arrays (stored as text/varchar in some schemas).
		p.Symbols = parseJSONStringArray(symbolsRaw)
		p.Tags = parseJSONStringArray(tagsRaw)
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListSubscriptions returns active subscriptions for a user.
func (s *Service) ListSubscriptions(ctx context.Context, userID string) ([]SubscriptionItem, error) {
	rows, err := s.pg.Query(ctx, `
		SELECT id, target_user_id, target_strategy_id, kind, active, created_at
		FROM user_subscriptions
		WHERE subscriber_user_id::text = $1 AND active = true
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubscriptionItem
	for rows.Next() {
		var sub SubscriptionItem
		if err := rows.Scan(&sub.SubscriptionID, &sub.TargetUserID, &sub.StrategyID, &sub.Kind, &sub.Active, &sub.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, rows.Err()
}

// parseJSONStringArray parses a PostgreSQL JSON array string like ["a","b"]
// into a Go []string. Returns empty slice on parse failure.
func parseJSONStringArray(raw string) []string {
	if raw == "" || raw == "[]" || raw == "null" {
		return nil
	}
	// Simple parser: strip brackets, split by comma, strip quotes.
	inner := raw
	if len(inner) >= 2 && inner[0] == '[' && inner[len(inner)-1] == ']' {
		inner = inner[1 : len(inner)-1]
	}
	if inner == "" {
		return nil
	}
	var result []string
	for _, part := range splitJSONArray(inner) {
		s := part
		// Strip surrounding quotes.
		if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
			s = s[1 : len(s)-1]
		}
		if s != "" {
			result = append(result, s)
		}
	}
	return result
}

// splitJSONArray splits a JSON array body by commas, respecting quoted strings.
func splitJSONArray(s string) []string {
	var parts []string
	var current string
	inQuote := false
	for _, c := range s {
		switch c {
		case '"':
			inQuote = !inQuote
			current += string(c)
		case ',':
			if inQuote {
				current += string(c)
			} else {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
