package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mustReadFile reads a file relative to the backend root for source-text assertions.
// service test wd = .../backend/internal/service → backendRoot = wd/../..
func mustReadFile(t *testing.T, relPath string) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	backendRoot := filepath.Join(wd, "..", "..")
	data, err := os.ReadFile(filepath.Join(backendRoot, relPath))
	if err != nil {
		t.Fatalf("read %s: %v", relPath, err)
	}
	return string(data)
}

// T4: TestUpdateAccountInfoTx_WritesAccountType guards S3b (UpdateAccountInfoTx SQL
// contains account_type = $). RED mutation: delete `account_type = $11` from SQL → RED.
func TestUpdateAccountInfoTx_WritesAccountType(t *testing.T) {
	content := mustReadFile(t, "internal/service/account_lifecycle.go")
	idx := strings.Index(content, "func (s *AccountService) UpdateAccountInfoTx")
	if idx < 0 {
		t.Fatal("UpdateAccountInfoTx not found")
	}
	// Bound the search to just this function (up to the next top-level func).
	body := content[idx:]
	if next := strings.Index(body[1:], "\nfunc "); next >= 0 {
		body = body[:next+1]
	}
	if !strings.Contains(body, "account_type = $") {
		t.Fatal("UpdateAccountInfoTx must write account_type (S3b)")
	}
}

// T4b: TestUpdateAccountType_MethodExists guards S3d (UpdateAccountType method exists
// and writes account_type). RED mutation: delete the method → RED.
func TestUpdateAccountType_MethodExists(t *testing.T) {
	content := mustReadFile(t, "internal/service/account_lifecycle.go")
	idx := strings.Index(content, "func (s *AccountService) UpdateAccountType")
	if idx < 0 {
		t.Fatal("UpdateAccountType method not found (S3d)")
	}
	body := content[idx:]
	if !strings.Contains(body, "account_type = $2") {
		t.Fatal("UpdateAccountType must write account_type = $2 (S3d)")
	}
}

// T4c: TestAccountInfoUpdate_HasAccountTypeField guards S3a.
// RED mutation: remove AccountType from struct → compile error.
func TestAccountInfoUpdate_HasAccountTypeField(t *testing.T) {
	u := AccountInfoUpdate{AccountType: "real"}
	if u.AccountType != "real" {
		t.Fatal("AccountInfoUpdate must have AccountType field (S3a)")
	}
}
