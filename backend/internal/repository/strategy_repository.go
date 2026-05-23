package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"anttrader/internal/model"
)

var (
	ErrStrategyNotFound = errors.New("strategy not found")
	ErrSignalNotFound   = errors.New("signal not found")
)

// StrategyRepository 策略仓库
type StrategyRepository struct {
	db *sqlx.DB
}

// NewStrategyRepository 创建策略仓库
func NewStrategyRepository(db *sqlx.DB) *StrategyRepository {
	return &StrategyRepository{db: db}
}

// Create 创建策略
func (r *StrategyRepository) Create(ctx context.Context, strategy *model.Strategy) error {
	query := `
		INSERT INTO strategies (
			id, user_id, account_id, name, description, symbol,
			conditions, actions, risk_control, status, auto_execute,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	now := time.Now()
	if strategy.ID == uuid.Nil {
		strategy.ID = uuid.New()
	}
	strategy.CreatedAt = now
	strategy.UpdatedAt = now

	_, err := r.db.ExecContext(ctx, query,
		strategy.ID, strategy.UserID, strategy.AccountID, strategy.Name,
		strategy.Description, strategy.Symbol, strategy.Conditions,
		strategy.Actions, strategy.RiskControl, strategy.Status,
		strategy.AutoExecute, strategy.CreatedAt, strategy.UpdatedAt,
	)

	return err
}

// GetByID 根据ID获取策略
func (r *StrategyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE id = $1`
	var strategy model.Strategy
	err := r.db.GetContext(ctx, &strategy, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrStrategyNotFound
		}
		return nil, err
	}
	return &strategy, nil
}

// GetByUserID 根据用户ID获取策略列表
func (r *StrategyRepository) GetByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE user_id = $1 ORDER BY created_at DESC`
	var strategies []*model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query, userID)
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// GetByAccountID 根据账户ID获取策略列表
func (r *StrategyRepository) GetByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE account_id = $1 ORDER BY created_at DESC`
	var strategies []*model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query, accountID)
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// GetActiveByUserID 获取用户激活的策略列表
func (r *StrategyRepository) GetActiveByUserID(ctx context.Context, userID uuid.UUID) ([]*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE user_id = $1 AND status = 'active' ORDER BY created_at DESC`
	var strategies []*model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query, userID)
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// GetActiveByAccountID 获取账户激活的策略列表
func (r *StrategyRepository) GetActiveByAccountID(ctx context.Context, accountID uuid.UUID) ([]*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE account_id = $1 AND status = 'active' ORDER BY created_at DESC`
	var strategies []*model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query, accountID)
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// GetActiveAutoExecuteStrategies 获取所有自动执行的激活策略
func (r *StrategyRepository) GetActiveAutoExecuteStrategies(ctx context.Context) ([]*model.Strategy, error) {
	query := `SELECT * FROM strategies WHERE status = 'active' AND auto_execute = true ORDER BY created_at DESC`
	var strategies []*model.Strategy
	err := r.db.SelectContext(ctx, &strategies, query)
	if err != nil {
		return nil, err
	}
	return strategies, nil
}

// Update 更新策略
func (r *StrategyRepository) Update(ctx context.Context, strategy *model.Strategy) error {
	query := `
		UPDATE strategies SET
			account_id = $2, name = $3, description = $4, symbol = $5,
			conditions = $6, actions = $7, risk_control = $8, status = $9,
			auto_execute = $10, updated_at = $11
		WHERE id = $1`

	strategy.UpdatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		strategy.ID, strategy.AccountID, strategy.Name, strategy.Description,
		strategy.Symbol, strategy.Conditions, strategy.Actions, strategy.RiskControl,
		strategy.Status, strategy.AutoExecute, strategy.UpdatedAt,
	)

	return err
}

// UpdateStatus 更新策略状态
func (r *StrategyRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := `UPDATE strategies SET status = $2, updated_at = $3 WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id, status, time.Now())
	return err
}

// Delete 删除策略
func (r *StrategyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM strategies WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrStrategyNotFound
	}
	return nil
}

// CreateSignal 创建策略信号
func (r *StrategyRepository) CreateSignal(ctx context.Context, signal *model.StrategySignal) error {
	query := `
		INSERT INTO strategy_signals (
			id, user_id, template_id, account_id, symbol, signal_type,
			volume, price, stop_loss, take_profit, reason,
			status, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	if signal.ID == uuid.Nil {
		signal.ID = uuid.New()
	}
	signal.CreatedAt = time.Now()

	_, err := r.db.ExecContext(ctx, query,
		signal.ID, signal.UserID, signal.TemplateID, signal.AccountID, signal.Symbol,
		signal.SignalType, signal.Volume, signal.Price, signal.StopLoss,
		signal.TakeProfit, signal.Reason, signal.Status, signal.CreatedAt,
	)

	return err
}

// GetSignalByID 根据ID获取信号
func (r *StrategyRepository) GetSignalByID(ctx context.Context, signalID uuid.UUID) (*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE id = $1`
	var signal model.StrategySignal
	err := r.db.GetContext(ctx, &signal, query, signalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrSignalNotFound
		}
		return nil, err
	}
	return &signal, nil
}

// GetPendingSignals 获取待处理的信号列表
func (r *StrategyRepository) GetPendingSignals(ctx context.Context, accountID uuid.UUID) ([]*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE account_id = $1 AND status = 'pending' ORDER BY created_at ASC`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, accountID)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// GetSignalsByTemplateID 根据模板ID获取信号列表
func (r *StrategyRepository) GetSignalsByTemplateID(ctx context.Context, templateID uuid.UUID, limit int) ([]*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE template_id = $1 ORDER BY created_at DESC LIMIT $2`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, templateID, limit)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// GetSignalsByUserID 根据用户ID获取信号列表
func (r *StrategyRepository) GetSignalsByUserID(ctx context.Context, userID uuid.UUID, limit int) ([]*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// GetSignalsByAccountID 根据账户ID获取信号列表
func (r *StrategyRepository) GetSignalsByAccountID(ctx context.Context, accountID uuid.UUID, limit int) ([]*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE account_id = $1 ORDER BY created_at DESC LIMIT $2`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, accountID, limit)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// UpdateSignalStatus 更新信号状态
func (r *StrategyRepository) UpdateSignalStatus(ctx context.Context, signalID uuid.UUID, status string, ticket int64, profit float64) error {
	query := `
		UPDATE strategy_signals SET
			status = $2, ticket = $3, profit = $4, executed_at = $5
		WHERE id = $1`

	var executedAt *time.Time
	if status == model.SignalStatusExecuted {
		now := time.Now()
		executedAt = &now
	}

	_, err := r.db.ExecContext(ctx, query, signalID, status, ticket, profit, executedAt)
	return err
}

// ConfirmSignal 确认信号
func (r *StrategyRepository) ConfirmSignal(ctx context.Context, signalID uuid.UUID) error {
	query := `UPDATE strategy_signals SET status = 'confirmed' WHERE id = $1 AND status = 'pending'`
	result, err := r.db.ExecContext(ctx, query, signalID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSignalNotFound
	}
	return nil
}

// CancelSignal 取消信号
func (r *StrategyRepository) CancelSignal(ctx context.Context, signalID uuid.UUID) error {
	query := `UPDATE strategy_signals SET status = 'cancelled' WHERE id = $1 AND status IN ('pending', 'confirmed')`
	result, err := r.db.ExecContext(ctx, query, signalID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrSignalNotFound
	}
	return nil
}

// GetSignalsByStatus 根据状态获取信号列表
func (r *StrategyRepository) GetSignalsByStatus(ctx context.Context, status string, limit int) ([]*model.StrategySignal, error) {
	query := `SELECT * FROM strategy_signals WHERE status = $1 ORDER BY created_at ASC LIMIT $2`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, status, limit)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// CountSignalsByStrategy 统计策略的信号数量
func (r *StrategyRepository) CountSignalsByStrategy(ctx context.Context, strategyID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM strategy_signals WHERE strategy_id = $1`
	var count int
	err := r.db.GetContext(ctx, &count, query, strategyID)
	return count, err
}

// GetRecentSignals 获取最近的信号列表
func (r *StrategyRepository) GetRecentSignals(ctx context.Context, userID uuid.UUID, limit int) ([]*model.StrategySignal, error) {
	query := `
		SELECT ss.* FROM strategy_signals ss
		JOIN strategies s ON ss.strategy_id = s.id
		WHERE s.user_id = $1
		ORDER BY ss.created_at DESC
		LIMIT $2`
	var signals []*model.StrategySignal
	err := r.db.SelectContext(ctx, &signals, query, userID, limit)
	if err != nil {
		return nil, err
	}
	return signals, nil
}

// GetSignalStats 获取信号统计
func (r *StrategyRepository) GetSignalStats(ctx context.Context, strategyID uuid.UUID) (map[string]int, error) {
	query := `
		SELECT status, COUNT(*) as count
		FROM strategy_signals
		WHERE strategy_id = $1
		GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query, strategyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats := make(map[string]int)
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		stats[status] = count
	}

	return stats, nil
}
