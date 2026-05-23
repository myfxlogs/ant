package mt4

import "google.golang.org/protobuf/types/known/timestamppb"

type Quote struct {
	Symbol string
	Bid    float64
	Ask    float64
	Time   *timestamppb.Timestamp
	High   float64
	Low    float64
}
