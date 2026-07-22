package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/hdwallet"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/model"
	"alphaforge/internal/repository"
	"alphaforge/internal/service"
	"alphaforge/internal/sweep"
)

// DepositServer implements ant.v1.DepositServiceHandler.
type DepositServer struct {
	svc             *service.DepositService
	platformSvc     *service.PlatformService
	sweepWorker     *sweep.Worker
	adminRepo       *repository.AdminRepository
	xpubFingerprint string // from env DEPOSIT_XPUB_FINGERPRINT, integrity anchor
	log             *zap.Logger
}

var _ antv1c.DepositServiceHandler = (*DepositServer)(nil)

func NewDepositServer(svc *service.DepositService, platformSvc *service.PlatformService, sweepWorker *sweep.Worker, adminRepo *repository.AdminRepository, xpubFingerprint string, log *zap.Logger) *DepositServer {
	return &DepositServer{svc: svc, platformSvc: platformSvc, sweepWorker: sweepWorker, adminRepo: adminRepo, xpubFingerprint: xpubFingerprint, log: log}
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
	addr, network, err := s.svc.GetOrDeriveAddress(ctx, uid)
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

func (s *DepositServer) ListDepositAddresses(ctx context.Context, req *connect.Request[antv1.ListDepositAddressesRequest]) (*connect.Response[antv1.ListDepositAddressesResponse], error) {
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
	addrs, total, available, err := s.svc.ListDepositAddresses(ctx, req.Msg.Status, page, pageSize)
	if err != nil {
		s.log.Error("ListDepositAddresses", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	items := make([]*antv1.DepositAddress, len(addrs))
	for i, a := range addrs {
		items[i] = depositAddressToProto(&a)
	}
	return connect.NewResponse(&antv1.ListDepositAddressesResponse{
		Addresses:      items,
		Total:          total,
		AvailableCount: int32(available),
	}), nil
}

func depositAddressToProto(a *model.DepositAddress) *antv1.DepositAddress {
	out := &antv1.DepositAddress{
		Id:              a.ID.String(),
		Address:         a.Address,
		DerivationIndex: int32(a.DerivationIndex),
		Network:         a.Network,
		Status:          a.Status,
		HasReceivedUsdt: a.HasReceivedUSDT,
		CreatedAt:       timestamppb.New(a.CreatedAt),
	}
	if a.UserID != nil {
		out.UserId = a.UserID.String()
	}
	if a.AssignedAt != nil {
		out.AssignedAt = timestamppb.New(*a.AssignedAt)
	}
	return out
}

func (s *DepositServer) ImportDepositAddresses(ctx context.Context, req *connect.Request[antv1.ImportDepositAddressesRequest]) (*connect.Response[antv1.ImportDepositAddressesResponse], error) {
	if _, err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	batch := &antv1.AddressBatch{}
	if err := proto.Unmarshal(req.Msg.BatchData, batch); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse batch: %w", err))
	}

	imported := 0
	skipped := 0
	xpubKey := s.svc.XpubKey()
	for _, entry := range batch.Entries {
		if entry.Address == "" || entry.DerivationIndex < 0 {
			skipped++
			continue
		}

		var expected string
		var err error
		if xpubKey != nil {
			expected, err = hdwallet.DeriveAddressFromExtKey(xpubKey, uint32(entry.DerivationIndex))
		} else {
			expected, err = hdwallet.DeriveAddressFromXpub(s.svc.Xpub(), uint32(entry.DerivationIndex))
		}
		if err != nil {
			s.log.Warn("import: derive failed", zap.Int32("index", entry.DerivationIndex), zap.Error(err))
			skipped++
			continue
		}

		if entry.Address != expected {
			s.log.Warn("import: address mismatch",
				zap.Int32("index", entry.DerivationIndex),
				zap.String("imported", entry.Address),
				zap.String("expected", expected))
			skipped++
			continue
		}

		imported++
	}

	s.log.Info("import deposit addresses cross-check",
		zap.Int("imported", imported),
		zap.Int("skipped", skipped))

	return connect.NewResponse(&antv1.ImportDepositAddressesResponse{
		Imported: int32(imported),
		Skipped:  int32(skipped),
	}), nil
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

// Sweep (归集) RPC handlers are in sweep_handler.go.
