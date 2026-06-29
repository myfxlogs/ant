package strategy

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	"anttrader/internal/repository"
	"anttrader/tools/mql2go"
	"anttrader/tools/mql2go/interp"
)

// paramGroup guesses a parameter group from its name.
func paramGroup(name string) string {
	lower := strings.ToLower(name)
	if strings.Contains(lower, "lot") || strings.Contains(lower, "volume") {
		return "sizing"
	}
	if strings.Contains(lower, "magic") || strings.Contains(lower, "comment") {
		return "system"
	}
	if strings.Contains(lower, "sl") || strings.Contains(lower, "tp") ||
		strings.Contains(lower, "stop") || strings.Contains(lower, "take") {
		return "exit"
	}
	return "entry"
}

// ── IR-based import RPCs ─────────────────────────────────────────────
// All three RPCs use CompileToIR + interp.Analyze as the single source
// for coverage/blind-spot reporting. GenerateFromIR produces Go source
// from interp.IR — used as export preview only (not execution path).
// Execution goes through the interpreter: CompileToIR → interp.Interpreter.

func (s *StrategyExecutionServer) AnalyzeImportCode(ctx context.Context, req *connect.Request[antv1.AnalyzeImportCodeRequest]) (*connect.Response[antv1.AnalyzeImportCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
	}

	ir, err := mql2go.CompileToIR(source)
	if err != nil {
		s.log.Warn("AnalyzeImportCode: compile to IR failed", zap.Error(err))
		return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{}), nil
	}

	rep := interp.Analyze(ir)

	paramFields := irParamFields(rep.Params)
	paramGroups := irParamGroups(rep.Params)
	blindSpots := irBlindSpotProtos(rep.BlindSpots)

	return connect.NewResponse(&antv1.AnalyzeImportCodeResponse{
		StrategyName:     deriveStrategyName(req.Msg.GetSourceName()),
		MqlVersion:       rep.Version,
		CoverageScore:    rep.Coverage,
		TotalBlocks:      int32(rep.TotalCalls),
		RecognizedBlocks: int32(rep.SupportedCalls),
		ExecutionKind:    rep.ExecKind,
		Params:           paramFields,
		Groups:           paramGroups,
		BlindSpots:       blindSpots,
		IndicatorNames:   rep.Indicators,
	}), nil
}

func (s *StrategyExecutionServer) GenerateImportCode(ctx context.Context, req *connect.Request[antv1.GenerateImportCodeRequest]) (*connect.Response[antv1.GenerateImportCodeResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.GenerateImportCodeResponse{Compiles: false}), nil
	}

	// IR path: coverage/blind-spot report
	ir, err := mql2go.CompileToIR(source)
	if err != nil {
		s.log.Warn("GenerateImportCode: compile to IR failed", zap.Error(err))
		return connect.NewResponse(&antv1.GenerateImportCodeResponse{Compiles: false}), nil
	}
	rep := interp.Analyze(ir)

	// GenerateFromIR produces Go source for export preview (not execution).
	// Execution path: MQL → CompileToIR → interp.Interpreter (WASM harness).
	strategyName := deriveStrategyName(req.Msg.GetSourceName())
	code := mql2go.GenerateFromIR(ir, strategyName)
	lines := int32(strings.Count(code, "\n") + 1)

	// Compile verification on exported Go code (quality hint, not execution gate)
	compiles := false
	compileError := ""
	if s.goExecutor != nil {
		compiles, compileError = s.goExecutor.CompileCheck(ctx, code)
		if !compiles {
			s.log.Warn("GenerateImportCode: exported code failed compilation",
				zap.String("error", compileError))
		}
	} else {
		compiles = true
	}

	resp := &antv1.GenerateImportCodeResponse{
		GoCode:    code,
		CodeLines: lines,
		Compiles:  compiles,
	}
	if !compiles && compileError != "" {
		resp.QualityGateFailures = append(resp.QualityGateFailures, compileError)
	}
	for _, bs := range rep.BlindSpots {
		resp.QualityGateFailures = append(resp.QualityGateFailures,
			bs.Severity+": "+bs.Builtin+" not supported by interpreter")
	}
	return connect.NewResponse(resp), nil
}

func (s *StrategyExecutionServer) ImportStrategy(ctx context.Context, req *connect.Request[antv1.ImportStrategyRequest]) (*connect.Response[antv1.ImportStrategyResponse], error) {
	source := req.Msg.GetSourceCode()
	if source == "" {
		return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
	}

	ir, err := mql2go.CompileToIR(source)
	if err != nil {
		s.log.Warn("ImportStrategy: compile to IR failed", zap.Error(err))
		return connect.NewResponse(&antv1.ImportStrategyResponse{}), nil
	}

	rep := interp.Analyze(ir)

	// Go code generated from IR (export preview only, not execution path)
	strategyName := deriveStrategyName(req.Msg.GetSourceName())
	code := mql2go.GenerateFromIR(ir, strategyName)

	// Persist raw MQL as source of truth (if repo is configured).
	strategyID := uuid.New().String()
	if s.importedRepo != nil {
		uid, uidErr := userIDRequire(ctx)
		if uidErr != nil {
			return nil, uidErr
		}
		row := &repository.ImportedStrategy{
			UserID:        uid,
			Name:          strategyName,
			SourceLang:    rep.Version,
			SourceCode:    source,
			Params:        interp.SerializeParams(ir.Params),
			CoverageScore: rep.Coverage,
		}
		if err := s.importedRepo.Create(ctx, row); err != nil {
			s.log.Warn("ImportStrategy: persist failed", zap.Error(err))
		} else {
			strategyID = row.ID.String()
		}
	}

	return connect.NewResponse(&antv1.ImportStrategyResponse{
		StrategyId:    strategyID,
		StrategyName:  strategyName,
		GoCode:        code,
		CoverageScore: rep.Coverage,
		BlindSpots:    irBlindSpotProtos(rep.BlindSpots),
	}), nil
}

// GetImportedStrategy retrieves a previously imported strategy by ID.
func (s *StrategyExecutionServer) GetImportedStrategy(ctx context.Context, req *connect.Request[antv1.GetImportedStrategyRequest]) (*connect.Response[antv1.GetImportedStrategyResponse], error) {
	strategyID := req.Msg.GetStrategyId()
	if strategyID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("strategy_id is required"))
	}
	id, err := uuid.Parse(strategyID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if s.importedRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("imported strategy repository not configured"))
	}

	uid, uidErr := userIDRequire(ctx)
	if uidErr != nil {
		return nil, uidErr
	}

	row, err := s.importedRepo.GetByIDAndUser(ctx, id, uid)
	if err != nil {
		if errors.Is(err, repository.ErrImportedStrategyNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, err
	}

	return connect.NewResponse(&antv1.GetImportedStrategyResponse{
		StrategyId:    row.ID.String(),
		StrategyName:  row.Name,
		SourceLang:    row.SourceLang,
		SourceCode:    row.SourceCode,
		CoverageScore: row.CoverageScore,
		Params:        row.Params,
	}), nil
}

// ── IR → proto conversion helpers ────────────────────────────────────

func irParamFields(params []interp.ParamDecl) []*antv1.ParamField {
	fields := make([]*antv1.ParamField, 0, len(params))
	for _, p := range params {
		fields = append(fields, &antv1.ParamField{
			Name:         p.Name,
			Label:        p.Name,
			Type:         p.Type,
			DefaultValue: irParamDefault(p),
		})
	}
	return fields
}

func irParamGroups(params []interp.ParamDecl) []*antv1.ParamGroupInfo {
	seen := make(map[string]bool)
	var groups []*antv1.ParamGroupInfo
	for _, p := range params {
		g := paramGroup(p.Name)
		if !seen[g] {
			seen[g] = true
			groups = append(groups, &antv1.ParamGroupInfo{Name: g})
		}
	}
	return groups
}

func irBlindSpotProtos(spots []interp.IRBlindSpot) []*antv1.BlindSpot {
	result := make([]*antv1.BlindSpot, 0, len(spots))
	for _, bs := range spots {
		result = append(result, &antv1.BlindSpot{
			Id:          bs.Builtin,
			Category:    classifyBlindSpotCategory(bs.Builtin),
			Severity:    bs.Severity,
			Description: bs.Builtin + " is not supported by the interpreter",
		})
	}
	return result
}

func irParamDefault(p interp.ParamDecl) string {
	if p.Default == nil {
		return ""
	}
	// Evaluate the default expression literal directly
	return interp.EvalExprLiteral(p.Default)
}

func classifyBlindSpotCategory(name string) string {
	if strings.HasPrefix(name, "i") && len(name) > 1 && name[1] >= 'A' && name[1] <= 'Z' {
		return "indicator"
	}
	if strings.HasPrefix(name, "Order") || strings.HasPrefix(name, "Position") {
		return "trade"
	}
	if strings.HasPrefix(name, "Account") {
		return "account"
	}
	return "other"
}

func deriveStrategyName(sourceName string) string {
	if sourceName == "" {
		return ""
	}
	base := strings.TrimSuffix(strings.TrimSuffix(sourceName, ".mq4"), ".mq5")
	base = strings.TrimSuffix(base, ".mqh")
	if base == "" {
		return ""
	}
	return toCamelCase(base)
}
