package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"alphaforge/internal/repository"
)

// Sentinel errors for deletion operations.
var (
	ErrCannotDeleteSelf     = fmt.Errorf("cannot delete yourself")
	ErrCannotDeleteLastAdmin = fmt.Errorf("cannot delete the last admin")
	ErrUserNotDeleted       = fmt.Errorf("user not found or not deleted")
)

// UserDeletionService encapsulates the deletion/restore flow:
// validation → scan affected tables → BEGIN tx → audit log + soft-delete → COMMIT.
// Audit log and soft-delete are in the same transaction — either both succeed
// or both roll back, preventing phantom audit records.
type UserDeletionService struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

func NewUserDeletionService(repo *repository.AdminRepository, log *zap.Logger) *UserDeletionService {
	return &UserDeletionService{repo: repo, log: log}
}

// SoftDeleteUser performs a single-user soft-delete within a transaction.
func (s *UserDeletionService) SoftDeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	// ── Pre-transaction validation ──
	if actorID != uuid.Nil && actorID == targetID {
		return ErrCannotDeleteSelf
	}
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		return fmt.Errorf("get target user: %w", err)
	}
	if target.Role == "admin" && adminCount <= 1 {
		return ErrCannotDeleteLastAdmin
	}

	// ── Pre-transaction: scan affected tables (read-only) ──
	affected, err := s.repo.GetAffectedTableCounts(ctx, targetID)
	if err != nil {
		s.log.Warn("user deletion: scan affected tables failed, auditing with empty snapshot",
			zap.String("target", targetID.String()), zap.Error(err))
		affected = nil
	}

	// ── Transaction: audit log + soft delete ──
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.InsertAuditLogTx(ctx, tx, actorID, "delete_user", targetID.String(), target.Email, affected); err != nil {
		return fmt.Errorf("audit log: %w", err) // tx rolls back
	}
	if err := s.repo.DeleteUserTx(ctx, tx, targetID); err != nil {
		return fmt.Errorf("soft delete: %w", err) // tx rolls back
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	s.log.Info("user deletion: soft-deleted",
		zap.String("actor", actorID.String()),
		zap.String("target", targetID.String()),
		zap.Int("affected_tables", len(affected)))
	return nil
}

// SoftDeleteUsers performs a batch soft-delete within a single transaction.
func (s *UserDeletionService) SoftDeleteUsers(ctx context.Context, actorID uuid.UUID, ids []string) (deleted int64, failed int, errors []string) {
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return 0, len(ids), []string{fmt.Sprintf("count admins: %v", err)}
	}

	type target struct {
		id     uuid.UUID
		email  string
		affected map[string]int64
	}
	var targets []target
	uuids := make([]uuid.UUID, 0, len(ids))

	for _, raw := range ids {
		uid, err := uuid.Parse(raw)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: invalid id", raw))
			failed++
			continue
		}
		if actorID != uuid.Nil && actorID == uid {
			errors = append(errors, fmt.Sprintf("%s: cannot delete yourself", raw))
			failed++
			continue
		}
		t, err := s.repo.GetUserByID(ctx, uid)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", raw, err))
			failed++
			continue
		}
		if t.Role == "admin" && adminCount <= 1 {
			errors = append(errors, fmt.Sprintf("%s: cannot delete the last admin", raw))
			failed++
			continue
		}
		// Pre-transaction: scan affected tables.
		affected, err := s.repo.GetAffectedTableCounts(ctx, uid)
		if err != nil {
			s.log.Warn("user deletion: batch scan affected tables failed",
				zap.String("target", uid.String()), zap.Error(err))
			affected = nil
		}
		uuids = append(uuids, uid)
		targets = append(targets, target{id: uid, email: t.Email, affected: affected})
	}

	if len(uuids) == 0 {
		return 0, failed, errors
	}

	// ── Transaction: audit logs + batch soft delete ──
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return 0, len(ids), []string{fmt.Sprintf("begin tx: %v", err)}
	}
	defer tx.Rollback(ctx)

	for _, t := range targets {
		if err := s.repo.InsertAuditLogTx(ctx, tx, actorID, "batch_delete", t.id.String(), t.email, t.affected); err != nil {
			return 0, len(ids), []string{fmt.Sprintf("audit log for %s: %v", t.id, err)}
		}
	}
	deleted, err = s.repo.DeleteUsersTx(ctx, tx, uuids)
	if err != nil {
		return 0, len(ids), []string{fmt.Sprintf("batch delete: %v", err)}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, len(ids), []string{fmt.Sprintf("commit: %v", err)}
	}

	s.log.Info("user deletion: batch soft-deleted",
		zap.String("actor", actorID.String()),
		zap.Int64("deleted", deleted),
		zap.Int("failed", failed))
	return deleted, failed, errors
}

// RestoreUser clears the soft-delete marker and records an audit entry
// within a single transaction.
func (s *UserDeletionService) RestoreUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	tx, err := s.repo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.RestoreUserTx(ctx, tx, targetID); err != nil {
		if err == repository.ErrUserNotFound {
			return ErrUserNotDeleted
		}
		return fmt.Errorf("restore user: %w", err)
	}
	if err := s.repo.InsertAuditLogTx(ctx, tx, actorID, "restore_user", targetID.String(), targetID.String(), nil); err != nil {
		return fmt.Errorf("audit log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	s.log.Info("user deletion: restored",
		zap.String("actor", actorID.String()),
		zap.String("target", targetID.String()))
	return nil
}
