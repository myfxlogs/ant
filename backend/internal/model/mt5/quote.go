package mt5

import "google.golang.org/protobuf/types/known/timestamppb"

type Quote struct {
	Symbol string
	Bid    float64
	Ask    float64
	Time   *timestamppb.Timestamp
	Last   float64
	Volume uint64
}
