package user

import (
	"context"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/service"
)

// WebAuthnServer implements ant.v1.WebAuthnServiceHandler.
type WebAuthnServer struct {
	svc         *service.WebAuthnService
	platformSvc *service.PlatformService
	log         *zap.Logger
}

var _ antv1c.WebAuthnServiceHandler = (*WebAuthnServer)(nil)

func NewWebAuthnServer(svc *service.WebAuthnService, platformSvc *service.PlatformService, log *zap.Logger) *WebAuthnServer {
	return &WebAuthnServer{svc: svc, platformSvc: platformSvc, log: log}
}

func (s *WebAuthnServer) BeginRegistration(ctx context.Context, req *connect.Request[antv1.BeginRegistrationRequest]) (*connect.Response[antv1.BeginRegistrationResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	options, err := s.svc.BeginRegistration(ctx, userID, req.Msg.Name)
	if err != nil {
		s.log.Error("BeginRegistration", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.BeginRegistrationResponse{
		CredentialCreationOptions: options,
	}), nil
}

func (s *WebAuthnServer) FinishRegistration(ctx context.Context, req *connect.Request[antv1.FinishRegistrationRequest]) (*connect.Response[antv1.FinishRegistrationResponse], error) {
	// The response bytes contain "sessionID|responseJSON" (set by BeginRegistration).
	parts := strings.SplitN(string(req.Msg.CredentialResponse), "|", 2)
	if len(parts) != 2 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid response format"))
	}
	sessionID := parts[0]
	responseBytes := []byte(parts[1])

	cred, err := s.svc.FinishRegistration(ctx, sessionID, responseBytes)
	if err != nil {
		s.log.Error("FinishRegistration", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.FinishRegistrationResponse{
		CredentialId: cred.CredentialID,
		Name:         cred.Name,
	}), nil
}

func (s *WebAuthnServer) ListCredentials(ctx context.Context, _ *connect.Request[antv1.ListCredentialsRequest]) (*connect.Response[antv1.ListCredentialsResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	creds, err := s.svc.ListCredentials(ctx, userID)
	if err != nil {
		s.log.Error("ListCredentials", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.CredentialEntry, len(creds))
	for i, c := range creds {
		items[i] = &antv1.CredentialEntry{
			CredentialId:  c.CredentialID,
			Name:          c.Name,
			CreatedAtTsMs: c.CreatedAt.UnixMilli(),
			SignCount:     int64(c.SignCount),
		}
	}

	return connect.NewResponse(&antv1.ListCredentialsResponse{Credentials: items}), nil
}

func (s *WebAuthnServer) RemoveCredential(ctx context.Context, req *connect.Request[antv1.RemoveCredentialRequest]) (*connect.Response[antv1.RemoveCredentialResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := s.svc.RemoveCredential(ctx, userID, req.Msg.CredentialId); err != nil {
		s.log.Error("RemoveCredential", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.RemoveCredentialResponse{}), nil
}

func (s *WebAuthnServer) BeginWithdrawal(ctx context.Context, req *connect.Request[antv1.BeginWithdrawalRequest]) (*connect.Response[antv1.BeginWithdrawalResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	challenge, nonce, withdrawalID, err := s.svc.BeginWithdrawal(ctx, userID, req.Msg.Amount, req.Msg.DestAddress)
	if err != nil {
		if err == service.ErrInsufficientWithdrawalBalance {
			return nil, connect.NewError(connect.CodeFailedPrecondition, err)
		}
		s.log.Error("BeginWithdrawal", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.BeginWithdrawalResponse{
		Challenge:    challenge,
		Nonce:        nonce,
		WithdrawalId: withdrawalID,
	}), nil
}

func (s *WebAuthnServer) FinishWithdrawal(ctx context.Context, req *connect.Request[antv1.FinishWithdrawalRequest]) (*connect.Response[antv1.FinishWithdrawalResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	wr, err := s.svc.FinishWithdrawal(ctx, userID, req.Msg.WithdrawalId, req.Msg.Assertion, req.Msg.CredentialId)
	if err != nil {
		if err == service.ErrWithdrawalNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if err == service.ErrWithdrawalNotOwner {
			return nil, connect.NewError(connect.CodePermissionDenied, err)
		}
		s.log.Error("FinishWithdrawal", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.FinishWithdrawalResponse{
		WithdrawalId: wr.ID.String(),
		Status:       wr.Status,
	}), nil
}

func (s *WebAuthnServer) ListWithdrawals(ctx context.Context, req *connect.Request[antv1.ListWithdrawalsRequest]) (*connect.Response[antv1.ListWithdrawalsResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	page := int(req.Msg.Page)
	if page < 1 {
		page = 1
	}
	pageSize := int(req.Msg.PageSize)
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	withdrawals, total, err := s.svc.ListWithdrawals(ctx, userID, page, pageSize)
	if err != nil {
		s.log.Error("ListWithdrawals", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.WithdrawalEntry, len(withdrawals))
	for i, w := range withdrawals {
		entry := &antv1.WithdrawalEntry{
			Id:             w.ID.String(),
			Amount:         w.Amount,
			DestAddress:    w.DestAddress,
			Nonce:          uint64(w.Nonce),
			Status:         w.Status,
			CreatedAtTsMs:  w.CreatedAt.UnixMilli(),
		}
		if w.TxHash != nil {
			entry.TxHash = *w.TxHash
		}
		if w.CompletedAt != nil {
			entry.CompletedAtTsMs = w.CompletedAt.UnixMilli()
		}
		items[i] = entry
	}

	return connect.NewResponse(&antv1.ListWithdrawalsResponse{
		Withdrawals: items,
		Total:       total,
	}), nil
}

func (s *WebAuthnServer) CancelWithdrawal(ctx context.Context, req *connect.Request[antv1.CancelWithdrawalRequest]) (*connect.Response[antv1.CancelWithdrawalResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := s.svc.CancelWithdrawal(ctx, userID, req.Msg.WithdrawalId); err != nil {
		if err == service.ErrWithdrawalNotFound {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		s.log.Error("CancelWithdrawal", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.CancelWithdrawalResponse{Status: "CANCELLED"}), nil
}

func (s *WebAuthnServer) AddWhitelistAddress(ctx context.Context, req *connect.Request[antv1.AddWhitelistAddressRequest]) (*connect.Response[antv1.AddWhitelistAddressResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	var userEmail string
	if s.platformSvc != nil {
		userEmail, _ = s.platformSvc.GetUserEmail(ctx, userID.String())
	}

	if err := s.svc.AddWhitelistAddress(ctx, userID, userEmail, req.Msg.Address, req.Msg.Label); err != nil {
		s.log.Error("AddWhitelistAddress", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.AddWhitelistAddressResponse{Status: "PENDING_CONFIRMATION"}), nil
}

func (s *WebAuthnServer) ListWhitelistAddresses(ctx context.Context, _ *connect.Request[antv1.ListWhitelistAddressesRequest]) (*connect.Response[antv1.ListWhitelistAddressesResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	entries, err := s.svc.ListWhitelistAddresses(ctx, userID)
	if err != nil {
		s.log.Error("ListWhitelistAddresses", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.WhitelistEntry, len(entries))
	for i, e := range entries {
		entry := &antv1.WhitelistEntry{
			Id:      e.ID.String(),
			Address: e.Address,
			Label:   e.Label,
			Status:  e.Status,
		}
		if e.ConfirmedAt != nil {
			entry.ConfirmedAtTsMs = e.ConfirmedAt.UnixMilli()
		}
		items[i] = entry
	}

	return connect.NewResponse(&antv1.ListWhitelistAddressesResponse{Addresses: items}), nil
}

func (s *WebAuthnServer) RemoveWhitelistAddress(ctx context.Context, req *connect.Request[antv1.RemoveWhitelistAddressRequest]) (*connect.Response[antv1.RemoveWhitelistAddressResponse], error) {
	userID, err := parseUserID(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	entryID, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid id"))
	}

	if err := s.svc.RemoveWhitelistAddress(ctx, userID, entryID); err != nil {
		s.log.Error("RemoveWhitelistAddress", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.RemoveWhitelistAddressResponse{Status: "REMOVED"}), nil
}

func (s *WebAuthnServer) ExportCredentialList(ctx context.Context, _ *connect.Request[antv1.ExportCredentialListRequest]) (*connect.Response[antv1.ExportCredentialListResponse], error) {
	actorStr := interceptor.GetUserID(ctx)
	actorID, err := uuid.Parse(actorStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid actor"))
	}
	if s.platformSvc == nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("platform service not available"))
	}
	isAdmin, err := s.platformSvc.IsAdmin(ctx, actorID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("check admin: %w", err))
	}
	if !isAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}

	creds, err := s.svc.ExportAllCredentials(ctx)
	if err != nil {
		s.log.Error("ExportCredentialList", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	items := make([]*antv1.UserCredentialEntry, len(creds))
	for i, c := range creds {
		items[i] = &antv1.UserCredentialEntry{
			UserId:       c.UserID.String(),
			CredentialId: c.CredentialID,
			PublicKey:    c.PublicKey,
			SignCount:    int64(c.SignCount),
		}
	}

	return connect.NewResponse(&antv1.ExportCredentialListResponse{
		Credentials:  items,
		ExportedAtMs: time.Now().UnixMilli(),
	}), nil
}

func (s *WebAuthnServer) ExportWhitelist(ctx context.Context, _ *connect.Request[antv1.ExportWhitelistRequest]) (*connect.Response[antv1.ExportWhitelistResponse], error) {
	actorStr := interceptor.GetUserID(ctx)
	actorID, err := uuid.Parse(actorStr)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("invalid user id"))
	}
	isAdmin, err := s.platformSvc.IsAdmin(ctx, actorID)
	if err != nil {
		s.log.Error("ExportWhitelist: admin check", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !isAdmin {
		return nil, connect.NewError(connect.CodePermissionDenied, fmt.Errorf("admin required"))
	}

	data, err := s.svc.ExportWhitelist(ctx)
	if err != nil {
		s.log.Error("ExportWhitelist", zap.Error(err))
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&antv1.ExportWhitelistResponse{
		WhitelistProto: data,
		ExportedAtMs:   time.Now().UnixMilli(),
	}), nil
}
