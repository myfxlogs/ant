package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// WebAuthnService orchestrates passkey registration and withdrawal authorization.
// The online server is an untrusted relay — real verification happens on coldsign (R11).
type WebAuthnService struct {
	webauthn     *webauthn.WebAuthn
	credRepo     *repository.WebAuthnRepository
	withdrawRepo *repository.WithdrawalRepository
	walletSvc    *WalletService
	walletRepo   *repository.WalletRepository
	log          *zap.Logger
	emailNotifier Notifier

	// sessionStore holds in-progress registration sessions keyed by session ID.
	sessions sync.Map
}

type sessionEntry struct {
	data      *webauthn.SessionData
	userID    uuid.UUID
	name      string
	expiresAt time.Time
}

type webauthnUser struct {
	id          uuid.UUID
	credentials []webauthn.Credential
}

func (u *webauthnUser) WebAuthnID() []byte                          { return u.id[:] }
func (u *webauthnUser) WebAuthnName() string                        { return u.id.String() }
func (u *webauthnUser) WebAuthnDisplayName() string                 { return u.id.String() }
func (u *webauthnUser) WebAuthnCredentials() []webauthn.Credential  { return u.credentials }
func (u *webauthnUser) WebAuthnIcon() string                        { return "" }

// NewWebAuthnService creates a new WebAuthnService.
func NewWebAuthnService(
	rpID, rpOrigin string,
	credRepo *repository.WebAuthnRepository,
	withdrawRepo *repository.WithdrawalRepository,
	walletSvc *WalletService,
	walletRepo *repository.WalletRepository,
	emailNotifier Notifier,
	log *zap.Logger,
) (*WebAuthnService, error) {
	wconfig := &webauthn.Config{
		RPDisplayName: "AlphaForge",
		RPID:          rpID,
		RPOrigins:     []string{rpOrigin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.Platform,
			UserVerification:        protocol.VerificationRequired,
			ResidentKey:             protocol.ResidentKeyRequirementRequired,
		},
		AttestationPreference: protocol.PreferNoAttestation,
	}

	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: new: %w", err)
	}

	svc := &WebAuthnService{
		webauthn:      w,
		credRepo:      credRepo,
		withdrawRepo:  withdrawRepo,
		walletSvc:     walletSvc,
		walletRepo:    walletRepo,
		emailNotifier: emailNotifier,
		log:           log,
	}

	go svc.cleanupExpiredSessions()

	return svc, nil
}

func (s *WebAuthnService) cleanupExpiredSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		s.sessions.Range(func(key, val any) bool {
			entry := val.(*sessionEntry)
			if now.After(entry.expiresAt) {
				s.sessions.Delete(key)
			}
			return true
		})
	}
}

// buildWithdrawalChallenge computes sha256(amount|dest|nonce|user_id).
// coldsign reconstructs this from the WithdrawalAuth fields to verify the assertion.
func buildWithdrawalChallenge(amount, dest string, nonce int64, userID string) []byte {
	h := sha256.New()
	h.Write([]byte(amount))
	h.Write([]byte("|"))
	h.Write([]byte(dest))
	h.Write([]byte("|"))
	h.Write([]byte(fmt.Sprintf("%d", nonce)))
	h.Write([]byte("|"))
	h.Write([]byte(userID))
	return h.Sum(nil)
}

// Errors.
var (
	ErrInsufficientWithdrawalBalance = errors.New("insufficient balance for withdrawal")
	ErrWithdrawalNotFound            = errors.New("withdrawal not found")
	ErrWithdrawalNotOwner            = errors.New("withdrawal does not belong to user")
)
