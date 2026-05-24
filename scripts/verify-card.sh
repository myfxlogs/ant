#!/usr/bin/env bash
# verify-card.sh — run a card's verify command, producing a rich, unique log.
# Usage: bash scripts/verify-card.sh M10.5-X "<verify command>"
# Output: docs/handover/verify-M10.5-X.log (overwrite)
set -uo pipefail

CARD="${1:?usage: verify-card.sh CARD CMD}"
CMD="${2:?usage: verify-card.sh CARD CMD}"
LOG="docs/handover/verify-${CARD}.log"

{
    echo "================================================================"
    echo "  verify-${CARD}  ($(date -u +%FT%TZ))"
    echo "================================================================"
    echo "Card:        ${CARD}"
    echo "Repo:        $(git rev-parse --show-toplevel)"
    echo "Branch:      $(git rev-parse --abbrev-ref HEAD)"
    echo "HEAD:        $(git rev-parse HEAD)"
    echo "HEAD (full): $(git log -1 --pretty='format:%H %s')"
    echo "HEAD parent: $(git log -1 --pretty='format:%P')"
    echo "Author:      $(git log -1 --pretty='format:%an <%ae>')"
    echo "AuthorDate:  $(git log -1 --pretty='format:%ai')"
    echo "Tree hash:   $(git log -1 --pretty='format:%T')"
    echo "Host:        $(hostname)"
    echo "Uname:       $(uname -srm)"
    echo "Go version:  $(go version 2>/dev/null || echo 'go not in PATH')"
    echo "PWD:         $(pwd)"
    echo "Cmd:         ${CMD}"
    echo "----------------------------------------------------------------"
    echo
    bash -c "${CMD}" 2>&1
    rc=$?
    echo
    echo "----------------------------------------------------------------"
    echo "Exit code:   ${rc}"
    echo "Log file:    ${LOG}"
    echo "Finished:    $(date -u +%FT%TZ)"
    exit ${rc}
} > "${LOG}" 2>&1

# Re-print last lines so caller sees outcome
tail -8 "${LOG}"
echo "[verify-card.sh] log → ${LOG} ($(wc -l < ${LOG}) lines)"
