// Package secrets provides AES-256-GCM envelope encryption with
// versioned KEK derivation. v2 implementation runs in-process
// (no external Vault service); interface is shaped for future
// migration to HashiCorp Vault transit or AWS KMS.
package secrets

import "context"

// Client 是 ant 唯一的加解密接口。所有需要密钥的模块通过此接口。
type Client interface {
	// Encrypt 用当前 latest 版本的 KEK 加密 plaintext，返回自包含密文
	// （版本号 + nonce + 密文 + tag）。
	// purpose 是用途子密钥（HKDF info），常量见下。
	Encrypt(ctx context.Context, purpose Purpose, plaintext []byte) (ciphertext []byte, err error)

	// Decrypt 自动从 ciphertext 头部提取版本号，用对应 KEK 解密。
	// 版本未知 → ErrUnknownKeyVersion。
	Decrypt(ctx context.Context, purpose Purpose, ciphertext []byte) (plaintext []byte, err error)

	// Reencrypt 解密后用 latest 版本重新加密。轮换 worker 用。
	Reencrypt(ctx context.Context, purpose Purpose, ciphertext []byte) ([]byte, error)

	// CurrentVersion 返回当前 latest 版本号（>= 1）。
	CurrentVersion() uint8
}

// Purpose 区分子密钥用途（HKDF info）。新增 purpose 必须 ADR。
type Purpose string

const (
	PurposeMTPassword   Purpose = "mt-password"   // mt_accounts.password_encrypted
	PurposeMTAPIToken   Purpose = "mtapi-token"   // mt_accounts.mtapi_token_encrypted
	PurposeBrokerCookie Purpose = "broker-cookie" // 预留：第三方登录态
)

// RotateClient extends Client with key rotation support. L-3.
// Not all Client implementations support rotation (e.g. KMS-backed clients
// may delegate rotation to the KMS API).
type RotateClient interface {
	Client
	// RotateKey generates a new master key version.
	// Returns the new version number and base64-encoded key material.
	// Old keys are retained for decryption of legacy ciphertexts.
	RotateKey() (newVersion uint8, newKeyB64 string, err error)
}

// ErrUnknownKeyVersion is returned when the ciphertext version byte
// references a KEK version that is not available.
var ErrUnknownKeyVersion = &SecretError{Msg: "unknown key version"}

// SecretError is a sentinel error type for secrets package errors.
type SecretError struct {
	Msg string
}

func (e *SecretError) Error() string { return "secrets: " + e.Msg }
