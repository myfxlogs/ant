# Notification System Audit Report

## Scope

- `internal/notification/sender.go` — Sender: persist + broadcast
- `internal/notification/pubsub.go` — in-process pubsub with per-user channels
- `internal/connect/notification/handler.go` — ListNotifications, MarkRead, MarkAllRead, StreamNotifications, SendNotification
- `internal/connect/notification/handler_prefs.go` — GetNotificationPrefs, SetNotificationPrefs
- `internal/repository/notification_repository.go` — ListByUser, MarkRead, MarkAllRead, Insert
- `internal/marketplace/notification_triggers.go` — notifyTrialExpiring, notifyNewStrategy, notifyPriceChange, notifyNewRating, notifySubExpiring, notifyPerformanceAnomaly

## Findings

### N1 — `SendNotification` IDOR: any user can send notifications to any other user 🟡 MEDIUM

**File**: `connect/notification/handler.go:164-185`

**Problem**: The `SendNotification` RPC handler accepts `req.Msg.UserId` from the request body without verifying the caller's identity or admin status. Any authenticated user can send arbitrary notifications (with arbitrary type, title, message) to any other user by specifying their `user_id` in the request.

**Attack scenario**: A malicious user sends a notification with `type: "deposit_confirmed"`, `title: "Deposit Confirmed"`, `message: "Your deposit of 1000 USDT has been confirmed."` to a victim user. The victim sees this notification in their notification panel and may believe a deposit was actually confirmed, leading to social engineering attacks.

**Fix**: Added admin-only access control — the handler now verifies the caller is an admin via `SELECT is_admin FROM users WHERE id = $1`. Non-admin users receive `CodePermissionDenied`.

**Risk if unfixed**: Social engineering, spam, phishing via fake system notifications.

### N2 — `MarkRead` IDOR: any user can mark any notification as read 🟢 LOW

**File**: `connect/notification/handler.go:84-97`, `repository/notification_repository.go:88-93`

**Problem**: The `MarkRead` handler did not verify that the notification belongs to the authenticated user. The repository method `MarkRead(ctx, id)` only used `WHERE id = $1` without a `user_id` filter. Any authenticated user could mark any other user's notification as read by knowing or guessing the notification ID.

**Fix**: 
- Handler: Added authentication check + passes `userID` to repository method.
- Repository: Replaced `MarkRead(ctx, id)` with `MarkReadForUser(ctx, id, userID)` that adds `AND user_id = $2` to the UPDATE query. Returns error if no rows affected (notification not found or not owned by user).

**Risk if unfixed**: Minor annoyance — a user's unread notifications could be silently dismissed by another user. No financial impact.

## Verified Safe (No Issues Found)

### Sender
- **Persistence + broadcast**: `Sender.Send` atomically inserts to PostgreSQL then publishes to in-process pubsub. If insert fails, no broadcast occurs. If broadcast fails (no subscribers), notification is still persisted.
- **Error handling**: Insert errors are logged and returned; broadcast is best-effort (non-blocking channel send).

### In-Process PubSub
- **Thread-safe**: `Subscriber` uses `sync.Mutex` for all operations (Subscribe, Unsubscribe, Publish).
- **No deadlock**: `Publish` uses non-blocking `select` with `default` case — drops message if subscriber buffer is full (client too slow). Never blocks under mutex.
- **No panic on close**: `Unsubscribe` removes channel from map before closing. `Publish` iterates map under same mutex, so never sends to a closed channel.
- **Buffered channels**: 16-element buffer per subscriber prevents drops under normal load.
- **Cleanup**: `Unsubscribe` removes empty user maps to prevent memory growth.

### StreamNotifications (SSE)
- **Authentication**: Requires authenticated user from context.
- **Unread filter**: `req.Msg.UnreadOnly` skips already-read notifications.
- **Graceful shutdown**: `defer s.sub.Unsubscribe(uid, ch)` ensures cleanup on stream end.
- **Context cancellation**: `case <-ctx.Done()` handles client disconnect.

### ListNotifications
- **User-scoped**: `WHERE user_id = $1` with authenticated user ID. No cross-user access.
- **Pagination**: Limit/offset with default limit=50.
- **Unread count**: Separate COUNT query, errors silently default to 0 (acceptable — doesn't block listing).

### MarkAllRead
- **User-scoped**: `WHERE user_id = $1 AND is_read = false` with authenticated user ID.

### Notification Preferences
- **User-scoped**: Both Get and Set use authenticated user ID from context.
- **Upsert**: `ON CONFLICT (user_id) DO UPDATE` for idempotent preference updates.
- **Default to true**: Missing prefs row defaults all notifications to enabled (sensible default — users see everything until they opt out).

### Marketplace Notification Triggers
- **Push-first**: All triggers are lazy (called from user-facing operations), no cron/polling.
- **Atomic claim**: `notifyTrialExpiring` and `notifySubExpiring` use atomic `UPDATE ... WHERE notified_expiring = false RETURNING` to prevent duplicate notifications from concurrent goroutines.
- **Preference-respected**: All triggers check `marketplace_notification_prefs` with `COALESCE(..., true)` fallback.
- **Non-blocking**: All triggers run in goroutines via `go s.notifyX(context.WithoutCancel(ctx), ...)` — don't block the calling operation.
- **Error-tolerant**: All triggers silently ignore errors (notifications are best-effort, shouldn't break the calling operation).

### Data Storage
- **Proto serialization**: Notification `data` field stored as protobuf `data_proto` column, not JSON.
- **No JSON**: All notification data uses `structpb.Struct` → `proto.Marshal` for persistence.

## Architecture Compliance

- ✅ No REST endpoints (ConnectRPC + SSE only)
- ✅ No JSON for data persistence (PostgreSQL + proto)
- ✅ No float64 in price calculations (notification triggers use `decimal.Decimal` for thresholds)
- ✅ Push-first: in-process pubsub + SSE, no polling
- ✅ No `//nolint` or `// @ts-ignore`

## Reuse Preflight

- **N1 fix**: NEW: admin check query (no existing admin check in notification handler)
- **N2 fix**: REUSE: `MarkReadForUser` pattern @ `notification_repository.go:88-101` (mirrors existing `MarkReadForUser` in `notification_repository.go` — actually this was already defined in the codebase for the `MarkReadForUser` method that was unused, replaced the old `MarkRead`)

## Deployment

- `go build ./...` ✅
- `go test ./internal/notification/... ./internal/connect/notification/... ./internal/repository/...` ✅
- `docker compose build backend` ✅
- `docker compose up -d backend` ✅
- Container health: `healthy` ✅

## Notes

- `notifyNewStrategy` comment says "matching asset class preferences" but the query doesn't filter by asset class — all users with `new_strategy_enabled=true` are notified. This is a minor logic inconsistency (comment vs code), not a security issue. Fixing would require adding asset class to the prefs table, which is a feature change beyond audit scope.
- `StreamNotifications` comment mentions "keepalive comment every 15s" but no keepalive logic exists in the code. Long-idle streams may be closed by proxies/load balancers. This is a reliability issue, not a security one.
- `SendNotification` admin check uses `_ = s.pg.QueryRow(...).Scan(&isAdmin)` which silently ignores DB errors. If the DB is down, `isAdmin` defaults to `false` (fail-closed). This is the correct security posture.
