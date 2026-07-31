package backfiller

import (
	"context"
	"fmt"

	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// BarAggregatorTarget is the interface that bar_aggregator provides for
// external bar ingestion with finality check (ADR-0009 §2.2).
type BarAggregatorTarget interface {
	IngestExternalBar(b *mdtick.Bar) bool
}

// PublisherTarget publishes bars to NATS (with X-Ant-Replay header and trace context).
type PublisherTarget interface {
	PublishBar(ctx context.Context, b *mdtick.Bar) error
}

// BarEnqueuer enqueues bars for storage insertion.
type BarEnqueuer interface {
	EnqueueBar(b *mdtick.Bar)
}

// TargetAdapter composes the downstream targets for backfilled bars.
// Each bar is: 1) finality-checked by aggregator, 2) published to NATS,
// 3) enqueued for PG storage.
type TargetAdapter struct {
	agg       BarAggregatorTarget
	publisher PublisherTarget
	pgWriter  BarEnqueuer
}

// NewTarget creates a TargetAdapter.
func NewTarget(agg BarAggregatorTarget, pub PublisherTarget, pgw BarEnqueuer) *TargetAdapter {
	return &TargetAdapter{agg: agg, publisher: pub, pgWriter: pgw}
}

// IngestBar routes a backfilled bar through the targets.
// Bars are skipped if their close_ts <= the finalized ceiling (ADR-0009 §2.2).
func (ta *TargetAdapter) IngestBar(ctx context.Context, bar *mdtick.Bar) error {
	// 1. Finality check.
	if !ta.agg.IngestExternalBar(bar) {
		return nil // skipped (already finalized)
	}

	// 2. Publish to NATS (with X-Ant-Replay header already set).
	if err := ta.publisher.PublishBar(ctx, bar); err != nil {
		return fmt.Errorf("publish backfilled bar to NATS: %w", err)
	}

	// 3. Enqueue for PG storage.
	ta.pgWriter.EnqueueBar(bar)
	return nil
}
