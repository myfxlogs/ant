package admin

import (
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestParseBacktestNotification_Degraded(t *testing.T) {
	t.Parallel()
	alert := parseBacktestNotification("DEGRADED|strategy-123|zero_volume_trade")
	if alert == nil {
		t.Fatal("DEGRADED payload should produce alert")
	}
	if alert.AlertType != "degraded" {
		t.Fatalf("alert type: want degraded, got %s", alert.AlertType)
	}
}

func TestParseBacktestNotification_Failed(t *testing.T) {
	t.Parallel()
	alert := parseBacktestNotification("FAILED|strategy-456|compile_error")
	if alert == nil {
		t.Fatal("FAILED payload should produce alert")
	}
	if alert.AlertType != "failed" {
		t.Fatalf("alert type: want failed, got %s", alert.AlertType)
	}
}

func TestParseBacktestNotification_Succeeded(t *testing.T) {
	t.Parallel()
	if alert := parseBacktestNotification("SUCCEEDED|strategy-123"); alert != nil {
		t.Fatal("SUCCEEDED payload should not produce alert")
	}
}

func TestParseBacktestNotification_Empty(t *testing.T) {
	t.Parallel()
	if alert := parseBacktestNotification(""); alert != nil {
		t.Fatal("empty payload should not produce alert")
	}
}

func TestParseBacktestNotification_HasTimestamp(t *testing.T) {
	t.Parallel()
	alert := parseBacktestNotification("DEGRADED|strategy|sig")
	if alert == nil {
		t.Fatal("expected alert")
	}
	if alert.Timestamp == nil {
		t.Fatal("alert should have timestamp")
	}
	if alert.Timestamp.Seconds <= 0 {
		t.Fatal("timestamp should be positive")
	}
	_ = timestamppb.Now()
}
