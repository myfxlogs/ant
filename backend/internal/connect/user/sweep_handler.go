package user

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/hdwallet"
)

// ── Sweep (归集) RPC handlers ────────────────────────────────────────────
// These methods are on DepositServer because the proto DepositService bundles
// deposit + sweep RPCs together. The DepositServer struct is defined in
// deposit_handler.go.

func (s *DepositServer) ListPendingSignBundles(ctx context.Context, _ *connect.Request[antv1.ListPendingSignBundlesRequest]) (*connect.Response[antv1.ListPendingSignBundlesResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	summaries, err := s.sweepWorker.ListPendingSignBundlesForAdmin(ctx)
	if err != nil {
		s.log.Error("ListPendingSignBundles", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.PendingSignBundleEntry, len(summaries))
	for i, sum := range summaries {
		entry := &antv1.PendingSignBundleEntry{
			BatchId:   sum.BatchID.String(),
			BuiltAtMs: sum.BuiltAtMs,
			Status:    sum.Status,
		}
		if sum.DepositAddressID != nil {
			entry.DepositAddressId = sum.DepositAddressID.String()
		}
		items[i] = entry
	}

	return connect.NewResponse(&antv1.ListPendingSignBundlesResponse{Bundles: items}), nil
}

func (s *DepositServer) ExportUnsignedSweepBundle(ctx context.Context, req *connect.Request[antv1.ExportUnsignedSweepBundleRequest]) (*connect.Response[antv1.ExportUnsignedSweepBundleResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	addrID, err := uuid.Parse(req.Msg.DepositAddressId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid deposit_address_id: %w", err))
	}

	bundle, err := s.sweepWorker.ExportUnsignedBundle(ctx, addrID)
	if err != nil {
		s.log.Error("ExportUnsignedSweepBundle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	data, err := proto.Marshal(bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal bundle: %w", err))
	}

	return connect.NewResponse(&antv1.ExportUnsignedSweepBundleResponse{UnsignedBundle: data}), nil
}

func (s *DepositServer) ExportBatchUnsignedSweepBundle(ctx context.Context, req *connect.Request[antv1.ExportBatchUnsignedSweepBundleRequest]) (*connect.Response[antv1.ExportBatchUnsignedSweepBundleResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	if len(req.Msg.DepositAddressIds) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deposit_address_ids is required"))
	}

	addrIDs := make([]uuid.UUID, 0, len(req.Msg.DepositAddressIds))
	for _, idStr := range req.Msg.DepositAddressIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid deposit_address_id %q: %w", idStr, err))
		}
		addrIDs = append(addrIDs, id)
	}

	bundle, err := s.sweepWorker.ExportBatchUnsignedBundle(ctx, addrIDs)
	if err != nil {
		s.log.Error("ExportBatchUnsignedSweepBundle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	data, err := proto.Marshal(bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal bundle: %w", err))
	}

	return connect.NewResponse(&antv1.ExportBatchUnsignedSweepBundleResponse{UnsignedBundle: data}), nil
}

func (s *DepositServer) ImportSignedSweepBundle(ctx context.Context, req *connect.Request[antv1.ImportSignedSweepBundleRequest]) (*connect.Response[antv1.ImportSignedSweepBundleResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	signed := &antv1.SignedSweepBundle{}
	if err := proto.Unmarshal(req.Msg.SignedBundle, signed); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse signed bundle: %w", err))
	}

	if err := s.sweepWorker.ImportSignedBundle(ctx, signed); err != nil {
		s.log.Error("ImportSignedSweepBundle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.ImportSignedSweepBundleResponse{
		BatchId:           signed.BundleId,
		BroadcastComplete: true,
	}), nil
}

func (s *DepositServer) GetSweepDashboard(ctx context.Context, req *connect.Request[antv1.GetSweepDashboardRequest]) (*connect.Response[antv1.GetSweepDashboardResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	page := int(req.Msg.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	entries, total, totalUnswept, threshold, err := s.sweepWorker.GetSweepDashboard(ctx, page, pageSize)
	if err != nil {
		s.log.Error("GetSweepDashboard", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.SweepDashboardEntry, len(entries))
	for i, e := range entries {
		aboveThreshold := false
		if dec, err := decimal.NewFromString(e.UnsweptAmount); err == nil {
			if th, err := decimal.NewFromString(threshold); err == nil {
				aboveThreshold = dec.GreaterThanOrEqual(th)
			}
		}
		items[i] = &antv1.SweepDashboardEntry{
			DepositAddressId: e.AddrID.String(),
			Address:          e.Address,
			UnsweptAmount:    e.UnsweptAmount,
			AboveThreshold:   aboveThreshold,
			SweepStatus:      e.SweepStatus,
			DerivationIndex:  e.DerivationIndex,
		}
	}

	return connect.NewResponse(&antv1.GetSweepDashboardResponse{
		Addresses:    items,
		Total:        total,
		TotalUnswept: totalUnswept,
		Threshold:    threshold,
	}), nil
}

func (s *DepositServer) BuildUndelegateOnlyBundle(ctx context.Context, req *connect.Request[antv1.BuildUndelegateOnlyBundleRequest]) (*connect.Response[antv1.BuildUndelegateOnlyBundleResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}
	if s.sweepWorker == nil {
		return nil, connect.NewError(connect.CodeUnavailable, fmt.Errorf("sweep worker not initialized"))
	}

	if len(req.Msg.DepositAddressIds) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("at least one deposit_address_id required"))
	}

	addrIDs := make([]uuid.UUID, 0, len(req.Msg.DepositAddressIds))
	for _, idStr := range req.Msg.DepositAddressIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid deposit_address_id %q: %w", idStr, err))
		}
		addrIDs = append(addrIDs, id)
	}

	bundle, err := s.sweepWorker.BuildUndelegateOnlyBundle(ctx, addrIDs)
	if err != nil {
		s.log.Error("BuildUndelegateOnlyBundle", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	data, err := proto.Marshal(bundle)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("marshal bundle: %w", err))
	}

	return connect.NewResponse(&antv1.BuildUndelegateOnlyBundleResponse{
		UnsignedBundle: data,
	}), nil
}

func (s *DepositServer) ImportXpub(ctx context.Context, req *connect.Request[antv1.ImportXpubRequest]) (*connect.Response[antv1.ImportXpubResponse], error) {
	if err := s.requireAdmin(ctx); err != nil {
		return nil, err
	}

	export := &antv1.XpubExport{}
	if err := proto.Unmarshal(req.Msg.XpubExport, export); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("parse xpub export: %w", err))
	}

	if err := hdwallet.ValidateXpub(export.Xpub); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid xpub: %w", err))
	}

	fp, err := hdwallet.XpubFingerprint(export.Xpub)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("compute fingerprint: %w", err))
	}

	fingerprintVerified := false
	if s.xpubFingerprint != "" {
		if fp != s.xpubFingerprint {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("xpub fingerprint mismatch — potential key substitution (expected %s, got %s)", s.xpubFingerprint, fp))
		}
		fingerprintVerified = true
	}

	if err := s.adminRepo.SetConfig(ctx, "deposit_xpub", export.Xpub, "HD wallet account-level xpub (imported via admin UI)"); err != nil {
		s.log.Error("ImportXpub: store config", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("store xpub in config: %w", err))
	}

	if err := s.svc.UpdateXpub(export.Xpub); err != nil {
		s.log.Error("ImportXpub: hot-reload", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("hot-reload xpub: %w", err))
	}

	s.log.Info("ImportXpub: xpub imported and hot-reloaded",
		zap.String("fingerprint", fp),
		zap.Bool("fingerprint_verified", fingerprintVerified))

	return connect.NewResponse(&antv1.ImportXpubResponse{
		Xpub:                export.Xpub,
		Fingerprint:         fp,
		FingerprintVerified: fingerprintVerified,
	}), nil
}
