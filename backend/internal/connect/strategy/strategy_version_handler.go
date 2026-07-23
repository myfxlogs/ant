package strategy

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/repository"
)

// ListStrategyVersions returns version history for a strategy.
func (s *StrategyExecutionServer) ListStrategyVersions(ctx context.Context, req *connect.Request[antv1.ListStrategyVersionsRequest]) (*connect.Response[antv1.ListStrategyVersionsResponse], error) {
	if s.versionRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("version repository not configured"))
	}
	strategyID, err := uuid.Parse(req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
	}
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	limit := int(req.Msg.GetLimit())
	offset := int(req.Msg.GetOffset())
	versions, err := s.versionRepo.ListVersions(ctx, strategyID, uid, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	total, _ := s.versionRepo.CountVersions(ctx, strategyID, uid)

	resp := &antv1.ListStrategyVersionsResponse{Total: int32(total)}
	for i := range versions {
		resp.Versions = append(resp.Versions, metaToProto(&versions[i]))
	}
	return connect.NewResponse(resp), nil
}

// GetStrategyVersion retrieves a specific version snapshot.
func (s *StrategyExecutionServer) GetStrategyVersion(ctx context.Context, req *connect.Request[antv1.GetStrategyVersionRequest]) (*connect.Response[antv1.GetStrategyVersionResponse], error) {
	if s.versionRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("version repository not configured"))
	}
	strategyID, err := uuid.Parse(req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
	}
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	v, err := s.versionRepo.GetVersion(ctx, strategyID, uid, int(req.Msg.GetVersionNumber()))
	if err != nil {
		if errors.Is(err, repository.ErrStrategyVersionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.GetStrategyVersionResponse{
		Version:    versionToProto(v),
		SourceCode: v.SourceCode,
	}), nil
}

// RollbackStrategyVersion restores strategy code from a specific version.
func (s *StrategyExecutionServer) RollbackStrategyVersion(ctx context.Context, req *connect.Request[antv1.RollbackStrategyVersionRequest]) (*connect.Response[antv1.RollbackStrategyVersionResponse], error) {
	if s.versionRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("version repository not configured"))
	}
	strategyID, err := uuid.Parse(req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
	}
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	newVer, err := s.versionRepo.RollbackToVersion(ctx, strategyID, uid, int(req.Msg.GetVersionNumber()))
	if err != nil {
		if errors.Is(err, repository.ErrStrategyVersionNotFound) || errors.Is(err, repository.ErrImportedStrategyNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.RollbackStrategyVersionResponse{
		NewVersion:         versionToProto(newVer),
		RestoredSourceCode: newVer.SourceCode,
	}), nil
}

// DiffStrategyVersions returns two version snapshots for client-side diffing.
func (s *StrategyExecutionServer) DiffStrategyVersions(ctx context.Context, req *connect.Request[antv1.DiffStrategyVersionsRequest]) (*connect.Response[antv1.DiffStrategyVersionsResponse], error) {
	if s.versionRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("version repository not configured"))
	}
	strategyID, err := uuid.Parse(req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
	}
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	from, to, err := s.versionRepo.DiffVersions(ctx, strategyID, uid,
		int(req.Msg.GetFromVersion()), int(req.Msg.GetToVersion()))
	if err != nil {
		if errors.Is(err, repository.ErrStrategyVersionNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.DiffStrategyVersionsResponse{
		FromVersion:    versionToProto(from),
		FromSourceCode: from.SourceCode,
		ToVersion:      versionToProto(to),
		ToSourceCode:   to.SourceCode,
	}), nil
}

// UpdateStrategyCode updates the source code of an existing strategy and creates a version snapshot.
func (s *StrategyExecutionServer) UpdateStrategyCode(ctx context.Context, req *connect.Request[antv1.UpdateStrategyCodeRequest]) (*connect.Response[antv1.UpdateStrategyCodeResponse], error) {
	if s.versionRepo == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("version repository not configured"))
	}
	strategyID, err := uuid.Parse(req.Msg.GetStrategyId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid strategy_id: %w", err))
	}
	uid, err := userIDRequire(ctx)
	if err != nil {
		return nil, err
	}

	sourceCode := req.Msg.GetSourceCode()
	if sourceCode == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("source_code is required"))
	}

	v, err := s.versionRepo.UpdateStrategyCode(ctx, strategyID, uid, sourceCode, req.Msg.GetChangeSummary())
	if err != nil {
		if errors.Is(err, repository.ErrImportedStrategyNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.UpdateStrategyCodeResponse{
		NewVersion: versionToProto(v),
	}), nil
}

func versionToProto(v *repository.StrategyVersion) *antv1.StrategyVersionInfo {
	info := &antv1.StrategyVersionInfo{
		VersionId:     v.ID.String(),
		VersionNumber: int32(v.VersionNumber),
		SourceLang:    v.SourceLang,
		ChangeSummary: v.ChangeSummary,
		CodeHash:      v.CodeHash,
	}
	if !v.CreatedAt.IsZero() {
		info.CreatedAt = timestamppb.New(v.CreatedAt)
	}
	return info
}

func metaToProto(m *repository.StrategyVersionMeta) *antv1.StrategyVersionInfo {
	info := &antv1.StrategyVersionInfo{
		VersionId:     m.ID.String(),
		VersionNumber: int32(m.VersionNumber),
		SourceLang:    m.SourceLang,
		ChangeSummary: m.ChangeSummary,
		CodeHash:      m.CodeHash,
	}
	if !m.CreatedAt.IsZero() {
		info.CreatedAt = timestamppb.New(m.CreatedAt)
	}
	return info
}
