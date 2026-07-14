package user

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
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

func depositToProto(d *model.DepositRequest) *antv1.DepositRequest {
	p := &antv1.DepositRequest{
		Id:        d.ID.String(),
		UserId:    d.UserID.String(),
		Amount:    d.Amount,
		AmountUsd: d.AmountUSD,
		Status:    d.Status,
		CreatedAt: timestamppb.New(d.CreatedAt),
		UpdatedAt: timestamppb.New(d.UpdatedAt),
	}
	if d.TxHash != nil {
		p.TxHash = *d.TxHash
	}
	if d.ReviewerID != nil {
		p.ReviewerId = d.ReviewerID.String()
	}
	if d.ReviewNote != nil {
		p.ReviewNote = *d.ReviewNote
	}
	if d.ReviewedAt != nil {
		p.ReviewedAt = timestamppb.New(*d.ReviewedAt)
	}
	if d.WalletTxID != nil {
		p.WalletTxId = d.WalletTxID.String()
	}
	if d.UserEmail != nil {
		p.UserEmail = *d.UserEmail
	}
	return p
}

func (s *DepositServer) CreateDeposit(ctx context.Context, req *connect.Request[antv1.CreateDepositRequest]) (*connect.Response[antv1.CreateDepositResponse], error) {
	uid, err := parseUserID(ctx)
	if err != nil {
		return nil, err
	}
	if req.Msg.Amount == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("amount is required"))
	}
	dep, err := s.svc.CreateDeposit(ctx, uid, req.Msg.Amount, req.Msg.TxHash)
	if err != nil {
		s.log.Error("CreateDeposit", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.CreateDepositResponse{Deposit: depositToProto(dep)}), nil
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
	items := make([]*antv1.DepositRequest, len(deps))
	for i, d := range deps {
		items[i] = depositToProto(&d)
	}
	return connect.NewResponse(&antv1.ListMyDepositsResponse{Deposits: items, Total: total}), nil
}

func (s *DepositServer) GetDepositInfo(ctx context.Context, _ *connect.Request[antv1.GetDepositInfoRequest]) (*connect.Response[antv1.GetDepositInfoResponse], error) {
	addr, network, rate, err := s.svc.GetDepositInfo(ctx)
	if err != nil {
		s.log.Error("GetDepositInfo", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.GetDepositInfoResponse{
		ReceivingAddress: addr,
		Network:          network,
		ExchangeRate:     rate,
	}), nil
}

func (s *DepositServer) ListDeposits(ctx context.Context, req *connect.Request[antv1.ListDepositsRequest]) (*connect.Response[antv1.ListDepositsResponse], error) {
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
	deps, total, err := s.svc.ListDeposits(ctx, page, pageSize, req.Msg.Status)
	if err != nil {
		s.log.Error("ListDeposits", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.DepositRequest, len(deps))
	for i, d := range deps {
		items[i] = depositToProto(&d)
	}
	return connect.NewResponse(&antv1.ListDepositsResponse{Deposits: items, Total: total}), nil
}

func (s *DepositServer) ApproveDeposit(ctx context.Context, req *connect.Request[antv1.ApproveDepositRequest]) (*connect.Response[antv1.ApproveDepositResponse], error) {
	reviewerID, err := s.requireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	depositID, err := uuid.Parse(req.Msg.DepositId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid deposit_id"))
	}
	dep, err := s.svc.ApproveDeposit(ctx, depositID, reviewerID, req.Msg.ReviewNote)
	if err != nil {
		if errors.Is(err, repository.ErrDepositAlreadyProcessed) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deposit already processed"))
		}
		if errors.Is(err, repository.ErrDepositNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deposit not found"))
		}
		s.log.Error("ApproveDeposit", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.ApproveDepositResponse{Deposit: depositToProto(dep)}), nil
}

func (s *DepositServer) RejectDeposit(ctx context.Context, req *connect.Request[antv1.RejectDepositRequest]) (*connect.Response[antv1.RejectDepositResponse], error) {
	reviewerID, err := s.requireAdmin(ctx)
	if err != nil {
		return nil, err
	}
	depositID, err := uuid.Parse(req.Msg.DepositId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid deposit_id"))
	}
	dep, err := s.svc.RejectDeposit(ctx, depositID, reviewerID, req.Msg.ReviewNote)
	if err != nil {
		if errors.Is(err, repository.ErrDepositAlreadyProcessed) {
			return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("deposit already processed"))
		}
		if errors.Is(err, repository.ErrDepositNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deposit not found"))
		}
		s.log.Error("RejectDeposit", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RejectDepositResponse{Deposit: depositToProto(dep)}), nil
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
