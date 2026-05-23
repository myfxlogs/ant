package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"google.golang.org/protobuf/encoding/protojson"

	v1 "anttrader/gen/proto"
	"anttrader/internal/repository"
)

type TickDatasetRepo interface {
	Create(ctx context.Context, ds *repository.TickDataset) (uuid.UUID, error)
	GetByID(ctx context.Context, id uuid.UUID) (*repository.TickDataset, error)
	SetFrozen(ctx context.Context, id uuid.UUID, frozen bool) error
	BatchInsertTicks(ctx context.Context, ticks []*repository.TickDatasetTick) error
	ListTicks(ctx context.Context, datasetID uuid.UUID, limit int) ([]*repository.TickDatasetTick, error)
}

type TickDatasetService struct {
	repo  TickDatasetRepo
	rdb   *redis.Client
	clock func() time.Time
}

func NewTickDatasetService(repo TickDatasetRepo, rdb *redis.Client) *TickDatasetService {
	return &TickDatasetService{repo: repo, rdb: rdb, clock: time.Now}
}

func (s *TickDatasetService) streamKey(accountID uuid.UUID) string {
	return "antrader:events:account:" + accountID.String()
}

func parseQuoteTime(s string) (time.Time, bool) {
	if s == "" {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, true
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, true
	}
	return time.Time{}, false
}

func (s *TickDatasetService) CreateFrozenTickDatasetFromRedis(
	ctx context.Context,
	userID uuid.UUID,
	accountID uuid.UUID,
	symbol string,
	from time.Time,
	to time.Time,
	limit int,
) (uuid.UUID, error) {
	if s == nil || s.repo == nil {
		return uuid.Nil, errors.New("tick dataset service not available")
	}
	if s.rdb == nil {
		return uuid.Nil, errors.New("redis not available")
	}
	if symbol == "" {
		return uuid.Nil, errors.New("symbol required")
	}
	if to.Before(from) {
		return uuid.Nil, errors.New("invalid range")
	}

	ds := &repository.TickDataset{
		ID:        uuid.New(),
		UserID:    userID,
		AccountID: accountID,
		Symbol:    symbol,
		FromTime:  from,
		ToTime:    to,
		Frozen:    false,
	}
	dsid, err := s.repo.Create(ctx, ds)
	if err != nil {
		return uuid.Nil, err
	}

	start := from.UTC().UnixMilli()
	end := to.UTC().UnixMilli()
	minID := fmtStreamID(start, 0)
	maxID := fmtStreamID(end, 999999)

	res, err := s.rdb.XRange(ctx, s.streamKey(accountID), minID, maxID).Result()
	if err != nil {
		return uuid.Nil, err
	}

	ticks := make([]*repository.TickDatasetTick, 0, len(res))
	for _, msg := range res {
		sev := &v1.StreamEvent{}
		raw, ok := msg.Values["event"].(string)
		if !ok || raw == "" {
			continue
		}
		if err := protojson.Unmarshal([]byte(raw), sev); err != nil {
			continue
		}
		if sev.GetType() != "quote_tick" || sev.GetQuote() == nil {
			continue
		}
		q := sev.GetQuote()
		if q.GetSymbol() != symbol {
			continue
		}
		tm, ok := parseQuoteTime(q.GetTime())
		if !ok {
			if sev.Timestamp != nil {
				tm = sev.Timestamp.AsTime()
			} else {
				continue
			}
		}
		tm = tm.UTC()
		if tm.Before(from.UTC()) || tm.After(to.UTC()) {
			continue
		}
		ticks = append(ticks, &repository.TickDatasetTick{DatasetID: dsid, Time: tm, Bid: q.GetBid(), Ask: q.GetAsk(), CreatedAt: s.clock()})
		if limit > 0 && len(ticks) >= limit {
			break
		}
	}

	if err := s.repo.BatchInsertTicks(ctx, ticks); err != nil {
		return uuid.Nil, err
	}
	if err := s.repo.SetFrozen(ctx, dsid, true); err != nil {
		return uuid.Nil, err
	}
	return dsid, nil
}

func (s *TickDatasetService) GetFrozenTickDatasetTicks(ctx context.Context, userID uuid.UUID, datasetID uuid.UUID, limit int) ([]*repository.TickDatasetTick, error) {
	if s == nil || s.repo == nil {
		return nil, errors.New("tick dataset service not available")
	}
	ds, err := s.repo.GetByID(ctx, datasetID)
	if err != nil {
		return nil, err
	}
	if ds.UserID != userID {
		return nil, ErrUnauthorized
	}
	return s.repo.ListTicks(ctx, datasetID, limit)
}

func fmtStreamID(ms int64, seq int64) string {
	return fmt.Sprintf("%d-%d", ms, seq)
}
