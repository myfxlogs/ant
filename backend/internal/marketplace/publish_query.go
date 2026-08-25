package marketplace

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// ListPublished returns strategies published to the marketplace with full
// metadata from marketplace_strategies (M12-B1). Supports keyword search, sorting, and offset pagination.
// Results are cached for 60s to reduce DB load on the market listing page.
func (s *Service) ListPublished(ctx context.Context, userID string, limit int, offset int, assetClass, keyword, sortBy, priceFilter string) ([]PublishedStrategy, int, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	// Check cache (skip for keyword searches — they vary too much to cache effectively).
	var cacheKey string
	if keyword == "" {
		cacheKey = s.pubCache.key(userID, assetClass, keyword, sortBy, priceFilter, limit, offset)
		if cached, cachedTotal, ok := s.pubCache.get(cacheKey); ok {
			return cached, cachedTotal, nil
		}
	}

	query, args := buildPublishedQuery(userID, assetClass, keyword, sortBy, priceFilter, limit, offset)
	rows, err := s.pg.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []PublishedStrategy
	for rows.Next() {
		var p PublishedStrategy
		var snapshotRaw []byte
		if err := rows.Scan(&p.PublishID, &p.StrategyID, &p.StrategyName, &p.PublisherUserID, &p.PublishedAt,
			&p.Title, &p.Description, &p.PriceModel, &p.PriceAmount,
			&p.AssetClass, &p.Symbols, &p.Timeframe, &p.RiskLevel, &p.Tags,
			&p.TotalSubscribers, &p.WinRate, &p.TotalPnL, &p.AvgRating, &p.RatingCount,
			&p.CodeSnippet, &snapshotRaw, &p.ProviderVerified, &p.ProviderType, &p.Disclaimer, &p.DecayStatus); err != nil {
			return nil, 0, err
		}
		if len(snapshotRaw) > 0 {
			var snap antv1.BacktestSnapshot
			if err := proto.Unmarshal(snapshotRaw, &snap); err == nil {
				p.BacktestSnapshotProto = &snap
			}
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	// Count total matching rows (without LIMIT/OFFSET) for pagination.
	total := len(out)
	if offset == 0 && len(out) < limit {
		// Short-circuit: first page with fewer results than limit = exact count.
	} else {
		countQuery, countArgs := buildPublishedCountQuery(userID, assetClass, keyword, priceFilter)
		if err := s.pg.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
			total = len(out)
		}
	}

	// Cache the result for non-keyword queries.
	if cacheKey != "" {
		s.pubCache.set(cacheKey, out, total)
	}
	return out, total, nil
}

func buildPublishedQuery(userID, assetClass, keyword, sortBy, priceFilter string, limit, offset int) (string, []interface{}) {
	query := `SELECT usp.id, usp.platform_strategy_id, COALESCE(ms.title,st.name,usp.platform_strategy_id::text),
			COALESCE(u.email, u.nickname, usp.user_id::text), usp.published_at, COALESCE(ms.title,''), COALESCE(ms.description,''),
			COALESCE(ms.price_model,''), ms.price_amount::text, COALESCE(ms.asset_class,''),
			ms.symbols, ms.timeframe, COALESCE(ms.risk_level,''),
			ms.tags, COALESCE(ms.total_subscribers,0), ms.win_rate, ms.total_pnl,
			COALESCE(r.avg_rating,0), COALESCE(r.rating_count,0),
			COALESCE(ms.code_snippet,''), ms.backtest_snapshot,
			COALESCE(u.verified_provider,false), COALESCE(u.provider_type,'human'),
			COALESCE(ms.disclaimer,''), COALESCE(ms.decay_status,'none')
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
	switch priceFilter {
	case "free":
		query += " AND (ms.price_amount IS NULL OR ms.price_amount = '0')"
	case "paid":
		query += " AND ms.price_amount IS NOT NULL AND ms.price_amount::numeric > 0"
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
			query += fmt.Sprintf(" ORDER BY COALESCE(ms.is_featured,false) DESC, COALESCE(ms.featured_priority,0) DESC, COALESCE(ms.total_subscribers,0) DESC LIMIT $%d", next())
		case "performance":
			query += fmt.Sprintf(" ORDER BY COALESCE(ms.is_featured,false) DESC, COALESCE(ms.featured_priority,0) DESC, COALESCE(ms.win_rate,0) DESC LIMIT $%d", next())
		default:
			query += fmt.Sprintf(" ORDER BY COALESCE(ms.is_featured,false) DESC, COALESCE(ms.featured_priority,0) DESC, usp.published_at DESC LIMIT $%d", next())
		}
		args = append(args, limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", next())
		args = append(args, offset)
	}
	return query, args
}

func buildPublishedCountQuery(userID, assetClass, keyword, priceFilter string) (string, []interface{}) {
	query := `SELECT COUNT(*) FROM user_strategy_publishes usp
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id=usp.platform_strategy_id
		 LEFT JOIN strategy_templates st ON st.id::text=usp.platform_strategy_id::text
		 LEFT JOIN users u ON u.id = usp.user_id
		 WHERE ms.status = 'published'`
	args := []interface{}{}
	p := 0
	next := func() int { p++; return p }

	if userID != "" {
		query += fmt.Sprintf(" AND usp.user_id::text = $%d", next())
		args = append(args, userID)
	}
	if assetClass != "" {
		query += fmt.Sprintf(" AND ms.asset_class = $%d", next())
		args = append(args, assetClass)
	}
	switch priceFilter {
	case "free":
		query += " AND (ms.price_amount IS NULL OR ms.price_amount = '0')"
	case "paid":
		query += " AND ms.price_amount IS NOT NULL AND ms.price_amount::numeric > 0"
	}
	if keyword != "" {
		n := next()
		query += fmt.Sprintf(
			" AND (similarity(ms.title, $%[1]d) > 0.2 OR similarity(ms.description, $%[1]d) > 0.2"+
				" OR similarity(ms.tags::text, $%[1]d) > 0.2"+
				" OR similarity(st.name, $%[1]d) > 0.2"+
				" OR similarity(u.nickname, $%[1]d) > 0.2"+
				" OR similarity(u.email, $%[1]d) > 0.2)", n)
		args = append(args, keyword)
	}
	return query, args
}
