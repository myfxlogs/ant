package marketplace

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ── Published listing cache ─────────────────────────────────────────────────

type publishedCacheEntry struct {
	data      []PublishedStrategy
	expiresAt time.Time
}

var (
	publishedCacheMap = make(map[string]publishedCacheEntry)
	publishedCacheMu  sync.RWMutex
	publishedCacheTTL = 60 * time.Second
)

func publishedCacheKey(userID, assetClass, keyword, sortBy string, limit, offset int) string {
	return fmt.Sprintf("%s|%s|%s|%s|%d|%d", userID, assetClass, keyword, sortBy, limit, offset)
}

func publishedCacheGet(key string) ([]PublishedStrategy, bool) {
	publishedCacheMu.RLock()
	defer publishedCacheMu.RUnlock()
	e, ok := publishedCacheMap[key]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.data, true
}

// publishedCacheClear removes all cached entries. Call after any mutation that
// affects the published listing (publish, purchase, pricing change).
func publishedCacheClear() {
	publishedCacheMu.Lock()
	defer publishedCacheMu.Unlock()
	for k := range publishedCacheMap {
		delete(publishedCacheMap, k)
	}
}

func publishedCacheSet(key string, data []PublishedStrategy) {
	publishedCacheMu.Lock()
	defer publishedCacheMu.Unlock()
	// Evict stale entries when map grows too large (simple cap).
	if len(publishedCacheMap) > 256 {
		for k, v := range publishedCacheMap {
			if time.Now().After(v.expiresAt) {
				delete(publishedCacheMap, k)
			}
		}
	}
	publishedCacheMap[key] = publishedCacheEntry{data: data, expiresAt: time.Now().Add(publishedCacheTTL)}
}

// Unpublish hides a strategy from the marketplace by setting its status to "hidden".
// The strategy still exists in the database but no longer appears in listings.
func (s *Service) Unpublish(ctx context.Context, strategyID, userID string, isAdmin bool) error {
	sid, err := uuid.Parse(strategyID)
	if err != nil {
		return fmt.Errorf("marketplace: invalid strategy_id: %w", err)
	}

	// Verify ownership unless admin.
	if !isAdmin {
		var publisherID string
		err := s.pg.QueryRow(ctx,
			`SELECT publisher_id::text FROM marketplace_strategies WHERE strategy_id = $1`,
			sid,
		).Scan(&publisherID)
		if err != nil {
			return fmt.Errorf("marketplace: strategy not found")
		}
		if publisherID != userID {
			return fmt.Errorf("marketplace: only the publisher can unpublish this strategy")
		}
	}

	_, err = s.pg.Exec(ctx,
		`UPDATE marketplace_strategies SET status = 'hidden', updated_at = now() WHERE strategy_id = $1`,
		sid,
	)
	if err != nil {
		return fmt.Errorf("marketplace: unpublish: %w", err)
	}
	publishedCacheClear()
	return nil
}

// GetPublisherStats returns aggregated dashboard statistics for a publisher.
func (s *Service) GetPublisherStats(ctx context.Context, userID string) (*PublisherStats, error) {
	var stats PublisherStats
	err := s.pg.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_subscribers),0)
		 FROM marketplace_strategies WHERE publisher_id::text = $1 AND status = 'published'`,
		userID,
	).Scan(&stats.TotalPublished, &stats.TotalSubscribers)
	if err != nil {
		return nil, err
	}

	// Total revenue from sale transactions.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount::numeric),0)::text FROM wallet_transactions
		 WHERE user_id::text = $1 AND tx_type = 'sale'`,
		userID,
	).Scan(&stats.TotalRevenue)

	// Monthly revenue (last 30 days).
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(SUM(amount::numeric),0)::text FROM wallet_transactions
		 WHERE user_id::text = $1 AND tx_type = 'sale'
		   AND created_at > now() - INTERVAL '30 days'`,
		userID,
	).Scan(&stats.MonthlyRevenue)

	// Average rating across all published strategies.
	_ = s.pg.QueryRow(ctx,
		`SELECT COALESCE(AVG(r.avg_rating),0)
		 FROM marketplace_strategies ms
		 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id = ms.id
		 WHERE ms.publisher_id::text = $1 AND ms.status = 'published'`,
		userID,
	).Scan(&stats.AvgRating)

	// Top strategy by subscribers.
	_ = s.pg.QueryRow(ctx,
		`SELECT strategy_id::text, COALESCE(title,''), total_subscribers
		 FROM marketplace_strategies
		 WHERE publisher_id::text = $1 AND status = 'published'
		 ORDER BY total_subscribers DESC LIMIT 1`,
		userID,
	).Scan(&stats.TopStrategyID, &stats.TopStrategyTitle, &stats.TopStrategySubs)

	return &stats, nil
}

// Publish adds a strategy to the marketplace. Writes to both
// user_strategy_publishes (ownership tracking) and marketplace_strategies
// (rich listing metadata). Uses a transaction for atomicity.
func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
	switch params.PriceModel {
	case PriceModelFree, PriceModelOnce, PriceModelSubscription:
	default:
		return "", fmt.Errorf("marketplace: unsupported price_model %q", params.PriceModel)
	}

	violations, err := s.ValidateBacktestQuality(ctx, params.BacktestSnapshotProto, params.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: quality gate check: %w", err)
	}
	if len(violations) > 0 {
		msgs := make([]string, len(violations))
		for i, v := range violations {
			msgs[i] = v.String()
		}
		return "", fmt.Errorf("marketplace: backtest quality gate failed: %s", strings.Join(msgs, "; "))
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: publish begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	publishID, stratID := uuid.New(), uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO user_strategy_publishes (id, user_id, platform_strategy_id, published_at) VALUES ($1,$2,$3,now())`,
		publishID, params.UserID, params.StrategyID)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert publish: %w", err)
	}

	_, err = tx.Exec(ctx, `INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, code_snippet, backtest_snapshot, platform_fee_rate, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9,$10,$11,$12,$13,$14,$15::numeric,'published',now(),now())`,
		stratID, params.StrategyID, params.UserID, params.Title, params.Description,
		params.PriceModel, params.PriceAmount, params.AssetClass,
		pgTextArray(params.Symbols), params.Timeframe, params.RiskLevel, pgTextArray(params.Tags),
		params.CodeSnippet, params.BacktestSnapshotProto, params.PlatformFeeRate)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert listing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: publish commit: %w", err)
	}
	publishedCacheClear()
	return publishID.String(), nil
}

// ListPublished returns strategies published to the marketplace with full
// metadata from marketplace_strategies (M12-B1). Supports keyword search, sorting, and offset pagination.
// Results are cached for 60s to reduce DB load on the market listing page.
func (s *Service) ListPublished(ctx context.Context, userID string, limit int, offset int, assetClass, keyword, sortBy string) ([]PublishedStrategy, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Check cache (skip for keyword searches — they vary too much to cache effectively).
	var cacheKey string
	if keyword == "" {
		cacheKey = publishedCacheKey(userID, assetClass, keyword, sortBy, limit, offset)
		if cached, ok := publishedCacheGet(cacheKey); ok {
			return cached, nil
		}
	}

	query, args := buildPublishedQuery(userID, assetClass, keyword, sortBy, limit, offset)
	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PublishedStrategy
	for rows.Next() {
		var p PublishedStrategy
		var symbolsRaw, tagsRaw string
		var snapshotRaw []byte
		if err := rows.Scan(&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt,
			&p.Title, &p.Description, &p.PriceModel, &p.PriceAmount,
			&p.AssetClass, &symbolsRaw, &p.Timeframe, &p.RiskLevel, &tagsRaw,
			&p.TotalSubscribers, &p.WinRate, &p.TotalPnL, &p.AvgRating, &p.RatingCount,
			&p.CodeSnippet, &snapshotRaw); err != nil {
			return nil, err
		}
		p.Symbols = parseJSONStringArray(symbolsRaw)
		p.Tags = parseJSONStringArray(tagsRaw)
		if len(snapshotRaw) > 0 {
			var snap antv1.BacktestSnapshot
			if err := proto.Unmarshal(snapshotRaw, &snap); err == nil {
				p.BacktestSnapshotProto = &snap
			}
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Cache the result for non-keyword queries.
	if cacheKey != "" {
		publishedCacheSet(cacheKey, out)
	}
	return out, nil
}

func buildPublishedQuery(userID, assetClass, keyword, sortBy string, limit, offset int) (string, []interface{}) {
	query := `SELECT usp.id, usp.platform_strategy_id, COALESCE(ms.title,st.name,usp.platform_strategy_id::text),
			COALESCE(u.email, u.nickname, usp.user_id::text), usp.published_at, COALESCE(ms.title,''), COALESCE(ms.description,''),
			COALESCE(ms.price_model,''), ms.price_amount::text, COALESCE(ms.asset_class,''),
			COALESCE(ms.symbols::text,'{}'), ms.timeframe, COALESCE(ms.risk_level,''),
			COALESCE(ms.tags::text,'{}'), COALESCE(ms.total_subscribers,0), ms.win_rate, ms.total_pnl,
			COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0),
			COALESCE(ms.code_snippet,''), ms.backtest_snapshot::text
		 FROM user_strategy_publishes usp
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id=usp.platform_strategy_id
		 LEFT JOIN strategy_templates st ON st.id::text=usp.platform_strategy_id::text
		 LEFT JOIN users u ON u.id = usp.user_id
		 LEFT JOIN (SELECT strategy_id, AVG(rating) AS avg_rating, COUNT(*)::int AS rating_count FROM marketplace_ratings GROUP BY strategy_id) r ON r.strategy_id=ms.id
		 WHERE ms.status = 'published'`
	args := []interface{}{}
	p := 0 // positional parameter counter
	next := func() int { p++; return p }

	if userID != "" {
		query += fmt.Sprintf(" AND usp.user_id::text = $%d", next())
		args = append(args, userID)
	}
	if assetClass != "" {
		query += fmt.Sprintf(" AND ms.asset_class = $%d", next())
		args = append(args, assetClass)
	}
	var hasFuzzySearch bool
	if keyword != "" {
		n := next()
		// pg_trgm similarity across title, description, tags, strategy name, and publisher.
		// Threshold 0.2 catches typos and partial matches while filtering noise.
		query += fmt.Sprintf(
			" AND (similarity(ms.title, $%[1]d) > 0.2 OR similarity(ms.description, $%[1]d) > 0.2"+
				" OR similarity(ms.tags::text, $%[1]d) > 0.2"+
				" OR similarity(st.name, $%[1]d) > 0.2"+
				" OR similarity(u.nickname, $%[1]d) > 0.2"+
				" OR similarity(u.email, $%[1]d) > 0.2)", n)
		args = append(args, keyword)
		hasFuzzySearch = true
	}
	if hasFuzzySearch {
		// ORDER BY highest relevance across all searchable fields.
		n := next()
		query += fmt.Sprintf(
			" ORDER BY GREATEST(similarity(ms.title, $%[1]d), similarity(ms.description, $%[1]d),"+
				" similarity(st.name, $%[1]d), similarity(u.nickname, $%[1]d)) DESC LIMIT $%d", n, next())
		args = append(args, keyword, limit)
	} else {
		switch sortBy {
		case "popular":
			query += fmt.Sprintf(" ORDER BY COALESCE(ms.total_subscribers,0) DESC LIMIT $%d", next())
		case "performance":
			query += fmt.Sprintf(" ORDER BY COALESCE(ms.win_rate,0) DESC LIMIT $%d", next())
		default:
			query += fmt.Sprintf(" ORDER BY usp.published_at DESC LIMIT $%d", next())
		}
		args = append(args, limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", next())
		args = append(args, offset)
	}
	return query, args
}
