package connect

import (
	"testing"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
)

func TestStreamCursorForXRead(t *testing.T) {
	if got := streamCursorForXRead(""); got != "$" {
		t.Fatalf("expected $, got=%q", got)
	}
	if got := streamCursorForXRead("0-0"); got != "0-0" {
		t.Fatalf("expected 0-0, got=%q", got)
	}
}

func TestParseStreamEventFromValues_PopulatesEventID(t *testing.T) {
	sev := &v1.StreamEvent{Type: "profit_update", AccountId: "a1"}
	blob, err := protojson.Marshal(sev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := parseStreamEventFromValues("123-0", map[string]any{"event": string(blob)})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil")
	}
	if got.EventId != "123-0" {
		t.Fatalf("expected event_id=123-0, got=%q", got.EventId)
	}
	if got.Type != "profit_update" {
		t.Fatalf("expected type profit_update, got=%q", got.Type)
	}
	if got.AccountId != "a1" {
		t.Fatalf("expected account_id a1, got=%q", got.AccountId)
	}
}

func TestParseStreamEventFromValues_FillsMissingFieldsFromRedis(t *testing.T) {
	sev := &v1.StreamEvent{}
	blob, err := protojson.Marshal(sev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := parseStreamEventFromValues("123-0", map[string]any{
		"event":      string(blob),
		"type":       "order_update",
		"account_id": "a2",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil")
	}
	if got.Type != "order_update" {
		t.Fatalf("expected type order_update, got=%q", got.Type)
	}
	if got.AccountId != "a2" {
		t.Fatalf("expected account_id a2, got=%q", got.AccountId)
	}
}

func TestParseStreamEventFromValues_FillsTimestampFromRedisTS(t *testing.T) {
	sev := &v1.StreamEvent{}
	blob, err := protojson.Marshal(sev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	got, err := parseStreamEventFromValues("123-0", map[string]any{
		"event": string(blob),
		"ts":    "1700000000000",
	})
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil")
	}
	if got.Timestamp == nil {
		t.Fatalf("expected timestamp to be filled")
	}
	if got.Timestamp.AsTime().UnixMilli() != 1700000000000 {
		t.Fatalf("expected ts=1700000000000, got=%d", got.Timestamp.AsTime().UnixMilli())
	}
	// Ensure it's a real protobuf timestamp.
	_ = timestamppb.New(got.Timestamp.AsTime())
}
