package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/model"
)

// BeginRegistration starts the WebAuthn registration ceremony.
// Returns the credential creation options (JSON) for the browser to pass to navigator.credentials.create().
func (s *WebAuthnService) BeginRegistration(ctx context.Context, userID uuid.UUID, name string) ([]byte, error) {
	creds, err := s.credRepo.ListCredentialsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: begin registration: list credentials: %w", err)
	}

	existing := make([]webauthn.Credential, len(creds))
	for i, c := range creds {
		existing[i] = webauthn.Credential{
			ID:              []byte(c.CredentialID),
			PublicKey:       c.PublicKey,
			AttestationType: c.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    []byte(c.AAGUID),
				SignCount: uint32(c.SignCount),
			},
		}
	}

	user := &webauthnUser{id: userID, credentials: existing}

	options, sessionData, err := s.webauthn.BeginRegistration(user)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: begin registration: %w", err)
	}

	sessionID := uuid.New().String()
	s.sessions.Store(sessionID, &sessionEntry{
		data:      sessionData,
		userID:    userID,
		name:      name,
		expiresAt: time.Now().Add(60 * time.Second),
	})

	optionsBytes, err := json.Marshal(options.Response)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: marshal options: %w", err)
	}

	sessionHeader := base64.StdEncoding.EncodeToString([]byte(sessionID))
	result := append([]byte(sessionHeader+"|"), optionsBytes...)
	return result, nil
}

// FinishRegistration completes the WebAuthn registration ceremony.
func (s *WebAuthnService) FinishRegistration(ctx context.Context, sessionID string, responseBytes []byte) (*model.WebAuthnCredential, error) {
	val, ok := s.sessions.LoadAndDelete(sessionID)
	if !ok {
		return nil, fmt.Errorf("webauthn service: finish registration: session not found or expired")
	}
	entry := val.(*sessionEntry)

	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(responseBytes))
	if err != nil {
		return nil, fmt.Errorf("webauthn service: parse registration response: %w", err)
	}

	user := &webauthnUser{id: entry.userID}

	credential, err := s.webauthn.CreateCredential(user, *entry.data, parsedResponse)
	if err != nil {
		return nil, fmt.Errorf("webauthn service: create credential: %w", err)
	}

	credID := base64.RawURLEncoding.EncodeToString(credential.ID)
	transports := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	cred := &model.WebAuthnCredential{
		UserID:          entry.userID,
		CredentialID:    credID,
		PublicKey:       credential.PublicKey,
		AttestationType: credential.AttestationType,
		AAGUID:          string(credential.Authenticator.AAGUID),
		SignCount:       int64(credential.Authenticator.SignCount),
		Transports:      transports,
		Name:            entry.name,
	}

	if err := s.credRepo.CreateCredential(ctx, cred); err != nil {
		return nil, fmt.Errorf("webauthn service: store credential: %w", err)
	}

	idemKey := "cred-add-" + credID
	if err := s.withdrawRepo.CreateCredentialChangeLog(ctx, &model.CredentialChangeLog{
		UserID:     entry.userID,
		ChangeType: "CREDENTIAL_ADD",
		TargetID:   credID,
		IdemKey:    idemKey,
	}); err != nil {
		return nil, fmt.Errorf("webauthn service: credential change log: %w", err)
	}

	if err := s.walletRepo.WriteCredentialChangeLedger(ctx, entry.userID, "CREDENTIAL_CHANGE",
		"CREDENTIAL_ADD:"+credID, idemKey); err != nil {
		return nil, fmt.Errorf("webauthn service: credential hash chain: %w", err)
	}

	s.log.Info("webauthn credential registered",
		zap.String("user_id", entry.userID.String()),
		zap.String("credential_id", credID),
		zap.String("name", entry.name))

	return cred, nil
}

// ListCredentials returns all passkeys for a user.
func (s *WebAuthnService) ListCredentials(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error) {
	return s.credRepo.ListCredentialsByUser(ctx, userID)
}

// RemoveCredential removes a passkey and logs the change (R12).
func (s *WebAuthnService) RemoveCredential(ctx context.Context, userID uuid.UUID, credentialID string) error {
	if err := s.credRepo.DeleteCredential(ctx, userID, credentialID); err != nil {
		return fmt.Errorf("webauthn service: remove credential: %w", err)
	}

	idemKey := "cred-remove-" + credentialID
	if err := s.withdrawRepo.CreateCredentialChangeLog(ctx, &model.CredentialChangeLog{
		UserID:     userID,
		ChangeType: "CREDENTIAL_REMOVE",
		TargetID:   credentialID,
		IdemKey:    idemKey,
	}); err != nil {
		return fmt.Errorf("webauthn service: remove credential: change log: %w", err)
	}

	if err := s.walletRepo.WriteCredentialChangeLedger(ctx, userID, "CREDENTIAL_CHANGE",
		"CREDENTIAL_REMOVE:"+credentialID, idemKey); err != nil {
		return fmt.Errorf("webauthn service: remove credential: hash chain: %w", err)
	}

	s.log.Info("webauthn credential removed",
		zap.String("user_id", userID.String()),
		zap.String("credential_id", credentialID))

	return nil
}

// ExportAllCredentials returns all credentials for coldsign USB sync (Q2).
func (s *WebAuthnService) ExportAllCredentials(ctx context.Context) ([]model.WebAuthnCredential, error) {
	return s.credRepo.ListAllCredentials(ctx)
}
