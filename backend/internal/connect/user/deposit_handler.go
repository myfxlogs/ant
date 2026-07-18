package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/service"
)

// DepositServer implements ant.v1.DepositServiceHandler.
type DepositServer struct {
	svc         *service.DepositService
	platformSvc *service.PlatformService
	log         *zap.Logger
}

var _ antv1c.DepositServiceHandler = (*DepositServer)(nil)

func NewDepositServer(svc *service.DepositService, platformSvc *service.PlatformService, log *zap.Logger) *DepositServer {
	return &DepositServer{svc: svc, platformSvc: platformSvc, log: log}
}

func depositToProto(d *model.Deposit) *antv1.Deposit {
	out := &antv1.Deposit{
		Id:               d.ID.String(),
		UserId:           d.UserID.String(),
		DepositAddressId: d.DepositAddressID.String(),
		TxHash:           d.TxHash,
		Amount:           d.Amount,
		BlockNumber:      d.BlockNumber,
		Confirmations:    int32(d.Confirmations),
		Status:           d.Status,
		CreatedAt:        timestamppb.New(d.CreatedAt),
	}
	if d.ConfirmedAt != nil {
		out.ConfirmedAt = timestamppb.New(*d.ConfirmedAt)
	}
	return out
}

func (s *DepositServer) GetDepositAddress(ctx context.Context, _ *connect.Request[antv1.GetDepositAddressRequest]) (*connect.Response[antv1.GetDepositAddressResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	addr, network, err := s.svc.GetOrClaimAddress(ctx, uid)
	if err != nil {
		s.log.Error("GetDepositAddress", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetDepositAddressResponse{
		Address: addr,
		Network: network,
	}), nil
}

func (s *DepositServer) ListMyDeposits(ctx context.Context, req *connect.Request[antv1.ListMyDepositsRequest]) (*connect.Response[antv1.ListMyDepositsResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	page := int(req.Msg.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	deps, total, err := s.svc.ListMyDeposits(ctx, uid, page, pageSize)
	if err != nil {
		s.log.Error("ListMyDeposits", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.Deposit, len(deps))
	for i, d := range deps {
		items[i] = depositToProto(&d)
	}
	return connect.NewResponse(&antv1.ListMyDepositsResponse{Deposits: items, Total: total}), nil
}

func (s *DepositServer) ListManualReviewDeposits(ctx context.Context, req *connect.Request[antv1.ListManualReviewDepositsRequest]) (*connect.Response[antv1.ListManualReviewDepositsResponse], error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	page := int(req.Msg.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	deps, total, err := s.svc.ListManualReviewDeposits(ctx, page, pageSize)
	if err != nil {
		s.log.Error("ListManualReviewDeposits", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.Deposit, len(deps))
	for i, d := range deps {
		items[i] = depositToProto(&d)
	}
	return connect.NewResponse(&antv1.ListManualReviewDepositsResponse{Deposits: items, Total: total}), nil
}

// requireAdmin extracts the current user ID and verifies admin status.
func (s *DepositServer) requireAdmin(ctx context.Context) (uuid.UUID, error) {
	actorStr := interceptor.GetUserID(ctx)
	actorID, err := uuid.Parse(actorStr)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid actor"))
	}
	if s.platformSvc == nil {
		return actorID, nil
	}
	isAdmin, err := s.platformSvc.IsAdmin(ctx, actorID)
	if err != nil {
		return uuid.Nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check admin: %w", err))
	}
	if !isAdmin {
		return uuid.Nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}
	return actorID, nil
}
