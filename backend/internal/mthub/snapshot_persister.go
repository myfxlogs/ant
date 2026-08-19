package mthub

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// SnapshotPersister subscribes to PositionSnapshotBroker and writes snapshots
// to mt_position_snapshots table with a 30-second throttle per account.
// During mtapi disconnection, the last-known snapshot is served from PG.
type SnapshotPersister struct {
	broker   *PositionSnapshotBroker
	pool     *pgxpool.Pool
	log      *zap.Logger
	throttle time.Duration

	mu        sync.Mutex
	lastWrite map[string]time.Time
}

// NewSnapshotPersister creates a persister with the given throttle interval.
func NewSnapshotPersister(broker *PositionSnapshotBroker, pool *pgxpool.Pool, log *zap.Logger) *SnapshotPersister {
	if log == nil {
		log = zap.NewNop()
	}
	return &SnapshotPersister{
		broker:    broker,
		pool:      pool,
		log:       log,
		throttle:  30 * time.Second,
		lastWrite: make(map[string]time.Time),
	}
}

// Start subscribes to all position snapshots and persists them to PG.
// Blocks until ctx is cancelled.
func (p *SnapshotPersister) Start(ctx context.Context) {
	ch, cancel := p.broker.SubscribeAll()
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return
		case snap, ok := <-ch:
			if !ok {
				return
			}
			p.persist(ctx, snap)
		}
	}
}

func (p *SnapshotPersister) persist(ctx context.Context, snap *PositionSnapshot) {
	if snap == nil || !snap.FinancialsAuthoritative {
		return
	}
	p.mu.Lock()
	last, exists := p.lastWrite[snap.AccountID]
	if exists && time.Since(last) < p.throttle {
		p.mu.Unlock()
		return
	}
	p.lastWrite[snap.AccountID] = Clk.Now()
	p.mu.Unlock()

	record := snapshotToProto(snap)
	data, err := proto.Marshal(record)
	if err != nil {
		p.log.Error("snapshot persister: marshal failed",
			zap.String("account", snap.AccountID), zap.Error(err))
		return
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = p.pool.Exec(writeCtx,
		`INSERT INTO mt_position_snapshots (account_id, payload_proto, captured_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (account_id) DO UPDATE SET payload_proto = EXCLUDED.payload_proto, captured_at = EXCLUDED.captured_at`,
		snap.AccountID, data, Clk.Now())
	if err != nil {
		p.log.Error("snapshot persister: DB write failed",
			zap.String("account", snap.AccountID), zap.Error(err))
	}
}

// GetSnapshot retrieves the last-known position snapshot for an account from PG.
// Returns nil if no snapshot exists.
func (p *SnapshotPersister) GetSnapshot(ctx context.Context, accountID string) (*antv1.MtPositionSnapshotRecord, error) {
	var data []byte
	err := p.pool.QueryRow(ctx,
		`SELECT payload_proto FROM mt_position_snapshots WHERE account_id = $1`, accountID).Scan(&data)
	if err != nil {
		return nil, err
	}
	record := &antv1.MtPositionSnapshotRecord{}
	if err := proto.Unmarshal(data, record); err != nil {
		return nil, err
	}
	return record, nil
}

func snapshotToProto(snap *PositionSnapshot) *antv1.MtPositionSnapshotRecord {
	record := &antv1.MtPositionSnapshotRecord{
		AccountId:   snap.AccountID,
		UserId:      snap.UserID,
		Platform:    snap.Platform,
		Balance:     snap.Balance.String(),
		Credit:      snap.Credit.String(),
		Equity:      snap.Equity.String(),
		Margin:      snap.Margin.String(),
		FreeMargin:  snap.FreeMargin.String(),
		MarginLevel: snap.MarginLevel.String(),
		Profit:      snap.Profit.String(),
		CapturedAt:  timestamppb.New(snap.CapturedAt),
	}
	for _, pos := range snap.Positions {
		record.Positions = append(record.Positions, &antv1.MtPositionSnapshotItem{
			Ticket:       pos.Ticket,
			Symbol:       pos.Symbol,
			Type:         pos.Type,
			MagicNumber:  int64(pos.Magic),
			Volume:       pos.Volume.String(),
			OpenPrice:    pos.OpenPrice.String(),
			CurrentPrice: pos.CurrentPrice.String(),
			StopLoss:     pos.StopLoss.String(),
			TakeProfit:   pos.TakeProfit.String(),
			Profit:       pos.Profit.String(),
			Swap:         pos.Swap.String(),
			Commission:   pos.Commission.String(),
			Comment:      pos.Comment,
			OpenTime:     pos.OpenTime,
		})
	}
	return record
}
