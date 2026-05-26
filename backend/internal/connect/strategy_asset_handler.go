package connect

import (
	"context"
	"time"

	"go.uber.org/zap"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "anttrader/gen/proto/ant/v1"
	antv1c "anttrader/gen/proto/ant/v1/antv1connect"
)

// StrategyAssetServer implements ant.v1.StrategyAssetServiceHandler.
type StrategyAssetServer struct{ log *zap.Logger }

var _ antv1c.StrategyAssetServiceHandler = (*StrategyAssetServer)(nil)

func NewStrategyAssetServer(log *zap.Logger) *StrategyAssetServer {
	return &StrategyAssetServer{log: log}
}

func mockAsset(id, name string) *antv1.StrategyAsset {
	now := timestamppb.New(time.Now())
	return &antv1.StrategyAsset{
		Id:           id,
		OwnerUserId:  "mock-user",
		Name:         name,
		Description:  "量化交易策略",
		Visibility:   "private",
		ReviewStatus: "approved",
		CloneCount:   5,
		LatestVersion: 1,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

func (s *StrategyAssetServer) ListStrategyAssets(ctx context.Context, req *connect.Request[antv1.ListStrategyAssetsRequest]) (*connect.Response[antv1.ListStrategyAssetsResponse], error) {
	return connect.NewResponse(&antv1.ListStrategyAssetsResponse{
		Assets: []*antv1.StrategyAsset{
			mockAsset("ast-001", "均线交叉策略"),
			mockAsset("ast-002", "动量突破策略"),
			mockAsset("ast-003", "均值回归策略"),
		},
	}), nil
}

func (s *StrategyAssetServer) GetStrategyAsset(ctx context.Context, req *connect.Request[antv1.GetStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	return connect.NewResponse(mockAsset(req.Msg.AssetId, "策略资产")), nil
}

func (s *StrategyAssetServer) SubmitAssetReview(ctx context.Context, req *connect.Request[antv1.SubmitAssetReviewRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	asset := mockAsset(req.Msg.SourceTemplateId, req.Msg.Name)
	asset.ReviewStatus = "pending_review"
	return connect.NewResponse(asset), nil
}

func (s *StrategyAssetServer) ReviewStrategyAsset(ctx context.Context, req *connect.Request[antv1.ReviewStrategyAssetRequest]) (*connect.Response[antv1.StrategyAsset], error) {
	asset := mockAsset(req.Msg.AssetId, "已审核策略")
	asset.ReviewStatus = "approved"
	return connect.NewResponse(asset), nil
}

func (s *StrategyAssetServer) CloneStrategyAsset(ctx context.Context, req *connect.Request[antv1.CloneStrategyAssetRequest]) (*connect.Response[antv1.CloneStrategyAssetResponse], error) {
	clone := &antv1.StrategyAssetClone{
		Id:               uuid.New().String(),
		AssetId:          req.Msg.AssetId,
		UserId:           "mock-user",
		ClonedTemplateId: uuid.New().String(),
		SourceVersion:    1,
		SyncAvailable:    true,
		CreatedAt:        timestamppb.New(time.Now()),
	}
	return connect.NewResponse(&antv1.CloneStrategyAssetResponse{
		TemplateId: clone.ClonedTemplateId,
		Clone:      clone,
	}), nil
}

func (s *StrategyAssetServer) CheckAssetUpdate(ctx context.Context, req *connect.Request[antv1.CheckAssetUpdateRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
	return connect.NewResponse(&antv1.StrategyAssetClone{
		Id:            req.Msg.CloneId,
		SyncAvailable: false,
	}), nil
}

func (s *StrategyAssetServer) SyncStrategyAsset(ctx context.Context, req *connect.Request[antv1.SyncStrategyAssetRequest]) (*connect.Response[antv1.StrategyAssetClone], error) {
	return connect.NewResponse(&antv1.StrategyAssetClone{
		Id:            req.Msg.CloneId,
		SyncAvailable: false,
		SourceVersion: 2,
	}), nil
}

func (s *StrategyAssetServer) ListAssetClones(ctx context.Context, req *connect.Request[antv1.ListAssetClonesRequest]) (*connect.Response[antv1.ListAssetClonesResponse], error) {
	return connect.NewResponse(&antv1.ListAssetClonesResponse{
		Clones: []*antv1.StrategyAssetClone{},
	}), nil
}
