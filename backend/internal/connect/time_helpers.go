package connect

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

func timeFromProto(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().UTC()
	if t.IsZero() {
		return nil
	}
	return &t
}
