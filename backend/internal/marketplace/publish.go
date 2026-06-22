package marketplace

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ── Published listing cache ─────────────────────────────────────────────────

type publishedCacheEntry struct {
	data      []PublishedStrategy
	expiresAt time.Time
}

var (
	publishedCacheMap   = make(map[string]publishedCacheEntry)
	publishedCacheMu    sync.RWMutex
	publishedCacheTTL   = 60 * time.Second
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

// Publish adds a strategy to the marketplace. Writes to both
// user_strategy_publishes (ownership tracking) and marketplace_strategies
// (rich listing metadata). Uses a transaction for atomicity.
func (s *Service) Publish(ctx context.Context, params PublishParams) (string, error) {
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

	_, err = tx.Exec(ctx, `INSERT INTO marketplace_strategies (id, strategy_id, publisher_id, title, description, price_model, price_amount, asset_class, symbols, timeframe, risk_level, tags, code_snippet, backtest_snapshot, platform_fee_rate, status, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,'published',now(),now())`,
		stratID, params.StrategyID, params.UserID, params.Title, params.Description,
		params.PriceModel, params.PriceAmount, params.AssetClass,
		pgTextArray(params.Symbols), params.Timeframe, params.RiskLevel, pgTextArray(params.Tags),
		params.CodeSnippet, params.BacktestSnapshotJSON, params.PlatformFeeRate)
	if err != nil {
		return "", fmt.Errorf("marketplace: insert listing: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: publish commit: %w", err)
	}
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
		var snapshotRaw *string
		if err := rows.Scan(&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt,
			&p.Title, &p.Description, &p.PriceModel, &p.PriceAmount,
			&p.AssetClass, &symbolsRaw, &p.Timeframe, &p.RiskLevel, &tagsRaw,
			&p.TotalSubscribers, &p.WinRate, &p.TotalPnL, &p.AvgRating, &p.RatingCount,
			&p.CodeSnippet, &snapshotRaw); err != nil {
			return nil, err
		}
		p.Symbols = parseJSONStringArray(symbolsRaw)
		p.Tags = parseJSONStringArray(tagsRaw)
		if snapshotRaw != nil && *snapshotRaw != "" {
			var snap BacktestSnapshot
			if err := json.Unmarshal([]byte(*snapshotRaw), &snap); err == nil {
				p.BacktestSnapshot = &snap
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
			COALESCE(ms.price_model,''), ms.price_amount, COALESCE(ms.asset_class,''),
			COALESCE(ms.symbols::text,'{}'), ms.timeframe, COALESCE(ms.risk_level,''),
			COALESCE(ms.tags::text,'{}'), COALESCE(ms.total_subscribers,0), ms.win_rate, ms.total_pnl,
			COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0),
			COALESCE(ms.code_snippet,''), ms.backtest_snapshot::text
		 FROM user_strategy_publishes usp
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id=usp.platform_strategy_id
		 LEFT JOIN strategy_templates st ON st.id::text=usp.platform_strategy_id::text
		 LEFT JOIN users u ON u.id = usp.user_id
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
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", len(args)+1)
		args = append(args, offset)
	}
	return query, args
}
