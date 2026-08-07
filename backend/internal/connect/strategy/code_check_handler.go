package strategy

import (
	"context"
	"fmt"

	"connectrpc.com/connect"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
)

// CheckCode compiles MQL source and returns diagnostics + blind spots without saving.
// Used for real-time editor feedback (debounced) and pre-save audit.
func (s *StrategyExecutionServer) CheckCode(ctx context.Context, req *connect.Request[antv1.CheckCodeRequest]) (*connect.Response[antv1.CheckCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.CheckCodeResponse{CompileSuccess: true}), nil
	}

	resp := compileAndAudit(source)
	return connect.NewResponse(resp), nil
}

// compileAndAudit compiles MQL source and returns a CheckCodeResponse with
// coverage score, blind spots, and indicator names. Compile errors are encoded
// in the response (CompileSuccess=false), not returned as Go errors.
func compileAndAudit(source string) *antv1.CheckCodeResponse {
	ir, err := mql2go.CompileToIR(source)
	if err != nil {
		return &antv1.CheckCodeResponse{
			CompileSuccess: false,
			CompileError:   fmt.Sprintf("parse error: %v", err),
		}
	}

	bc, compileErr := mql2go.CompileAST(ir)
	if compileErr != nil {
		rep := interp.Analyze(ir)
		blindSpots := irBlindSpotProtos(rep.BlindSpots)
		blindSpots = append(blindSpots, compileErrorBlindSpot(compileErr))
		return &antv1.CheckCodeResponse{
			CompileSuccess: false,
			CompileError:   compileErr.Error(),
			MqlVersion:     rep.Version,
			BlindSpots:     blindSpots,
			IndicatorNames: rep.Indicators,
		}
	}

	cov := mql2go.AnalyzeCoverage(ir, bc)
	blindSpots := coverageBlindSpotProtos(cov.BlindSpots)

	// Add lookahead violations as blind spots for real-time editor feedback.
	for _, lv := range cov.LookaheadViolations {
		blindSpots = append(blindSpots, &antv1.BlindSpot{
			Id:          "lookahead_" + lv.Function,
			Category:    "lookahead",
			Severity:    lv.Severity,
			Description: lv.Message,
			Location:    lv.ShiftExpr,
		})
	}

	return &antv1.CheckCodeResponse{
		CompileSuccess:   true,
		CoverageScore:    cov.Score,
		TotalBlocks:      int32(cov.TotalCalls),
		RecognizedBlocks: int32(cov.SupportedCalls),
		BlindSpots:       blindSpots,
		MqlVersion:       cov.Version,
		IndicatorNames:   cov.Indicators,
	}
}
