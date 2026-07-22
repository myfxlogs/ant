// Package marketplace — Phase 5.2: Strategy bundles.
//
// Bundles allow providers to group multiple strategies into a single
// purchasable product at a discounted price. Bundle purchase atomically
// creates individual subscriptions for each strategy in the bundle.
package marketplace

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

// Bundle represents a strategy bundle in the marketplace.
type Bundle struct {
	ID              string
	Title           string
	Description     string
	PublisherID     string
	PriceModel      string
	PriceAmount     decimal.Decimal
	PlatformFeeRate decimal.Decimal
	Status          string
	TotalPurchases  int32
	CreatedAt       time.Time
	UpdatedAt       time.Time
	Items           []BundleItem
}

// BundleItem represents a single strategy within a bundle.
type BundleItem struct {
	StrategyID  string
	SortOrder   int32
	Title       string // populated from marketplace_strategies
	PriceAmount string // individual strategy price for comparison
}

// CreateBundle creates a new strategy bundle.
// The publisher must own all strategies in the bundle.
func (s *Service) CreateBundle(ctx context.Context, publisherID, title, description, priceModel, priceAmount string, strategyIDs []string, platformFeeRate string) (string, error) {
	pid, err := uuid.Parse(publisherID)
	if err != nil {
		return "", fmt.Errorf("marketplace: create bundle: invalid publisher_id: %w", err)
	}
	if len(strategyIDs) < 2 {
		return "", fmt.Errorf("marketplace: create bundle: at least 2 strategies required")
	}
	if title == "" {
		return "", fmt.Errorf("marketplace: create bundle: title is required")
	}

	price, err := decimal.NewFromString(priceAmount)
	if err != nil {
		return "", fmt.Errorf("marketplace: create bundle: invalid price_amount: %w", err)
	}
	feeRate, err := decimal.NewFromString(platformFeeRate)
	if err != nil {
		feeRate = decimal.NewFromFloat(0.10) // default
	}

	// Parse all strategy IDs once.
	parsedIDs := make([]uuid.UUID, 0, len(strategyIDs))
	for _, sidStr := range strategyIDs {
		sid, err := uuid.Parse(sidStr)
		if err != nil {
			return "", fmt.Errorf("marketplace: create bundle: invalid strategy_id: %w", err)
		}
		parsedIDs = append(parsedIDs, sid)
	}

	// Verify ownership of all strategies in a single query.
	rows, err := s.pg.Query(ctx,
		`SELECT strategy_id, publisher_id FROM marketplace_strategies
		 WHERE strategy_id = ANY($1) AND status = 'published'`,
		parsedIDs)
	if err != nil {
		return "", fmt.Errorf("marketplace: create bundle: query strategies: %w", err)
	}
	owners := make(map[uuid.UUID]uuid.UUID, len(parsedIDs))
	for rows.Next() {
		var sid, ownerID uuid.UUID
		if err := rows.Scan(&sid, &ownerID); err != nil {
			rows.Close()
			return "", fmt.Errorf("marketplace: create bundle: scan owner: %w", err)
		}
		owners[sid] = ownerID
	}
	rows.Close()
	for _, sid := range parsedIDs {
		ownerID, ok := owners[sid]
		if !ok {
			return "", fmt.Errorf("marketplace: create bundle: strategy %s not found or not published", sid)
		}
		if ownerID != pid {
			return "", fmt.Errorf("marketplace: create bundle: not the owner of strategy %s", sid)
		}
	}

	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("marketplace: create bundle: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	bundleID := uuid.New()
	_, err = tx.Exec(ctx,
		`INSERT INTO marketplace_bundles (id, title, description, publisher_id, price_model, price_amount, platform_fee_rate)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		bundleID, title, description, pid, priceModel, price, feeRate,
	)
	if err != nil {
		return "", fmt.Errorf("marketplace: create bundle: insert: %w", err)
	}

	for i, sid := range parsedIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO marketplace_bundle_items (bundle_id, strategy_id, sort_order)
			 VALUES ($1, $2, $3)`,
			bundleID, sid, i,
		)
		if err != nil {
			return "", fmt.Errorf("marketplace: create bundle: insert item %d: %w", i, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("marketplace: create bundle: commit: %w", err)
	}

	return bundleID.String(), nil
}

// ListBundles returns published bundles, optionally filtered by publisher.
func (s *Service) ListBundles(ctx context.Context, publisherID string, limit, offset int) ([]Bundle, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var whereClause string
	args := []any{limit, offset}
	var pid uuid.UUID
	if publisherID != "" {
		var err error
		pid, err = uuid.Parse(publisherID)
		if err != nil {
			return nil, 0, fmt.Errorf("marketplace: list bundles: invalid publisher_id: %w", err)
		}
		whereClause = " AND publisher_id = $3"
		args = append(args, pid)
	}

	var total int
	countArgs := []any{}
	countQuery := "SELECT COUNT(*) FROM marketplace_bundles WHERE status = 'published'"
	if publisherID != "" {
		countQuery += " AND publisher_id = $1"
		countArgs = append(countArgs, pid)
	}
	err := s.pg.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list bundles: count: %w", err)
	}

	rows, err := s.pg.Query(ctx,
		`SELECT id::text, title, description, publisher_id::text, price_model,
		        price_amount, platform_fee_rate, status, total_purchases, created_at, updated_at
		 FROM marketplace_bundles
		 WHERE status = 'published'`+whereClause+`
		 ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		args...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("marketplace: list bundles: query: %w", err)
	}
	defer rows.Close()

	var bundles []Bundle
	for rows.Next() {
		var b Bundle
		if err := rows.Scan(&b.ID, &b.Title, &b.Description, &b.PublisherID,
			&b.PriceModel, &b.PriceAmount, &b.PlatformFeeRate, &b.Status,
			&b.TotalPurchases, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("marketplace: list bundles: scan: %w", err)
		}
		bundles = append(bundles, b)
	}
	rows.Close()

	// Load items for all bundles in a single query (avoids N+1).
	if len(bundles) > 0 {
		bundleIDs := make([]string, len(bundles))
		for i := range bundles {
			bundleIDs[i] = bundles[i].ID
		}
		itemRows, err := s.pg.Query(ctx,
			`SELECT bi.bundle_id::text, bi.strategy_id::text, bi.sort_order,
			        COALESCE(ms.title, '')
			 FROM marketplace_bundle_items bi
			 LEFT JOIN marketplace_strategies ms ON ms.strategy_id = bi.strategy_id
			 WHERE bi.bundle_id = ANY($1)
			 ORDER BY bi.bundle_id, bi.sort_order`,
			bundleIDs,
		)
		if err != nil {
			s.log.Warn("list bundles: failed to load items", zap.Error(err))
		} else {
			itemsByBundle := make(map[string][]BundleItem)
			for itemRows.Next() {
				var bid, sid, title string
				var sortOrder int32
				if err := itemRows.Scan(&bid, &sid, &sortOrder, &title); err != nil {
					continue
				}
				itemsByBundle[bid] = append(itemsByBundle[bid], BundleItem{
					StrategyID: sid,
					SortOrder:  sortOrder,
					Title:      title,
				})
			}
			itemRows.Close()
			for i := range bundles {
				bundles[i].Items = itemsByBundle[bundles[i].ID]
			}
		}
	}

	return bundles, total, rows.Err()
}

// GetBundle returns a single bundle with its items.
func (s *Service) GetBundle(ctx context.Context, bundleID string) (*Bundle, error) {
	bid, err := uuid.Parse(bundleID)
	if err != nil {
		return nil, fmt.Errorf("marketplace: get bundle: invalid bundle_id: %w", err)
	}

	var b Bundle
	err = s.pg.QueryRow(ctx,
		`SELECT id::text, title, description, publisher_id::text, price_model,
		        price_amount, platform_fee_rate, status, total_purchases, created_at, updated_at
		 FROM marketplace_bundles WHERE id = $1 AND status = 'published'`,
		bid,
	).Scan(&b.ID, &b.Title, &b.Description, &b.PublisherID,
		&b.PriceModel, &b.PriceAmount, &b.PlatformFeeRate, &b.Status,
		&b.TotalPurchases, &b.CreatedAt, &b.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("marketplace: get bundle: %w", err)
	}

	items, err := s.getBundleItems(ctx, b.ID)
	if err != nil {
		return nil, err
	}
	b.Items = items
	return &b, nil
}

// getBundleItems loads the strategy items for a bundle.
func (s *Service) getBundleItems(ctx context.Context, bundleID string) ([]BundleItem, error) {
	bid, err := uuid.Parse(bundleID)
	if err != nil {
		return nil, err
	}
	rows, err := s.pg.Query(ctx,
		`SELECT bi.strategy_id::text, bi.sort_order, COALESCE(ms.title, ''), COALESCE(ms.price_amount::text, '0')
		 FROM marketplace_bundle_items bi
		 LEFT JOIN marketplace_strategies ms ON ms.strategy_id = bi.strategy_id
		 WHERE bi.bundle_id = $1
		 ORDER BY bi.sort_order ASC`,
		bid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BundleItem
	for rows.Next() {
		var item BundleItem
		if err := rows.Scan(&item.StrategyID, &item.SortOrder, &item.Title, &item.PriceAmount); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
