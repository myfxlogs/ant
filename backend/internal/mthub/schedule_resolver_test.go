package mthub

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stubResolver struct {
	id    *uuid.UUID
	err   error
	calls int
}

func (s *stubResolver) ResolveScheduleIDByMagic(_ context.Context, _ uuid.UUID, magic int32) (*uuid.UUID, error) {
	s.calls++
	if magic == 0 {
		return nil, nil
	}
	return s.id, s.err
}

func TestResolveScheduleIDMagicZero(t *testing.T) {
	r := &stubResolver{id: nil}
	got := ResolveScheduleID(context.Background(), r, nil, uuid.New(), 0)
	if got != nil {
		t.Fatalf("expected nil for magic=0, got %v", got)
	}
	if r.calls != 0 {
		t.Fatalf("resolver should not be called for magic=0, calls=%d", r.calls)
	}
}

func TestResolveScheduleIDNilResolver(t *testing.T) {
	got := ResolveScheduleID(context.Background(), nil, nil, uuid.New(), 12345)
	if got != nil {
		t.Fatalf("expected nil for nil resolver, got %v", got)
	}
}

func TestResolveScheduleIDSuccess(t *testing.T) {
	sid := uuid.New()
	r := &stubResolver{id: &sid}
	got := ResolveScheduleID(context.Background(), r, zap.NewNop(), uuid.New(), 12345)
	if got == nil || *got != sid {
		t.Fatalf("expected %s, got %v", sid, got)
	}
	if r.calls != 1 {
		t.Fatalf("expected 1 call, got %d", r.calls)
	}
}
