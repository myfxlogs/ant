# Infrastructure/Deployment Security Audit Report (W3)

## Scope

- `docker-compose.yml` — Container orchestration
- `backend/Dockerfile` — Multi-stage Go build
- `frontend/Dockerfile` — Multi-stage Node build + nginx
- `backend/docker-entrypoint.sh` — Migration runner
- `nginx/nginx.conf` — External reverse proxy
- `frontend/nginx.conf` — Frontend container nginx
- `.github/workflows/ci.yml` — CI pipeline
- `.env.example` / `.env` — Secret management
- `deploy/nats/nats.conf` — NATS configuration
- `backend/internal/server/server.go` — HTTP server config
- `backend/internal/secrets/` — Master key management

## Findings

### W3-1 — Migrations not wrapped in transactions 🟢 LOW

**File**: `backend/docker-entrypoint.sh:52-53`

**Problem**: Each `.up.sql` migration is executed via `psql -f` with `ON_ERROR_STOP=1`, which stops on error but does not wrap the entire file in a `BEGIN/COMMIT` transaction. If a multi-statement migration fails midway, some DDL may be applied while the `schema_migrations` record is not inserted, leaving the database in an inconsistent state.

**Status**: Documented as a known limitation. PostgreSQL DDL is transactional per-statement, so single-statement migrations are safe. Multi-statement migrations could partially apply. The bootstrap mode (one-time baseline catch-up) mitigates this for initial setup.

**Recommendation**: Wrap each migration file in `BEGIN; ... COMMIT;` by modifying the entrypoint to prepend `BEGIN;` and append `COMMIT;` to each file before execution, or use a proper migration tool like `golang-migrate`.

### W3-2 — NATS has no authentication 🟡 MEDIUM

**File**: `deploy/nats/nats.conf`

**Problem**: The NATS server configuration has no authentication — no `authorization` block, no `creds` file, no token. Any container on the `alphaforge-network` Docker bridge network can connect and publish/consume all subjects, including order events, profit updates, and strategy signals.

**Mitigating factors**:
- NATS port (`4222`) is bound to `127.0.0.1` only — not reachable from the internet
- Only the backend container connects to NATS
- Docker network isolation provides some protection

**Risk**: If an attacker gains code execution in any container on the Docker network (e.g., via a compromised Umami or Redis container), they can inject fake order events or eavesdrop on all NATS messages.

**Recommendation**: Add NATS token or NKey authentication. The client code already supports `CredsFile` in `nats.Config`. Add a `NATS_CREDS_FILE` env var and generate credentials.

## Verified Safe (No Issues Found)

- **Secret management**: `.env` is gitignored and not tracked. `.env.example` contains only placeholder values (`CHANGE_ME_...`). Required secrets (`DB_PASSWORD`, `JWT_SECRET`, `UMAMI_APP_SECRET`) use `${VAR:?error}` syntax to fail fast if unset.
- **No hardcoded secrets in code**: No secrets in Go source, TypeScript, or YAML files. CI uses test credentials (`ant_test`) only.
- **Docker security**:
  - Both backend and frontend Dockerfiles create non-root users (`appuser`/`nginxuser`, UID 1000)
  - Multi-stage builds: builder stage has compilers, runtime stage has only binary + runtime deps
  - No sensitive build args (no `ARG PASSWORD`, `ARG SECRET`, etc.)
  - Resource limits set on all containers (memory limits + reservations)
  - Log rotation configured (`max-size: 50m, max-file: 3`)
  - Health checks on all services
- **Port exposure**: Only frontend port (`8022`) is publicly exposed. All internal services (postgres, redis, nats, clickhouse, umami) bind to `127.0.0.1` only.
- **CI/CD pipeline**: Comprehensive — lint, vet, test with race detector, build, proto breaking change detection, proto codegen drift check, migration down-file check, frontend lint + build. Coverage gate at 12%.
- **Migration safety**: `ON_ERROR_STOP=1` prevents silent failures. Bootstrap mode (one-time) tolerates already-applied DDL. Strict mode (after baseline) fails on any migration error. Critical table validation post-migration.
- **Cookie security**: `HttpOnly; SameSite=Strict; Secure` (Secure defaults to true, disabled only for dev).
- **Master key management**: `ANT_MASTER_KEY` from env, supports file-based (`ANT_MASTER_KEY_FILE`), key rotation supported, AES-256-GCM.
- **HTTP server**: `ReadTimeout: 15s`, `ReadHeaderTimeout: 10s` (added in W2-2), graceful shutdown with 10s timeout.
- **Security headers** (external nginx): `X-Content-Type-Options`, `X-Frame-Options: DENY`, `X-XSS-Protection`, `Referrer-Policy`, `Permissions-Policy`, `Strict-Transport-Security` (added in W2-3), `Content-Security-Policy` (added in W2-3).
- **Rate limiting**: External nginx has `limit_req zone=api_limit burst=20` on API endpoints and `limit_conn` (50 connections per IP).
- **No WebSocket endpoints**: ConnectRPC + SSE only (compliant with project rules).

## Reuse Preflight

- **W3-1**: NEW: Transaction wrapping recommendation (not implemented — documented as known limitation)
- **W3-2**: NEW: NATS authentication (not implemented — documented as recommendation)

## Migrations

No migrations required.

## Deployment

- `go build ./...` ✅ (server.go change from W2-2)
- nginx config changes (W2-3) require `docker compose restart nginx` or rebuild
