//go:build integration

package systemai

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"alphaforge/internal/pkg/secretbox"
	"alphaforge/internal/repository"
)

func mustSeal(t *testing.T, box *secretbox.Box, plaintext string) []byte {
	t.Helper()
	ct, salt, nonce, err := box.Seal([]byte(plaintext))
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	out := make([]byte, 2+len(salt)+2+len(nonce)+len(ct))
	out[0] = byte(len(salt) >> 8)
	out[1] = byte(len(salt))
	copy(out[2:], salt)
	off := 2 + len(salt)
	out[off] = byte(len(nonce) >> 8)
	out[off+1] = byte(len(nonce))
	copy(out[off+2:], nonce)
	copy(out[off+2+len(nonce):], ct)
	return out
}

// publicHTTPTestServer starts an httptest server on a non-loopback interface —
// ValidateBaseURL (correctly) rejects loopback hosts, so BYOK resolution
// needs a public-shaped base_url.
func publicHTTPTestServer(t *testing.T, h http.Handler) *httptest.Server {
	t.Helper()
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		t.Skipf("no interfaces: %v", err)
	}
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok || ipnet.IP.IsLoopback() || ipnet.IP.To4() == nil {
			continue
		}
		l, err := net.Listen("tcp", ipnet.IP.String()+":0")
		if err != nil {
			continue
		}
		srv := httptest.NewUnstartedServer(h)
		srv.Listener = l
		srv.Start()
		t.Cleanup(srv.Close)
		return srv
	}
	t.Skip("no non-loopback IPv4 address available")
	return nil
}

// FIX-2026-09-08-BYOK-QUOTA 对抗证明。
//
// 业主实测：自有 Key（BYOK）调用被平台每日配额拦截
// （"daily token quota exceeded (206382/200000)"）——平台配额的目的是
// 保护平台成本（网关调用平台垫钱），BYOK 调用由用户直付厂商，平台零成本，
// 不应被平台配额管制。
//
// 修复：walletChecker 预检查移到 provider 解析之后，仅对系统付费
// （Gateway）候选生效；BYOK 候选跳过配额/钱包门禁。
//
// mutation: 还原 chat.go/chat_stream.go/chat_failover.go → T-BYOK RED
// （BYOK 调用仍被 quota error 拦截）。

func byokTestPG(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "postgres://ant:ant@localhost:5433/ant?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Skipf("skipping: pg: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Skipf("skipping: pg ping: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newBYOKService(t *testing.T, pool *pgxpool.Pool, chatSrvURL string, walletErr string) (*Service, uuid.UUID) {
	t.Helper()
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))
	repo := repository.NewSystemAIConfigRepository(pool)
	svc := NewService(repo, box)
	svc.SetLogger(zap.NewNop())

	uid := uuid.New()
	_, err := pool.Exec(context.Background(),
		`INSERT INTO users (id, email, password_hash, role, status, created_at, updated_at)
		 VALUES ($1, $2, 'x', 'user', 'active', NOW(), NOW()) ON CONFLICT (id) DO NOTHING`,
		uid, "byok-quota-"+uid.String()+"@test.local")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, uid)
	})

	if chatSrvURL != "" {
		pid := "openai_compatible_byoktest"
		if err := repo.Upsert(context.Background(), &repository.SystemAIConfigRow{
			UserID: uid, ProviderID: pid, Name: "BYOK Test", BaseURL: chatSrvURL, DefaultModel: "test-model", Enabled: true,
		}, uid.String()); err != nil {
			t.Fatalf("upsert config: %v", err)
		}
		ct, salt, nonce, err := box.Seal([]byte("sk-byok-test"))
		if err != nil {
			t.Fatalf("seal: %v", err)
		}
		if err := repo.SetSecret(context.Background(), uid, pid, &repository.SystemAISecret{
			Ciphertext: ct, Salt: salt, Nonce: nonce,
		}, uid.String()); err != nil {
			t.Fatalf("set secret: %v", err)
		}
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), `DELETE FROM system_ai_configs WHERE user_id=$1`, uid)
		})
	}

	if walletErr != "" {
		svc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) (int, error) {
			return -1, errQuotaStub
		})
		_ = walletErr
	}
	return svc, uid
}

var errQuotaStub = errStub{}

type errStub struct{}

func (errStub) Error() string { return "platform daily AI quota exceeded (stub)" }

func TestBYOKCallSkipsPlatformQuota(t *testing.T) {
	pool := byokTestPG(t)

	srv := publicHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"byok-ok"}}]}`))
	}))

	svc, uid := newBYOKService(t, pool, srv.URL, "quota")
	checkerCalled := false
	svc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) (int, error) {
		checkerCalled = true
		return -1, errQuotaStub
	})

	res, err := svc.ChatCompletion(context.Background(), uid, []ChatMessage{{Role: "user", Content: "hi"}})
	if err != nil {
		t.Fatalf("BYOK call must skip platform quota gate: %v", err)
	}
	if res != "byok-ok" {
		t.Fatalf("content = %q", res)
	}
	if checkerCalled {
		t.Fatal("wallet/quota checker must not run for BYOK calls")
	}
}

func TestGatewayCallEnforcesPlatformQuota(t *testing.T) {
	pool := byokTestPG(t)
	box := secretbox.New([]byte("0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"))

	srv := publicHTTPTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"gw-ok"}}]}`))
	}))

	// Gateway provider + model rows (platform-paid path).
	gwRepo := repository.NewSystemAIProviderRepository(pool)
	modelRepo := repository.NewAIModelRepository(pool)
	pid := uuid.New()
	gwID := "gwtest_" + uuid.NewString()[:8]
	if err := gwRepo.Create(context.Background(), &repository.SystemAIProvider{
		ID: pid, ProviderID: gwID, Name: "GW Test", BaseURL: srv.URL,
		APIKeyEncrypted: mustSeal(t, box, "sk-gw"), Enabled: true,
	}); err != nil {
		t.Fatalf("create gw provider: %v", err)
	}
	t.Cleanup(func() { _ = gwRepo.Delete(context.Background(), pid) })
	if _, err := modelRepo.Upsert(context.Background(), &repository.AIModel{
		ProviderID: pid, ModelName: "gw-model", Enabled: true,
		PricePer1MInput: "0", PricePer1MOutput: "0",
	}); err != nil {
		t.Fatalf("upsert gw model: %v", err)
	}

	svc, uid := newBYOKService(t, pool, "", "quota")
	svc.SetGatewayProviderRepo(gwRepo)
	checkerCalled := false
	svc.SetWalletChecker(func(ctx context.Context, userID uuid.UUID) (int, error) {
		checkerCalled = true
		return -1, errQuotaStub
	})

	_, err := svc.ChatCompletion(context.Background(), uid, []ChatMessage{{Role: "user", Content: "hi"}})
	if err == nil {
		t.Fatal("system-paid call must be gated by the platform quota checker")
	}
	if !checkerCalled {
		t.Fatal("wallet/quota checker must run for gateway calls")
	}
}
