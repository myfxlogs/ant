package strategy

import (
	"context"
	"errors"
	"strings"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
	"alphaforge/tools/mql2go"
	"alphaforge/tools/mql2go/interp"
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

// ── Import RPCs ──────────────────────────────────────────────────────
// Analysis path: MQL → CompileToIR → interp.Analyze (coverage/blind-spot report).
// Execution path: MQL → CompileMQL → Bytecode VM (ADR-0023 D2: analysis uses AST, execution uses Bytecode).
// Per ADR-0023 D4, Go code generation is no longer used at runtime.

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
		EntryRulesCount:  int32(rep.EntryRules),
		ExitRulesCount:   int32(rep.ExitRules),
		Params:           paramFields,
		Groups:           paramGroups,
		BlindSpots:       blindSpots,
		IndicatorNames:   rep.Indicators,
	}), nil
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

	strategyName := deriveStrategyName(req.Msg.GetSourceName())

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
			// Create initial version snapshot
			if s.versionRepo != nil {
				if _, vErr := s.versionRepo.CreateVersion(ctx, row.ID, uid, source, rep.Version, "Initial import"); vErr != nil {
					s.log.Warn("ImportStrategy: create version snapshot failed", zap.Error(vErr))
				}
			}
		}
	}

	return connect.NewResponse(&antv1.ImportStrategyResponse{
		StrategyId:    strategyID,
		StrategyName:  strategyName,
		GoCode:        source, // ADR-0023: return MQL source, not generated Go
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
