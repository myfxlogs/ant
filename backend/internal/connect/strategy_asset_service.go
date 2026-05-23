package connect

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "anttrader/gen/proto"
	"anttrader/internal/model"
	"anttrader/internal/repository"
	"anttrader/internal/service"
)

type StrategyAssetService struct {
	repo        *repository.StrategyAssetRepository
	templateSvc *service.StrategyTemplateService
}

func NewStrategyAssetService(repo *repository.StrategyAssetRepository, templateSvc *service.StrategyTemplateService) *StrategyAssetService {
	return &StrategyAssetService{repo: repo, templateSvc: templateSvc}
}

func (s *StrategyAssetService) ListStrategyAssets(ctx context.Context, req *connect.Request[v1.ListStrategyAssetsRequest]) (*connect.Response[v1.ListStrategyAssetsResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.List(ctx, userID, int(req.Msg.GetLimit()), int(req.Msg.GetOffset()))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.StrategyAsset, 0, len(rows))
	for i := range rows {
		out = append(out, strategyAssetToProto(&rows[i]))
	}
	return connect.NewResponse(&v1.ListStrategyAssetsResponse{Assets: out}), nil
}

func (s *StrategyAssetService) GetStrategyAsset(ctx context.Context, req *connect.Request[v1.GetStrategyAssetRequest]) (*connect.Response[v1.StrategyAsset], error) {
	asset, err := s.getAsset(ctx, req.Msg.GetAssetId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(strategyAssetToProto(asset)), nil
}

func (s *StrategyAssetService) SubmitAssetReview(ctx context.Context, req *connect.Request[v1.SubmitAssetReviewRequest]) (*connect.Response[v1.StrategyAsset], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	templateID, err := parseRequestUUID(req.Msg.GetSourceTemplateId())
	if err != nil {
		return nil, err
	}
	tpl, err := s.templateSvc.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if tpl.UserID != userID {
		return nil, connect.NewError(connect.CodePermissionDenied, service.ErrTemplateUnauthorized)
	}
	visibility := strings.TrimSpace(req.Msg.GetVisibility())
	if visibility == "" {
		visibility = "private"
	}
	row := &repository.StrategyAsset{OwnerUserID: userID, SourceTemplateID: templateID, Name: strings.TrimSpace(req.Msg.GetName()), Description: req.Msg.GetDescription(), Visibility: visibility, ReviewStatus: "pending", SourceVersion: 1, LatestVersion: 1}
	if row.Name == "" {
		row.Name = tpl.Name
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(strategyAssetToProto(row)), nil
}

func (s *StrategyAssetService) ReviewStrategyAsset(ctx context.Context, req *connect.Request[v1.ReviewStrategyAssetRequest]) (*connect.Response[v1.StrategyAsset], error) {
	id, err := parseRequestUUID(req.Msg.GetAssetId())
	if err != nil {
		return nil, err
	}
	status := strings.TrimSpace(req.Msg.GetReviewStatus())
	if status == "" {
		status = "approved"
	}
	row, err := s.repo.Review(ctx, id, status, req.Msg.GetRatingSummary())
	if err != nil {
		return nil, strategyAssetError(err)
	}
	return connect.NewResponse(strategyAssetToProto(row)), nil
}

func (s *StrategyAssetService) CloneStrategyAsset(ctx context.Context, req *connect.Request[v1.CloneStrategyAssetRequest]) (*connect.Response[v1.CloneStrategyAssetResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	asset, err := s.getAsset(ctx, req.Msg.GetAssetId())
	if err != nil {
		return nil, err
	}
	source, err := s.templateSvc.GetTemplate(ctx, asset.SourceTemplateID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	name := strings.TrimSpace(req.Msg.GetName())
	if name == "" {
		name = fmt.Sprintf("%s 副本", asset.Name)
	}
	tpl, err := s.templateSvc.CreateTemplate(ctx, userID, name, asset.Description, source.Code, nil, false, []string{"asset_clone"})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	tpl.Status = model.StrategyTemplateStatusDraft
	_ = s.templateSvc.UpdateTemplate(ctx, tpl)
	clone := &repository.StrategyAssetClone{AssetID: asset.ID, UserID: userID, ClonedTemplateID: tpl.ID, SourceVersion: asset.LatestVersion}
	if err := s.repo.CreateClone(ctx, clone); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CloneStrategyAssetResponse{TemplateId: tpl.ID.String(), Clone: strategyAssetCloneToProto(clone)}), nil
}

func (s *StrategyAssetService) CheckAssetUpdate(ctx context.Context, req *connect.Request[v1.CheckAssetUpdateRequest]) (*connect.Response[v1.StrategyAssetClone], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseRequestUUID(req.Msg.GetCloneId())
	if err != nil {
		return nil, err
	}
	clone, err := s.repo.GetClone(ctx, userID, id)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	asset, err := s.repo.Get(ctx, userID, clone.AssetID)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	clone, err = s.repo.UpdateCloneSync(ctx, clone.ID, clone.SourceVersion, asset.LatestVersion > clone.SourceVersion)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	return connect.NewResponse(strategyAssetCloneToProto(clone)), nil
}

func (s *StrategyAssetService) SyncStrategyAsset(ctx context.Context, req *connect.Request[v1.SyncStrategyAssetRequest]) (*connect.Response[v1.StrategyAssetClone], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	id, err := parseRequestUUID(req.Msg.GetCloneId())
	if err != nil {
		return nil, err
	}
	clone, err := s.repo.GetClone(ctx, userID, id)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	asset, err := s.repo.Get(ctx, userID, clone.AssetID)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	clone, err = s.repo.UpdateCloneSync(ctx, clone.ID, asset.LatestVersion, false)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	return connect.NewResponse(strategyAssetCloneToProto(clone)), nil
}

func (s *StrategyAssetService) ListAssetClones(ctx context.Context, req *connect.Request[v1.ListAssetClonesRequest]) (*connect.Response[v1.ListAssetClonesResponse], error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	assetID, err := parseRequestUUID(req.Msg.GetAssetId())
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListClones(ctx, userID, assetID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := make([]*v1.StrategyAssetClone, 0, len(rows))
	for i := range rows {
		out = append(out, strategyAssetCloneToProto(&rows[i]))
	}
	return connect.NewResponse(&v1.ListAssetClonesResponse{Clones: out}), nil
}

func (s *StrategyAssetService) getAsset(ctx context.Context, id string) (*repository.StrategyAsset, error) {
	userID, err := getUserIDFromContext(ctx)
	if err != nil {
		return nil, err
	}
	assetID, err := parseRequestUUID(id)
	if err != nil {
		return nil, err
	}
	asset, err := s.repo.Get(ctx, userID, assetID)
	if err != nil {
		return nil, strategyAssetError(err)
	}
	return asset, nil
}

func strategyAssetToProto(row *repository.StrategyAsset) *v1.StrategyAsset {
	if row == nil {
		return nil
	}
	return &v1.StrategyAsset{Id: row.ID.String(), OwnerUserId: row.OwnerUserID.String(), SourceTemplateId: row.SourceTemplateID.String(), SourceVersion: int32(row.SourceVersion), Name: row.Name, Description: row.Description, Visibility: row.Visibility, ReviewStatus: row.ReviewStatus, RatingSummary: row.RatingSummary, CloneCount: int32(row.CloneCount), LatestVersion: int32(row.LatestVersion), CreatedAt: timestamppb.New(row.CreatedAt.UTC()), UpdatedAt: timestamppb.New(row.UpdatedAt.UTC())}
}

func strategyAssetCloneToProto(row *repository.StrategyAssetClone) *v1.StrategyAssetClone {
	if row == nil {
		return nil
	}
	out := &v1.StrategyAssetClone{Id: row.ID.String(), AssetId: row.AssetID.String(), UserId: row.UserID.String(), ClonedTemplateId: row.ClonedTemplateID.String(), SourceVersion: int32(row.SourceVersion), SyncAvailable: row.SyncAvailable, CreatedAt: timestamppb.New(row.CreatedAt.UTC())}
	if row.LastSyncCheckAt != nil {
		out.LastSyncCheckAt = timestamppb.New(row.LastSyncCheckAt.UTC())
	}
	return out
}

func strategyAssetError(err error) error {
	if errors.Is(err, repository.ErrStrategyAssetNotFound) || errors.Is(err, repository.ErrStrategyAssetCloneNotFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}
	return connect.NewError(connect.CodeInternal, err)
}
