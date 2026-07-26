# AI Agent Gateway Security Audit Report (W4)

## Scope

- `backend/internal/agent/gateway.go` — Agent gateway ConnectRPC handler
- `backend/internal/agent/generator.go` — Strategy generation orchestrator
- `backend/internal/agent/prompt_renderer.go` — Prompt construction with injection protection
- `backend/internal/ai/strategy_prompt.go` — System prompt builder
- `backend/internal/service/systemai/service.go` — Per-user AI provider configuration
- `backend/internal/service/systemai/chat.go` — Chat completion request builder
- `backend/internal/service/systemai/discovery.go` — Model discovery + URL validation
- `backend/internal/service/systemai/chat_failover.go` — Failover with circuit breaker
- `backend/internal/repository/ai_gateway_repository.go` — API key encryption
- `backend/internal/repository/system_ai_config_repository.go` — Secret storage
- `backend/tools/mql2go/vm.go` / `vm_execute.go` — VM execution sandbox

## Findings

### W4-1 — No SSRF protection on user-configured AI provider base_url 🟡 MEDIUM

**File**: `backend/internal/service/systemai/discovery.go:38-49`

**Problem**: `validateBaseURL` only checks that the URL has an `http`/`https` scheme and non-empty host. It does not block internal/private IP ranges (e.g., `127.0.0.1`, `169.254.169.254`, `10.x.x.x`, `172.16.x.x`, Docker hostnames like `postgres`, `redis`). A user can configure their AI provider `base_url` to point to an internal service, and the backend will make HTTP requests to it on their behalf, potentially:
- Scanning internal Docker network ports
- Accessing cloud metadata endpoints (e.g., AWS/GCP instance credentials)
- Probing Redis/PostgreSQL/NATS management interfaces

**Mitigating factors**:
- The user's API key (secret) is sent as `Authorization: Bearer <key>` — so the user would need to know the target service's expected auth format
- Internal services (postgres, redis, nats) don't speak HTTP on their standard ports, limiting the attack surface
- The Docker network is already somewhat exposed to users via the frontend proxy

**Recommendation**: Add private IP range validation to `validateBaseURL`:
- Reject `127.0.0.0/8`, `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `169.254.0.0/16`, `::1`, `fc00::/7`
- Reject hostnames matching Docker service names (`postgres`, `redis`, `nats`, `backend`, `clickhouse`, `umami`)
- Consider using a DNS resolver to catch hostnames that resolve to private IPs

## Verified Safe (No Issues Found)

- **Prompt injection defense**: User input is wrapped in XML tags (`wrapXML`) and sanitized (`sanitizeInput` strips `</` to prevent tag spoofing). This follows Anthropic/OpenAI recommendations for isolating untrusted input in prompts.
- **API key encryption**: Keys encrypted with AES-256-GCM via `secretbox.Box` with per-key salt + nonce. Ciphertext stored in PostgreSQL, never logged. Decrypted secrets cached in-memory with TTL (5 min).
- **API key access control**: `GetSecret` and `SetSecret` both filter by `user_id` — users can only access their own API keys. Admin AI Gateway providers use separate repository with admin-only access.
- **Wallet pre-check**: `walletChecker` called before each AI API call — users with insufficient balance are rejected before any tokens are consumed.
- **Post-call billing**: `PostCallBiller` called after successful AI call — if billing fails, the result is discarded, ensuring users cannot use AI without paying.
- **Circuit breaker**: Per-user, per-provider circuit breaker prevents cascading failures when an AI provider is down.
- **Model whitelist**: `modelFilter` (ADR-0025 §5.2) allows admins to restrict which models users can access.
- **LLM-generated code execution sandbox**:
  - `MaxSourceSize: 500KB` — source size limit before parsing
  - `MaxTicks: 10,000,000` — instruction counter prevents infinite loops
  - `MaxCallDepth: 256` — limits recursion depth
  - `MaxStackDepth: 4096` — limits operand stack size
  - `safeRun()` with `defer recover()` — panic recovery on all execution paths
  - `context.WithTimeout` — execution time bounded by context deadline
  - No network access from VM (no `net` package, no file I/O)
- **Source code size validation**: `SubmitStrategy` handler checks `len(sourceCode) > mql2go.MaxSourceSize` before compilation.
- **Authentication on all handlers**: `interceptor.GetUserID(ctx)` checked in every Agent gateway handler — unauthenticated requests rejected.
- **Base URL validation**: Scheme restricted to `http`/`https`, host must be non-empty (partial protection — see W4-1).
- **HTTP response body limits**: External API responses read with `io.LimitReader` (64KB for chat errors, 8KB for failover errors).
- **No JSON for data persistence**: API keys stored encrypted in PostgreSQL. External API responses parsed with `encoding/json` (exempted per project rules — external protocol constraint).
- **Token usage tracking**: Input/output tokens recorded for billing and usage analytics.

## Reuse Preflight

- **W4-1**: NEW: SSRF protection recommendation (not implemented — documented as recommendation)

## Migrations

No migrations required.

## Deployment

No code changes in W4 (documentation-only findings).
