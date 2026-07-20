#!/usr/bin/env bash
# ADR-0026 A8: CI assertion — zero private keys in online code paths.
#
# Scans backend source (excluding cmd/coldsign, cmd/hdgen, internal/hdwallet,
# gen/, tests) for patterns that indicate private key handling in the online
# server. Any match = CI failure.
#
# Allowed locations for private key code:
#   - cmd/coldsign/  (air-gapped signing tool)
#   - cmd/hdgen/     (air-gapped key generation tool)
#   - internal/hdwallet/ (derivation library — wallet.go has mnemonic derivation for testing)
#   - gen/           (generated proto code)
#   - *_test.go      (test files)
#   - vendor/        (third-party)

set -euo pipefail

BACKEND_DIR="${1:-backend}"
FAIL=0

# Patterns that indicate private key handling in online code.
# Each pattern is a regex searched with grep -rn.
PATTERNS=(
    'PrivateKey'
    'Mnemonic'
    'bip39\.NewSeed'
    'mnemonic\.FromSeed'
    'mnemonic\.Generate'
    'Encrypt.*privkey'
    'Decrypt.*privkey'
    'encrypted_privkey'
    'PurposeDepositPrivKey'
    'PurposeHotWalletKey'
)

# Directories excluded from the scan (private key code is allowed here).
EXCLUDES=(
    '--exclude-dir=coldsign'
    '--exclude-dir=coldsign-gui'
    '--exclude-dir=hdgen'
    '--exclude-dir=hdgen-gui'
    '--exclude-dir=hdwallet'
    '--exclude-dir=gen'
    '--exclude-dir=vendor'
    '--exclude-dir=.git'
    '--exclude=*_test.go'
    '--exclude=*.pb.go'
)

echo "=== ADR-0026 A8: Zero private key assertion ==="
echo "Scanning: ${BACKEND_DIR}/internal/ ${BACKEND_DIR}/cmd/"
echo "Excluding: coldsign, coldsign-gui, hdgen, hdgen-gui, hdwallet, gen, vendor, tests, generated proto"
echo ""

for pattern in "${PATTERNS[@]}"; do
    matches=$(grep -rn "${pattern}" \
        "${EXCLUDES[@]}" \
        "${BACKEND_DIR}/internal/" "${BACKEND_DIR}/cmd/" \
        2>/dev/null || true)

    if [ -n "${matches}" ]; then
        echo "FAIL: pattern '${pattern}' found in online code:"
        echo "${matches}"
        echo ""
        FAIL=1
    fi
done

if [ ${FAIL} -eq 0 ]; then
    echo "PASS: No private key patterns found in online code paths."
    exit 0
else
    echo ""
    echo "FAIL: Private key patterns detected in online code."
    echo "Private keys must ONLY exist in cmd/coldsign, cmd/hdgen, and internal/hdwallet."
    exit 1
fi
