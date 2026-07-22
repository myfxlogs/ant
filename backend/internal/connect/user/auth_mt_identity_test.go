package user

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/mdgateway/adapter/mdtick"
)

// mockMTTesterForAuth implements MTConnectionTester for unit tests.
type mockMTTesterForAuth struct {
	verifyErr error
}

func (m *mockMTTesterForAuth) Test(ctx context.Context, platform, brokerHost, login, password string) (*mdtick.MTAccountInfo, error) {
	return nil, nil
}

func (m *mockMTTesterForAuth) VerifyPassword(ctx context.Context, platform, brokerHost, login, password, brokerCompany, accountID string) error {
	return m.verifyErr
}

func TestVerifyMTIdentity_MissingFields(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	req := connect.NewRequest(&antv1.VerifyMTIdentityRequest{
		Email:      "",
		MtLogin:    "12345",
		MtPassword: "pass",
	})
	_, err := srv.VerifyMTIdentity(t.Context(), req)
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeInvalidArgument {
		t.Fatalf("expected CodeInvalidArgument, got %v", err)
	}
}

func TestVerifyMTIdentity_NotConfigured(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer() // pg and mtTester are nil
	req := connect.NewRequest(&antv1.VerifyMTIdentityRequest{
		Email:      "test@example.com",
		MtLogin:    "12345",
		MtPassword: "pass",
	})
	_, err := srv.VerifyMTIdentity(t.Context(), req)
	if err == nil {
		t.Fatal("expected error when MT identity verification not configured")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable, got %v", err)
	}
}

func TestVerifyMTIdentity_AllFieldsProvided(t *testing.T) {
	t.Parallel()
	// With the optimal design, email + mt_login + mt_password are all required.
	// broker_host and platform are looked up from DB by the backend.
	srv := NewAuthServer(nil, "secret", zap.NewNop())
	srv.WithMTIdentityVerification(nil, &mockMTTesterForAuth{})
	req := connect.NewRequest(&antv1.VerifyMTIdentityRequest{
		Email:      "test@example.com",
		MtLogin:    "12345",
		MtPassword: "pass",
	})
	_, err := srv.VerifyMTIdentity(t.Context(), req)
	if err == nil {
		t.Fatal("expected error when pg is nil (not configured)")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnavailable {
		t.Fatalf("expected CodeUnavailable, got %v", err)
	}
}

func TestAuthServer_WithMTIdentityVerification(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	if srv.pg != nil || srv.mtTester != nil {
		t.Fatal("expected nil pg and mtTester before wiring")
	}
	srv2 := srv.WithMTIdentityVerification(nil, nil)
	if srv != srv2 {
		t.Fatal("WithMTIdentityVerification should return same pointer")
	}
}
