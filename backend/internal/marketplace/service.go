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
	AvgRating        float64
	RatingCount      int32
}

// Publish adds a strategy to the marketplace. Writes to both
// user_strategy_publishes (ownership tracking) and marketplace_strategies
// (rich listing metadata). Uses a transaction for atomicity.
func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
	tx, err := s.pg.Begin(ctx)
	if err != nil { return "", fmt.Errorf("marketplace: publish begin tx: %w", err) }
	defer tx.Rollback(ctx)

	publishID, stratID := uuid.New(), uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO user_strategy_publishes (id, user_id, platform_strategy_id, published_at) VALUES ($1,$2,$3,now())`,
		publishID, params.UserID, params.StrategyID)
	if err != nil { return "", fmt.Errorf("marketplace: insert publish: %w", err) }

	_, err = tx.Exec(ctx, `INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'published',now(),now())`,
		stratID, params.StrategyID, params.UserID, params.Title, params.Description,
		params.PriceModel, params.PriceAmount, params.AssetClass,
		pgTextArray(params.Symbols), params.Timeframe, params.RiskLevel, pgTextArray(params.Tags))
	if err != nil { return "", fmt.Errorf("marketplace: insert listing: %w", err) }

	if err := tx.Commit(ctx); err != nil { return "", fmt.Errorf("marketplace: publish commit: %w", err) }
	return publishID.String(), nil
}

// pgTextArray formats a string slice as a PostgreSQL TEXT[] literal: {a,b,c}.
// Special characters in values (commas, braces, quotes) are backslash-escaped.
func pgTextArray(items []string) string {
	if len(items) == 0 {
		return "{}"
	}
	out := "{"
	for i, s := range items {
		if i > 0 {
			out += ","
		}
		out += pgEscape(s)
	}
	return out + "}"
}

func pgEscape(s string) string {
	b := make([]byte, 0, len(s)+4)
	for _, c := range []byte(s) {
		switch c {
		case '"', '\\', '{', '}', ',':
			b = append(b, '\\', c)
		default:
			b = append(b, c)
		}
	}
	return string(b)
}


// Unsubscribe deactivates a subscription.

// ListPublished returns strategies published to the marketplace with full
// metadata from marketplace_strategies (M12-B1). Supports keyword search and sorting.
func (s *Service) ListPublished(ctx context.Context, userID string, limit int, assetClass, keyword, sortBy string) ([]PublishedStrategy, error) {
	if limit <= 0 { limit = 50 }
	query, args := buildPublishedQuery(userID, assetClass, keyword, sortBy, limit)
	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	var out []PublishedStrategy
	for rows.Next() {
		var p PublishedStrategy
		var symbolsRaw, tagsRaw string
		if err := rows.Scan(&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt,
			&p.Title, &p.Description, &p.PriceModel, &p.PriceAmount,
			&p.AssetClass, &symbolsRaw, &p.Timeframe, &p.RiskLevel, &tagsRaw,
			&p.TotalSubscribers, &p.WinRate, &p.TotalPnL, &p.AvgRating, &p.RatingCount); err != nil {
			return nil, err
		}
		p.Symbols = parseJSONStringArray(symbolsRaw)
		p.Tags = parseJSONStringArray(tagsRaw)
		out = append(out, p)
	}
	return out, rows.Err()
}

func buildPublishedQuery(userID, assetClass, keyword, sortBy string, limit int) (string, []interface{}) {
	query := `SELECT usp.id, usp.platform_strategy_id, COALESCE(ms.title,ps.name,usp.platform_strategy_id::text),
		usp.user_id, usp.published_at, COALESCE(ms.title,''), COALESCE(ms.description,''),
		COALESCE(ms.price_model,''), ms.price_amount, COALESCE(ms.asset_class,''),
		COALESCE(ms.symbols,'{}'), ms.timeframe, COALESCE(ms.risk_level,''),
		COALESCE(ms.tags,'{}'), COALESCE(ms.total_subscribers,0), ms.win_rate, ms.total_pnl,
		COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0)
	 FROM user_strategy_publishes usp
	 LEFT JOIN marketplace_strategies ms ON ms.strategy_id=usp.platform_strategy_id
	 LEFT JOIN platform_strategies ps ON ps.id::text=usp.platform_strategy_id::text
	 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating, COUNT(*)::int AS rating_count FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id=ms.id
	 WHERE 1=1`
	args := []interface{}{}
	if userID != "" {
		query += fmt.Sprintf(" AND usp.user_id::text = $%d", len(args)+1)
		args = append(args, userID)
	}
	if assetClass != "" {
		query += fmt.Sprintf(" AND ms.asset_class = $%d", len(args)+1)
		args = append(args, assetClass)
	}
	if keyword != "" {
		kw := "%" + keyword + "%"
		query += fmt.Sprintf(" AND (ms.title ILIKE $%d OR ms.description ILIKE $%d OR ms.tags::text ILIKE $%d)", len(args)+1, len(args)+1, len(args)+1)
		args = append(args, kw)
	}
	switch sortBy {
	case "popular":
		query += fmt.Sprintf(" ORDER BY COALESCE(ms.total_subscribers,0) DESC LIMIT $%d", len(args)+1)
	case "performance":
		query += fmt.Sprintf(" ORDER BY COALESCE(ms.win_rate,0) DESC LIMIT $%d", len(args)+1)
	default:
		query += fmt.Sprintf(" ORDER BY usp.published_at DESC LIMIT $%d", len(args)+1)
	}
	args = append(args, limit)
	return query, args
}

// ListSubscriptions returns active subscriptions for a user.
// parseJSONStringArray parses a PostgreSQL JSON array string like ["a","b"]
// into a Go []string. Returns empty slice on parse failure.
func parseJSONStringArray(raw string) []string {
	if raw == "" || raw == "[]" || raw == "{}" || raw == "null" {
		return nil
	}
	// Simple parser: strip brackets, split by comma, strip quotes.
	// Handles both JSON array ["a","b"] and PostgreSQL array {a,b} formats.
	inner := raw
	if len(inner) >= 2 {
		first, last := inner[0], inner[len(inner)-1]
		if (first == '[' && last == ']') || (first == '{' && last == '}') {
			inner = inner[1 : len(inner)-1]
		}
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

// --- Rating --------------------------------------------------------------

// Rate inserts or updates a user's rating for a strategy and returns the new
// average and count (matching the RateStrategyResponse proto shape).
func (s *Service) Rate(ctx context.Context, userID, strategyID string, rating int32) (float64, int32, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`INSERT INTO marketplace_ratings (strategy_id, user_id, rating)
		 VALUES ($1,$2,$3) ON CONFLICT (strategy_id, user_id) DO UPDATE SET rating=$3`,
		sid, uid, rating)
	if err != nil {
		return 0, 0, fmt.Errorf("marketplace: rate: %w", err)
	}
	var avg float64
	var count int32
	err = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)
	return avg, count, err
}

func (s *Service) ListRatings(ctx context.Context, strategyID string) ([]RatingItem, float64, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	rows, err := s.pg.Query(ctx,
		`SELECT id, user_id, rating, created_at FROM marketplace_ratings WHERE strategy_id=$1 ORDER BY created_at DESC`, sid)
	if err != nil {
		return nil, 0, 0, err
	}
	defer rows.Close()
	var items []RatingItem
	for rows.Next() {
		var r RatingItem
		if err := rows.Scan(&r.ID, &r.UserID, &r.Rating, &r.CreatedAt); err != nil {
			return nil, 0, 0, err
		}
		items = append(items, r)
	}
	var avg float64
	var count int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(rating),0), COUNT(*) FROM marketplace_ratings WHERE strategy_id=$1`, sid,
	).Scan(&avg, &count)
	return items, avg, count, rows.Err()
}

// --- Comment --------------------------------------------------------------

func (s *Service) Comment(ctx context.Context, userID, strategyID, content string) (string, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var id uuid.UUID
	err = s.pg.QueryRow(ctx,
		`INSERT INTO marketplace_comments (strategy_id, user_id, content)
		 VALUES ($1,$2,$3) RETURNING id`, sid, uid, content).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("marketplace: comment: %w", err)
	}
	return id.String(), nil
}

func (s *Service) ListComments(ctx context.Context, strategyID string, limit, offset int32) ([]CommentItem, int32, error) {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	var total int32
	_ = s.pg.QueryRow(ctx,
		`SELECT COUNT(*) FROM marketplace_comments WHERE strategy_id=$1`, sid).Scan(&total)
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.pg.Query(ctx,
		`SELECT c.id, c.user_id, COALESCE(u.nickname,u.email,''), c.content, c.created_at
		 FROM marketplace_comments c LEFT JOIN users u ON u.id=c.user_id
		 WHERE c.strategy_id=$1 ORDER BY c.created_at ASC LIMIT $2 OFFSET $3`,
		sid, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []CommentItem
	for rows.Next() {
		var c CommentItem
		if err := rows.Scan(&c.ID, &c.UserID, &c.UserName, &c.Content, &c.CreatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, c)
	}
	return items, total, rows.Err()
}

// PurchaseResult holds the outcome of a paid strategy purchase.
type PurchaseResult struct {
	SubscriptionID string
	TransactionID  string
	AmountCharged  string
	BalanceAfter   string
}

// PurchaseStrategy atomically charges the user's wallet and creates a subscription
// for a one-time purchase strategy. All steps run in a single DB transaction
// with FOR UPDATE row locking to prevent races.
func (s *Service) PurchaseStrategy(ctx context.Context, userID, strategyID, publisherUserID string) (*PurchaseResult, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid user_id: %w", err)
	}
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	pid, err := uuid.Parse(publisherUserID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: invalid publisher_user_id: %w", err)
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("marketplace: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 1. Look up strategy price.
	var priceModel string
	var priceAmount float64
	var strategyTitle string
	err = tx.QueryRow(ctx,
		`SELECT price_model, COALESCE(price_amount, 0), title FROM marketplace_strategies WHERE strategy_id = $1`,
		sid,
	).Scan(&priceModel, &priceAmount, &strategyTitle)
	if err != nil {
		return nil, fmt.Errorf("marketplace: strategy not published")
	}
	if priceModel != "once" || priceAmount <= 0 {
		return nil, fmt.Errorf("marketplace: strategy is not purchasable")
	}

	// 2. Check for existing active subscription.
	var existing string
	_ = tx.QueryRow(ctx,
		`SELECT id::text FROM user_subscriptions WHERE subscriber_user_id = $1 AND target_strategy_id = $2 AND active = true`,
		uid, sid,
	).Scan(&existing)
	if existing != "" {
		return nil, fmt.Errorf("marketplace: already subscribed")
	}

	// 3. Lock wallet and read balance.
	var walletID uuid.UUID
	var balanceBefore string
	err = tx.QueryRow(ctx,
		`SELECT id, balance::text FROM user_wallets WHERE user_id = $1 FOR UPDATE`,
		uid,
	).Scan(&walletID, &balanceBefore)
	if err != nil {
		return nil, fmt.Errorf("marketplace: wallet not found")
	}

	// Compare as Go floats for the error message (DB does the real comparison).
	if balanceBefore == "" {
		return nil, fmt.Errorf("marketplace: wallet balance unavailable")
	}

	amountStr := fmt.Sprintf("%.2f", priceAmount)
	negAmountStr := fmt.Sprintf("-%.2f", priceAmount)

	// 4. Deduct balance (DB enforces non-negative via CHECK constraint).
	var balanceAfter string
	err = tx.QueryRow(ctx,
		`UPDATE user_wallets SET balance = balance - $1::numeric, updated_at = now()
		 WHERE user_id = $2 AND balance >= $1::numeric
		 RETURNING balance::text`,
		amountStr, uid,
	).Scan(&balanceAfter)
	if err != nil {
		return nil, fmt.Errorf("marketplace: insufficient balance")
	}

	// 5. Insert wallet_transactions row.
	var txID uuid.UUID
	desc := fmt.Sprintf("Purchase strategy: %s", strategyTitle)
	err = tx.QueryRow(ctx,
		`INSERT INTO wallet_transactions (id, wallet_id, user_id, tx_type, amount, balance_before, balance_after, description)
		 VALUES (uuid_generate_v4(), $1, $2, 'purchase', $3::numeric, $4::numeric, $5::numeric, $6)
		 RETURNING id`,
		walletID, uid, negAmountStr, balanceBefore, balanceAfter, desc,
	).Scan(&txID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: record transaction: %w", err)
	}

	// 6. Insert subscription row.
	var subID uuid.UUID
	err = tx.QueryRow(ctx,
		`INSERT INTO user_subscriptions (id, subscriber_user_id, target_user_id, target_strategy_id, kind, active)
		 VALUES (uuid_generate_v4(), $1, $2, $3, 'purchase', true)
		 RETURNING id`,
		uid, pid, sid,
	).Scan(&subID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: create subscription: %w", err)
	}

	// 7. Increment total_subscribers counter.
	_, err = tx.Exec(ctx,
		`UPDATE marketplace_strategies SET total_subscribers = total_subscribers + 1 WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return nil, fmt.Errorf("marketplace: update subscriber count: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("marketplace: commit purchase: %w", err)
	}

	return &PurchaseResult{
		SubscriptionID: subID.String(),
		TransactionID:  txID.String(),
		AmountCharged:  amountStr,
		BalanceAfter:   balanceAfter,
	}, nil
}

// --- Admin Pricing --------------------------------------------------------

func (s *Service) SetPricing(ctx context.Context, strategyID, priceModel string, priceAmount float64) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}
	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET price_model=$2, price_amount=$3, updated_at=now() WHERE strategy_id=$1`,
		sid, priceModel, priceAmount)
	return err
}

// --- Types ----------------------------------------------------------------

type RatingItem struct {
	ID        string
	UserID    string
	Rating    int32
	CreatedAt time.Time
}

type CommentItem struct {
	ID        string
	UserID    string
	UserName  string
	Content   string
	CreatedAt time.Time
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
