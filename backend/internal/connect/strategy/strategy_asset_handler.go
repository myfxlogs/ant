package connect

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
	"anttrader/internal/interceptor"
	"anttrader/internal/repository"
)

// StrategyAssetServer implements ant.v1.StrategyAssetServiceHandler.
type StrategyAssetServer struct {
	repo *repository.StrategyAssetRepository
	log  *zap.Logger
}

var _ antv1c.StrategyAssetServiceHandler = (*StrategyAssetServer)(nil)

func NewStrategyAssetServer(repo *repository.StrategyAssetRepository, log *zap.Logger) *StrategyAssetServer {
	return &StrategyAssetServer{repo: repo, log: log}
}

func (s *StrategyAssetServer) userID(ctx context.Context) uuid.UUID {
	id, _ := uuid.Parse(interceptor.GetUserID(ctx))
	return id
}

func assetToProto(a *repository.StrategyAsset) *antv1.StrategyAsset {
	return &antv1.StrategyAsset{
		Id:               a.ID.String(),
		OwnerUserId:      a.OwnerUserID.String(),
		SourceTemplateId: a.SourceTemplateID.String(),
		SourceVersion:    int32(a.SourceVersion),
		Name:             a.Name,
		Description:      a.Description,
		Visibility:       a.Visibility,
		ReviewStatus:     a.ReviewStatus,
		RatingSummary:    a.RatingSummary,
		CloneCount:       int32(a.CloneCount),
		LatestVersion:    int32(a.LatestVersion),
		CreatedAt:        timestamppb.New(a.CreatedAt),
		UpdatedAt:        timestamppb.New(a.UpdatedAt),
	}
}

func cloneToProto(c *repository.StrategyAssetClone) *antv1.StrategyAssetClone {
	p := &antv1.StrategyAssetClone{
		Id:               c.ID.String(),
		AssetId:          c.AssetID.String(),
		UserId:           c.UserID.String(),
		ClonedTemplateId: c.ClonedTemplateID.String(),
		SourceVersion:    int32(c.SourceVersion),
		SyncAvailable:    c.SyncAvailable,
		CreatedAt:        timestamppb.New(c.CreatedAt),
	}
	if c.LastSyncCheckAt != nil {
		p.LastSyncCheckAt = timestamppb.New(*c.LastSyncCheckAt)
	}
	return p
}

func (s *StrategyAssetServer) ListStrategyAssets(ctx context.Context, req *connect.Request[antv1.ListStrategyAssetsRequest]) (*connect.Response[antv1.ListStrategyAssetsResponse], error) {
	limit := int(req.Msg.Limit)
	offset := int(req.Msg.Offset)
	rows, err := s.repo.List(ctx, s.userID(ctx), limit, offset)
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.StrategyAsset, len(rows))
	for i := range rows {
		items[i] = assetToProto(&rows[i])
	}
	return connect.NewResponse(&antv1.ListStrategyAssetsResponse{Assets: items}), nil
}

func (s *StrategyAssetServer) GetStrategyAsset(ctx context.Context, req *connect.Request[antv1.GetStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.AssetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	a, err := s.repo.Get(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(assetToProto(a)), nil
}

func (s *StrategyAssetServer) SubmitAssetReview(ctx context.Context, req *connect.Request[antv1.SubmitAssetReviewRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	uid := s.userID(ctx)
	var sourceID uuid.UUID
	if req.Msg.SourceTemplateId != "" {
		id, err := uuid.Parse(req.Msg.SourceTemplateId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		sourceID = id
	}
	row := &repository.StrategyAsset{
		OwnerUserID:      uid,
		SourceTemplateID: sourceID,
		Name:             req.Msg.Name,
		Description:      req.Msg.Description,
		Visibility:       req.Msg.Visibility,
		ReviewStatus:     "pending_review",
	}
	if err := s.repo.Create(ctx, row); err != nil {
		return nil, err
	}
	return connect.NewResponse(assetToProto(row)), nil
}

func (s *StrategyAssetServer) ReviewStrategyAsset(ctx context.Context, req *connect.Request[antv1.ReviewStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	id, err := uuid.Parse(req.Msg.AssetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	a, err := s.repo.Review(ctx, id, req.Msg.ReviewStatus, req.Msg.RatingSummary)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(assetToProto(a)), nil
}

func (s *StrategyAssetServer) CloneStrategyAsset(ctx context.Context, req *connect.Request[antv1.CloneStrategyAssetRequest]) (*connect.Response[antv1.CloneStrategyAssetResponse], error) {
	uid := s.userID(ctx)
	assetID, err := uuid.Parse(req.Msg.AssetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	// Get source asset to determine current version.
	src, err := s.repo.Get(ctx, uid, assetID)
	if err != nil {
		return nil, err
	}
	clone := &repository.StrategyAssetClone{
		AssetID:          assetID,
		UserID:           uid,
		ClonedTemplateID: uuid.New(),
		SourceVersion:    src.SourceVersion,
	}
	if err := s.repo.CreateClone(ctx, clone); err != nil {
		return nil, err
	}
	return connect.NewResponse(&antv1.CloneStrategyAssetResponse{
		TemplateId: clone.ClonedTemplateID.String(),
		Clone:      cloneToProto(clone),
	}), nil
}

func (s *StrategyAssetServer) CheckAssetUpdate(ctx context.Context, req *connect.Request[antv1.CheckAssetUpdateRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.CloneId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	clone, err := s.repo.GetClone(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(cloneToProto(clone)), nil
}

func (s *StrategyAssetServer) SyncStrategyAsset(ctx context.Context, req *connect.Request[antv1.SyncStrategyAssetRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
	uid := s.userID(ctx)
	id, err := uuid.Parse(req.Msg.CloneId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	clone, err := s.repo.GetClone(ctx, uid, id)
	if err != nil {
		return nil, err
	}
	// Sync bumps the source version by 1 and marks not available.
	updated, err := s.repo.UpdateCloneSync(ctx, id, clone.SourceVersion+1, false)
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(cloneToProto(updated)), nil
}

func (s *StrategyAssetServer) ListAssetClones(ctx context.Context, req *connect.Request[antv1.ListAssetClonesRequest]) (*connect.Response[antv1.ListAssetClonesResponse], error) {
	uid := s.userID(ctx)
	assetID, err := uuid.Parse(req.Msg.AssetId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	rows, err := s.repo.ListClones(ctx, uid, assetID)
	if err != nil {
		return nil, err
	}
	items := make([]*antv1.StrategyAssetClone, len(rows))
	for i := range rows {
		items[i] = cloneToProto(&rows[i])
	}
	return connect.NewResponse(&antv1.ListAssetClonesResponse{Clones: items}), nil
}
