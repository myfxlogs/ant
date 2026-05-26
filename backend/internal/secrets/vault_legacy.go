// Package secrets legacy format detection and auto-migration (M10 ADR-0011 §2.2).
//
// Format v1 (legacy): version(1) + nonce(12) + AES-GCM(plaintext, key=KEK-derived)
//   - version byte in [1, 127], directly decryptable by aesGCMClient.
//
// Format v2 (envelope): version byte = 0x02, see envelope.go.
//
// IsLegacyFormat returns true when the ciphertext is in v1 format and should be
// migrated at next write time (lazy migration).
package secrets

// IsLegacyFormat reports whether the ciphertext uses the pre-envelope v1 format.
// Only v1 key versions (1-127) are legacy; v2 envelope (0x02) is current.
func IsLegacyFormat(ciphertext []byte) bool {
	if len(ciphertext) < 1 {
		return false
	}
	v := ciphertext[0]
	// v1 legacy: version byte 1-127 (direct KEK encryption).
	// v2 envelope: version byte 0x02 (DEK wrapping, see envelope.go).
	if v == 0x02 {
		return false
	}
	return v >= 1 && v <= 127
}

// MigrateToEnvelope re-encrypts legacy ciphertext into envelope format.
// Returns the input unchanged if already in envelope format.
func MigrateToEnvelope(client Client, purpose Purpose, ciphertext []byte) ([]byte, error) {
	if !IsLegacyFormat(ciphertext) {
		return ciphertext, nil
	}
	// Decrypt with legacy format, re-encrypt with envelope.
	plaintext, err := client.Decrypt(nil, purpose, ciphertext)
	if err != nil {
		return nil, err
	}
	// Wrap in envelope if client supports it.
	if ec, ok := client.(*EnvelopeClient); ok {
		return ec.Encrypt(nil, purpose, plaintext)
	}
	// No envelope wrapper — re-encrypt with current version.
	return client.Encrypt(nil, purpose, plaintext)
}
