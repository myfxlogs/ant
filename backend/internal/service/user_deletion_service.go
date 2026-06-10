package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"anttrader/internal/repository"
)

// Sentinel errors for deletion operations.
var (
	ErrCannotDeleteSelf     = fmt.Errorf("cannot delete yourself")
	ErrCannotDeleteLastAdmin = fmt.Errorf("cannot delete the last admin")
	ErrUserNotDeleted       = fmt.Errorf("user not found or not deleted")
)

// UserDeletionService encapsulates the deletion/restore flow:
// fetch target → scan affected tables → write audit log → soft-delete.
// Handlers delegate to this service so the orchestration lives in one place.
type UserDeletionService struct {
	repo *repository.AdminRepository
	log  *zap.Logger
}

func NewUserDeletionService(repo *repository.AdminRepository, log *zap.Logger) *UserDeletionService {
	return &UserDeletionService{repo: repo, log: log}
}

// SoftDeleteUser performs a single-user soft-delete with audit logging.
func (s *UserDeletionService) SoftDeleteUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	// Prevent self-deletion.
	if actorID != uuid.Nil && actorID == targetID {
		return ErrCannotDeleteSelf
	}
	// Prevent deleting the last admin.
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

	affected, err := s.repo.GetAffectedTableCounts(ctx, targetID)
	if err != nil {
		// Best-effort: audit with empty affected data if scan fails.
		s.log.Warn("user deletion: scan affected tables failed, auditing with empty snapshot",
			zap.String("target", targetID.String()), zap.Error(err))
		affected = nil
	}

	s.recordAuditLog(ctx, actorID, "delete_user", targetID.String(), target.Email, affected)

	if err := s.repo.DeleteUser(ctx, targetID); err != nil {
		return fmt.Errorf("soft delete user: %w", err)
	}

	s.log.Info("user deletion: soft-deleted",
		zap.String("actor", actorID.String()),
		zap.String("target", targetID.String()),
		zap.Int("affected_tables", len(affected)))
	return nil
}

// SoftDeleteUsers performs a batch soft-delete. Returns deleted count, failed
// count, and per-ID error descriptions for IDs that were skipped.
func (s *UserDeletionService) SoftDeleteUsers(ctx context.Context, actorID uuid.UUID, ids []string) (deleted int64, failed int, errors []string) {
	adminCount, err := s.repo.CountAdmins(ctx)
	if err != nil {
		return 0, len(ids), []string{fmt.Sprintf("count admins: %v", err)}
	}

	type target struct {
		id    uuid.UUID
		email string
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
		uuids = append(uuids, uid)
		targets = append(targets, target{id: uid, email: t.Email})
	}

	// Audit each user before batch soft-delete.
	for _, t := range targets {
		affected, err := s.repo.GetAffectedTableCounts(ctx, t.id)
		if err != nil {
			s.log.Warn("user deletion: batch scan affected tables failed",
				zap.String("target", t.id.String()), zap.Error(err))
			affected = nil
		}
		s.recordAuditLog(ctx, actorID, "batch_delete", t.id.String(), t.email, affected)
	}

	if len(uuids) > 0 {
		var err error
		deleted, err = s.repo.DeleteUsers(ctx, uuids)
		if err != nil {
			return 0, len(ids), []string{fmt.Sprintf("batch delete: %v", err)}
		}
	}

	s.log.Info("user deletion: batch soft-deleted",
		zap.String("actor", actorID.String()),
		zap.Int64("deleted", deleted),
		zap.Int("failed", failed))
	return deleted, failed, errors
}

// RestoreUser clears the soft-delete marker and records an audit entry.
func (s *UserDeletionService) RestoreUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if err := s.repo.RestoreUser(ctx, targetID); err != nil {
		if err == repository.ErrUserNotFound {
			return ErrUserNotDeleted
		}
		return fmt.Errorf("restore user: %w", err)
	}

	s.recordAuditLog(ctx, actorID, "restore_user", targetID.String(), targetID.String(), nil)

	s.log.Info("user deletion: restored",
		zap.String("actor", actorID.String()),
		zap.String("target", targetID.String()))
	return nil
}

// recordAuditLog writes an admin action to the audit log. Audit failures are
// logged but never block the primary operation.
func (s *UserDeletionService) recordAuditLog(ctx context.Context, actorID uuid.UUID, action, targetID, targetEmail string, affected map[string]int64) {
	if err := s.repo.InsertAuditLog(ctx, actorID, action, targetID, targetEmail, affected); err != nil {
		s.log.Error("user deletion: audit log write failed — operation continued without audit record",
			zap.String("action", action),
			zap.String("actor", actorID.String()),
			zap.String("target", targetID),
			zap.Error(err))
	}
}
