package jwt

import (
	"testing"
	"time"
)

func init() {
	Init("test-secret-key-for-jwt-testing-12345", 15*time.Minute, 7*24*time.Hour, "anttrader-test")
}

func TestGenerateTokenPair_Success(t *testing.T) {
	// Not parallel — uses global jwtConfig modified by init().
	pair, err := GenerateTokenPair("user-1", "test@example.com", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pair.AccessToken == "" {
		t.Fatal("expected non-empty access token")
	}
	if pair.RefreshToken == "" {
		t.Fatal("expected non-empty refresh token")
	}
	if pair.ExpiresIn <= 0 {
		t.Fatalf("expected positive expires_in, got %d", pair.ExpiresIn)
	}
}

func TestParseToken_ValidAccessToken(t *testing.T) {
	// Not parallel — uses global jwtConfig.
	pair, _ := GenerateTokenPair("user-2", "u2@test.com", "admin")
	claims, err := ParseToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "user-2" {
		t.Errorf("expected user-2, got %s", claims.UserID)
	}
	if claims.Email != "u2@test.com" {
		t.Errorf("expected u2@test.com, got %s", claims.Email)
	}
	if claims.Role != "admin" {
		t.Errorf("expected admin role, got %s", claims.Role)
	}
}

func TestParseToken_Expired(t *testing.T) {
	// Can't run in parallel — modifies global config.
	Init("expired-test-secret", 1*time.Millisecond, 1*time.Millisecond, "test")
	pair, err := GenerateTokenPair("u-e", "e@t.com", "user")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	time.Sleep(10 * time.Millisecond) // let token expire
	_, err = ParseToken(pair.AccessToken)
	if err != ErrTokenExpired {
		t.Fatalf("expected ErrTokenExpired, got %v", err)
	}
	// Restore
	Init("test-secret-key-for-jwt-testing-12345", 15*time.Minute, 7*24*time.Hour, "anttrader-test")
}

func TestParseToken_Invalid(t *testing.T) {
	// Not parallel — uses global jwtConfig.
	_, err := ParseToken("not-a-valid-jwt-token")
	if err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestParseToken_WrongSignature(t *testing.T) {
	// Not parallel — uses global jwtConfig.
	Init("key-a", 1*time.Hour, 24*time.Hour, "a")
	pair, _ := GenerateTokenPair("u-3", "u3@t.com", "user")
	// Switch secret and try to parse.
	Init("key-b", 1*time.Hour, 24*time.Hour, "b")
	_, err := ParseToken(pair.AccessToken)
	if err != ErrTokenInvalid {
		t.Fatalf("expected ErrTokenInvalid for wrong signature, got %v", err)
	}
	Init("test-secret-key-for-jwt-testing-12345", 15*time.Minute, 7*24*time.Hour, "anttrader-test")
}

func TestValidateToken_SynonymForParse(t *testing.T) {
	// Not parallel — uses global jwtConfig.
	pair, _ := GenerateTokenPair("u-4", "u4@t.com", "user")
	claims, err := ValidateToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "u-4" {
		t.Errorf("unexpected user id: %s", claims.UserID)
	}
}
