package admin

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"

	antv1 "alphaforge/gen/proto/ant/v1"
	antv1c "alphaforge/gen/proto/ant/v1/antv1connect"
	"github.com/google/uuid"
	"alphaforge/internal/interceptor"
	"alphaforge/internal/repository"
)

type AdminJurisdictionServer struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

var _ antv1c.AdminJurisdictionServiceHandler = (*AdminJurisdictionServer)(nil)

func NewAdminJurisdictionServer(repo *repository.AdminRepository, log *zap.Logger) *AdminJurisdictionServer {
	return &AdminJurisdictionServer{repo: repo, log: log}
}

func (s *AdminJurisdictionServer) GetJurisdictionStatus(ctx context.Context, req *connect.Request[antv1.GetJurisdictionStatusRequest]) (*connect.Response[antv1.GetJurisdictionStatusResponse], error) {
	st, err := s.repo.GetJurisdictionStatus(ctx, req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if st.CountryCode != "" {
		sanctioned, _ := s.repo.IsSanctioned(ctx, st.CountryCode)
		st.IsSanctioned = sanctioned
	}
	resp := &antv1.GetJurisdictionStatusResponse{
		Status: repoJurisdictionToProto(st),
	}
	return connect.NewResponse(resp), nil
}

func (s *AdminJurisdictionServer) SetKYCStatus(ctx context.Context, req *connect.Request[antv1.SetKYCStatusRequest]) (*connect.Response[antv1.SetKYCStatusResponse], error) {
	if !validKYCStatus(req.Msg.KycStatus) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid kyc status: %s", req.Msg.KycStatus))
	}
	verifiedBy := interceptor.GetUserID(ctx)
	if err := s.repo.SetKYCStatus(ctx, req.Msg.UserId, req.Msg.KycStatus, verifiedBy); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.SetKYCStatusResponse{}), nil
}

func (s *AdminJurisdictionServer) ListSanctionedCountries(ctx context.Context, _ *connect.Request[antv1.ListSanctionedCountriesRequest]) (*connect.Response[antv1.ListSanctionedCountriesResponse], error) {
	countries, err := s.repo.ListSanctionedCountries(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	pb := make([]*antv1.SanctionedCountry, 0, len(countries))
	for _, c := range countries {
		pb = append(pb, &antv1.SanctionedCountry{
			CountryCode: c.CountryCode,
			Label:       c.Label,
			AddedBy:     c.AddedBy,
			AddedAt:     timestamppb.New(c.AddedAt),
		})
	}
	return connect.NewResponse(&antv1.ListSanctionedCountriesResponse{Countries: pb}), nil
}

func (s *AdminJurisdictionServer) AddSanctionedCountry(ctx context.Context, req *connect.Request[antv1.AddSanctionedCountryRequest]) (*connect.Response[antv1.AddSanctionedCountryResponse], error) {
	addedBy := interceptor.GetUserID(ctx)
	if err := s.repo.AddSanctionedCountry(ctx, req.Msg.CountryCode, req.Msg.Label, addedBy); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.AddSanctionedCountryResponse{}), nil
}

func (s *AdminJurisdictionServer) RemoveSanctionedCountry(ctx context.Context, req *connect.Request[antv1.RemoveSanctionedCountryRequest]) (*connect.Response[antv1.RemoveSanctionedCountryResponse], error) {
	if err := s.repo.RemoveSanctionedCountry(ctx, req.Msg.CountryCode); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&antv1.RemoveSanctionedCountryResponse{}), nil
}

func (s *AdminJurisdictionServer) ListUsersByKYCStatus(ctx context.Context, req *connect.Request[antv1.ListUsersByKYCStatusRequest]) (*connect.Response[antv1.ListUsersByKYCStatusResponse], error) {
	users, total, err := s.repo.ListUsersByKYCStatus(ctx, req.Msg.KycStatus, int(req.Msg.Page), int(req.Msg.PageSize))
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// Load sanctioned countries to compute IsSanctioned per user.
	sanctioned := make(map[string]bool)
	if countries, err := s.repo.ListSanctionedCountries(ctx); err == nil {
		for _, c := range countries {
			sanctioned[c.CountryCode] = true
		}
	}
	pb := make([]*antv1.UserKYCItem, 0, len(users))
	for _, u := range users {
		pb = append(pb, &antv1.UserKYCItem{
			UserId:               u.UserID,
			Email:                u.Email,
			KycStatus:            u.KYCStatus,
			CountryCode:          u.CountryCode,
			IsSanctioned:         sanctioned[u.CountryCode],
			DisclaimerAccepted:   u.DisclaimerAccepted,
			QuestionnaireCompleted: u.QuestionnaireDone,
			RiskScore:            int32(u.RiskScore),
			SanctionedOverride:   u.SanctionedOverride,
		})
	}
	return connect.NewResponse(&antv1.ListUsersByKYCStatusResponse{Users: pb, Total: total}), nil
}

func (s *AdminJurisdictionServer) SetSanctionedOverride(ctx context.Context, req *connect.Request[antv1.SetSanctionedOverrideRequest]) (*connect.Response[antv1.SetSanctionedOverrideResponse], error) {
	if _, err := uuid.Parse(req.Msg.UserId); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid user id: %w", err))
	}
	if err := s.repo.SetSanctionedOverride(ctx, req.Msg.UserId, req.Msg.Override); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	s.log.Info("admin: sanctioned override",
		zap.String("actor", interceptor.GetUserID(ctx)),
		zap.String("target", req.Msg.UserId),
		zap.Bool("override", req.Msg.Override))
	return connect.NewResponse(&antv1.SetSanctionedOverrideResponse{}), nil
}

func repoJurisdictionToProto(st *repository.JurisdictionStatus) *antv1.JurisdictionStatus {
	p := &antv1.JurisdictionStatus{
		UserId:               st.UserID,
		KycStatus:            st.KYCStatus,
		CountryCode:          st.CountryCode,
		IsSanctioned:         st.IsSanctioned,
		DisclaimerAccepted:   st.DisclaimerAccepted,
		QuestionnaireCompleted: st.QuestionnaireDone,
		RiskScore:            int32(st.RiskScore),
		SanctionedOverride:   st.SanctionedOverride,
	}
	if st.KYCVerifiedAt != nil {
		p.KycVerifiedAt = timestamppb.New(*st.KYCVerifiedAt)
	}
	if st.DisclaimerAcceptedAt != nil {
		p.DisclaimerAcceptedAt = timestamppb.New(*st.DisclaimerAcceptedAt)
	}
	if st.QuestionnaireDoneAt != nil {
		p.QuestionnaireCompletedAt = timestamppb.New(*st.QuestionnaireDoneAt)
	}
	return p
}

// validKYCStatus returns true for recognized KYC status values.
func validKYCStatus(status string) bool {
	switch status {
	case "verified", "unverified", "rejected", "pending":
		return true
	}
	return false
}
