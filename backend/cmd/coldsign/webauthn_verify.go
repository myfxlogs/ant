package main

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/shopspring/decimal"
	"google.golang.org/protobuf/proto"

	antv1 "alphaforge/gen/proto/ant/v1"
)

// CredentialDB is coldsign's self-held public key database (Q2/R12).
// It is populated via USB import of ExportCredentialList proto from the online server.
// The file format is serialized ColdsignCredentialDB proto binary.
type CredentialDB struct {
	Credentials map[string]CredentialRecord // key: credential_id (base64url)
	filePath    string
	mu          sync.Mutex
}

type CredentialRecord struct {
	UserID    string
	PublicKey []byte
	SignCount int64
}

// LoadCredentialDB reads the coldsign self-held credential database from disk.
// The file is a serialized ColdsignCredentialDB proto message.
func LoadCredentialDB(path string) (*CredentialDB, error) {
	if path == "" {
		return nil, fmt.Errorf("credential database path not provided — use -cred-db flag")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credential database: %w", err)
	}
	var pb antv1.ColdsignCredentialDB
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("parse credential database (proto): %w", err)
	}
	db := &CredentialDB{Credentials: make(map[string]CredentialRecord), filePath: path}
	for _, e := range pb.Credentials {
		db.Credentials[e.CredentialId] = CredentialRecord{
			UserID:    e.UserId,
			PublicKey: e.PublicKey,
			SignCount: e.SignCount,
		}
	}
	return db, nil
}

// UpdateSignCount updates the sign count for a credential and persists to disk.
func (db *CredentialDB) UpdateSignCount(credentialID string, newCount int64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	rec, ok := db.Credentials[credentialID]
	if !ok {
		return fmt.Errorf("credential %s not found", credentialID)
	}
	if newCount <= rec.SignCount {
		return nil // authenticator didn't update count (common for platform authenticators)
	}
	rec.SignCount = newCount
	db.Credentials[credentialID] = rec
	return db.saveLocked()
}

// saveLocked serializes the credential DB back to disk (caller must hold db.mu).
func (db *CredentialDB) saveLocked() error {
	pb := &antv1.ColdsignCredentialDB{}
	for credID, rec := range db.Credentials {
		pb.Credentials = append(pb.Credentials, &antv1.ColdsignCredentialEntry{
			UserId:       rec.UserID,
			CredentialId: credID,
			PublicKey:    rec.PublicKey,
			SignCount:    rec.SignCount,
		})
	}
	data, err := proto.Marshal(pb)
	if err != nil {
		return fmt.Errorf("marshal credential DB: %w", err)
	}
	if err := os.WriteFile(db.filePath, data, 0600); err != nil {
		return fmt.Errorf("write credential DB: %w", err)
	}
	return nil
}

// verifyWithdrawalAuth verifies the WebAuthn assertion for a withdrawal transfer (R11).
// coldsign uses its self-held public key — it does NOT trust the online server's public key.
//
// Steps:
// 1. Extract WithdrawalAuth from TransferTx
// 2. Look up credential_id in self-held DB → get public key
// 3. Reconstruct challenge = sha256(amount|dest|nonce|user_id)
// 4. Verify WebAuthn assertion against the public key
// 5. Check dest ∈ user whitelist (if whitelist file provided)
// 6. Check amount ≤ per-withdrawal limit (if limit configured)
func verifyWithdrawalAuth(tx *antv1.UnsignedTx, credDB *CredentialDB, whitelistPath string, maxWithdrawal string) error {
	transferTx := tx.GetTransfer()
	auth := transferTx.GetAuth()
	if auth == nil {
		return fmt.Errorf("R11: withdrawal transfer has no auth — refusing to sign")
	}

	// 1. Look up credential in self-held DB.
	record, ok := credDB.Credentials[auth.CredentialId]
	if !ok {
		return fmt.Errorf("R11: credential_id %s not found in coldsign self-held DB — refusing to sign", auth.CredentialId)
	}

	// 2. Reconstruct challenge = sha256(amount|dest|nonce|user_id).
	challenge := buildChallenge(tx.Amount, tx.ToAddress, fmt.Sprintf("%d", auth.Nonce), auth.UserId)

	// 3. Parse the assertion response.
	parsedAssertion, err := protocol.ParseCredentialRequestResponseBytes(auth.Assertion)
	if err != nil {
		return fmt.Errorf("R11: parse assertion response: %w", err)
	}

	// 4. Verify the assertion against the self-held public key.
	// Build a webauthn.Credential for verification.
	credIDBytes, err := base64.RawURLEncoding.DecodeString(auth.CredentialId)
	if err != nil {
		return fmt.Errorf("R11: decode credential_id: %w", err)
	}

	credential := webauthn.Credential{
		ID:        credIDBytes,
		PublicKey: record.PublicKey,
		Authenticator: webauthn.Authenticator{
			SignCount: uint32(record.SignCount),
		},
	}

	// Create a webauthn user for verification.
	user := &coldsignUser{
		id:          []byte(auth.UserId),
		credentials: []webauthn.Credential{credential},
	}

	// Build session data for verification.
	sessionData := &webauthn.SessionData{
		Challenge:        base64.RawURLEncoding.EncodeToString(challenge),
		UserID:           []byte(auth.UserId),
		UserVerification: protocol.VerificationRequired,
	}

	// Use cached WebAuthn instance (RP ID/origin match registration).
	w, err := initWebAuthn()
	if err != nil {
		return err
	}

	updatedCred, err := w.ValidateLogin(user, *sessionData, parsedAssertion)
	if err != nil {
		return fmt.Errorf("R11: WebAuthn assertion verification FAILED — refusing to sign: %w", err)
	}

	// Persist updated sign count to prevent replay attacks (R11 first principle).
	// If this fails, refuse to sign — a stale sign count could allow replay.
	if err := credDB.UpdateSignCount(auth.CredentialId, int64(updatedCred.Authenticator.SignCount)); err != nil {
		return fmt.Errorf("R11: persist updated sign count: %w", err)
	}

	// 5. Check whitelist if provided.
	if whitelistPath != "" {
		if err := checkWhitelist(whitelistPath, auth.UserId, tx.ToAddress); err != nil {
			return fmt.Errorf("R11: whitelist check: %w", err)
		}
	}

	// 6. Check withdrawal limit if configured.
	if maxWithdrawal != "" {
		if err := checkWithdrawalLimit(tx.Amount, maxWithdrawal); err != nil {
			return fmt.Errorf("R11: limit check: %w", err)
		}
	}

	fmt.Fprintf(os.Stderr, "R11: WebAuthn assertion verified for user %s, credential %s\n",
		auth.UserId, auth.CredentialId)
	fmt.Fprintf(os.Stderr, "    amount=%s dest=%s nonce=%d\n", tx.Amount, tx.ToAddress, auth.Nonce)

	return nil
}

// buildChallenge reconstructs sha256(amount|dest|nonce|user_id).
// Must match the online server's buildWithdrawalChallenge exactly.
func buildChallenge(amount, dest, nonce, userID string) []byte {
	h := sha256.New()
	h.Write([]byte(amount))
	h.Write([]byte("|"))
	h.Write([]byte(dest))
	h.Write([]byte("|"))
	h.Write([]byte(nonce))
	h.Write([]byte("|"))
	h.Write([]byte(userID))
	return h.Sum(nil)
}

// cachedWhitelist is loaded once from disk and reused for all verifications.
var cachedWhitelist *antv1.ColdsignWhitelist

// loadWhitelist loads and caches the whitelist proto from disk.
// On subsequent calls the cached copy is returned.
func loadWhitelist(path string) (*antv1.ColdsignWhitelist, error) {
	if cachedWhitelist != nil {
		return cachedWhitelist, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read whitelist: %w", err)
	}
	var pb antv1.ColdsignWhitelist
	if err := proto.Unmarshal(data, &pb); err != nil {
		return nil, fmt.Errorf("parse whitelist (proto): %w", err)
	}
	cachedWhitelist = &pb
	return &pb, nil
}

// checkWhitelist verifies the destination address is in the user's whitelist.
// The whitelist file is a serialized ColdsignWhitelist proto message, loaded once and cached.
func checkWhitelist(path, userID, destAddress string) error {
	wl, err := loadWhitelist(path)
	if err != nil {
		return err
	}
	entry, ok := wl.Users[userID]
	if !ok {
		return fmt.Errorf("user %s has no whitelist entries", userID)
	}
	for _, addr := range entry.Addresses {
		if addr == destAddress {
			return nil
		}
	}
	return fmt.Errorf("destination %s not in whitelist for user %s", destAddress, userID)
}

// checkWithdrawalLimit verifies the amount does not exceed the per-withdrawal limit.
func checkWithdrawalLimit(amount, maxAmount string) error {
	amt, err := decimal.NewFromString(amount)
	if err != nil {
		return fmt.Errorf("invalid amount: %s", amount)
	}
	max, err := decimal.NewFromString(maxAmount)
	if err != nil {
		return fmt.Errorf("invalid max withdrawal: %s", maxAmount)
	}
	if amt.GreaterThan(max) {
		return fmt.Errorf("withdrawal amount %s exceeds max %s", amount, maxAmount)
	}
	return nil
}

// coldsignUser implements webauthn.User for verification on the cold signing machine.
type coldsignUser struct {
	id          []byte
	credentials []webauthn.Credential
}

func (u *coldsignUser) WebAuthnID() []byte                  { return u.id }
func (u *coldsignUser) WebAuthnName() string                { return string(u.id) }
func (u *coldsignUser) WebAuthnDisplayName() string         { return string(u.id) }
func (u *coldsignUser) WebAuthnCredentials() []webauthn.Credential { return u.credentials }
func (u *coldsignUser) WebAuthnIcon() string                { return "" }

// rpID and rpOrigin are set from command-line flags in main().
var rpID string
var rpOrigin string

// cachedWebAuthn is initialized once on first use to avoid repeated allocation.
var cachedWebAuthn *webauthn.WebAuthn

// initWebAuthn creates the WebAuthn instance once and caches it.
func initWebAuthn() (*webauthn.WebAuthn, error) {
	if cachedWebAuthn != nil {
		return cachedWebAuthn, nil
	}
	wconfig := &webauthn.Config{
		RPID:     rpID,
		RPOrigins: []string{rpOrigin},
	}
	w, err := webauthn.New(wconfig)
	if err != nil {
		return nil, fmt.Errorf("R11: init webauthn for verification: %w", err)
	}
	cachedWebAuthn = w
	return w, nil
}
