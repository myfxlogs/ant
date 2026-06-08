// Package ai — equity curve to daily returns conversion.
package ai

import (
	"google.golang.org/protobuf/proto"

	antv1 "anttrader/gen/proto/ant/v1"
)

// EquityCurveToDailyReturns extracts equity curve from proto binary ExecuteBacktestResponse
// and converts it to daily return series (equity[i] - equity[i-1]).
func EquityCurveToDailyReturns(protoResp []byte) []float64 {
	if len(protoResp) == 0 {
		return nil
	}
	var resp antv1.ExecuteBacktestResponse
	if err := proto.Unmarshal(protoResp, &resp); err != nil {
		return nil
	}
	equity := resp.GetEquityCurve()
	if len(equity) < 2 {
		return nil
	}
	rets := make([]float64, len(equity)-1)
	for i := 1; i < len(equity); i++ {
		rets[i-1] = equity[i] - equity[i-1]
	}
	return rets
}
