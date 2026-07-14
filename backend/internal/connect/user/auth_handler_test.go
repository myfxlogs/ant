package user

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	antv1 "alphaforge/gen/proto/ant/v1"
	"alphaforge/internal/interceptor"
)

func newTestAuthServer() *AuthServer {
	return NewAuthServer(nil, "test-secret-key-for-unit-test", zap.NewNop())
}

func TestLogin_EmptyLoginAndEmail(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	req := connect.NewRequest(&antv1.LoginRequest{
		Login:    "",
		Email:    "",
		Password: "pass",
	})
	_, err := srv.Login(t.Context(), req)
	if err == nil {
		t.Fatal("expected error for empty login/email")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestLogin_NilUsersRepo(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer() // users is nil
	req := connect.NewRequest(&antv1.LoginRequest{
		Login:    "testuser",
		Password: "pass",
	})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when users repo is nil")
		}
	}()
	srv.Login(t.Context(), req)
}

func TestLogout_Success(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	req := connect.NewRequest(&emptypb.Empty{})
	resp, err := srv.Logout(t.Context(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	cookies := resp.Header().Values("Set-Cookie")
	if len(cookies) == 0 {
		t.Fatal("expected Set-Cookie header")
	}
}

func TestGetMe_Unauthenticated(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	req := connect.NewRequest(&emptypb.Empty{})
	_, err := srv.GetMe(context.Background(), req) // no userID in ctx
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestGetMe_InvalidUserID(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	ctx := context.WithValue(context.Background(), interceptor.UserIDKey, "not-a-uuid")
	req := connect.NewRequest(&emptypb.Empty{})
	_, err := srv.GetMe(ctx, req)
	if err == nil {
		t.Fatal("expected error")
	}
	ce, ok := err.(*connect.Error)
	if !ok || ce.Code() != connect.CodeUnauthenticated {
		t.Fatalf("expected CodeUnauthenticated, got %v", err)
	}
}

func TestAuthServer_WithRegistration(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	if srv.registrationSvc != nil {
		t.Fatal("expected nil registrationSvc before wiring")
	}
	srv2 := srv.WithRegistration(nil)
	if srv != srv2 {
		t.Fatal("WithRegistration should return same pointer")
	}
}

func TestAuthServer_WithEmailVerification(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	if srv.emailVerifSvc != nil {
		t.Fatal("expected nil emailVerifSvc before wiring")
	}
	srv2 := srv.WithEmailVerification(nil)
	if srv != srv2 {
		t.Fatal("WithEmailVerification should return same pointer")
	}
}

func TestAuthServer_SetInsecureCookies(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	if srv.insecure {
		t.Fatal("expected insecure=false by default")
	}
	srv.SetInsecureCookies(true)
	if !srv.insecure {
		t.Fatal("expected insecure=true after SetInsecureCookies(true)")
	}
}

func TestAuthServer_SetRequireEmailVerification(t *testing.T) {
	t.Parallel()
	srv := newTestAuthServer()
	if srv.requireEmailVerif {
		t.Fatal("expected requireEmailVerif=false by default")
	}
	srv.SetRequireEmailVerification(true)
	if !srv.requireEmailVerif {
		t.Fatal("expected requireEmailVerif=true after set")
	}
}
